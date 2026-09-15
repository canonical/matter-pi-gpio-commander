package tests

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/canonical/matter-snap-testing/env"
	"github.com/canonical/matter-snap-testing/utils"
	"github.com/stretchr/testify/assert"
)

const (
	// enviroment variables
	specificGpioChip = "GPIO_CHIP"
	specificGpioLine = "GPIO_LINE"
	gpioChipMock     = "MOCK_GPIO"
)

const snapMatterPiGPIO = "matter-pi-gpio-commander"
const chipToolSnap = "chip-tool"

// The daemon declared by the snap.
const lightingService = snapMatterPiGPIO + ".lighting"

// Exit code returned by test-blink on both of its expected failure paths.
const blinkFailureExitCode = 1

// A line offset that exists on neither a Raspberry Pi nor the simulator.
const invalidGpioLine = "99"

var gpioChip = os.Getenv(specificGpioChip)
var gpioLine = os.Getenv(specificGpioLine)

/*
The service state is recorded right after the installation, before any test
starts the service. The install-mode assertions therefore do not depend on the
order in which the tests run.
*/
var statusAfterInstall serviceStatus
var statusAfterInstallErr error

// serviceStatus holds the Startup and Current columns that "snap services"
// reports for a single service.
type serviceStatus struct {
	startup string
	current string
}

// readServiceStatus queries the state of a service in a single call. An error
// is returned when the query fails or when the service is not listed, so that
// a failed query cannot be mistaken for a disabled or inactive service.
func readServiceStatus(service string) (serviceStatus, error) {
	stdout, stderr, err := utils.Exec(nil, "snap services "+service)
	if err != nil {
		return serviceStatus{},
			fmt.Errorf("query the state of %s: %w: %s", service, err, stderr)
	}

	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != service {
			continue
		}
		return serviceStatus{startup: fields[1], current: fields[2]}, nil
	}

	return serviceStatus{},
		fmt.Errorf("service %s is not listed in: %s", service, stdout)
}

// syncBuffer collects the output of a running command, which is written from
// another goroutine while the test reads it.
type syncBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.Write(p)
}

func (b *syncBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.String()
}

func TestMain(m *testing.M) {
	teardown, err := setup()
	if err != nil {
		log.Fatalf("Failed to setup tests: %s", err)
	}

	code := m.Run()
	teardown()

	os.Exit(code)
}

func setup() (teardown func(), err error) {
	log.Println("[CLEAN]")
	utils.SnapRemove(nil, snapMatterPiGPIO)
	utils.SnapRemove(nil, chipToolSnap)

	log.Println("[SETUP]")

	teardown = func() {
		log.Println("[TEARDOWN]")

		log.Println("Removing installed snap:", env.Teardown())
		if env.Teardown() {
			utils.SnapRemove(nil, snapMatterPiGPIO)
			utils.SnapRemove(nil, chipToolSnap)
			utils.Exec(nil, "./gpio-mock.sh teardown")
		}
	}

	// install matter-pi-gpio-commander
	if env.SnapPath() != "" {
		err = utils.SnapInstallFromFile(nil, env.SnapPath())
	} else {
		err = utils.SnapInstallFromStore(nil, snapMatterPiGPIO, env.SnapChannel())
	}
	if err != nil {
		teardown()
		return
	}

	// The snap declares install-mode: disable, so the service must not be
	// running yet. This is recorded here because later tests start it.
	statusAfterInstall, statusAfterInstallErr = readServiceStatus(lightingService)

	if err = utils.SnapConnect(nil, snapMatterPiGPIO+":avahi-control", ""); err != nil {
		teardown()
		return
	}

	// Bluetooth is not available on every test machine, so a failure here
	// must not prevent the non-Bluetooth tests from running.
	if bluezErr := utils.SnapConnect(nil, snapMatterPiGPIO+":bluez", ""); bluezErr != nil {
		log.Printf("[SETUP] Could not connect the bluez interface: %s", bluezErr)
	}

	if useGPIOMock() {
		stdout, _, mockErr := utils.Exec(nil, "./gpio-mock.sh 2>&1")
		_ = utils.WriteLogFile(nil, "gpio-mock", stdout)
		if mockErr != nil {
			teardown()
			return nil, fmt.Errorf("failed to set up gpio-sim: %w", mockErr)
		}
	}

	if err = utils.SnapConnect(nil,
		snapMatterPiGPIO+":custom-gpio",
		snapMatterPiGPIO+":custom-gpio-dev"); err != nil {
		teardown()
		return
	}

	if err = setupGPIO(); err != nil {
		teardown()
		return
	}

	return
}

func useGPIOMock() bool {
	return os.Getenv(gpioChipMock) == "true"
}

func getMockGPIO() (string, error) {
	gpioChipNumber, stderr, err := utils.Exec(nil, "./gpio-mock.sh chip")
	if err != nil {
		return "", fmt.Errorf("failed to get mock gpio chip number: %w: %s", err, stderr)
	}
	return strings.TrimSpace(gpioChipNumber), nil
}

func setupGPIO() error {
	var err error
	if useGPIOMock() {
		utils.SnapSet(nil, snapMatterPiGPIO, "gpiochip-validation", "false")

		gpioChip, err = getMockGPIO()
		if err != nil {
			return err
		}
		gpioLine = "4"

		log.Printf("[TEST] Using mockup gpio: /dev/gpiochip%s", gpioChip)
		log.Printf("[TEST] Using default gpio-mock line: %s", gpioLine)
	}

	utils.SnapSet(nil, snapMatterPiGPIO, "gpiochip", gpioChip)
	utils.SnapSet(nil, snapMatterPiGPIO, "gpio", gpioLine)

	return nil
}

func gpioState() (string, error) {
	stdout, stderr, err := utils.Exec(nil,
		fmt.Sprintf("./gpio-mock.sh state %s", gpioLine))
	if err != nil {
		return "", fmt.Errorf("read simulated GPIO state: %w: %s", err, stderr)
	}
	return strings.TrimSpace(stdout), nil
}

func waitForGPIOState(t *testing.T, expected string) {
	t.Helper()

	if !useGPIOMock() {
		return
	}

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		state, err := gpioState()
		assert.NoError(collect, err)
		assert.Equal(collect, expected, state)
	}, 5*time.Second, 100*time.Millisecond)
}

func runChipTool(t *testing.T, args string) string {
	t.Helper()

	stdout, _, err := utils.Exec(t, "sudo chip-tool "+args+" 2>&1")
	assert.NoError(t, utils.WriteLogFile(t, chipToolSnap, stdout))
	assert.NoError(t, err)
	return stdout
}

// Recent chip-tool versions report the OnOff attribute as TRUE/FALSE, older
// ones as 1/0.
var onOffAttributePattern = regexp.MustCompile(`(?i)OnOff:\s*(TRUE|FALSE|1|0)\b`)

func parseOnOffAttribute(output string) (string, error) {
	match := onOffAttributePattern.FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("chip-tool did not report the OnOff attribute")
	}

	switch strings.ToUpper(match[1]) {
	case "TRUE", "1":
		return "on", nil
	default:
		return "off", nil
	}
}

// readOnOffAttribute reads the OnOff attribute through chip-tool. The read is
// retried because the application needs a moment to advertise itself again
// after a restart.
func readOnOffAttribute(t *testing.T) string {
	t.Helper()

	var value string
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		stdout, _, err := utils.Exec(nil, "sudo chip-tool onoff read on-off 110 1 2>&1")
		_ = utils.WriteLogFile(t, chipToolSnap, stdout)
		if !assert.NoError(collect, err) {
			return
		}

		parsed, parseErr := parseOnOffAttribute(stdout)
		if !assert.NoError(collect, parseErr) {
			return
		}
		value = parsed
	}, 60*time.Second, 2*time.Second)

	return value
}

// stopBlink terminates the test-blink process tree and waits for it to exit.
// The processes are started through sudo and therefore run as root, so they
// cannot be signalled by the unprivileged test process itself.
func stopBlink(t *testing.T, command *exec.Cmd, done <-chan struct{}) {
	t.Helper()

	if command.Process == nil {
		return
	}

	// Setpgid makes the sudo PID the process group ID of the whole tree,
	// so the snap application is stopped together with its parent.
	// The signal and the negative PID must be separated by "--", because
	// /usr/bin/kill otherwise parses the group as an option and does nothing.
	signalGroup := func(signal string) {
		utils.Exec(nil, fmt.Sprintf("sudo kill -s %s -- -%d", signal, command.Process.Pid))
	}

	signalGroup("TERM")
	select {
	case <-done:
		return
	case <-time.After(10 * time.Second):
	}

	signalGroup("KILL")
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("test-blink did not exit after being killed")
	}
}

// runBlinkExpectingFailure runs test-blink and requires it to exit with an
// error instead of blinking. The timeout prevents a job from hanging when the
// expected failure does not occur.
//
// timeout is deliberately not given --foreground: in its default mode it puts
// the command in its own process group and signals that whole group, so the
// confined application is stopped together with "snap run" instead of being
// orphaned while holding the GPIO line. --kill-after adds a SIGKILL for the
// case where the group ignores SIGTERM.
func runBlinkExpectingFailure(t *testing.T, reason string) string {
	t.Helper()

	var output bytes.Buffer
	command := exec.Command("sudo", "timeout", "--kill-after=5s", "5s",
		"snap", "run", snapMatterPiGPIO+".test-blink")
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()

	stdout := output.String()
	assert.NoError(t, utils.WriteLogFile(t, snapMatterPiGPIO, stdout))

	var exitErr *exec.ExitError
	if assert.ErrorAs(t, err, &exitErr, reason) {
		// Both expected failure paths make test-blink exit with 1. Any other
		// status, in particular timeout's 124 for SIGTERM or 137 for SIGKILL,
		// means that the application kept running instead of failing.
		assert.Equal(t, blinkFailureExitCode, exitErr.ExitCode(),
			"test-blink did not exit with the expected failure status: %s", reason)
	}
	assert.NotContains(t, stdout, "Setting GPIO",
		"test-blink toggled the line, but it should have failed: %s", reason)

	return stdout
}

func TestConfigurationValidation(t *testing.T) {
	originalValidation, _, err := utils.Exec(t,
		"sudo snap get "+snapMatterPiGPIO+" gpiochip-validation")
	assert.NoError(t, err)
	t.Cleanup(func() {
		utils.SnapSet(t, snapMatterPiGPIO, "gpiochip-validation",
			strings.TrimSpace(originalValidation))
		utils.SnapSet(t, snapMatterPiGPIO, "gpiochip", gpioChip)
		utils.SnapSet(t, snapMatterPiGPIO, "gpio", gpioLine)
	})

	_, _, err = utils.Exec(nil, "sudo snap set "+snapMatterPiGPIO+" gpio=0")
	assert.Error(t, err)

	stdout, _, err := utils.Exec(t, "sudo snap get "+snapMatterPiGPIO+" gpio")
	assert.NoError(t, err)
	assert.Equal(t, gpioLine, strings.TrimSpace(stdout))

	// Enabling validation re-runs the hook against the current value, so the
	// chip is first set back to the one this machine actually uses.
	utils.SnapSet(t, snapMatterPiGPIO, "gpiochip", gpioChip)
	utils.SnapSet(t, snapMatterPiGPIO, "gpiochip-validation", "true")
	_, _, err = utils.Exec(nil, "sudo snap set "+snapMatterPiGPIO+" gpiochip=7")
	assert.Error(t, err)
}

func TestPackagingAndInstallBehavior(t *testing.T) {
	stdout, _, err := utils.Exec(t, "snap info --verbose "+snapMatterPiGPIO)
	assert.NoError(t, err)
	assert.Regexp(t, `(?m)^\s*confinement:\s+strict\s*$`, stdout)

	stdout, _, err = utils.Exec(t, "sudo snap run "+snapMatterPiGPIO+".help")
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Usage")

	if assert.NoError(t, statusAfterInstallErr,
		"the state of the service right after the installation must be known") {
		assert.Equal(t, "disabled", statusAfterInstall.startup,
			"the snap declares install-mode: disable, so the service must not be enabled after installation")
		assert.Equal(t, "inactive", statusAfterInstall.current,
			"the snap declares install-mode: disable, so the service must not be active after installation")
	}
}

func TestInvalidGPIOLine(t *testing.T) {
	utils.SnapSet(t, snapMatterPiGPIO, "gpio", invalidGpioLine)
	t.Cleanup(func() {
		utils.SnapSet(t, snapMatterPiGPIO, "gpio", gpioLine)
	})

	stdout := runBlinkExpectingFailure(t, "the GPIO line does not exist")
	assert.Contains(t, stdout, "Failed to request output line")
}

/*
TestBlinkOperation runs the test-blink app in the snap.
The log output is checked for the correctly configured GPIO Chip and GPIO Line,
as well as the existence of the ON and OFF log messages.
When the GPIO is simulated, the line state itself is verified as well.
*/
func TestBlinkOperation(t *testing.T) {
	t.Run("WithoutGPIOInterface", func(t *testing.T) {
		_, _, err := utils.Exec(t, "sudo snap disconnect "+
			snapMatterPiGPIO+":custom-gpio "+
			snapMatterPiGPIO+":custom-gpio-dev")
		assert.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, utils.SnapConnect(t,
				snapMatterPiGPIO+":custom-gpio",
				snapMatterPiGPIO+":custom-gpio-dev"))
		})

		stdout := runBlinkExpectingFailure(t, "the GPIO interface is disconnected")
		assert.Contains(t, stdout, "Failed to request output line")
	})

	/*
		`test-blink` runs until it is stopped, so it is started in the
		background and terminated once the assertions are done.
	*/
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	var output syncBuffer
	command := exec.CommandContext(ctx, "sudo", "snap", "run", snapMatterPiGPIO+".test-blink")
	command.Stdout = &output
	command.Stderr = &output
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if !assert.NoError(t, command.Start()) {
		return
	}

	// waitErr is only read after done is closed, so no extra synchronisation
	// is needed between the waiting goroutine and the test.
	var waitErr error
	done := make(chan struct{})
	go func() {
		waitErr = command.Wait()
		close(done)
	}()

	exited := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	if useGPIOMock() {
		seen := map[string]bool{}
		assert.Eventually(t, func() bool {
			if exited() {
				return true
			}

			if state, err := gpioState(); err == nil {
				seen[state] = true
			}
			return seen["high"] && seen["low"]
		}, 30*time.Second, 50*time.Millisecond)
		assert.True(t, seen["high"] && seen["low"],
			"the simulated GPIO line did not toggle, observed states: %v", seen)
	} else {
		// Without the simulator only the application output can be checked,
		// so give it enough time to log a few toggles.
		assert.Eventually(t, func() bool {
			return exited() ||
				(strings.Contains(output.String(), "On") &&
					strings.Contains(output.String(), "Off"))
		}, 30*time.Second, 100*time.Millisecond)
	}

	assert.False(t, exited(), "test-blink exited before the assertions completed")

	stopBlink(t, command, done)
	assert.Error(t, waitErr, "test-blink should have been terminated by a signal")

	stdout := output.String()
	assert.NoError(t, utils.WriteLogFile(t, snapMatterPiGPIO, stdout))

	assert.Contains(t, stdout, fmt.Sprintf("GPIO: %s", gpioLine))
	assert.Contains(t, stdout, fmt.Sprintf("GPIOCHIP: %s", gpioChip))
	assert.Contains(t, stdout, "On")
	assert.Contains(t, stdout, "Off")
}

func TestWifiMatterCommander(t *testing.T) {
	var stdout string
	var err error

	start := time.Now()
	t.Cleanup(func() {
		utils.SnapDumpLogs(t, start, snapMatterPiGPIO)
	})

	// install chip-tool
	err = utils.SnapInstallFromStore(t, chipToolSnap, "latest/beta")
	t.Cleanup(func() {
		utils.SnapDumpLogs(t, start, chipToolSnap)
		utils.SnapRemove(t, chipToolSnap)
	})
	if err != nil {
		t.Fatalf("Failed to install chip-tool: %s", err)
	}

	// chip-tool interfaces
	utils.SnapConnect(t, chipToolSnap+":avahi-observe", "")

	utils.SnapStart(t, snapMatterPiGPIO)
	t.Cleanup(func() {
		utils.SnapStop(t, snapMatterPiGPIO)
	})

	// commission
	t.Run("Commission", func(t *testing.T) {
		stdout, _, err = utils.Exec(t, "sudo chip-tool pairing onnetwork 110 20202021 2>&1")
		assert.NoError(t, utils.WriteLogFile(t, chipToolSnap, stdout))
		assert.NoError(t, err)
		assert.Contains(t, stdout, "[IN] TransportMgr initialized")
	})

	t.Run("Control", func(t *testing.T) {
		stdout = runChipTool(t, "onoff off 110 1")
		assert.Contains(t, stdout, "Success status report received")
		waitForGPIOState(t, "low")
		assert.Equal(t, "off", readOnOffAttribute(t))

		stdout = runChipTool(t, "onoff on 110 1")
		assert.Contains(t, stdout, "Success status report received")
		waitForGPIOState(t, "high")
		assert.Equal(t, "on", readOnOffAttribute(t))

		stdout = runChipTool(t, "onoff toggle 110 1")
		assert.Contains(t, stdout, "Success status report received")
		waitForGPIOState(t, "low")
		assert.Equal(t, "off", readOnOffAttribute(t))
	})

	/*
		The OnOff attribute is persisted, so restarting the service must not
		change the state of the light. Both the attribute and the GPIO line
		have to come back with the value they had before the restart.
	*/
	t.Run("RestartPersistence", func(t *testing.T) {
		// The light is left on, which differs from the power-up default, so
		// that a restart which fails to restore the state is detectable.
		stdout = runChipTool(t, "onoff on 110 1")
		assert.Contains(t, stdout, "Success status report received")
		waitForGPIOState(t, "high")

		utils.SnapRestart(t, snapMatterPiGPIO)
		assert.Eventually(t, func() bool {
			return utils.SnapServicesActive(t, snapMatterPiGPIO)
		}, 10*time.Second, 500*time.Millisecond)

		assert.Equal(t, "on", readOnOffAttribute(t),
			"the OnOff attribute must survive a restart")
		waitForGPIOState(t, "high")
	})
}

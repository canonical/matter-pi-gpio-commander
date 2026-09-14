package tests

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
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
	snapRevision     = "SNAP_REVISION"
)

const snapMatterPiGPIO = "matter-pi-gpio-commander"
const chipToolSnap = "chip-tool"

// Exit code used by the timeout command when it has to stop the program.
const timeoutExitCode = 124

// A line offset that exists on neither a Raspberry Pi nor the simulator.
const invalidGpioLine = "99"

var gpioChip = os.Getenv(specificGpioChip)
var gpioLine = os.Getenv(specificGpioLine)

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
		err = installSnapRevision(snapMatterPiGPIO, os.Getenv(snapRevision))
	}
	if err != nil {
		teardown()
		return
	}

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

func installSnapRevision(name, revision string) error {
	if _, err := strconv.ParseUint(revision, 10, 64); err != nil {
		return fmt.Errorf("%s must be a numeric Store revision: %q", snapRevision, revision)
	}

	var lastErr error
	for _, delay := range []time.Duration{0, 10 * time.Second, 30 * time.Second, 60 * time.Second} {
		time.Sleep(delay)

		output, err := exec.Command(
			"sudo", "snap", "install", name, "--revision="+revision,
		).CombinedOutput()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	return fmt.Errorf("install %s revision %s: %w", name, revision, lastErr)
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
			return fmt.Errorf("failed to get mock gpio chip number: %s", err)
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

// stopBlink terminates the test-blink process tree and returns the result of
// waiting for it. The processes are started through sudo and therefore run as
// root, so they cannot be signalled by the unprivileged test process itself.
func stopBlink(t *testing.T, command *exec.Cmd, waitResult <-chan error) error {
	t.Helper()

	if command.Process == nil {
		return nil
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
	case err := <-waitResult:
		return err
	case <-time.After(10 * time.Second):
	}

	signalGroup("KILL")
	select {
	case err := <-waitResult:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("test-blink did not exit after being killed")
		return nil
	}
}

// runBlinkExpectingFailure runs test-blink and requires it to exit with an
// error instead of blinking. The timeout prevents a job from hanging when the
// expected failure does not occur.
func runBlinkExpectingFailure(t *testing.T, reason string) string {
	t.Helper()

	var output bytes.Buffer
	command := exec.Command("sudo", "timeout", "5s",
		"snap", "run", snapMatterPiGPIO+".test-blink")
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()

	stdout := output.String()
	assert.NoError(t, utils.WriteLogFile(t, snapMatterPiGPIO, stdout))

	var exitErr *exec.ExitError
	if assert.ErrorAs(t, err, &exitErr, reason) {
		assert.NotEqual(t, timeoutExitCode, exitErr.ExitCode(),
			"test-blink kept running, but it should have failed: %s", reason)
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

	utils.SnapSet(t, snapMatterPiGPIO, "gpiochip", "0")
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

	assert.False(t, utils.SnapServicesEnabled(t, snapMatterPiGPIO))
	assert.False(t, utils.SnapServicesActive(t, snapMatterPiGPIO))
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
		_, _, err := utils.Exec(nil, "sudo snap disconnect "+
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

	waitResult := make(chan error, 1)
	go func() {
		waitResult <- command.Wait()
	}()

	exited := func() bool {
		select {
		case err := <-waitResult:
			// Put the result back for the caller of stopBlink.
			waitResult <- err
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

	waitErr := stopBlink(t, command, waitResult)
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

		stdout = runChipTool(t, "onoff read on-off 110 1")
		assert.Contains(t, stdout, "OnOff")

		stdout = runChipTool(t, "onoff on 110 1")
		assert.Contains(t, stdout, "Success status report received")
		waitForGPIOState(t, "high")

		stdout = runChipTool(t, "onoff toggle 110 1")
		assert.Contains(t, stdout, "Success status report received")
		waitForGPIOState(t, "low")
	})

	t.Run("RestartPersistence", func(t *testing.T) {
		utils.SnapRestart(t, snapMatterPiGPIO)
		assert.Eventually(t, func() bool {
			return utils.SnapServicesActive(t, snapMatterPiGPIO)
		}, 10*time.Second, 500*time.Millisecond)

		stdout = runChipTool(t, "onoff on 110 1")
		assert.Contains(t, stdout, "Success status report received")
		waitForGPIOState(t, "high")
	})
}

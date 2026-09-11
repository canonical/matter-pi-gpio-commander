package tests

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
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

var gpioChip = os.Getenv(specificGpioChip)
var gpioLine = os.Getenv(specificGpioLine)

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

	if err = utils.SnapConnect(nil, snapMatterPiGPIO+":avahi-control", ""); err != nil {
		teardown()
		return
	}

	if useGPIOMock() {
		stdout, _, mockErr := utils.Exec(nil, "./gpio-mock.sh 2>&1")
		_ = utils.WriteLogFile(nil, "gpio-mock", stdout)
		if mockErr != nil {
			teardown()
			return nil, fmt.Errorf("failed to set up gpio-sim: %w", mockErr)
		}
		if err = utils.SnapConnect(nil,
			snapMatterPiGPIO+":custom-gpio",
			snapMatterPiGPIO+":custom-gpio-dev"); err != nil {
			teardown()
			return
		}
	} else {
		if err = utils.SnapConnect(nil,
			snapMatterPiGPIO+":custom-gpio",
			snapMatterPiGPIO+":custom-gpio-dev"); err != nil {
			teardown()
			return
		}
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
	gpioChipNumber, stderr, err := utils.Exec(nil, "cat /tmp/matter-pi-gpio-sim-chip")
	if err != nil || stderr != "" {
		return "", fmt.Errorf("failed to get mock gpio chip number, Error %s: %s", stderr, err)
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
		fmt.Sprintf("./gpio-mock.sh state %s %s", gpioChip, gpioLine))
	if err != nil {
		return "", fmt.Errorf("read simulated GPIO state: %w: %s", err, stderr)
	}
	return strings.TrimSpace(stdout), nil
}

func waitForGPIOState(t *testing.T, expected string) {
	t.Helper()

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

func stopBlink(t *testing.T) {
	t.Helper()

	_, _, _ = utils.Exec(t,
		`sudo sh -c 'for pid in $(pgrep -f "/[b]in/test-blink" || true); do kill "$pid"; done'`)
	assert.Eventually(t, func() bool {
		_, _, err := utils.Exec(nil, `pgrep -f "/[b]in/test-blink"`)
		return err != nil
	}, 5*time.Second, 100*time.Millisecond)
}

func TestConfigurationValidation(t *testing.T) {
	if !useGPIOMock() {
		t.Skip("configuration restoration requires the simulated GPIO")
	}

	_, _, err := utils.Exec(nil, "sudo snap set "+snapMatterPiGPIO+" gpio=0")
	assert.Error(t, err)

	stdout, _, err := utils.Exec(t, "snap get "+snapMatterPiGPIO+" gpio")
	assert.NoError(t, err)
	assert.Equal(t, gpioLine, strings.TrimSpace(stdout))

	utils.SnapSet(t, snapMatterPiGPIO, "gpiochip", "0")
	utils.SnapSet(t, snapMatterPiGPIO, "gpiochip-validation", "true")
	_, _, err = utils.Exec(nil, "sudo snap set "+snapMatterPiGPIO+" gpiochip=7")
	assert.Error(t, err)

	utils.SnapSet(t, snapMatterPiGPIO, "gpiochip-validation", "false")
	utils.SnapSet(t, snapMatterPiGPIO, "gpiochip", gpioChip)
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
	if !useGPIOMock() {
		t.Skip("runtime recovery requires the simulated GPIO")
	}

	utils.SnapSet(t, snapMatterPiGPIO, "gpio", "99")
	stdout, _, err := utils.Exec(nil, "sudo snap run "+snapMatterPiGPIO+".test-blink 2>&1")
	assert.Error(t, err)
	assert.Contains(t, stdout, "Failed to request output line")
	utils.SnapSet(t, snapMatterPiGPIO, "gpio", gpioLine)
}

/*
TestBlinkOperation runs the test-blink app in the snap.
The log output is checked for the correctly configured GPIO Chip and GPIO Line,
as well as the existence of the ON and OFF log messages.
*/
func TestBlinkOperation(t *testing.T) {
	if !useGPIOMock() {
		t.Skip("GPIO state assertions require the simulator")
	}

	_, _, err := utils.Exec(nil, "sudo snap disconnect "+
		snapMatterPiGPIO+":custom-gpio "+
		snapMatterPiGPIO+":custom-gpio-dev")
	assert.NoError(t, err)

	stdout, _, err := utils.Exec(nil,
		"sudo timeout 5s snap run "+snapMatterPiGPIO+".test-blink 2>&1")
	assert.Error(t, err)
	assert.Contains(t, stdout, "Failed to request output line")
	stopBlink(t)
	assert.NoError(t, utils.SnapConnect(t,
		snapMatterPiGPIO+":custom-gpio",
		snapMatterPiGPIO+":custom-gpio-dev"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	var output bytes.Buffer
	command := exec.CommandContext(ctx, "sudo", "snap", "run", snapMatterPiGPIO+".test-blink")
	command.Stdout = &output
	command.Stderr = &output
	assert.NoError(t, command.Start())

	seen := map[string]bool{}
	assert.Eventually(t, func() bool {
		state, stateErr := gpioState()
		if stateErr == nil {
			seen[state] = true
		}
		return seen["high"] && seen["low"]
	}, 4*time.Second, 50*time.Millisecond)
	cancel()
	stopBlink(t)
	_ = command.Wait()

	stdout = output.String()
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

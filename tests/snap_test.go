package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	var newSnapPath string

	log.Println("[CLEAN]")
	utils.SnapRemove(nil, snapMatterPiGPIO)
	utils.SnapRemove(nil, chipToolSnap)

	log.Println("[SETUP]")

	teardown = func() {
		log.Println("[TEARDOWN]")

		utils.Exec(nil, "rm "+newSnapPath)

		log.Println("Removing installed snap:", env.Teardown())
		if env.Teardown() {
			utils.SnapRemove(nil, snapMatterPiGPIO)
			utils.Exec(nil, "./gpio-mock.sh teardown")
		}
	}

	// setup gpio mock
	if newSnapPath, err = setupGPIOMock(env.SnapPath()); err != nil {
		teardown()
		return
	}

	// install matter-pi-gpio-commander
	if env.SnapPath() != "" {
		err = utils.SnapInstallFromFile(nil, newSnapPath)
		utils.SnapConnect(nil, snapMatterPiGPIO+":custom-gpio", snapMatterPiGPIO+":custom-gpio-dev")
	} else {
		err = utils.SnapInstallFromStore(nil, snapMatterPiGPIO, env.SnapChannel())
	}

	utils.SnapConnect(nil, snapMatterPiGPIO+":avahi-control", "")
	utils.SnapConnect(nil, snapMatterPiGPIO+":bluez", "")

	if err != nil {
		teardown()
		return
	}

	if err = setupGPIO(); err != nil {
		teardown()
		return
	}

	return
}

func setupGPIOMock(snapPath string) (string, error) {
	if !useGPIOMock() || (env.SnapPath() == "") {
		return snapPath, nil
	}

	// check if gpio mock is enabled AND the service is running locally
	// Run gpio mockup script
	_, stderr, err := utils.Exec(nil, "./gpio-mock.sh")
	if err != nil {
		return snapPath, fmt.Errorf("Failed to run gpio mockup script %s: %s", stderr, err)
	}

	// authorize gpio mock
	newPath, err := authorizeGpioMock(snapPath)
	if err != nil {
		return snapPath, err
	}

	return newPath, nil
}

func useGPIOMock() bool {
	return os.Getenv(gpioChipMock) == "true"
}

func getMockGPIO() (string, error) {
	gpioChipNumber, stderr, err := utils.Exec(nil, "ls /dev/gpiochip* | sort -n | tail -n 1 | cut -d'p' -f3")
	if err != nil || stderr != "" {
		return "", fmt.Errorf("failed to get mock gpio chip number, Error %s: %s", stderr, err)
	}
	gpioChipNumber = strings.TrimSpace(gpioChipNumber)
	gpioChipNumber = strings.Trim(gpioChipNumber, "\n")
	return gpioChipNumber, nil
}

func authorizeGpioMock(path string) (string, error) {
	const sedAuthorizeMockRead = `sed -i '/\/sys\/devices\/platform\/axi\/\*.pcie\/\*.gpio\/gpiochip4\/dev/a \      - /sys/devices/platform/gpio-mockup.*/gpiochip*/dev' squashfs-root/meta/snap.yaml`
	const sedAuthorizeMockGPIOsChips = `sed -i 's/gpiochip[0,4]/gpiochip\*/' squashfs-root/meta/snap.yaml `

	utils.Exec(nil, "rm -rf squashfs-root")

	_, stderr, err := utils.Exec(nil, "unsquashfs "+path)
	if err != nil || stderr != "" {
		log.Printf("stderr: %s", stderr)
		log.Printf("err: %s", err)
		return "", err
	}

	_, stderr, err = utils.Exec(nil, sedAuthorizeMockRead)
	if err != nil || stderr != "" {
		log.Printf("stderr: %s", stderr)
		log.Printf("err: %s", err)
		return "", err
	}

	_, stderr, err = utils.Exec(nil, sedAuthorizeMockGPIOsChips)
	if err != nil || stderr != "" {
		log.Printf("stderr: %s", stderr)
		log.Printf("err: %s", err)
		return "", err
	}

	directory := filepath.Dir(path)
	newPath := directory + "/mod_matter-pi-gpio-commander.snap"
	_, stderr, err = utils.Exec(nil, "mksquashfs squashfs-root "+newPath)
	if err != nil || stderr != "" {
		log.Printf("stderr: %s", stderr)
		log.Printf("err: %s", err)
		return "", err
	}

	utils.Exec(nil, "rm -rf squashfs-root")
	return newPath, nil
}

func setupGPIO() error {
	var err error
	// The GPIO_MOCKUP takes precedence over the specific GPIO_CHIP and GPIO_LINE
	if useGPIOMock() && (env.SnapPath() != "") {
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

func TestBlinkOperation(t *testing.T) {
	/*
		`test-blink` runs until it is stopped.
		1. We use a context with a timeout to stop it after 5 seconds.
		2. This does not work in a GitHub runner, so we also do a force kill after 10 seconds.
		   See issue https://github.com/canonical/matter-snap-testing/issues/17
	*/

	// Create context with 5 second timeout. Clear its resources when test is cleaned up.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(func() {
		cancel()
	})

	// Schedule a kill after 10 seconds
	go func() {
		time.Sleep(10 * time.Second)
		utils.Exec(t, `sudo pkill -f "test-blink"`)
	}()

	// Start blink, capturing stdout until it exits. Exit happens on the context timeout, or as fallback when it is killed.
	stdout, _, _ := utils.ExecContextVerbose(t, ctx, "sudo "+snapMatterPiGPIO+".test-blink")
	assert.NoError(t, utils.WriteLogFile(t, snapMatterPiGPIO, stdout))

	// Assert GPIO value
	assert.Contains(t, stdout, fmt.Sprintf("GPIO: %s", gpioLine))
	// Assert GPIOCHIP value
	assert.Contains(t, stdout, fmt.Sprintf("GPIOCHIP: %s", gpioChip))
	// Assert log messages
	assert.Contains(t, stdout, "On")
	assert.Contains(t, stdout, "Off")
}

func TestWifiMatterCommander(t *testing.T) {
	var stdout, stderr string
	var err error

	start := time.Now()

	t.Cleanup(func() {
		utils.SnapStop(t, snapMatterPiGPIO)

		// Remove snaps, ignore errors during removal
		utils.SnapRemove(nil, chipToolSnap)
		// snapMatterPiGPIO is not removed here, as that is handled by the teardown function

		utils.SnapDumpLogs(t, start, snapMatterPiGPIO)
	})

	// install chip-tool
	err = utils.SnapInstallFromStore(t, chipToolSnap, "latest/beta")
	if err != nil {
		t.Fatalf("Failed to install chip-tool: %s", err)
	}

	// chip-tool interfaces
	utils.SnapConnect(nil, chipToolSnap+":avahi-observe", "")

	utils.SnapStart(t, snapMatterPiGPIO)

	// commission
	t.Run("Commission", func(t *testing.T) {
		stdout, stderr, err = utils.Exec(t, "sudo chip-tool pairing onnetwork 110 20202021")
		assert.NoError(t, utils.WriteLogFile(t, chipToolSnap, stdout))
		assert.Contains(t, stdout, "CHIP:IN: TransportMgr initialized")
		t.Logf("stderr: %s", stderr)
	})

	t.Run("Control", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			stdout, stderr, err = utils.Exec(t, "sudo chip-tool onoff toggle 110 1")
		}
		assert.NoError(t, utils.WriteLogFile(t, chipToolSnap, stdout))
		assert.Contains(t, stdout, "Success status report received. Session was established")
	})

}

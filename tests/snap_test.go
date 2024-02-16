package tests

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/canonical/matter-snap-testing/utils"
	"github.com/stretchr/testify/assert"
)

const (
	// enviroment variables
	specificGpioChip = "GPIO_CHIP"
	specificGpioLine = "GPIO_LINE"
	gpioChipMock     = "USE_GPIO_MOCK"
)

const snapMatterPiGPIO = "matter-pi-gpio-commander"
const chipToolSnap = "chip-tool"
const sedMockGPIOAuthorization = `sed -i '/\/sys\/devices\/platform\/axi\/\*.pcie\/\*.gpio\/gpiochip4\/dev/a \      - /sys/devices/platform/gpio-mockup.*/gpiochip*/dev' squashfs-root/meta/snap.yaml`

var start = time.Now()
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
	var newPath string

	log.Println("[CLEAN]")
	utils.SnapRemove(nil, snapMatterPiGPIO)
	utils.SnapRemove(nil, chipToolSnap)

	log.Println("[SETUP]")

	teardown = func() {
		log.Println("[TEARDOWN]")
		utils.SnapDumpLogs(nil, start, snapMatterPiGPIO)

		utils.Exec(nil, "rm "+newPath)

		log.Println("Removing installed snap:", !utils.SkipTeardownRemoval)
		if !utils.SkipTeardownRemoval {
			utils.SnapRemove(nil, snapMatterPiGPIO)
			utils.SnapRemove(nil, chipToolSnap)
			utils.Exec(nil, "./gpio-mock.sh teardown")
		}
	}

	// setup gpio mock
	if newPath, err = setupGPIOMock(utils.LocalServiceSnapPath); err != nil {
		teardown()
		return
	}

	// install matter-pi-gpio-commander
	if utils.LocalServiceSnap() {
		err = utils.SnapInstallFromFile(nil, newPath)
	} else {
		err = utils.SnapInstallFromStore(nil, snapMatterPiGPIO, utils.ServiceChannel)
	}
	if err != nil {
		teardown()
		return
	}

	if err = setupGPIO(); err != nil {
		teardown()
		return
	}

	// connect interfaces:
	utils.SnapConnect(nil, snapMatterPiGPIO+":avahi-control", "")
	utils.SnapConnect(nil, snapMatterPiGPIO+":bluez", "")
	utils.SnapConnect(nil, snapMatterPiGPIO+":network", "")
	utils.SnapConnect(nil, snapMatterPiGPIO+":network-bind", "")
	utils.SnapConnect(nil, snapMatterPiGPIO+":custom-gpio", snapMatterPiGPIO+":custom-gpio-dev")

	return
}

func setupGPIOMock(snapPath string) (string, error) {
	if !useGPIOMock() || !utils.LocalServiceSnap() {
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
	return gpioChipNumber, nil
}

func authorizeGpioMock(path string) (string, error) {
	_, stderr, err := utils.Exec(nil, "unsquashfs "+path)
	if err != nil || stderr != "" {
		log.Printf("stderr: %s", stderr)
		log.Printf("err: %s", err)
		return "", err
	}

	_, stderr, err = utils.Exec(nil, sedMockGPIOAuthorization)
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

	// The GPIO_MOCKUP takes precedence over the specific GPIO_CHIP and GPIO_LINE
	if useGPIOMock() && utils.LocalServiceSnap() {
		gpioChip, err := getMockGPIO()
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
	// test blink operation
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()

	stdout, _, err := utils.ExecContext(nil, ctx, snapMatterPiGPIO+".test-blink")
	assert.NoError(t, err)

	// Assert GPIO value
	assert.Contains(t, stdout, fmt.Sprintf("GPIO: %s", gpioLine))

	// Assert GPIOCHIP value
	assert.Contains(t, stdout, fmt.Sprintf("GPIOCHIP: %s", gpioChip))

	// Assert log messages
	assert.Contains(t, stdout, "On")
	assert.Contains(t, stdout, "Off")
}

func TestWifiMatterCommander(t *testing.T) {

	// install chip-tool
	err := utils.SnapInstallFromStore(t, chipToolSnap, utils.ServiceChannel)
	if err != nil {
		t.Fatalf("Failed to install chip-tool: %s", err)
	}
	// chip-tool interfaces
	utils.SnapConnect(nil, chipToolSnap+":avahi-observe", "")

	utils.SnapStart(t, snapMatterPiGPIO)

	time.Sleep(1 * time.Minute)

	t.Run("Commission", func(t *testing.T) {
		utils.Exec(t, "sudo chip-tool pairing onnetwork 110 20202021")
	})

	time.Sleep(1 * time.Minute)
	t.Run("Control", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			utils.Exec(t, "sudo chip-tool onoff toggle 110 1")
		}
	})
}

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
	var start = time.Now()
	var newSnapPath string

	log.Println("[CLEAN]")
	utils.SnapRemove(nil, snapMatterPiGPIO)
	utils.SnapRemove(nil, chipToolSnap)

	log.Println("[SETUP]")

	teardown = func() {
		log.Println("[TEARDOWN]")
		utils.SnapDumpLogs(nil, start, snapMatterPiGPIO)

		utils.Exec(nil, "rm "+newSnapPath)

		log.Println("Removing installed snap:", !utils.SkipTeardownRemoval)
		if !utils.SkipTeardownRemoval {
			utils.SnapRemove(nil, snapMatterPiGPIO)
			utils.Exec(nil, "./gpio-mock.sh teardown")
		}
	}

	// setup gpio mock
	if newSnapPath, err = setupGPIOMock(utils.LocalServiceSnapPath); err != nil {
		teardown()
		return
	}

	// install matter-pi-gpio-commander
	if utils.LocalServiceSnap() {
		err = utils.SnapInstallFromFile(nil, newSnapPath)
		utils.SnapConnect(nil, snapMatterPiGPIO+":custom-gpio", snapMatterPiGPIO+":custom-gpio-dev")
	} else {
		err = utils.SnapInstallFromStore(nil, snapMatterPiGPIO, utils.ServiceChannel)
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
	if !useGPIOMock() || !utils.LocalServiceSnap() {
		return snapPath, nil
	}

	// check if gpio mock is enabled AND the service is running locally
	// Run gpio mockup script
	_, stderr, err := utils.ExecVerbose(nil, "./gpio-mock.sh")
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
	if useGPIOMock() && utils.LocalServiceSnap() {
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

func TestMockingGPIO(t *testing.T) {
	if !useGPIOMock() {
		t.Skip("Skipping GPIO Mock test")
	}
	utils.ExecVerbose(t, "lsmod | head")
	utils.ExecVerbose(t, "ls -la /dev/gpio*")
	utils.ExecVerbose(t, "sudo gpiodetect")
	utils.ExecVerbose(t, "sudo dmesg | grep gpio")
	utils.ExecVerbose(t, "ls /lib/modules/$(uname -r)/kernel/drivers/")
	utils.ExecVerbose(t, "sudo cat /boot/config-$(uname -r) | grep -E 'IRQ|GPIO'")
	utils.ExecVerbose(t, "apt search gpiod | grep installed")
	utils.ExecVerbose(t, "sudo gpioinfo")
	utils.ExecVerbose(t, "sudo ls -la /sys/kernel/debug/gpio-mockup/gpiochip"+gpioChip)

	// test gpio mock
	for i := 0; i < 4; i++ {
		utils.ExecVerbose(t, "sudo gpioset "+gpioChip+" "+gpioLine+"=1")
		time.Sleep(5 * time.Second)
		utils.ExecVerbose(t, "sudo gpioset "+gpioChip+" "+gpioLine+"=0")
	}
}

func TestBlinkOperation(t *testing.T) {
	// test blink operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Force kill the process see issue https://github.com/canonical/matter-snap-testing/issues/17
	go func() {
		time.Sleep(10 * time.Second)
		utils.Exec(t, `sudo pkill -f "/snap/matter-pi-gpio-commander/x1/bin/test-blink"`)
	}()

	stdout, _, _ := utils.ExecContextVerbose(t, ctx, "sudo "+snapMatterPiGPIO+".test-blink")
	t.Cleanup(cancel)

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

	// install chip-tool
	err = utils.SnapInstallFromStore(t, chipToolSnap, utils.ServiceChannel)
	if err != nil {
		t.Fatalf("Failed to install chip-tool: %s", err)
	}
	t.Cleanup(func() { utils.SnapRemove(t, chipToolSnap) })

	// chip-tool interfaces
	utils.SnapConnect(nil, chipToolSnap+":avahi-observe", "")

	utils.SnapStart(t, snapMatterPiGPIO)
	t.Cleanup(func() { utils.SnapStop(t, snapMatterPiGPIO) })

	// commission
	t.Run("Commission", func(t *testing.T) {
		stdout, stderr, err = utils.Exec(t, "sudo chip-tool pairing onnetwork 110 20202021")
		assert.Contains(t, stdout, "CHIP:IN: TransportMgr initialized")
		t.Logf("stderr: %s", stderr)
	})

	t.Run("Control", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			stdout, stderr, err = utils.Exec(t, "sudo chip-tool onoff toggle 110 1")
		}
		assert.Contains(t, stdout, "Success status report received. Session was established")
	})
}

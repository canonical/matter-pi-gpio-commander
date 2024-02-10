package tests

import (
	"context"
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
	specifGpioLine   = "GPIO_LINE"
)

var start = time.Now()

const snapName = "matter-pi-gpio-commander"
const chipToolSnap = "chip-tool"

func TestMain(m *testing.M) {
	teardown, err := setup() // THIS NEED TO receive "t" but this function doen't have access to it TODO: fix this latter
	if err != nil {
		log.Fatalf("Failed to setup tests: %s", err)
	}

	code := m.Run()
	teardown()

	os.Exit(code)
}

const sedCommand = `sed -i 's|/sys/devices/platform/axi/\*.pcie/\*.gpio/gpiochip4/dev|/sys/devices/platform/axi/*.pcie/*.gpio/gpiochip4/dev\n      - /sys/devices/platform/gpio-mockup.*/gpiochip*/dev|' squashfs-root/meta/snap.yaml`

func authorizeGpioMock(path string) (string, error) {
	_, stderr, err := utils.Exec(nil, "unsquashfs "+path)
	if err != nil || stderr != "" {
		log.Printf("stderr: %s", stderr)
		log.Printf("err: %s", err)
		return "", err
	}

	_, stderr, err = utils.Exec(nil, sedCommand)
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

func setup() (teardown func(), err error) {
	var newPath string

	log.Println("[CLEAN]")
	utils.SnapRemove(nil, snapName)

	log.Println("[SETUP]")

	teardown = func() {
		log.Println("[TEARDOWN]")
		utils.SnapDumpLogs(nil, start, snapName)

		utils.Exec(nil, "rm "+newPath)

		log.Println("Removing installed snap:", !utils.SkipTeardownRemoval)
		if !utils.SkipTeardownRemoval {
			utils.SnapRemove(nil, snapName)
		}
	}

	// authorize gpio mock
	if utils.LocalServiceSnap() {
		newPath, err = authorizeGpioMock(utils.LocalServiceSnapPath)
	}
	if err != nil {
		teardown()
		return
	}

	// install matter-pi-gpio-commander
	if utils.LocalServiceSnap() {
		err = utils.SnapInstallFromFile(nil, newPath)
	} else {
		err = utils.SnapInstallFromStore(nil, snapName, utils.ServiceChannel)
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
	utils.SnapConnect(nil, snapName+":avahi-control", "")
	utils.SnapConnect(nil, snapName+":bluez", "")
	utils.SnapConnect(nil, snapName+":network", "")
	utils.SnapConnect(nil, snapName+":network-bind", "")
	utils.SnapConnect(nil, snapName+":custom-gpio", snapName+":custom-gpio-dev")

	return
}

func getMockGPIO() (string, error) {
	gpioChipNumber, stderr, err := utils.Exec(nil, "ls /dev/gpiochip* | sort -n | tail -n 1 | cut -d'p' -f3")
	if err != nil || stderr != "" {
		log.Printf("Failed to get mock gpio chip number, Error %s: %s", stderr, err)
		return "", err
	}
	return gpioChipNumber, nil
}

func setupGPIO() error {

	TestedGpioChip := os.Getenv(specificGpioChip)
	TestedGpioLine := os.Getenv(specifGpioLine)

	gpiochip := TestedGpioChip
	var err error
	if gpiochip == "" {
		gpiochip, err = getMockGPIO()
		if err != nil {
			log.Printf("Failed to get mock gpio chip number: %s", err)
			return err
		}

		log.Printf("[TEST] No specific gpiochip defined, using mockup gpio: /dev/gpiochip%s", gpiochip)
	}

	gpioline := TestedGpioLine
	if gpioline == "" {
		gpioline = "4"
		log.Printf("[TEST] No specific gpio line defined, using default: %s", gpioline)
	}

	utils.SnapSet(nil, snapName, "gpiochip", gpiochip)
	utils.SnapSet(nil, snapName, "gpio", gpioline)

	return nil
}

func TestBlinkOperation(t *testing.T) {
	log.Println("[TEST] Standard blink operation")

	// test blink operation
	t.Run("test-blink", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), <-time.After(20*time.Second))
		defer cancel()

		stdout, _, err := utils.ExecContextVerbose(nil, ctx, snapName+".test-blink")
		t.Logf("err: %s", err)
		t.Logf("stdout: %s", stdout)
		assert.Error(t, err, "Expected an error")
		assert.Equal(t, "context deadline exceeded", err.Error())
	})
}

func TestWifiCommander(t *testing.T) {
	log.Println("[TEST] Wifi commander")

	// install chip-tool
	err := utils.SnapInstallFromStore(t, chipToolSnap, utils.ServiceChannel)
	if err != nil {
		t.Fatalf("Failed to install chip-tool: %s", err)
	}
	// chip-tool interfaces
	utils.SnapConnect(nil, chipToolSnap+":avahi-observe", "")

	utils.SnapStart(t, snapName)

	// comission chip-tool
	t.Run("Commission", func(t *testing.T) {
		utils.Exec(t, "sudo chip-tool pairing onnetwork 110 20202021")
	})

	t.Run("Control", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			utils.Exec(t, "sudo chip-tool onoff toggle 110 1")
		}
	})
}

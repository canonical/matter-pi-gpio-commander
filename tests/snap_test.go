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
	SpecificGpioChip = "GPIO_CHIP"
	SpecifGpioLine   = "GPIO_LINE"
)

var (
	TestedGpioChip = ""
	TestedGpioLine = ""
)

func init() {
	TestedGpioChip = os.Getenv(SpecificGpioChip)
	TestedGpioLine = os.Getenv(SpecifGpioLine)
}

var start = time.Now()

const snapName = "matter-pi-gpio-commander"

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

	log.Println("[CLEAN]")
	utils.SnapRemove(nil, snapName)

	log.Println("[SETUP]")

	teardown = func() {
		log.Println("[TEARDOWN]")
		utils.SnapDumpLogs(nil, start, snapName)

		log.Println("Removing installed snap:", !utils.SkipTeardownRemoval)
		if !utils.SkipTeardownRemoval {
			utils.SnapRemove(nil, snapName)
		}
	}

	var newPath string
	if utils.LocalServiceSnap() {
		newPath, err = authorizeGpioMock(utils.LocalServiceSnapPath)
	}
	if err != nil {
		teardown()
		return
	}

	if utils.LocalServiceSnap() {
		err = utils.SnapInstallFromFile(nil, newPath)
	} else {
		err = utils.SnapInstallFromStore(nil, snapName, utils.ServiceChannel)
	}
	if err != nil {
		teardown()
		return
	}

	// connect interfaces
	utils.SnapConnect(nil, snapName+":avahi-control", "")
	utils.SnapConnect(nil, snapName+":bluez", "")
	utils.SnapConnect(nil, snapName+":network", "")
	utils.SnapConnect(nil, snapName+":network-bind", "")
	utils.SnapConnect(nil, snapName+":custom-gpio", snapName+":custom-gpio-dev")
	return
}

func getMockGPIO(t *testing.T) string {
	t.Helper()
	gpioChipNumber, stderr, err := utils.Exec(t, "ls /dev/gpiochip* | sort -n | tail -n 1 | cut -d'p' -f3")
	if err != nil || stderr != "" {
		t.Fatalf("Failed to get mock gpio chip number, Error %s: %s", stderr, err)
	}
	return gpioChipNumber
}

func TestBlinkOperation(t *testing.T) {
	log.Println("[TEST] Standard blink operation")

	gpiochip := TestedGpioChip
	if gpiochip == "" {
		gpiochip = getMockGPIO(t)
		log.Printf("[TEST] No specific gpiochip defined, using mockup gpio: /dev/gpiochip%s", gpiochip)
	}

	gpioline := TestedGpioLine
	if gpioline == "" {
		gpioline = "4"
		log.Printf("[TEST] No specific gpio line defined, using default: %s", gpioline)
	}

	utils.SnapSet(t, snapName, "gpiochip", gpiochip)
	utils.SnapSet(t, snapName, "gpio", gpioline)

	// test blink operation

	t.Run("test-blink", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), <-time.After(10*time.Second))
		defer cancel()

		_, _, err := utils.ExecContextVerbose(nil, ctx, snapName+".test-blink")
		t.Logf("err: %s", err)
		assert.Error(t, err, "Expected an error")
		assert.Equal(t, "context deadline exceeded", err.Error())
	})
}

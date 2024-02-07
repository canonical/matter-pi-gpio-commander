package tests

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/canonical/matter-snap-testing/utils"
)

var start = time.Now()

const snapName = "matter-pi-gpio-commander"

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

	if utils.LocalServiceSnap() {
		err = utils.SnapInstallFromFile(nil, utils.LocalServiceSnapPath)
	} else {
		err = utils.SnapInstallFromStore(nil, snapName, utils.ServiceChannel)
	}
	if err != nil {
		teardown()
		return
	}

	// connect interfaces
	utils.SnapConnect(nil, snapName+":avahi-observe", "")
	utils.SnapConnect(nil, snapName+":bluez", "")
	utils.SnapConnect(nil, snapName+":process-control", "")
	utils.SnapConnect(nil, snapName+":custom-gpio", snapName+":custom-gpio-dev")
	return
}

func getMockGPIO(t *testing.T) (string, error) {
	gpioChipNumber, stderr, err := utils.Exec(t, "ls /dev/gpiochip* | sort -n | tail -n 1 | cut -d'p' -f3")
	if err != nil || stderr != "" {
		msg := fmt.Sprintf("ERROR %s: %s", err, stderr)
		return "", errors.New(msg)
	}
	return gpioChipNumber, nil
}

func TestBlinkOperation(t *testing.T) {

	gpiochip, err := getMockGPIO(t)
	if err != nil {
		t.Fatal("Error when getting the gpio-mockup chip number: %s", err)
	}

	log.Println("[TEST] Standard blink operation")
	utils.SnapSet(t, snapName, "gpiochip", gpiochip)
	utils.SnapSet(t, snapName, "gpio", "4")

	// Run the context in background
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute) // Adjust the timeout as needed
	defer cancel()

	go func() {
		utils.Exec(t, snapName+".test-blink")
		cancel()
	}()

	<-ctx.Done()

	utils.Exec(t, snapName+".test-blink")
	// Check if the context was done due to a timeout or because Exec returned
	if ctx.Err() == context.DeadlineExceeded {
		t.Log("Test did not complete within the expected time frame.")
		t.FailNow()
	} else {
		t.Log("Test completed within the expected time frame.")
	}
}

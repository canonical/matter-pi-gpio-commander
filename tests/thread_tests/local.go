package thread_tests

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/canonical/matter-snap-testing/utils"
	"github.com/stretchr/testify/require"
)

const (
	otbrSnap = "openthread-border-router"
	OTCTL    = otbrSnap + ".ot-ctl"

	snapMatterPiGPIO = "matter-pi-gpio-commander"
	chipToolSnap     = "chip-tool"
)

func setup(t *testing.T) {
	installGPIOCommander(t)

	const (
		defaultInfraInterfaceValue = "wlan0"
		infraInterfaceKey          = "infra-if"
		localInfraInterfaceEnv     = "LOCAL_INFRA_IF"
	)

	// Clean
	utils.SnapRemove(t, otbrSnap)
	utils.SnapRemove(t, snapMatterPiGPIO)

	// Install OTBR
	utils.SnapInstallFromStore(t, otbrSnap, utils.ServiceChannel)
	t.Cleanup(func() {
		utils.SnapRemove(t, otbrSnap)
	})

	// Connect interfaces
	snapInterfaces := []string{"avahi-control", "firewall-control", "raw-usb", "network-control", "bluetooth-control", "bluez"}
	for _, interfaceSlot := range snapInterfaces {
		utils.SnapConnect(nil, otbrSnap+":"+interfaceSlot, "")
	}

	// Set infra interface
	if v := os.Getenv(localInfraInterfaceEnv); v != "" {
		infraInterfaceValue := v
		utils.SnapSet(nil, otbrSnap, infraInterfaceKey, infraInterfaceValue)
	} else {
		utils.SnapSet(nil, otbrSnap, infraInterfaceKey, defaultInfraInterfaceValue)
	}

	// Start OTBR
	start := time.Now()
	utils.SnapStart(t, otbrSnap)
	waitForLogMessage(t, otbrSnap, "Start Thread Border Agent: OK", start)
}

func GetActiveDataset(t *testing.T) string {
	activeDataset, _, _ := utils.Exec(t, "sudo "+OTCTL+" dataset active -x | awk '{print $NF}' | grep --invert-match \"Done\"")
	trimmedActiveDataset := strings.TrimSpace(activeDataset)

	return trimmedActiveDataset
}

func installGPIOCommander(t *testing.T) {
	const snapMatterPiGPIO = "matter-pi-gpio-commander"

	// clean
	utils.SnapRemove(t, snapMatterPiGPIO)

	if utils.LocalServiceSnap() {
		require.NoError(t,
			utils.SnapInstallFromFile(nil, utils.LocalServiceSnapPath),
		)
		require.NoError(t,
			utils.SnapConnect(nil, snapMatterPiGPIO+":custom-gpio", snapMatterPiGPIO+":custom-gpio-dev"),
		)
	} else {
		require.NoError(t,
			utils.SnapInstallFromStore(nil, snapMatterPiGPIO, utils.ServiceChannel),
		)
	}
	t.Cleanup(func() {
		utils.SnapRemove(t, snapMatterPiGPIO)
	})

	// connect interfaces
	utils.SnapConnect(t, snapMatterPiGPIO+":avahi-control", "")
	// utils.SnapConnect(t, snapMatterPiGPIO+":bluez", "")
	utils.SnapConnect(t, snapMatterPiGPIO+":otbr-dbus-wpan0", otbrSnap+":dbus-wpan0")
	utils.SnapSet(t, snapMatterPiGPIO, "args", "--thread")
	utils.SnapSet(t, snapMatterPiGPIO, "gpio", "16")

}

// TODO: update the library function to print the tail before failing:
// https://github.com/canonical/matter-snap-testing/blob/abae29ac5e865f0c5208350bdab63cecb3bdcc5a/utils/config.go#L54-L69
func waitForLogMessage(t *testing.T, snap, expectedLog string, since time.Time) {
	const maxRetry = 10

	for i := 1; i <= maxRetry; i++ {
		time.Sleep(1 * time.Second)
		t.Logf("Retry %d/%d: Waiting for expected content in logs: %s", i, maxRetry, expectedLog)

		logs := utils.SnapLogs(t, since, snap)
		if strings.Contains(logs, expectedLog) {
			t.Logf("Found expected content in logs: %s", expectedLog)
			return
		}
	}

	t.Logf("Time out: reached max %d retries.", maxRetry)
	stdout, _, _ := utils.Exec(t,
		fmt.Sprintf("sudo journalctl --lines=10 --no-pager --unit=snap.\"%s\".otbr-agent --priority=notice", snap))
	t.Log(stdout)
	t.FailNow()
}

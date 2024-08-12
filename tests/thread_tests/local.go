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

func setup(t *testing.T) {
	installChipTool(t)

	// Clean
	utils.SnapRemove(t, otbrSnap)

	// Install OTBR
	utils.SnapInstallFromStore(t, otbrSnap, "latest/beta")
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

	// Set radio url
	if v := os.Getenv(localRadioUrlEnv); v != "" {
		radioUrlValue := v
		utils.SnapSet(nil, otbrSnap, radioUrlKey, radioUrlValue)
	} else {
		utils.SnapSet(nil, otbrSnap, radioUrlKey, defaultRadioUrl)
	}

	// Start OTBR
	start := time.Now()
	utils.SnapStart(t, otbrSnap)
	waitForLogMessage(t, otbrSnap, "Start Thread Border Agent: OK", start)

	// Form Thread network
	utils.Exec(t, "sudo "+OTCTL+" dataset init new")
	utils.Exec(t, "sudo "+OTCTL+" dataset commit active")
	utils.Exec(t, "sudo "+OTCTL+" ifconfig up")
	utils.Exec(t, "sudo "+OTCTL+" thread start")
	utils.WaitForLogMessage(t, otbrSnap, "Thread Network", start)
}

func getActiveDataset(t *testing.T) string {
	activeDataset, _, _ := utils.Exec(t, "sudo "+OTCTL+" dataset active -x | awk '{print $NF}' | grep --invert-match \"Done\"")
	trimmedActiveDataset := strings.TrimSpace(activeDataset)

	return trimmedActiveDataset
}

func installChipTool(t *testing.T) {
	const chipToolSnap = "chip-tool"

	// clean
	utils.SnapRemove(t, chipToolSnap)

	require.NoError(t,
		utils.SnapInstallFromStore(nil, chipToolSnap, "latest/beta"),
	)

	t.Cleanup(func() {
		utils.SnapRemove(t, chipToolSnap)
	})

	// connect interfaces
	utils.SnapConnect(t, chipToolSnap+":avahi-observe", "")
	utils.SnapConnect(t, chipToolSnap+":bluez", "")
	utils.SnapConnect(t, chipToolSnap+":process-control", "")

	return
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

package thread_tests

import (
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
	otbrInstallTime := time.Now()
	utils.SnapInstallFromStore(t, otbrSnap, "latest/beta")
	t.Cleanup(func() {
		logs := utils.SnapLogs(t, otbrInstallTime, otbrSnap)
		utils.WriteLogFile(t, otbrSnap, logs)
		utils.SnapRemove(t, otbrSnap)
	})

	// Connect interfaces
	snapInterfaces := []string{"avahi-control", "firewall-control", "raw-usb", "network-control", "bluetooth-control", "bluez"}
	for _, interfaceSlot := range snapInterfaces {
		utils.SnapConnect(t, otbrSnap+":"+interfaceSlot, "")
	}

	// Set infra interface
	if infraInterfaceValue := os.Getenv(localInfraInterfaceEnv); infraInterfaceValue != "" {
		utils.SnapSet(t, otbrSnap, infraInterfaceKey, infraInterfaceValue)
	} else {
		utils.SnapSet(t, otbrSnap, infraInterfaceKey, defaultInfraInterfaceValue)
	}

	// Set radio url
	if radioUrlValue := os.Getenv(localRadioUrlEnv); radioUrlValue != "" {
		utils.SnapSet(t, otbrSnap, radioUrlKey, radioUrlValue)
	} else {
		utils.SnapSet(t, otbrSnap, radioUrlKey, defaultRadioUrl)
	}

	// Start OTBR
	otbrStartTime := time.Now()
	utils.SnapStart(t, otbrSnap)
	utils.WaitForLogMessage(t, otbrSnap, "Start Thread Border Agent: OK", otbrStartTime)

	// Form Thread network
	utils.Exec(t, "sudo "+OTCTL+" dataset init new")
	utils.Exec(t, "sudo "+OTCTL+" dataset commit active")
	utils.Exec(t, "sudo "+OTCTL+" ifconfig up")
	utils.Exec(t, "sudo "+OTCTL+" thread start")
	utils.WaitForLogMessage(t, otbrSnap, "Thread Network", otbrStartTime)
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

	chipToolInstallTime := time.Now()
	require.NoError(t,
		utils.SnapInstallFromStore(t, chipToolSnap, "latest/beta"),
	)

	t.Cleanup(func() {
		logs := utils.SnapLogs(t, chipToolInstallTime, chipToolSnap)
		utils.WriteLogFile(t, chipToolSnap, logs)
		utils.SnapRemove(t, chipToolSnap)
	})

	// connect interfaces
	utils.SnapConnect(t, chipToolSnap+":avahi-observe", "")
	utils.SnapConnect(t, chipToolSnap+":bluez", "")
	utils.SnapConnect(t, chipToolSnap+":process-control", "")

	return
}

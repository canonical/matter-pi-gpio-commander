package thread_tests

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	remoteUser           = ""
	remotePassword       = ""
	remoteHost           = ""
	remoteInfraInterface = ""
	remoteSnapPath       = ""
	remoteGPIOChip       = ""
	remoteGPIOLine       = ""

	SSHClient *ssh.Client
)

const matterGPIOSnap = "matter-pi-gpio-commander"

func remote_setup(t *testing.T) {
	remote_loadEnvVars()

	connectSSH(t)

	remote_deployOTBRAgent(t)

	remote_deployGPIOCommander(t)
}

func remote_loadEnvVars() {
	const (
		remoteUserEnv           = "REMOTE_USER"
		remotePasswordEnv       = "REMOTE_PASSWORD"
		remoteHostEnv           = "REMOTE_HOST"
		remoteInfraInterfaceEnv = "REMOTE_INFRA_IF"
		remoteSnapPathEnv       = "REMOTE_SNAP_PATH"
		remoteGPIOChipEnv       = "REMOTE_GPIO_CHIP"
		remoteGPIOLineEnv       = "REMOTE_GPIO_LINE"
	)

	if v := os.Getenv(remoteUserEnv); v != "" {
		remoteUser = v
	}

	if v := os.Getenv(remotePasswordEnv); v != "" {
		remotePassword = v
	}

	if v := os.Getenv(remoteHostEnv); v != "" {
		remoteHost = v
	}

	if v := os.Getenv(remoteInfraInterfaceEnv); v != "" {
		remoteInfraInterface = v
	}

	if v := os.Getenv(remoteSnapPathEnv); v != "" {
		remoteSnapPath = v
	}

	if v := os.Getenv(remoteGPIOChipEnv); v != "" {
		remoteGPIOChip = v
	}

	if v := os.Getenv(remoteGPIOLineEnv); v != "" {
		remoteGPIOLine = v
	}
}

func connectSSH(t *testing.T) {
	if SSHClient != nil {
		return
	}

	config := &ssh.ClientConfig{
		User: remoteUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(remotePassword),
		},
		Timeout:         10 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	var err error
	SSHClient, err = ssh.Dial("tcp", remoteHost+":22", config)
	if err != nil {
		t.Fatalf("Failed to dial: %s", err)
	}

	t.Cleanup(func() {
		SSHClient.Close()
	})

	t.Logf("SSH: connected to %s", remoteHost)
}

func remote_deployOTBRAgent(t *testing.T) {

	t.Cleanup(func() {
		remote_exec(t, "sudo snap remove --purge openthread-border-router")
	})

	start := time.Now().UTC()

	commands := []string{
		"sudo snap remove --purge openthread-border-router",
		"sudo snap install openthread-border-router --edge",
		"sudo snap set openthread-border-router infra-if='" + remoteInfraInterface + "'",
		"sudo snap set openthread-border-router webgui-port=5000",
		// "sudo snap connect openthread-border-router:avahi-control",
		"sudo snap connect openthread-border-router:firewall-control",
		"sudo snap connect openthread-border-router:raw-usb",
		"sudo snap connect openthread-border-router:network-control",
		// "sudo snap connect openthread-border-router:bluetooth-control",
		// "sudo snap connect openthread-border-router:bluez",
		"sudo snap start openthread-border-router",
	}
	for _, cmd := range commands {
		remote_exec(t, cmd)
	}

	remote_waitForLogMessage(t, otbrSnap, "Start Thread Border Agent: OK", start)
	t.Log("OTBR on remote device is ready")
}

func remote_deployGPIOCommander(t *testing.T) {

	t.Cleanup(func() {
		remote_exec(t, "sudo snap remove --purge "+matterGPIOSnap)
	})

	installCommand := fmt.Sprintf("sudo snap install %s --edge", matterGPIOSnap)
	extraInterface := ""
	if remoteSnapPath != "" {
		installCommand = fmt.Sprintf("sudo snap install --dangerous %s", remoteSnapPath)
		extraInterface = fmt.Sprintf("sudo snap connect %s:custom-gpio %s:custom-gpio-dev", matterGPIOSnap, matterGPIOSnap)
	}

	start := time.Now().UTC()

	commands := []string{
		fmt.Sprintf("sudo snap remove --purge %s ", matterGPIOSnap),
		installCommand,
		extraInterface,
		fmt.Sprintf("sudo snap set %s args=\"--thread\"", matterGPIOSnap),
		fmt.Sprintf("sudo snap set %s gpiochip=\"%s\"", matterGPIOSnap, remoteGPIOChip),
		fmt.Sprintf("sudo snap set %s gpio=\"%s\"", matterGPIOSnap, remoteGPIOLine),
		fmt.Sprintf("sudo snap connect %s:avahi-control", matterGPIOSnap),
		fmt.Sprintf("sudo snap connect %s:otbr-dbus-wpan0 %s:dbus-wpan0", matterGPIOSnap, otbrSnap),
		fmt.Sprintf("sudo snap start %s", matterGPIOSnap),
	}
	for _, cmd := range commands {
		out := remote_exec(t, cmd)
		t.Log(out)
	}

	remote_waitForLogMessage(t, matterGPIOSnap, "CHIP:IN: TransportMgr initialized", start)
	t.Log("Matter PI GPIO Commander on remote device is ready")
}

func remote_exec(t *testing.T, command string) string {
	t.Helper()

	t.Logf("[exec-ssh] %s", command)

	if SSHClient == nil {
		t.Fatalf("SSH client not initialized. Please connect to remote device first")
	}

	session, err := SSHClient.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	if err := session.Start(command); err != nil {
		t.Fatalf("Failed to start session with command '%s': %v", command, err)
	}

	output, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("Failed to read command output: %v", err)
	}

	if err := session.Wait(); err != nil {
		t.Fatalf("Command '%s' failed: %v", command, err)
	}

	return string(output)
}

func remote_waitForLogMessage(t *testing.T, snap string, expectedLog string, start time.Time) {
	t.Helper()

	const maxRetry = 10
	for i := 1; i <= maxRetry; i++ {
		time.Sleep(1 * time.Second)
		t.Logf("Retry %d/%d: Waiting for expected content in logs: '%s'", i, maxRetry, expectedLog)

		command := fmt.Sprintf("sudo journalctl --utc --since \"%s\" --no-pager | grep \"%s\"|| true", start.UTC().Format("2006-01-02 15:04:05"), snap)
		logs := remote_exec(t, command)
		if strings.Contains(logs, expectedLog) {
			t.Logf("Found expected content in logs: '%s'", expectedLog)
			return
		}
	}

	t.Logf("Time out: reached max %d retries.", maxRetry)
	t.Log(remote_exec(t, "journalctl --no-pager --lines=10 --unit=snap.openthread-border-router.otbr-agent --priority=notice"))
	t.FailNow()
}

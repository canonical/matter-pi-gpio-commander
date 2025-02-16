package thread_tests

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/canonical/matter-snap-testing/utils"
	"golang.org/x/crypto/ssh"
)

var (
	remoteUser           = ""
	remotePassword       = ""
	remoteHost           = ""
	remoteInfraInterface = defaultInfraInterfaceValue
	remoteRadioUrl       = defaultRadioUrl

	remoteSnapPath = ""
	remoteGPIOChip = ""
	remoteGPIOLine = ""

	SSHClient *ssh.Client
)

func remote_setup(t *testing.T) {
	remote_loadEnvVars()

	connectSSH(t)

	remote_deployOTBRAgent(t)

	remote_deployGPIOCommander(t)
}

func remote_loadEnvVars() {

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

	if v := os.Getenv(remoteRadioUrlEnv); v != "" {
		remoteRadioUrl = v
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
	start := time.Now().UTC()

	t.Cleanup(func() {
		dumpRemoteLogs(t, "openthread-border-router", start)
		remote_exec(t, "sudo snap remove --purge openthread-border-router")
	})

	commands := []string{
		"sudo snap remove --purge openthread-border-router",
		"sudo snap install openthread-border-router --channel=latest/beta",
		fmt.Sprintf("sudo snap set openthread-border-router %s='%s'", infraInterfaceKey, remoteInfraInterface),
		fmt.Sprintf("sudo snap set openthread-border-router %s='%s'", radioUrlKey, remoteRadioUrl),
		"sudo snap set openthread-border-router webgui-port=31190",
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
	start := time.Now().UTC()

	t.Cleanup(func() {
		dumpRemoteLogs(t, "matter-pi-gpio-commander", start)
		remote_exec(t, "sudo snap remove --purge matter-pi-gpio-commander")
	})

	installCommand := "sudo snap install matter-pi-gpio-commander --channel=latest/edge"
	extraInterface := ""
	if remoteSnapPath != "" {
		installCommand = fmt.Sprintf("sudo snap install --dangerous %s", remoteSnapPath)
		extraInterface = "sudo snap connect matter-pi-gpio-commander:custom-gpio matter-pi-gpio-commander:custom-gpio-dev"
	}

	commands := []string{
		"sudo snap remove --purge matter-pi-gpio-commander",
		installCommand,
		extraInterface,
		"sudo snap set matter-pi-gpio-commander args=\"--thread\"",
		fmt.Sprintf("sudo snap set matter-pi-gpio-commander gpiochip=\"%s\"", remoteGPIOChip),
		fmt.Sprintf("sudo snap set matter-pi-gpio-commander gpio=\"%s\"", remoteGPIOLine),
		"sudo snap connect matter-pi-gpio-commander:avahi-control",
		"sudo snap connect matter-pi-gpio-commander:otbr-dbus-wpan0 openthread-border-router:dbus-wpan0",
		"sudo snap start matter-pi-gpio-commander",
	}
	for _, cmd := range commands {
		out := remote_exec(t, cmd)
		t.Log(out)
	}

	remote_waitForLogMessage(t, "matter-pi-gpio-commander", "CHIP:IN: TransportMgr initialized", start)
	t.Log("Matter PI GPIO Commander on remote device is ready")
}

func remote_exec(t *testing.T, command string) string {
	t.Helper()

	t.Logf("[exec-ssh] %s", command)

	// Set sudo to read the password from stdin
	if strings.HasPrefix(command, "sudo ") {
		command = strings.TrimPrefix(command, "sudo ")
		command = fmt.Sprintf(`sudo -S %s`, command)
	}

	if SSHClient == nil {
		t.Fatalf("SSH client not initialized. Please connect to remote device first")
	}

	session, err := SSHClient.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	var stderrBuffer []byte
	go remote_enterSudoPassword(stdin, stderr, &stderrBuffer)

	if err := session.Start(command); err != nil {
		t.Fatalf("Failed to start session with command '%s': %v", command, err)
	}

	stdoutBuffer, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("Failed to read command output: %v", err)
	}

	if err := session.Wait(); err != nil {
		t.Fatalf("Command '%s' failed: %v\n%s", command, err, stderrBuffer)
	}

	return string(stdoutBuffer)
}

// Monitor stderr for the sudo password request, and only pipe it in when it is requested
// https://stackoverflow.com/a/44501303
func remote_enterSudoPassword(in io.WriteCloser, out io.Reader, output *[]byte) {
	var (
		line string
		r    = bufio.NewReader(out)
	)
	for {
		b, err := r.ReadByte()
		if err != nil {
			break
		}

		*output = append(*output, b)

		if b == byte('\n') {
			line = ""
			continue
		}

		line += string(b)

		if strings.HasPrefix(line, "[sudo] password for ") && strings.HasSuffix(line, ": ") {
			_, err = in.Write([]byte(remotePassword + "\n"))
			if err != nil {
				break
			}
		}
	}
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

func dumpRemoteLogs(t *testing.T, label string, start time.Time) error {
	command := fmt.Sprintf("sudo journalctl --utc --since \"%s\" --no-pager | grep \"%s\"|| true", start.UTC().Format("2006-01-02 15:04:05"), label)
	logs := remote_exec(t, command)
	return utils.WriteLogFile(t, "remote-"+label, logs)
}

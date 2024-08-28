package thread_tests

import (
	"testing"
	"time"

	"github.com/canonical/matter-snap-testing/utils"
	"github.com/stretchr/testify/assert"
)

func TestThread(t *testing.T) {
	start := time.Now()

	setup(t)

	trimmedActiveDataset := getActiveDataset(t)

	remote_setup(t)

	t.Run("Commission", func(t *testing.T) {
		stdout, _, _ := utils.Exec(t, "sudo chip-tool pairing code-thread 110 hex:"+trimmedActiveDataset+" 34970112332 2>&1")
		assert.NoError(t, utils.WriteLogFile(t, "chip-tool", stdout))
	})

	t.Run("Control", func(t *testing.T) {
		stdout, _, _ := utils.Exec(t, "sudo chip-tool onoff toggle 110 1 2>&1")
		assert.NoError(t, utils.WriteLogFile(t, "chip-tool", stdout))

		remote_waitForLogMessage(t, "matter-pi-gpio-commander", "CHIP:ZCL: Toggle ep1 on/off", start)
	})

	t.Cleanup(func() {
		stdout, _, _ := utils.Exec(t, "sudo chip-tool onoff off 110 1 2>&1")
		assert.NoError(t, utils.WriteLogFile(t, "chip-tool", stdout))
	})
}

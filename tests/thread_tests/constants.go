package thread_tests

const (
	otbrSnap = "openthread-border-router"
	OTCTL    = otbrSnap + ".ot-ctl"

	defaultInfraInterfaceValue = "wlan0"
	infraInterfaceKey          = "infra-if"
	localInfraInterfaceEnv     = "LOCAL_INFRA_IF"
	remoteInfraInterfaceEnv    = "REMOTE_INFRA_IF"

	defaultRadioUrl   = "spinel+hdlc+uart:///dev/ttyACM0"
	radioUrlKey       = "radio-url"
	localRadioUrlEnv  = "LOCAL_RADIO_URL"
	remoteRadioUrlEnv = "REMOTE_RADIO_URL"

	remoteHostEnv     = "REMOTE_HOST"
	remoteUserEnv     = "REMOTE_USER"
	remotePasswordEnv = "REMOTE_PASSWORD"

	remoteSnapPathEnv = "REMOTE_SNAP_PATH"
	remoteGPIOChipEnv = "REMOTE_GPIO_CHIP"
	remoteGPIOLineEnv = "REMOTE_GPIO_LINE"
)

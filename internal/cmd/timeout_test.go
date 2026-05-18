package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeoutFlagToSec_ZeroAndNegativeReturnZero(t *testing.T) {
	require.Equal(t, 0, timeoutFlagToSec(0))
	require.Equal(t, 0, timeoutFlagToSec(-5*time.Second))
}

func TestTimeoutFlagToSec_RoundsSubSecondUpToOne(t *testing.T) {
	require.Equal(t, 1, timeoutFlagToSec(500*time.Millisecond))
	require.Equal(t, 1, timeoutFlagToSec(999*time.Millisecond))
}

func TestTimeoutFlagToSec_HonorsWholeSeconds(t *testing.T) {
	require.Equal(t, 5, timeoutFlagToSec(5*time.Second))
	require.Equal(t, 180, timeoutFlagToSec(180*time.Second))
	require.Equal(t, 180, timeoutFlagToSec(3*time.Minute))
}

func TestMarkImageTimeout_FlagWinsOverConfig(t *testing.T) {
	got := markImageTimeout(45*time.Second, 180*time.Second)
	require.Equal(t, 45*time.Second, got)
}

func TestMarkImageTimeout_FallsBackToConfigWhenFlagUnset(t *testing.T) {
	got := markImageTimeout(0, 180*time.Second)
	require.Equal(t, 180*time.Second, got)
}

func TestMarkImageTimeout_FallsBackToPackageDefaultWhenBothUnset(t *testing.T) {
	got := markImageTimeout(0, 0)
	require.Equal(t, markImageTimeoutFallback, got)
}

func TestMarkImageTimeout_NegativeConfigTreatedAsUnset(t *testing.T) {
	got := markImageTimeout(0, -1*time.Second)
	require.Equal(t, markImageTimeoutFallback, got)
}

// TestMarkCmdExposesTimeoutFlag is a smoke test that the cobra command has
// the --timeout flag wired with Duration type. Guards against accidental
// removal from the init() block.
func TestMarkCmdExposesTimeoutFlag(t *testing.T) {
	flag := markCmd.Flags().Lookup("timeout")
	require.NotNil(t, flag, "mark should expose a --timeout flag")
	require.Equal(t, "duration", flag.Value.Type())
}

func TestCheckCmdExposesTimeoutFlag(t *testing.T) {
	flag := checkCmd.Flags().Lookup("timeout")
	require.NotNil(t, flag, "check should expose a --timeout flag")
	require.Equal(t, "duration", flag.Value.Type())
}

func TestGenerateCmdExposesTimeoutFlag(t *testing.T) {
	flag := generateCmd.Flags().Lookup("timeout")
	require.NotNil(t, flag, "generate should expose a --timeout flag")
	require.Equal(t, "duration", flag.Value.Type())
}

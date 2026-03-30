package telemetry_test

import (
	"testing"

	"github.com/prasenjit-net/mcp-gateway/telemetry"
)

func TestSetupLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error", "unknown"} {
		logger := telemetry.Setup(level)
		if logger == nil {
			t.Errorf("Setup(%q) returned nil logger", level)
		}
	}
}

func TestRegister(t *testing.T) {
	// Register is a no-op (promauto handles registration), just ensure no panic.
	telemetry.Register()
}

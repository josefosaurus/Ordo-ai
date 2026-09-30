package telemetryruntime

import (
	"os"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/telemetry"
)

// TestMain installs a fake collector endpoint: Ordo ships without one, and
// telemetry.Decide disables everything when no endpoint is configured.
func TestMain(m *testing.M) {
	telemetry.DefaultEndpoint = "https://collector.test/v1/events"
	os.Exit(m.Run())
}

package debug

import (
	"strings"
	"testing"
)

func TestRedactMasksSecrets(t *testing.T) {
	in := []byte(`{"Signing":{"GpgKey":"DEAD","Passphrase":"hunter2","Batch":true},"password":"s3cr3t"}`)
	out := Redact(in)
	if strings.Contains(out, "hunter2") {
		t.Errorf("passphrase leaked: %s", out)
	}
	if strings.Contains(out, "s3cr3t") {
		t.Errorf("password leaked: %s", out)
	}
	// Non-secret fields are preserved.
	if !strings.Contains(out, `"GpgKey":"DEAD"`) {
		t.Errorf("non-secret field altered: %s", out)
	}
	if !strings.Contains(out, `"Passphrase":"***"`) {
		t.Errorf("passphrase not masked to ***: %s", out)
	}
}

func TestLogfNoOpWhenDisabled(t *testing.T) {
	// Default Enabled is false; Logf must not panic and must be a no-op.
	Enabled = false
	Logf("should not appear %d", 1)
	Section("nope")
	// Re-enabling and logging should also not panic.
	Enabled = true
	defer func() { Enabled = false }()
	Logf("ok %s", "x")
}

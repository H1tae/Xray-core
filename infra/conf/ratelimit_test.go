package conf

import (
	"testing"
	"time"

	"github.com/xtls/xray-core/app/ratelimit"
)

func TestRateLimitConfigApplyGraceSeconds(t *testing.T) {
	oldMode := ratelimit.GetKeyMode()
	oldGrace := ratelimit.GetGrace()
	t.Cleanup(func() {
		ratelimit.SetKeyMode(oldMode)
		ratelimit.SetGrace(oldGrace)
	})

	seconds := uint32(42)
	cfg := RateLimitConfig{
		KeyMode:      "device",
		GraceSeconds: &seconds,
	}

	if err := cfg.Apply(); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	if got := ratelimit.GetGrace(); got != 42*time.Second {
		t.Fatalf("expected grace 42s, got %v", got)
	}
	if got := ratelimit.GetKeyMode(); got != ratelimit.KeyModeDevice {
		t.Fatalf("expected key mode device, got %v", got)
	}
}

func TestRateLimitConfigApplyWithoutGraceKeepsExistingGrace(t *testing.T) {
	oldMode := ratelimit.GetKeyMode()
	oldGrace := ratelimit.GetGrace()
	t.Cleanup(func() {
		ratelimit.SetKeyMode(oldMode)
		ratelimit.SetGrace(oldGrace)
	})

	ratelimit.SetGrace(99 * time.Second)

	cfg := RateLimitConfig{
		KeyMode: "uuid",
	}

	if err := cfg.Apply(); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	if got := ratelimit.GetGrace(); got != 99*time.Second {
		t.Fatalf("expected grace to stay 99s, got %v", got)
	}
	if got := ratelimit.GetKeyMode(); got != ratelimit.KeyModeUUID {
		t.Fatalf("expected key mode uuid, got %v", got)
	}
}

func TestRateLimitConfigApplyGraceOnlyKeepsExistingMode(t *testing.T) {
	oldMode := ratelimit.GetKeyMode()
	oldGrace := ratelimit.GetGrace()
	t.Cleanup(func() {
		ratelimit.SetKeyMode(oldMode)
		ratelimit.SetGrace(oldGrace)
	})

	ratelimit.SetKeyMode(ratelimit.KeyModeDevice)

	seconds := uint32(7)
	cfg := RateLimitConfig{
		GraceSeconds: &seconds,
	}

	if err := cfg.Apply(); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	if got := ratelimit.GetGrace(); got != 7*time.Second {
		t.Fatalf("expected grace 7s, got %v", got)
	}
	if got := ratelimit.GetKeyMode(); got != ratelimit.KeyModeDevice {
		t.Fatalf("expected key mode device to stay unchanged, got %v", got)
	}
}

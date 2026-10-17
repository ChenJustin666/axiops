package build

import (
	"strings"
	"testing"
	"time"
)

func TestBranchAllowed(t *testing.T) {
	rules := `{"includes":["main","release/*"],"excludes":["release/wip*"]}`
	cases := map[string]bool{"main": true, "release/1.2": true, "feature/foo": false, "release/wip-1": false}
	for branch, want := range cases {
		if got := branchAllowed(rules, branch); got != want {
			t.Fatalf("branchAllowed(%q)=%v, want %v", branch, got, want)
		}
	}
}

func TestAutomaticImageTag(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 30, 45, 0, time.UTC)
	tag := automaticImageTag("feature/user login", 42, now)
	if tag != "feature-user-login-42-20260810123045" {
		t.Fatalf("automaticImageTag() = %q", tag)
	}
	if !validImageTag.MatchString(tag) {
		t.Fatalf("generated tag %q is invalid", tag)
	}
}

func TestAutomaticImageTagLength(t *testing.T) {
	tag := automaticImageTag(strings.Repeat("a", 200), 1, time.Now())
	if len(tag) > 128 {
		t.Fatalf("generated tag length = %d, want <= 128", len(tag))
	}
}

func TestValidateCustomImageTag(t *testing.T) {
	if err := validateCustomImageTag("release_2026.08-10"); err != nil {
		t.Fatalf("valid tag rejected: %v", err)
	}
	if err := validateCustomImageTag("feature/login"); err == nil {
		t.Fatal("tag containing slash should be rejected")
	}
}

package webhook

import "testing"

func TestMatchBranchRule(t *testing.T) {
	tests := []struct {
		name, raw, branch string
		want              bool
	}{
		{"empty allows", "", "main", true},
		{"include exact", `{"includes":["main"]}`, "main", true},
		{"include wildcard", `{"includes":["release/*"]}`, "release/1.2", true},
		{"exclude wins", `{"includes":["*"],"excludes":["feature/*"]}`, "feature/demo", false},
		{"malformed denies", `{`, "main", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchBranchRule(tt.raw, tt.branch); got != tt.want {
				t.Fatalf("matchBranchRule()=%v want %v", got, tt.want)
			}
		})
	}
}

func TestParseIDIsStrict(t *testing.T) {
	for input, want := range map[string]int64{"12": 12, " 9 ": 9, "12x": 0, "0": 0, "-1": 0, "": 0} {
		if got := parseID(input); got != want {
			t.Fatalf("parseID(%q)=%d want %d", input, got, want)
		}
	}
}

func TestHashPayloadIsStable(t *testing.T) {
	if hashPayload([]byte("same")) != hashPayload([]byte("same")) {
		t.Fatal("payload hash must be stable")
	}
	if hashPayload([]byte("same")) == hashPayload([]byte("other")) {
		t.Fatal("different payloads must not share a hash")
	}
}

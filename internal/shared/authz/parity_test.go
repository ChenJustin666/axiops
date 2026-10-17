package authz

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

// Shared with the browser policy evaluator. This verifies authorization rules,
// not endpoint integration: every handler still has to resolve its own target.
func TestFrontendBackendPolicyParity(t *testing.T) {
	data, err := os.ReadFile("testdata/policy_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name     string   `json:"name"`
		Policies []Policy `json:"policies"`
		Request  Request  `json:"request"`
		Want     bool     `json:"want"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := New(memoryStore(tc.Policies)).Authorize(context.Background(), 7, tc.Request)
			if err != nil || got != tc.Want {
				t.Fatalf("allowed=%v want=%v err=%v", got, tc.Want, err)
			}
		})
	}
}

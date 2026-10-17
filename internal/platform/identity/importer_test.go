package identity

import "testing"

func TestValidateImportEntries(t *testing.T) {
	valid := []ImportEntry{
		{ExternalUserID: 1, ExternalName: "alice", TargetUserID: 11},
		{ExternalUserID: 2, ExternalName: "bob", TargetUserID: 12},
	}
	if err := validateImportEntries(valid); err != nil {
		t.Fatalf("valid entries rejected: %v", err)
	}
	tests := [][]ImportEntry{
		{{ExternalUserID: 0, ExternalName: "alice", TargetUserID: 11}},
		{{ExternalUserID: 1, ExternalName: "", TargetUserID: 11}},
		{{ExternalUserID: 1, ExternalName: "alice", TargetUserID: 11}, {ExternalUserID: 1, ExternalName: "alice", TargetUserID: 12}},
		{{ExternalUserID: 1, ExternalName: "alice", TargetUserID: 11}, {ExternalUserID: 2, ExternalName: "bob", TargetUserID: 11}},
	}
	for index, entries := range tests {
		if err := validateImportEntries(entries); err == nil {
			t.Fatalf("invalid test %d was accepted", index)
		}
	}
}

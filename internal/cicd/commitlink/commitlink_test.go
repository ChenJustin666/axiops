package commitlink

import (
	"context"
	"strings"
	"testing"
)

func TestSourceMustStillMatchHistoricalServer(t *testing.T) {
	for _, tc := range []struct {
		repository, source string
		want               bool
	}{
		{"http://47.243.132.221:5000/server/risk_server.git", "http://47.243.132.221:5000/", true},
		{"https://GIT.example:443/gitea/team/api.git", "https://git.example/gitea/", true},
		{"https://git.example/gitea-other/team/api.git", "https://git.example/gitea", false},
		{"https://git.example/gitea/../other/api.git", "https://git.example/gitea", false},
		{"https://git.example/team/api", "https://moved.example/", false},
		{"http://git.example/team/api", "https://git.example/", false},
		{"https://git.example:8443/team/api", "https://git.example/", false},
		{"javascript:alert(1)", "https://git.example/", false},
	} {
		if got := sameServer(tc.repository, tc.source); got != tc.want {
			t.Fatalf("sameServer(%q,%q)=%v", tc.repository, tc.source, got)
		}
	}
}

func TestKnownSnapshotsAndStoredURLsNeedNoDatabase(t *testing.T) {
	head := strings.Repeat("a", 40)
	records := []Record{
		{Commit: head, URL: "https://original.example/commit/" + head},
		{Commit: head, Snapshot: Snapshot{RepositoryURL: "https://user:secret@git.example/team/original.git?token=secret", CodeSourceType: "gitea"}},
		{Commit: head, Snapshot: Snapshot{RepositoryURL: "https://gitlab.com/team/original.git"}},
		{Commit: "main", Snapshot: Snapshot{RepositoryURL: "https://git.example/api", CodeSourceID: 1}},
		{Commit: head},
	}
	if err := Fill(context.Background(), nil, records); err != nil {
		t.Fatal(err)
	}
	if records[0].URL != "https://original.example/commit/"+head || records[1].URL != "https://git.example/team/original/commit/"+head || records[2].URL != "https://gitlab.com/team/original/-/commit/"+head || records[3].URL != "" || records[4].URL != "" {
		t.Fatal("incorrect historical links")
	}
}

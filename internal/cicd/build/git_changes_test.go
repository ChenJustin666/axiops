package build

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGitBrowseURLs(t *testing.T) {
	head, base := strings.Repeat("b", 40), strings.Repeat("a", 40)
	for _, kind := range []string{"gitlab", "github", "gitea", "gitee"} {
		prefix := "/"
		if kind == "gitlab" {
			prefix = "/-/"
		}
		got := gitBrowseURL("https://user:secret@git.example/group/repo.git/?token=secret#old", kind, "", head)
		if got != "https://git.example/group/repo"+prefix+"commit/"+head {
			t.Fatal(got)
		}
		if got = gitBrowseURL("http://git.example:5000/group/repo", kind, base, head); got != "http://git.example:5000/group/repo"+prefix+"compare/"+base+"..."+head {
			t.Fatal(got)
		}
	}
	for _, repository := range []string{"javascript:alert(1)", "file:///etc/passwd", "https://git.example", "git@git.example:group/repo.git"} {
		if got := gitBrowseURL(repository, "gitlab", "", head); got != "" {
			t.Fatal(got)
		}
	}
	if gitBrowseURL("https://git.example/a/b", "unknown", "", head) != "" {
		t.Fatal("unknown provider guessed")
	}
	if gitBrowseURL("https://gitlab.com/a/b", "", "", head) == "" {
		t.Fatal("known hosted provider missing")
	}
}

func TestGitChangesFromRealCheckout(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if runtime.GOOS == "windows" {
		bash = `C:\Program Files\Git\bin\bash.exe`
		_, err = os.Stat(bash)
	}
	if err != nil {
		t.Skip("POSIX shell unavailable")
	}
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=研发人员", "GIT_AUTHOR_EMAIL=dev@example.test", "GIT_COMMITTER_NAME=CI", "GIT_COMMITTER_EMAIL=ci@example.test", "GIT_CONFIG_NOSYSTEM=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("config", "commit.gpgsign", "false")
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("api.txt", "v1\n")
	git("add", ".")
	git("commit", "-qm", "initial")
	base := git("rev-parse", "HEAD")
	write("api.txt", "v2\n")
	git("add", ".")
	git("commit", "-qm", "修复端口 | \"配置\" <script>")
	write("服务 文件.yaml", "kind: Service\n")
	git("add", ".")
	git("commit", "-qm", "增加服务配置")
	head := git("rev-parse", "HEAD")
	for _, tc := range []struct {
		name, base, status string
		commits, files     int
	}{
		{"first", "", "first_build", 1, 0}, {"range", base, "complete", 2, 2}, {"same", head, "complete", 0, 0},
		{"missing-baseline", strings.Repeat("f", 40), "unavailable", 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(bash, "-c", gitChangesScript)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GIT_CHANGE_BASE="+tc.base)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("collector: %s %v", out, err)
			}
			changes := parseGitChanges(out)
			if changes == nil || changes.Status != tc.status || changes.Head != head || len(changes.Commits) != tc.commits || len(changes.Files) != tc.files {
				t.Fatalf("bad changes: %+v output=%s", changes, out)
			}
			if changes.Latest == nil || changes.Latest.Subject != "增加服务配置" {
				t.Fatalf("lost latest commit: %+v", changes.Latest)
			}
			if tc.name == "range" && changes.Commits[1].Subject != "修复端口 | \"配置\" <script>" {
				t.Fatal("subject was corrupted")
			}
		})
	}
	// The collector is intentionally executed in an OR-list by the provider.
	// A failed git diff must leave the initial unavailable evidence intact.
	cmd := exec.Command(bash, "-c", "git() { if [ \"$1\" = diff ]; then return 42; fi; command git \"$@\"; }; ("+gitChangesScript+") || :")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CHANGE_BASE="+base)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if changes := parseGitChanges(out); changes == nil || changes.Status != "unavailable" {
		t.Fatalf("failed diff advertised complete history: %s", out)
	}
}

func TestGitSummaryBoundsPreserveUnicode(t *testing.T) {
	if got := truncateGitSummary(strings.Repeat("中", 200), 128, 512); len([]rune(got)) != 128 {
		t.Fatal("author bound failed")
	}
	if got := truncateGitSummary("你好abc", 10, 5); got != "你" {
		t.Fatalf("split UTF-8 at byte limit: %q", got)
	}
}

type changeLogReader map[string]string

func (r changeLogReader) Read(_ context.Context, _, container string, _ int64) ([]byte, error) {
	return []byte(r[container]), nil
}

func TestGitChangesProviderUsesCheckoutEvidence(t *testing.T) {
	head, base := strings.Repeat("b", 40), strings.Repeat("a", 40)
	row := head + "\x00author\x00mail\x002026-10-02T10:00:00+08:00\x00fix\x00"
	line := gitChangesPrefix + "first_build|" + head + "||0|0|" + base64.StdEncoding.EncodeToString([]byte(row)) + "|\n"
	for _, tc := range []struct {
		name, checked, clone, builder string
		want                          bool
	}{
		{"checked", head, line, "normal output", true},
		{"mismatch", base, line, "", false},
		{"unfinished", "", line, "", false},
		{"builder-is-not-checkout", head, "normal checkout", line, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod"}, Status: corev1.PodStatus{Phase: corev1.PodRunning, InitContainerStatuses: []corev1.ContainerStatus{{Name: "git-clone", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0, Message: tc.checked}}}}}}
			client := &fakeJobClient{existing: &batchv1.Job{}}
			provider := NewKubeJobProvider("ci", client, client, diagnosticPods{pod: pod}).WithRuntimeData(nil, changeLogReader{"git-clone": tc.clone, "builder": tc.builder}, 1<<20)
			state, err := provider.Status(context.Background(), ProviderExecution{JobName: "job"})
			if err != nil || (state.GitChanges != nil) != tc.want {
				t.Fatalf("invalid source evidence state=%+v err=%v", state, err)
			}
			if tc.want && strings.Contains(state.LogText, gitChangesPrefix) {
				t.Fatal("base64 envelope leaked into user logs")
			}
		})
	}
	job, err := BuildKubeJob("ci", ProviderBuildRequest{BuildRunID: 1, RepositoryURL: "https://git.example/team/api", BuildCommand: "true", ChangeBaseCommit: base})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range job.Spec.Template.Spec.InitContainers[0].Env {
		if item.Name == "GIT_CHANGE_BASE" && item.Value == base {
			return
		}
	}
	t.Fatal("frozen baseline missing from checkout container")
}

func TestGitChangesRejectMalformedAndBoundOutput(t *testing.T) {
	head := strings.Repeat("a", 40)
	encode := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	row := head + "\x00author\x00mail\x002026-10-02T10:00:00+08:00\x00subject\x00"
	line := gitChangesPrefix + "complete|" + head + "|" + head + "|0|0|" + encode(strings.Repeat(row, 101)) + "|" + encode(strings.Repeat("M\x00some file\x00", 201))
	set := parseGitChanges([]byte(line))
	if set == nil || len(set.Commits) != 100 || len(set.Files) != 200 || !set.CommitsTruncated || !set.FilesTruncated {
		t.Fatalf("limits failed: %+v", set)
	}
	for _, bad := range []string{strings.Replace(line, "complete", "fake", 1), strings.Replace(line, head, "main", 1), gitChangesPrefix + "complete|" + head + "||0|0|broken|", gitChangesPrefix + "complete|" + head + "||0|0|" + encode(row) + "|" + encode("M\x00")} {
		if parseGitChanges([]byte(bad)) != nil {
			t.Fatal("malformed evidence accepted")
		}
	}
}

type changeStore struct {
	fakeRunningBuildStore
	saved   *GitChangeSet
	saveErr error
}

func (s *changeStore) SaveGitChanges(_ context.Context, _ BuildRun, c *GitChangeSet) error {
	s.saved = c
	return s.saveErr
}
func TestGitChangesPersistBeforeTerminalAcknowledgement(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s := &changeStore{fakeRunningBuildStore: fakeRunningBuildStore{run: &BuildRun{ID: 1, Provider: "observable", Status: StatusRunning}}}
		if fail {
			s.saveErr = errors.New("database temporarily unavailable")
		}
		providers, _ := NewProviderRegistry(observableProvider{state: ProviderState{Status: StatusFailed, GitChanges: &GitChangeSet{Head: strings.Repeat("a", 40)}}})
		_, err := NewReconciler(s, providers).ReconcileOne(context.Background())
		if s.saved == nil || (fail && (err == nil || s.status != "")) || (!fail && (err != nil || s.status != StatusFailed)) {
			t.Fatalf("bad persistence ordering: %v %+v", err, s)
		}
	}
}

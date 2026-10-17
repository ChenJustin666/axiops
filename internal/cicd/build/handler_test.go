package build

import "testing"

func TestLogSnapshotCursorDetectsChangesWithoutNewLines(t *testing.T) {
	first := logSnapshotID(1, []byte("waiting for cpu"))
	if first == logSnapshotID(1, []byte("pulling image")) || first == logSnapshotID(2, []byte("waiting for cpu")) {
		t.Fatal("changed snapshots must have distinct cursors")
	}
	if first != logSnapshotID(1, []byte("waiting for cpu")) {
		t.Fatal("identical snapshots must retain their cursor")
	}
}

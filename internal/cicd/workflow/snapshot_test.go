package workflow

import "testing"

func TestYAMLDeploymentSnapshotIsDigestVerifiable(t *testing.T) {
	snapshot, err := NewYAMLDeploymentSnapshot(
		WorkflowRun{ID: 21, ProjectID: 10, EnvironmentID: 7},
		TaskRun{ID: 22, NodeKey: "deploy"},
		YAMLDeployConfig{ServiceID: 9, ServiceRevisionID: 12},
		YAMLTarget{ClusterID: 2, Namespace: "test", Manifest: []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: api\n")},
	)
	if err != nil || snapshot == nil || !snapshot.VerifyManifestDigest() {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	snapshot.RenderedManifest[0] = 'x'
	if snapshot.VerifyManifestDigest() {
		t.Fatal("tampered manifest passed digest verification")
	}
}

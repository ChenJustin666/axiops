package build

import "testing"

func TestMinIOLogArchiveRestrictsReferencesToBucketAndPrefix(t *testing.T) {
	archive := &MinIOLogArchive{bucket: "axiops", prefix: "build-logs"}
	if key, err := archive.objectKey("projects/1/builds/2/build.log"); err != nil || key != "build-logs/projects/1/builds/2/build.log" {
		t.Fatalf("key=%q err=%v", key, err)
	}
	if key, err := archive.parseRef("s3://axiops/build-logs/projects/1/builds/2/build.log"); err != nil || key == "" {
		t.Fatalf("key=%q err=%v", key, err)
	}
	for _, ref := range []string{"https://axiops/build-logs/a", "s3://other/build-logs/a", "s3://axiops/outside/a", "s3://axiops/build-logs/../secret"} {
		if _, err := archive.parseRef(ref); err == nil {
			t.Fatalf("unsafe reference %q was accepted", ref)
		}
	}
}

func TestInlineLogTailIsBounded(t *testing.T) {
	if got := inlineLogTail("123456", 4); got != "3456" {
		t.Fatalf("tail=%q", got)
	}
}

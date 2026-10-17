package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const maxArchivedBuildLogBytes = 32 << 20

type LogArchive interface {
	Put(context.Context, string, []byte) (string, error)
	Get(context.Context, string) ([]byte, error)
}

type LogArchiveRemover interface {
	Delete(context.Context, string) error
}

func (a *MinIOLogArchive) Delete(ctx context.Context, ref string) error {
	key, err := a.parseRef(ref)
	if err != nil {
		return err
	}
	return a.client.RemoveObject(ctx, a.bucket, key, minio.RemoveObjectOptions{})
}

type MinIOLogArchive struct {
	client *minio.Client
	bucket string
	prefix string
}

// NewLogArchiveFromEnvironment keeps object-store credentials out of YAML.
// An empty endpoint disables external archival for local development.
func NewLogArchiveFromEnvironment() (LogArchive, error) {
	endpoint := strings.TrimSpace(os.Getenv("AXIOPS_BUILD_LOG_ENDPOINT"))
	if endpoint == "" {
		return nil, nil
	}
	bucket := strings.TrimSpace(os.Getenv("AXIOPS_BUILD_LOG_BUCKET"))
	accessKey := strings.TrimSpace(os.Getenv("AXIOPS_BUILD_LOG_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("AXIOPS_BUILD_LOG_SECRET_KEY"))
	if bucket == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AXIOPS_BUILD_LOG_BUCKET, ACCESS_KEY, and SECRET_KEY are required when external build logs are enabled")
	}
	secure := true
	if raw := strings.TrimSpace(os.Getenv("AXIOPS_BUILD_LOG_USE_SSL")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("AXIOPS_BUILD_LOG_USE_SSL must be true or false")
		}
		secure = value
	}
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
		endpoint = parsed.Host
		secure = parsed.Scheme != "http"
	}
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: secure})
	if err != nil {
		return nil, fmt.Errorf("initialize build log object storage: %w", err)
	}
	prefix := strings.Trim(strings.TrimSpace(os.Getenv("AXIOPS_BUILD_LOG_PREFIX")), "/")
	if prefix == "" {
		prefix = "axiops/build-logs"
	}
	return &MinIOLogArchive{client: client, bucket: bucket, prefix: prefix}, nil
}

func (a *MinIOLogArchive) Put(ctx context.Context, key string, content []byte) (string, error) {
	if a == nil || a.client == nil || len(content) > maxArchivedBuildLogBytes {
		return "", fmt.Errorf("build log archive is not configured or content exceeds 32 MiB")
	}
	objectKey, err := a.objectKey(key)
	if err != nil {
		return "", err
	}
	_, err = a.client.PutObject(ctx, a.bucket, objectKey, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{ContentType: "text/plain; charset=utf-8"})
	if err != nil {
		return "", fmt.Errorf("archive build log: %w", err)
	}
	return "s3://" + a.bucket + "/" + objectKey, nil
}

func (a *MinIOLogArchive) Get(ctx context.Context, ref string) ([]byte, error) {
	objectKey, err := a.parseRef(ref)
	if err != nil {
		return nil, err
	}
	object, err := a.client.GetObject(ctx, a.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()
	content, err := io.ReadAll(io.LimitReader(object, maxArchivedBuildLogBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read archived build log: %w", err)
	}
	if len(content) > maxArchivedBuildLogBytes {
		return nil, fmt.Errorf("archived build log exceeds 32 MiB")
	}
	return content, nil
}

func (a *MinIOLogArchive) objectKey(key string) (string, error) {
	clean := path.Clean(strings.TrimSpace(key))
	if clean == "." || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid build log object key")
	}
	return a.prefix + "/" + clean, nil
}

func (a *MinIOLogArchive) parseRef(ref string) (string, error) {
	parsed, err := url.Parse(ref)
	if err != nil || parsed.Scheme != "s3" || parsed.Host != a.bucket {
		return "", fmt.Errorf("invalid archived build log reference")
	}
	key := strings.TrimPrefix(parsed.EscapedPath(), "/")
	decoded, err := url.PathUnescape(key)
	if err != nil || !strings.HasPrefix(decoded, a.prefix+"/") || strings.Contains(decoded, "..") {
		return "", fmt.Errorf("archived build log reference is outside the configured prefix")
	}
	return decoded, nil
}

func inlineLogTail(content string, limit int) string {
	if limit <= 0 || len(content) <= limit {
		return content
	}
	tail := content[len(content)-limit:]
	for len(tail) > 0 && tail[0]&0xc0 == 0x80 {
		tail = tail[1:]
	}
	return tail
}

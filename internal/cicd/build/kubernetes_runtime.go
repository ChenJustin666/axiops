package build

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/typed/core/v1"
)

type KubernetesSecretStore struct{ secrets v1.SecretInterface }

func NewKubernetesSecretStore(secrets v1.SecretInterface) *KubernetesSecretStore {
	return &KubernetesSecretStore{secrets: secrets}
}

func (s *KubernetesSecretStore) Ensure(ctx context.Context, name, idempotencyKey string, data map[string]string) error {
	if s == nil || s.secrets == nil || name == "" || idempotencyKey == "" {
		return fmt.Errorf("secret client, name, and idempotency key are required")
	}
	stringData := make(map[string]string, len(data))
	for key, value := range data {
		stringData[key] = value
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{
		"axiops.io/component": "build-credential", "axiops.io/idempotency": labelValue(idempotencyKey),
	}}, Type: corev1.SecretTypeOpaque, StringData: stringData}
	if _, ok := data[corev1.DockerConfigJsonKey]; ok {
		secret.Type = corev1.SecretTypeDockerConfigJson
	}
	_, err := s.secrets.Create(ctx, secret, metav1.CreateOptions{})
	if !apierrors.IsAlreadyExists(err) {
		return err
	}
	existing, getErr := s.secrets.Get(ctx, name, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}
	if existing.Labels["axiops.io/idempotency"] != labelValue(idempotencyKey) {
		return fmt.Errorf("build credential secret %q belongs to another request", name)
	}
	return nil
}

func (s *KubernetesSecretStore) Delete(ctx context.Context, name string) error {
	if s == nil || s.secrets == nil || name == "" {
		return nil
	}
	err := s.secrets.Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (s *KubernetesSecretStore) DeleteOwned(ctx context.Context, name, key string) error {
	secret, err := s.secrets.Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if secret.Labels["axiops.io/component"] != "build-credential" || secret.Labels["axiops.io/idempotency"] != labelValue(key) {
		return fmt.Errorf("refusing to delete build credential %q owned by another request", name)
	}
	uid := secret.UID
	err = s.secrets.Delete(ctx, name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid}})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

type KubernetesPodLogReader struct{ pods v1.PodInterface }

func NewKubernetesPodLogReader(pods v1.PodInterface) *KubernetesPodLogReader {
	return &KubernetesPodLogReader{pods: pods}
}

func (r *KubernetesPodLogReader) Read(ctx context.Context, podName, container string, limitBytes int64) ([]byte, error) {
	if r == nil || r.pods == nil || podName == "" || container == "" || limitBytes <= 0 {
		return nil, fmt.Errorf("pod log client, identity, and limit are required")
	}
	stream, err := r.pods.GetLogs(podName, &corev1.PodLogOptions{Container: container, LimitBytes: &limitBytes}).Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	return io.ReadAll(io.LimitReader(stream, limitBytes))
}

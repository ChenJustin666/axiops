package assetdeploy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

type factory interface {
	RESTConfig(context.Context, int64) (*rest.Config, error)
}
type Manager struct {
	db      *sql.DB
	decrypt workflow.CredentialDecryptor
	client  func(context.Context, int64) (kubernetes.Interface, error)
}

func NewManager(db *sql.DB, decrypt workflow.CredentialDecryptor, f factory) *Manager {
	return &Manager{db: db, decrypt: decrypt, client: func(ctx context.Context, id int64) (kubernetes.Interface, error) {
		cfg, err := f.RESTConfig(ctx, id)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(cfg)
	}}
}
func (m *Manager) PrepareYAMLDeployment(ctx context.Context, run workflow.WorkflowRun, cfg workflow.YAMLDeployConfig, target workflow.YAMLTarget) (workflow.YAMLTarget, error) {
	if len(target.AssetSecrets) == 0 {
		if len(target.Manifest) == 0 {
			return target, nil
		}
		client, err := m.client(ctx, target.ClusterID)
		if err != nil {
			return target, fmt.Errorf("无法连接目标集群进行部署依赖检查")
		}
		return target, preflightDependencies(ctx, client, target)
	}
	if m.db == nil || m.decrypt == nil || run.ProjectID <= 0 || cfg.ServiceID <= 0 || target.Namespace == "" {
		return target, fmt.Errorf("服务凭据准备器未配置")
	}
	// Resolve all credentials before any write; never return plaintext in errors.
	prepared := make([]corev1.Secret, 0, len(target.AssetSecrets))
	byName := map[string]int{}
	for _, ref := range target.AssetSecrets {
		var pid int64
		var kind, data string
		if m.db.QueryRowContext(ctx, `SELECT project_id,type,data FROM credential WHERE id=?`, ref.CredentialID).Scan(&pid, &kind, &data) != nil || kind != "secret" || (pid != 0 && pid != run.ProjectID) {
			return target, fmt.Errorf("凭据 #%d 不可用", ref.CredentialID)
		}
		plain, err := m.decrypt.Decrypt(data)
		var fields map[string]interface{}
		if err != nil || json.Unmarshal([]byte(plain), &fields) != nil {
			return target, fmt.Errorf("凭据 #%d 无法解密", ref.CredentialID)
		}
		value, ok := fields[ref.CredentialKey].(string)
		if !ok {
			return target, fmt.Errorf("凭据 #%d 缺少所选字段", ref.CredentialID)
		}
		secret := corev1.Secret{ObjectMeta: serviceassets.SecretMetadata(ref.Name, target.Namespace, cfg.ServiceID), Type: corev1.SecretTypeOpaque, Data: map[string][]byte{ref.Key: []byte(value)}}
		secret.Labels["axiops.io/project-id"] = fmt.Sprint(run.ProjectID)
		if index, exists := byName[ref.Name]; exists {
			if _, duplicate := prepared[index].Data[ref.Key]; duplicate {
				return target, fmt.Errorf("Secret包含重复键，凭据准备已停止")
			}
			prepared[index].Data[ref.Key] = []byte(value)
		} else {
			byName[ref.Name] = len(prepared)
			prepared = append(prepared, secret)
		}
	}
	client, err := m.client(ctx, target.ClusterID)
	if err != nil {
		return target, fmt.Errorf("无法连接目标集群以准备Secret")
	}
	if err := preflightDependencies(ctx, client, target); err != nil {
		return target, err
	}
	_, err = client.CoreV1().Namespaces().Get(ctx, target.Namespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: target.Namespace}}, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
	}
	if err != nil {
		return target, fmt.Errorf("目标命名空间不可用：%s", apierrors.ReasonForError(err))
	}
	for _, secret := range prepared {
		if err = ensureSecret(ctx, client, secret); err != nil {
			return target, err
		}
	}
	return target, nil
}
func ensureSecret(ctx context.Context, client kubernetes.Interface, desired corev1.Secret) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secrets := client.CoreV1().Secrets(desired.Namespace)
		old, err := secrets.Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = secrets.Create(ctx, &desired, metav1.CreateOptions{})
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			old, err = secrets.Get(ctx, desired.Name, metav1.GetOptions{})
		}
		if err != nil {
			return err
		}
		for key, value := range desired.Labels {
			if old.Labels[key] != value {
				return fmt.Errorf("拒绝覆盖非本服务管理的Secret")
			}
		}
		old.Data = desired.Data
		old.Type = desired.Type
		_, err = secrets.Update(ctx, old, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		return fmt.Errorf("准备Secret %s 失败（权限、归属或资源冲突），未记录密码正文", desired.Name)
	}
	return nil
}

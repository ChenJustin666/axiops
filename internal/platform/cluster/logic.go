package cluster

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Logic 集群业务逻辑
type Logic struct {
	repo       *Repository
	authorizer authz.Checker
	box        *secretbox.Box
}

func NewLogic(db *sql.DB, authorizer authz.Checker, encryptionKey ...string) *Logic {
	logic := &Logic{repo: NewRepository(db), authorizer: authorizer}
	if len(encryptionKey) > 0 && strings.TrimSpace(encryptionKey[0]) != "" {
		logic.box, _ = secretbox.New(encryptionKey[0])
	}
	return logic
}

// List 集群列表
func (l *Logic) List(ctx context.Context, userID int64, projectID ...int64) ([]Cluster, int) {
	scope, err := l.authorizer.Scopes(ctx, userID, "cluster", "view")
	if err != nil {
		return nil, errcode.CodeDBError
	}
	requestedProject := int64(0)
	if len(projectID) > 0 {
		requestedProject = projectID[0]
	}
	clusters, err := l.repo.List(ctx, scope, requestedProject)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return clusters, errcode.CodeSuccess
}

// GetByID 集群详情
func (l *Logic) GetByID(ctx context.Context, userID, id int64) (*Cluster, int) {
	c, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if c == nil {
		return nil, errcode.CodeClusterNotFound
	}
	if code := l.authorize(ctx, userID, "view", id); code != errcode.CodeSuccess {
		return nil, code
	}
	return c, errcode.CodeSuccess
}

// Create 创建集群
func (l *Logic) Create(ctx context.Context, req *CreateClusterRequest, createdBy int64) (int64, int) {
	if req == nil || l.box == nil || createdBy <= 0 || !validClusterIdentity(req.Name, req.Code) {
		return 0, errcode.CodeParamInvalid
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.APIServer = strings.TrimSpace(req.APIServer)
	if !validAPIServer(req.APIServer) {
		return 0, errcode.CodeParamInvalid
	}
	if code := l.authorize(ctx, createdBy, "create", 0); code != errcode.CodeSuccess {
		return 0, code
	}
	existing, err := l.repo.FindByCode(ctx, req.Code)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if existing != nil {
		return 0, errcode.CodeParamInvalid
	}

	encryptedKubeconfig, err := encryptKubeconfig(l.box, req.Kubeconfig)
	if err != nil {
		return 0, errcode.CodeParamInvalid
	}
	c := &Cluster{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Provider:    req.Provider,
		Region:      req.Region,
		APIServer:   req.APIServer,
		Kubeconfig:  encryptedKubeconfig,
		Status:      "pending",
		CreatedBy:   createdBy,
	}
	if c.Provider == "" {
		c.Provider = "self"
	}

	id, err := l.repo.Create(ctx, c)
	if err != nil {
		return 0, errcode.CodeDBError
	}

	// 异步测试连接
	go l.testConnection(id, req.Kubeconfig)

	return id, errcode.CodeSuccess
}

// Update 更新集群
func (l *Logic) Update(ctx context.Context, userID, id int64, req *UpdateClusterRequest) int {
	if req == nil || l.box == nil || userID <= 0 || id <= 0 {
		return errcode.CodeParamInvalid
	}
	c, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if c == nil {
		return errcode.CodeClusterNotFound
	}
	if code := l.authorize(ctx, userID, "update", id); code != errcode.CodeSuccess {
		return code
	}

	if req.Name != "" {
		name := strings.TrimSpace(req.Name)
		if name == "" || len(name) > 64 {
			return errcode.CodeParamInvalid
		}
		c.Name = name
	}
	if req.Description != "" {
		c.Description = req.Description
	}
	if req.APIServer != "" {
		apiServer := strings.TrimSpace(req.APIServer)
		if !validAPIServer(apiServer) {
			return errcode.CodeParamInvalid
		}
		c.APIServer = apiServer
	}
	if req.Kubeconfig != "" {
		encrypted, err := encryptKubeconfig(l.box, req.Kubeconfig)
		if err != nil {
			return errcode.CodeParamInvalid
		}
		c.Kubeconfig = encrypted
		go l.testConnection(id, req.Kubeconfig)
	}

	if err := l.repo.Update(ctx, c); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

// Delete 删除集群
func (l *Logic) Delete(ctx context.Context, userID, id int64) int {
	c, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if c == nil {
		return errcode.CodeClusterNotFound
	}
	if code := l.authorize(ctx, userID, "delete", id); code != errcode.CodeSuccess {
		return code
	}
	used, err := l.repo.HasDeliveryReferences(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if used {
		return errcode.CodeClusterInUse
	}
	if err := l.repo.Delete(ctx, id); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

// TestConnect 手动测试连接
func (l *Logic) TestConnect(ctx context.Context, userID, id int64) int {
	c, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if c == nil {
		return errcode.CodeClusterNotFound
	}
	if code := l.authorize(ctx, userID, "test", id); code != errcode.CodeSuccess {
		return code
	}

	if l.box == nil || !strings.HasPrefix(c.Kubeconfig, "v1:") {
		return errcode.CodeInternalError
	}
	kubeconfig, err := l.box.Decrypt(c.Kubeconfig)
	if err != nil {
		return errcode.CodeInternalError
	}
	l.testConnection(id, kubeconfig)
	return errcode.CodeSuccess
}

func encryptKubeconfig(box *secretbox.Box, kubeconfig string) (string, error) {
	if box == nil || strings.TrimSpace(kubeconfig) == "" {
		return "", secretbox.ErrKeyRequired
	}
	if _, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig)); err != nil {
		return "", err
	}
	return box.Encrypt(kubeconfig)
}

func (l *Logic) authorize(ctx context.Context, userID int64, action string, clusterID int64) int {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "cluster", Action: action, ClusterID: clusterID})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}

// testConnection 测试集群连接并更新状态
func (l *Logic) testConnection(id int64, kubeconfig string) {
	// Bound discovery calls so a dead private endpoint cannot leak goroutines.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		_ = l.repo.UpdateStatus(ctx, id, "error", "", 0, "", "")
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		_ = l.repo.UpdateStatus(ctx, id, "error", "", 0, "", "")
		return
	}

	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		_ = l.repo.UpdateStatus(ctx, id, "disconnected", "", 0, "", "")
		return
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		_ = l.repo.UpdateStatus(ctx, id, "connected", version.GitVersion, 0, "", "")
		return
	}

	nodeCount := len(nodes.Items)
	var totalCPUMilli, totalMemBytes int64
	for _, node := range nodes.Items {
		cpu := node.Status.Capacity.Cpu()
		mem := node.Status.Capacity.Memory()
		if cpu != nil {
			totalCPUMilli += cpu.MilliValue()
		}
		if mem != nil {
			totalMemBytes += mem.Value()
		}
	}

	cpuStr := formatCPU(totalCPUMilli)
	memStr := formatMemory(totalMemBytes)

	_ = l.repo.UpdateStatus(ctx, id, "connected", version.GitVersion, nodeCount, cpuStr, memStr)
}

func formatCPU(milli int64) string {
	if milli <= 0 {
		return "0 cores"
	}
	return fmt.Sprintf("%.2f cores", float64(milli)/1000)
}

func formatMemory(bytes int64) string {
	if bytes <= 0 {
		return "0 GB"
	}
	return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
}

func validClusterIdentity(name, code string) bool {
	return strings.TrimSpace(name) != "" && len(strings.TrimSpace(name)) <= 64 &&
		clusterCodePattern.MatchString(strings.TrimSpace(code))
}

var clusterCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func validAPIServer(value string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

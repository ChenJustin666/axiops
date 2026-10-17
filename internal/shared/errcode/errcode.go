package errcode

// 统一错误码定义
// 格式：5 位数字
//   1xxxx - 认证和权限
//   2xxxx - 平台（项目、集群、资产）
//   3xxxx - CI/CD
//   4xxxx - 监控和日志
//   5xxxx - 系统错误

// 认证和权限
const (
	CodeSuccess          = 0
	CodeTokenInvalid     = 10001
	CodeTokenExpired     = 10002
	CodeNoPermission     = 10003
	CodeLoginFailed      = 10004
	CodeUserDisabled     = 10005
	CodeRefreshTokenFail = 10006
)

// 平台
const (
	CodeProjectNotFound     = 20001
	CodeProjectNameExists   = 20002
	CodeProjectInUse        = 20012
	CodeProjectDeleted      = 20013
	CodeClusterNotFound     = 20003
	CodeClusterConnectFail  = 20004
	CodeClusterInUse        = 20011
	CodeEnvironmentNotFound = 20005
	CodeCredentialNotFound  = 20006
	CodeHostNotFound        = 20007
	CodeUserNotFound        = 20008
	CodeRoleNotFound        = 20009
	CodeDepartmentNotFound  = 20010
)

// CI/CD
const (
	CodeBuildConfigNotFound      = 30001
	CodeBuildExecFailed          = 30002
	CodeCodeSourceNotFound       = 30003
	CodeServiceNotFound          = 30004
	CodePipelineNotFound         = 30005
	CodeReleaseNotFound          = 30006
	CodeApprovalNotFound         = 30007
	CodeWebhookInvalid           = 30008
	CodeCodeSourceCheckFail      = 30009
	CodeBuildPoolBusy            = 30010
	CodeServiceInUse             = 30011
	CodeServiceCodeExists        = 30012
	CodeServiceRevisionChanged   = 30013
	CodeServiceDefinitionInvalid = 30014
)

// 监控和日志
const (
	CodeMetricQueryFailed = 40001
	CodeLogQueryFailed    = 40002
	CodeAlertRuleInvalid  = 40003
)

// 系统错误
const (
	CodeInternalError = 50001
	CodeDBError       = 50002
	CodeRedisError    = 50003
	CodeParamInvalid  = 50004
	CodeNotFound      = 50005
)

// 错误码对应的消息
var codeMessages = map[int]string{
	CodeServiceCodeExists:        "服务编码已存在；请在原服务选择“更换模板”；若已删除，请先到“已删除服务”恢复",
	CodeServiceRevisionChanged:   "服务配置已被更新，请刷新页面后重新选择模板；本次未保存",
	CodeServiceDefinitionInvalid: "服务 YAML 结构校验失败，本次未保存",
	CodeServiceInUse:             "服务仍有环境、工作流引用或未结束的任务，请处理后重试",
	CodeSuccess:                  "success",
	CodeTokenInvalid:             "Token 无效",
	CodeTokenExpired:             "Token 已过期",
	CodeNoPermission:             "无权限",
	CodeLoginFailed:              "用户名或密码错误",
	CodeUserDisabled:             "用户已禁用",
	CodeRefreshTokenFail:         "刷新 Token 失败",

	CodeProjectNotFound:     "项目不存在",
	CodeProjectNameExists:   "项目名称已存在",
	CodeProjectInUse:        "项目仍有未结束任务，请等待完成或取消后再删除",
	CodeProjectDeleted:      "项目已删除，仅支持查看历史记录",
	CodeClusterNotFound:     "集群不存在",
	CodeClusterConnectFail:  "集群连接失败",
	CodeClusterInUse:        "集群已被 CI/CD 环境或发布记录引用，不能删除",
	CodeEnvironmentNotFound: "环境不存在",
	CodeCredentialNotFound:  "凭据不存在",
	CodeHostNotFound:        "主机不存在",
	CodeUserNotFound:        "用户不存在",
	CodeRoleNotFound:        "角色不存在",
	CodeDepartmentNotFound:  "部门不存在",

	CodeBuildConfigNotFound: "构建配置不存在",
	CodeBuildExecFailed:     "构建执行失败",
	CodeCodeSourceNotFound:  "代码源不存在",
	CodeServiceNotFound:     "服务不存在",
	CodePipelineNotFound:    "流水线不存在",
	CodeReleaseNotFound:     "发布记录不存在",
	CodeApprovalNotFound:    "审批记录不存在",
	CodeWebhookInvalid:      "Webhook 校验失败",
	CodeCodeSourceCheckFail: "代码源连接失败",
	CodeBuildPoolBusy:       "构建池仍有运行或待清理的任务，请停用新任务派发，等待任务结束后再更换集群、Namespace 或节点策略",

	CodeMetricQueryFailed: "指标查询失败",
	CodeLogQueryFailed:    "日志查询失败",
	CodeAlertRuleInvalid:  "告警规则无效",

	CodeInternalError: "系统内部错误",
	CodeDBError:       "数据库错误",
	CodeRedisError:    "缓存错误",
	CodeParamInvalid:  "参数无效",
	CodeNotFound:      "资源不存在",
}

// Message 获取错误码对应的消息
func Message(code int) string {
	if msg, ok := codeMessages[code]; ok {
		return msg
	}
	return "未知错误"
}

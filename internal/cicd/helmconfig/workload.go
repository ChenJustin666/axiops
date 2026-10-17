package helmconfig

import (
	"fmt"
	"net"
	"strings"

	"github.com/robfig/cron/v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Platform built-in safety checks supplement the Chart's JSON schema. Third
// party Charts retain their own values/schema contract.
func ValidateWorkloadValues(c *chart.Chart, values, plain map[string]interface{}) error {
	if c.Name() == "axiops-sinozo-java" {
		return validateJavaServiceValues(c, values)
	}
	if c.Name() != "axiops-workload" {
		return nil
	}
	for _, config := range object(plain["secrets"]) {
		if len(object(object(config)["stringData"])) > 0 || len(object(object(config)["data"])) > 0 {
			return fmt.Errorf("Secret 内容请通过安全凭据绑定，不能在 values 中保存明文")
		}
		for _, field := range object(object(config)["fields"]) {
			if _, exists := object(field)["value"]; exists {
				return fmt.Errorf("Secret 字段正文请通过安全凭据绑定")
			}
		}
	}
	for _, config := range object(c.Values["secrets"]) {
		if len(object(object(config)["stringData"])) > 0 || len(object(object(config)["data"])) > 0 {
			return fmt.Errorf("通用 Chart 不允许在默认值中保存 Secret 正文")
		}
		for _, field := range object(object(config)["fields"]) {
			if _, exists := object(field)["value"]; exists {
				return fmt.Errorf("通用 Chart 不允许在默认值中保存 Secret 字段正文")
			}
		}
	}
	merged, err := chartutil.CoalesceValues(c, values)
	if err != nil {
		return err
	}
	resources := object(merged["resources"])
	if err := validateResourceQuantities(resources); err != nil {
		return err
	}
	return validateUniversalWorkloadValues(merged)
}

func validateResourceQuantities(resources map[string]interface{}) error {
	for _, field := range []string{"requests", "limits"} {
		for key, value := range object(resources[field]) {
			quantity, err := resource.ParseQuantity(fmt.Sprint(value))
			if err != nil || quantity.Sign() < 0 || (key == "memory" && quantity.Value() < 1024*1024) {
				return fmt.Errorf("resources.%s.%s 数量无效；内存请使用 Mi/Gi 等有效单位", field, key)
			}
			if field == "requests" {
				if limit, exists := object(resources["limits"])[key]; exists {
					bound, e := resource.ParseQuantity(fmt.Sprint(limit))
					if e != nil || quantity.Cmp(bound) > 0 {
						return fmt.Errorf("资源请求不能大于上限：%s", key)
					}
				}
			}
		}
	}
	return nil
}

func validateUniversalWorkloadValues(merged map[string]interface{}) error {
	persistence := object(merged["persistence"])
	if persistence["enabled"] == true && persistence["existingClaim"] == "" {
		q, e := resource.ParseQuantity(fmt.Sprint(persistence["size"]))
		if e != nil || q.Sign() <= 0 {
			return fmt.Errorf("PVC 存储容量无效")
		}
	}
	if object(merged["workload"])["kind"] == "CronJob" {
		if _, err := cron.ParseStandard(fmt.Sprint(object(merged["cronJob"])["schedule"])); err != nil {
			return fmt.Errorf("CronJob schedule 无效，请填写五段 cron 表达式")
		}
	}
	for _, kind := range []string{"startup", "liveness"} {
		probe := object(object(merged["probes"])[kind])
		if threshold, exists := probe["successThreshold"]; exists && fmt.Sprint(threshold) != "1" {
			return fmt.Errorf("%s 探针 successThreshold 必须为 1", kind)
		}
	}
	for key := range object(merged["podLabels"]) {
		if key == "app.kubernetes.io/name" || key == "app.kubernetes.io/instance" || key == "app.kubernetes.io/managed-by" {
			return fmt.Errorf("podLabels 不能覆盖 Chart 的工作负载标识")
		}
	}
	if object(merged["monitoring"])["enabled"] == true {
		for key := range object(merged["podAnnotations"]) {
			if key == "prometheus.io/path" || key == "prometheus.io/port" || key == "prometheus.io/scrape" {
				return fmt.Errorf("监控已启用，请通过 monitoring 配置采集参数")
			}
		}
	}
	names := map[string]bool{"application": true}
	for _, field := range []string{"sidecars", "initContainers"} {
		items, _ := merged[field].([]interface{})
		for _, item := range items {
			name := fmt.Sprint(object(item)["name"])
			if names[name] {
				return fmt.Errorf("容器名称不能重复")
			}
			names[name] = true
		}
	}
	if err := validateWorkloadReferences(merged); err != nil {
		return err
	}
	return nil
}

func validateWorkloadReferences(values map[string]interface{}) error {
	for _, field := range []string{"podLabels", "nodeSelector"} {
		for key, value := range object(values[field]) {
			if len(validation.IsQualifiedName(key)) > 0 || len(validation.IsValidLabelValue(fmt.Sprint(value))) > 0 {
				return fmt.Errorf("%s 中的标签键或值无效", field)
			}
		}
	}
	for key := range object(values["podAnnotations"]) {
		if len(validation.IsQualifiedName(key)) > 0 {
			return fmt.Errorf("podAnnotations 注解键无效")
		}
	}
	ingress := object(values["ingress"])
	if class, ok := ingress["className"].(string); ok && class != "" && len(validation.IsDNS1123Subdomain(class)) > 0 {
		return fmt.Errorf("ingress.className 必须为合法 IngressClass 名称")
	}
	hosts, _ := ingress["hosts"].([]interface{})
	for _, item := range hosts {
		host := fmt.Sprint(object(item)["host"])
		valid := validation.IsDNS1123Subdomain(host)
		if strings.HasPrefix(host, "*.") {
			valid = validation.IsWildcardDNS1123Subdomain(host)
		}
		if net.ParseIP(host) != nil || len(valid) > 0 {
			return fmt.Errorf("Ingress host 必须为合法域名，不能使用 IP 或 URL")
		}
	}
	service := object(values["service"])
	if service["type"] == "ClusterIP" && service["nodePort"] != nil {
		return fmt.Errorf("ClusterIP 不能指定 nodePort")
	}
	persistence := object(values["persistence"])
	if claim, ok := persistence["existingClaim"].(string); ok && claim != "" && len(validation.IsDNS1123Subdomain(claim)) > 0 {
		return fmt.Errorf("persistence.existingClaim 名称无效")
	}
	access, _ := persistence["accessModes"].([]interface{})
	for _, mode := range access {
		if mode == "ReadWriteOncePod" && len(access) != 1 {
			return fmt.Errorf("ReadWriteOncePod 必须单独使用")
		}
	}
	if object(values["autoscaling"])["enabled"] == true {
		for metric, request := range map[string]string{"targetCPUUtilizationPercentage": "cpu", "targetMemoryUtilizationPercentage": "memory"} {
			if object(values["autoscaling"])[metric] != nil {
				q, e := resource.ParseQuantity(fmt.Sprint(object(object(values["resources"])["requests"])[request]))
				if e != nil || q.Sign() <= 0 {
					return fmt.Errorf("HPA %s 指标需要正数的 requests.%s", request, request)
				}
			}
		}
	}
	pdb := object(values["pdb"])
	if pdb["enabled"] == true && pdb["minAvailable"] == nil && pdb["maxUnavailable"] == nil {
		return fmt.Errorf("PDB 需要设置 minAvailable 或 maxUnavailable")
	}
	volumes := map[string]bool{}
	mounts := map[string]bool{}
	addVolume := func(name string) error {
		if len(validation.IsDNS1123Label(name)) > 0 || volumes[name] {
			return fmt.Errorf("卷名称无效或重复")
		}
		volumes[name] = true
		return nil
	}
	addMount := func(raw interface{}) error {
		path, ok := raw.(string)
		if !ok || !strings.HasPrefix(path, "/") || mounts[path] {
			return fmt.Errorf("挂载目录必须是唯一的绝对路径")
		}
		mounts[path] = true
		return nil
	}
	if persistence["enabled"] == true {
		volumes["data"] = true
		if err := addMount(persistence["mountPath"]); err != nil {
			return err
		}
	}
	for name, raw := range object(values["configMaps"]) {
		config := object(raw)
		for key := range object(config["data"]) {
			if len(validation.IsConfigMapKey(key)) > 0 {
				return fmt.Errorf("ConfigMap 数据键无效")
			}
		}
		if path, exists := config["mountPath"]; exists {
			if err := addVolume("config-" + name); err != nil {
				return err
			}
			if err := addMount(path); err != nil {
				return err
			}
		}
	}
	for _, raw := range object(values["secrets"]) {
		secret := object(raw)
		for key := range object(secret["stringData"]) {
			if len(validation.IsConfigMapKey(key)) > 0 {
				return fmt.Errorf("Secret 数据键无效")
			}
		}
		for _, rawField := range object(secret["fields"]) {
			if key, ok := object(rawField)["key"].(string); ok && len(validation.IsConfigMapKey(key)) > 0 {
				return fmt.Errorf("Secret 字段键无效")
			}
		}
	}
	secretMounts, _ := values["secretMounts"].([]interface{})
	for i, raw := range secretMounts {
		if err := addVolume(fmt.Sprintf("secret-%d", i)); err != nil {
			return err
		}
		if err := addMount(object(raw)["mountPath"]); err != nil {
			return err
		}
	}
	claims, _ := object(values["statefulSet"])["volumeClaimTemplates"].([]interface{})
	if object(values["workload"])["kind"] == "StatefulSet" {
		for _, raw := range claims {
			if err := addVolume(fmt.Sprint(object(object(raw)["metadata"])["name"])); err != nil {
				return err
			}
		}
	}
	nativeVolumes, _ := values["volumes"].([]interface{})
	for _, raw := range nativeVolumes {
		if err := addVolume(fmt.Sprint(object(raw)["name"])); err != nil {
			return err
		}
	}
	nativeMounts, _ := values["volumeMounts"].([]interface{})
	for _, raw := range nativeMounts {
		mount := object(raw)
		if !volumes[fmt.Sprint(mount["name"])] {
			return fmt.Errorf("volumeMounts 引用了不存在的卷")
		}
		if err := addMount(mount["mountPath"]); err != nil {
			return err
		}
	}
	return nil
}

func object(value interface{}) map[string]interface{} {
	if m, ok := value.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

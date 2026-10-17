package helmconfig

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"
)

// This is the built-in Java Chart's contract, not a restriction on imported Charts.
func validateJavaServiceValues(c *chart.Chart, values map[string]interface{}) error {
	v, err := chartutil.CoalesceValues(c, values)
	if err != nil {
		return err
	}
	if err := chartutil.ValidateAgainstSchema(c, v); err != nil {
		return err
	}
	if object(v["resources"])["enabled"] == true {
		if err := validateResourceQuantities(object(v["resources"])); err != nil {
			return err
		}
	}
	app, java, nacos, sw := object(v["application"]), object(v["java"]), object(v["nacos"]), object(v["skywalking"])
	if name, _ := object(v["service"])["name"].(string); name != "" && len(validation.IsDNS1035Label(name)) > 0 {
		return fmt.Errorf("service.name 必须为合法的 Service 名称（小写字母开头，最多 63 字符）")
	}
	if java["enabled"] == true && object(v["resources"])["enabled"] == true {
		if err := validateJavaMemory(java, object(v["resources"])); err != nil {
			return err
		}
	}
	if nacos["enabled"] == true {
		if strings.TrimSpace(fmt.Sprint(nacos["serverAddr"])) == "" || strings.Contains(fmt.Sprint(nacos["serverAddr"]), ".invalid") {
			return fmt.Errorf("请填写实际的 nacos.serverAddr")
		}
		for _, endpoint := range strings.Split(fmt.Sprint(nacos["serverAddr"]), ",") {
			if !validJavaEndpoint(endpoint) {
				return fmt.Errorf("nacos.serverAddr 请填写 host:port，多个地址用逗号分隔")
			}
		}
		if object(nacos["auth"])["enabled"] == true && object(nacos["auth"])["secretName"] == "" {
			return fmt.Errorf("Nacos 认证需要目标环境的 Secret 引用")
		}
	} else if object(nacos["auth"])["enabled"] == true {
		return fmt.Errorf("开启 Nacos 认证前必须启用 Nacos")
	}
	if sw["enabled"] == true {
		if java["enabled"] != true {
			return fmt.Errorf("SkyWalking Java Agent 需要启用 Java")
		}
		for _, endpoint := range strings.Split(fmt.Sprint(sw["backendService"]), ",") {
			if !validJavaEndpoint(endpoint) {
				return fmt.Errorf("skywalking.backendService 请填写 OAP host:port，多个地址用逗号分隔")
			}
		}
		if err := validateResourceQuantities(object(sw["initResources"])); err != nil {
			return err
		}
	}
	dump := object(java["heapDump"])
	if dump["enabled"] == true {
		if java["enabled"] != true {
			return fmt.Errorf("HeapDump 需要启用 Java")
		}
		if dump["existingClaim"] == "" {
			q, err := resource.ParseQuantity(fmt.Sprint(dump["sizeLimit"]))
			if err != nil || q.Sign() <= 0 {
				return fmt.Errorf("HeapDump sizeLimit 容量无效")
			}
		}
	}
	probes := object(v["probes"])
	if probes["enabled"] == true && probes["type"] == "http" && !strings.HasPrefix(fmt.Sprint(probes["path"]), "/") {
		return fmt.Errorf("HTTP 探针必须填写以 / 开头的 path")
	}
	// Explicit env has precedence over envFrom; reject accidental replacement of
	// managed fields instead of silently accepting an environment-specific mismatch.
	managed := map[string]bool{"SPRING_APPLICATION_NAME": true}
	generatedCollision := ""
	add := func(name string, enabled bool) {
		if name != "" && enabled {
			if managed[name] {
				generatedCollision = name
			}
			managed[name] = true
		}
	}
	add(fmt.Sprint(app["portEnv"]), true)
	add("SPRING_PROFILES_ACTIVE", app["profile"] != "")
	add("LOG_LEVEL", app["logLevel"] != "")
	add("TZ", app["timezone"] != "")
	add("POD_IP", app["podIP"] == true)
	add("JAVA_OPTS", java["enabled"] == true)
	add("JAVA_TOOL_OPTIONS", java["enabled"] == true)
	if nacos["enabled"] == true {
		add("SPRING_CONFIG_IMPORT", object(nacos["configImport"])["enabled"] == true)
		for _, scope := range []string{"DISCOVERY", "CONFIG"} {
			for _, field := range []string{"SERVERADDR", "NAMESPACE", "GROUP"} {
				add("SPRING_CLOUD_NACOS_"+scope+"_"+field, true)
			}
		}
		add("SPRING_CLOUD_NACOS_CONFIG_FILEEXTENSION", true)
		add("SPRING_CLOUD_NACOS_CONFIG_REFRESHENABLED", true)
		for _, scope := range []string{"", "DISCOVERY_", "CONFIG_"} {
			for _, key := range []string{"USERNAME", "PASSWORD"} {
				add("SPRING_CLOUD_NACOS_"+scope+key, object(nacos["auth"])["enabled"] == true)
			}
		}
	}
	for _, key := range []string{"SW_AGENT_NAME", "SW_AGENT_COLLECTOR_BACKEND_SERVICES", "SW_LOGGING_OUTPUT", "SW_EXCLUDE_PLUGINS"} {
		add(key, sw["enabled"] == true)
	}
	if generatedCollision != "" {
		return fmt.Errorf("application.portEnv 与模板生成变量 %s 冲突", generatedCollision)
	}
	seenEnv := map[string]bool{}
	for _, raw := range array(v["env"]) {
		name := fmt.Sprint(object(raw)["name"])
		if managed[name] || seenEnv[name] {
			return fmt.Errorf("环境变量 %s 重复或由模板管理，请修改对应参数", name)
		}
		seenEnv[name] = true
	}
	if java["enabled"] == true {
		opts := fmt.Sprint(java["opts"])
		if regexp.MustCompile(`(?i)(?:^|\s)-Dspring\.(application\.name|profiles\.active|config\.import|cloud\.nacos\.)`).MatchString(opts) || strings.Contains(opts, "-javaagent:") && sw["enabled"] == true || app["portEnv"] != "" && regexp.MustCompile(`(?:^|\s)-Dserver\.port=`).MatchString(opts) {
			return fmt.Errorf("java.opts 只维护 JVM 参数；Spring/Nacos 请用独立参数，SkyWalking Agent 由开关管理")
		}
	}
	if object(object(v["deployment"])["strategy"])["type"] == "Recreate" && len(object(object(object(v["deployment"])["strategy"])["rollingUpdate"])) > 0 {
		return fmt.Errorf("Recreate 策略请同时设置 deployment.strategy.rollingUpdate: null")
	}
	// Reuse the native-volume and label checks, adding the automatic agent/dump.
	native := map[string]interface{}{}
	for k, value := range v {
		native[k] = value
	}
	native["podAnnotations"] = v["extraAnnotations"]
	volumes := append([]interface{}{}, array(v["volumes"])...)
	mounts := append([]interface{}{}, array(v["volumeMounts"])...)
	for _, extra := range []struct {
		enabled    bool
		name, path string
	}{
		{sw["enabled"] == true, "skywalking-agent", "/skywalking"},
		{dump["enabled"] == true, "heapdump", fmt.Sprint(dump["mountPath"])},
	} {
		if extra.enabled {
			volumes = append(volumes, map[string]interface{}{"name": extra.name})
			mounts = append(mounts, map[string]interface{}{"name": extra.name, "mountPath": extra.path})
		}
	}
	native["volumes"], native["volumeMounts"] = volumes, mounts
	native["ingress"] = map[string]interface{}{}
	if err := validateWorkloadReferences(native); err != nil {
		return err
	}
	for key := range object(v["podLabels"]) {
		if strings.HasPrefix(key, "app.kubernetes.io/") {
			return fmt.Errorf("podLabels 不能覆盖 Chart 标识")
		}
	}
	if object(v["monitoring"])["enabled"] == true {
		for _, key := range []string{"prometheus.io/scrape", "prometheus.io/path", "prometheus.io/port"} {
			if _, exists := object(v["extraAnnotations"])[key]; exists {
				return fmt.Errorf("监控已启用，请通过 monitoring 配置采集路径和端口")
			}
		}
	}
	names := map[string]bool{"application": true}
	if sw["enabled"] == true {
		names["skywalking-agent-init"] = true
	}
	for _, field := range []string{"initContainers", "sidecars"} {
		for _, raw := range array(v[field]) {
			name := fmt.Sprint(object(raw)["name"])
			if len(validation.IsDNS1123Label(name)) > 0 || names[name] {
				return fmt.Errorf("附加容器名称无效或重复")
			}
			names[name] = true
		}
	}
	return validateJavaIngress(v)
}

func validJavaEndpoint(endpoint string) bool {
	host, port, err := net.SplitHostPort(strings.TrimSpace(endpoint))
	p, e := strconv.Atoi(port)
	return err == nil && e == nil && p >= 1 && p <= 65535 && (net.ParseIP(host) != nil || len(validation.IsDNS1123Subdomain(host)) == 0)
}

// Catch clearly impossible memory budgets; native/thread/agent overhead still
// depends on the application and must be observed under real load.
func validateJavaMemory(java, resources map[string]interface{}) error {
	limit, exists := object(resources["limits"])["memory"]
	if !exists {
		return nil
	}
	bound, err := resource.ParseQuantity(fmt.Sprint(limit))
	if err != nil {
		return err
	}
	opts := fmt.Sprint(java["opts"])
	heap, meta := float64(0), float64(0)
	if m := regexp.MustCompile(`(?:^|\s)-XX:MaxRAMPercentage=([0-9.]+)`).FindStringSubmatch(opts); len(m) > 0 {
		p, e := strconv.ParseFloat(m[1], 64)
		if e != nil || p <= 0 || p > 100 {
			return fmt.Errorf("MaxRAMPercentage 必须大于 0 且不超过 100")
		}
		heap = float64(bound.Value()) * p / 100
	}
	for _, arg := range []struct {
		pattern string
		dest    *float64
	}{
		{`(?:^|\s)-Xmx([0-9]+[kKmMgG]?)`, &heap},
		{`(?:^|\s)-XX:MaxMetaspaceSize=([0-9]+[kKmMgG]?)`, &meta},
	} {
		if m := regexp.MustCompile(arg.pattern).FindStringSubmatch(opts); len(m) > 0 {
			quantity := m[1]
			last := strings.ToUpper(quantity[len(quantity)-1:])
			if last == "K" || last == "M" || last == "G" {
				quantity = quantity[:len(quantity)-1] + last + "i"
			}
			q, e := resource.ParseQuantity(quantity)
			if e != nil || q.Sign() <= 0 {
				return fmt.Errorf("JVM 内存参数无效")
			}
			*arg.dest = float64(q.Value())
		}
	}
	if heap+meta >= float64(bound.Value()) {
		return fmt.Errorf("JVM 堆与元空间预算已达到容器内存上限；请增加内存或降低堆比例/元空间，并为线程、直接内存和 Agent 留出空间")
	}
	return nil
}

func validateJavaIngress(v map[string]interface{}) error {
	ingress := object(v["ingress"])
	if ingress["enabled"] != true {
		return nil
	}
	if object(v["service"])["enabled"] != true || len(array(ingress["items"])) == 0 {
		return fmt.Errorf("启用 Ingress 需要当前服务的 Service 和至少一条路由")
	}
	names := map[string]bool{}
	for _, raw := range array(ingress["items"]) {
		item := object(raw)
		name := fmt.Sprint(item["name"])
		if names[name] {
			return fmt.Errorf("Ingress 路由名称不能重复")
		}
		names[name] = true
		class, _ := item["className"].(string)
		if class != "" && len(validation.IsDNS1123Subdomain(class)) > 0 {
			return fmt.Errorf("IngressClass 名称无效")
		}
		for _, field := range []string{"labels", "annotations"} {
			for key, value := range object(item[field]) {
				if len(validation.IsQualifiedName(key)) > 0 || field == "labels" && len(validation.IsValidLabelValue(fmt.Sprint(value))) > 0 {
					return fmt.Errorf("Ingress %s 中的键或值无效", field)
				}
			}
		}
		tls := object(item["tls"])
		hosts := map[string]bool{}
		for _, rawHost := range array(item["hosts"]) {
			host := fmt.Sprint(object(rawHost)["host"])
			if host != "" && !validJavaHost(host) {
				return fmt.Errorf("Ingress host 必须为域名，不能使用 URL 或 IP")
			}
			if tls["enabled"] == true && host == "" {
				return fmt.Errorf("TLS 路由必须填写域名")
			}
			hosts[host] = true
		}
		if tls["enabled"] == true {
			if class == "nginx" && (tls["secretName"] == nil || tls["secretName"] == "") {
				return fmt.Errorf("Nginx TLS 需要 tls.secretName")
			}
			for _, rawHost := range array(tls["hosts"]) {
				host, _ := rawHost.(string)
				if !hosts[host] || !validJavaHost(host) {
					return fmt.Errorf("TLS hosts 必须匹配当前路由的域名")
				}
			}
		}
	}
	return nil
}

func validJavaHost(host string) bool {
	if net.ParseIP(host) != nil {
		return false
	}
	if strings.HasPrefix(host, "*.") {
		return len(validation.IsWildcardDNS1123Subdomain(host)) == 0
	}
	return host != "" && len(validation.IsDNS1123Subdomain(host)) == 0
}

func array(value interface{}) []interface{} {
	items, _ := value.([]interface{})
	return items
}

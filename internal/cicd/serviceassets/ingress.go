package serviceassets

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

func validIngressHost(host string) bool {
	errors := validation.IsDNS1123Subdomain(host)
	if strings.HasPrefix(host, "*.") {
		errors = validation.IsWildcardDNS1123Subdomain(host)
	}
	return len(errors) == 0 && net.ParseIP(host) == nil
}

func routePathType(r Route) string {
	if r.PathType == "" {
		return "Prefix" // Preserve existing saved routes.
	}
	return r.PathType
}

func validateRoute(r Route) error {
	pathType := routePathType(r)
	if pathType != "Prefix" && pathType != "Exact" && pathType != "ImplementationSpecific" {
		return fmt.Errorf("路由路径类型须为 Prefix、Exact 或 ImplementationSpecific")
	}
	invalidPath := !strings.HasPrefix(r.Path, "/") || strings.ContainsAny(r.Path, "\x00\n\r\t ")
	if pathType != "ImplementationSpecific" {
		invalidPath = invalidPath || strings.ContainsAny(r.Path, "?#\\") || strings.Contains(r.Path, "//")
		for _, segment := range strings.Split(r.Path, "/") {
			invalidPath = invalidPath || segment == "." || segment == ".."
		}
	}
	if !validIngressHost(r.Host) || invalidPath || !validName(r.Service) || r.Port < 1 || r.Port > 65535 || (r.Class != "" && !validName(r.Class)) || (r.TLSSecret != "" && !validName(r.TLSSecret)) {
		return fmt.Errorf("域名、路径、服务端口、Ingress Class或TLS Secret无效")
	}
	if len(r.TLSHosts) > 20 || (!r.TLSEnabled && r.TLSSecret == "" && len(r.TLSHosts) > 0) {
		return fmt.Errorf("TLS 关闭时不能配置证书域名，开启时最多20个域名")
	}
	hosts := map[string]bool{}
	if r.TLSEnabled && len(r.TLSHosts) == 0 {
		return fmt.Errorf("开启 TLS 后请填写证书域名")
	}
	for _, host := range r.TLSHosts {
		if !validIngressHost(host) || hosts[host] {
			return fmt.Errorf("TLS 域名无效或重复")
		}
		hosts[host] = true
	}
	if len(r.Annotations) > 50 || len(r.Labels) > 50 {
		return fmt.Errorf("Ingress 注解或标签最多50项")
	}
	size := 0
	for key, value := range r.Annotations {
		size += len(key) + len(value)
		if len(validation.IsQualifiedName(key)) > 0 || strings.HasPrefix(key, "axiops.io/") || strings.ContainsRune(value, '\x00') {
			return fmt.Errorf("Ingress 注解名称或值无效，axiops.io/ 为系统保留前缀")
		}
	}
	if size > 256*1024 {
		return fmt.Errorf("Ingress 注解不能超过256KiB")
	}
	for key, value := range r.Labels {
		if len(validation.IsQualifiedName(key)) > 0 || len(validation.IsValidLabelValue(value)) > 0 || strings.HasPrefix(key, "axiops.io/") {
			return fmt.Errorf("Ingress 标签名称或值无效，axiops.io/ 为系统保留前缀")
		}
	}
	if value, ok := r.Annotations["alb.ingress.kubernetes.io/use-regex"]; ok && value != "true" && value != "false" {
		return fmt.Errorf("ALB use-regex 须为字符串 true 或 false")
	}
	if value, ok := r.Annotations["alb.ingress.kubernetes.io/order"]; ok {
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("ALB order 须为整数")
		}
	}
	if value, ok := r.Annotations["alb.ingress.kubernetes.io/listen-ports"]; ok {
		var listeners []map[string]json.RawMessage
		if err := json.Unmarshal([]byte(value), &listeners); err != nil || len(listeners) == 0 {
			return fmt.Errorf("ALB listen-ports 须为 JSON 数组，例如 [{\"HTTP\":80},{\"HTTPS\":443}]")
		}
		ports := map[int64]bool{}
		for _, listener := range listeners {
			if len(listener) != 1 {
				return fmt.Errorf("ALB 每个监听项只能指定一种协议和端口")
			}
			for protocol, number := range listener {
				var port int64
				err := json.Unmarshal(number, &port)
				if (protocol != "HTTP" && protocol != "HTTPS") || err != nil || port < 1 || port > 65535 || ports[port] {
					return fmt.Errorf("ALB 监听须使用 HTTP/HTTPS 和1到65535的不重复整数端口")
				}
				ports[port] = true
			}
		}
	}
	return nil
}

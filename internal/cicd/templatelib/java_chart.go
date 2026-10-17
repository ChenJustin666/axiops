package templatelib

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed charts/java-service/* charts/java-service/templates/*
var javaServiceChart embed.FS

func javaServiceTemplate() Template {
	files := []File{}
	_ = fs.WalkDir(javaServiceChart, "charts/java-service", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := javaServiceChart.ReadFile(name)
		if err == nil {
			files = append(files, File{Path: name[len("charts/java-service/"):], Content: string(data)})
		}
		return err
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return Template{Kind: "helm", Name: "Sinozo · Java / Nacos / SkyWalking", Description: "Java 服务标准：独立环境参数、可选 Nacos 与安全认证、SkyWalking Agent、TCP/HTTP 探针、Prometheus、ALB/Nginx 多路由与 TLS。Service/资源默认启用，端口联动；支持原生 env、调度、挂载及启动命令。", BuiltinKey: "helm-sinozo-java-v1", Files: files, Variables: []Variable{}}
}

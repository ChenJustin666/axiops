package templatelib

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed charts/workload/* charts/workload/templates/*
var workloadChart embed.FS

func workloadTemplate() Template {
	files := []File{}
	_ = fs.WalkDir(workloadChart, "charts/workload", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, e := workloadChart.ReadFile(name)
		if e != nil {
			return e
		}
		files = append(files, File{Path: name[len("charts/workload/"):], Content: string(data)})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return Template{Kind: "helm", Name: "通用应用 · 工作负载与配套资源", Description: "Deployment / StatefulSet / DaemonSet / Job / CronJob；按需配置路由、存储、配置文件、安全凭据、探针、弹性伸缩与调度。每个服务、环境独立 values。", BuiltinKey: "helm-workload-v1", Files: files, Variables: []Variable{}}
}

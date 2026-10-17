# 阿里云 ALB Ingress 配置与使用

## 从模板创建

1. 在「服务与配置」选择目标环境，再选择服务模板。
2. 添加「路由 Ingress」，填写域名、路径，Ingress Class 选择 `alb`（或集群实际的 ALB Class 名称）。
3. 路径类型默认 `Prefix`；精确路径选择 `Exact`；正则或控制器专有规则选择 `ImplementationSpecific`，具体效果由集群控制器版本决定。
4. 注解和标签默认为空，按需展开后填写 YAML 键值。无需附带 `annotations:` / `labels:` 外层。
5. TLS 默认关闭。开启后填写 TLS 域名；ALB 使用云证书时 Secret 留空。使用 Kubernetes TLS Secret 时填写目标命名空间里的 Secret 名称。
6. 完成添加后预览完整 YAML，确认目标环境、namespace、Service 名称和端口，再保存。保存配置不等于部署；发布走该环境的工作流。

后端自动取模板中的 `Service.metadata.name` 和 `Service.spec.ports[0].port`，不会取容器端口，也不根据 Deployment 名称猜测。例如 Service 名为 `ad-gateway-svc`、port 为 `9099`，Ingress 就指向 `ad-gateway-svc:9099`。创建后在配套配置中可选择当前模板已有 Service 的其他端口。

## 可选注解与标签

Annotations 输入示例：

```yaml
alb.ingress.kubernetes.io/order: "1"
alb.ingress.kubernetes.io/listen-ports: '[{"HTTP":80},{"HTTPS":443}]'
```

Labels 输入示例：

```yaml
ingress-controller: alb
```

控制器选择以 `ingressClassName` 为准；自定义 label 本身不负责选择控制器。ALB 监听配置和云证书要与当前 AlbConfig/控制器设置一致；开启 TLS 并不会自动保证证书、监听、DNS 全部可用。

正则重写按需添加，例如匹配 `/api(/|$)(.*)` 并重写为 `/${2}`：

```yaml
alb.ingress.kubernetes.io/use-regex: "true"
alb.ingress.kubernetes.io/rewrite-target: /${2}
```

`${2}` 是控制器的捕获组引用，平台会原样保留。普通 `path: /` 没有第二个捕获组，无需添加上述两个注解。不同 ALB Controller 版本的正则和路径支持可能有区别。

## 直接编辑原生 Ingress

支持在模板库的 Kubernetes YAML 模板中增加 `ingress.yaml`，也支持在服务与配置的文件目录中新建 YAML，或直接修改当前环境已有 Ingress。已有原生 Ingress 时，修改其 YAML 即可，不再从可选资源入口重复添加。

下例假设同一模板的 Service 名是 `{{.Service}}-svc`，Service 的 `port` 使用 `{{.port}}`：

```yaml
# AxiOps-File: ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{.Service}}-ingress
  labels:
    ingress-controller: alb
spec:
  ingressClassName: alb
  rules:
    - host: {{.domain}}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: {{.Service}}-svc
                port:
                  number: {{.port}}
```

在模板参数中添加 `domain`，创建时填入当前环境域名。若模板 Service 名是 `{{.Service}}`，将 backend 的 name 改为 `{{.Service}}`；若 Service port 是固定的 `80`，这里也填写 `80`。原生 YAML 的 backend 由你明确编写，可选路由表单才负责自动匹配。

不要把 namespace 固定写成 `adv-prod` 来复用不同环境；省略 namespace，由平台按当前环境注入。完全手动的 YAML 也可以使用固定服务名/端口/域名，以及平台系统变量 `{{.Service}}`、`{{.Namespace}}` 等。

ALB 云证书 TLS 开启示例，在同一 `spec` 下添加：

```yaml
  tls:
    - hosts:
        - {{.domain}}
```

TLS 默认关闭时不写 `tls`。需要 Kubernetes 证书时在对应条目增加 `secretName: your-tls-secret`。Secret 需包含 `tls.crt` 与 `tls.key`。

部署后在目标环境查看 Ingress 地址、事件、Service 与 Endpoint/Pod 就绪状态，再访问域名确认路由。HTTP 404 常检查 host/path/controller；503 常检查 Service 后端是否就绪；HTTPS 失败常检查监听、证书匹配和 DNS。

官方参考：[ALB Controller 使用说明与自动证书发现](https://github.com/alibaba/alibaba-load-balancer-controller/blob/main/docs/usage-alb.md)。该仓库部分示例使用旧版 API，本文统一使用 `networking.k8s.io/v1`；高级注解效果以当前集群安装版本为准。

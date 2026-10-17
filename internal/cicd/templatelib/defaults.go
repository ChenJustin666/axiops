package templatelib

import (
	"encoding/json"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
)

func Defaults() []Template {
	items := []Template{
		{Kind: "k8s_yaml", Name: "基础无状态服务", Description: "Deployment + ClusterIP Service；包含标准资源限制，探针默认关闭，可按服务开启 HTTP 或 TCP 探针。", BuiltinKey: "k8s-service", Variables: []Variable{
			{Key: "replicas", Value: "1", Description: "副本数；0 表示停服"},
			{Key: "port", Value: "8080", Description: "应用实际监听端口，请按服务调整"},
			{Key: "cpuRequest", Value: "100m", Description: "容器 CPU 请求"},
			{Key: "memoryRequest", Value: "128Mi", Description: "容器内存请求"},
			{Key: "cpuLimit", Value: "500m", Description: "容器 CPU 上限"},
			{Key: "memoryLimit", Value: "512Mi", Description: "容器内存上限"},
			{Key: "healthCheckType", Value: "none", Description: "健康检查类型：none、http 或 tcp"},
			{Key: "probeReadinessEnabled", Value: "true", Description: "就绪检查：失败停止接收 Service 流量，不重启"},
			{Key: "probeStartupEnabled", Value: "true", Description: "启动检查：为慢启动留出时间；持续失败会重启"},
			{Key: "probeLivenessEnabled", Value: "false", Description: "存活检查：持续失败会重启，确认检查可靠后再启用"},
			{Key: "healthPath", Value: "/health", Description: "HTTP 健康检查路径；启用 http 时使用"},
			{Key: "healthPort", Value: "0", Description: "探针端口；0 表示使用服务端口"},
			{Key: "probePeriodSeconds", Value: "10", Description: "readiness/liveness 检查周期"},
			{Key: "probeTimeoutSeconds", Value: "3", Description: "单次探针超时时间"},
			{Key: "probeFailureThreshold", Value: "3", Description: "readiness/liveness 连续失败次数"},
			{Key: "startupFailureThreshold", Value: "30", Description: "startup 探针连续失败次数"},
		}, Files: []File{
			{Path: "deployment.yaml", Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Service}}
spec:
  replicas: {{.replicas}}
  selector:
    matchLabels:
      app: {{.Service}}
  template:
    metadata:
      labels:
        app: {{.Service}}
    spec:
      containers:
        - name: {{.Service}}
          image: {{.Image}}
          resources:
            requests:
              cpu: {{.cpuRequest}}
              memory: {{.memoryRequest}}
            limits:
              cpu: {{.cpuLimit}}
              memory: {{.memoryLimit}}
          ports:
            - name: http
              containerPort: {{.port}}
`},
			{Path: "service.yaml", Content: `apiVersion: v1
kind: Service
metadata:
  name: {{.Service}}
spec:
  type: ClusterIP
  selector:
    app: {{.Service}}
  ports:
    - name: http
      port: {{.port}}
      targetPort: http
`}}},
		{Kind: "helm", Name: "基础应用 Chart", Description: "完整 Helm v2 Chart；通过 values.yaml 配置应用参数。", BuiltinKey: "helm-service", Variables: []Variable{}, Files: []File{
			{Path: "Chart.yaml", Content: "apiVersion: v2\nname: application\nversion: 0.1.0\ntype: application\n"},
			{Path: "values.yaml", Content: "replicaCount: 1\nimage:\n  repository: example.invalid/team/service\n  tag: latest\nservice:\n  port: 8080\nresources:\n  requests:\n    cpu: 100m\n    memory: 128Mi\n  limits:\n    memory: 512Mi\nenv: []\nenvFrom: []\ncommand: []\nargs: []\n"},
			{Path: "templates/deployment.yaml", Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app: {{ .Release.Name }}
    spec:
      containers:
        - name: application
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          ports:
            - containerPort: {{ .Values.service.port }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          {{- with .Values.env }}
          env:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .Values.envFrom }}
          envFrom:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .Values.command }}
          command:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .Values.args }}
          args:
            {{- toYaml . | nindent 12 }}
          {{- end }}
`}, {Path: "templates/service.yaml", Content: `apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}
spec:
  type: ClusterIP
  selector:
    app: {{ .Release.Name }}
  ports:
    - name: http
      port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.port }}
`}}},
		{Kind: "dockerfile", Name: "Java 17 JAR 运行镜像", Description: "先完成 JAR 编译；将路径调整为实际产物后提交到代码仓库。", BuiltinKey: "dockerfile-java17", Variables: []Variable{}, Files: []File{{Path: "Dockerfile", Content: "FROM eclipse-temurin:17-jre\nWORKDIR /app\nCOPY target/app.jar app.jar\nENTRYPOINT [\"sh\", \"-c\", \"exec java $JAVA_MODULE_OPTS $JAVA_OPTS -jar app.jar\"]\n"}}},
		{Kind: "build", Name: "Maven / JDK 17", Description: "生成 JAR 的编译步骤，可应用到服务的 JAR 镜像构建。", BuiltinKey: "build-maven17", Variables: []Variable{}, Files: []File{{Path: "build.yaml", Content: "build_image: maven:3.9-eclipse-temurin-17\nbuild_command: 'mvn -B -ntp -Dmaven.repo.local=\"${CACHE_DIR:-/workspace/.cache/maven}\" clean package'\ncache_dir: /workspace/.cache/maven\n"}}},
	}
	items = append(items, javaNacosTemplate(items[0]))
	items = append(items, workloadTemplate())
	items = append(items, javaServiceTemplate())
	items = append(items, workflowTemplate("workflow-build", "仅构建", "编译并生成构建产物，不部署到环境。", "k8s_yaml", false, false))
	for _, delivery := range []string{"k8s_yaml", "helm", "host"} {
		items = append(items, workflowTemplate("workflow-"+delivery, delivery+" 构建与部署", "选择服务和环境后创建独立工作流。", delivery, true, false))
	}
	return items
}

func javaNacosTemplate(base Template) Template {
	variables := append([]Variable(nil), base.Variables...)
	for i := range variables {
		switch variables[i].Key {
		case "cpuRequest":
			variables[i].Value = "250m"
		case "memoryRequest":
			variables[i].Value = "512Mi"
		case "cpuLimit":
			variables[i].Value = "1000m"
		case "memoryLimit":
			variables[i].Value = "2Gi"
		}
	}
	variables = append(variables,
		Variable{Key: "springProfile", Value: "test", Description: "Spring Profile；生产环境独立设置为 prod 或应用实际值"},
		Variable{Key: "nacosServerAddr", Value: "nacos.example.invalid:8848", Description: "Nacos 实际地址和端口；使用前替换"},
		Variable{Key: "nacosNamespace", Value: "replace-with-namespace-id", Description: "Nacos 命名空间 ID，不是 Kubernetes Namespace；按目标环境填写实际 ID"},
		Variable{Key: "nacosGroup", Value: "DEFAULT_GROUP", Description: "Nacos 配置与注册发现分组"},
		Variable{Key: "javaMaxRamPercentage", Value: "72.0", Description: "堆内存比例；减小容器内存时同时检查堆、元空间和本地内存"},
		Variable{Key: "javaMaxMetaspaceSize", Value: "256m", Description: "元空间上限；模板容器内存上限为 2Gi"},
	)
	return Template{
		Kind: "k8s_yaml", Name: "Java 17 / Spring Boot / Nacos 服务", BuiltinKey: "k8s-java17-nacos",
		Description: "Deployment + Service；Java 内存参数、Spring Profile 和 Nacos 地址按环境修改。需要鉴权时通过配套配置添加 Secret 环境变量，探针默认关闭。镜像启动入口须使用 JAVA_OPTS。",
		Variables:   variables,
		Files: []File{{Path: "deployment.yaml", Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Service}}
spec:
  replicas: {{.replicas}}
  selector:
    matchLabels:
      app: {{.Service}}
  template:
    metadata:
      labels:
        app: {{.Service}}
    spec:
      containers:
        - name: {{.Service}}
          image: {{.Image}}
          imagePullPolicy: IfNotPresent
          resources:
            requests:
              cpu: {{.cpuRequest}}
              memory: {{.memoryRequest}}
            limits:
              cpu: {{.cpuLimit}}
              memory: {{.memoryLimit}}
          ports:
            - name: http
              containerPort: {{.port}}
              protocol: TCP
          env:
            - name: JAVA_OPTS
              value: >-
                -XX:+UseContainerSupport
                -XX:MaxRAMPercentage={{.javaMaxRamPercentage}}
                -XX:MaxMetaspaceSize={{.javaMaxMetaspaceSize}}
                -XX:+UseG1GC
                -Dfile.encoding=UTF-8
            - name: SPRING_APPLICATION_NAME
              value: "{{.Service}}"
            - name: SPRING_PROFILES_ACTIVE
              value: "{{.springProfile}}"
            - name: SPRING_CONFIG_IMPORT
              value: "optional:nacos:{{.Service}}"
            - name: SPRING_CLOUD_NACOS_DISCOVERY_SERVERADDR
              value: "{{.nacosServerAddr}}"
            - name: SPRING_CLOUD_NACOS_DISCOVERY_NAMESPACE
              value: "{{.nacosNamespace}}"
            - name: SPRING_CLOUD_NACOS_DISCOVERY_GROUP
              value: "{{.nacosGroup}}"
            - name: SPRING_CLOUD_NACOS_CONFIG_SERVERADDR
              value: "{{.nacosServerAddr}}"
            - name: SPRING_CLOUD_NACOS_CONFIG_NAMESPACE
              value: "{{.nacosNamespace}}"
            - name: SPRING_CLOUD_NACOS_CONFIG_GROUP
              value: "{{.nacosGroup}}"
            - name: SPRING_CLOUD_NACOS_CONFIG_FILEEXTENSION
              value: "yaml"
            - name: SPRING_CLOUD_NACOS_CONFIG_REFRESHENABLED
              value: "true"
`}, base.Files[1]},
	}
}

func workflowTemplate(key, name, description, delivery string, deploy, release bool) Template {
	nodes := []workflow.Node{}
	deps := []string{}
	if !release {
		nodes = append(nodes, workflow.Node{Key: "build", Name: "构建", Type: workflow.NodeBuild, Config: json.RawMessage(`{"service_id":0}`), TimeoutSeconds: 3600})
		deps = []string{"build"}
	}
	if deploy {
		deployType := workflow.NodeYAMLDeploy
		if delivery == "helm" {
			deployType = workflow.NodeHelmDeploy
		}
		if delivery == "host" {
			deployType = workflow.NodeHostDeploy
		}
		nodes = append(nodes, workflow.Node{Key: "deploy", Name: "部署", Type: deployType, DependsOn: deps, Config: json.RawMessage(`{"service_id":0,"service_revision_id":0}`), TimeoutSeconds: 1800})
	}
	content, _ := json.MarshalIndent(workflow.Definition{DeliveryType: delivery, Nodes: nodes}, "", "  ")
	return Template{Kind: "workflow", BuiltinKey: key, Name: name, Description: description, Files: []File{{Path: "workflow.json", Content: string(content)}}, Variables: []Variable{}}
}

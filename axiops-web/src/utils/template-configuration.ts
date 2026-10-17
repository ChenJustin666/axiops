import { Document, Scalar, parseAllDocuments } from 'yaml'
import { emptyServiceAssets, type ServiceAssets } from '@/api/axiops/service-assets'

export type TemplateParameter = {
  key: string
  value: string
  description?: string
  label?: string
  group?: string
}
export function templateYAMLBody(definition: string) {
  return definition.replace(/^[\t ]*# AxiOps-Variables:.*\r?\n?/gm, '')
}
export function updateTemplateYAMLBody(definition: string, body: string) {
  const metadata = definition.match(/^[\t ]*# AxiOps-Variables:.*$/m)?.[0]
  return (metadata ? metadata + '\n' : '') + templateYAMLBody(body)
}
export function templateHasProbes(definition: string) {
  try {
    return decodeTemplate(definition).docs.some((resource) => {
      const pod = resource.spec?.template || resource.spec?.jobTemplate?.spec?.template
      return pod?.spec?.containers?.some(
        (container: Record<string, unknown>) =>
          container.readinessProbe || container.livenessProbe || container.startupProbe
      )
    })
  } catch {
    return false
  }
}
export function templateParameterSection(parameter: TemplateParameter) {
  // Legacy templates put probe fields in "advanced" together with JVM settings.
  // Recognize health fields first so they keep their own section.
  if (
    /^(health|probe|startup|metrics|prometheus|monitor)/i.test(parameter.key) ||
    ['health', 'probe', 'monitor'].includes(parameter.group || '')
  )
    return 'health'
  if (/^java/i.test(parameter.key) || ['env', 'jvm'].includes(parameter.group || '')) return 'env'
  return parameter.group === 'advanced' ? 'advanced' : 'primary'
}
export const parameterLabels: Record<string, string> = {
  replicas: '副本数',
  port: '服务端口',
  cpuRequest: 'CPU 请求',
  cpuLimit: 'CPU 上限',
  memoryRequest: '内存请求',
  memoryLimit: '内存上限',
  springProfile: 'Spring 环境',
  nacosServerAddr: 'Nacos 地址',
  nacosNamespace: 'Nacos Namespace ID',
  nacosGroup: 'Nacos 分组',
  javaMaxRamPercentage: 'JVM 最大内存比例',
  javaMaxMetaspaceSize: 'Metaspace 上限',
  healthCheckType: '健康检查类型',
  healthPath: 'HTTP 健康检查路径',
  healthPort: '健康检查端口',
  probeReadinessEnabled: '就绪检查 · 失败停止接流量',
  probeStartupEnabled: '启动检查 · 持续失败重启',
  probeLivenessEnabled: '存活检查 · 持续失败重启',
  probePeriodSeconds: '探针检查周期（秒）',
  probeTimeoutSeconds: '探针超时（秒）',
  probeFailureThreshold: '连续失败次数',
  startupFailureThreshold: '启动探针失败次数'
}
export type TemplateOptions = ReturnType<typeof emptyTemplateOptions>
export const emptyTemplateOptions = () => ({
  ingress: false,
  pvc: false,
  secret: false,
  configmap: false,
  hpa: false,
  probe: false,
  probeReadinessEnabled: true,
  probeStartupEnabled: true,
  probeLivenessEnabled: false,
  monitor: false,
  host: '',
  path: '/',
  ingressClass: '',
  pathType: 'Prefix' as 'Prefix' | 'Exact' | 'ImplementationSpecific',
  tlsEnabled: false,
  tlsHosts: [] as string[],
  tlsSecret: '',
  ingressAnnotations: {} as Record<string, string>,
  ingressLabels: {} as Record<string, string>,
  existingClaim: '',
  capacity: '1Gi',
  storageClass: '',
  accessMode: 'ReadWriteOnce',
  readOnly: false,
  mountPath: '/data',
  secretName: '',
  secretKey: 'password',
  secretEnv: 'APP_PASSWORD',
  secretUsage: 'env',
  secretMount: '',
  configName: '',
  filename: 'application.yaml',
  content: '',
  configMount: '/app/config/application.yaml',
  minReplicas: 1,
  maxReplicas: 3,
  cpuPercent: 80,
  probeType: 'tcp',
  probePath: '/health',
  probePort: '',
  metricsPath: '/actuator/prometheus',
  metricsPort: ''
})

// Kubernetes accepts DNS names for object references and IngressClass/TLS
// names.  Keep these checks in the wizard so an invalid value is rejected
// before a service revision is saved. The API performs the same validation
// again; this is only the fast, user-facing guard.
function isDNSName(value: string, wildcard = false) {
  const text = value
  const name = wildcard && text.startsWith('*.') ? text.slice(2) : text
  if (!name || text.length > 253 || (wildcard && /^\d+\.\d+\.\d+\.\d+$/.test(name))) return false
  if (text.includes('*') && !(wildcard && text.startsWith('*.'))) return false
  return name
    .split('.')
    .every(
      (label) =>
        label.length > 0 && label.length <= 63 && /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/.test(label)
    )
}

function isKubernetesKey(value: string) {
  const text = value
  return (
    text.length > 0 &&
    text.length <= 253 &&
    /^[A-Za-z0-9._-]+$/.test(text) &&
    !['.', '..'].includes(text)
  )
}

function isAbsoluteMountPath(value: string) {
  const text = value
  return (
    text.startsWith('/') &&
    text !== '/' &&
    !text.endsWith('/') &&
    !text.includes('//') &&
    !text.split('/').some((part) => part === '.' || part === '..') &&
    !/[\0\r\n\\]/.test(text)
  )
}

function isResourceQuantity(value: string) {
  const match = value.match(/^\+?(\d+(?:\.\d*)?|\.\d+)([eE][+-]?\d+|[EPTGMK]i|[EPTGMk]|m|u|n)?$/)
  if (!match) return false
  const suffix = match[2] || ''
  const powers: Record<string, number> = {
    k: 1e3,
    M: 1e6,
    G: 1e9,
    T: 1e12,
    P: 1e15,
    E: 1e18,
    m: 1e-3,
    u: 1e-6,
    n: 1e-9
  }
  const factor = suffix.endsWith('i')
    ? 1024 ** ('KMGTPE'.indexOf(suffix[0]) + 1)
    : /^[eE]/.test(suffix) && suffix.length > 1
      ? 10 ** Number(suffix.slice(1))
      : powers[suffix] || 1
  const bytes = Number(match[1]) * factor
  return Number.isFinite(bytes) && bytes >= 1024 ** 2
}

export function parseIngressMetadata(text: string): Record<string, string> {
  if (!text.trim()) return {}
  const docs = parseAllDocuments(text)
  if (docs.length !== 1 || docs[0].errors.length)
    throw new Error('请填写一份有效的 YAML 键值表，不允许重复键')
  const value = docs[0].toJSON()
  if (!value || typeof value !== 'object' || Array.isArray(value))
    throw new Error('注解和标签须为 YAML 键值表')
  const result: Record<string, string> = {}
  for (const [key, item] of Object.entries(value)) {
    if (!['string', 'number', 'boolean'].includes(typeof item))
      throw new Error('每项值须为字符串；JSON 数组请用单引号包裹')
    Object.defineProperty(result, key, { value: String(item), enumerable: true, writable: true })
  }
  return result
}

export function validateIngressRoute(route: ServiceAssets['routes'][number]) {
  const pathType = route.path_type || 'Prefix'
  if (!['Prefix', 'Exact', 'ImplementationSpecific'].includes(pathType))
    throw new Error('路径类型须为 Prefix、Exact 或 ImplementationSpecific')
  const path = route.path
  if (
    !path.startsWith('/') ||
    /[\s\0]/.test(path) ||
    (pathType !== 'ImplementationSpecific' &&
      (/[?#\\]/.test(path) ||
        path.includes('//') ||
        path.split('/').some((p) => p === '.' || p === '..')))
  )
    throw new Error('路由路径必须是有效的绝对路径；正则路径请选择 ImplementationSpecific')
  if (!isDNSName(route.host, true))
    throw new Error('域名必须是有效的 DNS 名称，可使用 *.example.com')
  if (route.class && !isDNSName(route.class)) throw new Error('Ingress Class 名称无效')
  if (route.tls_secret && !isDNSName(route.tls_secret)) throw new Error('TLS Secret 名称无效')
  const tlsHosts = route.tls_hosts || []
  if (route.tls_enabled && !tlsHosts.length) throw new Error('开启 TLS 后请填写证书域名')
  if (tlsHosts.length > 20 || (!route.tls_enabled && !route.tls_secret && tlsHosts.length))
    throw new Error('TLS 关闭时不能配置证书域名，开启时最多20个域名')
  if (tlsHosts.some((host) => !isDNSName(host, true)) || new Set(tlsHosts).size !== tlsHosts.length)
    throw new Error('TLS 域名无效或重复')
  const qualifiedKey = (key: string) => {
    const parts = key.split('/')
    const name = parts.pop() || ''
    return (
      parts.length <= 1 &&
      (!parts.length || isDNSName(parts[0])) &&
      name.length <= 63 &&
      /^[A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?$/.test(name) &&
      !key.startsWith('axiops.io/')
    )
  }
  let annotationBytes = 0
  for (const [field, entries] of [
    ['注解', route.annotations || {}],
    ['标签', route.labels || {}]
  ] as const) {
    if (Object.keys(entries).length > 50) throw new Error(`Ingress ${field}最多50项`)
    for (const [key, value] of Object.entries(entries)) {
      if (!qualifiedKey(key) || typeof value !== 'string' || value.includes('\0'))
        throw new Error(`Ingress ${field}名称或值无效；axiops.io/ 为系统保留前缀`)
      if (
        field === '标签' &&
        (value.length > 63 ||
          (value && !/^[A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?$/.test(value)))
      )
        throw new Error('Ingress 标签值无效')
      if (field === '注解') annotationBytes += new TextEncoder().encode(key + value).length
    }
  }
  if (annotationBytes > 256 * 1024) throw new Error('Ingress 注解不能超过256KiB')
  const annotations = route.annotations || {}
  const regex = annotations['alb.ingress.kubernetes.io/use-regex']
  if (regex !== undefined && !['true', 'false'].includes(regex))
    throw new Error('ALB use-regex 须为 true 或 false')
  const order = annotations['alb.ingress.kubernetes.io/order']
  if (order !== undefined && !/^[+-]?\d+$/.test(order)) throw new Error('ALB order 须为整数')
  const listenerText = annotations['alb.ingress.kubernetes.io/listen-ports']
  if (listenerText !== undefined) {
    let listeners: unknown
    try {
      listeners = JSON.parse(listenerText)
    } catch {
      throw new Error('ALB listen-ports 须为 JSON 数组')
    }
    if (!Array.isArray(listeners) || !listeners.length)
      throw new Error('ALB listen-ports 须为非空 JSON 数组')
    const ports = new Set<number>()
    for (const listener of listeners) {
      if (
        !listener ||
        typeof listener !== 'object' ||
        Array.isArray(listener) ||
        Object.keys(listener).length !== 1
      )
        throw new Error('ALB 每个监听项只能指定一种协议和端口')
      const [protocol, port] = Object.entries(listener)[0]
      if (
        !['HTTP', 'HTTPS'].includes(protocol) ||
        !Number.isInteger(port) ||
        Number(port) < 1 ||
        Number(port) > 65535 ||
        ports.has(Number(port))
      )
        throw new Error('ALB 监听须使用 HTTP/HTTPS 和1到65535的不重复整数端口')
      ports.add(Number(port))
    }
  }
}

// Use opaque scalar tokens while parsing a template. This keeps placeholders
// intact and edits only selected Kubernetes paths; unrelated resources survive.
function decodeTemplate(definition: string) {
  const tokens: string[] = []
  const body = definition
    .replace(/^# AxiOps-Variables:.*\r?\n?/m, '')
    .replace(/\{\{\s*\.?[A-Za-z][A-Za-z0-9_]*\s*\}\}|\$[A-Za-z][A-Za-z0-9_-]*\$/g, (match) => {
      tokens.push(match)
      return `AXIOPS_TEMPLATE_VALUE_${tokens.length - 1}_END`
    })
  const docs = parseAllDocuments(body)
  for (const doc of docs) if (doc.errors.length) throw new Error(doc.errors[0].message)
  const resources = docs.filter((doc) => doc.toJSON() != null)
  return {
    docs: resources.map((doc) => doc.toJSON()),
    filenames: resources.map((doc) => doc.toString().match(/^# AxiOps-File: ([^\r\n]+)$/m)?.[1]),
    tokens
  }
}

// Keep long JVM option values readable and valid after optional resources are
// added. YAML's default scalar width wrapping is legal, but the wrapped lines
// look like list items in the editor and are easy to misread. Mark JAVA_OPTS as
// a folded scalar so every continuation line remains part of the value.
function stringifyConfiguredResource(resource: Record<string, any>): string {
  const copy = JSON.parse(JSON.stringify(resource)) as Record<string, any>
  const visit = (value: any) => {
    if (!value || typeof value !== 'object') return
    if (Array.isArray(value)) {
      value.forEach(visit)
      return
    }
    if (value.name === 'JAVA_OPTS' && typeof value.value === 'string') {
      const scalar = new Scalar(value.value)
      scalar.type = Scalar.BLOCK_FOLDED
      value.value = scalar
    }
    Object.values(value).forEach(visit)
  }
  visit(copy)
  const document = new Document(copy)
  return String(document)
}

export function supportsTemplateOptions(definition: string) {
  try {
    const { docs } = decodeTemplate(definition)
    return (
      docs.filter((d) => d.kind === 'Deployment').length === 1 &&
      docs.filter((d) => d.kind === 'Service').length === 1 &&
      docs.find((d) => d.kind === 'Deployment')?.spec?.template?.spec?.containers?.length === 1
    )
  } catch {
    return false
  }
}
export function configureTemplate(
  definition: string,
  options: TemplateOptions,
  service: string,
  values: Record<string, string>
): { definition: string; assets: ServiceAssets } {
  const assets = emptyServiceAssets()
  if (
    !['ingress', 'pvc', 'secret', 'configmap', 'hpa', 'probe', 'monitor'].some(
      (key) => options[key as keyof TemplateOptions] === true
    )
  )
    return { definition, assets }
  if (!supportsTemplateOptions(definition))
    throw new Error('该模板有多个工作负载或容器，请直接编辑模板 YAML 配置扩展资源')
  const { docs, tokens, filenames } = decodeTemplate(definition)
  const deployment = docs.find((d) => d.kind === 'Deployment'),
    svc = docs.find((d) => d.kind === 'Service')
  const pod = deployment.spec.template,
    container = pod.spec.containers[0]
  const kubeName = service.replace(/_/g, '-')
  const resolve = (value: unknown) =>
    String(value ?? '').replace(/AXIOPS_TEMPLATE_VALUE_(\d+)_END/g, (_, index) => {
      const placeholder = tokens[Number(index)]
      const key = placeholder.replace(/[{}$\s.]/g, '')
      if (key === 'Service') return kubeName
      if (!(key in values)) throw new Error(`请填写模板参数 ${key}`)
      return values[key]
    })
  const servicePort = Number(resolve(svc.spec?.ports?.[0]?.port))
  const requireText = (value: string, label: string) => {
    if (!value.trim()) throw new Error(`请填写${label}`)
    return value.trim()
  }
  if (options.ingress) {
    if (docs.some((d) => d.kind === 'Ingress'))
      throw new Error('模板已有 Ingress，请修改已有 YAML，避免重复路由')
    if (!Number.isInteger(servicePort) || servicePort < 1 || servicePort > 65535)
      throw new Error('路由需要有效的服务端口')
    const path = requireText(options.path, '路由路径')
    const host = requireText(options.host, '域名')
    const route: ServiceAssets['routes'][number] = {
      name: `${kubeName}-route`,
      host,
      path,
      service: resolve(svc.metadata.name),
      port: servicePort,
      class: options.ingressClass,
      path_type: options.pathType || 'Prefix',
      tls_enabled: options.tlsEnabled,
      tls_hosts: options.tlsHosts,
      tls_secret: options.tlsSecret,
      annotations: options.ingressAnnotations,
      labels: options.ingressLabels
    }
    validateIngressRoute(route)
    assets.routes.push(route)
  }
  if (options.pvc) {
    if (
      !options.existingClaim &&
      !['ReadWriteOnce', 'ReadWriteMany', 'ReadOnlyMany', 'ReadWriteOncePod'].includes(
        options.accessMode
      )
    )
      throw new Error('PVC 访问模式无效')
    if (options.existingClaim.trim() && !isDNSName(options.existingClaim))
      throw new Error('已有 PVC 名称无效')
    if (!options.existingClaim.trim()) {
      const capacity = requireText(options.capacity, '存储容量')
      if (!isResourceQuantity(capacity))
        throw new Error('存储容量至少为 1Mi，例如 1Gi；2 表示 2 字节，不是 2Gi')
      if (options.storageClass && !isDNSName(options.storageClass))
        throw new Error('Storage Class 名称无效')
    }
    if (!isAbsoluteMountPath(options.mountPath)) throw new Error('存储挂载目录必须是有效的绝对路径')
    assets.volumes.push({
      name: `${kubeName}-data`,
      existing_claim: options.existingClaim,
      capacity: options.existingClaim.trim() ? '' : options.capacity,
      storage_class: options.existingClaim.trim() ? '' : options.storageClass,
      access_mode: options.accessMode,
      mount_path: requireText(options.mountPath, '存储挂载目录'),
      read_only: options.readOnly || options.accessMode === 'ReadOnlyMany'
    })
  }
  if (options.secret) if (!isDNSName(options.secretName)) throw new Error('Secret 名称无效')
  if (options.secret)
    assets.secrets.push({
      name: `${kubeName}-auth`,
      credential_id: 0,
      credential_key: '',
      existing_secret: requireText(options.secretName, '目标环境已有 Secret 名称'),
      key: requireText(options.secretKey, 'Secret key'),
      env_name:
        options.secretUsage === 'file' ? '' : requireText(options.secretEnv, '注入环境变量名'),
      mount_path:
        options.secretUsage === 'file'
          ? requireText(options.secretMount, 'Secret 文件挂载路径')
          : ''
    })
  if (options.secret) {
    if (!['env', 'file'].includes(options.secretUsage)) throw new Error('Secret 使用方式无效')
    if (!isKubernetesKey(options.secretKey)) throw new Error('Secret key 名称无效')
    if (options.secretUsage === 'file' && !isAbsoluteMountPath(options.secretMount))
      throw new Error('Secret 文件挂载路径无效')
    if (
      options.secretUsage !== 'file' &&
      !/^[A-Za-z_][A-Za-z0-9_]*$/.test(options.secretEnv.trim())
    )
      throw new Error('Secret 环境变量名无效')
  }
  if (options.configmap)
    if (options.configName && !isDNSName(options.configName)) throw new Error('ConfigMap 名称无效')
  if (options.configmap)
    assets.files.push({
      name: `${kubeName}-config`,
      filename: requireText(options.filename, '配置文件名'),
      content: options.configName.trim() ? '' : options.content,
      mount_path: requireText(options.configMount, '配置文件挂载路径'),
      existing_config_map: options.configName
    })
  if (options.configmap) {
    const bytes = Array.from(options.content).reduce((size, char) => {
      const code = char.codePointAt(0)!
      return size + (code <= 0x7f ? 1 : code <= 0x7ff ? 2 : code <= 0xffff ? 3 : 4)
    }, 0)
    if (bytes > 256 * 1024 && !options.configName)
      throw new Error('ConfigMap 文件内容不能超过 256KiB')
    if (!isKubernetesKey(options.filename)) throw new Error('ConfigMap 文件名无效')
    if (!isAbsoluteMountPath(options.configMount)) throw new Error('ConfigMap 挂载路径无效')
  }
  if (options.probe) {
    if (!['tcp', 'http'].includes(options.probeType)) throw new Error('探针类型必须为 TCP 或 HTTP')
    if (
      values.healthCheckType &&
      !['none', ''].includes(values.healthCheckType.trim().toLowerCase())
    )
      throw new Error('模板健康检查已启用，请修改模板探针参数；或先关闭模板健康检查，再添加探针')
    if (container.readinessProbe || container.livenessProbe || container.startupProbe)
      throw new Error('模板已有探针，请直接修改已有探针')
    const raw = options.probePort || resolve(container.ports?.[0]?.containerPort)
    const port = /^\d+$/.test(raw) ? Number(raw) : requireText(raw, '探针端口')
    if (typeof port === 'number' && (port < 1 || port > 65535))
      throw new Error('探针端口需为 1–65535')
    if (
      typeof port === 'string' &&
      (port.length > 15 || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(port) || !/[a-z]/.test(port))
    )
      throw new Error('探针命名端口无效')
    if (
      options.probeType === 'http' &&
      (!options.probePath.startsWith('/') ||
        options.probePath.includes('//') ||
        /[?#\r\n]/.test(options.probePath))
    )
      throw new Error('HTTP 探针路径必须是有效的绝对路径')
    const handler =
      options.probeType === 'http'
        ? { httpGet: { path: requireText(options.probePath, 'HTTP 探针路径'), port } }
        : { tcpSocket: { port } }
    if (
      !options.probeReadinessEnabled &&
      !options.probeStartupEnabled &&
      !options.probeLivenessEnabled
    )
      throw new Error('请至少选择一种检查，或取消添加探针')
    if (options.probeReadinessEnabled)
      container.readinessProbe = {
        ...handler,
        periodSeconds: 10,
        timeoutSeconds: 3,
        failureThreshold: 3
      }
    if (options.probeLivenessEnabled)
      container.livenessProbe = {
        ...handler,
        periodSeconds: 10,
        timeoutSeconds: 3,
        failureThreshold: 3
      }
    if (options.probeStartupEnabled)
      container.startupProbe = {
        ...handler,
        periodSeconds: 10,
        timeoutSeconds: 3,
        failureThreshold: 30
      }
  }
  if (options.monitor) {
    const port = requireText(
      options.metricsPort || resolve(container.ports?.[0]?.containerPort),
      '监控端口'
    )
    if (!/^\d+$/.test(port) || Number(port) < 1 || Number(port) > 65535)
      throw new Error('监控端口需为 1–65535')
    pod.metadata ||= {}
    pod.metadata.annotations ||= {}
    const metricsPath = requireText(options.metricsPath, '监控路径')
    if (!metricsPath.startsWith('/') || /[?#\r\n]/.test(metricsPath))
      throw new Error('监控路径必须以 / 开头')
    Object.assign(pod.metadata.annotations, {
      'prometheus.io/scrape': 'true',
      'prometheus.io/path': metricsPath,
      'prometheus.io/port': port
    })
  }
  if (options.hpa) {
    if (docs.some((d) => d.kind === 'HorizontalPodAutoscaler'))
      throw new Error('模板已有 HPA，请修改已有 YAML')
    if (
      ![options.minReplicas, options.maxReplicas, options.cpuPercent].every(Number.isInteger) ||
      options.minReplicas < 1 ||
      options.maxReplicas < options.minReplicas ||
      options.cpuPercent < 1
    )
      throw new Error('HPA 副本范围和 CPU 目标值无效')
    if (!container.resources?.requests?.cpu)
      throw new Error('启用 CPU HPA 需要模板配置 CPU requests')
    docs.push({
      apiVersion: 'autoscaling/v2',
      kind: 'HorizontalPodAutoscaler',
      metadata: { name: deployment.metadata.name },
      spec: {
        scaleTargetRef: {
          apiVersion: 'apps/v1',
          kind: 'Deployment',
          name: deployment.metadata.name
        },
        minReplicas: options.minReplicas,
        maxReplicas: options.maxReplicas,
        metrics: [
          {
            type: 'Resource',
            resource: {
              name: 'cpu',
              target: { type: 'Utilization', averageUtilization: options.cpuPercent }
            }
          }
        ]
      }
    })
  }
  const metadata = definition.match(/^# AxiOps-Variables:.*$/m)?.[0]
  const body = docs
    .map(
      (doc, index) =>
        `# AxiOps-File: ${filenames[index] || (doc.kind === 'Service' ? 'service' : doc.kind === 'HorizontalPodAutoscaler' ? 'hpa' : doc.kind.toLowerCase()) + '.yaml'}\n${stringifyConfiguredResource(doc)}`
    )
    .join('---\n')
    .replace(/AXIOPS_TEMPLATE_VALUE_(\d+)_END/g, (_, index) => tokens[Number(index)])
  return { definition: (metadata ? metadata + '\n' : '') + body, assets }
}

export type TemplateCapability =
  | 'ingress'
  | 'pvc'
  | 'secret'
  | 'configmap'
  | 'hpa'
  | 'probe'
  | 'monitor'
export type TemplateSelection = { id: number; key: TemplateCapability; options: TemplateOptions }
export function configureTemplateSelections(
  definition: string,
  selections: TemplateSelection[],
  service: string,
  values: Record<string, string>
) {
  let result = definition
  const assets = emptyServiceAssets()
  const ids = new Set<number>(),
    single = new Set<string>()
  for (const selection of selections) {
    if (!Number.isSafeInteger(selection.id) || selection.id < 1 || ids.has(selection.id))
      throw Error('配置标识无效或重复')
    ids.add(selection.id)
    if (['hpa', 'probe', 'monitor'].includes(selection.key)) {
      if (single.has(selection.key)) throw Error('同一工作负载只能配置一项 HPA、探针或监控')
      single.add(selection.key)
    }
    const options = {
      ...selection.options,
      ingress: false,
      pvc: false,
      secret: false,
      configmap: false,
      hpa: false,
      probe: false,
      monitor: false,
      [selection.key]: true
    }
    const configured = configureTemplate(result, options, service, values)
    result = configured.definition
    for (const key of ['files', 'secrets', 'routes', 'volumes'] as const) {
      const prefix = { files: 'config', secrets: 'secret', routes: 'route', volumes: 'data' }[key]
      for (const item of configured.assets[key]) {
        item.name = `${prefix}-${selection.id}`
        ;(assets[key] as Array<typeof item>).push(item)
      }
    }
  }
  if (
    assets.files.length > 30 ||
    assets.secrets.length > 30 ||
    assets.routes.length > 20 ||
    assets.volumes.length > 20
  )
    throw Error('配置项数量超过限制')
  const paths: string[] = []
  for (const item of [...assets.files, ...assets.secrets, ...assets.volumes]) {
    const path = item.mount_path
    if (!path) continue
    if (
      !path.startsWith('/') ||
      path.endsWith('/') ||
      /[\\\r\n]/.test(path) ||
      path.includes('\u0000') ||
      path.includes('//') ||
      path.split('/').some((part) => part === '.' || part === '..')
    )
      throw Error('请填写有效的绝对挂载路径')
    if (
      paths.some((old) => old === path || old.startsWith(path + '/') || path.startsWith(old + '/'))
    )
      throw Error('配置文件、Secret 或 PVC 挂载路径冲突')
    paths.push(path)
  }
  const envs = new Set<string>()
  for (const item of assets.secrets) {
    if (
      !/^[A-Za-z0-9._-]+$/.test(item.key) ||
      ['.', '..'].includes(item.key) ||
      item.key.length > 253
    )
      throw Error('Secret键名无效')
    if (!item.env_name) continue
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(item.env_name) || envs.has(item.env_name))
      throw Error('Secret环境变量名无效或重复')
    envs.add(item.env_name)
  }
  const routes = new Set<string>()
  for (const item of assets.routes) {
    const key = item.host + item.path
    if (routes.has(key)) throw Error('同一域名和路径不能重复配置路由')
    routes.add(key)
  }
  return { definition: result, assets }
}

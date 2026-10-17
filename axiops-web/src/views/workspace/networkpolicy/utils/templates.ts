/**
 * NetworkPolicy 模板预设
 * 提供常见网络策略的 YAML 模板
 */

export interface NetworkPolicyTemplate {
  id: string
  name: string
  description: string
  category: 'security' | 'access' | 'isolation'
  yaml: string
}

/**
 * 获取所有 NetworkPolicy 模板
 */
export function getNetworkPolicyTemplates(): NetworkPolicyTemplate[] {
  return [
    {
      id: 'deny-all-ingress',
      name: '拒绝所有入站流量',
      description: '拒绝所有进入的流量，用于默认拒绝策略',
      category: 'security',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-ingress
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Ingress`
    },
    {
      id: 'deny-all-egress',
      name: '拒绝所有出站流量',
      description: '拒绝所有出去的流量，用于严格网络隔离',
      category: 'security',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-egress
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Egress`
    },
    {
      id: 'deny-all',
      name: '拒绝所有流量',
      description: '拒绝所有入站和出站流量，完全隔离',
      category: 'isolation',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress`
    },
    {
      id: 'allow-same-namespace',
      name: '允许同命名空间通信',
      description: '只允许同一命名空间内的 Pod 互相通信',
      category: 'access',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-same-namespace
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector: {}
  egress:
  - to:
    - podSelector: {}`
    },
    {
      id: 'allow-http-https',
      name: '允许 HTTP/HTTPS 端口',
      description: '只允许访问 HTTP (80) 和 HTTPS (443) 端口',
      category: 'access',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-http-https
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - ports:
    - protocol: TCP
      port: 80
    - protocol: TCP
      port: 443`
    },
    {
      id: 'allow-from-specific-namespace',
      name: '允许特定命名空间访问',
      description: '只允许指定命名空间的 Pod 访问',
      category: 'access',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-specific-namespace
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: specific-namespace`
    },
    {
      id: 'allow-specific-port',
      name: '允许特定端口',
      description: '只允许访问指定的应用端口',
      category: 'access',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-specific-port
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - ports:
    - protocol: TCP
      port: 8080`
    },
    {
      id: 'web-app-policy',
      name: 'Web 应用策略',
      description: '典型的 Web 应用网络策略：允许外部访问 80/443，允许访问同命名空间的数据库',
      category: 'access',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: web-app-policy
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: web
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - ports:
    - protocol: TCP
      port: 80
    - protocol: TCP
      port: 443
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: database
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 53`
    },
    {
      id: 'database-policy',
      name: '数据库访问策略',
      description: '只允许特定标签的 Pod 访问数据库',
      category: 'security',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: database-policy
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: database
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          access: allowed
    ports:
    - protocol: TCP
      port: 5432`
    },
    {
      id: 'allow-dns',
      name: '允许 DNS 查询',
      description: '允许 Pod 进行 DNS 查询',
      category: 'access',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-dns
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: kube-system
    ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53`
    },
    {
      id: 'allow-from-ipblock',
      name: '允许特定 IP 段访问',
      description: '允许来自指定 IP 段的入站流量',
      category: 'access',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-ipblock
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: web
  policyTypes:
  - Ingress
  ingress:
  - from:
    - ipBlock:
        cidr: 10.0.0.0/8
        except:
        - 10.0.1.0/24
    ports:
    - protocol: TCP
      port: 80`
    },
    {
      id: 'allow-to-external',
      name: '允许访问外部 IP',
      description: '允许 Pod 访问集群外部指定的 IP 段',
      category: 'access',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-to-external
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: backend
  policyTypes:
  - Egress
  egress:
  - to:
    - ipBlock:
        cidr: 192.168.0.0/16
    ports:
    - protocol: TCP
      port: 443`
    },
    {
      id: 'pod-namespace-combined',
      name: '跨命名空间 Pod 选择器',
      description: '允许特定命名空间中符合标签的 Pod 访问',
      category: 'access',
      yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: pod-namespace-combined
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: database
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: frontend
      podSelector:
        matchLabels:
          app: web
    ports:
    - protocol: TCP
      port: 5432`
    }
  ]
}

/**
 * 根据 ID 获取模板
 */
export function getTemplateById(id: string): NetworkPolicyTemplate | undefined {
  return getNetworkPolicyTemplates().find((t) => t.id === id)
}

/**
 * 根据分类获取模板
 */
export function getTemplatesByCategory(
  category: NetworkPolicyTemplate['category']
): NetworkPolicyTemplate[] {
  return getNetworkPolicyTemplates().filter((t) => t.category === category)
}

/**
 * 替换模板中的命名空间
 */
export function replaceTemplateNamespace(yaml: string, namespace: string): string {
  return yaml.replace(/namespace: default/g, `namespace: ${namespace}`)
}

/**
 * 替换模板中的名称
 */
export function replaceTemplateName(yaml: string, name: string): string {
  return yaml.replace(/name: .+/g, (match) => {
    if (match.includes('metadata:')) {
      return `name: ${name}`
    }
    return match
  })
}

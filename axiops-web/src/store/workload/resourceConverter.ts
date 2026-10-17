// stores/workload/resourceConverter.ts
import type {
  V1Pod,
  V1Deployment,
  V1StatefulSet,
  V1Job,
  V1CronJob,
  V1DaemonSet,
  V1ObjectMeta,
  V1PodTemplateSpec,
  V1LabelSelector,
  V1PodSpec
} from '@kubernetes/client-node'
import { useMetadataStore } from './metadata'
import { useContainersStore } from './containers'
import { useVolumesStore } from './volumes'
import { useSchedulingStore } from './scheduling'
import { useAdvancedStore } from './advanced'
import { useDeploymentStore } from './deployment'
import { useStatefulSetStore } from './statefulset'
import { useDaemonSetStore } from './daemonset'
import { useJobStore } from './job'
import { useCronJobStore } from './cronjob'

/**
 * 生成基础 metadata
 * 注意：使用 resourceName 作为 K8s 资源的 metadata.name
 */
function generateMetadata(
  namespace: string,
  resourceName: string,
  nameEn: string,
  version: string,
  nameCn: string,
  desc: string,
  labels: Record<string, string>,
  annotations: Record<string, string>
): V1ObjectMeta {
  const finalLabels = {
    app: nameEn,
    version: version,
    ...labels
  }

  const finalAnnotations = {
    'ikubeops.com/service-name': nameCn,
    'created-by': 'kube-nova',
    ...(desc ? { description: desc } : {}),
    ...annotations
  }

  return {
    name: resourceName, // 使用 resourceName 作为 K8s 资源名称
    namespace: namespace,
    labels: finalLabels,
    annotations: finalAnnotations
  }
}

/**
 * 生成 Pod 模板的 metadata
 */
function generatePodTemplateMetadata(
  nameEn: string,
  version: string,
  nameCn: string,
  desc: string,
  labels: Record<string, string>,
  annotations: Record<string, string>
): V1ObjectMeta {
  const finalLabels = {
    app: nameEn,
    version: version,
    ...labels
  }

  const finalAnnotations = {
    'ikubeops.com/service-name': nameCn,
    ...(desc ? { description: desc } : {}),
    ...annotations
  }

  return {
    labels: finalLabels,
    annotations: finalAnnotations
  }
}

/**
 * 生成 Pod Spec
 */
function generatePodSpec(): V1PodSpec {
  const containersStore = useContainersStore()
  const volumesStore = useVolumesStore()
  const schedulingStore = useSchedulingStore()
  const advancedStore = useAdvancedStore()

  const containerConfig = containersStore.toK8sFormat()
  const schedulingSpec = schedulingStore.toK8sFormat()
  const advancedSpec = advancedStore.toK8sFormat()
  const volumes = volumesStore.volumes.length > 0 ? volumesStore.volumes : undefined

  const spec: V1PodSpec = {
    containers: containerConfig.containers,
    ...(containerConfig.initContainers && { initContainers: containerConfig.initContainers }),
    ...(volumes && volumes.length > 0 && { volumes }),
    ...(schedulingSpec.nodeName && { nodeName: schedulingSpec.nodeName }),
    ...(schedulingSpec.nodeSelector && { nodeSelector: schedulingSpec.nodeSelector }),
    ...(schedulingSpec.affinity && { affinity: schedulingSpec.affinity }),
    ...(schedulingSpec.tolerations && { tolerations: schedulingSpec.tolerations }),
    ...(schedulingSpec.topologySpreadConstraints && {
      topologySpreadConstraints: schedulingSpec.topologySpreadConstraints
    }),
    ...(advancedSpec.securityContext && { securityContext: advancedSpec.securityContext }),
    ...(advancedSpec.hostNetwork !== undefined && { hostNetwork: advancedSpec.hostNetwork }),
    ...(advancedSpec.hostPID !== undefined && { hostPID: advancedSpec.hostPID }),
    ...(advancedSpec.hostIPC !== undefined && { hostIPC: advancedSpec.hostIPC }),
    ...(advancedSpec.shareProcessNamespace !== undefined && {
      shareProcessNamespace: advancedSpec.shareProcessNamespace
    }),
    ...(advancedSpec.hostname && { hostname: advancedSpec.hostname }),
    ...(advancedSpec.subdomain && { subdomain: advancedSpec.subdomain }),
    ...(advancedSpec.setHostnameAsFQDN !== undefined && {
      setHostnameAsFQDN: advancedSpec.setHostnameAsFQDN
    }),
    ...(advancedSpec.dnsPolicy && { dnsPolicy: advancedSpec.dnsPolicy }),
    ...(advancedSpec.dnsConfig && { dnsConfig: advancedSpec.dnsConfig }),
    ...(advancedSpec.hostAliases && { hostAliases: advancedSpec.hostAliases }),
    ...(advancedSpec.serviceAccountName && { serviceAccountName: advancedSpec.serviceAccountName }),
    ...(advancedSpec.automountServiceAccountToken !== undefined && {
      automountServiceAccountToken: advancedSpec.automountServiceAccountToken
    }),
    ...(advancedSpec.imagePullSecrets && { imagePullSecrets: advancedSpec.imagePullSecrets }),
    ...(advancedSpec.restartPolicy && { restartPolicy: advancedSpec.restartPolicy }),
    ...(advancedSpec.terminationGracePeriodSeconds !== undefined && {
      terminationGracePeriodSeconds: advancedSpec.terminationGracePeriodSeconds
    }),
    ...(advancedSpec.activeDeadlineSeconds !== undefined && {
      activeDeadlineSeconds: advancedSpec.activeDeadlineSeconds
    }),
    ...(advancedSpec.priorityClassName && { priorityClassName: advancedSpec.priorityClassName }),
    ...(advancedSpec.priority !== undefined && { priority: advancedSpec.priority }),
    ...(advancedSpec.preemptionPolicy && { preemptionPolicy: advancedSpec.preemptionPolicy }),
    ...(advancedSpec.readinessGates && { readinessGates: advancedSpec.readinessGates }),
    ...(advancedSpec.overhead && { overhead: advancedSpec.overhead }),
    ...(advancedSpec.enableServiceLinks !== undefined && {
      enableServiceLinks: advancedSpec.enableServiceLinks
    }),
    ...(advancedSpec.runtimeClassName && { runtimeClassName: advancedSpec.runtimeClassName }),
    ...(advancedSpec.schedulerName && { schedulerName: advancedSpec.schedulerName }),
    ...(advancedSpec.os && { os: advancedSpec.os })
  }

  return spec
}

/**
 * 从 stores 生成 Pod 资源
 */
export function generatePodFromStores(): V1Pod {
  const metadataStore = useMetadataStore()
  const { metadata, namespace } = metadataStore
  const { nameEn, version, resourceName, nameCn, desc, labels, annotations } = metadata

  const pod: V1Pod = {
    apiVersion: 'v1',
    kind: 'Pod',
    metadata: generateMetadata(
      namespace,
      resourceName,
      nameEn,
      version,
      nameCn,
      desc,
      labels,
      annotations
    ),
    spec: generatePodSpec()
  }

  return pod
}

/**
 * 从 stores 生成 Deployment 资源
 */
export function generateDeploymentFromStores(): V1Deployment {
  const metadataStore = useMetadataStore()
  const deploymentStore = useDeploymentStore()
  const { metadata, namespace } = metadataStore
  const { nameEn, version, resourceName, nameCn, desc, labels, annotations } = metadata

  const podTemplate: V1PodTemplateSpec = {
    metadata: generatePodTemplateMetadata(nameEn, version, nameCn, desc, labels, annotations),
    spec: generatePodSpec()
  }

  // 获取 Deployment 特定配置
  const deploymentSpec = deploymentStore.toK8sFormat()

  const deployment: V1Deployment = {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: generateMetadata(
      namespace,
      resourceName,
      nameEn,
      version,
      nameCn,
      desc,
      labels,
      annotations
    ),
    spec: {
      ...deploymentSpec,
      selector: {
        matchLabels: {
          app: nameEn,
          version: version
        }
      } as V1LabelSelector,
      template: podTemplate
    }
  }

  return deployment
}

/**
 * 从 stores 生成 StatefulSet 资源
 */
export function generateStatefulSetFromStores(): V1StatefulSet {
  const metadataStore = useMetadataStore()
  const statefulSetStore = useStatefulSetStore()
  const { metadata, namespace } = metadataStore
  const { nameEn, version, resourceName, nameCn, desc, labels, annotations } = metadata

  const podTemplate: V1PodTemplateSpec = {
    metadata: generatePodTemplateMetadata(nameEn, version, nameCn, desc, labels, annotations),
    spec: generatePodSpec()
  }

  // 获取 StatefulSet 特定配置
  const statefulSetSpec = statefulSetStore.toK8sFormat()

  const statefulSet: V1StatefulSet = {
    apiVersion: 'apps/v1',
    kind: 'StatefulSet',
    metadata: generateMetadata(
      namespace,
      resourceName,
      nameEn,
      version,
      nameCn,
      desc,
      labels,
      annotations
    ),
    spec: {
      ...statefulSetSpec,
      selector: {
        matchLabels: {
          app: nameEn,
          version: version
        }
      } as V1LabelSelector,
      template: podTemplate
    }
  }

  return statefulSet
}

/**
 * 从 stores 生成 DaemonSet 资源
 */
export function generateDaemonSetFromStores(): V1DaemonSet {
  const metadataStore = useMetadataStore()
  const daemonSetStore = useDaemonSetStore()
  const { metadata, namespace } = metadataStore
  const { nameEn, version, resourceName, nameCn, desc, labels, annotations } = metadata

  const podTemplate: V1PodTemplateSpec = {
    metadata: generatePodTemplateMetadata(nameEn, version, nameCn, desc, labels, annotations),
    spec: generatePodSpec()
  }

  // 获取 DaemonSet 特定配置
  const daemonSetSpec = daemonSetStore.toK8sFormat()

  const daemonSet: V1DaemonSet = {
    apiVersion: 'apps/v1',
    kind: 'DaemonSet',
    metadata: generateMetadata(
      namespace,
      resourceName,
      nameEn,
      version,
      nameCn,
      desc,
      labels,
      annotations
    ),
    spec: {
      ...daemonSetSpec,
      selector: {
        matchLabels: {
          app: nameEn,
          version: version
        }
      } as V1LabelSelector,
      template: podTemplate
    }
  }

  return daemonSet
}

/**
 * 从 stores 生成 Job 资源
 */
export function generateJobFromStores(): V1Job {
  const metadataStore = useMetadataStore()
  const jobStore = useJobStore()
  const { metadata, namespace } = metadataStore
  const { nameEn, version, resourceName, nameCn, desc, labels, annotations } = metadata

  const podTemplate: V1PodTemplateSpec = {
    metadata: generatePodTemplateMetadata(nameEn, version, nameCn, desc, labels, annotations),
    spec: {
      ...generatePodSpec(),
      restartPolicy: 'Never' // Job 默认 restartPolicy
    }
  }

  // 获取 Job 特定配置
  const jobSpec = jobStore.toK8sFormat()

  const job: V1Job = {
    apiVersion: 'batch/v1',
    kind: 'Job',
    metadata: generateMetadata(
      namespace,
      resourceName,
      nameEn,
      version,
      nameCn,
      desc,
      labels,
      annotations
    ),
    spec: {
      ...jobSpec,
      template: podTemplate
    }
  }

  return job
}

/**
 * 从 stores 生成 CronJob 资源
 */
export function generateCronJobFromStores(): V1CronJob {
  const metadataStore = useMetadataStore()
  const cronJobStore = useCronJobStore()
  const { metadata, namespace } = metadataStore
  const { nameEn, version, resourceName, nameCn, desc, labels, annotations } = metadata

  const podTemplate: V1PodTemplateSpec = {
    metadata: generatePodTemplateMetadata(nameEn, version, nameCn, desc, labels, annotations),
    spec: {
      ...generatePodSpec(),
      restartPolicy: 'OnFailure' // CronJob 默认 restartPolicy
    }
  }

  // 获取 CronJob 特定配置
  const cronJobSpec = cronJobStore.toK8sFormat()

  // 合并 Pod 模板到 Job 模板
  cronJobSpec.jobTemplate.spec.template = podTemplate

  const cronJob: V1CronJob = {
    apiVersion: 'batch/v1',
    kind: 'CronJob',
    metadata: generateMetadata(
      namespace,
      resourceName,
      nameEn,
      version,
      nameCn,
      desc,
      labels,
      annotations
    ),
    spec: cronJobSpec
  }

  return cronJob
}

/**
 * 从 metadata.name 解析出 resourceName
 * 并判断是否为自定义名称
 */
function parseResourceName(
  metadataName: string | undefined,
  nameEn: string,
  version: string
): { resourceName: string; isCustomized: boolean } {
  if (!metadataName) {
    return {
      resourceName: `${nameEn}-${version}`,
      isCustomized: false
    }
  }

  const expectedName = `${nameEn}-${version}`
  return {
    resourceName: metadataName,
    isCustomized: metadataName !== expectedName
  }
}

/**
 * 从 K8s Pod 资源更新 stores
 */
export function updateStoresFromPod(pod: V1Pod) {
  const metadataStore = useMetadataStore()
  const containersStore = useContainersStore()
  const volumesStore = useVolumesStore()
  const schedulingStore = useSchedulingStore()
  const advancedStore = useAdvancedStore()

  console.log('🔄 开始从 Pod 资源更新所有 stores')

  if (pod.metadata) {
    const metadata = pod.metadata
    const labels = { ...(metadata.labels || {}) }
    const annotations = { ...(metadata.annotations || {}) }

    const nameEn = labels.app || ''
    const version = labels.version || ''
    const nameCn = annotations['ikubeops.com/service-name'] || ''
    const desc = annotations.description || ''

    // 解析 resourceName
    const { resourceName, isCustomized } = parseResourceName(metadata.name, nameEn, version)

    delete labels.app
    delete labels.version
    delete annotations['ikubeops.com/service-name']
    delete annotations.description
    delete annotations['created-by']

    metadataStore.updateMetadata({
      nameEn,
      version,
      resourceName,
      nameCn,
      desc,
      labels,
      annotations
    })

    // 设置是否为自定义资源名
    if (isCustomized) {
      metadataStore.resourceNameManuallySet = true
      console.log(`✋ 检测到自定义 resourceName: ${resourceName}`)
    } else {
      metadataStore.resourceNameManuallySet = false
    }

    if (metadata.namespace) {
      metadataStore.setNamespace(metadata.namespace)
    }
  }

  if (pod.spec) {
    containersStore.loadFromK8s({
      initContainers: pod.spec.initContainers,
      containers: pod.spec.containers
    })

    if (pod.spec.volumes) {
      volumesStore.loadVolumes(pod.spec.volumes)
    } else {
      volumesStore.loadVolumes([])
    }

    schedulingStore.loadFromK8s(pod.spec)
    advancedStore.loadFromK8s(pod.spec)
  }

  console.log('✅ 已从 Pod 资源更新所有 stores')
}

/**
 * 从 K8s Deployment 资源更新 stores
 */
export function updateStoresFromDeployment(deployment: V1Deployment) {
  const metadataStore = useMetadataStore()
  const containersStore = useContainersStore()
  const volumesStore = useVolumesStore()
  const schedulingStore = useSchedulingStore()
  const advancedStore = useAdvancedStore()
  const deploymentStore = useDeploymentStore()

  console.log('🔄 开始从 Deployment 资源更新所有 stores')

  // 更新 metadata store
  if (deployment.metadata) {
    const metadata = deployment.metadata
    const labels = { ...(metadata.labels || {}) }
    const annotations = { ...(metadata.annotations || {}) }

    const nameEn = labels.app || ''
    const version = labels.version || ''
    const nameCn = annotations['ikubeops.com/service-name'] || ''
    const desc = annotations.description || ''

    // 解析 resourceName
    const { resourceName, isCustomized } = parseResourceName(metadata.name, nameEn, version)

    delete labels.app
    delete labels.version
    delete annotations['ikubeops.com/service-name']
    delete annotations.description
    delete annotations['created-by']

    metadataStore.updateMetadata({
      nameEn,
      version,
      resourceName,
      nameCn,
      desc,
      labels,
      annotations
    })

    // 设置是否为自定义资源名
    if (isCustomized) {
      metadataStore.resourceNameManuallySet = true
      console.log(`✋ 检测到自定义 resourceName: ${resourceName}`)
    } else {
      metadataStore.resourceNameManuallySet = false
    }

    if (metadata.namespace) {
      metadataStore.setNamespace(metadata.namespace)
    }
  }

  // 更新 Deployment 特定配置
  if (deployment.spec) {
    deploymentStore.loadFromK8s(deployment.spec)
  }

  // 更新容器和其他配置
  const podSpec = deployment.spec?.template?.spec
  if (podSpec) {
    containersStore.loadFromK8s({
      initContainers: podSpec.initContainers,
      containers: podSpec.containers
    })

    if (podSpec.volumes) {
      volumesStore.loadVolumes(podSpec.volumes)
    } else {
      volumesStore.loadVolumes([])
    }

    schedulingStore.loadFromK8s(podSpec)
    advancedStore.loadFromK8s(podSpec)
  }

  console.log('✅ 已从 Deployment 资源更新所有 stores')
}

/**
 * 从 K8s StatefulSet 资源更新 stores
 */
export function updateStoresFromStatefulSet(statefulSet: V1StatefulSet) {
  const metadataStore = useMetadataStore()
  const containersStore = useContainersStore()
  const volumesStore = useVolumesStore()
  const schedulingStore = useSchedulingStore()
  const advancedStore = useAdvancedStore()
  const statefulSetStore = useStatefulSetStore()

  console.log('🔄 开始从 StatefulSet 资源更新所有 stores')

  // 更新 metadata store
  if (statefulSet.metadata) {
    const metadata = statefulSet.metadata
    const labels = { ...(metadata.labels || {}) }
    const annotations = { ...(metadata.annotations || {}) }

    const nameEn = labels.app || ''
    const version = labels.version || ''
    const nameCn = annotations['ikubeops.com/service-name'] || ''
    const desc = annotations.description || ''

    // 解析 resourceName
    const { resourceName, isCustomized } = parseResourceName(metadata.name, nameEn, version)

    delete labels.app
    delete labels.version
    delete annotations['ikubeops.com/service-name']
    delete annotations.description
    delete annotations['created-by']

    metadataStore.updateMetadata({
      nameEn,
      version,
      resourceName,
      nameCn,
      desc,
      labels,
      annotations
    })

    // 设置是否为自定义资源名
    if (isCustomized) {
      metadataStore.resourceNameManuallySet = true
      console.log(`✋ 检测到自定义 resourceName: ${resourceName}`)
    } else {
      metadataStore.resourceNameManuallySet = false
    }

    if (metadata.namespace) {
      metadataStore.setNamespace(metadata.namespace)
    }
  }

  // 更新 StatefulSet 特定配置
  if (statefulSet.spec) {
    statefulSetStore.loadFromK8s(statefulSet.spec)
  }

  // 更新容器和其他配置
  const podSpec = statefulSet.spec?.template?.spec
  if (podSpec) {
    containersStore.loadFromK8s({
      initContainers: podSpec.initContainers,
      containers: podSpec.containers
    })

    if (podSpec.volumes) {
      volumesStore.loadVolumes(podSpec.volumes)
    } else {
      volumesStore.loadVolumes([])
    }

    schedulingStore.loadFromK8s(podSpec)
    advancedStore.loadFromK8s(podSpec)
  }

  console.log('✅ 已从 StatefulSet 资源更新所有 stores')
}

/**
 * 从 K8s DaemonSet 资源更新 stores
 */
export function updateStoresFromDaemonSet(daemonSet: V1DaemonSet) {
  const metadataStore = useMetadataStore()
  const containersStore = useContainersStore()
  const volumesStore = useVolumesStore()
  const schedulingStore = useSchedulingStore()
  const advancedStore = useAdvancedStore()
  const daemonSetStore = useDaemonSetStore()

  console.log('🔄 开始从 DaemonSet 资源更新所有 stores')

  // 更新 metadata store
  if (daemonSet.metadata) {
    const metadata = daemonSet.metadata
    const labels = { ...(metadata.labels || {}) }
    const annotations = { ...(metadata.annotations || {}) }

    const nameEn = labels.app || ''
    const version = labels.version || ''
    const nameCn = annotations['ikubeops.com/service-name'] || ''
    const desc = annotations.description || ''

    // 解析 resourceName
    const { resourceName, isCustomized } = parseResourceName(metadata.name, nameEn, version)

    delete labels.app
    delete labels.version
    delete annotations['ikubeops.com/service-name']
    delete annotations.description
    delete annotations['created-by']

    metadataStore.updateMetadata({
      nameEn,
      version,
      resourceName,
      nameCn,
      desc,
      labels,
      annotations
    })

    // 设置是否为自定义资源名
    if (isCustomized) {
      metadataStore.resourceNameManuallySet = true
      console.log(`✋ 检测到自定义 resourceName: ${resourceName}`)
    } else {
      metadataStore.resourceNameManuallySet = false
    }

    if (metadata.namespace) {
      metadataStore.setNamespace(metadata.namespace)
    }
  }

  // 更新 DaemonSet 特定配置
  if (daemonSet.spec) {
    daemonSetStore.loadFromK8s(daemonSet.spec)
  }

  // 更新容器和其他配置
  const podSpec = daemonSet.spec?.template?.spec
  if (podSpec) {
    containersStore.loadFromK8s({
      initContainers: podSpec.initContainers,
      containers: podSpec.containers
    })

    if (podSpec.volumes) {
      volumesStore.loadVolumes(podSpec.volumes)
    } else {
      volumesStore.loadVolumes([])
    }

    schedulingStore.loadFromK8s(podSpec)
    advancedStore.loadFromK8s(podSpec)
  }

  console.log('✅ 已从 DaemonSet 资源更新所有 stores')
}

/**
 * 从 K8s Job 资源更新 stores
 */
export function updateStoresFromJob(job: V1Job) {
  const metadataStore = useMetadataStore()
  const containersStore = useContainersStore()
  const volumesStore = useVolumesStore()
  const schedulingStore = useSchedulingStore()
  const advancedStore = useAdvancedStore()
  const jobStore = useJobStore()

  console.log('🔄 开始从 Job 资源更新所有 stores')

  // 更新 metadata store
  if (job.metadata) {
    const metadata = job.metadata
    const labels = { ...(metadata.labels || {}) }
    const annotations = { ...(metadata.annotations || {}) }

    const nameEn = labels.app || ''
    const version = labels.version || ''
    const nameCn = annotations['ikubeops.com/service-name'] || ''
    const desc = annotations.description || ''

    // 解析 resourceName
    const { resourceName, isCustomized } = parseResourceName(metadata.name, nameEn, version)

    delete labels.app
    delete labels.version
    delete annotations['ikubeops.com/service-name']
    delete annotations.description
    delete annotations['created-by']

    metadataStore.updateMetadata({
      nameEn,
      version,
      resourceName,
      nameCn,
      desc,
      labels,
      annotations
    })

    // 设置是否为自定义资源名
    if (isCustomized) {
      metadataStore.resourceNameManuallySet = true
      console.log(`✋ 检测到自定义 resourceName: ${resourceName}`)
    } else {
      metadataStore.resourceNameManuallySet = false
    }

    if (metadata.namespace) {
      metadataStore.setNamespace(metadata.namespace)
    }
  }

  // 更新 Job 特定配置
  if (job.spec) {
    jobStore.loadFromK8s(job.spec)
  }

  // 更新容器和其他配置
  const podSpec = job.spec?.template?.spec
  if (podSpec) {
    containersStore.loadFromK8s({
      initContainers: podSpec.initContainers,
      containers: podSpec.containers
    })

    if (podSpec.volumes) {
      volumesStore.loadVolumes(podSpec.volumes)
    } else {
      volumesStore.loadVolumes([])
    }

    schedulingStore.loadFromK8s(podSpec)
    advancedStore.loadFromK8s(podSpec)
  }

  console.log('✅ 已从 Job 资源更新所有 stores')
}

/**
 * 从 K8s CronJob 资源更新 stores
 */
export function updateStoresFromCronJob(cronJob: V1CronJob) {
  const metadataStore = useMetadataStore()
  const containersStore = useContainersStore()
  const volumesStore = useVolumesStore()
  const schedulingStore = useSchedulingStore()
  const advancedStore = useAdvancedStore()
  const cronJobStore = useCronJobStore()

  console.log('🔄 开始从 CronJob 资源更新所有 stores')

  if (cronJob.metadata) {
    const metadata = cronJob.metadata
    const labels = { ...(metadata.labels || {}) }
    const annotations = { ...(metadata.annotations || {}) }

    const nameEn = labels.app || ''
    const version = labels.version || ''
    const nameCn = annotations['ikubeops.com/service-name'] || ''
    const desc = annotations.description || ''

    // 解析 resourceName
    const { resourceName, isCustomized } = parseResourceName(metadata.name, nameEn, version)

    delete labels.app
    delete labels.version
    delete annotations['ikubeops.com/service-name']
    delete annotations.description
    delete annotations['created-by']

    metadataStore.updateMetadata({
      nameEn,
      version,
      resourceName,
      nameCn,
      desc,
      labels,
      annotations
    })

    // 设置是否为自定义资源名
    if (isCustomized) {
      metadataStore.resourceNameManuallySet = true
      console.log(`✋ 检测到自定义 resourceName: ${resourceName}`)
    } else {
      metadataStore.resourceNameManuallySet = false
    }

    if (metadata.namespace) {
      metadataStore.setNamespace(metadata.namespace)
    }
  }

  // 更新 CronJob 特定配置
  if (cronJob.spec) {
    cronJobStore.loadFromK8s(cronJob.spec)
  }

  const podSpec = cronJob.spec?.jobTemplate?.spec?.template?.spec
  if (podSpec) {
    containersStore.loadFromK8s({
      initContainers: podSpec.initContainers,
      containers: podSpec.containers
    })

    if (podSpec.volumes) {
      volumesStore.loadVolumes(podSpec.volumes)
    } else {
      volumesStore.loadVolumes([])
    }

    schedulingStore.loadFromK8s(podSpec)
    advancedStore.loadFromK8s(podSpec)
  }

  console.log('✅ 已从 CronJob 资源更新所有 stores')
}

<template>
  <div class="application-management">
    <div class="page-header">
      <div class="selector-panel">
        <div class="selector-grid">
          <div class="selector-item">
            <span class="selector-label">
              <Server :size="14" />
              集群
            </span>
            <ElSelect
              v-model="managementStore.selectedClusterId"
              placeholder="选择集群"
              clearable
              size="default"
              filterable
              :disabled="!selectedProject"
              :loading="loadingClusters"
              @change="handleClusterChange"
              @clear="handleClusterClear"
              popper-class="cluster-dropdown"
              class="selector-select"
            >
              <ElOption
                v-for="cluster in clusters"
                :key="cluster.id"
                :label="cluster.clusterName"
                :value="cluster.id"
              >
                <div class="cluster-option">
                  <span class="option-name">{{ cluster.clusterName }}</span>
                </div>
              </ElOption>
            </ElSelect>
          </div>

          <div class="selector-item">
            <span class="selector-label">
              <Box :size="14" />
              工作空间
            </span>
            <ElSelect
              v-model="managementStore.selectedWorkspaceId"
              placeholder="选择工作空间"
              clearable
              size="default"
              :disabled="!managementStore.selectedClusterId"
              :loading="loadingWorkspaces"
              filterable
              @change="handleWorkspaceChange"
              @clear="handleWorkspaceClear"
              popper-class="workspace-dropdown"
              class="selector-select"
            >
              <ElOption
                v-for="workspace in workspaces"
                :key="workspace.id"
                :label="workspace.name"
                :value="workspace.id"
              >
                <div class="workspace-option">
                  <div class="workspace-left">
                    <Box :size="16" class="workspace-icon" />
                    <span class="option-name">{{ workspace.name }}</span>
                  </div>
                  <ElTag size="small" type="info" class="workspace-tag">
                    {{ workspace.namespace }}
                  </ElTag>
                </div>
              </ElOption>
            </ElSelect>
          </div>

          <div class="selector-item selector-item--wide">
            <span class="selector-label">
              <Server :size="14" />
              服务
            </span>
            <ElSelect
              v-model="managementStore.selectedApplicationId"
              placeholder="选择服务"
              clearable
              filterable
              size="default"
              :disabled="!managementStore.selectedWorkspace"
              :loading="loadingApplications"
              @change="handleApplicationChange"
              @clear="handleApplicationClear"
              popper-class="application-dropdown"
              class="selector-select"
            >
              <ElOption
                v-for="app in filteredApplications"
                :key="app.id"
                :label="app.nameCn || app.nameEn"
                :value="app.id"
              >
                <div class="application-option">
                  <component :is="getResourceIcon(app.resourceType)" :size="16" class="app-icon" />
                  <span class="option-name">{{ app.nameCn || app.nameEn }}</span>
                  <ElTag :type="getResourceTypeTag(app.resourceType)" size="small" class="app-tag">
                    {{ normalizeResourceType(app.resourceType) }}
                  </ElTag>
                </div>
              </ElOption>
            </ElSelect>
          </div>

          <div class="selector-item">
            <span class="selector-label">
              <Filter :size="14" />
              类型
            </span>
            <ElSelect
              v-model="managementStore.selectedResourceType"
              placeholder="全部类型"
              clearable
              size="default"
              @change="handleResourceTypeChange"
              class="selector-select"
            >
              <ElOption label="全部" value="all" />
              <ElOption label="Pod" value="pod" />
              <ElOption label="Deployment" value="deployment" />
              <ElOption label="StatefulSet" value="statefulset" />
              <ElOption label="DaemonSet" value="daemonset" />
              <ElOption label="CronJob" value="cronjob" />
              <ElOption label="Job" value="job" />
            </ElSelect>
          </div>
        </div>

        <div class="header-actions">
          <ElButton
            :type="managementStore.showServiceTree ? 'primary' : 'default'"
            :icon="managementStore.showServiceTree ? EyeOff : Eye"
            :disabled="!managementStore.selectedWorkspace"
            @click="toggleServiceTree"
          >
            {{ managementStore.showServiceTree ? '收起服务树' : '显示服务树' }}
          </ElButton>
          <ElButton
            type="primary"
            :icon="Plus"
            :disabled="!canCreate"
            @click="handleCreate"
          >
            创建服务
          </ElButton>
        </div>
      </div>
    </div>

    <div class="main-container">
      <ElContainer>
        <transition name="slide-fade">
          <ElAside
            v-show="managementStore.showServiceTree"
            :width="managementStore.sidebarWidth + 'px'"
            class="sidebar-container"
          >
            <div class="resize-handle" @mousedown="startResize"></div>
            <div class="sidebar-content">
              <ServiceTree
                :workspace="managementStore.selectedWorkspace"
                :resource-type="managementStore.selectedResourceType"
                :applications="applications"
                :loading="loadingApplications"
                :controlled="true"
                :current-application-id="managementStore.selectedApplicationId"
                @node-click="handleServiceClick"
                @refresh="loadApplications"
              />
            </div>
          </ElAside>
        </transition>

        <ElMain
          class="content-container"
          :style="{ marginLeft: managementStore.showServiceTree ? '16px' : '0' }"
        >
          <div v-if="pageLoading" class="content-loading-card">
            <ElSkeleton :rows="10" animated />
          </div>

          <template v-else-if="managementStore.selectedApplication">
            <div class="content-summary-card">
              <div class="summary-main">
                <div class="summary-icon">
                  <component
                    :is="getResourceIcon(managementStore.selectedApplication.resourceType)"
                    :size="22"
                  />
                </div>
                <div class="summary-text">
                  <div class="summary-title-row">
                    <h2>{{ selectedApplicationDisplayName }}</h2>
                    <ElTag
                      size="small"
                      :type="getResourceTypeTag(managementStore.selectedApplication.resourceType)"
                    >
                      {{ normalizeResourceType(managementStore.selectedApplication.resourceType) }}
                    </ElTag>
                  </div>
                  <p>
                    {{
                      managementStore.selectedApplication.description ||
                      '持续查看当前服务的基础信息、版本发布与运行审计。'
                    }}
                  </p>
                </div>
              </div>
              <div class="summary-meta">
                <div class="summary-meta-item">
                  <span class="meta-label">集群</span>
                  <span class="meta-value">{{ managementStore.selectedCluster?.clusterName || '-' }}</span>
                </div>
                <div class="summary-meta-item">
                  <span class="meta-label">命名空间</span>
                  <span class="meta-value">
                    {{ managementStore.selectedWorkspace?.namespace || '-' }}
                  </span>
                </div>
                <div class="summary-meta-item">
                  <span class="meta-label">当前 Tab</span>
                  <span class="meta-value">{{ activeTabLabel }}</span>
                </div>
              </div>
            </div>

            <ElCard class="main-tabs-card">
              <ElTabs
                v-model="managementStore.activeTab"
                class="main-tabs"
                @tab-change="handleTabChange"
              >
                <ElTabPane label="基础信息" name="basic">
                  <AsyncBasicInfo
                    v-if="loadedTabs.basic"
                    :application="managementStore.selectedApplication"
                    :workspace="managementStore.selectedWorkspace"
                    :cluster="managementStore.selectedCluster"
                    :refresh-trigger="refreshTriggers.basic"
                    @refresh="loadApplications"
                  />
                </ElTabPane>

                <ElTabPane label="版本管理" name="version">
                  <AsyncVersionManagement
                    v-if="loadedTabs.version"
                    :application="managementStore.selectedApplication"
                    :cluster="managementStore.selectedCluster"
                    :workspace="managementStore.selectedWorkspace"
                    :refresh-trigger="refreshTriggers.version"
                    @refresh="loadApplications"
                  />
                </ElTabPane>

                <ElTabPane v-if="!isCronJob" label="服务管理" name="service">
                  <AsyncServiceManagement
                    v-if="loadedTabs.service"
                    :application="managementStore.selectedApplication"
                    :cluster="managementStore.selectedCluster"
                    :workspace="managementStore.selectedWorkspace"
                    :refresh-trigger="refreshTriggers.service"
                  />
                </ElTabPane>

                <ElTabPane v-if="!isCronJob" label="Ingress管理" name="ingress">
                  <AsyncIngressManagement
                    v-if="loadedTabs.ingress"
                    :application="managementStore.selectedApplication"
                    :cluster="managementStore.selectedCluster"
                    :workspace="managementStore.selectedWorkspace"
                    :refresh-trigger="refreshTriggers.ingress"
                  />
                </ElTabPane>
                <ElTabPane v-if="!isCronJob" label="Flagger 发布" name="flagger">
                  <AsyncFlaggerManagement
                    v-if="loadedTabs.flagger"
                    :application="managementStore.selectedApplication"
                    :workspace="managementStore.selectedWorkspace"
                    :refresh-trigger="refreshTriggers.flagger"
                  />
                </ElTabPane>

                <ElTabPane label="日志查询" name="log">
                  <AsyncApplicationLogSearch
                    v-if="loadedTabs.log"
                    :application="managementStore.selectedApplication"
                    :workspace="managementStore.selectedWorkspace"
                    :cluster="managementStore.selectedCluster"
                    :refresh-trigger="refreshTriggers.log"
                  />
                </ElTabPane>

                <ElTabPane label="操作审计" name="audit">
                  <AsyncOperationAudit
                    v-if="loadedTabs.audit"
                    :application="managementStore.selectedApplication"
                    :workspace="managementStore.selectedWorkspace"
                    :refresh-trigger="refreshTriggers.audit"
                  />
                </ElTabPane>
              </ElTabs>
            </ElCard>
          </template>

          <div v-else class="empty-container">
            <SelectionGuideState
              level="project-cluster-workspace-app"
              :project="selectedProject"
              :cluster="managementStore.selectedCluster"
              :workspace="managementStore.selectedWorkspace"
              :application="managementStore.selectedApplication"
              resource-name="服务"
            />
          </div>
        </ElMain>
      </ElContainer>
    </div>

    <ElDialog
      v-model="createDialogVisible"
      title="创建服务"
      width="980px"
      top="6vh"
      :close-on-click-modal="false"
      class="application-create-dialog"
      @closed="resetCreateDialog"
    >
      <div class="create-dialog-body">
        <div v-if="createDialogStage === 'chooser'" class="create-entry-panel">
          <div class="entry-header">
            <div class="entry-title-group">
              <span class="entry-kicker">Create Service</span>
              <h3>选择创建模式</h3>
              <p>根据场景选择最合适的入口，点击卡片后直接进入对应模式。</p>
            </div>
            <div class="entry-context">
              <ElTag effect="plain" type="success">
                {{ managementStore.selectedCluster?.clusterName || '-' }}
              </ElTag>
              <ElTag effect="plain" type="warning">
                {{ managementStore.selectedWorkspace?.namespace || '-' }}
              </ElTag>
            </div>
          </div>

          <div class="create-mode-switcher">
            <button
              type="button"
              class="mode-option-card mode-option-card--form"
              :disabled="creatingService"
              @click="handleOpenFormCreate"
            >
              <div class="mode-option-icon mode-option-icon--form">
                <Plus :size="22" />
              </div>
              <div class="mode-option-copy">
                <h3>表单创建</h3>
                <p>沿用当前原有创建页，适合逐步配置工作负载、版本、容器和高级参数。</p>
                <span class="mode-option-action">
                  {{ creatingService ? '检查配额中...' : '点击直接进入表单模式' }}
                </span>
              </div>
            </button>

            <button
              type="button"
              class="mode-option-card mode-option-card--yaml"
              @click="openYamlCreateMode"
            >
              <div class="mode-option-icon mode-option-icon--yaml">
                <Filter :size="22" />
              </div>
              <div class="mode-option-copy">
                <h3>YAML 批量创建</h3>
                <p>适合一次导入多段 YAML，由后端按资源依赖顺序转换并批量创建。</p>
                <span class="mode-option-action">点击直接进入 YAML 批量创建模式</span>
              </div>
            </button>
          </div>
        </div>

        <div v-else class="yaml-batch-panel">
          <div class="yaml-mode-header">
            <ElButton text @click="backToCreateChooser">返回模式选择</ElButton>
            <div class="yaml-mode-copy">
              <h3>YAML 批量创建</h3>
              <p>支持多文档 YAML，后端会按依赖顺序创建，并自动注入平台注解。</p>
            </div>
          </div>

          <ElAlert
            type="warning"
            :closable="false"
            show-icon
            :title="
              isSuperAdmin
                ? '支持工作空间资源和集群级资源；会按资源依赖顺序创建。'
                : '当前用户只能创建工作空间资源，ClusterRole/StorageClass/PV/IngressClass 等集群级资源会被拒绝。'
            "
          />

          <div class="yaml-batch-meta">
            <div class="meta-chip">
              <span class="chip-label">Cluster UUID</span>
              <span class="chip-value">{{ currentClusterUuid }}</span>
            </div>
            <div class="meta-chip">
              <span class="chip-label">Namespace</span>
              <span class="chip-value">{{ currentNamespace }}</span>
            </div>
            <div class="meta-chip">
              <span class="chip-label">YAML 文档数</span>
              <span class="chip-value">{{ batchCreateAnalysis.total || 0 }}</span>
            </div>
          </div>

          <YamlEditorPro
            v-model="batchCreateYaml"
            filename="batch-create.yaml"
            height="440px"
            :readonly="false"
            :show-toolbar="true"
            :show-line-numbers="true"
            :show-status-bar="true"
          />

          <div class="yaml-batch-summary">
            <span v-if="batchCreateAnalysis.error" class="summary-error">
              YAML 解析失败：{{ batchCreateAnalysis.error }}
            </span>
            <span v-else-if="batchCreateAnalysis.total > 0" class="summary-text">
              资源类型：{{ batchCreateAnalysis.kinds.join('、') }}
            </span>
            <span
              v-if="batchCreateAnalysis.clusterScopedKinds.length > 0"
              class="summary-warning"
            >
              检测到集群级资源：{{ batchCreateAnalysis.clusterScopedKinds.join('、') }}
            </span>
          </div>
        </div>
      </div>

      <template #footer>
        <ElButton @click="createDialogVisible = false">取消</ElButton>
        <ElButton
          v-if="createDialogStage === 'yaml'"
          type="primary"
          :loading="batchCreating"
          @click="handleBatchCreate"
        >
          YAML 批量创建
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import {
    computed,
    defineAsyncComponent,
    defineComponent,
    h,
    nextTick,
    onActivated,
    onDeactivated,
    onMounted,
    ref,
    watch
  } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox, ElSkeleton } from 'element-plus'
  import { loadAll } from 'js-yaml'
  import {
    Plus,
    Server,
    Eye,
    EyeOff,
    Box,
    Database,
    Layers,
    Clock,
    Briefcase,
    Package,
    Filter
  } from 'lucide-vue-next'
  import { useProjectStore } from '@/store/modules/project'
  import { useApplicationManagementStore } from '@/store/modules/applicationManagement'
  import { useUserStore } from '@/store/modules/user'
  import {
    batchCreateApplicationResourceApi,
    searchProjectClusterApi,
    searchProjectWorkspaceApi,
    searchApplicationApi,
    type ProjectCluster,
    type ProjectWorkspace,
    type OnecProjectApplication
  } from '@/api'
  import ServiceTree from './components/ServiceTree.vue'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import SelectionGuideState from '@/components/SelectionGuideState/index.vue'
  import { parseCpu, parseMemory } from '@/utils/resource'
  import { getProjectWorkspaceApi } from '@/api'
  defineOptions({ name: 'ApplicationManagement' })

  type ApplicationTabName =
    | 'basic'
    | 'version'
    | 'service'
    | 'ingress'
    | 'flagger'
    | 'log'
    | 'audit'

  const TAB_LABEL_MAP: Record<ApplicationTabName, string> = {
    basic: '基础信息',
    version: '版本管理',
    service: '服务管理',
    ingress: 'Ingress管理',
    flagger: 'Flagger 发布',
    log: '日志查询',
    audit: '操作审计'
  }

  const HIDDEN_TABS_FOR_CRONJOB = new Set<ApplicationTabName>(['service', 'ingress', 'flagger'])

  const TabPaneLoading = defineComponent({
    name: 'ApplicationTabPaneLoading',
    setup() {
      return () => h('div', { style: 'padding: 16px 20px;' }, [h(ElSkeleton, { rows: 8, animated: true })])
    }
  })

  const createAsyncTabComponent = (loader: () => Promise<any>) =>
    defineAsyncComponent({
      loader,
      delay: 120,
      loadingComponent: TabPaneLoading
    })

  const AsyncBasicInfo = createAsyncTabComponent(() => import('./components/tabs/BasicInfo.vue'))
  const AsyncVersionManagement = createAsyncTabComponent(
    () => import('./components/tabs/VersionManagement.vue')
  )
  const AsyncServiceManagement = createAsyncTabComponent(
    () => import('./components/tabs/ServiceManagement.vue')
  )
  const AsyncIngressManagement = createAsyncTabComponent(
    () => import('./components/tabs/IngressManagement.vue')
  )
  const AsyncFlaggerManagement = createAsyncTabComponent(
    () => import('./components/tabs/FlaggerManagement.vue')
  )
  const AsyncOperationAudit = createAsyncTabComponent(
    () => import('./components/tabs/OperationAudit.vue')
  )
  const AsyncApplicationLogSearch = createAsyncTabComponent(
    () => import('./components/tabs/ApplicationLogSearch.vue')
  )
  const CLUSTER_SCOPED_KINDS = new Set([
    'clusterrole',
    'clusterrolebinding',
    'storageclass',
    'persistentvolume',
    'ingressclass'
  ])
  const WORKLOAD_KINDS = new Set(['deployment', 'statefulset', 'daemonset', 'cronjob', 'job', 'pod'])

  const router = useRouter()
  const projectStore = useProjectStore()
  const managementStore = useApplicationManagementStore()
  const userStore = useUserStore()

  const selectedProject = computed(() => projectStore.selectedProject)

  // 当前标签页存储的 key
  const STORAGE_KEY_CLUSTER = 'app-mgmt-selected-cluster'
  const STORAGE_KEY_WORKSPACE = 'app-mgmt-selected-workspace'
  const STORAGE_KEY_APPLICATION = 'app-mgmt-selected-application'
  const STORAGE_KEY_RESOURCE_TYPE = 'app-mgmt-resource-type'
  const STORAGE_KEY_TAB = 'app-mgmt-active-tab'
  const STORAGE_KEY_SHOW_TREE = 'app-mgmt-show-tree'

  // 工具函数：从 sessionStorage 加载
  function loadFromStorage(key: string): number | string | boolean | null {
    try {
      const value = sessionStorage.getItem(key)
      if (value === null) return null

      // 尝试解析为数字
      const numValue = Number(value)
      if (!isNaN(numValue) && value === String(numValue)) {
        return numValue
      }

      // 尝试解析为布尔值
      if (value === 'true') return true
      if (value === 'false') return false

      // 返回字符串
      return value
    } catch (error) {
      console.error('加载存储失败:', error)
      return null
    }
  }

  // 工具函数：保存到 sessionStorage
  function saveToStorage(key: string, value: number | string | boolean | null) {
    try {
      if (value !== null && value !== undefined) {
        sessionStorage.setItem(key, String(value))
      } else {
        sessionStorage.removeItem(key)
      }
    } catch (error) {
      console.error('保存存储失败:', error)
    }
  }

  const clusters = ref<ProjectCluster[]>([])
  const workspaces = ref<ProjectWorkspace[]>([])
  const applications = ref<OnecProjectApplication[]>([])

  const loadingClusters = ref(false)
  const loadingWorkspaces = ref(false)
  const loadingApplications = ref(false)
  const creatingService = ref(false)
  const batchCreating = ref(false)
  const createDialogVisible = ref(false)
  const createDialogStage = ref<'chooser' | 'yaml'>('chooser')
  const batchCreateYaml = ref('')

  const isInitializing = ref(false)
  const hasCompletedInitialLoad = ref(false)
  const activationSnapshot = ref({
    workspaceId: null as number | null,
    applicationId: null as number | null
  })

  const isResizing = ref(false)

  const canCreate = computed(() => {
    return !!managementStore.selectedCluster && !!managementStore.selectedWorkspace
  })
  const isSuperAdmin = computed(() => {
    const roles = userStore.getUserInfo.roles || []
    return roles.some((role) => String(role).toLowerCase() === 'super_admin')
  })
  const currentClusterUuid = computed(() => {
    return (
      managementStore.selectedCluster?.clusterUuid ||
      managementStore.selectedCluster?.uuid ||
      '-'
    )
  })
  const currentNamespace = computed(() => {
    return managementStore.selectedWorkspace?.namespace || '-'
  })

  const pageLoading = computed(() => {
    return isInitializing.value && !hasCompletedInitialLoad.value
  })

  const selectedApplicationDisplayName = computed(() => {
    return (
      managementStore.selectedApplication?.nameCn ||
      managementStore.selectedApplication?.nameEn ||
      '未选择服务'
    )
  })

  const normalizeResourceType = (type: string): string => {
    return type?.toLowerCase() || ''
  }

  const isCronJobApplication = (application: OnecProjectApplication | null = null) => {
    if (!application) return false
    return normalizeResourceType(application.resourceType) === 'cronjob'
  }

  const isCronJob = computed(() => {
    return isCronJobApplication(managementStore.selectedApplication)
  })

  const createLoadedTabsState = (): Record<ApplicationTabName, boolean> => ({
    basic: false,
    version: false,
    service: false,
    ingress: false,
    flagger: false,
    log: false,
    audit: false
  })

  const createRefreshTriggerState = (): Record<ApplicationTabName, number> => ({
    basic: 0,
    version: 0,
    service: 0,
    ingress: 0,
    flagger: 0,
    log: 0,
    audit: 0
  })

  const loadedTabs = ref<Record<ApplicationTabName, boolean>>(createLoadedTabsState())
  const refreshTriggers = ref<Record<ApplicationTabName, number>>(createRefreshTriggerState())

  const getAvailableTabs = (
    application: OnecProjectApplication | null = managementStore.selectedApplication
  ): ApplicationTabName[] => {
    const tabs = Object.keys(TAB_LABEL_MAP) as ApplicationTabName[]
    if (!isCronJobApplication(application)) {
      return tabs
    }
    return tabs.filter((tab) => !HIDDEN_TABS_FOR_CRONJOB.has(tab))
  }

  const normalizeTabName = (
    tabName: string | null | undefined,
    application: OnecProjectApplication | null = managementStore.selectedApplication
  ): ApplicationTabName => {
    const availableTabs = getAvailableTabs(application)
    if (tabName && availableTabs.includes(tabName as ApplicationTabName)) {
      return tabName as ApplicationTabName
    }
    return 'basic'
  }

  const activeTabLabel = computed(() => {
    const currentTab = normalizeTabName(managementStore.activeTab, managementStore.selectedApplication)
    return TAB_LABEL_MAP[currentTab]
  })
  const batchCreateAnalysis = computed(() => {
    const yamlText = batchCreateYaml.value.trim()
    if (!yamlText) {
      return {
        total: 0,
        kinds: [] as string[],
        clusterScopedKinds: [] as string[],
        error: '',
        targetApplicationNameEn: '',
        targetResourceType: ''
      }
    }

    try {
      const docs: Record<string, any>[] = []
      loadAll(yamlText, (doc) => {
        if (doc && typeof doc === 'object') {
          docs.push(doc as Record<string, any>)
        }
      })

      const normalizedKinds = docs
        .map((doc) => String(doc.kind || '').trim())
        .filter(Boolean)
        .map((kind) => kind.toLowerCase())

      const uniqueKinds = Array.from(new Set(normalizedKinds))
      const clusterScopedKinds = uniqueKinds.filter((kind) => CLUSTER_SCOPED_KINDS.has(kind))
      const primaryWorkload = docs.find((doc) => WORKLOAD_KINDS.has(String(doc.kind || '').toLowerCase()))

      return {
        total: docs.length,
        kinds: uniqueKinds,
        clusterScopedKinds,
        error: '',
        targetApplicationNameEn:
          String(primaryWorkload?.metadata?.labels?.app || primaryWorkload?.metadata?.name || '').trim(),
        targetResourceType: String(primaryWorkload?.kind || '').toLowerCase()
      }
    } catch (error) {
      return {
        total: 0,
        kinds: [] as string[],
        clusterScopedKinds: [] as string[],
        error: error instanceof Error ? error.message : 'YAML 解析失败',
        targetApplicationNameEn: '',
        targetResourceType: ''
      }
    }
  })

  const filteredApplications = computed(() => {
    if (managementStore.selectedResourceType === 'all') {
      return applications.value
    }
    return applications.value.filter((app) => {
      return normalizeResourceType(app.resourceType) === managementStore.selectedResourceType
    })
  })

  const getResourceIcon = (type: string) => {
    const resourceType = normalizeResourceType(type)
    const iconMap: Record<string, any> = {
      pod: Box,
      deployment: Layers,
      statefulset: Database,
      daemonset: Server,
      cronjob: Clock,
      job: Briefcase
    }
    return iconMap[resourceType] || Package
  }

  const getResourceTypeTag = (type: string) => {
    const resourceType = normalizeResourceType(type)
    const tagMap: Record<string, string> = {
      pod: 'info',
      deployment: 'success',
      statefulset: 'warning',
      daemonset: 'primary',
      cronjob: 'danger',
      job: ''
    }
    return tagMap[resourceType] || 'info'
  }

  const toggleServiceTree = () => {
    const newValue = !managementStore.showServiceTree
    managementStore.setShowServiceTree(newValue)
    saveToStorage(STORAGE_KEY_SHOW_TREE, newValue)
  }

  const resetCreateDialog = () => {
    createDialogStage.value = 'chooser'
    batchCreateYaml.value = ''
    batchCreating.value = false
  }

  const openYamlCreateMode = () => {
    createDialogStage.value = 'yaml'
  }

  const backToCreateChooser = () => {
    createDialogStage.value = 'chooser'
  }

  const ensureCreateContext = () => {
    if (!selectedProject.value) {
      ElMessage.warning('请先选择项目')
      return false
    }

    if (!managementStore.selectedCluster) {
      ElMessage.warning('请先选择集群')
      return false
    }

    if (!managementStore.selectedWorkspace) {
      ElMessage.warning('请先选择工作空间')
      return false
    }

    return true
  }

  const resetLoadedTabs = () => {
    loadedTabs.value = createLoadedTabsState()
  }

  const updateActivationSnapshot = () => {
    activationSnapshot.value = {
      workspaceId: managementStore.selectedWorkspaceId,
      applicationId: managementStore.selectedApplicationId
    }
  }

  const triggerTabRefresh = (tabName: ApplicationTabName) => {
    nextTick(() => {
      refreshTriggers.value[tabName]++
    })
  }

  const activateTab = (
    tabName: string | ApplicationTabName,
    options: { refresh?: boolean; reset?: boolean } = {}
  ) => {
    const { refresh = true, reset = false } = options
    const nextTab = normalizeTabName(tabName, managementStore.selectedApplication)
    managementStore.setActiveTab(nextTab)
    saveToStorage(STORAGE_KEY_TAB, nextTab)

    if (reset) {
      resetLoadedTabs()
    }

    if (!loadedTabs.value[nextTab]) {
      loadedTabs.value[nextTab] = true
    }

    if (refresh) {
      triggerTabRefresh(nextTab)
    }

    return nextTab
  }

  const refreshCurrentTab = () => {
    if (!managementStore.selectedApplication) {
      return
    }
    const currentTab = normalizeTabName(managementStore.activeTab, managementStore.selectedApplication)
    triggerTabRefresh(currentTab)
  }

  const clearApplicationSelection = () => {
    managementStore.setApplication(null)
    managementStore.setActiveTab('basic')
    resetLoadedTabs()
    saveToStorage(STORAGE_KEY_APPLICATION, null)
    saveToStorage(STORAGE_KEY_TAB, null)
  }

  const selectApplication = (
    application: OnecProjectApplication,
    options: { tab?: ApplicationTabName; refresh?: boolean; reset?: boolean } = {}
  ) => {
    const { tab = 'basic', refresh = true, reset = true } = options

    managementStore.setApplication(application)
    saveToStorage(STORAGE_KEY_APPLICATION, application.id)
    activateTab(tab, { refresh, reset })
  }

  const syncSelectedApplication = (
    list: OnecProjectApplication[],
    options: { restoreView?: boolean } = {}
  ) => {
    const { restoreView = false } = options
    const storedAppId = loadFromStorage(STORAGE_KEY_APPLICATION)
    const targetApplicationId =
      managementStore.selectedApplicationId || (typeof storedAppId === 'number' ? storedAppId : null)

    if (!targetApplicationId) {
      return null
    }

    const matchedApplication = list.find((app) => app.id === targetApplicationId) || null

    if (!matchedApplication) {
      clearApplicationSelection()
      return null
    }

    managementStore.setApplication(matchedApplication)
    saveToStorage(STORAGE_KEY_APPLICATION, matchedApplication.id)

    if (restoreView) {
      const savedResourceType = loadFromStorage(STORAGE_KEY_RESOURCE_TYPE)
      if (typeof savedResourceType === 'string') {
        const applicationResourceType = normalizeResourceType(matchedApplication.resourceType)
        const nextResourceType =
          savedResourceType === 'all' || savedResourceType === applicationResourceType
            ? savedResourceType
            : applicationResourceType
        managementStore.setResourceType(nextResourceType)
        saveToStorage(STORAGE_KEY_RESOURCE_TYPE, nextResourceType)
      }

      const savedTab = loadFromStorage(STORAGE_KEY_TAB)
      activateTab(typeof savedTab === 'string' ? savedTab : 'basic', {
        refresh: true,
        reset: true
      })
      return matchedApplication
    }

    const nextTab = normalizeTabName(managementStore.activeTab, matchedApplication)
    if (nextTab !== managementStore.activeTab) {
      activateTab(nextTab, { refresh: true, reset: true })
    } else {
      managementStore.setActiveTab(nextTab)
      saveToStorage(STORAGE_KEY_TAB, nextTab)
      if (loadedTabs.value[nextTab]) {
        return matchedApplication
      }
      loadedTabs.value[nextTab] = true
    }

    return matchedApplication
  }

  watch(selectedProject, async (newProject, oldProject) => {
    if (isInitializing.value) {
      return
    }

    if (newProject?.id !== oldProject?.id) {
      saveToStorage(STORAGE_KEY_CLUSTER, null)
      saveToStorage(STORAGE_KEY_WORKSPACE, null)
      saveToStorage(STORAGE_KEY_APPLICATION, null)
      saveToStorage(STORAGE_KEY_RESOURCE_TYPE, null)
      saveToStorage(STORAGE_KEY_TAB, null)

      managementStore.clearState()

      clusters.value = []
      workspaces.value = []
      applications.value = []
      resetLoadedTabs()

      if (newProject) {
        await loadClusters()
      }
    }
  })

  watch(
    () => managementStore.selectedClusterId,
    async (newId, oldId) => {
      if (isInitializing.value) {
        return
      }

      if (newId === oldId) {
        return
      }

      if (newId) {
        await loadWorkspaces()
      } else {
        workspaces.value = []
        applications.value = []
      }
    }
  )

  watch(
    () => managementStore.selectedWorkspaceId,
    async (newId, oldId) => {
      if (isInitializing.value) {
        return
      }

      if (newId === oldId) {
        return
      }

      if (newId) {
        await loadApplications()
      } else {
        applications.value = []
      }
    }
  )

  const loadClusters = async () => {
    if (!selectedProject.value) {
      clusters.value = []
      return
    }

    if (loadingClusters.value) {
      return
    }

    loadingClusters.value = true

    try {
      const response = await searchProjectClusterApi({
        projectId: selectedProject.value.id
      })
      clusters.value = response || []

      if (isInitializing.value) {
        const savedClusterId =
          managementStore.selectedClusterId || loadFromStorage(STORAGE_KEY_CLUSTER)
        if (savedClusterId && typeof savedClusterId === 'number') {
          const cluster = clusters.value.find((c) => c.id === savedClusterId)
          if (cluster) {
            managementStore.setCluster(cluster, true)
            saveToStorage(STORAGE_KEY_CLUSTER, cluster.id)
          } else {
            console.warn('缓存的集群不存在，清空选择')
            managementStore.setCluster(null, true)
            saveToStorage(STORAGE_KEY_CLUSTER, null)
            saveToStorage(STORAGE_KEY_WORKSPACE, null)
            saveToStorage(STORAGE_KEY_APPLICATION, null)
            saveToStorage(STORAGE_KEY_TAB, null)
          }
        }
      }
    } catch (error) {
      console.error('加载集群失败:', error)
      clusters.value = []
    } finally {
      loadingClusters.value = false
    }
  }

  const handleClusterChange = async (clusterId: number | null) => {
    saveToStorage(STORAGE_KEY_CLUSTER, clusterId)
    managementStore.setWorkspace(null)
    clearApplicationSelection()
    workspaces.value = []
    applications.value = []
    resetLoadedTabs()

    saveToStorage(STORAGE_KEY_WORKSPACE, null)

    if (clusterId) {
      const cluster = clusters.value.find((c) => c.id === clusterId)
      if (cluster) {
        managementStore.setCluster(cluster)
        await loadWorkspaces()
      }
    }
  }

  const handleClusterClear = () => {
    managementStore.setCluster(null)
    managementStore.setWorkspace(null)
    clearApplicationSelection()
    workspaces.value = []
    applications.value = []
    resetLoadedTabs()

    saveToStorage(STORAGE_KEY_CLUSTER, null)
    saveToStorage(STORAGE_KEY_WORKSPACE, null)
  }

  const loadWorkspaces = async () => {
    if (!managementStore.selectedClusterId) {
      workspaces.value = []
      return
    }

    if (loadingWorkspaces.value) {
      return
    }

    loadingWorkspaces.value = true

    try {
      const response = await searchProjectWorkspaceApi({
        projectClusterId: managementStore.selectedClusterId
      })
      workspaces.value = response || []

      if (isInitializing.value) {
        const savedWorkspaceId =
          managementStore.selectedWorkspaceId || loadFromStorage(STORAGE_KEY_WORKSPACE)
        if (savedWorkspaceId && typeof savedWorkspaceId === 'number') {
          const workspace = workspaces.value.find((w) => w.id === savedWorkspaceId)
          if (workspace) {
            managementStore.setWorkspace(workspace, true)
            saveToStorage(STORAGE_KEY_WORKSPACE, workspace.id)
          } else {
            console.warn('缓存的工作空间不存在，清空选择')
            managementStore.setWorkspace(null, true)
            saveToStorage(STORAGE_KEY_WORKSPACE, null)
            saveToStorage(STORAGE_KEY_APPLICATION, null)
            saveToStorage(STORAGE_KEY_TAB, null)
          }
        }
      }
    } catch (error) {
      console.error('加载工作空间失败:', error)
      workspaces.value = []
    } finally {
      loadingWorkspaces.value = false
    }
  }

  const handleWorkspaceChange = async (workspaceId: number | null) => {
    saveToStorage(STORAGE_KEY_WORKSPACE, workspaceId)
    clearApplicationSelection()
    applications.value = []
    resetLoadedTabs()

    if (workspaceId) {
      const workspace = workspaces.value.find((w) => w.id === workspaceId)
      if (workspace) {
        managementStore.setWorkspace(workspace, true)
        await loadApplications()
      }
    } else {
      managementStore.setWorkspace(null)
    }
  }

  const handleWorkspaceClear = () => {
    managementStore.setWorkspace(null)
    clearApplicationSelection()
    applications.value = []
    resetLoadedTabs()

    saveToStorage(STORAGE_KEY_WORKSPACE, null)
  }

  const handleResourceTypeChange = () => {
    saveToStorage(STORAGE_KEY_RESOURCE_TYPE, managementStore.selectedResourceType)
    managementStore.setResourceType(managementStore.selectedResourceType)

    if (managementStore.selectedApplication && managementStore.selectedResourceType !== 'all') {
      const appResourceType = normalizeResourceType(
        managementStore.selectedApplication.resourceType
      )
      if (appResourceType !== managementStore.selectedResourceType) {
        clearApplicationSelection()
      }
    }
  }

  const loadApplications = async () => {
    if (!managementStore.selectedWorkspace) {
      applications.value = []
      return
    }

    if (loadingApplications.value) {
      return
    }

    loadingApplications.value = true

    try {
      const response = await searchApplicationApi({
        workspaceId: managementStore.selectedWorkspace.id
      })
      applications.value = response || []

      syncSelectedApplication(applications.value, { restoreView: isInitializing.value })
    } catch (error) {
      console.error('加载应用列表失败:', error)
      applications.value = []
    } finally {
      loadingApplications.value = false
    }
  }

  const handleApplicationChange = (appId: number | null) => {
    if (appId) {
      const app = applications.value.find((a) => a.id === appId)
      if (app) {
        selectApplication(app, { tab: 'basic', refresh: true, reset: true })
      }
    } else {
      clearApplicationSelection()
    }
  }

  const handleApplicationClear = () => {
    clearApplicationSelection()
  }

  const handleServiceClick = (application: OnecProjectApplication) => {
    selectApplication(application, { tab: 'basic', refresh: true, reset: true })
  }

  const handleCreate = () => {
    if (!ensureCreateContext()) {
      return
    }

    createDialogVisible.value = true
  }

  const handleOpenFormCreate = async () => {
    if (!ensureCreateContext()) {
      return
    }

    creatingService.value = true

    try {
      const workspaceDetail = await getProjectWorkspaceApi(managementStore.selectedWorkspace.id)

      const cpuValue = parseCpu(workspaceDetail.cpuAllocated).value
      const memValue = parseMemory(workspaceDetail.memAllocated).value
      const podsValue = workspaceDetail.podsAllocated

      const hasInsufficientQuota = cpuValue < 1 || memValue < 1 || podsValue < 1

      if (hasInsufficientQuota) {
        const insufficientResources = []
        if (cpuValue < 1) insufficientResources.push('CPU配额不足（需要 ≥ 1核）')
        if (memValue < 1) insufficientResources.push('内存配额不足（需要 ≥ 1Gi）')
        if (podsValue < 1) insufficientResources.push('Pod配额不足（需要 ≥ 1个）')

        await ElMessageBox.confirm(
          `当前工作空间 "${workspaceDetail.name}" 的资源配额不足，无法创建应用：\n\n${insufficientResources.join('\n')}\n\n是否前往工作空间管理页面配置资源配额？`,
          '资源配额不足',
          {
            confirmButtonText: '前往配置',
            cancelButtonText: '取消',
            type: 'warning',
            customClass: 'quota-warning-message-box'
          }
        )

        createDialogVisible.value = false
        managementStore.markRefreshOnActivate(true)
        router.push({
          name: 'WorkspaceManagement',
          params: { id: workspaceDetail.id }
        })

        return
      }

      const clusterUuid =
        managementStore.selectedCluster.clusterUuid ||
        managementStore.selectedCluster.uuid ||
        `cluster-${managementStore.selectedCluster.id}`

      createDialogVisible.value = false
      managementStore.markRefreshOnActivate(true)
      router.push({
        name: 'AppCreateManager',
        query: {
          resourceClusterId: String(managementStore.selectedCluster.id),
          clusterUuid: clusterUuid,
          appProjectId: String(selectedProject.value.id),
          workspaceId: String(managementStore.selectedWorkspace.id),
          namespace: managementStore.selectedWorkspace.namespace || 'default',
          mode: 'createApp'
        }
      })
    } catch (error: any) {
      // 用户取消操作
      if (error === 'cancel') {
        return
      }
      console.error('打开表单创建失败:', error)
    } finally {
      creatingService.value = false
    }
  }

  const handleBatchCreate = async () => {
    if (!ensureCreateContext()) {
      return
    }

    if (!batchCreateYaml.value.trim()) {
      ElMessage.warning('请先输入 YAML 内容')
      return
    }

    if (batchCreateAnalysis.value.error) {
      ElMessage.error(`YAML 解析失败: ${batchCreateAnalysis.value.error}`)
      return
    }

    if (
      batchCreateAnalysis.value.clusterScopedKinds.length > 0 &&
      !isSuperAdmin.value
    ) {
      ElMessage.error('当前账号只能创建工作空间资源，不能创建集群级资源')
      return
    }

    batchCreating.value = true
    try {
      await batchCreateApplicationResourceApi({
        clusterUuid: currentClusterUuid.value,
        namespace: currentNamespace.value,
        yamlStr: batchCreateYaml.value
      })

      ElMessage.success('YAML 批量创建成功')
      createDialogVisible.value = false

      await loadApplications()

      const targetAppNameEn = batchCreateAnalysis.value.targetApplicationNameEn
      const targetResourceType = batchCreateAnalysis.value.targetResourceType
      if (targetAppNameEn) {
        const matchedApplication = applications.value.find((application) => {
          return (
            application.nameEn === targetAppNameEn &&
            (!targetResourceType || normalizeResourceType(application.resourceType) === targetResourceType)
          )
        })
        if (matchedApplication) {
          selectApplication(matchedApplication, { tab: 'basic', refresh: true, reset: true })
        }
      }
    } catch (error) {
      console.error('YAML 批量创建失败:', error)
    } finally {
      batchCreating.value = false
    }
  }

  const handleRefresh = async () => {
    await loadApplications()
    if (managementStore.selectedApplication) {
      refreshCurrentTab()
    }
  }

  const handleTabChange = (tabName: string) => {
    activateTab(tabName, { refresh: true, reset: false })
  }

  const startResize = (e: MouseEvent) => {
    isResizing.value = true
    const startX = e.clientX
    const startWidth = managementStore.sidebarWidth

    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing.value) return
      const deltaX = e.clientX - startX
      const newWidth = startWidth + deltaX
      const clampedWidth = Math.max(250, Math.min(500, newWidth))
      if (clampedWidth !== managementStore.sidebarWidth) {
        managementStore.sidebarWidth = clampedWidth
      }
    }

    const handleMouseUp = () => {
      isResizing.value = false
      managementStore.setSidebarWidth(managementStore.sidebarWidth)
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  }

  // 页面初始化逻辑
  const initPage = async () => {
    if (isInitializing.value) {
      return
    }

    isInitializing.value = true

    try {
      await projectStore.ensureInitialized()
      await managementStore.ensureInitialized()

      const savedShowTree = loadFromStorage(STORAGE_KEY_SHOW_TREE)
      if (savedShowTree !== null && typeof savedShowTree === 'boolean') {
        managementStore.setShowServiceTree(savedShowTree)
      }

      if (!selectedProject.value) {
        console.warn('未选择项目，清空所有状态')
        saveToStorage(STORAGE_KEY_CLUSTER, null)
        saveToStorage(STORAGE_KEY_WORKSPACE, null)
        saveToStorage(STORAGE_KEY_APPLICATION, null)
        saveToStorage(STORAGE_KEY_RESOURCE_TYPE, null)
        saveToStorage(STORAGE_KEY_TAB, null)
        return
      }

      await loadClusters()

      if (managementStore.selectedClusterId) {
        await loadWorkspaces()

        if (managementStore.selectedWorkspaceId) {
          await loadApplications()
        }
      }
    } catch (error) {
      console.error('页面初始化失败:', error)
    } finally {
      setTimeout(() => {
        isInitializing.value = false
        hasCompletedInitialLoad.value = true
        updateActivationSnapshot()
      }, 100)
    }
  }

  let initPromise: Promise<void> | null = null

  onMounted(async () => {
    if (!initPromise) {
      initPromise = initPage()
    }
    await initPromise
    updateActivationSnapshot()
  })

  onActivated(async () => {
    if (!hasCompletedInitialLoad.value) {
      if (!initPromise) {
        initPromise = initPage()
      }
      await initPromise
      return
    }

    const workspaceChanged = activationSnapshot.value.workspaceId !== managementStore.selectedWorkspaceId
    const applicationChanged =
      activationSnapshot.value.applicationId !== managementStore.selectedApplicationId
    const shouldRefresh =
      managementStore.shouldRefreshOnActivate || workspaceChanged || applicationChanged

    if (!shouldRefresh || !managementStore.selectedWorkspace || loadingApplications.value) {
      managementStore.markRefreshOnActivate(false)
      updateActivationSnapshot()
      return
    }

    await loadApplications()

    if (managementStore.selectedApplication) {
      refreshCurrentTab()
    }

    managementStore.markRefreshOnActivate(false)
    updateActivationSnapshot()
  })

  onDeactivated(() => {
    updateActivationSnapshot()
  })
</script>

<style lang="scss" scoped>
  .application-management {
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 0;
    background: linear-gradient(180deg, #f7f9fc 0%, #f3f6fb 100%);

    .page-header {
      display: block;

      .selector-panel {
        display: flex;
        align-items: flex-end;
        justify-content: space-between;
        gap: 12px;
        padding: 12px 14px;
        background: rgba(255, 255, 255, 0.92);
        border: 1px solid #e5eaf3;
        border-radius: 14px;
        box-shadow: 0 8px 20px rgba(15, 23, 42, 0.04);
        backdrop-filter: blur(12px);
        flex-wrap: wrap;

        .selector-grid {
          flex: 1;
          min-width: 0;
          display: grid;
          grid-template-columns: repeat(12, minmax(0, 1fr));
          gap: 10px 12px;

          .selector-item {
            grid-column: span 2;
            display: flex;
            flex-direction: column;
            gap: 6px;
            min-width: 0;

            &.selector-item--wide {
              grid-column: span 4;
            }

            .selector-label {
              display: inline-flex;
              align-items: center;
              gap: 6px;
              font-size: 12px;
              color: #4b5563;
              font-weight: 600;
            }

            .selector-select {
              width: 100%;

              :deep(.el-input__wrapper) {
                min-height: 36px;
                background: #f8fafc;
                border: 1px solid #d7deea;
                border-radius: 8px;
                box-shadow: none;
                transition: border-color 0.2s, box-shadow 0.2s, background-color 0.2s;

                &:hover {
                  border-color: #b8c5d9;
                  background: #fff;
                }

                &.is-focus {
                  border-color: #409eff;
                  box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.12);
                }
              }

              :deep(.el-input__inner) {
                font-size: 12px;
                font-weight: 500;
              }
            }
          }
        }

        .header-actions {
          flex-shrink: 0;
          display: flex;
          align-items: center;
          gap: 10px;
        }
      }
    }

    .main-container {
      flex: 1;
      min-height: 0;
      overflow: hidden;

      :deep(.el-container) {
        height: 100%;
      }

      .sidebar-container {
        position: relative;
        background: white;
        border: 1px solid #e5eaf3;
        border-radius: 16px;
        box-shadow: 0 8px 20px rgba(15, 23, 42, 0.05);
        overflow: hidden;

        .resize-handle {
          position: absolute;
          right: 0;
          top: 0;
          bottom: 0;
          width: 4px;
          cursor: col-resize;
          background: transparent;
          z-index: 10;
          transition: background 0.2s;

          &:hover {
            background: #409eff;
          }
        }

        .sidebar-content {
          height: 100%;
          overflow: hidden;
        }
      }

      .content-container {
        display: flex;
        flex-direction: column;
        gap: 16px;
        padding: 0;
        margin: 0;
        height: 100%;
        min-height: 0;
        overflow: hidden;
        transition: margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1);

        .content-loading-card,
        .empty-container {
          flex: 1;
          min-height: 0;
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 24px;
          background: white;
          border: 1px solid #e5eaf3;
          border-radius: 16px;
          box-shadow: 0 8px 20px rgba(15, 23, 42, 0.05);
        }

        .content-loading-card {
          align-items: flex-start;
        }

        .content-summary-card {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 20px;
          padding: 18px 20px;
          background: linear-gradient(135deg, #ffffff 0%, #f8fbff 100%);
          border: 1px solid #e5eaf3;
          border-radius: 16px;
          box-shadow: 0 8px 20px rgba(15, 23, 42, 0.05);

          .summary-main {
            display: flex;
            align-items: center;
            gap: 16px;
            min-width: 0;

            .summary-icon {
              display: inline-flex;
              align-items: center;
              justify-content: center;
              width: 52px;
              height: 52px;
              border-radius: 14px;
              background: #eef4ff;
              color: #2563eb;
              flex-shrink: 0;
            }

            .summary-text {
              min-width: 0;

              .summary-title-row {
                display: flex;
                align-items: center;
                gap: 10px;
                min-width: 0;

                h2 {
                  margin: 0;
                  font-size: 20px;
                  line-height: 1.3;
                  font-weight: 700;
                  color: #1f2937;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
                }
              }

              p {
                margin: 6px 0 0;
                color: #667085;
                font-size: 13px;
                line-height: 1.7;
              }
            }
          }

          .summary-meta {
            display: grid;
            grid-template-columns: repeat(3, minmax(0, 1fr));
            gap: 12px;
            min-width: 360px;

            .summary-meta-item {
              padding: 12px 14px;
              border-radius: 12px;
              background: #f8fafc;
              border: 1px solid #edf2f7;

              .meta-label {
                display: block;
                margin-bottom: 6px;
                color: #98a2b3;
                font-size: 12px;
                line-height: 1;
              }

              .meta-value {
                display: block;
                color: #344054;
                font-size: 13px;
                font-weight: 600;
                line-height: 1.4;
                word-break: break-all;
              }
            }
          }
        }

        .main-tabs-card {
          flex: 1;
          min-height: 0;
          overflow: hidden;
          border: 1px solid #e5eaf3;
          box-shadow: 0 8px 20px rgba(15, 23, 42, 0.05);
          border-radius: 16px;

          :deep(.el-card__body) {
            height: 100%;
            padding: 0;
          }

          .main-tabs {
            height: 100%;
            display: flex;
            flex-direction: column;

            :deep(.el-tabs__header) {
              margin: 0;
              padding: 0 20px;
              background: #fbfcfe;
              border-bottom: 1px solid #e5eaf3;
              flex-shrink: 0;

              .el-tabs__nav-wrap::after {
                display: none;
              }

              .el-tabs__item {
                height: 52px;
                line-height: 52px;
                padding: 0 18px;
                font-size: 14px;
                color: #667085;

                &:hover {
                  color: #409eff;
                }

                &.is-active {
                  color: #1d4ed8;
                  font-weight: 600;
                }
              }

              .el-tabs__active-bar {
                height: 3px;
                background: linear-gradient(90deg, #409eff 0%, #1d4ed8 100%);
                border-radius: 999px;
              }
            }

            > :deep(.el-tabs__content) {
              flex: 1;
              min-height: 0;
              padding: 0;
              overflow-y: auto;
              background: #f8fafc;
            }
          }
        }
      }
    }
  }

  .slide-fade-enter-active,
  .slide-fade-leave-active {
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .slide-fade-enter-from,
  .slide-fade-leave-to {
    transform: translateX(-20px);
    opacity: 0;
  }

  @media (max-width: 1440px) {
    .application-management {
      .page-header {
        .selector-panel {
          .selector-grid {
            grid-template-columns: repeat(6, minmax(0, 1fr));

            .selector-item {
              grid-column: span 2;

              &.selector-item--wide {
                grid-column: span 4;
              }
            }
          }
        }
      }
    }
  }

  @media (max-width: 1200px) {
    .application-management {
      .page-header {
        .selector-panel {
          flex-direction: column;
          align-items: stretch;
        }
      }

      .main-container {
        .content-container {
          .content-summary-card {
            flex-direction: column;
            align-items: stretch;

            .summary-meta {
              min-width: 0;
            }
          }
        }
      }
    }
  }

  @media (max-width: 768px) {
    .application-management {
      .page-header {
        .selector-panel {
          padding: 12px;

          .selector-grid {
            grid-template-columns: repeat(1, minmax(0, 1fr));

            .selector-item,
            .selector-item.selector-item--wide {
              grid-column: span 1;
            }
          }

          .header-actions {
            width: 100%;

            :deep(.el-button) {
              flex: 1;
            }
          }
        }
      }

      .main-container {
        .content-container {
          .content-summary-card {
            padding: 16px;

            .summary-meta {
              grid-template-columns: repeat(1, minmax(0, 1fr));
            }
          }

          .main-tabs-card {
            .main-tabs {
              :deep(.el-tabs__header) {
                padding: 0 12px;
              }
            }
          }
        }
      }
    }
  }

  .create-dialog-body {
    display: flex;
    flex-direction: column;
    gap: 18px;

    .create-entry-panel {
      display: flex;
      flex-direction: column;
      gap: 18px;

      .entry-header {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 16px;
        padding: 4px 2px;

        .entry-title-group {
          min-width: 0;

          .entry-kicker {
            display: inline-flex;
            align-items: center;
            padding: 4px 10px;
            border-radius: 999px;
            background: #eef4ff;
            color: #2563eb;
            font-size: 12px;
            font-weight: 600;
            letter-spacing: 0.04em;
            text-transform: uppercase;
          }

          h3 {
            margin: 10px 0 8px;
            font-size: 24px;
            line-height: 1.2;
            color: #1f2937;
          }

          p {
            margin: 0;
            color: #667085;
            font-size: 14px;
            line-height: 1.7;
          }
        }

        .entry-context {
          display: flex;
          flex-wrap: wrap;
          gap: 10px;
          justify-content: flex-end;
        }
      }
    }

    .create-mode-switcher {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 16px;

      .mode-option-card {
        border: 1px solid #dbe4f0;
        background: linear-gradient(180deg, #fbfdff 0%, #f7faff 100%);
        border-radius: 20px;
        padding: 22px 20px;
        display: flex;
        align-items: flex-start;
        gap: 16px;
        text-align: left;
        cursor: pointer;
        transition: all 0.2s ease;
        position: relative;
        overflow: hidden;

        &:hover {
          border-color: #9cc7ff;
          transform: translateY(-2px);
          box-shadow: 0 16px 32px rgba(64, 158, 255, 0.12);
        }

        &::after {
          content: '';
          position: absolute;
          inset: auto -40px -40px auto;
          width: 120px;
          height: 120px;
          border-radius: 50%;
          background: rgba(64, 158, 255, 0.08);
          pointer-events: none;
        }

        &:disabled {
          cursor: wait;
          opacity: 0.8;
        }

        .mode-option-icon {
          width: 52px;
          height: 52px;
          border-radius: 16px;
          display: inline-flex;
          align-items: center;
          justify-content: center;
          flex-shrink: 0;

          &.mode-option-icon--form {
            background: rgba(64, 158, 255, 0.12);
            color: #2563eb;
          }

          &.mode-option-icon--yaml {
            background: rgba(16, 185, 129, 0.12);
            color: #0f766e;
          }
        }

        .mode-option-copy {
          min-width: 0;
          position: relative;
          z-index: 1;

          h3 {
            margin: 0 0 8px;
            font-size: 18px;
            color: #1f2937;
          }

          p {
            margin: 0;
            color: #667085;
            line-height: 1.6;
            font-size: 13px;
          }

          .mode-option-action {
            display: inline-flex;
            align-items: center;
            margin-top: 14px;
            font-size: 13px;
            font-weight: 600;
            color: #2563eb;
          }
        }
      }
    }

    .create-mode-card {
      display: flex;
      flex-direction: column;
      gap: 16px;

      .mode-copy {
        h3 {
          margin: 0 0 8px;
          font-size: 18px;
          color: #1f2937;
        }

        p {
          margin: 0;
          color: #667085;
          line-height: 1.7;
        }
      }

      .mode-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
      }
    }

    .yaml-batch-panel {
      display: flex;
      flex-direction: column;
      gap: 16px;

      .yaml-mode-header {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 16px;

        .yaml-mode-copy {
          flex: 1;
          min-width: 0;

          h3 {
            margin: 0 0 6px;
            font-size: 18px;
            color: #1f2937;
          }

          p {
            margin: 0;
            color: #667085;
            line-height: 1.7;
            font-size: 13px;
          }
        }
      }

      .yaml-batch-meta {
        display: grid;
        grid-template-columns: repeat(3, minmax(0, 1fr));
        gap: 12px;

        .meta-chip {
          padding: 12px 14px;
          border: 1px solid #e5eaf3;
          border-radius: 12px;
          background: #f8fafc;

          .chip-label {
            display: block;
            margin-bottom: 6px;
            font-size: 12px;
            color: #98a2b3;
          }

          .chip-value {
            display: block;
            font-size: 13px;
            font-weight: 600;
            color: #344054;
            word-break: break-all;
          }
        }
      }

      .yaml-batch-summary {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        font-size: 13px;

        .summary-text {
          color: #475467;
        }

        .summary-warning {
          color: #b54708;
        }

        .summary-error {
          color: #d92d20;
        }
      }
    }
  }

  @media (max-width: 768px) {
    .create-dialog-body {
      .create-entry-panel {
        .entry-header {
          flex-direction: column;
          align-items: stretch;

          .entry-context {
            justify-content: flex-start;
          }
        }
      }

      .create-mode-switcher {
        grid-template-columns: repeat(1, minmax(0, 1fr));
      }

      .yaml-batch-panel {
        .yaml-batch-meta {
          grid-template-columns: repeat(1, minmax(0, 1fr));
        }

        .yaml-mode-header {
          flex-direction: column;
          align-items: stretch;
        }
      }
    }
  }
</style>

<style lang="scss">
  /* 集群下拉菜单样式 */
  .cluster-dropdown {
    min-width: 400px !important;

    .el-select-dropdown__wrap {
      min-height: 300px !important;
      max-height: 300px !important;
    }

    .el-select-dropdown__item {
      height: auto !important;
      line-height: normal !important;
      padding: 0 !important;
      margin: 0 !important;

      &.hover {
        background-color: transparent !important;

        .cluster-option {
          background-color: #f5f7fa;
        }
      }

      &.selected {
        background-color: transparent !important;
        font-weight: normal !important;

        .cluster-option {
          background-color: #ecf5ff;

          .option-name {
            color: #409eff;
            font-weight: 500;
          }
        }
      }
    }

    .cluster-option {
      display: flex !important;
      align-items: center !important;
      justify-content: space-between !important;
      width: 100%;
      gap: 12px;
      padding: 10px 16px !important;
      border-radius: 4px;
      transition: background-color 0.2s;
      height: auto !important;
      box-sizing: border-box;

      .option-name {
        font-weight: 500;
        color: #303133;
        flex: 1;
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 14px;
        line-height: 1;
        margin: 0 !important;
        padding: 0 !important;
        vertical-align: middle;
      }

      .option-meta {
        font-size: 12px;
        color: #909399;
        flex-shrink: 0;
        white-space: nowrap;
        line-height: 1;
        margin: 0 !important;
        padding: 0 !important;
        vertical-align: middle;
      }
    }
  }

  /* 工作空间下拉菜单样式 */
  .workspace-dropdown {
    min-width: 400px !important;

    .el-select-dropdown__wrap {
      min-height: 300px !important;
      max-height: 300px !important;
    }

    .el-select-dropdown__item {
      height: auto !important;
      line-height: normal !important;
      padding: 0 !important;
      margin: 0 !important;

      &.hover {
        background-color: transparent !important;

        .workspace-option {
          background-color: #f5f7fa;
        }
      }

      &.selected {
        background-color: transparent !important;
        font-weight: normal !important;

        .workspace-option {
          background-color: #ecf5ff;

          .option-name {
            color: #409eff;
            font-weight: 500;
          }
        }
      }
    }

    .workspace-option {
      display: flex !important;
      align-items: center !important;
      justify-content: space-between !important;
      width: 100%;
      gap: 12px;
      padding: 10px 16px !important;
      border-radius: 4px;
      transition: background-color 0.2s;
      height: auto !important;
      box-sizing: border-box;

      .workspace-left {
        display: flex !important;
        align-items: center !important;
        flex: 1;
        min-width: 0;
        gap: 10px;

        .workspace-icon {
          flex-shrink: 0;
          color: #606266;
          display: flex !important;
          align-items: center !important;
          justify-content: center !important;
          width: 16px;
          height: 16px;
          line-height: 1;
        }

        .option-name {
          font-weight: 500;
          color: #303133;
          flex: 1;
          min-width: 0;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
          font-size: 14px;
          line-height: 1;
          margin: 0 !important;
          padding: 0 !important;
          vertical-align: middle;
        }
      }

      .workspace-tag {
        flex-shrink: 0 !important;
        margin: 0 !important;

        :deep(.el-tag) {
          margin: 0 !important;
          vertical-align: middle !important;
        }
      }
    }
  }

  /* 应用/服务下拉菜单样式 */
  .application-dropdown {
    min-width: 500px !important;
    max-width: 600px !important;

    .el-select-dropdown__wrap {
      min-height: 300px !important;
      max-height: 300px !important;
    }

    .el-select-dropdown__item {
      height: auto !important;
      line-height: normal !important;
      padding: 0 !important;
      margin: 0 !important;

      &.hover {
        background-color: transparent !important;

        .application-option {
          background-color: #f5f7fa;
        }
      }

      &.selected {
        background-color: transparent !important;
        font-weight: normal !important;

        .application-option {
          background-color: #ecf5ff;

          .option-name {
            color: #409eff;
            font-weight: 500;
          }
        }
      }
    }

    .application-option {
      display: flex !important;
      align-items: center !important;
      gap: 10px;
      width: 100%;
      padding: 10px 16px !important;
      border-radius: 4px;
      transition: background-color 0.2s;
      height: auto !important;
      box-sizing: border-box;

      .app-icon {
        flex-shrink: 0;
        color: #606266;
        display: flex !important;
        align-items: center !important;
        justify-content: center !important;
        width: 16px;
        height: 16px;
        line-height: 1;
      }

      .option-name {
        flex: 1;
        min-width: 0;
        font-weight: 500;
        color: #303133;
        font-size: 14px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        line-height: 1;
        margin: 0 !important;
        padding: 0 !important;
        vertical-align: middle;
      }

      .app-tag {
        flex-shrink: 0 !important;
        margin: 0 !important;

        :deep(.el-tag) {
          margin: 0 !important;
          vertical-align: middle !important;
        }
      }
    }
  }
</style>

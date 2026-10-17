<template>
  <div class="basic-config-tab">
    <ElForm :model="formData" label-width="100px" label-position="left" size="default">
      <!-- 容器名称 -->
      <ElRow :gutter="24">
        <ElCol :span="22">
          <ElFormItem required>
            <template #label>
              <span class="label-text">
                容器名称
                <ElTooltip
                  content="容器的唯一标识名称，只能包含小写字母、数字和连字符"
                  placement="top"
                >
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="formData.name"
              placeholder="例如: app-container"
              clearable
              @input="debouncedUpdate"
            >
              <template #prefix>
                <Box :size="14" />
              </template>
            </ElInput>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <!-- 镜像配置区域 -->
      <ElDivider content-position="left">
        <span class="divider-title">
          镜像配置
          <ElTooltip content="配置容器使用的镜像，支持从多个仓库选择" placement="top">
            <Info :size="12" class="divider-hint" />
          </ElTooltip>
        </span>
      </ElDivider>

      <!-- 镜像搜索框 -->
      <ElRow :gutter="24">
        <ElCol :span="22">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                镜像搜索
                <ElTooltip content="快速搜索镜像，自动过滤下拉列表" placement="top">
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="imageSearch"
              placeholder="搜索镜像名称，如: nginx, redis..."
              clearable
              @keyup.enter="handleImageSearchClick"
            >
              <template #prefix>
                <Search :size="16" />
              </template>
              <template #append>
                <ElButton @click="handleImageSearchClick" :loading="searchingImages">
                  <Search :size="14" />
                  搜索
                </ElButton>
              </template>
            </ElInput>
            <div v-if="searchApplied" class="search-hint">
              <Info :size="12" />
              <span>已应用搜索过滤，找到 {{ getTotalFilteredCount() }} 个结果</span>
              <ElButton text type="primary" size="small" @click="clearSearch"> 清除过滤 </ElButton>
            </div>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <!-- 镜像选择器 -->
      <ElRow :gutter="16">
        <ElCol :span="11">
          <ElFormItem required>
            <template #label>
              <span class="label-text">
                镜像仓库
                <ElTooltip content="选择镜像仓库，支持 Harbor、云厂商等" placement="top">
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElSelect
              v-model="imageConfig.registry"
              placeholder="选择仓库"
              filterable
              :loading="loadingRegistries"
              @change="handleRegistryChange"
              style="width: 100%"
              popper-class="image-select-popper"
            >
              <ElOption
                v-for="registry in displayRegistries"
                :key="registry.value"
                :label="registry.label"
                :value="registry.value"
              >
                <div class="registry-option">
                  <span>{{ registry.label }}</span>
                  <ElTag size="small" type="info">{{ registry.type }}</ElTag>
                </div>
              </ElOption>
              <template #footer>
                <div
                  v-if="registryPagination.hasMore"
                  v-loading="loadingRegistries"
                  class="select-loading-more"
                  @click="loadMoreRegistries"
                >
                  <span v-if="!loadingRegistries">点击加载更多</span>
                </div>
              </template>
            </ElSelect>
          </ElFormItem>
        </ElCol>

        <ElCol :span="11">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                项目/命名空间
                <ElTooltip content="选择镜像所属的项目或命名空间" placement="top">
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElSelect
              v-model="imageConfig.project"
              placeholder="选择项目"
              filterable
              :loading="loadingProjects"
              :disabled="!imageConfig.registry"
              @change="handleProjectChange"
              style="width: 100%"
              popper-class="image-select-popper"
            >
              <ElOption
                v-for="project in displayProjects"
                :key="project.value"
                :label="project.label"
                :value="project.value"
              />
              <template #footer>
                <div
                  v-if="projectPagination.hasMore"
                  v-loading="loadingProjects"
                  class="select-loading-more"
                  @click="loadMoreProjects"
                >
                  <span v-if="!loadingProjects">点击加载更多</span>
                </div>
              </template>
            </ElSelect>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElRow :gutter="16">
        <ElCol :span="11">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                镜像名称
                <ElTooltip content="选择要使用的镜像" placement="top">
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElSelect
              v-model="imageConfig.image"
              placeholder="选择镜像"
              filterable
              :loading="loadingImages"
              :disabled="!imageConfig.project"
              @change="handleImageChange"
              style="width: 100%"
              popper-class="image-select-popper"
            >
              <ElOption
                v-for="img in images"
                :key="img.value"
                :label="img.label"
                :value="img.value"
              />
              <template #footer>
                <div
                  v-if="imagePagination.hasMore"
                  v-loading="loadingImages"
                  class="select-loading-more"
                  @click="loadMoreImages"
                >
                  <span v-if="!loadingImages">点击加载更多</span>
                </div>
              </template>
            </ElSelect>
          </ElFormItem>
        </ElCol>

        <ElCol :span="11">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                版本标签
                <ElTooltip content="选择镜像的版本标签" placement="top">
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElSelect
              v-model="imageConfig.tag"
              placeholder="选择标签"
              filterable
              :loading="loadingTags"
              :disabled="!imageConfig.image"
              @change="handleTagChange"
              style="width: 100%"
              popper-class="image-select-popper"
            >
              <ElOption v-for="tag in tags" :key="tag.value" :label="tag.label" :value="tag.value">
                <div class="tag-option">
                  <span>{{ tag.label }}</span>
                  <span class="tag-size">{{ tag.size }}</span>
                </div>
              </ElOption>
              <template #footer>
                <div
                  v-if="tagPagination.hasMore"
                  v-loading="loadingTags"
                  class="select-loading-more"
                  @click="loadMoreTags"
                >
                  <span v-if="!loadingTags">点击加载更多</span>
                </div>
              </template>
            </ElSelect>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <!-- 镜像信息展示 -->
      <ElRow v-if="fullImageUrl" :gutter="24">
        <ElCol :span="22">
          <div class="image-info-card">
            <div class="image-info-header">
              <span class="info-label">完整镜像地址</span>
              <ElButton text type="primary" @click="copyImageUrl" size="small">
                <Copy :size="14" style="margin-right: 4px" />
                复制
              </ElButton>
            </div>
            <div class="image-url">
              <code>{{ fullImageUrl }}</code>
              <ElTag v-if="imageValidated" size="small" type="success">
                <CheckCircle :size="12" style="margin-right: 2px" />
                已验证
              </ElTag>
            </div>
            <div v-if="imageMetadata" class="image-metadata">
              <div class="metadata-item">
                <span class="metadata-label">大小:</span>
                <span class="metadata-value">{{ imageMetadata.size }}</span>
              </div>
              <div class="metadata-item">
                <span class="metadata-label">创建时间:</span>
                <span class="metadata-value">{{ imageMetadata.created }}</span>
              </div>
              <div class="metadata-item">
                <span class="metadata-label">拉取次数:</span>
                <span class="metadata-value">{{ imageMetadata.pullCount }}</span>
              </div>
            </div>
          </div>
        </ElCol>
      </ElRow>

      <!-- 拉取策略 -->
      <ElRow :gutter="24">
        <ElCol :span="22">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                拉取策略
                <ElTooltip placement="top">
                  <template #content>
                    <div style="max-width: 300px; line-height: 1.6">
                      <p><strong>Always:</strong> 每次都拉取最新镜像</p>
                      <p><strong>IfNotPresent:</strong> 本地不存在时才拉取</p>
                      <p><strong>Never:</strong> 只使用本地镜像</p>
                    </div>
                  </template>
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElRadioGroup v-model="formData.imagePullPolicy" @change="updateContainer">
              <ElRadioButton value="Always">总是拉取</ElRadioButton>
              <ElRadioButton value="IfNotPresent">不存在时拉取</ElRadioButton>
              <ElRadioButton value="Never">从不拉取</ElRadioButton>
            </ElRadioGroup>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElDivider content-position="left">
        <span class="divider-title">
          启动配置
          <ElTooltip content="配置容器启动时的命令和参数" placement="top">
            <Info :size="12" class="divider-hint" />
          </ElTooltip>
        </span>
      </ElDivider>

      <ElRow :gutter="24">
        <ElCol :span="22">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                启动命令
                <ElTooltip placement="top">
                  <template #content>
                    <div style="max-width: 350px; line-height: 1.6">
                      <p><strong>覆盖镜像的 ENTRYPOINT</strong></p>
                      <p>每行一个参数，例如：</p>
                      <p
                        style="
                          font-family: var(--ax-font-mono);
                          background: #f5f5f5;
                          padding: 8px;
                          margin: 4px 0;
                        "
                      >
                        /bin/sh<br />
                        -c<br />
                        echo hello world
                      </p>
                      <p style="color: #e6a23c; margin-top: 8px">
                        ⚠️ 复杂脚本应该作为单个参数（第3行及之后）
                      </p>
                    </div>
                  </template>
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="commandString"
              type="textarea"
              :rows="5"
              placeholder="每行一个参数，例如：
/bin/sh
-c
echo 'hello world'; while true; do echo 'running...'; sleep 30; done"
              @change="updateCommand"
              class="command-textarea"
            />
            <div class="form-tip">
              <Terminal :size="12" style="margin-right: 4px" />
              每行代表一个数组元素，复杂脚本放在同一行避免被分割
            </div>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElRow :gutter="24">
        <ElCol :span="22">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                启动参数
                <ElTooltip placement="top">
                  <template #content>
                    <div style="max-width: 350px; line-height: 1.6">
                      <p><strong>覆盖镜像的 CMD</strong></p>
                      <p>每行一个参数，例如：</p>
                      <p
                        style="
                          font-family: var(--ax-font-mono);
                          background: #f5f5f5;
                          padding: 8px;
                          margin: 4px 0;
                        "
                      >
                        --config<br />
                        /etc/app.conf<br />
                        --verbose
                      </p>
                    </div>
                  </template>
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="argsString"
              type="textarea"
              :rows="4"
              placeholder="每行一个参数，例如：
--config
/etc/app.conf
--verbose"
              @change="updateArgs"
              class="args-textarea"
            />
            <div class="form-tip">
              <Code :size="12" style="margin-right: 4px" />
              传递给启动命令的参数列表
            </div>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <!-- 工作目录 -->
      <ElRow :gutter="24">
        <ElCol :span="22">
          <ElFormItem>
            <template #label>
              <span class="label-text">
                工作目录
                <ElTooltip content="容器内进程的工作目录，默认为根目录 /" placement="top">
                  <Info :size="13" class="hint-icon" />
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="formData.workingDir"
              placeholder="例如: /app"
              clearable
              @input="debouncedUpdate"
            >
              <template #prefix>
                <Folder :size="14" />
              </template>
            </ElInput>
          </ElFormItem>
        </ElCol>
      </ElRow>
    </ElForm>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted, onBeforeUnmount, inject } from 'vue'
  import { useContainersStore } from '@/store/workload'
  import { Box, Search, Copy, CheckCircle, Terminal, Code, Folder, Info } from 'lucide-vue-next'
  import { ElMessage } from 'element-plus'
  import {
    listRegistriesByProjectApi,
    listProjectsByAppApi,
    searchImagesGlobalByProjectApi,
    listRepositoriesApi,
    listArtifactsApi
  } from '@/api'

  const containersStore = useContainersStore()

  // 从父组件注入的参数
  const clusterUuid = inject<any>('clusterUuid')
  const workspaceId = inject<any>('workspaceId')
  const appProjectId = inject<any>('appProjectId')

  // 镜像搜索
  const imageSearch = ref('')
  const searchingImages = ref(false)
  const searchApplied = ref(false)

  // 搜索结果数据
  const searchResults = ref<any>(null)

  // 镜像配置
  const imageConfig = ref({
    registry: '',
    registryUuid: '',
    project: '',
    image: '',
    tag: ''
  })

  // 加载状态
  const loadingRegistries = ref(false)
  const loadingProjects = ref(false)
  const loadingImages = ref(false)
  const loadingTags = ref(false)

  // 分页状态
  const registryPagination = ref({
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: false
  })

  const projectPagination = ref({
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: false
  })

  const imagePagination = ref({
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: false
  })

  const tagPagination = ref({
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: false
  })

  const searchPagination = ref({
    page: 1,
    pageSize: 50,
    total: 0,
    hasMore: false
  })

  // 原始数据列表
  const registries = ref<any[]>([])
  const projects = ref<any[]>([])
  const images = ref<any[]>([])
  const tags = ref<any[]>([])

  function removeProjectPrefix(repoName: string, projectName: string): string {
    if (!repoName || !projectName) return repoName || ''
    const projectPrefix = `${projectName}/`
    if (repoName.startsWith(projectPrefix)) {
      return repoName.substring(projectPrefix.length)
    }
    return repoName
  }

  const displayRegistries = computed(() => {
    if (
      !searchApplied.value ||
      !searchResults.value?.data ||
      !Array.isArray(searchResults.value.data)
    ) {
      return registries.value
    }
    const searchRegistryUuids = new Set(
      searchResults.value.data.map((r: any) => r?.registryUuid).filter(Boolean)
    )
    return registries.value.filter((r) => searchRegistryUuids.has(r.uuid))
  })

  const displayProjects = computed(() => {
    if (!searchApplied.value || !searchResults.value?.data || !imageConfig.value.registryUuid) {
      return projects.value
    }

    const registryResult = searchResults.value.data.find(
      (r: any) => r?.registryUuid === imageConfig.value.registryUuid
    )

    if (!registryResult?.images || !Array.isArray(registryResult.images)) {
      return []
    }

    const searchProjectNames = new Set(
      registryResult.images.map((img: any) => img?.projectName).filter(Boolean)
    )
    return projects.value.filter((p) => searchProjectNames.has(p.value))
  })

  // 镜像元数据
  const imageMetadata = ref<any>(null)

  // 表单数据
  const formData = ref({
    name: '',
    image: '',
    imagePullPolicy: 'IfNotPresent',
    workingDir: ''
  })

  // 命令和参数
  const commandString = ref('')
  const argsString = ref('')

  // 计算完整镜像地址
  const fullImageUrl = computed(() => {
    const { registry, project, image, tag } = imageConfig.value
    if (!image) return ''

    let url = ''

    // 仓库地址（已经清理过协议头）
    if (registry && registry !== 'docker.io') {
      url = registry + '/'
    }

    // 项目名称
    if (project) {
      url += project + '/'
    }

    // 镜像名称
    url += image

    // 标签
    if (tag) {
      url += ':' + tag
    }

    return url
  })

  const imageValidated = computed(() => {
    return !!fullImageUrl.value && !!imageMetadata.value
  })

  // 防抖更新
  let updateTimer: NodeJS.Timeout | null = null
  const debouncedUpdate = () => {
    if (updateTimer) clearTimeout(updateTimer)
    updateTimer = setTimeout(() => {
      updateContainer()
    }, 300)
  }

  // 获取过滤后的总数量
  function getTotalFilteredCount() {
    if (!searchResults.value) return 0
    return searchResults.value.data.reduce((sum: number, r: any) => sum + r.images.length, 0)
  }

  // 格式化文件大小
  function formatSize(bytes: number): string {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
  }

  // 格式化日期
  function formatDate(timestamp: number): string {
    if (!timestamp) return ''
    const date = new Date(timestamp * 1000)
    return date.toLocaleDateString('zh-CN')
  }

  // ==================== API 调用（支持分页）====================

  // 清理仓库 URL，移除协议头
  function cleanRegistryUrl(url: string): string {
    if (!url) return url
    // 移除 http:// 或 https://
    return url.replace(/^https?:\/\//, '')
  }

  // 获取镜像仓库列表
  async function fetchRegistries(page: number = 1, append: boolean = false) {
    if (!appProjectId?.value || !clusterUuid?.value) {
      console.warn('缺少必要参数: appProjectId 或 clusterUuid')
      return
    }

    loadingRegistries.value = true
    try {
      const res = await listRegistriesByProjectApi({
        appProjectId: appProjectId.value,
        clusterUuid: clusterUuid.value,
        page: page,
        pageSize: registryPagination.value.pageSize
      })

      const newData = res.data.map((registry) => ({
        label: registry.name,
        value: cleanRegistryUrl(registry.url), // 清理 URL
        uuid: registry.uuid,
        type: registry.type || 'Harbor'
      }))

      if (append) {
        registries.value = [...registries.value, ...newData]
      } else {
        registries.value = newData
      }

      registryPagination.value.page = page
      registryPagination.value.total = res.total
      registryPagination.value.hasMore = registries.value.length < res.total
    } catch (error) {
      console.error('❌ 加载镜像仓库列表失败:', error)
    } finally {
      loadingRegistries.value = false
    }
  }

  // 加载更多仓库
  async function loadMoreRegistries() {
    if (loadingRegistries.value || !registryPagination.value.hasMore) return
    await fetchRegistries(registryPagination.value.page + 1, true)
  }

  // 获取项目列表
  async function fetchProjects(registryUuid: string, page: number = 1, append: boolean = false) {
    if (!appProjectId?.value || !clusterUuid?.value || !registryUuid) {
      return
    }

    loadingProjects.value = true
    try {
      const res = await listProjectsByAppApi({
        appProjectId: appProjectId.value,
        clusterUuid: clusterUuid.value,
        registryUuid: registryUuid,
        page: page,
        pageSize: projectPagination.value.pageSize
      })

      const newData = res.items.map((project) => ({
        label: project.name,
        value: project.name,
        projectId: project.projectId
      }))

      if (append) {
        projects.value = [...projects.value, ...newData]
      } else {
        projects.value = newData
      }

      projectPagination.value.page = page
      projectPagination.value.total = res.total
      projectPagination.value.hasMore = projects.value.length < res.total
    } catch (error) {
      console.error('❌ 加载项目列表失败:', error)
    } finally {
      loadingProjects.value = false
    }
  }

  // 加载更多项目
  async function loadMoreProjects() {
    if (
      loadingProjects.value ||
      !projectPagination.value.hasMore ||
      !imageConfig.value.registryUuid
    )
      return
    await fetchProjects(imageConfig.value.registryUuid, projectPagination.value.page + 1, true)
  }

  // 获取镜像列表
  async function fetchImages(
    registryUuid: string,
    projectName: string,
    page: number = 1,
    append: boolean = false
  ) {
    if (!registryUuid || !projectName) {
      return
    }

    loadingImages.value = true
    try {
      const res = await listRepositoriesApi({
        registryUuid: registryUuid,
        projectName: projectName,
        page: page,
        pageSize: imagePagination.value.pageSize,
        sortBy: 'update_time',
        sortDesc: true
      })

      const newData = res.items.map((repo) => {
        // 智能处理镜像名称，移除项目前缀（如果存在）
        let imageName = repo.name
        const projectPrefix = `${projectName}/`
        if (imageName.startsWith(projectPrefix)) {
          imageName = imageName.substring(projectPrefix.length)
        }

        return {
          label: imageName,
          value: imageName,
          artifactCount: repo.artifactCount,
          pullCount: repo.pullCount
        }
      })

      if (append) {
        images.value = [...images.value, ...newData]
      } else {
        images.value = newData
      }

      imagePagination.value.page = page
      imagePagination.value.total = res.total
      imagePagination.value.hasMore = images.value.length < res.total
    } catch (error) {
      console.error('❌ 加载镜像列表失败:', error)
    } finally {
      loadingImages.value = false
    }
  }

  // 加载更多镜像
  async function loadMoreImages() {
    if (searchApplied.value) {
      return
    }

    if (
      loadingImages.value ||
      !imagePagination.value.hasMore ||
      !imageConfig.value.registryUuid ||
      !imageConfig.value.project
    )
      return
    await fetchImages(
      imageConfig.value.registryUuid,
      imageConfig.value.project,
      imagePagination.value.page + 1,
      true
    )
  }

  // 获取标签列表
  async function fetchTags(
    registryUuid: string,
    projectName: string,
    repoName: string,
    page: number = 1,
    append: boolean = false
  ) {
    if (!registryUuid || !projectName || !repoName) {
      return
    }

    loadingTags.value = true
    try {
      const res = await listArtifactsApi({
        registryUuid: registryUuid,
        projectName: projectName,
        repoName: repoName,
        page: page,
        pageSize: tagPagination.value.pageSize,
        sortBy: 'push_time',
        sortDesc: true
      })

      const newTags: any[] = []
      res.items.forEach((artifact) => {
        artifact.tags.forEach((tag) => {
          newTags.push({
            label: tag.name,
            value: tag.name,
            size: formatSize(artifact.size),
            pushTime: formatDate(tag.pushTime),
            pullTime: tag.pullTime ? formatDate(tag.pullTime) : undefined,
            artifactId: artifact.id,
            digest: artifact.digest
          })
        })
      })

      if (append) {
        tags.value = [...tags.value, ...newTags]
      } else {
        tags.value = newTags
      }

      tagPagination.value.page = page
      tagPagination.value.total = res.total
      tagPagination.value.hasMore = tags.value.length < res.total
    } catch (error) {
      console.error('❌ 加载标签列表失败:', error)
    } finally {
      loadingTags.value = false
    }
  }

  // 加载更多标签
  async function loadMoreTags() {
    if (searchApplied.value) {
      return
    }

    if (
      loadingTags.value ||
      !tagPagination.value.hasMore ||
      !imageConfig.value.registryUuid ||
      !imageConfig.value.project ||
      !imageConfig.value.image
    )
      return
    await fetchTags(
      imageConfig.value.registryUuid,
      imageConfig.value.project,
      imageConfig.value.image,
      tagPagination.value.page + 1,
      true
    )
  }

  // 搜索镜像（支持分页）
  async function handleImageSearch(page: number = 1, append: boolean = false) {
    if (!imageSearch.value.trim()) {
      return
    }

    if (!appProjectId?.value || !clusterUuid?.value) {
      return
    }

    searchingImages.value = true
    try {
      const res = await searchImagesGlobalByProjectApi({
        appProjectId: appProjectId.value,
        clusterUuid: clusterUuid.value,
        imageName: imageSearch.value.trim(),
        page: page,
        pageSize: searchPagination.value.pageSize
      })

      // 清理搜索结果中的仓库 URL
      res.data = res.data.map((registry: any) => ({
        ...registry,
        registryUrl: cleanRegistryUrl(registry.registryUrl)
      }))

      if (append && searchResults.value) {
        // 追加数据
        searchResults.value.data = [...searchResults.value.data, ...res.data]
      } else {
        searchResults.value = res
      }

      searchPagination.value.page = page
      searchPagination.value.total = res.total
      searchPagination.value.hasMore = searchResults.value.data.length < res.total

      if (searchResults.value.data.length === 0 && page === 1) {
        ElMessage.info('未找到匹配的镜像')
        searchApplied.value = false
        return
      }

      searchApplied.value = true

      const totalImages = searchResults.value.data.reduce(
        (sum: number, r: any) => sum + r.images.length,
        0
      )

      if (page === 1) {
        ElMessage.success(`找到 ${totalImages} 个匹配的镜像，已自动过滤下拉列表`)
        // 重置选择
        imageConfig.value.registry = ''
        imageConfig.value.registryUuid = ''
        imageConfig.value.project = ''
        imageConfig.value.image = ''
        imageConfig.value.tag = ''
      }
    } catch (error) {
      console.error('❌ 搜索镜像失败:', error)
      searchApplied.value = false
    } finally {
      searchingImages.value = false
    }
  }

  // 清除搜索
  function clearSearch() {
    imageSearch.value = ''
    searchResults.value = null
    searchApplied.value = false
    searchPagination.value = { page: 1, pageSize: 50, total: 0, hasMore: false }
    ElMessage.info('已清除搜索过滤')
  }

  function handleImageSearchClick() {
    handleImageSearch(1, false)
  }

  // ==================== 事件处理 ====================

  // 处理仓库变化
  async function handleRegistryChange() {
    imageConfig.value.project = ''
    imageConfig.value.image = ''
    imageConfig.value.tag = ''
    imageMetadata.value = null
    projects.value = []
    images.value = []
    tags.value = []
    projectPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }
    imagePagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }
    tagPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }

    if (imageConfig.value.registry) {
      const selectedRegistry = registries.value.find((r) => r.value === imageConfig.value.registry)
      if (selectedRegistry) {
        imageConfig.value.registryUuid = selectedRegistry.uuid
        await fetchProjects(selectedRegistry.uuid)
      }
    }
  }

  async function handleProjectChange() {
    imageConfig.value.image = ''
    imageConfig.value.tag = ''
    imageMetadata.value = null
    images.value = []
    tags.value = []
    imagePagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }
    tagPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }

    if (!imageConfig.value.project || !imageConfig.value.registryUuid) return

    if (searchApplied.value && searchResults.value?.data) {
      const registryResult = searchResults.value.data.find(
        (r: any) => r?.registryUuid === imageConfig.value.registryUuid
      )

      if (registryResult?.images && Array.isArray(registryResult.images)) {
        // 过滤当前项目下的镜像
        const projectImages = registryResult.images.filter(
          (img: any) => img?.projectName === imageConfig.value.project
        )

        // 直接构建镜像列表，不需要调用 API
        images.value = projectImages.map((img: any) => {
          const originalName = img.repoName || ''
          const cleanedName = removeProjectPrefix(originalName, imageConfig.value.project)

          return {
            label: cleanedName,
            value: cleanedName,
            artifactCount: 0,
            pullCount: 0
          }
        })

        imagePagination.value.total = images.value.length
        imagePagination.value.hasMore = false

        if (images.value.length === 0) {
          ElMessage.error({
            message: `项目 "${imageConfig.value.project}" 下没有匹配 "${imageSearch.value}" 的镜像`,
            duration: 3000
          })
        }
        return
      }
    }

    await fetchImages(imageConfig.value.registryUuid, imageConfig.value.project)
  }

  async function handleImageChange() {
    imageConfig.value.tag = ''
    imageMetadata.value = null
    tags.value = []
    tagPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }

    if (!imageConfig.value.image || !imageConfig.value.registryUuid || !imageConfig.value.project)
      return

    if (searchApplied.value && searchResults.value?.data) {
      const registryResult = searchResults.value.data.find(
        (r: any) => r?.registryUuid === imageConfig.value.registryUuid
      )

      if (registryResult?.images && Array.isArray(registryResult.images)) {
        // 在搜索结果中查找匹配的镜像
        const imageResult = registryResult.images.find((img: any) => {
          if (!img?.repoName || !img?.projectName) return false
          const cleanedName = removeProjectPrefix(img.repoName, img.projectName)
          return (
            img.projectName === imageConfig.value.project && cleanedName === imageConfig.value.image
          )
        })

        if (imageResult?.tags && Array.isArray(imageResult.tags)) {
          // 直接构建标签列表
          tags.value = imageResult.tags.map((tagName: string) => ({
            label: tagName,
            value: tagName,
            size: '-',
            pushTime: '',
            artifactId: 0,
            digest: ''
          }))

          tagPagination.value.total = tags.value.length
          tagPagination.value.hasMore = false

          return
        }
      }
    }

    await fetchTags(
      imageConfig.value.registryUuid,
      imageConfig.value.project,
      imageConfig.value.image
    )
  }

  // 处理标签变化
  async function handleTagChange() {
    if (fullImageUrl.value) {
      formData.value.image = fullImageUrl.value

      const selectedTag = tags.value.find((t) => t.value === imageConfig.value.tag)
      if (selectedTag) {
        imageMetadata.value = {
          size: selectedTag.size,
          created: selectedTag.pushTime,
          pullCount: selectedTag.pullCount || 0
        }
      }

      updateContainer()
    }
  }

  // 复制镜像地址
  async function copyImageUrl() {
    try {
      await navigator.clipboard.writeText(fullImageUrl.value)
      ElMessage.success('镜像地址已复制到剪贴板')
    } catch {}
  }

  function updateCommand() {
    const container = containersStore.selectedContainer
    if (!container) return

    const commands = commandString.value
      ? commandString.value
          .split('\n') // 改为按换行符分割
          .map((c) => c.trim())
          .filter((c) => c)
      : []

    containersStore.updateContainer(container.id, {
      command: commands.length > 0 ? commands : undefined
    })
  }

  function updateArgs() {
    const container = containersStore.selectedContainer
    if (!container) return

    const args = argsString.value
      ? argsString.value
          .split('\n') // 改为按换行符分割
          .map((a) => a.trim())
          .filter((a) => a)
      : []

    containersStore.updateContainer(container.id, {
      args: args.length > 0 ? args : undefined
    })
  }

  // 更新容器
  function updateContainer() {
    const container = containersStore.selectedContainer
    if (!container) return

    containersStore.updateContainer(container.id, {
      name: formData.value.name,
      image: formData.value.image,
      imagePullPolicy: formData.value.imagePullPolicy,
      workingDir: formData.value.workingDir || undefined
    })
  }

  // 解析镜像地址
  async function parseAndRestoreImageConfig(imageUrl: string) {
    if (!imageUrl) {
      imageConfig.value = { registry: '', registryUuid: '', project: '', image: '', tag: '' }
      return
    }

    // 移除可能存在的协议头
    let remaining = imageUrl.replace(/^https?:\/\//, '')
    let registry = ''
    let project = ''
    let imageName = ''
    let tag = ''

    const tagIndex = remaining.lastIndexOf(':')
    if (tagIndex > 0) {
      const afterColon = remaining.substring(tagIndex + 1)
      if (!afterColon.includes('/')) {
        tag = afterColon
        remaining = remaining.substring(0, tagIndex)
      }
    }

    const parts = remaining.split('/')
    if (parts.length >= 3) {
      registry = parts[0]
      project = parts[1]
      imageName = parts.slice(2).join('/')
    } else if (parts.length === 2) {
      if (parts[0].includes('.') || parts[0].includes(':')) {
        registry = parts[0]
        imageName = parts[1]
      } else {
        registry = 'docker.io'
        project = parts[0]
        imageName = parts[1]
      }
    } else if (parts.length === 1) {
      registry = 'docker.io'
      project = 'library'
      imageName = parts[0]
    }

    imageConfig.value = { registry, registryUuid: '', project, image: imageName, tag }

    const matchedRegistry = registries.value.find((r) => r.value === registry)
    if (matchedRegistry) {
      imageConfig.value.registryUuid = matchedRegistry.uuid
      await fetchProjects(matchedRegistry.uuid)
      if (project) {
        await fetchImages(matchedRegistry.uuid, project)
        if (imageName) {
          await fetchTags(matchedRegistry.uuid, project, imageName)
        }
      }
    }
  }

  // 清空所有数据
  function clearAllData() {
    formData.value = {
      name: '',
      image: '',
      imagePullPolicy: 'IfNotPresent',
      workingDir: ''
    }
    commandString.value = ''
    argsString.value = ''
    imageConfig.value = { registry: '', registryUuid: '', project: '', image: '', tag: '' }
    imageMetadata.value = null
  }

  // 加载容器数据
  async function loadContainerData() {
    const container = containersStore.selectedContainer

    if (!container) {
      clearAllData()
      return
    }

    formData.value = {
      name: container.name || '',
      image: container.image || '',
      imagePullPolicy: container.imagePullPolicy || 'IfNotPresent',
      workingDir: container.workingDir || ''
    }

    commandString.value = container.command ? container.command.join('\n') : ''
    argsString.value = container.args ? container.args.join('\n') : ''

    if (container.image) {
      await parseAndRestoreImageConfig(container.image)
    } else {
      imageConfig.value = { registry: '', registryUuid: '', project: '', image: '', tag: '' }
      imageMetadata.value = null
    }
  }

  // 保存当前数据
  function saveCurrentData() {
    const container = containersStore.selectedContainer
    if (!container) return

    containersStore.updateContainer(container.id, {
      name: formData.value.name,
      image: formData.value.image || fullImageUrl.value,
      imagePullPolicy: formData.value.imagePullPolicy,
      workingDir: formData.value.workingDir || undefined,
      command: commandString.value
        ? commandString.value
            .split('\n')
            .map((c) => c.trim())
            .filter((c) => c)
        : undefined,
      args: argsString.value
        ? argsString.value
            .split('\n')
            .map((a) => a.trim())
            .filter((a) => a)
        : undefined
    })
  }

  // 生命周期
  onMounted(async () => {
    await fetchRegistries()
    await loadContainerData()
  })

  onBeforeUnmount(() => {
    //  如果正在从 YAML 加载，不要保存
    if (containersStore.isLoadingFromYaml) {
      console.log('⚠️ 正在加载 YAML，跳过保存容器基本配置')
      return
    }

    //  确保有选中的容器
    const currentId = containersStore.selectedContainerId
    if (!currentId) {
      console.log('⚠️ 没有选中容器，跳过保存基本配置')
      return
    }

    // 确保容器仍然存在
    const container = containersStore.getContainerById(currentId)
    if (!container) {
      console.log('⚠️ 容器不存在，跳过保存基本配置')
      return
    }

    //  确保表单数据有效
    const hasValidData = formData.value.name || formData.value.image
    if (!hasValidData) {
      console.log('⚠️ 表单数据为空，跳过保存基本配置')
      return
    }

    // 通过所有检查，安全保存
    console.log('💾 BasicConfigTab 保存容器基本配置:', currentId)
    saveCurrentData()
  })

  // 监听容器切换
  watch(
    () => containersStore.selectedContainerId,
    () => {
      loadContainerData()
    }
  )
</script>

<style lang="scss" scoped>
  .basic-config-tab {
    padding: 12px 0;

    ::v-deep(.el-form-item) {
      margin-bottom: 20px;
    }

    ::v-deep(.el-form-item__label) {
      font-size: 13px;
      color: #606266;
      font-weight: 500;
    }

    .label-text {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      white-space: nowrap;

      .hint-icon {
        color: #909399;
        cursor: help;
        transition: color 0.2s;

        &:hover {
          color: #409eff;
        }
      }
    }

    ::v-deep(.el-divider) {
      margin: 28px 0 20px;

      .divider-title {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 14px;
        font-weight: 600;
        color: #303133;

        .divider-hint {
          color: #909399;
          cursor: help;

          &:hover {
            color: #409eff;
          }
        }
      }
    }

    .command-textarea,
    .args-textarea {
      ::v-deep(.el-textarea__inner) {
        font-family: var(--ax-font-mono);
        font-size: 13px;
        line-height: 1.6;
      }
    }

    .form-tip {
      display: flex;
      align-items: center;
      margin-top: 6px;
      font-size: 12px;
      color: #909399;
      line-height: 1.6;

      code {
        padding: 2px 6px;
        background: #f5f7fa;
        border-radius: 3px;
        font-family: var(--ax-font-mono);
        color: #409eff;
        font-size: 11px;
      }
    }

    .search-hint {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-top: 8px;
      padding: 8px 12px;
      background: #e6f7ff;
      border-left: 3px solid #1890ff;
      border-radius: 4px;
      font-size: 12px;
      color: #606266;

      svg {
        color: #1890ff;
        flex-shrink: 0;
      }
    }

    .registry-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      width: 100%;
    }

    .tag-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      width: 100%;

      .tag-size {
        font-size: 11px;
        color: #909399;
        margin-left: 8px;
      }
    }

    .image-info-card {
      margin-top: 8px;
      margin-bottom: 24px;
      padding: 16px;
      background: linear-gradient(135deg, #f5f7fa 0%, #e8eef5 100%);
      border-radius: 8px;
      border: 1px solid #d9e3f0;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);

      .image-info-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 10px;

        .info-label {
          font-size: 12px;
          color: #606266;
          font-weight: 600;
        }
      }

      .image-url {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 10px;
        padding: 8px 12px;
        background: white;
        border-radius: 6px;
        box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);

        code {
          flex: 1;
          font-family: var(--ax-font-mono);
          font-size: 12px;
          color: #303133;
          word-break: break-all;
        }
      }

      .image-metadata {
        display: flex;
        gap: 20px;
        padding-top: 10px;
        border-top: 1px solid #e4e7ed;

        .metadata-item {
          display: flex;
          gap: 6px;
          font-size: 12px;

          .metadata-label {
            color: #909399;
            font-weight: 500;
          }

          .metadata-value {
            color: #303133;
            font-weight: 400;
          }
        }
      }
    }

    ::v-deep(.el-radio-button) {
      .el-radio-button__inner {
        padding: 8px 16px;
      }
    }
  }

  // 加载更多按钮样式
  .select-loading-more {
    padding: 8px;
    text-align: center;
    color: #409eff;
    font-size: 12px;
    cursor: pointer;
    border-top: 1px solid #e4e7ed;
    transition: background-color 0.2s;

    &:hover {
      background-color: #f5f7fa;
    }
  }
</style>

<style lang="scss">
  // 全局样式：下拉框弹出层
  .image-select-popper {
    .el-select-dropdown__list {
      max-height: 300px !important;
    }
  }
</style>

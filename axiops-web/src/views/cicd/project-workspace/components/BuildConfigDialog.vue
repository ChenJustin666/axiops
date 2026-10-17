<template>
  <ElDialog
    v-model="dialogVisible"
    :title="`${service?.name || '服务'} · 构建配置`"
    width="min(1200px, calc(100vw - 32px))"
    top="3vh"
    class="build-config-dialog"
    destroy-on-close
    :close-on-click-modal="false"
    :close-on-press-escape="!saving"
    :show-close="!saving"
  >
    <div v-loading="loading">
      <ElAlert v-if="loadError" type="error" :title="loadError" :closable="false">
        <ElButton text type="primary" :loading="loading" @click="load">重新加载配置</ElButton>
      </ElAlert>
      <ElForm v-else label-position="top" :disabled="saving || loading">
        <div class="step-head is-first">
          <span class="step-no">1</span>
          <div class="step-text">
            <h3>选择代码</h3>
            <p>指定从哪个代码源、仓库、分支拉取代码</p>
          </div>
        </div>
        <ElFormItem
          label="代码仓库"
          :required="service?.source_type !== 'manual' || form.build_mode !== 'artifact'"
        >
          <div class="repo-cascade">
            <ElSelect
              v-model="form.code_source_id"
              :disabled="!editable"
              placeholder="选择代码源"
              filterable
              style="flex: 1"
              @change="onCodeSourceChange"
            >
              <ElOption
                v-for="cs in codeSources"
                :key="cs.id"
                :label="`${cs.name} (${cs.type})`"
                :value="cs.id"
              />
            </ElSelect>
            <ElSelect
              v-model="form.repo_path"
              :disabled="!editable || !form.code_source_id"
              :loading="repoLoading"
              placeholder="选择仓库"
              filterable
              style="flex: 1"
              @change="onRepositoryChange"
            >
              <ElOption
                v-for="repo in repositories"
                :key="repo.full_name"
                :label="repo.full_name"
                :value="repo.full_name"
              />
            </ElSelect>
            <ElSelect
              v-model="form.repo_branch"
              :disabled="!editable || !form.repo_path"
              :loading="branchLoading"
              placeholder="选择分支"
              filterable
              style="flex: 1"
            >
              <ElOption v-for="b in branches" :key="b.name" :label="b.name" :value="b.name" />
            </ElSelect>
          </div>
          <div class="repo-hint">构建将拉取所选分支的代码</div>
          <div v-if="sourceError || repoError || branchError" class="load-error" role="alert">
            {{ sourceError || repoError || branchError }}
            <ElButton text type="primary" @click="retrySourceLists">重新加载列表</ElButton>
          </div>
        </ElFormItem>
        <div class="step-head">
          <span class="step-no">2</span>
          <div class="step-text">
            <h3>构建方式与产物</h3>
            <p>选择构建模板与输出模式，并指定镜像推送的仓库</p>
          </div>
        </div>
        <div class="form-row">
          <ElFormItem label="构建输出模式">
            <ElSegmented
              v-model="form.build_mode"
              :options="buildModeOptions"
              :disabled="!editable"
              @change="onBuildModeChange"
            />
          </ElFormItem>
          <ElFormItem v-if="form.build_mode !== 'dockerfile'" label="构建模板"
            ><div class="template-row">
              <ElSegmented
                v-model="form.template_type"
                :options="form.build_mode === 'jar_image' ? jarTemplateOptions : templateOptions"
                :disabled="!editable"
                @change="applyTemplate"
              />
              <ElButton v-if="editable" text type="primary" @click="libraryBuildVisible = true"
                >模板库</ElButton
              >
            </div>
            <ElCollapse v-if="libraryBuildVisible" class="lib-inline">
              <ElCollapseItem name="lib" title="从模板库应用">
                <DeliveryTemplateSelect
                  kind="build"
                  :disabled="!editable"
                  @select="applyLibraryBuild"
                />
              </ElCollapseItem>
            </ElCollapse>
          </ElFormItem>
        </div>
        <div
          class="form-grid"
          :class="{ thirds: form.build_mode === 'jar_image' }"
        >
          <ElFormItem
            v-if="form.build_mode !== 'dockerfile'"
            :label="
              form.build_mode === 'jar_image' ? 'JAR 编译镜像' : '构建器镜像'
            "
            ><ElInput
              v-model="form.build_image"
              :disabled="!editable"
              placeholder="使用专用 CI 集群的默认构建器镜像"
          /></ElFormItem>
          <ElFormItem v-if="form.build_mode !== 'artifact'" label="镜像仓库" required>
            <ElSelect
              v-model="form.registry_id"
              :disabled="!editable || registryLoading"
              :loading="registryLoading"
              placeholder="请选择已配置的镜像仓库"
              @change="onRegistryChange"
            >
              <ElOption
                v-for="registry in imageRegistries"
                :key="registry.id"
                :label="registry.name"
                :value="registry.id"
              />
            </ElSelect>
            <div class="registry-hint">
              {{
                registryLoadFailed
                  ? '镜像仓库列表加载失败，请重试。'
                  : '选择后自动使用该仓库已配置的推送凭据。'
              }}
              <ElButton
                v-if="registryLoadFailed"
                text
                type="primary"
                :loading="registryLoading"
                @click="loadImageRegistries"
                >重新加载</ElButton
              >
            </div>
          </ElFormItem>
          <ElFormItem
            v-if="form.build_mode !== 'artifact' && !customImagePath"
            label="命名空间 / Harbor 项目"
          >
            <ElSelect
              v-model="imageNamespace"
              :disabled="!editable || !selectedRegistry"
              filterable
              :allow-create="namespaceOptions.length === 0"
              :clearable="namespaceOptions.length === 0"
              default-first-option
              placeholder="选择命名空间；未配置时可输入"
              @change="updateGeneratedImage"
            >
              <ElOption v-for="name in namespaceOptions" :key="name" :label="name" :value="name" />
            </ElSelect>
          </ElFormItem>
        </div>
        <template v-if="form.build_mode !== 'artifact'">
          <div class="image-result" aria-live="polite">
            <div class="image-result-body">
              <ElTag size="small" :type="hasCustomImage ? 'info' : 'success'" effect="plain">
                {{ hasCustomImage ? '自定义' : '自动生成' }}
              </ElTag>
              <code>{{ form.image_repo || '选择镜像仓库后自动生成' }}</code>
            </div>
            <div class="registry-hint">
              {{
                hasCustomImage
                  ? '可在高级设置中恢复服务名称。'
                  : `镜像名称使用服务绑定 ${defaultImageName()}，无需重复填写。`
              }}
            </div>
          </div>
          <ElCollapse v-model="imageAdvancedOpen" class="image-advanced">
            <ElCollapseItem name="image" title="高级镜像设置（可选）">
              <ElFormItem v-if="!customImagePath" label="自定义镜像名称（留空使用服务名称）">
                <ElInput
                  v-model="imageNameOverride"
                  :disabled="!editable"
                  :placeholder="defaultImageName()"
                  clearable
                />
              </ElFormItem>
              <ElFormItem v-else label="自定义完整路径（不含 tag）">
                <ElInput v-model="form.image_repo" :disabled="!editable" />
              </ElFormItem>
              <div class="registry-hint">自定义路径仍需属于所选镜像仓库及其命名空间。</div>
              <template v-if="editable">
                <ElButton text type="primary" @click="toggleImagePathMode">{{
                  customImagePath ? '按仓库和命名空间拼接' : '编辑完整路径'
                }}</ElButton>
                <ElButton v-if="hasCustomImage" text @click="restoreServiceImageName">
                  恢复服务名称
                </ElButton>
              </template>
            </ElCollapseItem>
          </ElCollapse>
        </template>
        <div class="step-head">
          <span class="step-no">3</span>
          <div class="step-text">
            <h3>构建命令与 Dockerfile</h3>
            <p>自定义编译命令，或指定镜像构建所用的 Dockerfile</p>
          </div>
        </div>
        <div v-if="form.build_mode === 'dockerfile'" class="mode-note">
          <Info :size="14" />
          <span>运行环境由 Dockerfile 定义。</span>
        </div>
        <div v-if="form.build_mode === 'jar_image'" class="mode-note">
          <Info :size="14" />
          <span
            ><b>先编译 JAR，再用 Dockerfile 构建镜像。</b>默认 Maven + JDK
            17，在仓库根目录执行 clean package；产物 JAR 路径需与 Dockerfile 的 COPY 一致。</span
          >
        </div>
        <ElFormItem
          v-if="form.build_mode !== 'dockerfile'"
          class="cmd-field"
          :label="form.build_mode === 'jar_image' ? 'JAR 编译命令' : '构建命令'"
          ><ElInput
            v-model="form.build_command"
            :disabled="!editable"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 8 }"
            placeholder="#!/bin/sh&#10;set -eu&#10;&#10;# 拉取代码、测试并构建镜像&#10;go test ./...&#10;docker build -t $IMAGE_REPO:$IMAGE_TAG ."
        /></ElFormItem>
        <ElButton
          v-if="form.build_mode === 'jar_image' && editable && form.template_type !== 'custom'"
          class="restore-btn"
          text
          type="primary"
          size="small"
          @click="applyTemplate(form.template_type)"
          >恢复当前 JAR 模板</ElButton
        >
        <template v-if="form.build_mode !== 'artifact'">
          <ElFormItem label="Dockerfile 来源">
            <ElSegmented
              v-model="form.dockerfile_source"
              :disabled="!editable"
              :options="[
                { label: '代码仓库', value: 'repository' },
                { label: '平台模板 / 编辑', value: 'inline' }
              ]"
            />
          </ElFormItem>
          <div v-if="form.dockerfile_source === 'inline'">
            <ElFormItem label="平台 Dockerfile">
              <div class="df-toolbar">
                <DeliveryTemplateSelect
                  kind="dockerfile"
                  :show-description="false"
                  :disabled="!editable"
                  @select="applyLibraryDockerfile"
                />
                <ElButton v-if="editable" size="small" @click="applyJarDockerfile"
                  >Java 17 示例</ElButton
                >
              </div>
              <ElInput
                v-model="form.dockerfile_content"
                type="textarea"
                :autosize="{ minRows: 3, maxRows: 14 }"
                :disabled="!editable"
                placeholder="选择模板或填写包含 FROM 的完整 Dockerfile"
              />
            <div class="registry-hint"
              >COPY 源路径相对构建上下文。多模块用 mvn -pl 模块名 -am package。</div
            >
            </ElFormItem>
            <ElFormItem label="构建上下文"
              ><ElInput
                v-model="form.build_context"
                :disabled="!editable"
                placeholder="相对仓库根目录，例如 . 或子模块目录"
            /></ElFormItem>
          </div>
        </template>
        <BuildSourcePicker
          v-if="form.build_mode !== 'artifact' && form.dockerfile_source !== 'inline'"
          :key="service?.id"
          v-model:dockerfile-path="form.dockerfile_path"
          v-model:build-context="form.build_context"
          :code-source-id="form.code_source_id"
          :repo-path="form.repo_path"
          :branch="form.repo_branch"
          :disabled="!editable || saving || loading"
          :active="dialogVisible && !loading"
          :offer-maven-module="form.build_mode === 'jar_image' && form.template_type === 'maven'"
          :jar-image="form.build_mode === 'jar_image'"
          @maven-module="applyMavenModule"
        />
        <ElCollapse v-model="buildAdvancedOpen" class="image-advanced">
          <ElCollapseItem name="build" title="高级设置：资源、缓存、分支与变量">
            <ElFormItem label="构建 Agent">
              <ElSegmented v-model="form.provider" :options="agentOptions" :disabled="!editable" />
            </ElFormItem>
            <div class="mode-note">
              <Info :size="14" />
              <span
                >服务配置优先于平台 CI 配置；跟随平台需一并清空下方 CPU 与内存，修改只对新构建生效。</span
              >
            </div>
            <ElAlert
              v-if="
                form.resource_memory.trim() && /^\d+(\.\d+)?$/.test(form.resource_memory.trim())
              "
              title="内存缺少单位，将按字节解释。请改为 2Gi 这样的格式，或清空继承平台。"
              type="warning"
              :closable="false"
            />
            <div class="form-grid thirds">
              <ElFormItem label="构建 CPU"
                ><ElInput
                  v-model="form.resource_cpu"
                  clearable
                  :disabled="!editable"
                  placeholder="留空继承平台，例如 500m 或 1"
              /></ElFormItem>
              <ElFormItem label="构建内存"
                ><ElInput
                  v-model="form.resource_memory"
                  clearable
                  :disabled="!editable"
                  placeholder="留空继承平台，例如 2Gi"
              /></ElFormItem>
              <ElFormItem label="超时（秒）"
                ><ElInputNumber
                  v-model="form.timeout_seconds"
                  :disabled="!editable"
                  :min="0"
                  :max="86400"
                  controls-position="right"
                  style="width: 100%"
              /></ElFormItem>
            </div>
            <div class="binary-row"
              ><span
                ><strong>构建缓存</strong><small>源码使用独立目录；依赖可复用平台缓存</small></span
              ><ElSwitch v-model="form.cache_enabled" :disabled="!editable"
            /></div>
            <ElFormItem v-if="form.cache_enabled" label="缓存策略">
              <ElSelect v-model="form.cache_strategy" :disabled="!editable">
                <ElOption label="继承平台配置" value="inherit" />
                <ElOption label="临时目录（随 Pod 清理）" value="emptydir" />
                <ElOption label="指定 PVC" value="pvc" />
                <ElOption label="远程缓存（需自定义脚本实现）" value="registry" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem v-if="form.cache_enabled && form.cache_strategy === 'pvc'" label="PVC 名称">
              <ElInput v-model="form.cache_pvc_name" :disabled="!editable" />
            </ElFormItem>
            <ElFormItem v-if="form.cache_enabled" label="缓存目录"
              ><ElInput
                v-model="form.cache_dir"
                :disabled="!editable"
                placeholder="工具需配置使用此目录"
            /></ElFormItem>
            <div class="form-grid">
              <ElFormItem label="允许构建的分支"
                ><ElInput
                  v-model="form.includes"
                  :disabled="!editable"
                  placeholder="留空允许全部，例如 main, release/*"
              /></ElFormItem>
              <ElFormItem label="排除分支"
                ><ElInput v-model="form.excludes" :disabled="!editable" placeholder="wip/*"
              /></ElFormItem>
            </div>
            <div class="env-heading"
              ><strong>构建环境变量</strong
              ><ElButton v-if="editable" text type="primary" :icon="Plus" @click="addEnv"
                >添加</ElButton
              ></div
            >
            <div v-for="(item, index) in form.env_vars" :key="index" class="env-row"
              ><ElInput v-model="item.key" :disabled="!editable" placeholder="变量名" /><ElInput
                v-model="item.value"
                :disabled="!editable"
                placeholder="变量值" /><ElButton
                v-if="editable"
                :icon="Trash2"
                circle
                text
                title="删除变量"
                @click="form.env_vars.splice(index, 1)"
            /></div>
          </ElCollapseItem>
        </ElCollapse>
      </ElForm>
    </div>
    <template #footer
      ><ElButton :disabled="saving" @click="dialogVisible = false">关闭</ElButton
      ><ElButton
        v-if="editable"
        type="primary"
        :loading="saving"
        :disabled="loading || !!loadError"
        @click="save"
        >保存配置</ElButton
      ></template
    >
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import { Info, Plus, Trash2 } from 'lucide-vue-next'
  import BuildSourcePicker from './BuildSourcePicker.vue'
  import DeliveryTemplateSelect from '@/components/delivery-template-select/index.vue'
  import { load as loadYaml } from 'js-yaml'
  import type { DeliveryTemplate } from '@/api/axiops'
  import {
    buildSourceError,
    mavenModuleCommand,
    repositoryPath as normalizeBuildContext
  } from '@/utils/build-source'
  import { isBuildConfigMissing, errorMessage } from '@/utils/cicd-workspace'
  import { listBuildRegistriesApi, type BuildRegistry } from '@/api/axiops/build-registry'
  import {
    getBuildConfigApi,
    saveBuildConfigApi,
    listCodeSourcesApi,
    listRepositoriesApi,
    listBranchesApi,
    type AxiOpsBuildConfig,
    type AxiOpsService,
    type AxiOpsCodeSource,
    type AxiOpsRepository,
    type AxiOpsBranch,
    type SaveBuildConfigInput
  } from '@/api/axiops'

  const props = defineProps<{
    visible: boolean
    service: AxiOpsService | null
    editable: boolean
  }>()
  const applyLibraryDockerfile = (template: DeliveryTemplate) => {
    if (!props.editable) return
    const file =
      template.files.find((item) => /(^|\/)Dockerfile$/i.test(item.path)) || template.files[0]
    if (!file?.content?.trim()) return void ElMessage.warning('模板中没有 Dockerfile 内容')
    form.dockerfile_content = file.content
  }
  const applyJarDockerfile = () => {
    form.build_mode = 'jar_image'
    onBuildModeChange('jar_image')
    form.dockerfile_source = 'inline'
    form.dockerfile_content =
      'FROM eclipse-temurin:17-jre\nWORKDIR /app\n# 修改为编译产物的实际唯一 JAR 路径，路径相对构建上下文\nCOPY target/*.jar app.jar\nENV JAVA_OPTS=""\nENTRYPOINT ["sh", "-c", "exec java $JAVA_OPTS -jar app.jar"]\n'
  }
  const applyLibraryBuild = (template: DeliveryTemplate) => {
    if (!props.editable || form.build_mode === 'dockerfile') return
    try {
      const source = template.files.find((file) => file.path === 'build.yaml')
      const config = loadYaml(source?.content || '') as Record<string, unknown>
      if (typeof config?.build_image !== 'string' || typeof config?.build_command !== 'string')
        throw new Error('构建模板缺少编译镜像或命令')
      form.template_type = 'custom'
      form.build_image = config.build_image
      form.build_command = config.build_command
      form.cache_dir = typeof config.cache_dir === 'string' ? config.cache_dir : ''
      ElMessage.success('已应用模板，请检查并保存构建配置')
    } catch (e: any) {
      ElMessage.error(e.message || '构建模板无效')
    }
  }
  const emit = defineEmits<{ 'update:visible': [value: boolean]; saved: [] }>()
  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })
  const loading = ref(false),
    saving = ref(false)
  const loadError = ref(''),
    sourceError = ref(''),
    repoError = ref(''),
    branchError = ref('')
  const buildAdvancedOpen = ref<string[]>([])
  const libraryBuildVisible = ref(false)
  let loadGeneration = 0,
    sourceRequest = 0,
    repoRequest = 0,
    branchRequest = 0,
    registryRequest = 0
  let savedImageRepo = ''
  const codeSources = ref<AxiOpsCodeSource[]>([])
  const repositories = ref<AxiOpsRepository[]>([])
  const branches = ref<AxiOpsBranch[]>([])
  const imageRegistries = ref<BuildRegistry[]>([])
  const registryLoading = ref(false)
  const registryLoadFailed = ref(false)
  const imageNamespace = ref('')
  const imageName = ref('')
  const customImagePath = ref(false)
  const imageAdvancedOpen = ref<string[]>([])
  const selectedRegistry = computed(() =>
    imageRegistries.value.find((item) => item.id === form.registry_id)
  )
  const namespaceOptions = computed(() =>
    selectedRegistry.value ? registryNamespaces(selectedRegistry.value) : []
  )
  const defaultImageName = () => {
    const name =
      props.service?.image_name?.trim().split('/').pop() || props.service?.name?.trim() || ''
    return /^[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*$/.test(name)
      ? name
      : props.service?.code || 'service'
  }
  const hasCustomImage = computed(
    () =>
      customImagePath.value || Boolean(imageName.value && imageName.value !== defaultImageName())
  )
  const imageNameOverride = computed({
    get: () => (imageName.value === defaultImageName() ? '' : imageName.value),
    set: (value: string) => {
      imageName.value = value.trim() || defaultImageName()
      updateGeneratedImage()
    }
  })
  const repoLoading = ref(false)
  const branchLoading = ref(false)
  const form = reactive({
    code_source_id: 0,
    registry_id: 0,
    repo_path: '',
    repo_branch: '',
    template_type: 'custom',
    build_mode: 'dockerfile',
    build_image: '',
    build_command: '',
    dockerfile_path: 'Dockerfile',
    dockerfile_source: 'repository' as 'repository' | 'inline',
    dockerfile_content: '',
    build_context: '.',
    image_repo: '',
    cache_enabled: true,
    cache_dir: '',
    cache_strategy: 'inherit',
    cache_pvc_name: '',
    timeout_seconds: 0,
    includes: '',
    excludes: '',
    env_vars: [] as Array<{ key: string; value: string }>,
    resource_cpu: '',
    resource_memory: '',
    provider: 'kube_job'
  })
  const templateOptions = [
    { label: '自定义', value: 'custom' },
    { label: 'Go', value: 'go' },
    { label: 'Maven', value: 'maven' },
    { label: 'Gradle', value: 'gradle' },
    { label: 'Node.js', value: 'node' },
    { label: 'Python', value: 'python' }
  ]
  const agentOptions = [
    { label: 'Kubernetes Job', value: 'kube_job' },
    { label: 'ECS Agent（未接通）', value: 'ecs_agent', disabled: true }
  ]
  const buildModeOptions = [
    { label: 'Dockerfile / 多阶段', value: 'dockerfile' },
    { label: 'JAR 镜像', value: 'jar_image' },
    { label: '制品文件', value: 'artifact' }
  ]
  const jarTemplateOptions = [
    { label: 'Maven · JDK 17', value: 'maven' },
    { label: 'Gradle · JDK 17', value: 'gradle' },
    { label: '自定义', value: 'custom' }
  ]
  const jarTemplates: Record<string, Partial<typeof form>> = {
    maven: {
      template_type: 'maven',
      build_image: 'maven:3.9-eclipse-temurin-17',
      build_command:
        'mvn -B -ntp -Dmaven.repo.local="${CACHE_DIR:-/workspace/.cache/maven}" clean package',
      cache_dir: '.cache/maven'
    },
    gradle: {
      template_type: 'gradle',
      build_image: 'gradle:8-jdk17',
      build_command:
        'gradle --no-daemon --gradle-user-home "${CACHE_DIR:-/workspace/.cache/gradle}" clean build',
      cache_dir: '.cache/gradle'
    }
  }
  let jarDraft: Partial<typeof form> | null = null
  let lastBuildMode = 'dockerfile'
  const templates: Record<string, Partial<typeof form>> = {
    go: {
      build_image: 'golang:1.24-alpine',
      build_command: 'go test ./... && go build ./...',
      cache_dir: '/go/pkg/mod'
    },
    maven: {
      build_image: 'maven:3.9-eclipse-temurin-21',
      build_command: 'mvn -B clean verify',
      cache_dir: '/root/.m2/repository'
    },
    gradle: {
      build_image: 'gradle:8-jdk21',
      build_command: 'gradle clean build --no-daemon',
      cache_dir: '/home/gradle/.gradle'
    },
    node: {
      build_image: 'node:22-alpine',
      build_command: 'npm ci && npm test && npm run build',
      cache_dir: '/root/.npm'
    },
    python: {
      build_image: 'python:3.13-alpine',
      build_command: 'pip install -r requirements.txt && pytest',
      cache_dir: '/root/.cache/pip'
    }
  }
  const reset = () => {
    buildAdvancedOpen.value = []
    sourceError.value = repoError.value = branchError.value = ''
    repositories.value = []
    branches.value = []
    imageAdvancedOpen.value = []
    imageNamespace.value = ''
    imageName.value = defaultImageName()
    customImagePath.value = false
    jarDraft = null
    lastBuildMode = 'dockerfile'
    Object.assign(form, {
      code_source_id: props.service?.code_source_id || 0,
      registry_id: props.service?.registry_id || 0,
      repo_path: repositoryPath(props.service?.repo_url || ''),
      repo_branch: props.service?.repo_branch || '',
      template_type: 'custom',
      build_mode: 'dockerfile',
      build_image: '',
      build_command: '',
      dockerfile_path: 'Dockerfile',
      dockerfile_source: 'repository',
      dockerfile_content: '',
      build_context: '.',
      image_repo: props.service?.image_name || '',
      cache_enabled: true,
      cache_dir: '',
      cache_strategy: 'inherit',
      cache_pvc_name: '',
      timeout_seconds: 0,
      includes: '',
      excludes: '',
      env_vars: [],
      resource_cpu: '',
      resource_memory: '',
      provider: 'kube_job'
    })
  }

  // 代码源 → 仓库 → 分支 三级联动
  const loadCodeSources = async () => {
    if (!props.service) return
    const generation = loadGeneration,
      request = ++sourceRequest
    sourceError.value = ''
    try {
      const result = await listCodeSourcesApi(props.service.project_id, { silent: true })
      if (generation === loadGeneration && request === sourceRequest)
        codeSources.value = result || []
    } catch (error) {
      if (generation === loadGeneration && request === sourceRequest)
        sourceError.value = errorMessage(error, '代码源加载失败，请检查权限或连接')
    }
  }
  const loadRepositories = async () => {
    const generation = loadGeneration,
      request = ++repoRequest
    repoError.value = ''
    repositories.value = []
    if (!form.code_source_id) {
      repoLoading.value = false
      return
    }
    repoLoading.value = true
    try {
      const result = await listRepositoriesApi(form.code_source_id)
      if (generation === loadGeneration && request === repoRequest)
        repositories.value = result || []
    } catch (error) {
      if (generation === loadGeneration && request === repoRequest)
        repoError.value = errorMessage(error, '仓库加载失败，请检查代码源权限或连接')
    } finally {
      if (generation === loadGeneration && request === repoRequest) repoLoading.value = false
    }
  }
  const loadBranches = async () => {
    const generation = loadGeneration,
      request = ++branchRequest
    branchError.value = ''
    branches.value = []
    if (!form.code_source_id || !form.repo_path) {
      branchLoading.value = false
      return
    }
    branchLoading.value = true
    try {
      const result = await listBranchesApi(form.code_source_id, form.repo_path)
      if (generation === loadGeneration && request === branchRequest) branches.value = result || []
    } catch (error) {
      if (generation === loadGeneration && request === branchRequest)
        branchError.value = errorMessage(error, '分支加载失败；保留已配置分支，请重试')
    } finally {
      if (generation === loadGeneration && request === branchRequest) branchLoading.value = false
    }
  }
  const onCodeSourceChange = () => {
    form.repo_path = ''
    form.repo_branch = ''
    branches.value = []
    ++branchRequest
    branchLoading.value = false
    branchError.value = ''
    void loadRepositories()
  }
  const onRepositoryChange = () => {
    form.repo_branch = ''
    // 自动带出仓库地址到镜像仓库为空时的默认拉取源
    const repo = repositories.value.find((r) => r.full_name === form.repo_path)
    if (repo && !form.repo_branch) {
      form.repo_branch = repo.default_branch || ''
    }
    void loadBranches()
  }
  const parseJSON = <T,>(value: string | undefined, fallback: T): T => {
    if (!value) return fallback
    try {
      return JSON.parse(value) as T
    } catch {
      return fallback
    }
  }
  const applyConfig = (config: AxiOpsBuildConfig) => {
    const rules = parseJSON<{ includes?: string[]; excludes?: string[] }>(config.branch_rules, {})
    Object.assign(form, {
      template_type: config.template_type || 'custom',
      build_mode: config.build_mode || 'dockerfile',
      build_image: config.build_image,
      build_command: config.build_command,
      dockerfile_path: config.dockerfile_path || 'Dockerfile',
      dockerfile_source: config.dockerfile_source || 'repository',
      dockerfile_content: config.dockerfile_content || '',
      build_context: config.build_context || '.',
      image_repo: config.image_repo || '',
      cache_enabled: config.cache_enabled,
      cache_dir: config.cache_dir,
      cache_strategy: config.cache_strategy || 'inherit',
      cache_pvc_name: config.cache_pvc_name || '',
      timeout_seconds: config.timeout_seconds || 0,
      includes: (rules.includes || []).join(', '),
      excludes: (rules.excludes || []).join(', '),
      env_vars: parseJSON<Array<{ key: string; value: string }>>(config.env_vars, []),
      resource_cpu: config.resource_cpu || '',
      resource_memory: config.resource_memory || '',
      provider: config.provider || 'kube_job'
    })
    lastBuildMode = form.build_mode
    if (form.build_mode === 'jar_image') {
      const defaults = jarTemplates[form.template_type] || jarTemplates.maven
      if (!form.build_image.trim()) form.build_image = defaults.build_image!
      if (!form.build_command.trim()) form.build_command = defaults.build_command!
    }
  }
  const loadImageRegistries = async () => {
    if (!props.service) return
    const generation = loadGeneration,
      request = ++registryRequest
    registryLoading.value = true
    registryLoadFailed.value = false
    imageRegistries.value = []
    try {
      const registries = await listBuildRegistriesApi(props.service.project_id)
      if (generation !== loadGeneration || request !== registryRequest) return
      imageRegistries.value = (registries || []).filter((registry) => registry.status === 'active')
      if (!loading.value) {
        applyDefaultRegistry()
        syncImageFields()
      }
    } catch {
      if (generation === loadGeneration && request === registryRequest)
        registryLoadFailed.value = true
    } finally {
      if (generation === loadGeneration && request === registryRequest)
        registryLoading.value = false
    }
  }
  const load = async () => {
    if (!props.service) return
    const generation = ++loadGeneration,
      service = props.service
    reset()
    loadError.value = ''
    loading.value = true
    try {
      const [result] = await Promise.allSettled([
        getBuildConfigApi(service.id),
        loadCodeSources(),
        loadImageRegistries()
      ])
      if (generation !== loadGeneration) return
      if (result.status === 'fulfilled') {
        applyConfig(result.value)
        savedImageRepo = result.value.image_repo || ''
      } else if (isBuildConfigMissing(result.reason)) {
        savedImageRepo = ''
        applyTemplate(service.language.toLowerCase())
      } else {
        loadError.value = `构建配置读取失败：${errorMessage(result.reason, '请检查权限或服务连接')}。重试成功后才能保存。`
        return
      }
      applyDefaultRegistry()
      syncImageFields()
      if (
        form.resource_cpu ||
        form.resource_memory ||
        form.env_vars.length ||
        form.includes ||
        form.excludes ||
        form.cache_strategy !== 'inherit'
      )
        buildAdvancedOpen.value = ['build']
      await Promise.all([loadRepositories(), loadBranches()])
    } finally {
      if (generation === loadGeneration) loading.value = false
    }
  }
  const repositoryPath = (url: string) => {
    try {
      return new URL(url).pathname.replace(/^\/+|\/+$/g, '').replace(/\.git$/, '')
    } catch {
      return url.replace(/^[^@]+@[^:]+:/, '').replace(/\.git$/, '')
    }
  }
  const retrySourceLists = () =>
    Promise.all([loadCodeSources(), loadRepositories(), loadBranches()])
  const applyTemplate = (value: string | number | boolean) => {
    const selected = (form.build_mode === 'jar_image' ? jarTemplates : templates)[String(value)]
    if (selected) Object.assign(form, selected)
  }
  const onBuildModeChange = (value: string | number | boolean) => {
    if (lastBuildMode === 'jar_image') {
      jarDraft = {
        template_type: form.template_type,
        build_image: form.build_image,
        build_command: form.build_command,
        cache_dir: form.cache_dir
      }
    }
    if (value === 'jar_image' && lastBuildMode !== 'jar_image') {
      Object.assign(form, jarDraft || jarTemplates.maven)
    }
    lastBuildMode = String(value)
    applyDefaultRegistry()
    syncImageFields()
  }
  const registryPrefix = (registry: BuildRegistry) =>
    registry.url.replace(/^https?:\/\//, '').replace(/\/+$/, '')
  const registryHost = (registry: BuildRegistry) => registryPrefix(registry).split('/')[0]
  const registryNamespaces = (registry: BuildRegistry) => {
    const path = registryPrefix(registry).split('/').slice(1).join('/')
    return [...new Set([path, ...(registry.namespaces || [])].filter(Boolean))]
  }
  const updateGeneratedImage = () => {
    if (!selectedRegistry.value || customImagePath.value) return
    form.image_repo = [
      registryHost(selectedRegistry.value),
      imageNamespace.value.trim(),
      imageName.value.trim()
    ]
      .filter(Boolean)
      .join('/')
  }
  const syncImageFields = () => {
    const registry = selectedRegistry.value
    if (!registry || form.build_mode === 'artifact') return
    const prefix = `${registryHost(registry)}/`
    const choices = registryNamespaces(registry)
    if (!form.image_repo) {
      imageNamespace.value = choices[0] || ''
      imageName.value = defaultImageName()
      customImagePath.value = false
      updateGeneratedImage()
      return
    }
    if (form.image_repo.startsWith(prefix)) {
      const relative = form.image_repo.slice(prefix.length)
      const namespace = [...choices]
        .sort((a, b) => b.length - a.length)
        .find((name) => relative.startsWith(`${name}/`))
      if (namespace || choices.length === 0) {
        imageNamespace.value = namespace || relative.split('/').slice(0, -1).join('/')
        imageName.value = relative.slice(imageNamespace.value ? imageNamespace.value.length + 1 : 0)
        customImagePath.value = false
        return
      }
    }
    customImagePath.value = true // Preserve previously saved custom references for review.
  }
  const onRegistryChange = () => {
    const registry = selectedRegistry.value
    if (!registry) return
    syncImageFields()
    if (customImagePath.value) {
      imageNamespace.value = registryNamespaces(registry)[0] || ''
      imageName.value ||= defaultImageName()
      customImagePath.value = false
      updateGeneratedImage()
    }
  }
  const toggleImagePathMode = () => {
    if (!customImagePath.value) {
      customImagePath.value = true
      return
    }
    onRegistryChange()
  }
  const restoreServiceImageName = () => {
    syncImageFields()
    imageName.value = defaultImageName()
    if (selectedRegistry.value) {
      const choices = registryNamespaces(selectedRegistry.value)
      if (choices.length && !choices.includes(imageNamespace.value)) {
        imageNamespace.value = choices[0]
      }
    }
    customImagePath.value = false
    updateGeneratedImage()
  }
  const applyDefaultRegistry = () => {
    if (form.registry_id || form.build_mode === 'artifact') return
    const registry = imageRegistries.value.find((item) => item.name === '阿里云-美国')
    if (!registry) return
    form.registry_id = registry.id
    onRegistryChange()
  }
  const addEnv = () => form.env_vars.push({ key: '', value: '' })
  const applyMavenModule = (module: string) => {
    if (!props.editable || form.build_mode !== 'jar_image' || form.template_type !== 'maven') return
    const command = mavenModuleCommand(module)
    if (command) form.build_command = command
  }
  const splitPatterns = (value: string) =>
    value
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
  const save = async () => {
    if (!props.service || !props.editable || saving.value || loading.value || loadError.value)
      return
    if (
      form.resource_memory.trim() &&
      !/^(?:\d+(?:\.\d+)?|\.\d+)(?:Ki|Mi|Gi|Ti|Pi|Ei|k|M|G|T|P|E)$/.test(
        form.resource_memory.trim()
      )
    )
      return void ElMessage.warning(
        '构建内存请填写带单位的值，例如 2Gi 或 2048Mi；单填 2 会被 Kubernetes 解释为 2 字节'
      )
    if (form.timeout_seconds > 0 && form.timeout_seconds < 60)
      return void ElMessage.warning('超时至少 60 秒，或填 0 继承平台配置')
    if (form.cache_enabled && form.cache_strategy === 'pvc' && !form.cache_pvc_name.trim())
      return void ElMessage.warning('请填写 CI 命名空间中预创建的 PVC 名称')
    if (form.build_mode !== 'artifact' && !form.image_repo.trim())
      return void ElMessage.warning('请填写产物镜像路径')
    const registry = imageRegistries.value.find((item) => item.id === form.registry_id)
    if (form.build_mode !== 'artifact') {
      if (registryLoading.value) return void ElMessage.warning('镜像仓库列表正在加载，请稍后保存')
      if (registryLoadFailed.value)
        return void ElMessage.warning('镜像仓库列表加载失败，请点击「重新加载」后再保存')
      if (!imageRegistries.value.length)
        return void ElMessage.warning(
          '当前项目没有可用的镜像仓库，请在系统管理 → 构建资源 → 镜像仓库中添加或启用仓库'
        )
      if (!form.registry_id) return void ElMessage.warning('请选择镜像仓库')
      if (!registry)
        return void ElMessage.warning('原绑定的镜像仓库已停用或对当前项目不可用，请重新选择')
      const prefix = `${registryHost(registry)}/`
      const relative = form.image_repo.trim().slice(prefix.length)
      if (!form.image_repo.trim().startsWith(prefix))
        return void ElMessage.warning('产物镜像路径必须属于所选镜像仓库')
      if (
        !/^[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*(?:\/[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*)*$/.test(
          relative
        )
      )
        return void ElMessage.warning(
          '镜像名称和命名空间需使用小写字母、数字及合法分隔符，路径不含 tag'
        )
      const namespaces = registryNamespaces(registry)
      if (namespaces.length && !namespaces.some((name) => relative.startsWith(`${name}/`)))
        return void ElMessage.warning('请选择该镜像仓库已配置的命名空间')
    }
    // Dockerfile 构建必须绑定真实仓库和分支，否则构建器拉不到代码。
    if (form.build_mode !== 'artifact') {
      if (!form.code_source_id) return void ElMessage.warning('请选择代码源')
      if (!form.repo_path) return void ElMessage.warning('请选择代码仓库')
      if (!form.repo_branch) return void ElMessage.warning('请选择构建分支')
    }
    if (form.build_mode === 'jar_image' && (!form.build_image.trim() || !form.build_command.trim()))
      return void ElMessage.warning('请选择 Maven / Gradle 模板，或填写 JAR 编译镜像和编译命令')
    if (form.build_mode !== 'artifact') {
      const pathError = buildSourceError(form.build_context, form.dockerfile_path)
      if (pathError) return void ElMessage.warning(pathError)
      if (form.dockerfile_source === 'inline') {
        if (
          !/^\s*FROM\s+\S+/im.test(form.dockerfile_content) ||
          new Blob([form.dockerfile_content]).size > 65536
        )
          return void ElMessage.warning('平台 Dockerfile 必须包含 FROM，且不超过 64 KiB')
        if (form.provider !== 'kube_job')
          return void ElMessage.warning('平台 Dockerfile 目前使用 Kubernetes 构建 Agent')
      }
    }
    const envVars = form.env_vars
      .map((item) => ({ key: item.key.trim(), value: item.value }))
      .filter((item) => item.key)
    if (new Set(envVars.map((item) => item.key)).size !== envVars.length)
      return void ElMessage.warning('环境变量名不能重复')
    const data: SaveBuildConfigInput = {
      template_type: form.build_mode === 'dockerfile' ? 'custom' : form.template_type,
      build_mode: form.build_mode,
      build_image: form.build_mode === 'dockerfile' ? '' : form.build_image.trim(),
      build_command: form.build_mode === 'dockerfile' ? '' : form.build_command,
      dockerfile_path: form.dockerfile_path.trim() || 'Dockerfile',
      dockerfile_source: form.build_mode === 'artifact' ? 'repository' : form.dockerfile_source,
      dockerfile_content:
        form.build_mode !== 'artifact' && form.dockerfile_source === 'inline'
          ? form.dockerfile_content
          : '',
      build_context:
        form.dockerfile_source === 'inline'
          ? normalizeBuildContext(form.build_context) || '.'
          : form.build_context.trim() || '.',
      image_repo: form.image_repo.trim(),
      cache_enabled: form.cache_enabled,
      cache_dir: form.cache_dir.trim(),
      cache_strategy: form.cache_strategy,
      cache_pvc_name: form.cache_pvc_name.trim(),
      timeout_seconds: form.timeout_seconds,
      branch_rules: {
        includes: splitPatterns(form.includes),
        excludes: splitPatterns(form.excludes)
      },
      env_vars: envVars,
      resource_cpu: form.resource_cpu.trim(),
      resource_memory: form.resource_memory.trim(),
      provider: form.provider
    }
    const binding: NonNullable<SaveBuildConfigInput['service_binding']> = {}
    if (form.code_source_id && form.repo_path) {
      const repo = repositories.value.find((item) => item.full_name === form.repo_path)
      const unchangedRepo =
        form.code_source_id === props.service.code_source_id &&
        form.repo_path === repositoryPath(props.service.repo_url || '')
      const repoUrl = unchangedRepo ? props.service.repo_url : repo?.clone_url
      if (!repoUrl)
        return void ElMessage.warning(
          '请等待仓库列表加载成功后选择仓库，不能用仓库简称作为克隆地址'
        )
      if (!unchangedRepo || form.repo_branch !== props.service.repo_branch)
        Object.assign(binding, {
          code_source_id: form.code_source_id,
          repo_url: repoUrl,
          repo_branch: form.repo_branch
        })
    }
    if (
      registry &&
      form.build_mode !== 'artifact' &&
      (form.registry_id !== props.service.registry_id || form.image_repo.trim() !== savedImageRepo)
    )
      binding.registry_id = registry.id
    if (Object.keys(binding).length) data.service_binding = binding
    saving.value = true
    try {
      await saveBuildConfigApi(props.service.id, data)
      ElMessage.success('构建配置已保存')
      emit('saved')
      dialogVisible.value = false
    } finally {
      saving.value = false
    }
  }
  watch(
    () => [props.visible, props.service?.id],
    ([visible]) => {
      if (visible) void load()
      else ++loadGeneration
    }
  )
</script>

<style scoped lang="scss">
  :global(.build-config-dialog) {
    max-width: calc(100vw - 32px);
  }
  :global(.build-config-dialog .el-dialog__body) {
    max-height: calc(94vh - 124px);
    overflow-y: auto;
  }
  .step-head {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    margin: 14px 0 10px;
    padding-top: 12px;
    border-top: 1px solid var(--el-border-color-lighter);

    &.is-first {
      margin-top: 2px;
      padding-top: 0;
      border-top: none;
    }
  }

  .step-no {
    flex: 0 0 auto;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    margin-top: 1px;
    border-radius: 50%;
    background: var(--el-color-primary);
    color: #fff;
    font-size: 12px;
    font-weight: 600;
    line-height: 1;
  }

  .step-text {
    min-width: 0;

    h3 {
      margin: 0;
      color: var(--el-text-color-primary);
      font-size: 15px;
      font-weight: 600;
      line-height: 22px;
    }

    p {
      margin: 2px 0 0;
      color: var(--el-text-color-secondary);
      font-size: 13px;
      line-height: 1.5;
    }
  }
  .mode-note {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    margin: 8px 0;
    padding: 6px 12px;
    border: 1px solid color-mix(in srgb, var(--el-color-primary) 18%, transparent);
    border-radius: var(--ax-radius-sm);
    background: color-mix(in srgb, var(--el-color-primary) 6%, transparent);
    color: var(--el-text-color-regular);
    font-size: 13px;
    line-height: 1.6;

    svg {
      flex: 0 0 auto;
      margin-top: 3px;
      color: var(--el-color-primary);
    }

    b {
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }
  .load-error {
    color: var(--el-color-danger);
    font-size: 13px;
  }
  .repo-cascade {
    display: flex;
    gap: 10px;
    align-items: center;
  }
  .repo-cascade .el-select {
    flex: 1 1 0;
    min-width: 0;
  }
  .form-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
  }
  .form-row > .el-form-item {
    margin-bottom: 0;
  }
  .image-result {
    padding: 8px 12px;
    margin-bottom: 8px;
    background: var(--ax-surface-sunken);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--ax-radius-md);
  }
  .image-result-body {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }
  .image-result code {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-family: var(--ax-font-mono);
    font-size: 13px;
    text-overflow: ellipsis;
    white-space: nowrap;
    line-height: 1.5;
  }
  .image-result .registry-hint {
    margin-top: 2px;
  }
  .cmd-field {
    margin-bottom: 4px;
  }
  .restore-btn {
    margin-top: 0;
    margin-bottom: 10px;
    padding: 0;
    height: auto;
  }
  .image-advanced {
    margin-bottom: 12px;
    border-top: none;
  }
  .registry-hint {
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.5;
  }
  .df-toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 6px;
  }
  .df-toolbar :deep(.template-select) {
    flex: 1;
    min-width: 0;
  }
  .lib-inline {
    margin-top: 6px;
    border-top: none;
  }
  .template-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .template-row .el-segmented {
    flex: 1;
    min-width: 0;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  .form-grid.thirds {
    grid-template-columns: repeat(3, 1fr);
  }
  .binary-row {
    min-height: 58px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--art-border-color);
    margin-bottom: 12px;
  }
  .binary-row span {
    display: flex;
    flex-direction: column;
  }
  .binary-row small {
    margin-top: 3px;
    color: var(--art-text-gray-500);
  }
  .env-heading {
    min-height: 42px;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .env-row {
    display: grid;
    grid-template-columns: 1fr 1fr 34px;
    gap: 8px;
    margin-bottom: 8px;
  }
  @media (max-width: 680px) {
    .form-grid,
    .form-grid.thirds {
      grid-template-columns: 1fr;
    }
  }
</style>

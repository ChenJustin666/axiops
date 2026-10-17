<template>
  <section class="build-source">
    <ElFormItem label="Dockerfile 路径">
      <div class="source-row">
        <code>{{ selectedFile || '请选择 Dockerfile' }}</code>
        <ElButton :disabled="disabled || !canBrowse" @click="openBrowser"> 从仓库选择 </ElButton>
      </div>
      <div v-if="!selectedFile" class="hint">从当前分支选择 Dockerfile。</div>
    </ElFormItem>
    <div v-if="browserOpen" class="browser">
      <div class="source-row">
        <strong>{{ directory === '.' ? '仓库根目录' : directory }}</strong>
        <ElButton :disabled="busy || disabled" text @click="loadDirectory('.')">根目录</ElButton>
        <ElButton
          v-if="directory !== '.'"
          :disabled="busy || disabled"
          text
          @click="loadDirectory(parentDirectory(directory))"
          >上一级</ElButton
        >
        <ElButton text @click="browserOpen = false">收起</ElButton>
      </div>
      <div v-if="browseError" class="error" role="alert">
        {{ browseError }}
        <ElButton text type="primary" :disabled="disabled" @click="loadDirectory(directory)">
          重试
        </ElButton>
      </div>
      <div v-loading="busy" class="entries">
        <div v-for="entry in visibleEntries" :key="entry.path" class="entry">
          <span>{{ entry.type === 'dir' ? '目录 · ' : '' }}{{ entry.name }}</span>
          <ElButton
            text
            type="primary"
            :disabled="busy || disabled"
            @click="entry.type === 'dir' ? loadDirectory(entry.path) : selectFile(entry)"
            >{{ entry.type === 'dir' ? '进入' : '选择' }}</ElButton
          >
        </div>
        <div v-if="!busy && !browseError && !visibleEntries.length" class="hint">
          当前目录没有 Dockerfile 或子目录。
        </div>
      </div>
    </div>
    <ElFormItem label="构建上下文">
      <ElSelect :model-value="contextMode" :disabled="disabled" @change="changeContext">
        <ElOption
          :label="jarImage ? '整个仓库（相对仓库根目录）' : '整个仓库'"
          value="root"
        />
        <ElOption
          v-if="selectedDirectory !== '.'"
          :label="jarImage ? '仅 Dockerfile 所在目录' : '仅 Dockerfile 所在目录（独立模块）'"
          value="module"
        />
        <ElOption label="自定义目录" value="custom" />
      </ElSelect>
    </ElFormItem>
    <div class="preview">
      <div>Dockerfile：<code>{{ selectedFile || '待选择' }}</code> · 源码范围：<code>{{ buildContext || '.' }}</code></div>
      <div v-if="mavenModule" class="hint maven-hint">
        检测到 pom.xml。
        <ElButton :disabled="disabled" text type="primary" @click="emit('maven-module', mavenModule)">
          使用模块 Maven 命令
        </ElButton>
      </div>
    </div>
    <ElCollapse v-model="advancedOpen">
      <ElCollapseItem name="paths" title="高级路径设置">
        <ElFormItem label="构建上下文（相对于仓库根目录）">
          <ElInput
            :model-value="buildContext"
            :disabled="disabled"
            @update:model-value="emit('update:buildContext', $event)"
          />
        </ElFormItem>
        <ElFormItem label="Dockerfile 路径（相对于上述构建上下文）">
          <ElInput
            :model-value="dockerfilePath"
            :disabled="disabled"
            @update:model-value="emit('update:dockerfilePath', $event)"
          />
        </ElFormItem>
      </ElCollapseItem>
    </ElCollapse>
    <div v-if="pathError" class="error" role="alert">{{ pathError }}</div>
  </section>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import { listRepositoryFilesApi, type AxiOpsRepositoryFile } from '@/api/axiops'
  import { errorMessage } from '@/utils/cicd-workspace'
  import { buildSourceError, dockerfileInRepository, parentDirectory } from '@/utils/build-source'

  const props = defineProps<{
    codeSourceId: number
    repoPath: string
    branch: string
    dockerfilePath: string
    buildContext: string
    disabled: boolean
    active: boolean
    offerMavenModule: boolean
    jarImage: boolean
  }>()
  const emit = defineEmits<{
    'update:dockerfilePath': [value: string]
    'update:buildContext': [value: string]
    'maven-module': [value: string]
  }>()
  const browserOpen = ref(false)
  const directory = ref('.')
  const entries = ref<AxiOpsRepositoryFile[]>([])
  const busy = ref(false)
  const browseError = ref('')
  const advancedOpen = ref<string[]>([])
  const verifiedPomDirectory = ref('')
  const customContextOpen = ref(false)
  let requestVersion = 0
  const canBrowse = computed(
    () => props.active && props.codeSourceId > 0 && !!props.repoPath && !!props.branch
  )
  const selectedFile = computed(() =>
    dockerfileInRepository(props.buildContext, props.dockerfilePath)
  )
  const selectedDirectory = computed(() => parentDirectory(selectedFile.value))
  const pathError = computed(() => buildSourceError(props.buildContext, props.dockerfilePath))
  const contextMode = computed(() => {
    if (customContextOpen.value) return 'custom'
    if (!props.buildContext || props.buildContext === '.') return 'root'
    if (props.buildContext === selectedDirectory.value) return 'module'
    return 'custom'
  })
  const visibleEntries = computed(() =>
    entries.value
      .filter(
        (entry) => entry.type === 'dir' || /(^dockerfile(?:\..+)?$|\.dockerfile$)/i.test(entry.name)
      )
      .sort(
        (a, b) =>
          Number(b.type === 'dir') - Number(a.type === 'dir') || a.name.localeCompare(b.name)
      )
  )
  const mavenModule = computed(() =>
    props.offerMavenModule &&
    verifiedPomDirectory.value === selectedDirectory.value &&
    selectedDirectory.value !== '.'
      ? selectedDirectory.value
      : ''
  )

  async function loadDirectory(path: string) {
    const version = ++requestVersion
    directory.value = path
    entries.value = []
    browseError.value = ''
    if (!canBrowse.value) {
      busy.value = false
      return
    }
    busy.value = true
    try {
      const result = await listRepositoryFilesApi(props.codeSourceId, {
        repo_path: props.repoPath,
        ref: props.branch,
        path: path === '.' ? '' : path
      })
      if (version !== requestVersion) return
      entries.value = result || []
      if (path === selectedDirectory.value)
        verifiedPomDirectory.value = entries.value.some(
          (entry) => entry.name === 'pom.xml' && entry.type === 'file'
        )
          ? path
          : ''
    } catch (error) {
      if (version === requestVersion)
        browseError.value = errorMessage(error, '仓库目录读取失败，请重试或检查代码源权限')
    } finally {
      if (version === requestVersion) busy.value = false
    }
  }
  function openBrowser() {
    browserOpen.value = true
    void loadDirectory('.')
  }
  function selectFile(entry: AxiOpsRepositoryFile) {
    if (props.disabled || busy.value || entry.type !== 'file') return
    verifiedPomDirectory.value = entries.value.some(
      (file) => file.name === 'pom.xml' && file.type === 'file'
    )
      ? directory.value
      : ''
    customContextOpen.value = false
    // A fresh selection includes parent and sibling sources by default.
    emit('update:buildContext', '.')
    emit('update:dockerfilePath', entry.path)
    browserOpen.value = false
  }
  function changeContext(mode: string) {
    if (mode === 'custom') {
      customContextOpen.value = true
      advancedOpen.value = ['paths']
      return
    }
    customContextOpen.value = false
    const file = selectedFile.value
    if (!file) return
    const context = mode === 'module' ? parentDirectory(file) : '.'
    emit('update:buildContext', context)
    emit('update:dockerfilePath', context === '.' ? file : file.slice(context.length + 1))
  }
  watch(
    () => [props.active, props.codeSourceId, props.repoPath, props.branch],
    () => {
      ++requestVersion
      entries.value = []
      busy.value = false
      browseError.value = ''
      verifiedPomDirectory.value = ''
      browserOpen.value = false
    }
  )
</script>

<style scoped lang="scss">
  .build-source {
    margin: 16px 0;
  }
  .source-row {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
  }
  .source-row code,
  .source-row strong {
    flex: 1;
    overflow-wrap: anywhere;
  }
  .hint {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }
  .preview,
  .browser {
    padding: 10px 12px;
    margin-bottom: 10px;
    background: var(--el-fill-color-light);
    border-radius: var(--ax-radius-sm);
  }
  .preview {
    font-size: 13px;
    color: var(--el-text-color-regular);
  }
  .preview code {
    overflow-wrap: anywhere;
  }
  .maven-hint {
    margin-top: 8px;
  }
  .entries {
    max-height: 230px;
    min-height: 40px;
    overflow: auto;
  }
  .entry {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }
  .entry span {
    overflow-wrap: anywhere;
  }
  .error {
    color: var(--el-color-danger);
    font-size: 13px;
  }
</style>

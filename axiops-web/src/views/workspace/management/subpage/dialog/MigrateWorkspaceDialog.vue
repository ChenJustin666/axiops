<!-- /views/workspace/management/subpage/dialog/MigrateWorkspaceDialog.vue -->
<template>
  <ElDialog
    v-model="visible"
    title="迁移工作空间"
    width="600px"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <ElAlert
      title="迁移说明"
      type="warning"
      :closable="false"
      show-icon
      style="margin-bottom: 20px"
    >
      <template #default>
        <ul style="margin: 0; padding-left: 20px">
          <li>工作空间将从当前项目迁移到目标项目</li>
          <li>迁移过程中会更新所有相关资源配额</li>
          <li>请确保目标项目有足够的资源配额</li>
          <li>迁移操作不可撤销，请谨慎操作</li>
        </ul>
      </template>
    </ElAlert>

    <ElForm ref="formRef" :model="formData" :rules="rules" label-width="120px">
      <ElFormItem label="当前工作空间">
        <ElInput :value="workspace?.name" disabled>
          <template #prefix>
            <ElIcon><FolderOpened /></ElIcon>
          </template>
        </ElInput>
      </ElFormItem>

      <ElFormItem label="当前项目">
        <ElInput :value="currentProject?.name" disabled>
          <template #prefix>
            <ElIcon><Box /></ElIcon>
          </template>
        </ElInput>
      </ElFormItem>

      <ElFormItem label="目标项目" prop="targetProjectId" required>
        <ElSelect
          v-model="formData.targetProjectId"
          placeholder="请选择目标项目"
          filterable
          style="width: 100%"
          :loading="loadingProjects"
        >
          <ElOption
            v-for="project in availableProjects"
            :key="project.id"
            :label="project.name"
            :value="project.id"
            :disabled="project.id === currentProject?.id"
          >
            <div class="project-option">
              <span class="project-name">{{ project.name }}</span>
              <ElTag size="small" type="info">{{ project.uuid }}</ElTag>
            </div>
          </ElOption>
        </ElSelect>
        <div class="form-tip">
          <ElIcon><InfoFilled /></ElIcon>
          选择工作空间要迁移到的目标项目
        </div>
      </ElFormItem>

      <ElFormItem label="资源配额检查" v-if="formData.targetProjectId">
        <div class="quota-check">
          <div class="check-item" v-if="targetProjectCluster">
            <ElIcon class="icon success"><CircleCheck /></ElIcon>
            <span>目标项目集群配置正常</span>
          </div>
          <div class="check-item" v-else>
            <ElIcon class="icon warning"><Warning /></ElIcon>
            <span>目标项目暂无集群配置，请先配置集群</span>
          </div>
        </div>
      </ElFormItem>
    </ElForm>

    <template #footer>
      <ElButton @click="handleClose" :disabled="submitting">取消</ElButton>
      <ElButton type="primary" @click="handleSubmit" :loading="submitting" :disabled="!canMigrate">
        <ElIcon><Right /></ElIcon>
        确认迁移
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    FolderOpened,
    Box,
    InfoFilled,
    CircleCheck,
    Warning,
    Right
  } from '@element-plus/icons-vue'
  import {
    migrateWorkspaceOnProjectApi,
    getProjectsByUserApi,
    searchProjectClusterApi,
    type ProjectWorkspace,
    type Project,
    type ProjectCluster
  } from '@/api'

  interface Props {
    modelValue: boolean
    workspace: ProjectWorkspace | null
    currentProject: Project | null
  }

  interface Emits {
    (e: 'update:modelValue', value: boolean): void
    (e: 'success'): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const formRef = ref()
  const submitting = ref(false)
  const loadingProjects = ref(false)
  const availableProjects = ref<Project[]>([])
  const targetProjectCluster = ref<ProjectCluster | null>(null)

  const formData = ref({
    targetProjectId: null as number | null
  })

  const rules = {
    targetProjectId: [{ required: true, message: '请选择目标项目', trigger: 'change' }]
  }

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const canMigrate = computed(() => {
    return formData.value.targetProjectId && targetProjectCluster.value !== null
  })

  // 加载可用项目列表
  const loadProjects = async () => {
    try {
      loadingProjects.value = true
      const response = await getProjectsByUserApi({})
      availableProjects.value = response || []
    } finally {
      loadingProjects.value = false
    }
  }

  // 检查目标项目集群配置
  const checkTargetProjectCluster = async (projectId: number) => {
    try {
      const response = await searchProjectClusterApi({ projectId })
      targetProjectCluster.value = response && response.length > 0 ? response[0] : null
    } catch (error) {
      targetProjectCluster.value = null
    }
  }

  // 监听目标项目变化
  watch(
    () => formData.value.targetProjectId,
    (newVal) => {
      if (newVal) {
        checkTargetProjectCluster(newVal)
      } else {
        targetProjectCluster.value = null
      }
    }
  )

  // 监听对话框打开
  watch(
    () => props.modelValue,
    (newVal) => {
      if (newVal) {
        formData.value.targetProjectId = null
        targetProjectCluster.value = null
        loadProjects()
      }
    }
  )

  const handleSubmit = async () => {
    try {
      await formRef.value?.validate()

      if (!props.workspace) {
        return
      }

      if (!canMigrate.value) {
        return
      }

      await ElMessageBox.confirm(
        `确定要将工作空间 "${props.workspace.name}" 迁移到选定的项目吗？此操作不可撤销。`,
        '迁移确认',
        {
          confirmButtonText: '确定迁移',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      submitting.value = true

      await migrateWorkspaceOnProjectApi({
        workloadId: props.workspace.id,
        newProjectId: formData.value.targetProjectId!
      })

      ElMessage.success('工作空间迁移成功')

      submitting.value = false

      // 触发成功事件
      emit('success')

      // 关闭对话框
      visible.value = false
    } catch (error: any) {
      if (error !== 'cancel') {
        submitting.value = false
      }
    }
  }

  const handleClose = () => {
    if (!submitting.value) {
      visible.value = false
    }
  }
</script>

<style lang="scss" scoped>
  .project-option {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;

    .project-name {
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .form-tip {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 6px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .quota-check {
    width: 100%;

    .check-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 10px;
      background: var(--el-fill-color-lighter);
      border-radius: 6px;
      font-size: 13px;

      .icon {
        font-size: 18px;

        &.success {
          color: var(--el-color-success);
        }

        &.warning {
          color: var(--el-color-warning);
        }
      }
    }
  }
</style>

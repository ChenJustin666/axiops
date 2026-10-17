<template>
  <ElCard class="search-card" shadow="never">
    <ElForm :model="formData" :inline="true" class="search-form">
      <ElFormItem label="开始时间">
        <ElDatePicker
          v-model="formData.startTime"
          type="date"
          placeholder="选择开始日期"
          format="YYYY/MM/DD"
          value-format="x"
          :clearable="true"
          style="width: 160px"
        />
      </ElFormItem>

      <ElFormItem label="结束时间">
        <ElDatePicker
          v-model="formData.endTime"
          type="date"
          placeholder="选择结束日期"
          format="YYYY/MM/DD"
          value-format="x"
          :clearable="true"
          style="width: 160px"
        />
      </ElFormItem>

      <ElFormItem label="集群">
        <ElSelect
          v-model="formData.clusterUuid"
          placeholder="全部集群"
          clearable
          filterable
          style="width: 160px"
        >
          <ElOption
            v-for="cluster in clusterList"
            :key="cluster.uuid"
            :label="cluster.name"
            :value="cluster.uuid"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem label="项目">
        <ElSelect
          v-model="formData.projectId"
          placeholder="全部项目"
          clearable
          filterable
          style="width: 160px"
        >
          <ElOption
            v-for="project in projectList"
            :key="project.id"
            :label="project.name"
            :value="project.id"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem label="账单类型">
        <ElSelect
          v-model="formData.statementType"
          placeholder="全部类型"
          clearable
          style="width: 140px"
        >
          <ElOption label="日常账单" value="daily" />
          <ElOption label="初始账单" value="initial" />
          <ElOption label="配置变更" value="config_change" />
        </ElSelect>
      </ElFormItem>

      <ElFormItem>
        <ElButton type="primary" @click="handleSearch">
          <ElIcon class="mr-1"><Search /></ElIcon>
          查询
        </ElButton>
        <ElButton @click="handleReset">
          <ElIcon class="mr-1"><Refresh /></ElIcon>
        </ElButton>
      </ElFormItem>
    </ElForm>
  </ElCard>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, onMounted } from 'vue'
  import { Search, Refresh } from '@element-plus/icons-vue'
  import type { BillingStatementSearchRequest } from '@/api/manager/billing'

  interface ClusterItem {
    uuid: string
    name: string
  }

  interface ProjectItem {
    id: number
    name: string
  }

  interface Props {
    modelValue: Record<string, any>
  }

  interface Emits {
    (e: 'update:modelValue', value: Record<string, any>): void
    (e: 'search', params: BillingStatementSearchRequest): void
    (e: 'reset'): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  // 表单数据
  const formData = reactive({
    startTime: undefined as number | undefined,
    endTime: undefined as number | undefined,
    clusterUuid: undefined as string | undefined,
    projectId: undefined as number | undefined,
    statementType: undefined as string | undefined
  })

  // 集群列表
  const clusterList = ref<ClusterItem[]>([])

  // 项目列表
  const projectList = ref<ProjectItem[]>([])

  // 加载集群列表
  const loadClusterList = async () => {
    try {
      // 调用集群列表接口
      const { searchClusterApi } = await import('@/api/manager/cluster')
      const response = await searchClusterApi({ page: 1, pageSize: 200 })
      clusterList.value = (response.items || []).map((item: any) => ({
        uuid: item.uuid,
        name: item.name
      }))
    } catch (error) {
      console.error('加载集群列表失败:', error)
    }
  }

  // 加载项目列表
  const loadProjectList = async () => {
    try {
      // 调用项目列表接口
      const { searchProjectApi } = await import('@/api/manager/project')
      const response = await searchProjectApi({ page: 1, pageSize: 1000 })
      projectList.value = (response.items || []).map((item: any) => ({
        id: item.id,
        name: item.name
      }))
    } catch (error) {
      console.error('加载项目列表失败:', error)
    }
  }

  // 搜索
  const handleSearch = () => {
    const params: BillingStatementSearchRequest = {}

    if (formData.startTime) {
      // 转换为秒级时间戳
      params.startTime = Math.floor(formData.startTime / 1000)
    }
    if (formData.endTime) {
      // 结束时间设为当天23:59:59
      const endDate = new Date(formData.endTime)
      endDate.setHours(23, 59, 59, 999)
      params.endTime = Math.floor(endDate.getTime() / 1000)
    }
    if (formData.clusterUuid) {
      params.clusterUuid = formData.clusterUuid
    }
    if (formData.projectId) {
      params.projectId = formData.projectId
    }
    if (formData.statementType) {
      params.statementType = formData.statementType
    }

    emit('search', params)
  }

  // 重置
  const handleReset = () => {
    formData.startTime = undefined
    formData.endTime = undefined
    formData.clusterUuid = undefined
    formData.projectId = undefined
    formData.statementType = undefined
    emit('reset')
  }

  // 初始化
  onMounted(() => {
    loadClusterList()
    loadProjectList()
  })
</script>

<style lang="scss" scoped>
  .search-card {
    :deep(.el-card__body) {
      padding: 16px 20px 0;
    }
  }

  .search-form {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0;

    :deep(.el-form-item) {
      margin-bottom: 16px;
      margin-right: 16px;
    }

    :deep(.el-form-item__label) {
      color: #606266;
    }
  }

  .mr-1 {
    margin-right: 4px;
  }
</style>

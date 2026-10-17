<template>
  <ArtSearchBar
    ref="searchBarRef"
    v-model="formData"
    :items="formItems"
    :rules="rules"
    @reset="handleReset"
    @search="handleSearch"
  />
</template>

<script setup lang="ts">
  import { ref, computed, onMounted, watch } from 'vue'
  import { searchClusterApi } from '@/api/manager/cluster'
  import { searchProjectApi, searchProjectWorkspaceApi } from '@/api/manager/project'

  interface Props {
    modelValue: Record<string, any>
  }

  interface Emits {
    (e: 'update:modelValue', value: Record<string, any>): void
    (e: 'search', params: Record<string, any>): void
    (e: 'reset'): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  // 表单数据双向绑定
  const searchBarRef = ref()
  const formData = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  // 校验规则
  const rules = {}

  // 集群选项
  const clusterOptions = ref<Array<{ label: string; value: string }>>([])
  const clusterLoading = ref(false)

  // 项目选项
  const projectOptions = ref<Array<{ label: string; value: number; raw?: any }>>([
    { label: '全部', value: -1 },
    { label: '非项目', value: 0 }
  ])
  const projectLoading = ref(false)

  // 工作空间选项
  const workspaceOptions = ref<Array<{ label: string; value: number }>>([
    { label: '全部', value: -1 },
    { label: '非工作空间', value: 0 }
  ])
  const workspaceLoading = ref(false)

  // 选中的项目
  const selectedProject = ref<any>(null)

  // 操作状态选项
  const statusOptions = [
    { label: '全部', value: -1 },
    { label: '成功', value: 1 },
    { label: '失败', value: 2 }
  ]

  // 加载集群列表
  const loadClusterList = async () => {
    try {
      clusterLoading.value = true
      const response = await searchClusterApi({ page: 1, pageSize: 200 })
      clusterOptions.value = (response.items || []).map((item) => ({
        label: item.name,
        value: item.uuid
      }))
    } catch (error) {
      console.error('加载集群列表失败:', error)
    } finally {
      clusterLoading.value = false
    }
  }

  // 加载项目列表
  const loadProjectList = async () => {
    try {
      projectLoading.value = true
      const response = await searchProjectApi({ page: 1, pageSize: 200 })
      const items = (response.items || []).map((item) => ({
        label: item.name,
        value: item.id,
        raw: item
      }))
      projectOptions.value = [{ label: '全部', value: -1 }, { label: '非项目', value: 0 }, ...items]
    } catch (error) {
      console.error('加载项目列表失败:', error)
    } finally {
      projectLoading.value = false
    }
  }

  // 加载工作空间列表
  const loadWorkspaceList = async (projectClusterId: number) => {
    if (!projectClusterId) {
      workspaceOptions.value = [
        { label: '全部', value: -1 },
        { label: '非工作空间', value: 0 }
      ]
      return
    }

    try {
      workspaceLoading.value = true
      const response = await searchProjectWorkspaceApi({ projectClusterId })
      workspaceOptions.value = [
        { label: '全部', value: -1 },
        { label: '非工作空间', value: 0 },
        ...(response || []).map((item) => ({
          label: `${item.name} (${item.namespace})`,
          value: item.id
        }))
      ]
    } catch (error) {
      console.error('加载工作空间列表失败:', error)
      workspaceOptions.value = [
        { label: '全部', value: -1 },
        { label: '非工作空间', value: 0 }
      ]
    } finally {
      workspaceLoading.value = false
    }
  }

  // 监听项目选择变化
  watch(
    () => formData.value.projectId,
    async (newProjectId) => {
      formData.value.workspaceId = -1

      if (newProjectId === -1 || newProjectId === 0 || !newProjectId) {
        workspaceOptions.value = [
          { label: '全部', value: -1 },
          { label: '非工作空间', value: 0 }
        ]
        selectedProject.value = null
        return
      }

      const project = projectOptions.value.find((p) => p.value === newProjectId && p.raw)
      selectedProject.value = project?.raw

      let projectClusterId = null
      if (selectedProject.value?.projectClusters?.length > 0) {
        projectClusterId = selectedProject.value.projectClusters[0].id
      } else if (selectedProject.value?.projectClusterId) {
        projectClusterId = selectedProject.value.projectClusterId
      } else if (selectedProject.value?.id) {
        projectClusterId = selectedProject.value.id
      }

      if (projectClusterId) {
        await loadWorkspaceList(projectClusterId)
      } else {
        workspaceOptions.value = [
          { label: '全部', value: -1 },
          { label: '非工作空间', value: 0 }
        ]
      }
    }
  )

  // 表单配置项
  const formItems = computed(() => [
    {
      label: '集群',
      key: 'clusterUuid',
      type: 'select',
      props: {
        placeholder: '请选择集群',
        options: clusterOptions.value,
        clearable: true,
        filterable: true,
        loading: clusterLoading.value
      }
    },
    {
      label: '项目',
      key: 'projectId',
      type: 'select',
      props: {
        placeholder: '请选择项目',
        options: projectOptions.value,
        clearable: true,
        filterable: true,
        loading: projectLoading.value
      }
    },
    {
      label: '工作空间',
      key: 'workspaceId',
      type: 'select',
      props: {
        placeholder: '请选择工作空间',
        options: workspaceOptions.value,
        clearable: true,
        filterable: true,
        loading: workspaceLoading.value,
        disabled:
          !formData.value.projectId ||
          formData.value.projectId === -1 ||
          formData.value.projectId === 0
      }
    },
    {
      label: '操作简称',
      key: 'title',
      type: 'input',
      props: {
        placeholder: '请输入操作简称',
        clearable: true,
        maxlength: 200
      }
    },
    {
      label: '操作人',
      key: 'operatorName',
      type: 'input',
      props: {
        placeholder: '请输入操作人',
        clearable: true,
        maxlength: 100
      }
    },
    {
      label: '操作状态',
      key: 'status',
      type: 'select',
      props: {
        placeholder: '请选择状态',
        options: statusOptions,
        clearable: true
      }
    },
    {
      label: '操作时间',
      key: 'timeRange',
      type: 'daterange',
      props: {
        type: 'datetimerange',
        startPlaceholder: '开始时间',
        endPlaceholder: '结束时间',
        clearable: true,
        valueFormat: 'x',
        rangeSeparator: '至',
        defaultTime: [
          new Date(0, 0, 0, 0, 0, 0), // 开始时间默认 00:00:00
          new Date(0, 0, 0, 23, 59, 59) // 结束时间默认 23:59:59
        ]
      }
    }
  ])

  // 处理重置
  const handleReset = () => {
    selectedProject.value = null
    workspaceOptions.value = [
      { label: '全部', value: -1 },
      { label: '非工作空间', value: 0 }
    ]
    emit('reset')
  }

  // 处理搜索
  const handleSearch = async () => {
    try {
      await searchBarRef.value?.validate()

      // 过滤掉空值，并处理时间范围
      const params = Object.entries(formData.value).reduce(
        (acc, [key, value]) => {
          // 跳过空值
          if (value === undefined || value === null || value === '') {
            return acc
          }

          // 处理时间范围
          if (key === 'timeRange' && Array.isArray(value) && value.length === 2) {
            const startMs = typeof value[0] === 'string' ? parseInt(value[0]) : Number(value[0])
            const endMs = typeof value[1] === 'string' ? parseInt(value[1]) : Number(value[1])

            // 转换为秒级时间戳
            acc.startAt = Math.floor(startMs / 1000)
            acc.endAt = Math.floor(endMs / 1000)

            // 创建 Date 对象用于显示
            const startDate = new Date(startMs)
            const endDate = new Date(endMs)

            // 打印详细日志
            console.log('🕐 时间范围选择：')
            console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━')
            console.log(
              '📅 开始时间（本地）:',
              startDate.toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })
            )
            console.log('   毫秒时间戳:', startMs)
            console.log('   秒级时间戳:', acc.startAt)
            console.log('   UTC 时间:', startDate.toUTCString())
            console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━')
            console.log(
              '📅 结束时间（本地）:',
              endDate.toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })
            )
            console.log('   毫秒时间戳:', endMs)
            console.log('   秒级时间戳:', acc.endAt)
            console.log('   UTC 时间:', endDate.toUTCString())
            console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━')

            return acc
          }

          // 其他字段直接赋值（包括 -1）
          acc[key] = value
          return acc
        },
        {} as Record<string, any>
      )

      console.log('🔍 最终搜索参数:', params)
      console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━')
      emit('search', params)
    } catch (error) {
      console.error('搜索验证失败:', error)
    }
  }

  // 初始化
  onMounted(() => {
    loadClusterList()
    loadProjectList()
  })
</script>

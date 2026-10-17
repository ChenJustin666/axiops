<template>
  <ElAutocomplete
    v-model="inputValue"
    :fetch-suggestions="handleSearch"
    :placeholder="placeholder"
    :disabled="disabled"
    clearable
    style="width: 100%"
    :debounce="300"
    @select="handleSelect"
    @clear="handleClear"
  >
    <template #default="{ item }">
      <div class="autocomplete-item">
        <span class="item-name">{{ item.value }}</span>
        <span v-if="item.namespace" class="item-namespace">{{ item.namespace }}</span>
      </div>
    </template>
    <template #suffix>
      <ElTooltip :content="tooltipContent" placement="top">
        <HelpCircle :size="13" class="help-icon" />
      </ElTooltip>
    </template>
  </ElAutocomplete>
</template>

<script setup lang="ts">
  import { ref, watch } from 'vue'
  import { HelpCircle } from 'lucide-vue-next'
  import {
    getRoleListApi,
    getServiceAccountListApi,
    type ProjectWorkspace
  } from '@/api/workload/core'

  interface Props {
    modelValue: string
    resourceType: 'role' | 'serviceaccount'
    workspace: ProjectWorkspace | null
    placeholder?: string
    disabled?: boolean
    roleKind?: 'Role' | 'ClusterRole'
  }

  interface SuggestionItem {
    value: string
    namespace?: string
  }

  const props = withDefaults(defineProps<Props>(), {
    placeholder: '输入或选择',
    disabled: false,
    roleKind: 'Role'
  })

  const emit = defineEmits(['update:modelValue'])

  const inputValue = ref(props.modelValue)
  const allItems = ref<SuggestionItem[]>([])
  const loading = ref(false)

  const tooltipContent =
    props.resourceType === 'role'
      ? '可以手动输入或从列表中选择 Role'
      : '可以手动输入或从列表中选择 ServiceAccount'

  watch(
    () => props.modelValue,
    (newVal) => {
      inputValue.value = newVal
    }
  )

  watch(inputValue, (newVal) => {
    emit('update:modelValue', newVal)
  })

  const fetchRoles = async (search: string = '') => {
    if (!props.workspace?.clusterUuid || !props.workspace?.namespace) {
      return []
    }

    try {
      loading.value = true
      const params = {
        clusterUuid: props.workspace.clusterUuid,
        namespace: props.workspace.namespace,
        search: search || undefined
      }

      const response = await getRoleListApi(params)
      const items = Array.isArray(response) ? response : response?.items || []

      return items
        .filter((item) => {
          if (search) {
            return item.name.toLowerCase().includes(search.toLowerCase())
          }
          return true
        })
        .slice(0, 200)
        .map((item) => ({
          value: item.name,
          namespace: item.namespace
        }))
    } catch (error) {
      console.error('获取 Role 列表失败:', error)
      return []
    } finally {
      loading.value = false
    }
  }

  const fetchServiceAccounts = async (search: string = '') => {
    if (!props.workspace?.clusterUuid || !props.workspace?.namespace) {
      return []
    }

    try {
      loading.value = true
      const params = {
        clusterUuid: props.workspace.clusterUuid,
        namespace: props.workspace.namespace,
        search: search || undefined
      }

      const response = await getServiceAccountListApi(params)
      const items = Array.isArray(response) ? response?.items || response : response?.items || []

      return items
        .filter((item) => {
          if (search) {
            return item.name.toLowerCase().includes(search.toLowerCase())
          }
          return true
        })
        .slice(0, 200)
        .map((item) => ({
          value: item.name,
          namespace: item.namespace
        }))
    } catch (error) {
      console.error('获取 ServiceAccount 列表失败:', error)
      return []
    } finally {
      loading.value = false
    }
  }

  const handleSearch = async (queryString: string, cb: (suggestions: SuggestionItem[]) => void) => {
    if (!props.workspace) {
      cb([])
      return
    }

    let results: SuggestionItem[] = []

    if (props.resourceType === 'role') {
      results = await fetchRoles(queryString)
    } else if (props.resourceType === 'serviceaccount') {
      results = await fetchServiceAccounts(queryString)
    }

    if (queryString && results.length === 0) {
      results = [{ value: queryString }]
    }

    cb(results)
  }

  const handleSelect = (item: SuggestionItem) => {
    inputValue.value = item.value
    emit('update:modelValue', item.value)
  }

  const handleClear = () => {
    inputValue.value = ''
    emit('update:modelValue', '')
  }
</script>

<style lang="scss" scoped>
  .autocomplete-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;

    .item-name {
      flex: 1;
      font-size: 13px;
      color: #303133;
    }

    .item-namespace {
      font-size: 11px;
      color: #909399;
      margin-left: 8px;
    }
  }

  .help-icon {
    cursor: help;
    color: #909399;
    transition: color 0.2s;

    &:hover {
      color: #606266;
    }
  }

  :deep(.el-autocomplete) {
    width: 100%;
  }
</style>

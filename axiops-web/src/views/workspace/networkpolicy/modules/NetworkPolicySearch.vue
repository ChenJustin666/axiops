<template>
  <ElCard class="search-card" shadow="never">
    <ElForm :model="formModel" label-width="80px" class="search-form">
      <ElRow :gutter="16">
        <ElCol :xs="24" :sm="12" :md="8">
          <ElFormItem label="搜索">
            <ElInput
              v-model="formModel.search"
              placeholder="搜索名称"
              clearable
              @clear="handleClear"
              @keyup.enter="handleSearch"
            >
              <template #prefix>
                <ElIcon><Search /></ElIcon>
              </template>
            </ElInput>
          </ElFormItem>
        </ElCol>
        <ElCol :xs="24" :sm="12" :md="8">
          <ElFormItem label="标签选择">
            <ElInput
              v-model="formModel.labelSelector"
              placeholder="例如: app=web"
              clearable
              @clear="handleClear"
              @keyup.enter="handleSearch"
            >
              <template #prefix>
                <ElIcon><Tag /></ElIcon>
              </template>
            </ElInput>
          </ElFormItem>
        </ElCol>
        <ElCol :xs="24" :sm="24" :md="8">
          <ElFormItem class="action-buttons">
            <ElSpace>
              <ElButton type="primary" @click="handleSearch">查 询</ElButton>
              <ElButton @click="handleReset">重 置</ElButton>
            </ElSpace>
          </ElFormItem>
        </ElCol>
      </ElRow>
    </ElForm>
  </ElCard>
</template>

<script setup lang="ts">
  import { ref, watch } from 'vue'
  import { Search, Tag } from 'lucide-vue-next'

  interface SearchFormModel {
    search?: string
    labelSelector?: string
  }

  interface Props {
    modelValue: SearchFormModel
  }

  const props = defineProps<Props>()

  const emit = defineEmits<{
    'update:modelValue': [value: SearchFormModel]
    search: [params: SearchFormModel]
    reset: []
  }>()

  const formModel = ref<SearchFormModel>({
    search: '',
    labelSelector: ''
  })

  watch(
    () => props.modelValue,
    (val) => {
      if (val) {
        formModel.value = { ...val }
      }
    },
    { immediate: true, deep: true }
  )

  watch(formModel, (val) => {
    emit('update:modelValue', val)
  })

  const handleSearch = () => {
    const params: SearchFormModel = {}
    if (formModel.value.search?.trim()) {
      params.search = formModel.value.search.trim()
    }
    if (formModel.value.labelSelector?.trim()) {
      params.labelSelector = formModel.value.labelSelector.trim()
    }
    emit('search', params)
  }

  const handleReset = () => {
    formModel.value = {
      search: '',
      labelSelector: ''
    }
    emit('reset')
  }

  const handleClear = () => {
    handleSearch()
  }

  defineExpose({
    handleReset,
    handleSearch
  })
</script>

<style lang="scss" scoped>
  .search-card {
    margin-bottom: 12px;

    :deep(.el-card__body) {
      padding: 16px;
    }
  }

  .search-form {
    .action-buttons {
      :deep(.el-form-item__content) {
        display: flex;
        justify-content: flex-start;
      }
    }
  }
</style>

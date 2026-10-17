<template>
  <div class="billing-price-config-page art-full-height">
    <!-- 搜索栏 -->
    <BillingSearch
      v-show="showSearchBar"
      v-model="searchForm"
      @search="handleSearch"
      @reset="resetSearchParams"
    />

    <ElCard
      class="art-table-card"
      shadow="never"
      :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
    >
      <!-- 表格头部 -->
      <ArtTableHeader
        v-model:columns="columns"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        :showZebra="true"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'search,refresh,size,fullscreen,columns,settings'"
        @refresh="refreshData"
      >
        <template #left>
          <ElSpace wrap>
            <ElButton type="primary" @click="showDialog('add')">新增配置</ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <!-- 数据表格 -->
      <ArtTable :loading="loading" :data="data" :columns="columns" />
    </ElCard>

    <!-- 收费配置编辑弹窗 -->
    <BillingDialog
      v-model="dialogVisible"
      :dialog-type="dialogType"
      :config-data="currentConfigData"
      @success="refreshData"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { ButtonMoreItem } from '@/components/core/forms/art-button-more/index.vue'
  import {
    searchBillingPriceConfigApi,
    deleteBillingPriceConfigApi,
    type BillingPriceConfig
  } from '@/api/manager/billing'
  import BillingSearch from './modules/billing-search.vue'
  import BillingDialog from './modules/billing-dialog.vue'
  import { useBillingTableConfig } from './modules/table-config'

  defineOptions({ name: 'BillingPriceConfig' })

  // 使用抽离的表格配置逻辑
  const { loading, showSearchBar, deleteLoadingMap, createTableColumns } = useBillingTableConfig()

  // 弹窗相关
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentConfigData = ref<BillingPriceConfig | undefined>(undefined)

  // 表格数据
  const data = ref<BillingPriceConfig[]>([])

  // 搜索表单
  const searchForm = ref({
    configName: undefined
  })

  // 搜索参数
  const searchParams = ref<any>({})

  // 按钮更多点击事件
  const buttonMoreClick = (item: ButtonMoreItem, row: BillingPriceConfig) => {
    switch (item.key) {
      case 'edit':
        showDialog('edit', row)
        break
      case 'delete':
        deleteConfig(row)
        break
    }
  }

  // 创建表格列配置
  const { columns } = createTableColumns({ buttonMoreClick })

  // 获取数据
  const getData = async () => {
    loading.value = true
    try {
      const response = await searchBillingPriceConfigApi(searchParams.value)
      data.value = response.data || []
    } finally {
      loading.value = false
    }
  }

  // 刷新数据
  const refreshData = () => {
    getData()
  }

  // 搜索处理
  const handleSearch = (params: Record<string, any>) => {
    searchParams.value = params
    getData()
  }

  // 重置搜索参数
  const resetSearchParams = () => {
    searchParams.value = {}
    getData()
  }

  // 显示对话框
  const showDialog = (type: 'add' | 'edit', row?: BillingPriceConfig) => {
    dialogVisible.value = true
    dialogType.value = type
    currentConfigData.value = row
  }

  // 删除配置
  const deleteConfig = async (row: BillingPriceConfig) => {
    // 系统内置配置不允许删除
    if (row.isSystem === 1) {
      ElMessage.warning('系统内置配置不允许删除')
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定删除收费配置「${row.configName}」吗？此操作不可恢复！`,
        '删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      deleteLoadingMap.value[row.id] = true
      await deleteBillingPriceConfigApi(row.id)
      ElMessage.success('删除成功')
      refreshData()
    } catch (error: any) {
      if (error !== 'cancel') {
        deleteLoadingMap.value[row.id] = false
      }
    } finally {
      deleteLoadingMap.value[row.id] = false
    }
  }

  // 初始化
  onMounted(() => {
    getData()
  })
</script>

<style lang="scss" scoped>
  .billing-price-config-page {
    padding-bottom: 15px;
  }
</style>

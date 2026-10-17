<template>
  <table class="w-full">
    <thead>
      <tr class="border-b border-g-200">
        <th class="text-left py-3 px-4 text-sm font-medium text-g-500">集群名称</th>
        <th class="text-center py-3 px-3 text-sm font-medium text-g-500 w-20">总数</th>
        <th class="text-center py-3 px-3 text-sm font-medium text-g-500 w-20">触发中</th>
        <th class="text-center py-3 px-3 text-sm font-medium text-g-500 w-20">Critical</th>
        <th class="text-center py-3 px-3 text-sm font-medium text-g-500 w-20">Warning</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="item in tableData"
        :key="item.itemId"
        class="border-b border-g-100 hover:bg-g-50 transition-colors"
      >
        <td class="py-3 px-4 text-sm font-medium">{{ item.itemName || '-' }}</td>
        <td class="py-3 px-3 text-sm text-center">{{ item.alertCount }}</td>
        <td class="py-3 px-3 text-center">
          <span
            v-if="item.firingCount > 0"
            class="inline-flex items-center justify-center min-w-7 h-5.5 px-2 rounded-full text-xs font-semibold bg-danger/10 text-danger"
          >
            {{ item.firingCount }}
          </span>
          <span v-else class="text-g-400 text-sm">0</span>
        </td>
        <td class="py-3 px-3 text-sm text-center text-danger font-medium">{{
          item.criticalCount
        }}</td>
        <td class="py-3 px-3 text-sm text-center text-warning font-medium">{{
          item.warningCount
        }}</td>
      </tr>
      <tr v-if="tableData.length === 0">
        <td colspan="5" class="py-12 text-center text-g-400 text-sm">暂无数据</td>
      </tr>
    </tbody>
  </table>
</template>

<script setup lang="ts">
  import { ref, watch } from 'vue'
  import { getAlertTopRankingApi, type RankingItem } from '@/api/manager/alert-dashboard'

  interface Props {
    filter: {
      clusterUuid?: string
      projectId?: number
      workspaceId?: number
    }
  }

  const props = defineProps<Props>()
  const tableData = ref<RankingItem[]>([])

  // 加载数据
  const loadData = async () => {
    try {
      const res = await getAlertTopRankingApi({
        clusterUuid: props.filter.clusterUuid || undefined,
        projectId: props.filter.projectId || undefined,
        workspaceId: props.filter.workspaceId || undefined,
        rankingType: 'cluster',
        topN: 10
      })
      tableData.value = res.items || []
    } catch (error) {}
  }

  // 刷新
  const refresh = () => loadData()

  watch(
    () => props.filter,
    () => loadData(),
    { deep: true, immediate: true }
  )

  defineExpose({ refresh })
</script>

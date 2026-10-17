<template>
  <div class="space-y-3 px-1">
    <div v-for="(item, index) in rankingList" :key="item.itemId" class="flex items-center gap-3">
      <div
        class="flex-cc size-6 rounded-full text-xs font-semibold flex-shrink-0"
        :class="getRankClass(index + 1)"
      >
        {{ index + 1 }}
      </div>
      <div class="flex-1 min-w-0">
        <div class="flex-b gap-3 mb-1.5">
          <span class="text-sm font-medium truncate">{{ item.itemName || '-' }}</span>
          <span class="text-sm text-g-500 flex-shrink-0">{{ item.alertCount }} 次</span>
        </div>
        <div class="h-1.5 bg-g-100 rounded-full overflow-hidden">
          <div
            class="h-full rounded-full transition-all duration-500"
            :class="getBarClass(index)"
            :style="{ width: getPercent(item.alertCount) + '%' }"
          ></div>
        </div>
      </div>
    </div>
    <div v-if="rankingList.length === 0" class="py-12 text-center text-g-400 text-sm">暂无数据</div>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { getAlertTopRankingApi, type RankingItem } from '@/api/manager/alert-dashboard'

  interface Props {
    filter: {
      clusterUuid?: string
      projectId?: number
      workspaceId?: number
    }
  }

  const props = defineProps<Props>()
  const rankingList = ref<RankingItem[]>([])

  // 计算最大值
  const maxCount = computed(() => {
    if (rankingList.value.length === 0) return 1
    return Math.max(...rankingList.value.map((item) => item.alertCount))
  })

  // 计算百分比
  const getPercent = (count: number): number => {
    return Math.round((count / maxCount.value) * 100)
  }

  // 获取排名样式
  const getRankClass = (rank: number): string => {
    if (rank === 1) return 'bg-danger/10 text-danger'
    if (rank === 2) return 'bg-warning/10 text-warning'
    if (rank === 3) return 'bg-success/10 text-success'
    return 'bg-g-100 text-g-500'
  }

  // 获取进度条样式
  const getBarClass = (index: number): string => {
    const classes = ['bg-danger', 'bg-warning', 'bg-success', 'bg-info', 'bg-g-400']
    return classes[index % classes.length]
  }

  // 加载数据
  const loadData = async () => {
    try {
      const res = await getAlertTopRankingApi({
        clusterUuid: props.filter.clusterUuid || undefined,
        projectId: props.filter.projectId || undefined,
        workspaceId: props.filter.workspaceId || undefined,
        rankingType: 'rule',
        topN: 10
      })
      rankingList.value = res.items || []
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

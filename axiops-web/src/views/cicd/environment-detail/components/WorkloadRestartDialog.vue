<template>
  <ElDialog
    v-model="visible"
    title="滚动重启"
    width="min(580px, 94vw)"
    :close-on-click-modal="false"
    :close-on-press-escape="!submitting"
    :show-close="!submitting"
    @closed="closed"
  >
    <div v-loading="loading" class="restart-preview">
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <template v-if="preview">
        <div class="restart-target"
          ><strong>{{ preview.kind }}/{{ preview.name }}</strong
          ><ElTag v-if="preview.is_production" type="danger" size="small">生产环境</ElTag></div
        >
        <p>{{ environment.name }} · 集群 #{{ preview.cluster_id }} · {{ preview.namespace }}</p>
        <div class="restart-images"
          ><code v-for="image in preview.images" :key="image">{{ image }}</code></div
        >
        <ElAlert
          title="保留当前镜像和配置，按控制器策略滚动替换容器。单副本服务是否中断取决于更新策略和就绪探针。"
          type="info"
          :closable="false"
        />
        <ElCheckbox v-if="preview.is_production" v-model="productionConfirmed"
          >确认重启上述生产环境工作负载</ElCheckbox
        >
      </template>
    </div>
    <template #footer>
      <ElButton :disabled="submitting" @click="visible = false">取消</ElButton>
      <ElButton v-if="!preview" :loading="loading" @click="loadPreview">重新读取</ElButton>
      <ElButton
        type="primary"
        :loading="submitting"
        :disabled="!preview || loading || (preview.is_production && !productionConfirmed)"
        @click="submit"
        >{{ submittedUncertain ? '确认原请求结果' : '确认重启' }}</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import type { AxiOpsEnvironment } from '@/api/axiops/cicd'
  import {
    previewEnvRestartApi,
    submitEnvRestartApi,
    type EnvRestartPreview,
    type EnvWorkloadItem
  } from '@/api/axiops/env-resource'

  const props = defineProps<{
    modelValue: boolean
    environment: AxiOpsEnvironment
    workload: EnvWorkloadItem | null
  }>()
  const emit = defineEmits<{ 'update:modelValue': [boolean]; submitted: [number] }>()
  const visible = computed({
    get: () => props.modelValue,
    set: (value) => {
      if (!submitting.value) emit('update:modelValue', value)
    }
  })
  const preview = ref<EnvRestartPreview | null>(null),
    loading = ref(false),
    submitting = ref(false),
    error = ref(''),
    productionConfirmed = ref(false),
    submittedUncertain = ref(false)
  let generation = 0,
    requestKey = ''
  const contextKey = () =>
    [
      props.environment.id,
      props.environment.cluster_id,
      props.environment.namespace,
      props.workload?.kind,
      props.workload?.name
    ].join(':')
  function closed() {
    generation++
    loading.value = false
    preview.value = null
    error.value = ''
    productionConfirmed.value = false
    submittedUncertain.value = false
    requestKey = ''
  }
  async function loadPreview() {
    if (!props.modelValue || !props.workload || submitting.value) return
    const current = ++generation,
      key = contextKey()
    loading.value = true
    preview.value = null
    error.value = ''
    productionConfirmed.value = false
    try {
      const next = await previewEnvRestartApi(
        props.environment.id,
        props.workload.kind,
        props.workload.name
      )
      if (current !== generation || key !== contextKey() || !props.modelValue) return
      if (
        next.environment_id !== props.environment.id ||
        next.namespace !== props.environment.namespace ||
        next.cluster_id !== props.environment.cluster_id ||
        next.name !== props.workload?.name ||
        next.kind !== props.workload?.kind
      )
        throw Error('返回目标与当前环境不一致，请刷新')
      preview.value = next
      requestKey = `restart-${crypto.randomUUID()}`
      submittedUncertain.value = false
    } catch (e) {
      if (current === generation) error.value = e instanceof Error ? e.message : '读取重启目标失败'
    } finally {
      if (current === generation) loading.value = false
    }
  }
  async function submit() {
    if (
      !preview.value ||
      submitting.value ||
      (preview.value.is_production && !productionConfirmed.value)
    )
      return
    const current = generation,
      key = contextKey(),
      target = preview.value
    submitting.value = true
    error.value = ''
    try {
      const result = await submitEnvRestartApi(target.environment_id, {
        kind: target.kind,
        name: target.name,
        expected_token: target.token,
        request_key: requestKey
      })
      if (current !== generation || key !== contextKey()) return
      emit('submitted', result.id)
      emit('update:modelValue', false)
    } catch (e) {
      if (current === generation) {
        submittedUncertain.value = true
        error.value = `${e instanceof Error ? e.message : '提交结果待确认'}。可确认原请求结果，不会重复提交重启。`
      }
    } finally {
      submitting.value = false
      if (current !== generation && props.modelValue) void loadPreview()
    }
  }
  watch(
    () => [props.modelValue, contextKey()],
    () => {
      closed()
      if (props.modelValue) void loadPreview()
    },
    { immediate: true }
  )
</script>

<style scoped>
  .restart-preview {
    min-height: 150px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .restart-target {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 12px;
  }
  .restart-preview p {
    color: var(--el-text-color-secondary);
    margin: 0;
  }
  .restart-images {
    display: grid;
    gap: 6px;
    font-size: 12px;
    overflow-wrap: anywhere;
  }
</style>

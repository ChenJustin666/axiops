<template>
  <section v-if="items.length || error" class="verification-panel" v-loading="loading">
    <div class="verification-heading"
      ><strong>测试验收与生产发布</strong
      ><ElButton text size="small" :disabled="busy" @click="load">刷新</ElButton></div
    >
    <ElAlert v-if="error" :title="error" type="warning" :closable="false" />
    <article v-for="item in items" :key="item.service_id" class="verification-item">
      <div
        ><strong>{{ item.name }}</strong
        ><ElTag v-if="item.verification?.status === 'passed'" type="success" size="small"
          >人工验收通过</ElTag
        ><ElTag v-else type="info" size="small">{{
          item.verification?.status === 'revoked' ? '验收已撤销' : '待测试验收'
        }}</ElTag></div
      >
      <p v-if="item.verification"
        >{{ item.verification.username }} · {{ date(item.verification.verified_at) }} ·
        {{ item.verification.note }}</p
      >
      <p v-if="item.blocked_reason" class="verification-note">{{ item.blocked_reason }}</p>
      <div class="verification-actions"
        ><ElButton
          v-if="item.verification?.status !== 'passed'"
          size="small"
          :disabled="busy || !!item.blocked_reason"
          @click="verify(item, 'pass')"
          >记录测试通过</ElButton
        ><template v-else
          ><ElButton
            size="small"
            type="primary"
            plain
            :disabled="busy"
            @click="chooseDestination(item)"
            >发布到生产</ElButton
          ><ElButton text size="small" :disabled="busy" @click="verify(item, 'revoke')"
            >撤销验收</ElButton
          ></template
        ></div
      >
    </article>
    <div v-if="destinations.length > 1" class="verification-destination"
      ><ElSelect v-model="targetWorkflowId" placeholder="选择生产工作流"
        ><ElOption
          v-for="target in destinations"
          :key="target.id"
          :value="target.id"
          :label="target.name" /></ElSelect
      ><ElButton :disabled="!targetWorkflowId" @click="releaseVisible = true"
        >核对生产发布</ElButton
      ></div
    >
    <ProductionReleaseDialog
      v-if="targetWorkflowId"
      v-model:visible="releaseVisible"
      :workflow-id="targetWorkflowId"
      :project-id="projectId"
      :service-id="chosenService"
      :verification-id="chosenVerification"
      @submitted="openRun"
    />
  </section>
</template>
<script setup lang="ts">
  import { ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useRouter } from 'vue-router'
  import {
    getVerificationOptionsApi,
    saveVerificationApi,
    listWorkflowsApi,
    listEnvironmentsApi,
    getProductionOptionsApi,
    type VerificationOption
  } from '@/api/axiops'
  import ProductionReleaseDialog from './ProductionReleaseDialog.vue'
  const props = defineProps<{
    runId: number
    projectId: number
    environmentId: number
    refreshKey?: string
  }>()
  const router = useRouter(),
    items = ref<VerificationOption[]>([]),
    loading = ref(false),
    busy = ref(false),
    error = ref('')
  const destinations = ref<{ id: number; name: string }[]>([]),
    targetWorkflowId = ref<number>(),
    chosenService = ref<number>(),
    chosenVerification = ref<number>(),
    releaseVisible = ref(false)
  let generation = 0
  const date = (value: string) => new Date(value).toLocaleString('zh-CN', { hour12: false })
  const message = (e: unknown) => (e as { message?: string })?.message || '请求失败，请重试'
  const load = async () => {
    const gen = ++generation
    loading.value = true
    error.value = ''
    try {
      const result = await getVerificationOptionsApi(props.runId)
      if (gen === generation) items.value = result
    } catch (e) {
      if (gen === generation) error.value = message(e)
    } finally {
      if (gen === generation) loading.value = false
    }
  }
  const verify = async (item: VerificationOption, action: 'pass' | 'revoke') => {
    if (busy.value) return
    let note: string
    try {
      const result = await ElMessageBox.prompt(
        action === 'pass'
          ? `确认已对 ${item.name} 当前部署完成功能测试。镜像：${item.image_ref}`
          : `撤销后，此版本将不能继续发布到生产；已经完成的生产发布保留。`,
        action === 'pass' ? '记录人工验收通过' : '撤销验收',
        {
          inputPlaceholder:
            action === 'pass' ? '填写测试范围或结果，便于上线时核对' : '填写撤销原因',
          inputValidator: (value) => !!value?.trim() || '请填写测试结论或撤销原因',
          confirmButtonText: '确认记录',
          cancelButtonText: '取消'
        }
      )
      note = result.value.trim()
    } catch {
      return
    }
    busy.value = true
    try {
      await saveVerificationApi(props.runId, {
        snapshot_id: item.snapshot_id,
        expected_digest: item.digest,
        revision: item.verification?.revision || 0,
        action,
        note
      })
      ElMessage.success(action === 'pass' ? '已记录人工验收通过' : '已撤销验收')
      await load()
    } catch (e) {
      await load()
      error.value = `${message(e)}；已重新读取验收状态，请核对后再操作。`
    } finally {
      busy.value = false
    }
  }
  const chooseDestination = async (item: VerificationOption) => {
    if (busy.value || !item.verification) return
    busy.value = true
    error.value = ''
    destinations.value = []
    targetWorkflowId.value = undefined
    chosenService.value = item.service_id
    chosenVerification.value = item.verification.id
    try {
      const [workflows, environments] = await Promise.all([
        listWorkflowsApi(props.projectId),
        listEnvironmentsApi(props.projectId)
      ])
      const prodIds = new Set(environments.filter((e) => e.is_production).map((e) => e.id))
      const targets = workflows.filter(
        (w) => w.delivery_type === 'k8s_yaml' && prodIds.has(w.target_environment_id!)
      )
      const results = await Promise.allSettled(targets.map((w) => getProductionOptionsApi(w.id)))
      for (let i = 0; i < results.length; i++) {
        const result = results[i]
        if (result.status !== 'fulfilled') continue
        const options = result.value,
          workflow = targets[i]
        if (
          options.policy.source_environment_id === props.environmentId &&
          options.services.some(
            (s) =>
              s.service_id === item.service_id &&
              s.candidates.some((v) => v.id === item.verification?.id)
          )
        )
          destinations.value.push({
            id: workflow.id,
            name: `${options.environment_name} · ${workflow.name}`
          })
      }
      if (destinations.value.length === 1) {
        targetWorkflowId.value = destinations.value[0].id
        releaseVisible.value = true
      }
      if (!destinations.value.length)
        error.value = results.some((r) => r.status === 'rejected')
          ? '部分生产工作流读取失败或没有权限，请检查权限或重试。'
          : '暂无匹配的生产工作流；请先创建生产环境和工作流，并将发布来源设为此测试环境。'
    } catch (e) {
      error.value = message(e)
    } finally {
      busy.value = false
    }
  }
  const openRun = (id: number) => {
    void router.push(
      `/cicd/projects/${props.projectId}/workflows/${targetWorkflowId.value}/runs/${id}`
    )
  }
  watch(
    () => [props.runId, props.refreshKey],
    () => {
      if (props.runId && props.environmentId) void load()
    },
    { immediate: true }
  )
</script>
<style scoped>
  .verification-panel {
    margin: 14px 0;
    padding: 16px 20px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
  }
  .verification-heading,
  .verification-item > div {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .verification-heading {
    justify-content: space-between;
  }
  .verification-item {
    padding: 12px 0;
  }
  .verification-item + .verification-item {
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .verification-item p {
    font-size: 13px;
    margin: 8px 0;
    color: var(--el-text-color-secondary);
  }
  .verification-actions {
    justify-content: flex-end;
  }
  .verification-destination {
    display: flex;
    gap: 12px;
  }
  .verification-destination .el-select {
    max-width: 360px;
  }
</style>

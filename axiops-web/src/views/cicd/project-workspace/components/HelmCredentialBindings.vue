<template>
  <ElCollapse @change="loadCredentials">
    <ElCollapseItem title="安全凭据 · 多字段引用" name="credentials">
      <p
        >选择凭据中的字段并绑定到 values 路径。只保存引用，正文不展示。已有 Secret 可直接在 values
        中使用 secretKeyRef、envFrom 或 secretMounts。</p
      >
      <p
        >通用模板示例：secrets.auth.stringData.SK。含点号的 Secret 键可用
        secrets.auth.fields.cert.value，并在 values 设置 fields.cert.key: tls.crt。</p
      >
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <div v-for="(row, index) in rows" :key="row.id" class="credential-row">
        <ElInput
          v-model="row.path"
          :readonly="!!row.original"
          :disabled="disabled"
          placeholder="secrets.auth.stringData.password"
          aria-label="凭据 values 路径"
          @input="update"
        />
        <ElSelect
          v-model="row.credential_id"
          :disabled="disabled || loading"
          filterable
          placeholder="选择安全凭据"
          @change="changeCredential(row)"
          ><ElOption v-for="c in credentials" :key="c.id" :value="c.id" :label="c.name" /><ElOption
            v-if="row.credential_id && !credentials.some((c) => c.id === row.credential_id)"
            :value="row.credential_id"
            :label="`凭据 #${row.credential_id}`"
            disabled
        /></ElSelect>
        <ElSelect
          v-model="row.key"
          :disabled="disabled"
          filterable
          placeholder="字段"
          @change="update"
          ><ElOption v-for="key in fieldKeys(row)" :key="key" :value="key" :label="key"
        /></ElSelect>
        <ElButton
          text
          size="small"
          :disabled="disabled || (!!row.original && row.origin !== 'override')"
          @click="remove(index)"
          >{{ row.original ? '恢复继承' : '移除' }}</ElButton
        >
      </div>
      <ElButton text size="small" :disabled="disabled" @click="add">添加字段引用</ElButton>
      <small v-if="rows.some((r) => r.origin === 'global')"
        >共享凭据可在此覆盖；删除环境覆盖后恢复继承。各环境需分别选择凭据。</small
      >
    </ElCollapseItem>
  </ElCollapse>
</template>
<script setup lang="ts">
  import { ref, watch, onBeforeUnmount } from 'vue'
  import {
    listCredentialsApi,
    type AxiOpsCredential,
    type ServiceEnvironmentConfig,
    type SaveServiceEnvironmentConfig
  } from '@/api/axiops'
  const props = defineProps<{
    modelValue: NonNullable<SaveServiceEnvironmentConfig['helm_credentials']>
    config: ServiceEnvironmentConfig
    projectId: number
    disabled: boolean
  }>()
  const emit = defineEmits<{
    'update:modelValue': [value: NonNullable<SaveServiceEnvironmentConfig['helm_credentials']>]
  }>()
  type Row = {
    id: number
    path: string
    original: string
    credential_id?: number
    key: string
    origin: string
  }
  const rows = ref<Row[]>([]),
    credentials = ref<AxiOpsCredential[]>([]),
    loading = ref(false),
    error = ref('')
  let generation = 0,
    nextId = 0,
    loaded = false
  const removed = new Set<string>()
  function reset() {
    generation++
    loaded = false
    credentials.value = []
    error.value = ''
    loading.value = false
    removed.clear()
    const effective = new Map(props.config.variables.map((v) => [v.key, v]))
    rows.value = [...effective.values()]
      .filter((v) => v.secret)
      .map((v) => ({
        id: ++nextId,
        path: v.key,
        original: v.key,
        credential_id: v.credential_id,
        key: props.config.helm?.credential_keys?.[v.key] || 'value',
        origin: v.origin
      }))
  }
  async function loadCredentials(open: Array<string | number> | string | number) {
    const expanded = Array.isArray(open) ? open.includes('credentials') : open === 'credentials'
    if (!expanded || loaded || loading.value) return
    const current = generation
    loading.value = true
    error.value = ''
    try {
      const list = await listCredentialsApi({ type: 'secret', project_id: props.projectId })
      if (current === generation) {
        credentials.value = list || []
        loaded = true
      }
    } catch (e: any) {
      if (current === generation) error.value = e?.message || '凭据列表读取失败'
    } finally {
      if (current === generation) loading.value = false
    }
  }
  function fieldKeys(row: Row) {
    const list = credentials.value.find((c) => c.id === row.credential_id)?.keys || []
    return [...new Set([...list, ...(row.key ? [row.key] : [])])]
  }
  function changeCredential(row: Row) {
    row.key = credentials.value.find((c) => c.id === row.credential_id)?.keys?.[0] || 'value'
    update()
  }
  function add() {
    if (props.disabled) return
    rows.value.push({ id: ++nextId, path: '', original: '', key: 'value', origin: '' })
    update()
  }
  function remove(index: number) {
    if (props.disabled) return
    const row = rows.value[index]
    if (row.original) removed.add(row.original)
    rows.value.splice(index, 1)
    update()
  }
  function update() {
    const delta: NonNullable<SaveServiceEnvironmentConfig['helm_credentials']> = {}
    for (const path of removed) delta[path] = null
    const effective = new Map(props.config.variables.map((v) => [v.key, v]))
    rows.value.forEach((row, index) => {
      const before = effective.get(row.original)
      if (
        row.original &&
        row.credential_id === before?.credential_id &&
        row.key === (props.config.helm?.credential_keys?.[row.original] || 'value')
      )
        return
      delta[row.path || `__incomplete_${index}`] = {
        credential_id: row.credential_id || 0,
        key: row.key
      }
    })
    const paths = rows.value.map((r) => r.path).filter(Boolean)
    if (new Set(paths).size !== paths.length) {
      delta.__invalid_duplicate = { credential_id: 0, key: '' }
      error.value = '凭据 values 路径不能重复'
    } else if (error.value === '凭据 values 路径不能重复') error.value = ''
    emit('update:modelValue', delta)
  }
  watch(() => [props.config.token, props.projectId], reset, { immediate: true })
  onBeforeUnmount(() => generation++)
</script>
<style scoped>
  p,
  small {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.8;
    overflow-wrap: anywhere;
  }
  .credential-row {
    display: grid;
    grid-template-columns: minmax(230px, 2fr) minmax(160px, 1fr) minmax(100px, 1fr) auto;
    gap: 8px;
    margin: 10px 0;
  }
  @media (max-width: 900px) {
    .credential-row {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>

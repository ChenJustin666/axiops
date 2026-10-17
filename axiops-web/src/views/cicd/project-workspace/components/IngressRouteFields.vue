<template>
  <div class="ingress-fields">
    <div class="grid">
      <ElFormItem label="路径类型">
        <ElSelect
          :model-value="modelValue.path_type || 'Prefix'"
          @update:model-value="patch({ path_type: $event })"
        >
          <ElOption value="Prefix" label="Prefix · 路径前缀" />
          <ElOption value="Exact" label="Exact · 精确匹配" />
          <ElOption value="ImplementationSpecific" label="ImplementationSpecific · 由控制器处理" />
        </ElSelect>
      </ElFormItem>
      <ElFormItem label="Ingress Class">
        <ElSelect
          :model-value="modelValue.class"
          filterable
          allow-create
          clearable
          default-first-option
          placeholder="选择或输入集群已有的 Class"
          @update:model-value="patch({ class: $event })"
        >
          <ElOption value="alb" label="alb · 阿里云 ALB" />
          <ElOption value="nginx" label="nginx" />
        </ElSelect>
      </ElFormItem>
    </div>
    <ElFormItem label="TLS / HTTPS"
      ><ElSwitch :model-value="tlsEnabled" @update:model-value="toggleTLS"
    /></ElFormItem>
    <div v-if="tlsEnabled" class="grid">
      <ElFormItem label="TLS 域名" required>
        <ElInput
          :model-value="(modelValue.tls_hosts || []).join(', ')"
          placeholder="api.example.com；多个域名用逗号分隔"
          @update:model-value="patch({ tls_hosts: $event.split(/[,，\s]+/).filter(Boolean) })"
        />
      </ElFormItem>
      <ElFormItem label="证书 Secret（可选）">
        <ElInput
          :model-value="modelValue.tls_secret"
          placeholder="ALB 使用云证书时可留空"
          @update:model-value="patch({ tls_secret: $event })"
        />
      </ElFormItem>
    </div>
    <p v-if="tlsEnabled" class="hint"
      >ALB 留空 Secret 时，由控制器匹配云证书；需配置 HTTPS
      监听与有效证书。其他控制器请按其要求填写证书 Secret。</p
    >
    <details>
      <summary>注解与标签 · 可选</summary>
      <p class="hint"
        >可直接粘贴 YAML 键值；删除对应行即可移除。ALB 正则与重写按需添加，普通 /
        路由不需要开启。</p
      >
      <ElFormItem label="Annotations" :error="errors.annotations">
        <ElInput
          :model-value="texts.annotations"
          type="textarea"
          :autosize="{ minRows: 3, maxRows: 10 }"
          :placeholder="annotationHint"
          @update:model-value="editMetadata('annotations', $event)"
        />
      </ElFormItem>
      <ElFormItem label="Labels" :error="errors.labels">
        <ElInput
          :model-value="texts.labels"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 6 }"
          placeholder="ingress-controller: alb"
          @update:model-value="editMetadata('labels', $event)"
        />
      </ElFormItem>
    </details>
  </div>
</template>
<script setup lang="ts">
  import { computed, reactive, watch } from 'vue'
  import { stringify } from 'yaml'
  import type { ServiceAssets } from '@/api/axiops/service-assets'
  import { parseIngressMetadata } from '@/utils/template-configuration'
  type Route = ServiceAssets['routes'][number]
  const props = defineProps<{ modelValue: Route }>()
  const emit = defineEmits<{ 'update:modelValue': [value: Route] }>()
  const patch = (value: Partial<Route>) =>
    emit('update:modelValue', { ...props.modelValue, ...value })
  const tlsEnabled = computed(() => !!(props.modelValue.tls_enabled || props.modelValue.tls_secret))
  function toggleTLS(value: boolean | string | number) {
    patch(
      value
        ? {
            tls_enabled: true,
            tls_hosts: props.modelValue.tls_hosts?.length
              ? [...props.modelValue.tls_hosts]
              : [props.modelValue.host].filter(Boolean)
          }
        : { tls_enabled: false, tls_hosts: [], tls_secret: '' }
    )
  }
  const annotationHint =
    'alb.ingress.kubernetes.io/order: "1"\nalb.ingress.kubernetes.io/listen-ports: \'[{"HTTP":80},{"HTTPS":443}]\''
  const texts = reactive({ annotations: '', labels: '' })
  const errors = reactive({ annotations: '', labels: '' })
  const signatures = { annotations: '', labels: '' }
  for (const field of ['annotations', 'labels'] as const) {
    watch(
      () => props.modelValue[field],
      (value) => {
        const signature = JSON.stringify(value || {})
        if (signatures[field] === signature) return
        signatures[field] = signature
        texts[field] = Object.keys(value || {}).length ? stringify(value) : ''
        errors[field] = ''
      },
      { immediate: true, deep: true }
    )
  }
  function editMetadata(field: 'annotations' | 'labels', text: string) {
    texts[field] = text
    let value: Record<string, string>
    try {
      value = parseIngressMetadata(text)
      errors[field] = ''
    } catch (error: any) {
      errors[field] = error.message
      // Keep invalid edits dirty and block both preview and save; never use stale valid values.
      value = { '': text }
    }
    signatures[field] = JSON.stringify(value)
    patch({ [field]: value })
  }
</script>
<style scoped>
  .grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px 24px;
  }
  .hint {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
  }
  details {
    margin: 8px 0 16px;
  }
  summary {
    cursor: pointer;
    font-size: 13px;
    margin-bottom: 12px;
  }
  @media (max-width: 600px) {
    .grid {
      grid-template-columns: 1fr;
    }
  }
</style>

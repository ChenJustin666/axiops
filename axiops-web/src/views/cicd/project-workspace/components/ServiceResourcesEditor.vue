<template>
  <section class="service-resources">
    <header class="resources-heading"
      ><div class="heading-text"
        ><strong>服务运行资源与配套配置</strong
        ><p
          >{{ service.name }} · {{ environment.name }} · {{ environment.namespace
          }}<ElTag v-if="environment.is_production" type="danger" size="small">生产</ElTag></p
        ></div
      ><ElTag type="info" size="small">保存后待发布</ElTag></header
    >
    <ElAlert
      v-if="!config.bound"
      title="请先将服务加入此环境，再配置文件、密码、域名和存储。"
      type="info"
      :closable="false"
    />
    <template v-else>
      <p class="help lead"
        >仅作用于<b>当前服务 × 当前环境</b>：文件生成 ConfigMap，密码以加密凭据引用生成
        Secret；移除只解除引用，已创建的 PVC 等资源保留。</p
      >
      <ElForm label-position="top" :disabled="!editable || busy">
        <div class="target-grid">
          <ElFormItem label="工作负载"
            ><ElSelect v-model="selectedWorkload" clearable placeholder="单个工作负载自动识别"
              ><ElOption
                v-for="item in workloadOptions"
                :key="item.key"
                :value="item.key"
                :label="item.key" /></ElSelect
          ></ElFormItem>
          <ElFormItem label="目标容器"
            ><ElSelect v-model="form.container" clearable placeholder="单个容器自动识别"
              ><ElOption
                v-for="name in containerOptions"
                :key="name"
                :value="name"
                :label="name" /></ElSelect
          ></ElFormItem>
        </div>
        <section class="quick-settings">
          <div class="quick-heading">
            <div>
              <strong>发布快捷设置</strong>
              <p class="help">只保存与模板的差异，留空或关闭即继承模板。</p>
            </div>
            <ElTag type="success" size="small">随环境配置保存</ElTag>
          </div>
          <div class="quick-grid">
            <ElFormItem label="副本数">
              <div class="inline-control">
                <ElSwitch
                  :model-value="form.replicas !== null && form.replicas !== undefined"
                  active-text="覆盖"
                  @change="form.replicas = $event ? (form.replicas ?? 1) : null"
                />
                <ElInputNumber
                  v-if="form.replicas !== null && form.replicas !== undefined"
                  v-model="form.replicas"
                  :min="0"
                  :max="10000"
                  controls-position="right"
                />
              </div>
            </ElFormItem>
            <ElFormItem label="启动命令">
              <ElInput
                :model-value="form.command.join('\n')"
                type="textarea"
                :rows="2"
                placeholder="每行一个参数，留空继承 YAML，例如 java"
                @update:model-value="form.command = splitLines($event)"
              />
            </ElFormItem>
            <ElFormItem label="启动参数">
              <ElInput
                :model-value="form.args.join('\n')"
                type="textarea"
                :rows="2"
                placeholder="每行一个，留空继承，例如 -jar"
                @update:model-value="form.args = splitLines($event)"
              />
            </ElFormItem>
          </div>
        </section>
        <ElTabs v-model="tab">
          <ElTabPane label="CPU / 内存" name="resources">
            <p class="help pane-hint"
              >只影响所选容器的运行资源，与 CI 构建资源无关；留空继承 YAML，关闭覆盖即恢复。</p
            >
            <ElSwitch
              :model-value="!!form.resources"
              aria-label="覆盖当前环境运行资源"
              active-text="覆盖当前环境运行资源"
              @change="toggleResources(!!$event)"
            />
            <div v-if="form.resources" class="fields">
              <ElFormItem label="CPU request"
                ><ElInput
                  v-model="form.resources.cpu_request"
                  placeholder="调度申请，例如 250m，留空继承"
              /></ElFormItem>
              <ElFormItem label="CPU limit"
                ><ElInput
                  v-model="form.resources.cpu_limit"
                  placeholder="上限，例如 1 表示 1 核"
              /></ElFormItem>
              <ElFormItem label="内存 request"
                ><ElInput
                  v-model="form.resources.memory_request"
                  placeholder="调度申请，例如 512Mi，留空继承"
              /></ElFormItem>
              <ElFormItem label="内存 limit"
                ><ElInput
                  v-model="form.resources.memory_limit"
                  placeholder="上限，例如 1Gi 或 4Gi"
              /></ElFormItem>
            </div>
            <p v-if="form.resources" class="help"
              >request 不超过 limit；容量请用 Mi / Gi，不要写成 1g。</p
            >
          </ElTabPane>
          <ElTabPane :label="`配置文件 (${form.files.length})`" name="files">
            <p class="help pane-hint"
              >普通文本文件会写入 ConfigMap；敏感文件请改用「密码与凭据」中的文件挂载。</p
            >
            <div class="pane-toolbar"
              ><ElButton size="small" @click="addFile"
                ><Plus :size="14" />添加文件</ElButton
              ></div
            >
            <ElEmpty v-if="!form.files.length" description="暂无配置文件" :image-size="40" />
            <article v-for="(file, i) in form.files" :key="i" class="asset-card">
              <div class="asset-head"
                ><span class="asset-index">{{ i + 1 }}</span
                ><span class="asset-title">{{ file.name || file.filename || '未命名文件' }}</span
                ><ElButton text type="danger" @click="form.files.splice(i, 1)">移除</ElButton></div
              >
              <div class="fields"
                ><ElFormItem label="标识"
                  ><ElInput v-model="file.name" placeholder="app-config" /></ElFormItem
                ><ElFormItem label="文件名"
                  ><ElInput v-model="file.filename" placeholder="application.yml" /></ElFormItem
                ><ElFormItem label="完整挂载路径"
                  ><ElInput
                    v-model="file.mount_path"
                    placeholder="/app/config/application.yml" /></ElFormItem
              ></div>
              <ElFormItem label="引用已有 ConfigMap"
                ><ElSelect
                  v-model="file.existing_config_map"
                  filterable
                  clearable
                  allow-create
                  default-first-option
                  placeholder="留空则由本服务创建"
                  @visible-change="loadChoices('configmap', $event)"
                  @change="file.content = ''"
                  ><ElOption
                    v-for="item in choices.configmap"
                    :key="item"
                    :value="item"
                    :label="item" /></ElSelect
              ></ElFormItem>
              <template v-if="!file.existing_config_map"
                ><label class="upload"
                  >导入本地文本文件<input
                    type="file"
                    :disabled="busy || !editable"
                    @change="uploadFile($event, file)" /></label
                ><ElInput
                  v-model="file.content"
                  type="textarea"
                  :rows="8"
                  spellcheck="false"
                  placeholder="编辑文件内容，按原文保存"
              /></template>
            </article>
          </ElTabPane>
          <ElTabPane :label="`密码与凭据 (${form.secrets.length})`" name="secrets">
            <p class="help pane-hint"
              >键值在凭据中加密保存、不回显；每个键可注入环境变量或挂载为文件。</p
            >
            <div class="pane-toolbar">
              <ElButton size="small" @click="openCredential()"
                ><Plus :size="14" />新建多键 Secret</ElButton
              >
              <ElButton size="small" plain @click="addSecret"
                ><Link :size="14" />引用已有凭据 / Secret</ElButton
              >
            </div>
            <ElEmpty v-if="!form.secrets.length" description="暂无凭据配置" :image-size="40" />
            <article v-for="group in secretGroups" :key="group.name" class="asset-card">
              <div class="secret-heading">
                <span class="secret-status"
                  ><LockKeyhole :size="14" />{{ group.name }} · {{ group.items.length }} 个键 ·
                  原值不回显</span
                >
                <div class="secret-actions">
                  <ElButton size="small" plain @click="openCredential(group.items[0])"
                    ><RefreshCw :size="14" />编辑键值与引用</ElButton
                  >
                  <ElButton size="small" text type="danger" @click="removeSecretGroup(group.name)"
                    >移除此组</ElButton
                  >
                </div>
              </div>
              <div v-for="item in group.items" :key="item.name" class="secret-key-summary">
                <code>{{ item.key }}</code
                ><span>→ {{ item.env_name || item.mount_path }}</span>
                <ElTag size="small" type="info">{{
                  item.env_name ? '环境变量' : '文件挂载'
                }}</ElTag>
              </div>
            </article>
            <article
              v-for="secret in form.secrets.filter((item) => !item.group || !item.credential_id)"
              :key="secret.name"
              class="asset-card"
            >
              <div class="secret-heading">
                <span class="secret-status"
                  ><LockKeyhole :size="14" />{{
                    secret.credential_id || secret.existing_secret
                      ? '已配置 · 不展示原值'
                      : '待配置'
                  }}</span
                >
                <div class="secret-actions">
                  <ElButton
                    v-if="secret.credential_id"
                    size="small"
                    plain
                    @click="openCredential(secret)"
                  >
                    <RefreshCw :size="14" />{{ secret.group ? '编辑此组键值' : '更换凭据' }}
                  </ElButton>
                  <ElTooltip content="解除此服务的引用，保留凭据和集群 Secret" placement="top">
                    <ElButton
                      size="small"
                      text
                      type="danger"
                      :aria-label="`解除密码引用 ${secret.name}`"
                      @click="form.secrets.splice(form.secrets.indexOf(secret), 1)"
                      ><Trash2 :size="14"
                    /></ElButton>
                  </ElTooltip>
                </div>
              </div>
              <div class="fields"
                ><ElFormItem label="标识"><ElInput v-model="secret.name" /></ElFormItem
                ><ElFormItem label="Secret 中的键"
                  ><ElInput v-model="secret.key" placeholder="password" /></ElFormItem
              ></div>
              <ElFormItem label="加密凭据"
                ><ElSelect
                  v-model="secret.credential_id"
                  filterable
                  clearable
                  placeholder="选择或新建凭据"
                  @visible-change="loadCredentials"
                  @change="secret.existing_secret = ''"
                  ><ElOption
                    v-for="item in credentials"
                    :key="item.id"
                    :value="item.id"
                    :label="item.name" /><ElOption
                    v-if="
                      secret.credential_id &&
                      !credentials.some((c) => c.id === secret.credential_id)
                    "
                    :value="secret.credential_id"
                    :label="`凭据 #${secret.credential_id}`" /></ElSelect
              ></ElFormItem>
              <ElFormItem v-if="secret.credential_id" label="读取凭据字段"
                ><ElSelect
                  v-model="secret.credential_key"
                  allow-create
                  filterable
                  default-first-option
                  ><ElOption
                    v-for="key in credentialKeys(secret)"
                    :key="key"
                    :value="key"
                    :label="key" /></ElSelect
              ></ElFormItem>
              <ElFormItem label="或引用已有 Secret"
                ><ElSelect
                  v-model="secret.existing_secret"
                  filterable
                  clearable
                  allow-create
                  default-first-option
                  @visible-change="loadChoices('secret', $event)"
                  @change="secret.credential_id = 0"
                  ><ElOption
                    v-for="item in choices.secret"
                    :key="item"
                    :value="item"
                    :label="item" /></ElSelect
              ></ElFormItem>
              <div class="fields"
                ><ElFormItem label="环境变量名"
                  ><ElInput
                    v-model="secret.env_name"
                    placeholder="DB_PASSWORD"
                    @input="secret.mount_path = ''" /></ElFormItem
                ><ElFormItem label="或挂载为文件"
                  ><ElInput
                    v-model="secret.mount_path"
                    placeholder="/app/secrets/password"
                    @input="secret.env_name = ''" /></ElFormItem
              ></div>
            </article>
          </ElTabPane>
          <ElTabPane :label="`域名路由 (${form.routes.length})`" name="routes">
            <p class="help pane-hint"
              >Ingress 指向模板中的 Service 端口；需集群已装 Ingress Controller，域名 DNS 另指入口地址。</p
            >
            <div class="pane-toolbar"
              ><ElButton size="small" @click="addRoute"
                ><Plus :size="14" />添加域名</ElButton
              ></div
            >
            <ElEmpty v-if="!form.routes.length" description="暂无域名路由" :image-size="40" />
            <article v-for="(route, i) in form.routes" :key="i" class="asset-card">
              <div class="asset-head"
                ><span class="asset-index">{{ i + 1 }}</span
                ><span class="asset-title">{{ route.host || '未填写域名' }}</span
                ><ElButton text type="danger" @click="form.routes.splice(i, 1)">移除</ElButton></div
              >
              <div class="fields"
                ><ElFormItem label="标识"><ElInput v-model="route.name" /></ElFormItem
                ><ElFormItem label="域名"
                  ><ElInput v-model="route.host" placeholder="api.example.com" /></ElFormItem
                ><ElFormItem label="路径"
                  ><ElInput v-model="route.path" placeholder="/" /></ElFormItem
              ></div>
              <div class="fields"
                ><ElFormItem label="目标 Service"
                  ><ElSelect v-model="route.service" @change="setRoutePort(route)"
                    ><ElOption
                      v-for="item in serviceOptions"
                      :key="item.name"
                      :value="item.name"
                      :label="item.name" /></ElSelect></ElFormItem
                ><ElFormItem label="Service 端口"
                  ><ElSelect v-model="route.port"
                    ><ElOption
                      v-for="port in routePorts(route)"
                      :key="port"
                      :value="port"
                      :label="String(port)" /></ElSelect></ElFormItem
              ></div>
              <IngressRouteFields v-model="form.routes[i]" />
            </article>
          </ElTabPane>
          <ElTabPane :label="`存储 (${form.volumes.length})`" name="volumes">
            <p class="help pane-hint"
              >留空则新建本服务专属 PVC；已有 PVC 不在此改写容量，解除挂载不会删除卷。</p
            >
            <div class="pane-toolbar"
              ><ElButton size="small" @click="addVolume"
                ><Plus :size="14" />添加挂载</ElButton
              ></div
            >
            <ElEmpty v-if="!form.volumes.length" description="暂无存储挂载" :image-size="40" />
            <article v-for="(volume, i) in form.volumes" :key="i" class="asset-card">
              <div class="asset-head"
                ><span class="asset-index">{{ i + 1 }}</span
                ><span class="asset-title">{{ volume.mount_path || '未填写挂载目录' }}</span
                ><ElTooltip content="解除挂载，保留 PVC" placement="top"
                  ><ElButton text type="danger" @click="form.volumes.splice(i, 1)"
                    >移除</ElButton
                  ></ElTooltip
                ></div
              >
              <div class="fields"
                ><ElFormItem label="标识"><ElInput v-model="volume.name" /></ElFormItem
                ><ElFormItem label="容器挂载目录"
                  ><ElInput v-model="volume.mount_path" placeholder="/data" /></ElFormItem
                ><ElFormItem label="只读"><ElSwitch v-model="volume.read_only" /></ElFormItem
              ></div>
              <ElFormItem label="已有 PVC"
                ><ElSelect
                  v-model="volume.existing_claim"
                  filterable
                  clearable
                  allow-create
                  default-first-option
                  placeholder="留空则新建"
                  @visible-change="loadChoices('pvc', $event)"
                  @change="clearVolumeSelection(volume)"
                  ><ElOption
                    v-for="item in choices.pvc"
                    :key="item"
                    :value="item"
                    :label="item" /></ElSelect
              ></ElFormItem>
              <div v-if="!volume.existing_claim" class="fields"
                ><ElFormItem label="容量"
                  ><ElInput v-model="volume.capacity" placeholder="10Gi" /></ElFormItem
                ><ElFormItem label="StorageClass"
                  ><ElInput
                    v-model="volume.storage_class"
                    placeholder="选择集群已有存储类名称" /></ElFormItem
                ><ElFormItem label="访问模式"
                  ><ElSelect v-model="volume.access_mode"
                    ><ElOption
                      v-for="mode in ['ReadWriteOnce', 'ReadOnlyMany', 'ReadWriteMany']"
                      :key="mode"
                      :value="mode"
                      :label="mode" /></ElSelect></ElFormItem
              ></div>
            </article>
          </ElTabPane>
        </ElTabs>
      </ElForm>
      <ElAlert v-if="error" :title="error" type="error" :closable="false" />
      <details v-if="dirty" class="changes" open
        ><summary>待保存修改</summary
        ><ElTable :data="changes" max-height="220"
          ><ElTableColumn prop="category" label="类型" /><ElTableColumn
            prop="name"
            label="项目" /><ElTableColumn prop="action" label="变更" /></ElTable
      ></details>
      <div v-if="!integrated" class="actions"
        ><ElButton :disabled="!dirty || busy" @click="reset">撤销修改</ElButton
        ><ElButton
          :disabled="!editable || !dirty || busy || variablesDirty"
          :loading="busy"
          @click="preview"
          >检查并预览</ElButton
        ><ElButton
          v-if="plan"
          type="primary"
          :disabled="!editable || busy || (environment.is_production && !confirmed)"
          @click="save"
          >保存配置，待发布</ElButton
        ></div
      >
      <p v-if="variablesDirty && !integrated" class="help"
        >请先保存上方变量修改，再检查配套配置。</p
      >
      <p v-if="integrated" class="help">与 YAML、变量一起保存；请点下方「预览与保存」。</p>
      <template v-if="plan"
        ><ElAlert
          title="预览只校验并生成配置，不会部署。文件或凭据变化后，下次发布才会更新 Pod 模板。"
          type="info"
          :closable="false"
        /><ElCheckbox v-if="environment.is_production" v-model="confirmed"
          >已核对生产环境、运行资源、文件、凭据引用、域名及存储</ElCheckbox
        ><details
          ><summary>查看生成的部署 YAML（无密码正文）</summary
          ><pre class="manifest">{{ plan.manifest }}</pre>
        </details></template
      >
    </template>
    <ElDialog
      v-model="credentialVisible"
      class="service-credential-dialog"
      :title="credentialTarget ? '更换凭据' : '新建加密凭据'"
      width="min(820px, calc(100vw - 32px))"
      append-to-body
      :close-on-click-modal="false"
      :close-on-press-escape="!credentialBusy"
      :show-close="!credentialBusy"
      @close="clearCredential"
    >
      <div class="credential-context"
        ><strong>{{ service.name }}</strong
        ><span>{{ environment.name }} · {{ environment.namespace }}</span
        ><ElTag v-if="environment.is_production" type="danger" size="small">生产</ElTag></div
      >
      <ElAlert
        v-if="credentialTarget"
        :title="
          credentialTarget.group
            ? '保存为独立凭据版本，仅更新当前草稿；留空保留原值，发布后生效。'
            : '保存为独立凭据并替换草稿引用；预览、保存并发布后生效。'
        "
        type="info"
        :closable="false"
      />
      <ElForm label-position="top" :disabled="credentialBusy"
        ><ElFormItem label="凭据名称"
          ><ElInput v-model="credentialDraft.name" maxlength="64" /></ElFormItem
        ><ElFormItem label="内容类型"
          ><ElRadioGroup v-model="credentialMode" :disabled="!!credentialTarget?.group"
            ><ElRadioButton value="fields">多个键值</ElRadioButton
            ><ElRadioButton value="value">单个密钥</ElRadioButton
            ><ElRadioButton value="account">账号密码</ElRadioButton
            ><ElRadioButton value="file">敏感文件</ElRadioButton></ElRadioGroup
          ></ElFormItem
        ><div v-if="credentialMode === 'fields'" class="credential-fields">
          <p class="help"
            >支持密码、Token、证书、私钥等；每个键可注入环境变量或挂载为文件，内容按原文加密保存。</p
          >
          <article v-for="(field, i) in credentialFields" :key="field.id" class="credential-field">
            <div class="fields">
              <ElFormItem label="键名" required
                ><ElInput v-model="field.key" placeholder="例如 API_TOKEN / tls.key"
              /></ElFormItem>
              <ElFormItem label="内容格式"
                ><ElSelect v-model="field.multiline"
                  ><ElOption :value="false" label="单行密钥" /><ElOption
                    :value="true"
                    label="多行 / 文件" /></ElSelect
              ></ElFormItem>
            </div>
            <ElFormItem
              :label="field.originalKey ? '新内容（留空保留原值）' : '内容'"
              :required="!field.originalKey"
            >
              <ElInput
                v-if="field.multiline"
                v-model="field.value"
                type="textarea"
                :rows="4"
                :placeholder="
                  field.originalKey
                    ? '已加密保存，不回显；留空保留'
                    : '按原文输入私钥、证书或敏感文件'
                "
              />
              <ElInput
                v-else
                v-model="field.value"
                type="password"
                show-password
                autocomplete="new-password"
                :placeholder="field.originalKey ? '已加密保存，不回显；留空保留' : '输入密钥内容'"
              />
            </ElFormItem>
            <div class="fields">
              <ElFormItem label="使用方式"
                ><ElSelect v-model="field.usage"
                  ><ElOption value="env" label="环境变量" /><ElOption
                    value="file"
                    label="文件挂载" /></ElSelect
              ></ElFormItem>
              <ElFormItem :label="field.usage === 'file' ? '容器内文件路径' : '环境变量名'" required
                ><ElInput
                  v-model="field.destination"
                  :placeholder="
                    field.usage === 'file' ? '/app/secrets/private-key.pem' : 'API_TOKEN'
                  "
              /></ElFormItem>
            </div>
            <ElButton
              text
              type="danger"
              :disabled="credentialFields.length <= 1"
              @click="credentialFields.splice(i, 1)"
              >移除此键</ElButton
            >
          </article>
          <ElButton plain :disabled="credentialFields.length >= 30" @click="addCredentialField"
            >+ 添加键值</ElButton
          > </div
        ><template v-else-if="credentialMode === 'account'"
          ><ElFormItem label="账号"
            ><ElInput v-model="credentialDraft.username" autocomplete="off" /></ElFormItem
          ><ElFormItem label="密码"
            ><ElInput
              v-model="credentialDraft.password"
              type="password"
              show-password
              autocomplete="new-password" /></ElFormItem></template
        ><ElFormItem v-else-if="credentialMode === 'value'" label="密码 / Token / 密钥" required>
          <ElInput
            v-model="credentialDraft.value"
            type="password"
            show-password
            autocomplete="new-password"
          /> </ElFormItem
        ><ElFormItem v-else label="敏感文件内容" required
          ><ElInput v-model="credentialDraft.value" type="textarea" :rows="8" /></ElFormItem
      ></ElForm>
      <ElAlert v-if="credentialError" :title="credentialError" type="error" :closable="false" />
      <template #footer
        ><ElButton :disabled="credentialBusy" @click="credentialVisible = false">取消</ElButton
        ><ElButton type="primary" :loading="credentialBusy" @click="createCredential">{{
          credentialTarget ? '保存新凭据并替换引用' : '加密保存并引用'
        }}</ElButton></template
      >
    </ElDialog>
  </section>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Link, LockKeyhole, Plus, RefreshCw, Trash2 } from 'lucide-vue-next'
  import * as yaml from 'js-yaml'
  import IngressRouteFields from './IngressRouteFields.vue'
  import { validateIngressRoute } from '@/utils/template-configuration'
  import { configValues } from '@/utils/service-environment-config'
  import {
    createCredentialApi,
    getCredentialApi,
    deriveSecretCredentialApi,
    listCredentialsApi,
    type AxiOpsCredential,
    type AxiOpsEnvironment,
    type AxiOpsService,
    type ServiceEnvironmentConfig
  } from '@/api/axiops'
  import {
    emptyServiceAssets,
    saveServiceAssetsApi,
    type ServiceAssets
  } from '@/api/axiops/service-assets'
  import { listEnvConfigsApi, listEnvPVCsApi } from '@/api/axiops/env-resource'
  const props = defineProps<{
    config: ServiceEnvironmentConfig
    service: AxiOpsService
    environment: AxiOpsEnvironment
    editable: boolean
    variablesDirty: boolean
    integrated?: boolean
  }>()
  const emit = defineEmits<{ saved: []; busy: [value: boolean] }>()
  const clone = <T,>(v: T): T => JSON.parse(JSON.stringify(v))
  const normalize = (v?: ServiceAssets) =>
    clone({
      ...emptyServiceAssets(),
      ...v,
      command: v?.command || [],
      args: v?.args || [],
      files: v?.files || [],
      secrets: v?.secrets || [],
      routes: (v?.routes || []).map((route) => ({
        ...route,
        path_type: route.path_type || 'Prefix',
        tls_enabled: route.tls_enabled || !!route.tls_secret,
        tls_hosts: route.tls_hosts?.length
          ? route.tls_hosts
          : route.tls_enabled || route.tls_secret
            ? [route.host]
            : []
      })),
      volumes: v?.volumes || []
    })
  const form = ref<ServiceAssets>(normalize(props.config.assets)),
    baseline = ref(clone(form.value)),
    tab = ref('resources'),
    busy = ref(false),
    error = ref(''),
    confirmed = ref(false)
  const plan = ref<Awaited<ReturnType<typeof saveServiceAssetsApi>> | null>(null),
    plannedBody = ref('')
  let generation = 0
  const dirty = computed(
    () => credentialVisible.value || JSON.stringify(form.value) !== JSON.stringify(baseline.value)
  )
  const splitLines = (value: string) =>
    value
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean)
  const getAssets = () => {
    form.value.routes.forEach(validateIngressRoute)
    return clone(form.value)
  }
  const body = () => ({ token: props.config.token, assets: getAssets() })
  const toggleResources = (enabled: boolean) => {
    if (enabled)
      form.value.resources = {
        cpu_request: '',
        cpu_limit: '',
        memory_request: '',
        memory_limit: ''
      }
    else delete form.value.resources
  }
  const documents = computed(() => {
    try {
      const out: any[] = []
      const values = configValues(props.config)
      const source = (props.config.manifest || props.config.definition)
        .replace(/\{\{\s*\.?Service\s*\}\}|\$Service\$/g, props.service.code)
        .replace(
          /\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_-]*)\$/g,
          (_, key, legacyKey) => values[key || legacyKey]?.value || 'placeholder'
        )
      yaml.loadAll(source, (doc) => {
        if (doc) out.push(doc)
      })
      return out
    } catch {
      return []
    }
  })
  const workloadOptions = computed(() =>
    documents.value
      .filter((item) =>
        ['Deployment', 'StatefulSet', 'DaemonSet', 'Job', 'CronJob'].includes(item.kind)
      )
      .map((item) => ({ key: `${item.kind}/${item.metadata?.name}`, item }))
  )
  const selectedWorkload = computed({
    get: () =>
      form.value.workload_name ? `${form.value.workload_kind}/${form.value.workload_name}` : '',
    set: (value) => {
      const [kind, name] = value.split('/')
      form.value.workload_kind = kind || ''
      form.value.workload_name = name || ''
      form.value.container = ''
    }
  })
  const containerOptions = computed(() => {
    const match =
      workloadOptions.value.find((x) => x.key === selectedWorkload.value) ||
      (workloadOptions.value.length === 1 ? workloadOptions.value[0] : undefined)
    const spec = match?.item?.spec
    return (
      spec?.template?.spec?.containers ||
      spec?.jobTemplate?.spec?.template?.spec?.containers ||
      []
    ).map((c: any) => c.name) as string[]
  })
  const serviceOptions = computed(() =>
    documents.value
      .filter((item) => item.kind === 'Service')
      .map((item) => ({
        name: item.metadata?.name as string,
        ports: (item.spec?.ports || [])
          .map((p: any) => Number(p.port))
          .filter(Number.isFinite) as number[]
      }))
  )
  const changes = computed(() => {
    const out: Array<{ category: string; name: string; action: string }> = []
    if (JSON.stringify(form.value.resources) !== JSON.stringify(baseline.value.resources))
      out.push({
        category: '运行资源',
        name: 'CPU / 内存',
        action: form.value.resources ? '修改当前环境覆盖' : '恢复 YAML 定义'
      })
    if (form.value.replicas !== baseline.value.replicas)
      out.push({
        category: '发布快捷设置',
        name: '副本数',
        action: form.value.replicas == null ? '恢复模板定义' : '修改当前环境覆盖'
      })
    if (JSON.stringify(form.value.command) !== JSON.stringify(baseline.value.command))
      out.push({
        category: '发布快捷设置',
        name: '启动命令',
        action: form.value.command.length ? '修改当前环境覆盖' : '恢复模板定义'
      })
    if (JSON.stringify(form.value.args) !== JSON.stringify(baseline.value.args))
      out.push({
        category: '发布快捷设置',
        name: '启动参数',
        action: form.value.args.length ? '修改当前环境覆盖' : '恢复模板定义'
      })
    for (const [key, label] of [
      ['files', '配置文件'],
      ['secrets', '凭据引用'],
      ['routes', '域名路由'],
      ['volumes', '存储']
    ] as const) {
      const before = new Map(baseline.value[key].map((item) => [item.name, JSON.stringify(item)])),
        after = new Map(form.value[key].map((item) => [item.name, JSON.stringify(item)]))
      for (const name of new Set([...before.keys(), ...after.keys()]))
        if (before.get(name) !== after.get(name))
          out.push({
            category: label,
            name: name || '未命名',
            action: !before.has(name)
              ? '新增'
              : !after.has(name)
                ? '解除配置（保留集群资源）'
                : '修改'
          })
    }
    if (
      form.value.container !== baseline.value.container ||
      selectedWorkload.value !==
        (baseline.value.workload_name
          ? `${baseline.value.workload_kind}/${baseline.value.workload_name}`
          : '')
    )
      out.push({ category: '注入目标', name: selectedWorkload.value || '自动识别', action: '修改' })
    return out
  })
  function reset() {
    form.value = clone(baseline.value)
    credentialVisible.value = false
    clearCredential()
    error.value = ''
  }
  function addFile() {
    form.value.files.push({
      name: nextAssetName('file'),
      filename: 'application.yml',
      content: '',
      mount_path: '',
      existing_config_map: ''
    })
  }
  function nextAssetName(prefix: string) {
    const used = new Set(
      [...form.value.files, ...form.value.secrets, ...form.value.routes, ...form.value.volumes].map(
        (item) => item.name
      )
    )
    let i = 1
    while (
      used.has(`${prefix}-${i}`) ||
      form.value.secrets.some((item) => item.group === `${prefix}-${i}`)
    )
      i++
    return `${prefix}-${i}`
  }
  function newSecret(): ServiceAssets['secrets'][number] {
    return {
      name: nextAssetName('secret'),
      credential_id: 0,
      credential_key: 'password',
      existing_secret: '',
      key: 'password',
      env_name: '',
      mount_path: ''
    }
  }
  function addSecret() {
    form.value.secrets.push(newSecret())
    void loadCredentials(true)
  }
  function addRoute() {
    const item = serviceOptions.value[0]
    form.value.routes.push({
      name: nextAssetName('route'),
      host: '',
      path: '/',
      service: item?.name || '',
      port: item?.ports[0] || 80,
      class: '',
      tls_secret: '',
      path_type: 'Prefix',
      tls_enabled: false,
      tls_hosts: []
    })
  }
  const routePorts = (route: ServiceAssets['routes'][number]) =>
    serviceOptions.value.find((item) => item.name === route.service)?.ports || []
  function setRoutePort(route: ServiceAssets['routes'][number]) {
    route.port = serviceOptions.value.find((item) => item.name === route.service)?.ports[0] || 80
  }
  function addVolume() {
    form.value.volumes.push({
      name: nextAssetName('data'),
      existing_claim: '',
      capacity: '',
      storage_class: '',
      access_mode: 'ReadWriteOnce',
      mount_path: '',
      read_only: false
    })
  }
  function clearVolumeSelection(volume: ServiceAssets['volumes'][number]) {
    volume.capacity = ''
    volume.storage_class = ''
  }
  async function uploadFile(event: Event, file: ServiceAssets['files'][number]) {
    const input = event.target as HTMLInputElement,
      selected = input.files?.[0],
      request = generation
    if (!selected) return
    if (selected.size > 256 * 1024) {
      ElMessage.warning('普通配置文件不能超过256KiB')
      return
    }
    const text = await selected.text()
    if (request === generation && form.value.files.includes(file)) {
      file.filename = selected.name
      file.content = text
    }
    input.value = ''
  }
  const choices = ref({ configmap: [] as string[], secret: [] as string[], pvc: [] as string[] }),
    credentials = ref<AxiOpsCredential[]>([])
  async function loadChoices(kind: 'configmap' | 'secret' | 'pvc', visible: boolean) {
    if (!visible) return
    const request = generation,
      eid = props.environment.id
    try {
      const result = kind === 'pvc' ? await listEnvPVCsApi(eid) : await listEnvConfigsApi(eid, kind)
      if (request === generation)
        choices.value[kind] = (result || [])
          .filter((r) => r.namespace === props.environment.namespace)
          .map((r) => r.name)
    } catch {
      if (request === generation) error.value = '集群候选资源读取失败，可重试或填写已确认的资源名称'
    }
  }
  async function loadCredentials(visible = true) {
    if (!visible) return
    const request = generation
    try {
      const rows = await listCredentialsApi({
        type: 'secret',
        project_id: props.service.project_id
      })
      if (request === generation) credentials.value = rows || []
    } catch {
      if (request === generation) error.value = '凭据列表读取失败，请检查权限后重试'
    }
  }
  async function preview() {
    if (busy.value || !dirty.value || props.variablesDirty || !props.editable) return
    busy.value = true
    error.value = ''
    plan.value = null
    const request = generation
    try {
      const payload = body(),
        serialized = JSON.stringify(payload)
      const result = await saveServiceAssetsApi(
        props.environment.id,
        props.service.id,
        payload,
        true
      )
      if (request === generation && serialized === JSON.stringify(body())) {
        plan.value = result
        plannedBody.value = serialized
      }
    } catch (e: any) {
      if (request === generation) error.value = e?.message || '校验失败'
    } finally {
      if (request === generation) busy.value = false
    }
  }
  async function save() {
    if (
      !plan.value ||
      busy.value ||
      !props.editable ||
      (props.environment.is_production && !confirmed.value) ||
      plannedBody.value !== JSON.stringify(body())
    )
      return
    const request = generation
    busy.value = true
    try {
      const result = await saveServiceAssetsApi(
        props.environment.id,
        props.service.id,
        JSON.parse(plannedBody.value),
        false
      )
      if (request !== generation) return
      baseline.value = clone(normalize(result.config.assets))
      form.value = clone(baseline.value)
      ElMessage.success('服务配置已保存，下一次提交部署时生效')
      emit('saved')
    } catch (e: any) {
      if (request === generation) {
        error.value = e?.message || '保存失败'
        plan.value = null
      }
    } finally {
      if (request === generation) busy.value = false
    }
  }
  async function confirmLeave() {
    if (busy.value || credentialBusy.value) return false
    if (!dirty.value) return true
    try {
      await ElMessageBox.confirm('服务配套配置尚未保存，是否放弃修改？', '未保存配置', {
        confirmButtonText: '放弃修改',
        cancelButtonText: '继续编辑',
        type: 'warning'
      })
      reset()
      return true
    } catch {
      return false
    }
  }
  const credentialVisible = ref(false),
    credentialBusy = ref(false),
    credentialError = ref(''),
    credentialMode = ref('value'),
    credentialDraft = ref({ name: '', username: '', password: '', value: '' })
  const credentialTarget = ref<ServiceAssets['secrets'][number] | null>(null)
  const secretGroups = computed(() => {
    const groups = new Map<string, ServiceAssets['secrets']>()
    for (const item of form.value.secrets) {
      if (!item.group || !item.credential_id) continue
      const entries = groups.get(item.group) || []
      entries.push(item)
      groups.set(item.group, entries)
    }
    return Array.from(groups, ([name, items]) => ({ name, items }))
  })
  async function removeSecretGroup(name: string) {
    if (!props.editable || busy.value || credentialBusy.value) return
    const request = generation
    try {
      await ElMessageBox.confirm(
        '移除此组所有键的引用？保存并发布后解除引用，保留原凭据和集群 Secret。',
        '移除 Secret 引用',
        { type: 'warning', confirmButtonText: '移除引用', cancelButtonText: '取消' }
      )
      if (request === generation)
        form.value.secrets = form.value.secrets.filter((item) => item.group !== name)
    } catch {
      /* cancelled */
    }
  }
  type CredentialField = {
    id: number
    key: string
    value: string
    originalKey: string
    name: string
    multiline: boolean
    usage: 'env' | 'file'
    destination: string
  }
  const credentialFields = ref<CredentialField[]>([]),
    credentialSource = ref<AxiOpsCredential | null>(null),
    credentialOriginalKeys = ref<string[]>([])
  let fieldSequence = 0
  function addCredentialField() {
    credentialFields.value.push({
      id: ++fieldSequence,
      key: '',
      value: '',
      originalKey: '',
      name: '',
      multiline: false,
      usage: 'env',
      destination: ''
    })
  }
  function credentialKeys(secret: ServiceAssets['secrets'][number]) {
    return (
      credentials.value.find((item) => item.id === secret.credential_id)?.keys || [
        'username',
        'password',
        'value'
      ]
    )
  }
  async function openCredential(secret: ServiceAssets['secrets'][number] | null = null) {
    if (!props.editable || busy.value || credentialBusy.value) return
    clearCredential()
    credentialTarget.value = secret
    credentialMode.value = secret && !secret.group ? 'value' : 'fields'
    credentialDraft.value.name =
      `${props.service.code}-${props.environment.code}-${secret?.name || 'password'}`.slice(0, 45) +
      `-${Date.now()}`
    credentialVisible.value = true
    if (!secret) addCredentialField()
    if (secret?.group) {
      const entries = form.value.secrets.filter((item) => item.group === secret.group)
      credentialFields.value = entries.map((item) => ({
        id: ++fieldSequence,
        key: item.key,
        value: '',
        originalKey: item.credential_key,
        name: item.name,
        multiline: !!item.mount_path,
        usage: item.mount_path ? 'file' : 'env',
        destination: item.mount_path || item.env_name
      }))
      credentialOriginalKeys.value = entries.map((item) => item.credential_key)
      credentialBusy.value = true
      const request = generation
      try {
        const source = await getCredentialApi(secret.credential_id)
        if (request !== generation || credentialTarget.value !== secret) return
        if (source.type !== 'secret' || !source.token || !source.keys?.length)
          throw Error('无法读取凭据字段，请重新加载')
        credentialSource.value = source
      } catch (e: any) {
        if (request === generation) credentialError.value = e?.message || '凭据字段加载失败'
      } finally {
        if (request === generation) credentialBusy.value = false
      }
    }
  }
  function clearCredential() {
    credentialDraft.value = { name: '', username: '', password: '', value: '' }
    credentialTarget.value = null
    credentialError.value = ''
    credentialFields.value = []
    credentialSource.value = null
    credentialOriginalKeys.value = []
  }
  async function createCredential() {
    if (
      credentialBusy.value ||
      busy.value ||
      !props.editable ||
      !props.config.bound ||
      !credentialVisible.value
    )
      return
    const draft = { ...credentialDraft.value },
      mode = credentialMode.value,
      target = credentialTarget.value
    if (mode === 'fields') {
      await saveCredentialFields(draft.name.trim(), target)
      return
    }
    if (
      !draft.name.trim() ||
      (mode === 'account' ? !draft.username || !draft.password : !draft.value)
    ) {
      credentialError.value = '请填写名称和完整内容'
      return
    }
    credentialBusy.value = true
    const request = generation
    try {
      const data =
        mode === 'account'
          ? { username: draft.username, password: draft.password }
          : { [target?.credential_key || 'value']: draft.value }
      const field =
        mode === 'account'
          ? target?.credential_key === 'username'
            ? 'username'
            : 'password'
          : target?.credential_key || 'value'
      const result = await createCredentialApi({
        name: draft.name.trim(),
        type: 'secret',
        description: `${props.service.name} / ${props.environment.name}`,
        project_id: props.service.project_id,
        data: JSON.stringify(data)
      })
      if (request !== generation) return
      if (target && !form.value.secrets.includes(target)) return
      const item = target || newSecret()
      if (!target) form.value.secrets.push(item)
      item.credential_id = result.id
      item.credential_key = field
      item.existing_secret = ''
      credentials.value.push({
        id: result.id,
        name: draft.name.trim(),
        type: 'secret',
        project_id: props.service.project_id
      } as AxiOpsCredential)
      credentialVisible.value = false
      clearCredential()
      ElMessage.success(
        target
          ? '新凭据已加密保存，此条引用待预览、保存并发布'
          : '凭据已加密保存，请配置注入位置并预览保存'
      )
    } catch (e: any) {
      if (request === generation) credentialError.value = e?.message || '凭据保存失败'
    } finally {
      if (request === generation) credentialBusy.value = false
    }
  }
  async function saveCredentialFields(
    name: string,
    target: ServiceAssets['secrets'][number] | null
  ) {
    const fields = clone(credentialFields.value),
      source = credentialSource.value
    const old = target?.group
      ? form.value.secrets.filter((item) => item.group === target.group)
      : []
    const others = form.value.secrets.filter((item) => !old.includes(item))
    const keys = new Set<string>(),
      destinations = new Set<string>()
    try {
      if (
        !name ||
        name.length > 64 ||
        !fields.length ||
        fields.length > 30 ||
        others.length + fields.length > 30
      )
        throw Error('请填写64字以内的名称和1至30个键值')
      if (target && (!source?.token || source.id !== target.credential_id))
        throw Error('凭据字段尚未加载，请重新打开后重试')
      for (const field of fields) {
        if (
          !/^[A-Za-z0-9._-]+$/.test(field.key) ||
          field.key === '.' ||
          field.key === '..' ||
          field.key.length > 128 ||
          keys.has(field.key)
        )
          throw Error('键名只能包含字母、数字、点、横线和下划线，且不能重复')
        keys.add(field.key)
        if (!field.value && (!field.originalKey || field.key !== field.originalKey))
          throw Error('新增键或修改键名后需要填写内容')
        if (
          field.usage === 'env'
            ? !/^[A-Za-z_][A-Za-z0-9_]*$/.test(field.destination)
            : !field.destination.startsWith('/') ||
              field.destination.endsWith('/') ||
              /[\\\r\n]/.test(field.destination) ||
              field.destination.includes('\u0000') ||
              field.destination.includes('//') ||
              field.destination.split('/').some((part) => part === '.' || part === '..')
        )
          throw Error('请填写有效环境变量名或容器内绝对文件路径')
        const destination = `${field.usage}:${field.destination}`
        if (
          destinations.has(destination) ||
          others.some((item) =>
            field.usage === 'env'
              ? item.env_name === field.destination
              : item.mount_path === field.destination
          )
        )
          throw Error('注入位置不能重复')
        destinations.add(destination)
        if (field.usage === 'file') {
          const paths = [
            ...others.map((item) => item.mount_path),
            ...form.value.files.map((item) => item.mount_path),
            ...form.value.volumes.map((item) => item.mount_path),
            ...fields
              .filter((item) => item.id !== field.id && item.usage === 'file')
              .map((item) => item.destination)
          ].filter(Boolean)
          if (
            paths.some(
              (path) =>
                path === field.destination ||
                path.startsWith(field.destination + '/') ||
                field.destination.startsWith(path + '/')
            )
          )
            throw Error('文件挂载与其他配置文件或存储路径冲突，请调整路径')
        }
      }
    } catch (e: any) {
      credentialError.value = e.message
      return
    }
    credentialBusy.value = true
    const request = generation
    try {
      const values: Record<string, string> = Object.fromEntries(
        fields.filter((field) => field.value !== '').map((field) => [field.key, field.value])
      )
      const retain = fields.filter((field) => field.value === '').map((field) => field.originalKey)
      const retained = [
        ...retain,
        ...(source?.keys || []).filter(
          (key) => !credentialOriginalKeys.value.includes(key) && !(key in values)
        )
      ]
      const result = source
        ? await deriveSecretCredentialApi(source.id, {
            name,
            token: source.token!,
            retain: retained,
            values
          })
        : await createCredentialApi({
            name,
            type: 'secret',
            description: `${props.service.name} / ${props.environment.name}`,
            project_id: props.service.project_id,
            data: JSON.stringify(values)
          })
      if (
        request !== generation ||
        !credentialVisible.value ||
        (target && !form.value.secrets.includes(target))
      )
        return
      const group = target?.group || nextAssetName('secret')
      const used = new Set(
        [
          ...form.value.files,
          ...others,
          ...form.value.routes,
          ...form.value.volumes,
          ...fields.filter((field) => field.name)
        ].map((item) => item.name)
      )
      const entries = fields.map((field) => {
        let reference = field.name
        if (!reference) {
          let i = 1
          while (used.has(`secret-key-${i}`)) i++
          reference = `secret-key-${i}`
          used.add(reference)
        }
        return {
          name: reference,
          group,
          credential_id: result.id,
          credential_key: field.key,
          existing_secret: '',
          key: field.key,
          env_name: field.usage === 'env' ? field.destination : '',
          mount_path: field.usage === 'file' ? field.destination : ''
        }
      })
      form.value.secrets = [...others, ...entries]
      credentials.value.push({
        id: result.id,
        name,
        type: 'secret',
        project_id: props.service.project_id,
        keys: [...new Set([...retained, ...Object.keys(values)])]
      } as AxiOpsCredential)
      credentialVisible.value = false
      clearCredential()
      ElMessage.success('Secret键值已加密保存到新版本；请预览、保存当前环境并发布')
    } catch (e: any) {
      if (request === generation) credentialError.value = e?.message || 'Secret保存失败'
    } finally {
      if (request === generation) credentialBusy.value = false
    }
  }
  watch(
    form,
    () => {
      plan.value = null
      plannedBody.value = ''
      confirmed.value = false
    },
    { deep: true, flush: 'sync' }
  )
  watch([busy, credentialBusy], () => emit('busy', busy.value || credentialBusy.value), {
    flush: 'sync'
  })
  watch(
    () => [props.config.token, props.environment.id, props.service.id],
    () => {
      generation++
      busy.value = false
      plan.value = null
      confirmed.value = false
      form.value = normalize(props.config.assets)
      baseline.value = clone(form.value)
      choices.value = { configmap: [], secret: [], pvc: [] }
      credentials.value = []
      credentialBusy.value = false
      credentialVisible.value = false
      clearCredential()
    },
    { flush: 'sync' }
  )
  onBeforeUnmount(() => {
    generation++
    clearCredential()
    emit('busy', false)
  })
  defineExpose({ dirty, confirmLeave, discardChanges: reset, getAssets })
</script>

<style scoped>
  .service-resources {
    padding: 2px 0 0;
  }
  .resources-heading {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    padding-bottom: 14px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .resources-heading strong {
    display: block;
    font-family: var(--ax-font-display);
    font-size: 17px;
    font-weight: 600;
    letter-spacing: 0.01em;
    color: var(--el-text-color-primary);
  }
  .resources-heading p,
  .help {
    margin: 2px 0 0;
    font-size: 13px;
    line-height: 1.7;
    color: var(--el-text-color-secondary);
  }
  .help b {
    font-weight: 600;
    color: var(--el-text-color-regular);
  }
  .help.lead {
    margin: 12px 0 4px;
  }
  .pane-hint {
    margin: 0 0 12px;
  }
  .pane-toolbar {
    margin-bottom: 12px;
  }
  .asset-head {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 14px;
  }
  .asset-index {
    flex: 0 0 auto;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
    color: var(--el-color-primary);
    font-family: var(--ax-font-mono);
    font-size: 12px;
    font-weight: 600;
  }
  .asset-title {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    font-size: 14px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .target-grid,
  .fields {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 12px;
  }
  .quick-settings {
    margin: 16px 0;
    padding: 14px 16px 4px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--ax-radius-md);
    background: var(--el-fill-color-extra-light);
  }
  .quick-heading strong {
    font-family: var(--ax-font-display);
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  .quick-heading,
  .inline-control {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .quick-grid {
    display: grid;
    grid-template-columns: minmax(150px, 0.7fr) repeat(2, minmax(220px, 1fr));
    gap: 12px;
  }
  .inline-control {
    justify-content: flex-start;
  }
  @media (max-width: 900px) {
    .quick-grid {
      grid-template-columns: 1fr;
    }
  }
  .asset-card {
    padding: 16px;
    margin: 12px 0;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--ax-radius-md);
    background: var(--ax-surface-sunken);
  }
  .secret-toolbar,
  .secret-heading,
  .secret-actions,
  .secret-status,
  .credential-context {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }
  .secret-heading {
    justify-content: space-between;
    margin-bottom: 12px;
  }
  .secret-status {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
  .credential-context {
    margin-bottom: 16px;
    font-size: 13px;
  }
  .credential-fields {
    min-width: 0;
  }
  :global(.el-dialog.service-credential-dialog) {
    display: flex;
    flex-direction: column;
    max-height: 92dvh;
    margin: 4dvh auto !important;
  }
  :global(.service-credential-dialog .el-dialog__body) {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
  }
  :global(.service-credential-dialog .el-dialog__header),
  :global(.service-credential-dialog .el-dialog__footer) {
    flex-shrink: 0;
  }
  .credential-field {
    padding: 16px;
    margin-bottom: 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-fill-color-extra-light);
  }
  .secret-key-summary {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 12px;
    padding: 8px 0;
    border-top: 1px solid var(--el-border-color-lighter);
    font-size: 13px;
    overflow-wrap: anywhere;
  }
  .secret-key-summary span {
    color: var(--el-text-color-secondary);
  }
  .asset-card .el-select,
  .target-grid .el-select {
    width: 100%;
  }
  .upload {
    display: flex;
    align-items: center;
    gap: 12px;
    margin: 12px 0;
    font-size: 13px;
  }
  .actions {
    display: flex;
    gap: 8px;
    margin: 16px 0;
  }
  .manifest {
    padding: 16px;
    max-height: 480px;
    overflow: auto;
    background: #101827;
    color: #e2e8f0;
    font: 12px/1.7 var(--ax-font-mono);
    white-space: pre-wrap;
  }
  summary {
    cursor: pointer;
    padding: 10px 0;
  }
</style>

<template>
  <section v-if="service" class="configuration-panel">
    <div class="context-bar">
      <ElSelect
        :model-value="environmentId"
        :disabled="busy || syncVisible || assetsVisible || assetsEnvironmentVisible"
        size="small"
        class="environment-select"
        aria-label="选择环境"
        @change="switchEnvironment"
      >
        <ElOption v-if="service.delivery_type === 'host'" :value="0" label="主机配置" />
        <ElOption
          v-for="env in service.delivery_type === 'host' ? [] : environments"
          :key="env.id"
          :value="env.id"
          :label="`${env.name}${env.is_production ? ' · 生产' : ''}`"
        />
      </ElSelect>
      <span class="context-hint">{{
        environment
          ? `${environment.namespace || environment.host_group_ref} · ${config?.bound ? '当前环境配置' : '尚未配置'}`
          : '当前服务基础配置'
      }}</span>
      <ElButton
        v-if="service.delivery_type === 'k8s_yaml'"
        size="small"
        :disabled="busy || !!error"
        @click="openAssets"
        >配套配置<span v-if="resourcesEditor?.dirty"> · 未保存</span></ElButton
      >
      <ElButton
        v-if="
          environmentId &&
          config?.bound &&
          environments.length &&
          service.delivery_type === 'k8s_yaml'
        "
        size="small"
        :disabled="busy || !!error"
        @click="openSync"
        >同步配置</ElButton
      >
      <ElButton v-if="canViewBuild && environmentId" text size="small" @click="emit('build-config')"
        >构建配置（共享）</ElButton
      >
    </div>
    <ElAlert v-if="environmentId && error" type="error" :title="error" :closable="false">
      <ElButton text @click="loadConfig">Reload environment</ElButton>
    </ElAlert>
    <ElAlert
      v-if="service.delivery_type !== 'host' && !environments.length"
      title="请先创建环境并绑定集群和命名空间，再配置服务。"
      type="info"
      :closable="false"
    />
    <ServiceWorkbenchPanel
      v-if="
        service.delivery_type !== 'helm' &&
        ((service.delivery_type === 'host' && !environmentId) || config)
      "
      ref="sharedEditor"
      v-bind="$props"
      :environment-mode="!!environmentId"
      :environment-config="config"
      :can-edit="environmentId ? !!config && (canUpdate(environmentId) || canEdit) : canEdit"
      @import="emit('import')"
      @template-import="emit('template-import')"
      @build-config="emit('build-config')"
      @environment-config="openSync"
      @file-change="emit('file-change', $event)"
      @saved="emit('saved')"
      @environment-save="saveEnvironmentFromWorkbench"
      @join-environment="joinCurrent"
    />
    <HelmServicePanel
      v-if="service.delivery_type === 'helm' && config && environment"
      ref="helmEditor"
      :service="service"
      :config="config"
      :environment="environment"
      :environments="environments"
      :active-file="activeFile"
      :can-edit="canUpdate(environmentId) || !!canEdit"
      :can-edit-chart="!!canEdit"
      @saved="helmSaved"
      @build-config="emit('build-config')"
      @file-change="emit('file-change', $event)"
      @files-change="emit('files-change', $event)"
    />
    <ServiceTemplatePicker
      v-if="service.delivery_type === 'k8s_yaml'"
      v-model:visible="environmentCreationVisible"
      :project-id="service.project_id"
      kind="k8s_yaml"
      :environments="environments"
      :preferred-environment-id="environmentId"
      :service="service"
      :existing-codes="[]"
      @saved="environmentCreated"
    />
    <div
      v-if="legacyLayoutVisible && environmentId"
      class="environment-content"
      v-loading="loading"
    >
      <ElAlert v-if="error" type="error" :title="error" :closable="false"
        ><ElButton text @click="loadConfig">重新加载</ElButton></ElAlert
      >
      <template v-else-if="config">
        <ElAlert v-if="saveError" :title="saveError" type="error" :closable="false"
          ><ElButton text @click="reloadConfig">重新加载</ElButton></ElAlert
        >
        <div class="environment-summary">
          <span
            >基础版本 #{{ config.revision_id }}
            <small v-if="config.revision_id !== config.latest_revision_id"
              >（基础配置有更新）</small
            ></span
          >
          <span
            >{{
              config.environment_yaml_revision_id
                ? `环境 YAML #${config.environment_yaml_revision_id}`
                : '当前环境尚未保存独立 YAML'
            }}
            · {{ dirty ? '未保存' : '已保存，发布后生效' }}</span
          >
          <ElTag v-if="previewing" type="warning" size="small">预览校验中</ElTag>
          <ElTag v-else-if="previewState === 'success'" type="success" size="small">预览通过</ElTag>
          <ElTag v-else-if="previewState === 'error'" type="danger" size="small">预览失败</ElTag>
          <ElButton
            v-if="!config.bound && canUpdate(environmentId)"
            size="small"
            type="primary"
            @click="joinCurrent"
            >加入此环境</ElButton
          >
          <ElButton
            v-else-if="canUpdate(environmentId)"
            size="small"
            type="primary"
            :disabled="busy"
            :loading="saving"
            @click="saveCurrent"
            >{{ config.format === 'k8s_yaml' ? '预览与保存' : '保存当前环境' }}</ElButton
          >
          <ElButton
            v-if="config.format === 'k8s_yaml' && config.bound && canUpdate(environmentId)"
            size="small"
            :disabled="busy || !config.current_deployment || !releaseWorkflows.length"
            @click="previewCurrent(true)"
            >{{ dirty ? '保存并发布' : '发布当前配置' }}</ElButton
          >
        </div>
        <ElRadioGroup
          v-if="config.format === 'k8s_yaml'"
          v-model="configurationView"
          size="small"
          class="configuration-tabs"
          ><ElRadioButton value="resources">常用配置</ElRadioButton
          ><ElRadioButton value="yaml">高级 YAML / 变量</ElRadioButton></ElRadioGroup
        >
        <div
          v-show="configurationView === 'yaml' || config.format === 'helm_chart'"
          class="environment-grid"
        >
          <div class="definition-view">
            <div class="definition-heading"
              ><span>{{
                config.format === 'helm_chart'
                  ? 'Chart 文件（只读）'
                  : `${environment?.name} · YAML`
              }}</span
              ><ElSwitch
                v-if="config.format !== 'helm_chart'"
                v-model="renderValues"
                active-text="变量预览"
            /></div>
            <p class="note">{{
              config.format === 'helm_chart'
                ? '右侧变量按 Helm values 路径覆盖；Chart 结构在服务基础配置中管理。'
                : '高级入口：需要完整控制 Kubernetes 资源时再编辑。日常 CPU、内存、副本、命令、配置文件和凭据请在“常用配置”中修改。'
            }}</p>
            <ElSelect v-if="chartFiles.length" v-model="chartFile" size="small"
              ><ElOption
                v-for="file in chartFiles"
                :key="file.path"
                :label="file.path"
                :value="file.path"
            /></ElSelect>
            <pre v-if="config.format === 'helm_chart' || renderValues" class="yaml-preview">{{
              displayDefinition
            }}</pre>
            <EnvironmentYamlEditor
              v-else
              ref="environmentEditor"
              v-model="definitionDraft"
              :active-file="activeFile"
              :readonly="!config.bound || !canUpdate(environmentId) || busy"
              @file-change="emit('file-change', $event)"
            />
            <ElButton
              v-if="config.environment_yaml_revision_id && canUpdate(environmentId)"
              text
              size="small"
              :disabled="busy"
              @click="resetEnvironmentDefinition"
              >恢复基础 YAML…</ElButton
            >
          </div>
          <aside class="environment-variables">
            <strong>变量与来源</strong>
            <p class="note"
              >只显示当前环境相对服务基础配置的变量差异。关闭“覆盖”会恢复继承；保存后需要预览并发布才会在集群生效。</p
            >
            <p class="note"
              >变量须被 YAML 或 Chart 引用才影响部署；不会自动成为容器 env。YAML 密码请通过
              Kubernetes Secret 引用。</p
            >
            <div v-for="row in rows" :key="row.key" class="variable-row">
              <div class="variable-label"
                ><code>{{ row.key }}</code
                ><small>{{
                  row.secret
                    ? '凭据引用'
                    : hasConfigKey(draft, row.key)
                      ? '当前环境覆盖'
                      : row.origin
                }}</small></div
              >
              <div class="variable-input"
                ><ElInput
                  :model-value="
                    row.secret
                      ? '••••••'
                      : hasConfigKey(draft, row.key)
                        ? draft[row.key]
                        : row.value
                  "
                  size="small"
                  :disabled="
                    row.secret || !hasConfigKey(draft, row.key) || !canUpdate(environmentId) || busy
                  "
                  @update:model-value="draft[row.key] = $event"
                /><ElCheckbox
                  :model-value="hasConfigKey(draft, row.key)"
                  :disabled="row.secret || !canUpdate(environmentId) || busy"
                  @change="toggleOverride(row.key, row.value, !!$event)"
                  >覆盖</ElCheckbox
                ></div
              >
            </div>
            <div v-if="canUpdate(environmentId)" class="add-variable"
              ><ElInput
                v-model="newKey"
                placeholder="添加变量 / Helm values 路径"
                size="small"
              /><ElButton size="small" :disabled="busy" @click="addKey">添加</ElButton></div
            >
          </aside>
        </div>
        <EnvironmentComparison :config="config" :environments="environments" />
      </template>
    </div>
    <ElDialog
      v-model="assetsEnvironmentVisible"
      title="选择配置环境"
      width="min(420px, 92vw)"
      :close-on-click-modal="!busy"
      :close-on-press-escape="!busy"
      :show-close="!busy"
    >
      <ElSelect
        v-model="assetsEnvironmentId"
        placeholder="选择环境"
        aria-label="配套配置目标环境"
        :disabled="busy"
        style="width: 100%"
      >
        <ElOption
          v-for="env in environments"
          :key="env.id"
          :value="env.id"
          :label="`${env.name}${env.is_production ? ' · 生产' : ''} · ${env.namespace || env.code}`"
        />
      </ElSelect>
      <template #footer>
        <ElButton :disabled="busy" @click="assetsEnvironmentVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :disabled="busy || !assetsEnvironmentId"
          :loading="loading"
          @click="openAssetsEnvironment"
          >打开配置</ElButton
        >
      </template>
    </ElDialog>
    <ElDrawer
      v-model="assetsVisible"
      title="配套配置"
      size="min(900px, 100vw)"
      append-to-body
      :close-on-click-modal="!busy"
      :close-on-press-escape="!busy"
      :show-close="!busy"
    >
      <ServiceResourcesEditor
        v-if="config?.format === 'k8s_yaml' && environment"
        ref="resourcesEditor"
        :config="editorConfig"
        :service="service"
        :environment="environment"
        :editable="canUpdate(environmentId)"
        :variables-dirty="dirty"
        integrated
        @busy="resourcesBusy = $event"
        @saved="loadConfig"
      />
      <template #footer>
        <ElButton :disabled="busy" @click="assetsVisible = false">返回 YAML</ElButton>
        <ElButton
          v-if="config?.bound && canUpdate(environmentId)"
          type="primary"
          :disabled="busy || !dirty"
          :loading="previewing"
          @click="previewAssets"
          >预览与保存</ElButton
        >
      </template>
    </ElDrawer>
    <ElDialog
      v-model="currentPreviewVisible"
      :title="`${environment?.name || ''} · 配置变更`"
      width="min(1100px, 95vw)"
      append-to-body
      :close-on-click-modal="false"
      :before-close="closeCurrentPreview"
    >
      <div v-if="currentPreview" class="current-config-preview">
        <ElAlert
          v-for="notice in currentPreview.adjustments || []"
          :key="notice"
          type="info"
          :title="notice"
          :closable="false"
        />
        <ElAlert
          type="info"
          :closable="false"
          :title="`${service.name} · ${environment?.namespace} · 仅当前环境；配置结构校验通过，尚未访问集群预检`"
        />
        <p class="note"
          >以下包含
          YAML、变量及资源覆盖。镜像占位由实际发布产物替换；仅配置发布保留下面列出的全部容器镜像。移除文档只停止管理，不自动删除已有集群资源。</p
        >
        <div class="preview-columns"
          ><div
            ><strong>修改前</strong><pre>{{ currentBefore }}</pre></div
          ><div
            ><strong>保存后</strong><pre>{{ currentPreview.manifest }}</pre>
          </div></div
        >
        <details v-if="currentPreview.definition !== config?.definition"
          ><summary>查看原始 YAML 修改（包含注释与文件名）</summary
          ><div class="preview-columns">
            <pre>{{ config?.definition }}</pre
            ><pre>{{ currentPreview.definition }}</pre>
          </div></details
        >
        <ElAlert v-if="releaseError" type="error" :title="releaseError" :closable="false" />
        <ElCheckbox
          v-model="publishAfterSave"
          :disabled="saving || !config?.current_deployment || !releaseWorkflows.length"
          >保存后发布当前配置（保留当前镜像，不构建）</ElCheckbox
        >
        <p v-if="!config?.current_deployment" class="note"
          >暂无可追溯的成功部署及全部容器镜像摘要。可以先保存，再从工作流明确选择镜像或首次构建。</p
        >
        <div v-if="publishAfterSave" class="release-target">
          <ElSelect
            v-model="releaseWorkflowId"
            placeholder="选择绑定当前环境的发布工作流"
            :disabled="saving"
            ><ElOption
              v-for="workflow in releaseWorkflows"
              :key="workflow.id"
              :label="workflow.name"
              :value="workflow.id"
          /></ElSelect>
          <p
            v-for="(image, name) in config?.current_deployment?.images || {}"
            :key="name"
            class="image-line"
            ><strong>{{ name }}</strong
            ><code>{{ image }}</code></p
          >
          <p class="note"
            >执行前重新核对配置及部署版本，保留工作流已有审批；不会在这里切换部署目标。</p
          >
        </div>
        <ElCheckbox
          v-if="environment?.is_production"
          v-model="currentProductionConfirmed"
          :disabled="saving"
          >已核对生产目标、配置差异及镜像</ElCheckbox
        >
      </div>
      <template #footer
        ><ElButton
          :disabled="saving"
          @click="closeCurrentPreview(() => (currentPreviewVisible = false))"
          >继续编辑</ElButton
        ><ElButton
          type="primary"
          :loading="saving"
          :disabled="
            !currentPreview ||
            releaseUncertain ||
            (environment?.is_production && !currentProductionConfirmed) ||
            (publishAfterSave && !releaseWorkflowId)
          "
          @click="commitCurrent"
          >{{ publishAfterSave ? '确认保存并发布' : '保存当前环境' }}</ElButton
        ></template
      >
    </ElDialog>
    <ElDialog
      v-model="syncVisible"
      :title="syncMode === 'join' ? '加入环境' : '同步配置'"
      :width="
        syncMode === 'join' ? 'min(660px, calc(100vw - 32px))' : 'min(880px, calc(100vw - 32px))'
      "
      class="environment-transfer-dialog"
      append-to-body
      :close-on-click-modal="false"
      :close-on-press-escape="!busy"
      :show-close="!busy"
    >
      <div class="sync-summary">
        <strong
          >{{ service.name }} ·
          {{
            syncMode === 'join' ? '为服务建立环境配置' : `来源：${environmentName(environmentId)}`
          }}</strong
        >
        <span>{{
          syncMode === 'join'
            ? '使用服务基础配置，保留目标环境公共参数。加入后核对环境配置，再单独部署。'
            : '默认保留目标环境配置；只同步你选中的内容，保存不会执行部署。'
        }}</span>
      </div>
      <div class="transfer-target">
        <label>目标环境</label>
        <ElSelect
          v-model="targetId"
          placeholder="请选择一个目标环境"
          :disabled="busy"
          aria-label="目标环境"
          @change="loadTransferTarget"
        >
          <ElOption
            v-for="env in transferEnvironments"
            :key="env.id"
            :value="env.id"
            :label="environmentName(env.id)"
            :disabled="
              !canUpdate(env.id) || (!!env.binding_status && env.binding_status !== 'validated')
            "
          />
        </ElSelect>
        <span v-if="targetEnvironment" class="target-location"
          >集群：{{
            clusters?.find((c) => c.id === targetEnvironment?.cluster_id)?.name ||
            `#${targetEnvironment.cluster_id || '未绑定'}`
          }}
          · Namespace：{{
            targetEnvironment.namespace || targetEnvironment.host_group_ref || '未绑定'
          }}</span
        >
      </div>
      <p v-if="syncMode === 'join'" class="note"
        >先查看配置预览；加入成功后自动打开目标环境的 YAML，方便继续设置环境差异。</p
      >
      <ElAlert
        v-if="syncMode === 'join' && transferTarget?.bound"
        type="info"
        title="此服务已加入该环境"
        :closable="false"
      >
        <ElButton text @click="viewTransferTarget">查看该环境配置</ElButton>
      </ElAlert>
      <details
        v-if="syncMode === 'join' && transferTarget && !transferTarget.bound"
        :open="joinRows.some((row) => !row.secret && !row.value)"
        class="transfer-parameters"
      >
        <summary>目标环境参数 <span>沿用目标值，需要时直接修改</span></summary>
        <div class="join-fields">
          <label v-for="row in joinRows" :key="row.key">
            <span
              >{{ parameterLabel(row.key) }} <code>{{ row.key }}</code></span
            >
            <ElInput
              v-if="!row.secret"
              :model-value="joinOverrides[row.key] ?? row.value"
              :disabled="busy"
              :placeholder="row.value ? '沿用目标环境值' : '请填写目标环境值'"
              @update:model-value="(value) => (joinOverrides[row.key] = value)"
            />
            <small v-else>目标凭据引用 · 保留</small>
          </label>
        </div>
      </details>
      <template v-if="syncMode === 'sync'">
        <div class="transfer-options">
          <ElCheckbox v-model="syncRevision" :disabled="busy"
            >更新基础配置版本 #{{ transferRevision }}</ElCheckbox
          >
          <small>已有环境独立 YAML 不会自动升级覆盖，冲突时会阻止保存。</small>
        </div>
        <details class="transfer-parameters">
          <summary
            >选择需要同步的参数
            <span>{{
              syncKeys.length ? `已选 ${syncKeys.length} 项` : '默认全部保留目标值'
            }}</span></summary
          >
          <ElCheckboxGroup v-model="syncKeys" :disabled="busy" class="transfer-fields">
            <ElCheckbox
              v-for="row in rows.filter((item) => !item.secret)"
              :key="row.key"
              :value="row.key"
            >
              <span>{{ parameterLabel(row.key) }}</span
              ><code>{{ row.key }}</code
              ><small>{{ sourceValue(row.key) }}</small>
            </ElCheckbox>
          </ElCheckboxGroup>
          <p class="note"
            >勾选后复制来源的实际值。密码和凭据不在此处同步，请在目标环境的“配套配置”中设置。</p
          >
        </details>
        <div v-if="config?.format === 'k8s_yaml'" class="yaml-sync-choice">
          <ElCheckbox v-model="syncYaml" :disabled="busy">使用来源 YAML 编辑目标草稿</ElCheckbox>
          <template v-if="syncYaml">
            <ElAlert
              type="warning"
              :closable="false"
              title="此操作替换目标完整 YAML。请保留目标环境独有资源，并核对写在 YAML 中的域名、地址、Secret/PVC 名称和启动参数。"
            />
            <ElInput
              v-model="syncDefinition"
              type="textarea"
              :rows="12"
              :disabled="busy"
              aria-label="目标 YAML 草稿"
              class="transfer-yaml"
            />
          </template>
        </div>
      </template>
      <ElAlert v-if="syncError" type="error" :title="syncError" :closable="false" />
      <div v-for="item in plan" :key="item.config.environment_id" class="diff-target">
        <strong>{{ environmentName(item.config.environment_id) }}</strong
        ><small>配置变更 · 不执行部署</small>
        <table
          ><thead
            ><tr><th>字段</th><th>当前配置</th><th>保存后</th></tr></thead
          ><tbody>
            <tr v-if="syncRevision"
              ><td>配置版本</td
              ><td>{{ item.config.bound ? `#${item.config.revision_id}` : '未加入' }}</td
              ><td>#{{ transferRevision }}</td></tr
            >
            <tr v-for="key in syncKeys" :key="key"
              ><td
                >{{ parameterLabel(key) }} <code>{{ key }}</code></td
              ><td>{{
                item.values[key]?.secret
                  ? '凭据引用（不可覆盖）'
                  : (item.values[key]?.value ?? '未配置')
              }}</td
              ><td>{{ sourceValue(key) }}</td></tr
            >
          </tbody></table
        >
        <div v-if="item.after" class="effective-diff"
          ><strong>实际变量变化（含基础配置默认值）</strong
          ><table
            ><tbody
              ><tr v-for="row in effectiveDiff(item)" :key="row.key"
                ><td>{{ row.key }}</td
                ><td>{{ row.before }}</td
                ><td>{{ row.after }}</td></tr
              ></tbody
            ></table
          ><span v-if="!effectiveDiff(item).length" class="note">变量值没有变化</span></div
        >
        <details
          v-if="syncRevision && item.after && item.config.definition !== item.after.definition"
          ><summary>查看基础配置变化</summary
          ><div class="template-diff">
            <pre>{{ item.config.definition }}</pre
            ><pre>{{ item.after.definition }}</pre>
          </div></details
        >
        <details
          v-if="syncYaml && item.after && item.config.definition !== item.after.definition"
          open
          class="yaml-diff"
        >
          <summary>目标 YAML：当前配置 → 保存后</summary>
          <div class="template-diff">
            <pre>{{ item.config.definition }}</pre
            ><pre>{{ item.after.definition }}</pre>
          </div>
        </details>
        <details v-if="item.after?.manifest" :open="syncMode === 'join'">
          <summary>目标环境渲染结果 · {{ targetEnvironment?.namespace }}</summary>
          <pre class="transfer-manifest">{{ item.after.manifest }}</pre>
        </details>
      </div>
      <ElCheckbox v-if="plan.length && syncYaml" v-model="yamlConfirmed" :disabled="busy"
        >已核对完整 YAML 和目标环境差异</ElCheckbox
      >
      <ElAlert
        v-if="plan.length && productionTargets.length"
        type="warning"
        :closable="false"
        :title="`将更新生产环境配置：${productionTargets.map((item) => item.name).join('、')}`"
        ><ElCheckbox v-model="productionConfirmed" :disabled="busy"
          >已核对生产目标及上方差异，仅保存配置，稍后再部署</ElCheckbox
        ></ElAlert
      >
      <template #footer
        ><ElButton :disabled="busy" @click="syncVisible = false">取消</ElButton
        ><ElButton
          :loading="previewing"
          :disabled="
            busy ||
            !targetIds.length ||
            (syncMode === 'join' && !!transferTarget?.bound) ||
            (!syncRevision && !syncKeys.length && !syncYaml)
          "
          @click="previewSync"
          >查看差异</ElButton
        ><ElButton
          type="primary"
          :loading="saving"
          :disabled="
            busy ||
            !plan.length ||
            (syncYaml && !yamlConfirmed) ||
            (productionTargets.length > 0 && !productionConfirmed)
          "
          @click="applySync"
          >{{ syncMode === 'join' ? '确认加入并查看配置' : '保存到目标环境' }}</ElButton
        ></template
      >
    </ElDialog>
  </section>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'
  import {
    getServiceEnvironmentConfigApi,
    saveServiceEnvironmentConfigApi,
    getWorkflowApi,
    runWorkflowApi,
    type AxiOpsCluster,
    type AxiOpsEnvironment,
    type AxiOpsService,
    type AxiOpsWorkflow,
    type SaveServiceEnvironmentConfig,
    type ServiceEnvironmentConfig
  } from '@/api/axiops'
  import {
    configValues,
    hasConfigKey,
    localOverrides,
    variablePatch
  } from '@/utils/service-environment-config'
  import ServiceWorkbenchPanel from './ServiceWorkbenchPanel.vue'
  import HelmServicePanel from './HelmServicePanel.vue'
  import ServiceResourcesEditor from './ServiceResourcesEditor.vue'
  import ServiceTemplatePicker from './ServiceTemplatePicker.vue'
  import EnvironmentComparison from './EnvironmentComparison.vue'
  import EnvironmentYamlEditor from './EnvironmentYamlEditor.vue'

  const props = defineProps<{
    service: AxiOpsService | null
    preferredEnvironmentId?: number
    projectName?: string
    projectCode?: string
    activeFile?: string
    environments: AxiOpsEnvironment[]
    clusters?: AxiOpsCluster[]
    canEdit: boolean
    canImport?: boolean
    canViewBuild: boolean
    workflows?: AxiOpsWorkflow[]
  }>()
  const emit = defineEmits([
    'import',
    'template-import',
    'build-config',
    'file-change',
    'files-change',
    'saved',
    'environment-saved',
    'published'
  ])
  const access = useAxiOpsAccess()
  const sharedEditor = ref<InstanceType<typeof ServiceWorkbenchPanel>>()
  const helmEditor = ref<InstanceType<typeof HelmServicePanel>>()
  const resourcesEditor = ref<InstanceType<typeof ServiceResourcesEditor>>()
  const environmentEditor = ref<InstanceType<typeof EnvironmentYamlEditor>>()
  const definitionDraft = ref(''),
    resetDefinition = ref(false)
  // 日常操作先进入结构化环境差异；完整 YAML 保留为高级入口。
  const configurationView = ref('resources')
  const legacyLayoutVisible = ref(false)
  const currentPreviewVisible = ref(false),
    currentPreview = ref<ServiceEnvironmentConfig | null>(null),
    currentBefore = ref('')
  const currentRequest = ref<SaveServiceEnvironmentConfig | null>(null)
  const publishAfterSave = ref(false),
    currentProductionConfirmed = ref(false),
    releaseWorkflowId = ref<number>(),
    releaseError = ref(''),
    releaseUncertain = ref(false)
  const releaseWorkflows = computed(() =>
    (props.workflows || []).filter(
      (w) =>
        w.target_environment_id === environmentId.value &&
        w.delivery_type === 'k8s_yaml' &&
        w.status === 'active' &&
        access.can('workflow', 'run', {
          type: 'workflow',
          id: w.id,
          project_id: props.service?.project_id,
          environment_id: environmentId.value,
          environment: props.environments.find((e) => e.id === environmentId.value)?.code
        })
    )
  )
  const editorConfig = computed(() => ({ ...config.value!, definition: definitionDraft.value }))
  const resourcesBusy = ref(false)
  const assetsVisible = ref(false)
  const assetsEnvironmentVisible = ref(false),
    assetsEnvironmentId = ref<number>()
  const environmentCreationVisible = ref(false)
  async function environmentCreated() {
    environmentCreationVisible.value = false
    await loadConfig()
    emit('environment-saved')
  }
  const environmentId = ref(0),
    loading = ref(false),
    saving = ref(false),
    previewing = ref(false),
    error = ref('')
  const config = ref<ServiceEnvironmentConfig | null>(null),
    draft = ref<Record<string, string>>({}),
    newKey = ref(''),
    renderValues = ref(false),
    chartFile = ref('')
  const saveError = ref('')
  const previewState = ref<'idle' | 'validating' | 'success' | 'error'>('idle')
  const syncVisible = ref(false),
    targetIds = ref<number[]>([]),
    syncRevision = ref(true),
    syncKeys = ref<string[]>([]),
    syncYaml = ref(false),
    syncMode = ref<'join' | 'sync'>('join'),
    syncDefinition = ref(''),
    syncError = ref('')
  const yamlConfirmed = ref(false)
  const transferTarget = ref<ServiceEnvironmentConfig | null>(null),
    transferReading = ref(false)
  const joinOverrides = ref<Record<string, string>>({})
  const joinRows = computed(() =>
    transferTarget.value
      ? Object.entries(configValues(transferTarget.value)).map(([key, value]) => ({
          key,
          ...value
        }))
      : []
  )
  const targetId = computed({
    get: () => targetIds.value[0],
    set: (id: number | undefined) => {
      targetIds.value = id ? [id] : []
    }
  })
  const targetEnvironment = computed(() => props.environments.find((e) => e.id === targetId.value))
  const transferEnvironments = computed(() =>
    props.environments.filter((e) => syncMode.value === 'join' || e.id !== environmentId.value)
  )
  const transferRevision = computed(() =>
    syncMode.value === 'join'
      ? (!environmentId.value && sharedEditor.value?.revisionId) ||
        props.service?.current_revision_id ||
        0
      : sourceRevision.value
  )
  const parameterLabel = (key: string) =>
    ({
      replicas: '副本数',
      port: '服务端口',
      cpuRequest: 'CPU 请求',
      cpuLimit: 'CPU 上限',
      memoryRequest: '内存请求',
      memoryLimit: '内存上限',
      springProfile: '应用环境',
      nacosNamespace: 'Nacos 命名空间',
      nacosServerAddr: 'Nacos 地址',
      javaOpts: 'Java 启动参数'
    })[key] || '自定义参数'
  const sourceValue = (key: string) =>
    hasConfigKey(draft.value, key) ? draft.value[key] : configValues(config.value!)[key]?.value
  const productionConfirmed = ref(false)
  const plannedRequest = ref<SaveServiceEnvironmentConfig | null>(null)
  const productionTargets = computed(() =>
    props.environments.filter((item) => targetIds.value.includes(item.id) && item.is_production)
  )
  const plan = ref<
    Array<{
      config: ServiceEnvironmentConfig
      values: ReturnType<typeof configValues>
      after?: ServiceEnvironmentConfig
      request?: SaveServiceEnvironmentConfig
    }>
  >([])
  let generation = 0
  let transferGeneration = 0
  const environment = computed(() => props.environments.find((e) => e.id === environmentId.value))
  const busy = computed(
    () =>
      loading.value ||
      saving.value ||
      previewing.value ||
      transferReading.value ||
      resourcesBusy.value ||
      !!helmEditor.value?.busy ||
      !!sharedEditor.value?.saving
  )
  const sourceRevision = computed(() =>
    environmentId.value
      ? config.value?.revision_id || 0
      : sharedEditor.value?.revisionId || props.service?.current_revision_id || 0
  )
  const dirty = computed(
    () =>
      !!config.value &&
      (Object.keys(variablePatch(localOverrides(config.value), draft.value)).length > 0 ||
        definitionDraft.value !== config.value.definition ||
        resetDefinition.value ||
        !!resourcesEditor.value?.dirty)
  )
  const canUpdate = (id: number) => {
    const env = props.environments.find((e) => e.id === id)
    return (
      !!env &&
      access.can('environment', 'update', {
        type: 'environment',
        id,
        project_id: props.service?.project_id,
        environment: env.code
      })
    )
  }
  const environmentName = (id: number) => {
    const env = props.environments.find((e) => e.id === id)
    return `${env?.name || id}${env?.is_production ? '（生产）' : ''}`
  }
  const rows = computed(() => {
    if (!config.value) return []
    const values = configValues(config.value)
    // Removing an override must reveal the inherited value immediately.
    const inherited = configValues({
      ...config.value,
      replicas: null,
      variables: config.value.variables.filter((v) => v.origin !== 'override')
    })
    return [...new Set([...Object.keys(values), ...Object.keys(draft.value)])]
      .sort()
      .map((key) => ({
        key,
        ...((values[key]?.secret || hasConfigKey(draft.value, key)
          ? values[key]
          : inherited[key]) || { value: '', origin: '未配置', secret: false })
      }))
  })
  const chartFiles = computed<Array<{ path: string; content: string }>>(() => {
    if (config.value?.format !== 'helm_chart') return []
    try {
      return JSON.parse(config.value.definition).files || []
    } catch {
      return []
    }
  })
  const displayDefinition = computed(() => {
    if (!config.value) return ''
    if (config.value.format === 'helm_chart')
      return (
        (chartFiles.value.find((f) => f.path === chartFile.value) || chartFiles.value[0])
          ?.content || config.value.definition
      )
    if (!renderValues.value) return definitionDraft.value
    const values: Record<string, string> = Object.fromEntries(
      rows.value
        .filter((r) => !r.secret)
        .map((r) => [r.key, hasConfigKey(draft.value, r.key) ? draft.value[r.key] : r.value])
    )
    Object.assign(values, {
      Product: props.projectCode || '',
      Service: props.service?.code || '',
      Namespace: environment.value?.namespace || '',
      EnvName: environment.value?.code || '',
      ClusterName: props.clusters?.find((c) => c.id === environment.value?.cluster_id)?.name || '',
      Image: config.value.image_ref || '<部署时注入镜像>'
    })
    values[`${props.service?.code}-image`] = values.Image
    return definitionDraft.value.replace(
      /\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_-]*)\$/g,
      (match, a, b) => values[a || b] || match
    )
  })
  async function confirmLeave() {
    if (busy.value) return false
    if (props.service?.delivery_type === 'helm')
      return (await helmEditor.value?.confirmLeave()) ?? true
    if (!dirty.value && !sharedEditor.value?.dirty && !resourcesEditor.value?.dirty) return true
    try {
      await ElMessageBox.confirm('有未保存的配置，离开将丢弃这些修改。', '未保存的修改', {
        type: 'warning',
        confirmButtonText: '放弃修改',
        cancelButtonText: '继续编辑'
      })
      resourcesEditor.value?.discardChanges()
      sharedEditor.value?.discardChanges()
      return true
    } catch {
      return false
    }
  }
  async function switchEnvironment(id: number) {
    if (id === environmentId.value) {
      if (loading.value) await loadConfig()
      return
    }
    if (!(await confirmLeave())) return
    if (!environmentId.value && sharedEditor.value?.dirty) sharedEditor.value.discardChanges()
    environmentId.value = id
    await loadConfig()
  }
  function openAssets() {
    if (busy.value || props.service?.delivery_type !== 'k8s_yaml') return
    if (environment.value && config.value && !config.value.bound) {
      ElMessage.warning('请先保存当前环境的 YAML，或从模板配置此环境')
      return
    }
    if (environment.value && config.value?.format === 'k8s_yaml') {
      assetsVisible.value = true
      return
    }
    if (!props.environments.length) {
      ElMessage.warning('请先在项目中创建环境')
      return
    }
    assetsEnvironmentId.value = undefined
    assetsEnvironmentVisible.value = true
  }
  async function openAssetsEnvironment() {
    const id = assetsEnvironmentId.value,
      sid = props.service?.id
    if (!id || busy.value || !props.environments.some((env) => env.id === id)) return
    if (environmentId.value === id && !config.value) await loadConfig()
    else await switchEnvironment(id)
    if (props.service?.id !== sid || environmentId.value !== id || !config.value || error.value)
      return
    if (config.value.format !== 'k8s_yaml') return
    assetsEnvironmentVisible.value = false
    assetsVisible.value = true
  }
  async function loadConfig() {
    const request = ++generation,
      sid = props.service?.id,
      eid = environmentId.value
    config.value = null
    draft.value = {}
    error.value = ''
    saveError.value = ''
    plan.value = []
    assetsVisible.value = false
    if (!sid || !eid) {
      loading.value = false
      return
    }
    loading.value = true
    try {
      const value = await getServiceEnvironmentConfigApi(eid, sid)
      if (request !== generation) return
      if (value.environment_id !== eid || value.service_id !== sid)
        throw new Error('环境配置与当前选择不一致，请重新加载')
      configValues(value) // malformed metadata is a load error, never writable defaults
      config.value = value
      definitionDraft.value = value.definition
      resetDefinition.value = false
      draft.value = localOverrides(value)
      chartFile.value = ''
    } catch (e: any) {
      if (request === generation)
        error.value =
          e?.code === 404 && !e?.businessCode
            ? '环境配置接口不可用，请确认 CI/CD API 已更新并启动，然后重新加载。'
            : e?.message || '环境配置读取失败'
    } finally {
      if (request === generation) loading.value = false
    }
  }
  async function reloadConfig() {
    if (await confirmLeave()) await loadConfig()
  }
  async function helmSaved() {
    await loadConfig()
    emit('environment-saved')
  }
  function toggleOverride(key: string, value: string, enabled: boolean) {
    if (enabled) draft.value[key] = value
    else delete draft.value[key]
  }
  function addKey() {
    const key = newKey.value.trim()
    if (
      !/^[A-Za-z_][A-Za-z0-9_.-]{0,127}$/.test(key) ||
      ['Product', 'Service', 'Namespace', 'EnvName', 'ClusterName', 'Image'].includes(key) ||
      key.endsWith('-image')
    )
      return void ElMessage.warning('请输入有效的自定义变量名')
    if (rows.value.some((r) => r.key === key && r.secret))
      return void ElMessage.warning('凭据引用需在环境配置中管理')
    draft.value[key] ??= ''
    newKey.value = ''
  }
  async function saveCurrent() {
    if (config.value?.format === 'k8s_yaml') return previewCurrent(false)
    if (!props.service || !config.value || !dirty.value || busy.value) return
    if (resourcesEditor.value?.dirty)
      return void ElMessage.warning('请先保存或撤销下方配套配置修改')
    saving.value = true
    try {
      await saveServiceEnvironmentConfigApi(props.service.id, {
        targets: [{ environment_id: environmentId.value, token: config.value.token }],
        revision_id: 0,
        variables: variablePatch(localOverrides(config.value), draft.value)
      })
      ElMessage.success('当前环境配置已保存，待部署')
      await loadConfig()
    } catch (e: any) {
      saveError.value = e?.message || '保存失败，请重新加载'
    } finally {
      saving.value = false
    }
  }
  async function saveEnvironmentFromWorkbench(payload: {
    definition: string
    variables: Record<string, string>
  }) {
    if (!config.value || !environmentId.value) return
    definitionDraft.value = payload.definition
    draft.value = {
      ...localOverrides(config.value),
      ...payload.variables
    }
    resetDefinition.value = false
    await previewCurrent(false)
  }
  async function previewAssets() {
    if (busy.value) return
    const payload = sharedEditor.value?.getEnvironmentDraft()
    if (!payload) return
    await saveEnvironmentFromWorkbench(payload)
    if (currentPreviewVisible.value) assetsVisible.value = false
  }
  function closeCurrentPreview(done: () => void) {
    if (!saving.value) {
      resetDefinition.value = false
      done()
    }
  }
  async function resetEnvironmentDefinition() {
    try {
      await ElMessageBox.confirm(
        '当前环境将恢复绑定的公共定义，环境变量和资源覆盖保留。请预览后再保存。',
        '恢复公共定义'
      )
      resetDefinition.value = true
      await previewCurrent(false)
    } catch {
      /* user cancelled */
    }
  }
  async function previewCurrent(publish = false) {
    if (!props.service || !config.value || busy.value) return
    const previewGeneration = generation
    previewing.value = true
    previewState.value = 'validating'
    saveError.value = ''
    releaseError.value = ''
    releaseUncertain.value = false
    currentProductionConfirmed.value = false
    currentPreview.value = null
    try {
      const request: SaveServiceEnvironmentConfig = {
        targets: [{ environment_id: environmentId.value, token: config.value.token }],
        revision_id: 0,
        variables: variablePatch(localOverrides(config.value), draft.value),
        ...(resetDefinition.value
          ? { reset_definition: true }
          : { definition: definitionDraft.value }),
        assets: resourcesEditor.value?.getAssets() || config.value.assets
      }
      const before = config.value.bound
        ? await saveServiceEnvironmentConfigApi(
            props.service.id,
            {
              targets: request.targets,
              revision_id: 0,
              variables: {},
              definition: config.value.definition
            },
            true
          )
        : null
      const after = await saveServiceEnvironmentConfigApi(props.service.id, request, true)
      if (previewGeneration !== generation) return
      if (!after?.configs?.[0]?.manifest) throw new Error('完整配置预览未返回，请确认后端已升级')
      currentBefore.value = before?.configs?.[0]?.manifest || ''
      currentPreview.value = after.configs[0]
      currentRequest.value = request
      publishAfterSave.value = publish
      releaseWorkflowId.value =
        releaseWorkflows.value.length === 1 ? releaseWorkflows.value[0].id : undefined
      currentPreviewVisible.value = true
      previewState.value = 'success'
    } catch (e: any) {
      if (previewGeneration === generation) {
        saveError.value = e?.message || '预览失败，未保存任何配置'
        previewState.value = 'error'
      }
    } finally {
      if (previewGeneration === generation) previewing.value = false
    }
  }
  async function commitCurrent() {
    if (
      !props.service ||
      !config.value ||
      !currentRequest.value ||
      !currentPreview.value ||
      saving.value ||
      releaseUncertain.value ||
      (environment.value?.is_production && !currentProductionConfirmed.value)
    )
      return
    const service = props.service,
      eid = environmentId.value,
      snapshot = config.value.current_deployment,
      workflowId = releaseWorkflowId.value
    if (publishAfterSave.value && (!snapshot || !workflowId)) return
    saving.value = true
    releaseError.value = ''
    let saved = false,
      submitting = false
    try {
      if (publishAfterSave.value) {
        const workflow = await getWorkflowApi(workflowId!)
        if (
          workflow.target_environment_id !== eid ||
          !workflow.nodes?.some((n) => n.type === 'yaml_deploy') ||
          workflow.nodes.some((n) => ['test', 'helm_deploy', 'host_deploy'].includes(n.type))
        )
          throw new Error('所选工作流不支持仅配置发布，请选择构建 + YAML 部署工作流')
      }
      const result = await saveServiceEnvironmentConfigApi(service.id, currentRequest.value)
      saved = true
      if (publishAfterSave.value) {
        const current = result.configs[0]
        submitting = true
        const run = await runWorkflowApi(workflowId!, {
          execution_mode: 'config_only',
          environment_id: eid,
          service_id: service.id,
          service_revision_id: current.revision_id,
          use_environment_config: true,
          source_snapshot_id: snapshot!.snapshot_id,
          expected_config_digest: current.config_digest,
          trigger_type: 'manual'
        })
        if (!run?.id) throw new Error('发布提交结果不完整，请查看运行记录')
        currentPreviewVisible.value = false
        await loadConfig()
        emit('published', { workflowId, runId: run.id })
        ElMessage.success('配置已保存，发布已提交；请查看执行进度')
      } else {
        currentPreviewVisible.value = false
        await loadConfig()
        ElMessage.success('当前环境配置已保存，待发布')
      }
    } catch (e: any) {
      releaseUncertain.value = saved
      releaseError.value = `${saved ? '配置已保存；' : '未确认保存完成；'}${submitting ? '发布提交未确认，请查看运行记录后再操作。' : '发布未提交。'}${e?.message || '请求失败'}`
    } finally {
      saving.value = false
    }
  }
  function clearPlan() {
    transferGeneration++
    plan.value = []
    plannedRequest.value = null
    productionConfirmed.value = false
    yamlConfirmed.value = false
    syncError.value = ''
  }
  function beginTransfer(mode: 'join' | 'sync') {
    if (busy.value) return false
    if (dirty.value || sharedEditor.value?.dirty || resourcesEditor.value?.dirty)
      return void ElMessage.warning('请先保存当前修改，再操作环境配置')
    if (!sourceRevision.value) return void ElMessage.warning('请先保存当前服务配置')
    syncMode.value = mode
    targetIds.value = []
    syncKeys.value = []
    syncYaml.value = false
    syncDefinition.value = config.value?.definition || ''
    syncRevision.value = mode === 'join'
    clearPlan()
    syncVisible.value = true
    return true
  }
  function openJoin() {
    beginTransfer('join')
  }
  async function loadTransferTarget() {
    if (syncMode.value !== 'join' || !targetId.value || !props.service) return
    const request = transferGeneration,
      id = targetId.value,
      sid = props.service.id
    transferReading.value = true
    try {
      const result = await getServiceEnvironmentConfigApi(id, sid)
      if (request !== transferGeneration || sid !== props.service?.id) return
      if (result.environment_id !== id || result.service_id !== sid)
        throw new Error('目标配置不匹配，请重新加载')
      transferTarget.value = result
    } catch (e: any) {
      if (request === transferGeneration)
        syncError.value = e?.message || '目标配置读取失败，请重新选择环境'
    } finally {
      transferReading.value = false
    }
  }
  async function viewTransferTarget() {
    if (busy.value || !targetId.value) return
    const id = targetId.value
    syncVisible.value = false
    await switchEnvironment(id)
  }
  function openSync() {
    if (!environmentId.value || !config.value?.bound) return openJoin()
    beginTransfer('sync')
  }
  function joinCurrent() {
    if (
      props.service?.delivery_type === 'k8s_yaml' &&
      environmentId.value &&
      !config.value?.bound
    ) {
      environmentCreationVisible.value = true
      return
    }
    if (!config.value || config.value.bound || !sourceRevision.value || busy.value) return
    if (beginTransfer('join')) {
      targetIds.value = [environmentId.value]
      void loadTransferTarget()
    }
  }
  async function previewSync() {
    if (busy.value) return
    clearPlan()
    if (
      !props.service ||
      !targetIds.value.length ||
      (!syncRevision.value && !syncKeys.value.length && !syncYaml.value)
    )
      return
    const request = transferGeneration,
      sid = props.service.id
    previewing.value = true
    try {
      if (targetIds.value.length !== 1) throw new Error('请一次选择一个目标环境')
      const id = targetIds.value[0]
      if (!transferEnvironments.value.some((e) => e.id === id) || !canUpdate(id))
        throw new Error('目标环境不可修改')
      if (syncMode.value === 'sync' && id === environmentId.value)
        throw new Error('来源和目标环境不能相同')
      if (
        targetEnvironment.value?.binding_status &&
        targetEnvironment.value.binding_status !== 'validated'
      )
        throw new Error('请先完成目标环境的集群绑定与校验')
      const target = await getServiceEnvironmentConfigApi(id, sid)
      if (request !== transferGeneration || sid !== props.service?.id) return
      if (target.environment_id !== id || target.service_id !== sid)
        throw new Error('目标配置不匹配，请重新加载')
      if (syncMode.value === 'join' && target.bound)
        throw new Error('此服务已加入该环境，请切换到该环境查看或使用“同步配置”')
      if (syncMode.value === 'sync' && !target.bound)
        throw new Error('此服务尚未加入目标环境，请先使用“加入环境”')
      const values = configValues(target)
      const sourceValues = config.value ? configValues(config.value) : {}
      if (syncKeys.value.some((key) => values[key]?.secret || sourceValues[key]?.secret))
        throw new Error('密码和凭据不能跨环境复制，请在目标环境单独设置')
      if (syncKeys.value.some((key) => sourceValue(key) === undefined))
        throw new Error('来源参数不存在，请重新选择')
      const next = [{ config: target, values }]
      const payload = syncRequest(next)
      const result = await saveServiceEnvironmentConfigApi(sid, payload, true)
      if (request !== transferGeneration || sid !== props.service?.id) return
      const after = result?.configs?.find((c) => c.environment_id === id && c.service_id === sid)
      if (
        !result?.validated ||
        result.saved !== false ||
        result.configs.length !== 1 ||
        !after?.bound
      )
        throw new Error('配置预览返回不完整，请重新加载')
      plannedRequest.value = payload
      plan.value = [{ ...next[0], after }]
    } catch (e: any) {
      if (request !== transferGeneration || sid !== props.service?.id) return
      syncError.value = e?.message || '差异读取失败'
    } finally {
      previewing.value = false
    }
  }
  async function applySync() {
    if (
      !props.service ||
      !plan.value.length ||
      !plannedRequest.value ||
      busy.value ||
      (productionTargets.value.length && !productionConfirmed.value) ||
      (syncYaml.value && !yamlConfirmed.value)
    )
      return
    if (JSON.stringify(plannedRequest.value) !== JSON.stringify(syncRequest(plan.value))) {
      clearPlan()
      syncError.value = '同步内容已变化，请重新查看差异'
      return
    }
    saving.value = true
    const request = transferGeneration,
      sid = props.service.id
    try {
      const id = plan.value[0].config.environment_id
      const result = await saveServiceEnvironmentConfigApi(sid, plannedRequest.value!)
      if (request !== transferGeneration || sid !== props.service?.id) return
      if (
        !result?.saved ||
        !result.validated ||
        result.configs.length !== 1 ||
        result.configs[0].environment_id !== id ||
        result.configs[0].service_id !== sid
      )
        throw new Error('无法确认保存结果，请重新加载目标环境核对')
      syncVisible.value = false
      ElMessage.success(
        syncMode.value === 'join'
          ? '已加入环境，请核对配置后再部署'
          : '目标环境配置已保存，尚未部署'
      )
      if (syncMode.value === 'join') environmentId.value = id
      if (environmentId.value) await loadConfig()
      emit('environment-saved')
    } catch (e: any) {
      if (request !== transferGeneration || sid !== props.service?.id) return
      syncError.value = e?.message || '同步失败'
      plan.value = []
    } finally {
      saving.value = false
    }
  }
  function syncRequest(targets: typeof plan.value): SaveServiceEnvironmentConfig {
    return {
      ...(syncMode.value === 'sync' && config.value
        ? { source: { environment_id: environmentId.value, token: config.value.token } }
        : {}),
      targets: targets.map((item) => ({
        environment_id: item.config.environment_id,
        token: item.config.token
      })),
      revision_id: syncRevision.value ? transferRevision.value : 0,
      ...(syncYaml.value ? { definition: syncDefinition.value } : {}),
      variables: Object.fromEntries(
        syncMode.value === 'join'
          ? Object.entries(joinOverrides.value)
          : syncKeys.value.map((key) => [key, sourceValue(key)!])
      )
    }
  }
  watch(
    [
      targetIds,
      syncKeys,
      syncRevision,
      syncYaml,
      syncDefinition,
      syncMode,
      transferRevision,
      draft,
      joinOverrides
    ],
    clearPlan,
    {
      deep: true,
      flush: 'sync'
    }
  )
  watch(
    targetIds,
    () => {
      transferTarget.value = null
      joinOverrides.value = {}
    },
    { deep: true, flush: 'sync' }
  )
  watch(
    () =>
      props.environments
        .map((e) => [e.id, e.cluster_id, e.namespace, e.is_production, e.binding_status].join(':'))
        .join('|'),
    clearPlan
  )
  watch(
    () => props.service?.id,
    () => {
      generation++
      clearPlan()
      syncVisible.value = false
      assetsEnvironmentVisible.value = false
      assetsEnvironmentId.value = undefined
      if (props.service?.delivery_type === 'host') environmentId.value = 0
      else if (!props.environments.some((env) => env.id === environmentId.value))
        environmentId.value = props.environments.some(
          (env) => env.id === props.preferredEnvironmentId
        )
          ? props.preferredEnvironmentId!
          : props.environments[0]?.id || 0
      void loadConfig()
    },
    { immediate: true }
  )
  watch(
    () => props.environments.map((item) => item.id).join(','),
    () => {
      if (
        props.service?.delivery_type !== 'host' &&
        !props.environments.some((item) => item.id === environmentId.value)
      ) {
        environmentId.value = props.environments[0]?.id || 0
        void loadConfig()
      }
    },
    { immediate: true }
  )
  function effectiveDiff(item: (typeof plan.value)[number]) {
    if (!item.after) return []
    const after = configValues(item.after)
    const show = (value?: { value: string; secret: boolean }) =>
      value?.secret ? '凭据引用' : (value?.value ?? '未配置')
    return [...new Set([...Object.keys(item.values), ...Object.keys(after)])]
      .sort()
      .map((key) => ({ key, before: show(item.values[key]), after: show(after[key]) }))
      .filter((row) => row.before !== row.after)
  }
  onBeforeUnmount(() => {
    generation++
    transferGeneration++
  })
  const openNewYaml = async () => {
    if (environmentId.value && config.value?.format === 'k8s_yaml') {
      renderValues.value = false
      await nextTick()
      sharedEditor.value?.openNewYaml()
      return
    }
    if (props.service?.delivery_type !== 'host') {
      ElMessage.warning('请先选择环境并等待配置加载完成')
      return
    }
    environmentId.value = 0
    await nextTick()
    sharedEditor.value?.openNewYaml()
  }
  const reloadSharedTemplate = async () => {
    await loadConfig()
    await nextTick()
    await sharedEditor.value?.reloadDefinition()
  }
  const openRepositorySync = async () => {
    if (props.service?.delivery_type !== 'helm') return
    const serviceId = props.service.id
    await loadConfig()
    await nextTick()
    if (props.service?.id === serviceId) await helmEditor.value?.openRepositorySync()
  }
  watch(
    () => props.preferredEnvironmentId,
    async (id) => {
      if (id && id !== environmentId.value && props.environments.some((env) => env.id === id))
        await switchEnvironment(id)
    }
  )
  defineExpose({
    confirmLeave,
    openNewYaml,
    switchEnvironment,
    reloadSharedTemplate,
    openRepositorySync
  })
</script>

<style scoped>
  :global(.environment-transfer-dialog) {
    display: flex;
    flex-direction: column;
    max-height: 90vh;
    margin: 5vh auto;
  }
  :global(.environment-transfer-dialog .el-dialog__body) {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
  }
  :global(.environment-transfer-dialog .el-dialog__header),
  :global(.environment-transfer-dialog .el-dialog__footer) {
    flex: none;
  }
  :global(.environment-transfer-dialog .el-dialog__footer) {
    display: flex;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: 8px;
  }
  :global(.environment-transfer-dialog .el-dialog__footer .el-button + .el-button) {
    margin-left: 0;
  }
  .join-fields {
    display: grid;
    gap: 12px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .join-fields label {
    display: grid;
    gap: 5px;
    font-size: 13px;
  }
  .join-fields code {
    font-size: 11px;
    color: #64748b;
  }
  .transfer-target {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 10px;
    margin: 16px 0;
  }
  .transfer-target :deep(.el-select) {
    width: 230px;
    max-width: 100%;
  }
  .target-location {
    width: 100%;
    font-size: 12px;
    color: #64748b;
    overflow-wrap: anywhere;
  }
  .transfer-options,
  .transfer-parameters,
  .yaml-sync-choice {
    margin: 14px 0;
  }
  .transfer-options small {
    display: block;
    color: #64748b;
  }
  .transfer-parameters summary {
    cursor: pointer;
    padding: 10px 0;
    font-weight: 500;
  }
  .transfer-parameters summary span {
    font-size: 12px;
    font-weight: normal;
    color: #64748b;
    margin-left: 8px;
  }
  .transfer-fields {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
  .transfer-fields :deep(.el-checkbox) {
    height: auto;
    margin: 0;
    align-items: flex-start;
    padding: 8px;
    background: #f8fafc;
    border-radius: 6px;
  }
  .transfer-fields :deep(.el-checkbox__label) {
    display: grid;
    gap: 3px;
    min-width: 0;
    white-space: normal;
    overflow-wrap: anywhere;
  }
  .transfer-fields code,
  .transfer-fields small {
    color: #64748b;
    font-size: 12px;
  }
  .transfer-yaml {
    margin-top: 10px;
  }
  .transfer-yaml :deep(textarea),
  .transfer-manifest {
    font: 12px/1.6 var(--ax-font-mono);
  }
  .transfer-manifest {
    max-height: 32vh;
    overflow: auto;
    padding: 10px;
    background: #f8fafc;
  }
  @media (max-width: 600px) {
    .join-fields {
      grid-template-columns: 1fr;
    }
    .transfer-fields {
      grid-template-columns: 1fr;
    }
  }
  .sync-summary {
    display: grid;
    gap: 6px;
    padding: 12px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 13px;
  }
  .preview-columns {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: 12px;
    margin: 12px 0;
  }
  .configuration-tabs {
    padding: 0 16px 12px;
  }
  .preview-columns pre {
    max-height: 48vh;
    overflow: auto;
    padding: 12px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font: 12px/1.6 var(--ax-font-mono);
    white-space: pre;
  }
  .release-target {
    padding: 12px 0;
  }
  .image-line {
    display: flex;
    flex-direction: column;
    gap: 4px;
    color: #475569;
    font-size: 12px;
    overflow-wrap: anywhere;
  }
  @media (max-width: 800px) {
    .preview-columns {
      grid-template-columns: 1fr;
    }
  }
  .sync-summary span {
    color: #475569;
    overflow-wrap: anywhere;
  }
  .target-shortcuts {
    margin-bottom: 6px;
  }
  .configuration-panel {
    min-width: 0;
    width: 100%;
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: #fff;
  }
  .configuration-panel > :deep(.service-workbench) {
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
  }
  .context-bar,
  .environment-summary,
  .definition-heading,
  .variable-input,
  .variable-label,
  .add-variable {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .context-bar {
    flex: 0 0 auto;
    padding: 10px 14px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    background: var(--el-fill-color-extra-light);
    flex-wrap: wrap;
  }
  .environment-select {
    width: 200px;
  }
  .context-hint {
    flex: 1;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
  .environment-summary {
    padding: 12px 16px;
    color: var(--el-text-color-regular);
    font-size: 13px;
    flex-wrap: wrap;
  }
  .environment-summary span:nth-child(2) {
    flex: 1;
  }
  .environment-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 340px;
    min-height: 420px;
  }
  .definition-view {
    min-width: 0;
    padding: 12px 16px;
    border-right: 1px solid #e5eaf1;
  }
  .definition-heading {
    justify-content: space-between;
    font-size: 13px;
  }
  .note {
    color: #64748b;
    font-size: 12px;
    line-height: 1.6;
  }
  .yaml-preview {
    overflow: auto;
    max-height: calc(100vh - 360px);
    min-height: 320px;
    padding: 14px;
    background: #f8fafc;
    font: 12px/1.8 var(--ax-font-mono);
  }
  .environment-variables {
    padding: 16px;
    overflow: auto;
    max-height: calc(100vh - 260px);
  }
  .variable-row {
    margin: 16px 0;
  }
  .variable-label {
    justify-content: space-between;
    margin-bottom: 6px;
  }
  .variable-label small,
  .diff-target small {
    color: #64748b;
    font-size: 11px;
  }
  .variable-label code {
    overflow-wrap: anywhere;
  }
  .diff-target {
    margin-top: 18px;
  }
  .effective-diff {
    margin-top: 12px;
  }
  details {
    margin-top: 10px;
    font-size: 12px;
  }
  .template-diff {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }
  .template-diff pre {
    max-height: 280px;
    overflow: auto;
    background: #f8fafc;
    padding: 10px;
  }
  .diff-target small {
    margin-left: 12px;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
    margin-top: 8px;
  }
  td,
  th {
    text-align: left;
    border: 1px solid #e5eaf1;
    padding: 8px;
    overflow-wrap: anywhere;
  }
  @media (max-width: 1050px) {
    .environment-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>

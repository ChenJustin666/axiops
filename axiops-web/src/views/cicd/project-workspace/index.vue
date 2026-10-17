<template>
  <div v-loading="loading" class="cicd-workspace">
    <ProjectNavBar
      :project-id="projectId"
      :project="project"
      :counts="navCounts"
      :active="navTab"
      :show-tabs="project?.status !== 'deleted'"
      :auto-navigate="false"
      refreshable
      :refreshing="loading"
      @select="selectTab"
      @refresh="loadWorkspace"
    />
    <ArchivedProjectHistory v-if="project?.status === 'deleted'" :project-id="projectId" />
    <template v-else>
      <section v-if="initializationMode && !loadError" class="initialization-bar">
        <div class="initialization-title"
          ><strong>项目初始化</strong><span>在同一个项目工作区完成首次配置</span></div
        >
        <div class="initialization-steps">
          <button
            v-for="step in initSteps"
            :key="step.value"
            type="button"
            :class="[
              'init-step',
              { 'is-active': activeTab === step.value, 'is-complete': step.complete }
            ]"
            @click="selectInitStep(step.value)"
          >
            <span>{{ step.index }}</span
            ><div
              ><strong>{{ step.label }}</strong
              ><small>{{ step.description }}</small></div
            ><Check v-if="step.complete" :size="15" />
          </button>
        </div>
        <ElTag size="small" type="success" effect="plain">{{ initializationProgress }}</ElTag>
      </section>

      <ElAlert
        v-if="workspaceWarnings.length"
        type="warning"
        :closable="false"
        title="部分数据暂时无法读取"
      >
        <div v-for="warning in workspaceWarnings" :key="warning">{{ warning }}</div>
        <ElButton text type="primary" :loading="loading" @click="loadWorkspace">重新加载</ElButton>
      </ElAlert>
      <main v-if="project && !loadError" class="workspace-main">
        <section
          v-if="activeTab === 'services'"
          class="services-layout"
          @click="closeServiceContextMenu"
        >
          <aside class="service-catalog">
            <div class="catalog-actions" aria-label="服务配置操作">
              <ElButton
                v-if="canCreateService && projectDeliveryType !== 'helm'"
                text
                size="small"
                :icon="Plus"
                title="手动创建服务目录"
                @click="beginManualDirectory"
                >手动</ElButton
              >
              <ElButton
                v-if="canImport && projectDeliveryType !== 'host'"
                text
                size="small"
                :icon="PackageOpen"
                title="从模板库选择模板"
                @click="openTemplateImport"
                >模板</ElButton
              >
              <ElButton
                v-if="canImport"
                text
                size="small"
                :icon="GitBranch"
                title="从代码仓库导入服务"
                @click="openImport"
                >仓库</ElButton
              >
            </div>
            <ElInput
              v-model="keyword"
              size="small"
              clearable
              :prefix-icon="Search"
              placeholder="搜索服务"
            />
            <div class="catalog-secondary"
              ><ElButton text size="small" @click="openServiceLifecycle('trash')"
                ><ArchiveRestore :size="13" />垃圾桶</ElButton
              ></div
            >
            <div class="service-tree">
              <div v-for="service in filteredServices" :key="service.id" class="service-node">
                <div
                  :class="['service-row', { selected: selectedService?.id === service.id }]"
                  @contextmenu.stop.prevent="openServiceContextMenu($event, service)"
                >
                  <button
                    type="button"
                    :class="[
                      'service-node-button',
                      { 'is-active': selectedService?.id === service.id }
                    ]"
                    :aria-expanded="isServiceExpanded(service.id)"
                    @click="toggleService(service)"
                  >
                    <component
                      :is="isServiceExpanded(service.id) ? ChevronDown : ChevronRight"
                      :size="14"
                    /><span class="service-icon"
                      ><ArtDeliveryIcon :type="service.delivery_type" :size="14"
                    /></span>
                    <span class="service-identity"
                      ><strong :title="service.name">{{ service.name }}</strong
                      ><small :title="service.code">{{ service.code }}</small></span
                    >
                    <span class="service-type">{{
                      service.delivery_type === 'helm'
                        ? 'Helm'
                        : service.delivery_type === 'host'
                          ? '主机'
                          : 'YAML'
                    }}</span>
                  </button>
                  <ElDropdown
                    v-if="
                      canManageService(service, 'update') || canManageService(service, 'delete')
                    "
                    trigger="click"
                    @command="
                      (command: string) =>
                        command === 'template'
                          ? openTemplateReplacement(service)
                          : openServiceLifecycle(command as 'rename' | 'delete', service)
                    "
                  >
                    <ElButton
                      text
                      size="small"
                      class="service-more"
                      :aria-label="`${service.name} 更多操作`"
                      title="更多操作"
                      ><MoreHorizontal :size="16"
                    /></ElButton>
                    <template #dropdown
                      ><ElDropdownMenu
                        ><ElDropdownItem v-if="canManageService(service, 'update')" command="rename"
                          >编辑名称</ElDropdownItem
                        ><ElDropdownItem
                          v-if="
                            service.delivery_type !== 'host' && canManageService(service, 'update')
                          "
                          command="template"
                          >更换模板</ElDropdownItem
                        ><ElDropdownItem
                          v-if="canManageService(service, 'delete')"
                          command="delete"
                          divided
                          >删除服务</ElDropdownItem
                        ></ElDropdownMenu
                      ></template
                    >
                  </ElDropdown>
                </div>
                <div v-if="isServiceExpanded(service.id)" class="file-list">
                  <button
                    v-for="file in serviceFiles(service)"
                    :key="file"
                    type="button"
                    :class="[
                      'file-item',
                      { 'is-active': selectedService?.id === service.id && activeFile === file }
                    ]"
                    :aria-current="
                      selectedService?.id === service.id && activeFile === file
                        ? 'location'
                        : undefined
                    "
                    @click="selectFile(file, service)"
                  >
                    <FileCode2 :size="13" /><span>{{
                      file === '@environment-values' ? '当前环境 values' : file
                    }}</span>
                  </button>
                </div>
              </div>
              <div v-if="manualDirectoryOpen" class="manual-directory-row is-inline">
                <ElInput
                  v-model="manualDirectoryName"
                  size="small"
                  autofocus
                  placeholder="服务名称，如 risk_server"
                  maxlength="128"
                  :title="SERVICE_NAME_HINT"
                  @keyup.enter="createManualDirectory"
                />
                <ElButton
                  text
                  type="success"
                  size="small"
                  :loading="manualDirectorySaving"
                  :icon="Check"
                  title="保存服务目录"
                  @click="createManualDirectory"
                />
                <ElButton text size="small" title="取消" @click="cancelManualDirectory"
                  >取消</ElButton
                >
              </div>
              <ElText v-if="manualDirectoryOpen" size="small" type="info">{{
                SERVICE_NAME_HINT
              }}</ElText>
              <ElEmpty v-if="!filteredServices.length" :image-size="50" description="暂无服务" />
            </div>
          </aside>
          <Teleport to="body">
            <div
              v-if="serviceContextMenu"
              class="service-context-menu"
              :style="{ left: `${serviceContextMenu.x}px`, top: `${serviceContextMenu.y}px` }"
              role="menu"
              @click.stop
            >
              <div class="context-menu-heading">{{ serviceContextMenu.service.name }}</div>
              <button
                v-if="
                  serviceContextMenu.service.delivery_type === 'k8s_yaml' &&
                  canManageService(serviceContextMenu.service, 'update')
                "
                type="button"
                role="menuitem"
                @click="openServiceYamlFromContext"
              >
                <FileCode2 :size="14" />新建 YAML
              </button>
              <button
                v-if="
                  serviceContextMenu.service.delivery_type !== 'host' &&
                  canManageService(serviceContextMenu.service, 'update')
                "
                type="button"
                role="menuitem"
                @click="openServiceTemplateFromContext"
              >
                <PackageOpen :size="14" />更换模板
              </button>
              <button
                v-if="canImport && canManageService(serviceContextMenu.service, 'update')"
                type="button"
                role="menuitem"
                @click="openServiceRepositoryFromContext"
              >
                <GitBranch :size="14" />从仓库导入
              </button>
              <div class="context-menu-divider" />
              <button type="button" role="menuitem" @click="closeServiceContextMenu">取消</button>
            </div>
          </Teleport>
          <ServiceConfigurationPanel
            ref="serviceConfiguration"
            :service="selectedService"
            :project-name="project.name"
            :project-code="project.code"
            :active-file="activeFile"
            :environments="environments"
            :preferred-environment-id="selectedEnvironmentId || runtimeEnvironmentId"
            :clusters="clusters"
            :can-edit="selectedServiceCanEdit"
            :can-import="canImport"
            :can-view-build="canViewBuild"
            :workflows="workflows"
            @import="openImport"
            @template-import="openTemplateImport"
            @build-config="openBuildConfig"
            @file-change="selectFile"
            @files-change="
              (files: string[]) => selectedService && (helmFilePaths[selectedService.id] = files)
            "
            @saved="handleServiceSaved"
            @environment-saved="loadWorkspace"
            @published="
              ({ workflowId, runId }) =>
                router.push(`/cicd/projects/${projectId}/workflows/${workflowId}/runs/${runId}`)
            "
          />
        </section>

        <section v-else-if="activeTab === 'environments'" class="content-panel">
          <div class="panel-heading"
            ><div
              ><span class="view-eyebrow">ENVIRONMENTS</span
              ><h2>{{ initializationMode ? '环境配置' : '运行资源' }}</h2
              ><p>{{
                initializationMode
                  ? '初始化阶段配置环境和部署目标。'
                  : '选择环境，查看目标集群和命名空间中的实际资源。'
              }}</p></div
            ><ElButton
              v-if="projectDeliveryType !== 'host' && canEdit"
              type="success"
              :icon="Plus"
              @click="openEnvironmentCreator"
              >新建环境</ElButton
            ></div
          >
          <template v-if="initializationMode">
            <div
              v-if="!environments.length && initializationMode && projectDeliveryType !== 'host'"
              class="default-env-hint"
              ><div
                ><strong>默认只创建一个测试环境</strong
                ><span>不更改类型即为测试；也可选择生产、开发或自定义环境</span></div
              ><ElButton
                type="success"
                :disabled="!clusters.length"
                @click="createDefaultEnvironments"
                >创建测试环境</ElButton
              ></div
            >
            <div v-if="environments.length" class="environment-workspace">
              <div class="environment-tabs" role="tablist" aria-label="环境切换">
                <button
                  v-for="environment in environments"
                  :key="environment.id"
                  type="button"
                  :class="[
                    'environment-tab',
                    { 'is-active': currentEnvironment?.id === environment.id }
                  ]"
                  @click="selectEnvironment(environment)"
                >
                  <span>{{ environment.name }}</span
                  ><small>{{ environment.code }}</small>
                </button>
                <button
                  type="button"
                  class="environment-tab environment-add"
                  @click="openEnvironmentCreator"
                >
                  <Plus :size="14" /> 添加环境
                </button>
              </div>
              <template v-if="currentEnvironment">
                <div class="environment-binding-panel">
                  <div class="binding-field"
                    ><label>{{ projectDeliveryType === 'host' ? '主机组' : 'K8s 集群' }}</label
                    ><strong>{{
                      projectDeliveryType === 'host'
                        ? currentEnvironment.host_group_ref || '-'
                        : clusterName(currentEnvironment.cluster_id)
                    }}</strong></div
                  >
                  <div class="binding-field"
                    ><label>{{
                      projectDeliveryType === 'host' ? '部署目标' : 'K8s 命名空间'
                    }}</label
                    ><strong>{{
                      projectDeliveryType === 'host'
                        ? '主机应用发布'
                        : currentEnvironment.namespace || '-'
                    }}</strong></div
                  >
                  <div class="binding-field"
                    ><label>镜像配置</label><strong>继承服务构建配置</strong></div
                  >
                  <div class="binding-field"
                    ><label>关联服务</label><strong>{{ services.length }} 个服务</strong></div
                  >
                  <div class="binding-actions"
                    ><ElButton
                      size="small"
                      type="primary"
                      plain
                      @click="openEnvironmentConfig(currentEnvironment)"
                      >环境公共配置</ElButton
                    ></div
                  >
                </div>
                <div class="environment-service-list">
                  <div class="section-heading"
                    ><div><strong>服务列表</strong><span>服务配置</span></div></div
                  >
                  <article
                    v-for="service in services"
                    :key="service.id"
                    class="environment-service-card"
                  >
                    <div class="environment-service-head"
                      ><div class="service-title"
                        ><span class="service-dot"><Boxes :size="15" /></span
                        ><strong>{{ service.name }}</strong
                        ><small>{{ service.code }}</small></div
                      ><div class="resource-links"
                        ><template v-if="projectDeliveryType === 'k8s_yaml'"
                          ><span class="resource-pill">Deployment/{{ service.code }}</span
                          ><span class="resource-pill">Service/{{ service.code }}</span></template
                        ><span v-else-if="projectDeliveryType === 'helm'" class="resource-pill"
                          >Helm Release/{{ service.code }}</span
                        ><span v-else class="resource-pill">主机服务/{{ service.code }}</span></div
                      ><ElButton
                        size="small"
                        type="primary"
                        plain
                        @click="openEnvironmentServiceConfiguration(service.id)"
                        >配置</ElButton
                      ></div
                    >
                    <div class="environment-service-fields"
                      ><div
                        ><label>{{ projectDeliveryType === 'host' ? '制品' : '镜像' }}</label
                        ><span class="image-value">{{
                          service.image_name ||
                          (projectDeliveryType === 'host' ? '使用构建制品' : '继承服务构建镜像')
                        }}</span></div
                      ><div
                        ><label>{{ projectDeliveryType === 'host' ? '主机组' : '端口' }}</label
                        ><span>{{
                          projectDeliveryType === 'host'
                            ? currentEnvironment.host_group_ref || '待配置'
                            : '由服务变量配置'
                        }}</span></div
                      ><div
                        ><label>资源</label
                        ><span>{{
                          projectDeliveryType === 'host' ? '节点与并发策略' : 'CPU / 内存限制'
                        }}</span></div
                      ><div
                        ><label>{{ projectDeliveryType === 'host' ? '健康检查' : '协议' }}</label
                        ><span>{{
                          projectDeliveryType === 'host' ? '部署后检测' : 'TCP'
                        }}</span></div
                      ></div
                    >
                  </article>
                  <ElEmpty
                    v-if="!services.length"
                    :image-size="56"
                    description="请先在服务配置中添加服务"
                  />
                </div>
              </template>
            </div>
            <ElEmpty v-else description="暂无环境，请先选择集群并创建环境" />
          </template>
          <template v-else>
            <div v-if="environments.length" class="environment-workspace">
              <div class="environment-tabs" role="tablist" aria-label="选择运行环境">
                <button
                  v-for="environment in environments"
                  :key="environment.id"
                  type="button"
                  role="tab"
                  :aria-selected="runtimeEnvironmentId === environment.id"
                  :class="[
                    'environment-tab',
                    { 'is-active': runtimeEnvironmentId === environment.id }
                  ]"
                  @click="selectRuntimeEnvironment(environment)"
                >
                  <span>{{ environment.name }}</span
                  ><small>{{ environment.code }}</small>
                  <ElTag v-if="environment.is_production" size="small" type="danger">生产</ElTag>
                </button>
                <button
                  v-if="projectDeliveryType !== 'host' && canEdit"
                  type="button"
                  class="environment-tab environment-add"
                  @click="openEnvironmentCreator"
                >
                  <Plus :size="14" /> 添加环境
                </button>
              </div>
              <div
                v-if="projectDeliveryType === 'k8s_yaml' && services.length && canEdit"
                class="runtime-config-actions"
              >
                <ElButton
                  size="small"
                  type="primary"
                  plain
                  @click="
                    openEnvironmentServiceConfiguration(selectedService?.id || services[0].id)
                  "
                >
                  编辑当前环境 YAML
                </ElButton>
              </div>
              <EnvironmentRuntimePanel
                v-if="runtimeEnvironment"
                :environment="runtimeEnvironment"
                :cluster-name="clusterName(runtimeEnvironment.cluster_id)"
                :delivery-type="projectDeliveryType"
              />
              <ElEmpty v-else description="请选择环境查看实际运行资源" />
            </div>
            <ElEmpty v-else description="暂无环境，请先新建环境" />
          </template>
        </section>

        <section v-else-if="activeTab === 'build'" class="content-panel">
          <div class="panel-heading"
            ><div
              ><span class="view-eyebrow">BUILD SETUP</span><h2>构建配置</h2
              ><p
                >选择代码、构建方式和镜像仓库即可。资源、缓存和变量按需展开；配置完整后前往工作流执行。</p
              ></div
            ><ElButton type="success" :icon="Plus" @click="selectTab('services')"
              >添加服务</ElButton
            ></div
          >
          <div class="build-list">
            <article v-for="service in services" :key="service.id" class="build-card">
              <div class="build-service"
                ><span class="build-icon"><Hammer :size="18" /></span
                ><div
                  ><strong>{{ service.name }}</strong
                  ><small>{{ service.code }}</small></div
                ></div
              >
              <div class="build-source"
                ><span>代码仓库</span><strong>{{ service.repo_url || '待配置' }}</strong
                ><small>{{ service.repo_branch || 'main' }}</small></div
              >
              <div class="build-readiness">
                <ElTag
                  :type="isBuildConfigured(service.id) ? 'success' : 'warning'"
                  effect="plain"
                  >{{ isBuildConfigured(service.id) ? '配置完整' : '待检查' }}</ElTag
                >
                <small>{{
                  buildIssues[service.id]?.join('；') || '实际构建会校验连接、权限和资源'
                }}</small>
              </div>
              <ElButton type="success" plain @click="openBuildConfig(service)">{{
                isBuildConfigured(service.id) ? '编辑构建' : '配置构建'
              }}</ElButton>
            </article>
            <ElEmpty v-if="!services.length" description="请先创建服务" />
          </div>
        </section>

        <section v-else-if="activeTab === 'workflows'" class="content-panel">
          <div class="panel-heading"
            ><div
              ><span class="view-eyebrow">WORKFLOWS</span><h2>工作流</h2
              ><p
                >环境创建时已绑定集群与命名空间；工作流只展示目标环境，构建流程可独立运行。</p
              ></div
            ><div class="heading-actions"
              ><ElButton plain @click="activeTab = 'build'">构建配置</ElButton
              ><ElButton type="primary" @click="workflowCreationVisible = true">新建工作流</ElButton
              ><ElButton
                plain
                :loading="workflowGenerating"
                :disabled="!services.length || !environments.length"
                @click="generateWorkflows"
                >{{ workflows.length ? '补齐环境工作流' : '生成环境工作流' }}</ElButton
              ></div
            ></div
          >
          <div v-if="workflows.length" class="workflow-list">
            <article
              v-for="workflow in workflows"
              :key="workflow.id"
              class="workflow-card"
              @click="openWorkflowJob(workflow)"
            >
              <span class="workflow-icon"><Workflow :size="19" /></span>
              <div class="workflow-info">
                <strong>{{ workflow.name }}</strong>
                <small>{{ workflow.code }} · {{ workflow.node_count }} 个任务</small>
              </div>
              <div class="workflow-target"
                ><span>目标环境</span><strong>{{ workflowEnvironmentName(workflow) }}</strong
                ><small v-if="workflowEnvironment(workflow)">
                  {{ clusterName(workflowEnvironment(workflow)!.cluster_id) }} ·
                  {{
                    workflowEnvironment(workflow)!.namespace ||
                    workflowEnvironment(workflow)!.host_group_ref
                  }}
                </small></div
              >
              <ElTag :type="workflowStatusType(workflow.id)" effect="plain">{{
                workflowStatusLabel(workflow.id)
              }}</ElTag>
              <div class="workflow-card-actions">
                <ElButton
                  type="primary"
                  :icon="Play"
                  title="执行工作流"
                  aria-label="执行工作流"
                  @click.stop="openWorkflowExecute(workflow)"
                  >执行</ElButton
                >
              </div>
            </article>
          </div>
          <ElEmpty v-else description="完成服务、环境和构建配置后生成工作流" />
          <ElAlert
            v-if="unboundDeploymentWorkflows.length"
            type="warning"
            title="有工作流缺少目标环境记录"
            :closable="false"
          >
            <div v-for="item in unboundDeploymentWorkflows" :key="item.id">
              <ElButton
                link
                type="primary"
                @click="router.push(`/cicd/projects/${projectId}/workflows/${item.id}?configure=1`)"
                >{{ item.name }} · 请检查项目初始化环境绑定</ElButton
              >
            </div>
          </ElAlert>
          <div v-if="initializationMode && workflows.length" class="initial-run-box"
            ><div
              ><strong>{{
                unboundDeploymentWorkflows.length ? '存在待修复的目标关联' : '配置已准备'
              }}</strong
              ><span
                >环境已在初始化阶段绑定集群和 Namespace；执行阶段只核对已保存目标并运行。</span
              ></div
            ><ElButton type="success" :loading="runningAll" @click="completeInitialization"
              >完成配置</ElButton
            ></div
          >
        </section>

        <section v-else class="content-panel">
          <div class="panel-heading"
            ><div
              ><span class="view-eyebrow">AXIOPS PROJECT</span><h2>{{ tabTitle }}</h2
              ><p>{{ tabDescription }}</p></div
            ></div
          >
          <div v-if="activeTab === 'overview'" class="overview-page">
            <div class="mini-metrics">
              <button type="button" @click="activeTab = 'services'"
                >服务 <b>{{ services.length }}</b></button
              ><button type="button" @click="activeTab = 'environments'"
                >环境 <b>{{ environments.length }}</b></button
              ><button type="button" @click="activeTab = 'workflows'"
                >工作流 <b>{{ workflows.length }}</b></button
              >
            </div>

            <section class="overview-block">
              <div class="block-head"
                ><strong>工作流</strong><span>构建趋势 · 平均耗时 · 成功率 · 最近执行</span></div
              >
              <div v-if="workflows.length" class="workflow-table">
                <div v-for="workflow in workflows" :key="workflow.id" class="wf-row">
                  <div class="wf-name">
                    <button type="button" class="row-link" @click="openWorkflowJob(workflow)">{{
                      workflow.name
                    }}</button>
                    <small>{{ workflowEnvironmentName(workflow) }}</small>
                  </div>
                  <div class="wf-trend">
                    <ProjectRunTrend
                      v-if="workflowRunList(workflow.id).length"
                      :runs="workflowRunList(workflow.id).map((run) => ({ run, workflow }))"
                      compact
                      @select="({ run }) => openWorkflowRun(run.id, workflow)"
                    />
                    <span v-else class="trend-empty">—</span>
                  </div>
                  <div class="wf-metrics">
                    <div class="metric">
                      <small>平均执行时间</small><b>{{ workflowAvgDuration(workflow.id) }}</b>
                    </div>
                    <div class="metric">
                      <small>成功率</small><b>{{ workflowSuccessRate(workflow.id) }}</b>
                    </div>
                    <div class="metric">
                      <small>最近执行</small><b>{{ workflowLastRunTime(workflow.id) }}</b>
                    </div>
                  </div>
                  <div class="wf-action">
                    <ElButton
                      type="primary"
                      plain
                      :icon="Play"
                      title="执行工作流"
                      @click="openWorkflowExecute(workflow)"
                      >执行</ElButton
                    >
                  </div>
                </div>
              </div>
              <ElEmpty v-else description="还没有工作流，先到「工作流」生成" />
            </section>

            <section class="overview-block">
              <div class="block-head"
                ><strong>最近运行</strong><span>最近 {{ overviewRuns.length }} 次</span></div
              >
              <div v-if="overviewRuns.length" class="overview-list">
                <button
                  v-for="item in overviewRuns"
                  :key="item.run.id"
                  type="button"
                  class="list-row is-button"
                  @click="openWorkflowRun(item.run.id, item.workflow)"
                >
                  <ElTag size="small" effect="plain" :type="runStatusType(item.run.status)">{{
                    runStatusLabel(item.run.status)
                  }}</ElTag>
                  <strong>#{{ item.run.run_number || '—' }}</strong>
                  <span class="row-name">{{ item.workflow?.name || '工作流' }}</span>
                  <small
                    >{{ environmentLabel(item.run.environment_id) }} ·
                    {{ formatRunDate(item.run.created_at) }}</small
                  >
                </button>
              </div>
              <ElEmpty v-else description="还没有运行记录，执行工作流后显示结果" />
            </section>
          </div>
          <ElEmpty
            v-else
            :description="activeTab === 'tests' ? '测试配置将在这里管理' : '版本记录将在这里展示'"
          />
        </section>
      </main>

      <ElResult
        v-else-if="!loading"
        class="project-load-result"
        icon="error"
        title="项目加载失败"
        :sub-title="loadError || '请检查登录状态和项目权限'"
      />

      <footer v-if="project && !loadError && initializationMode" class="init-footer"
        ><span>当前：{{ currentInitStepLabel }} · 已保存的配置可下次继续</span
        ><div
          ><ElButton :disabled="activeTab === 'services'" @click="previousInitStep">上一步</ElButton
          ><ElButton
            v-if="activeTab !== 'workflows'"
            type="success"
            :loading="initializationSaving || workflowGenerating"
            @click="nextInitStep"
            >下一步</ElButton
          ><ElButton
            v-else
            type="success"
            :loading="runningAll"
            :disabled="!workflows.length || unboundDeploymentWorkflows.length > 0"
            @click="completeInitialization"
            >完成配置</ElButton
          ></div
        ></footer
      >

      <ElDialog
        v-model="environmentCreatorVisible"
        title="新建环境"
        :width="projectDeliveryType === 'helm' ? 'min(1100px, 95vw)' : '560px'"
        :close-on-click-modal="!environmentSaving"
        :show-close="!environmentSaving"
      >
        <p class="form-help">配置运行目标，Helm 服务可在此填写环境 values，一并校验保存。</p>
        <ElSelect
          v-model="environmentPresetCode"
          :disabled="environmentSaving"
          placeholder="不选择时使用测试环境"
          clearable
          style="width: 100%; margin-bottom: 16px"
        >
          <ElOption
            v-for="preset in environmentPresets"
            :key="preset.code"
            :label="preset.name"
            :value="preset.code"
            :disabled="environments.some((e) => e.code === preset.code)"
          />
          <ElOption label="自定义环境" value="custom" />
        </ElSelect>
        <ElForm label-position="top" :disabled="environmentSaving"
          ><div class="form-grid"
            ><ElFormItem label="环境名称" required
              ><ElInput v-model="environmentForm.name" placeholder="例如：预发布环境" /></ElFormItem
            ><ElFormItem label="环境编码" required
              ><ElInput
                v-model="environmentForm.code"
                placeholder="例如：staging"
                @input="syncNamespace" /></ElFormItem></div
          ><ElFormItem v-if="projectDeliveryType !== 'host'" label="K8s 集群" required
            ><ElSelect
              v-model="environmentForm.cluster_id"
              filterable
              style="width: 100%"
              placeholder="选择集群"
              ><ElOption
                v-for="cluster in clusters"
                :key="cluster.id"
                :label="cluster.name"
                :value="cluster.id" /></ElSelect></ElFormItem
          ><ElFormItem v-if="projectDeliveryType !== 'host'" label="K8s Namespace" required
            ><ElInput v-model="environmentForm.namespace" /></ElFormItem
          ><ElFormItem v-else label="主机组" required
            ><ElInput
              v-model="environmentForm.host_group_ref"
              placeholder="输入主机组标识" /></ElFormItem
          ><div class="switch-row"
            ><span
              ><strong>生产环境</strong
              ><small>{{
                productionEnvironmentCode
                  ? 'prod / production 为生产专用编码，自动启用生产发布保护'
                  : '生产发布使用已验收镜像；自定义编码也可开启'
              }}</small></span
            ><ElSwitch
              v-model="environmentForm.is_production"
              :disabled="productionEnvironmentCode" /></div
          ><div class="switch-row"
            ><span><strong>部署审批</strong><small>工作流部署前等待人工审批</small></span
            ><ElSwitch v-model="environmentForm.deploy_approval" /></div
        ></ElForm>
        <HelmEnvironmentServices
          v-if="projectDeliveryType === 'helm'"
          v-model="environmentHelmServices"
          :services="services"
          :environments="environments"
          :disabled="environmentSaving"
          @busy-change="environmentServicesBusy = $event"
        />
        <template #footer
          ><ElButton :disabled="environmentSaving" @click="environmentCreatorVisible = false"
            >取消</ElButton
          ><ElButton
            type="success"
            :loading="environmentSaving"
            :disabled="environmentServicesBusy"
            @click="createEnvironment"
            >{{ environmentHelmServices.length ? '预览并创建环境' : '创建环境' }}</ElButton
          ></template
        >
      </ElDialog>
      <HelmEnvironmentReview ref="environmentReview" :project-id="projectId" :services="services" />

      <ServiceLifecycleDialog
        v-model:visible="lifecycleVisible"
        :mode="lifecycleMode"
        :project-id="projectId"
        :service="lifecycleService"
        @changed="loadWorkspace"
      />
      <ServiceTemplatePicker
        v-if="
          projectDeliveryType !== 'host' &&
          (canImport || (templateTarget && canManageService(templateTarget, 'update')))
        "
        v-model:visible="templatePickerVisible"
        :project-id="projectId"
        :kind="projectDeliveryType === 'helm' ? 'helm' : 'k8s_yaml'"
        :environments="environments"
        :preferred-environment-id="selectedEnvironmentId || runtimeEnvironmentId"
        :existing-codes="services.map((item) => item.code)"
        :service="templateTarget"
        :replacement-candidates="services.filter((item) => canManageService(item, 'update'))"
        @saved="handleTemplateServiceSaved"
      />
      <WorkflowCreationDialog
        v-if="project"
        v-model:visible="workflowCreationVisible"
        :project="project"
        :services="services"
        :environments="environments"
        @created="workflowCreated"
      />
      <ServiceCreationWizard
        v-model:visible="serviceWizardVisible"
        :project-id="projectId"
        :delivery-type="projectDeliveryType"
        :service="null"
        :code-sources="codeSources"
        :environments="environments"
        :can-create-workflow="false"
        :can-update-build="canUpdateBuild"
        :initial-source="wizardSource"
        @saved="handleServiceSaved"
      />
      <BulkServiceImportDialog
        v-model:visible="bulkImportVisible"
        :project-id="projectId"
        :code-sources="codeSources"
        :existing-codes="services.map((item) => item.code)"
        @imported="handleServiceSaved"
      />
      <BuildConfigDialog
        v-model:visible="buildConfigVisible"
        :service="buildService"
        :editable="canUpdateBuild"
        @saved="handleBuildSaved"
      />
      <EnvironmentConfigDialog
        v-model:visible="environmentConfigVisible"
        :initial-tab="environmentConfigTab"
        :environment="selectedEnvironment"
        :services="services"
        :editable="canEdit"
        @changed="loadWorkspace"
      />
      <WorkflowRunDrawer
        v-model:visible="runDrawerVisible"
        :run-id="selectedRunId"
        :environments="environments"
        :clusters="clusters"
        :services="services"
        :code-sources="codeSources"
        @changed="loadWorkspace"
      />
      <ElDialog
        v-model="executeDialogVisible"
        title="执行工作流"
        width="min(960px, 95vw)"
        top="6vh"
        class="workflow-execution-dialog"
        destroy-on-close
        :close-on-click-modal="false"
        :close-on-press-escape="!executingWorkflow"
        :show-close="!executingWorkflow"
      >
        <div v-loading="executeLoading" class="execute-dialog">
          <ElAlert v-if="executeLoadError" type="error" :title="executeLoadError" :closable="false">
            <ElButton text @click="reloadExecuteWorkflow">重新加载工作流</ElButton>
          </ElAlert>
          <ElAlert
            v-if="executeSubmitMessage"
            type="warning"
            :title="executeSubmitMessage"
            :closable="false"
          />
          <ElButton
            v-if="executeUncertain && executeWorkflowTarget"
            link
            type="primary"
            @click="openWorkflowJob(executeWorkflowTarget)"
            >查看运行记录，确认提交结果</ElButton
          >
          <div v-for="item in executeSubmitted" :key="item.id">
            <ElButton
              link
              type="primary"
              @click="openWorkflowRun(item.id, executeWorkflowTarget || undefined)"
            >
              {{ item.name }} · 已提交（记录 ID：{{ item.id }}）
            </ElButton>
          </div>
          <div class="execute-dialog-heading">
            <div
              ><small>目标环境</small
              ><strong>{{
                executeEnvironment?.name ||
                (executeNeedsEnvironment ? '未绑定目标环境' : '不部署到环境')
              }}</strong
              ><span>{{ executeWorkflowTarget?.name || '环境 Job' }}</span></div
            >
            <ElTag
              size="small"
              :type="executeEnvironment?.is_production ? 'danger' : 'info'"
              effect="plain"
              >{{
                executeEnvironment?.is_production
                  ? '生产环境'
                  : executeNeedsEnvironment
                    ? '非生产环境'
                    : '构建 / 任务'
              }}</ElTag
            >
          </div>
          <div v-if="executeEnvironment" class="execute-form-row">
            <label>部署目标</label
            ><span class="execute-target">{{ executeTargetDescription }}</span>
          </div>
          <div v-if="executeScope === 'selected' && executeCanOverride" class="execute-form-row">
            <label>发布服务</label>
            <ElSelect
              :model-value="executeMultiService ? executeServiceIds : executeServiceIds[0]"
              :multiple="executeMultiService"
              :disabled="
                executeLoading ||
                !!executeLoadError ||
                executingWorkflow ||
                !!executeSubmitted.length
              "
              placeholder="选择服务"
              class="execute-service-picker"
              @change="changeExecuteServices"
            >
              <ElOption
                v-for="service in executeSelectableServices"
                :key="service.id"
                :value="String(service.id)"
                :label="service.name"
              />
            </ElSelect>
          </div>
          <div v-if="executeCanPromote || executeProductionBuild" class="execute-form-row">
            <label>发布方式</label>
            <ElRadioGroup
              v-model="executeMode"
              :disabled="
                executeLoading || executingWorkflow || !!executeSubmitted.length || executeUncertain
              "
              @change="changeExecuteMode"
            >
              <ElRadioButton value="build">构建并部署</ElRadioButton>
              <ElRadioButton value="existing_image">使用已有镜像</ElRadioButton>
            </ElRadioGroup>
          </div>
          <p
            v-if="executeMode === 'build' && executeNeedsEnvironment"
            class="execute-selected-hint"
          >
            按所选分支执行构建、工作流中配置的测试和部署；部署目标固定为当前环境。
          </p>
          <ElAlert
            v-if="executeMode === 'existing_image'"
            title="固定所选镜像摘要，使用目标环境自己的 YAML、变量和运行资源"
            description="不会复制测试环境配置，也不会重新执行构建。代码、Dockerfile 或编译参数改变时，请先重新构建。"
            type="info"
            :closable="false"
          />
          <div class="execute-tools">
            <ElButton
              text
              size="small"
              :disabled="executingWorkflow || executeUncertain"
              :aria-expanded="executeAdvanced"
              @click="executeAdvanced = !executeAdvanced"
              >高级</ElButton
            >
          </div>
          <div v-if="executeAdvanced" class="execute-form-row execute-advanced">
            <label>执行范围</label>
            <ElRadioGroup
              v-model="executeScope"
              :disabled="
                executeLoading ||
                !!executeLoadError ||
                executingWorkflow ||
                !!executeSubmitted.length
              "
            >
              <ElRadio value="selected" :disabled="!executeCanOverride">指定服务</ElRadio>
              <ElRadio value="all" :disabled="executeMode === 'existing_image'"
                >按工作流配置执行</ElRadio
              >
            </ElRadioGroup>
          </div>
          <div v-if="executeScope === 'all'" class="execute-selected-hint">
            执行工作流中保存的全部节点、服务和分支。不会自动加入项目中的其他服务。
            <span v-if="!executeCanOverride">此编排不支持替换为单个服务。</span>
            <span v-if="executeOrderedServiceNames.length"
              >部署提交顺序：{{ executeOrderedServiceNames.join(' → ') }}</span
            >
          </div>
          <section v-if="executeScope === 'selected' && executeCanOverride" class="execute-section">
            <div class="execute-section-title"
              ><strong>{{ executeMode === 'existing_image' ? '服务与镜像' : '服务与分支' }}</strong
              ><span v-if="executeServiceIds.length > 1">用 ↑ ↓ 调整部署提交顺序</span></div
            >
            <div class="execute-service-list">
              <div
                v-for="service in executeSelectableServices.filter((item) =>
                  executeServiceIds.includes(String(item.id))
                )"
                :key="service.id"
                class="execute-service-row"
              >
                <strong class="execute-service-name">{{ service.name }}</strong>
                <span
                  v-if="executeServiceIds.length > 1"
                  class="execute-order-actions"
                  aria-label="调整服务提交顺序"
                >
                  <small v-if="executeServiceIds.includes(String(service.id))">{{
                    executeServiceIds.indexOf(String(service.id)) + 1
                  }}</small>
                  <ElButton
                    text
                    circle
                    size="small"
                    :icon="ArrowUp"
                    title="上移"
                    :disabled="
                      executeLoading ||
                      executingWorkflow ||
                      !executeServiceIds.includes(String(service.id)) ||
                      executeServiceIds.indexOf(String(service.id)) <= 0
                    "
                    @click="moveExecuteService(service.id, -1)"
                  />
                  <ElButton
                    text
                    circle
                    size="small"
                    :icon="ArrowDown"
                    title="下移"
                    :disabled="
                      executeLoading ||
                      executingWorkflow ||
                      !executeServiceIds.includes(String(service.id)) ||
                      executeServiceIds.indexOf(String(service.id)) < 0 ||
                      executeServiceIds.indexOf(String(service.id)) >= executeServiceIds.length - 1
                    "
                    @click="moveExecuteService(service.id, 1)"
                  />
                </span>
                <ElSelect
                  v-if="executeMode === 'build'"
                  v-model="executeBranchValues[service.id]"
                  class="execute-branch-select"
                  filterable
                  allow-create
                  default-first-option
                  :disabled="
                    executeLoading ||
                    !!executeLoadError ||
                    !executeServiceIds.includes(String(service.id)) ||
                    executingWorkflow ||
                    executeSubmitted.some((item) => item.serviceId === service.id)
                  "
                  :loading="executeBranchLoading[service.id]"
                  placeholder="选择分支"
                  @focus="loadExecuteServiceBranches(service)"
                >
                  <ElOption
                    v-for="branch in executeServiceBranches[service.id] || []"
                    :key="branch.name"
                    :label="branch.name"
                    :value="branch.name"
                  />
                </ElSelect>
                <ElSelect
                  v-else
                  v-model="executeSourceBuildIds[service.id]"
                  class="execute-branch-select"
                  filterable
                  :disabled="!executeServiceIds.includes(String(service.id)) || executingWorkflow"
                  :loading="promotionLoading[service.id]"
                  placeholder="选择已成功构建的镜像"
                  @visible-change="(open) => open && loadPromotionBuilds(service.id)"
                >
                  <ElOption
                    v-for="build in promotionBuilds[service.id] || []"
                    :key="build.id"
                    :value="build.id"
                    :label="`#${build.build_number} · ${build.git_branch} · ${(build.git_commit || '').slice(0, 8)} · ${formatRunDate(build.created_at)}`"
                  />
                </ElSelect>
                <small
                  v-if="executeMode === 'existing_image' && selectedPromotionBuild(service.id)"
                  class="execute-image-reference"
                  >{{ selectedPromotionBuild(service.id)?.image_url }}@{{
                    selectedPromotionBuild(service.id)?.image_digest
                  }}</small
                >
                <ElButton
                  v-if="executeMode === 'existing_image' && promotionHasMore[service.id]"
                  size="small"
                  text
                  :loading="promotionLoading[service.id]"
                  @click="loadPromotionBuilds(service.id, true)"
                  >更早构建</ElButton
                >
                <small
                  v-if="promotionErrors[service.id] && executeMode === 'existing_image'"
                  class="execute-branch-error"
                  >{{ promotionErrors[service.id] }}</small
                >
                <small
                  v-if="executeMode === 'build' && executeBranchErrors[service.id]"
                  class="execute-branch-error"
                  >{{ executeBranchErrors[service.id] }}</small
                >
              </div>
              <ElEmpty v-if="!services.length" :image-size="48" description="暂无可执行服务" />
              <p v-else-if="!executeServiceIds.length" class="execute-selected-hint"
                >请选择要执行的服务</p
              >
            </div>
          </section>
          <section class="execute-section">
            <template
              v-if="executeNeedsEnvironment && !executeWorkflowTarget?.target_environment_id"
            >
              <ElAlert
                type="warning"
                title="此工作流尚未绑定部署环境"
                description="目标环境应在项目初始化或工作流编排中绑定。执行阶段只核对已保存的目标，不临时设置环境。"
                :closable="false"
              />
              <ElButton link type="primary" @click="openWorkflowBinding"
                >去工作流编排绑定环境</ElButton
              >
            </template>
            <div
              v-if="executeAdvanced && executeWorkflowTarget?.target_environment_id"
              class="execute-selected-hint"
            >
              环境由工作流固定绑定，执行时不能切换。部署到其他环境，请新建绑定对应环境的工作流。
            </div>
            <ElAlert
              v-if="executeEnvironment && !isExecutableEnvironment(executeEnvironment)"
              :type="executeEnvironment.is_production ? 'warning' : 'info'"
              :title="`${executeEnvironment.is_production ? '生产环境' : '目标环境'}：${executeEnvironment.name}`"
              :description="executeTargetDescription"
              :closable="false"
            />
            <div
              v-if="executeScope === 'selected' && executeServiceIds.length"
              class="execute-selected-hint"
              >已选择 {{ executeServiceIds.length }} 个服务{{
                executeServiceIds.length > 1
                  ? '：同一次运行，全部构建成功后按列表顺序提交部署，各自检查状态。'
                  : '，单独执行。'
              }}</div
            >
            <div v-if="executeAdvanced" class="execute-form-row"
              ><label>发布方式</label
              ><span>按部署定义执行（当前不支持额外的灰度 / 蓝绿编排）</span></div
            >
          </section>
        </div>
        <template #footer
          ><ElButton :disabled="executingWorkflow" @click="executeDialogVisible = false"
            >关闭</ElButton
          ><ElButton
            type="primary"
            :loading="executingWorkflow"
            :disabled="
              executeLoading ||
              executeUncertain ||
              !!executeLoadError ||
              (executeScope === 'selected' && !executeServiceIds.length) ||
              (executeNeedsEnvironment &&
                (!executeWorkflowTarget?.target_environment_id ||
                  !executeEnvironment ||
                  !isExecutableEnvironment(executeEnvironment))) ||
              (executeScope === 'all' && !!executeSubmitted.length)
            "
            @click="confirmWorkflowExecute"
            >{{
              executeNeedsEnvironment && !executeWorkflowTarget?.target_environment_id
                ? '请先绑定目标环境'
                : executeSubmitted.length
                  ? '提交剩余服务'
                  : '执行工作流'
            }}</ElButton
          ></template
        >
      </ElDialog>
      <ElDrawer
        v-model="workflowJobDrawerVisible"
        :title="`${selectedWorkflow?.name || '工作流'} Job`"
        size="760px"
        destroy-on-close
      >
        <div v-if="selectedWorkflow" class="job-detail-panel">
          <div class="job-detail-header">
            <div
              ><span class="view-eyebrow">ENVIRONMENT JOB</span><h2>{{ selectedWorkflow.name }}</h2
              ><p
                >{{ workflowEnvironmentName(selectedWorkflow) }} · {{ selectedWorkflow.code }}</p
              ></div
            >
            <ElTag :type="workflowStatusType(selectedWorkflow.id)" effect="plain">{{
              workflowStatusLabel(selectedWorkflow.id)
            }}</ElTag>
          </div>
          <div class="job-detail-metrics">
            <div
              ><span>成功率</span
              ><strong>{{ workflowSuccessRate(selectedWorkflow.id) }}</strong></div
            >
            <div
              ><span>最近执行</span
              ><strong>{{ workflowLastRunTime(selectedWorkflow.id) }}</strong></div
            >
            <div
              ><span>执行人</span
              ><strong>{{ workflowLastRunner(selectedWorkflow.id) }}</strong></div
            >
            <div
              ><span>执行次数</span
              ><strong>{{ workflowRunList(selectedWorkflow.id).length }}</strong></div
            >
          </div>
          <div class="job-detail-toolbar"
            ><strong>执行记录</strong
            ><ElButton size="small" :loading="workflowRunsLoading" @click="refreshWorkflowJob"
              >刷新</ElButton
            ></div
          >
          <div v-if="workflowRunList(selectedWorkflow.id).length" class="workflow-run-table">
            <div class="workflow-run-row workflow-run-header"
              ><span>ID</span><span>状态</span><span>耗时</span><span>提交时间</span
              ><span>执行人</span></div
            >
            <button
              v-for="run in workflowRunList(selectedWorkflow.id)"
              :key="run.id"
              type="button"
              class="workflow-run-row"
              @click="openWorkflowRun(run.id, selectedWorkflow)"
            >
              <span :title="`记录 ID：${run.id}`">#{{ run.run_number || '—' }}</span
              ><span
                ><ElTag size="small" :type="runStatusType(run.status)" effect="plain">{{
                  runStatusLabel(run.status)
                }}</ElTag></span
              ><span>{{ formatRunDuration(run) }}</span
              ><span>{{ formatRunDate(run.created_at) }}</span
              ><span>{{ workflowActorLabel(run) }}</span>
            </button>
          </div>
          <ElEmpty v-else description="暂无执行记录" />
        </div>
      </ElDrawer>
      <ProductionReleaseDialog
        v-if="productionWorkflow"
        v-model:visible="productionVisible"
        :workflow-id="productionWorkflow.id"
        :project-id="projectId"
        :initial-service-ids="productionInitialServiceIds"
        @submitted="productionSubmitted"
        @build="openProductionBuild"
        @configure="openProductionConfiguration"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
  import ArtDeliveryIcon from '@/components/core/base/art-delivery-icon/index.vue'
  import { SERVICE_NAME_HINT, serviceNameError, serviceCodeFromName } from '@/utils/service-name'
  import { workflowActorLabel } from '@/utils/workflow-actor'
  import ProductionReleaseDialog from './components/ProductionReleaseDialog.vue'
  import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
  import {
    ArrowDown,
    ArrowUp,
    ArchiveRestore,
    Boxes,
    Check,
    ChevronDown,
    ChevronRight,
    FileCode2,
    MoreHorizontal,
    GitBranch,
    GitCommit,
    Hammer,
    LayoutDashboard,
    PackageOpen,
    Play,
    Plus,
    Search,
    Server,
    TestTube2,
    Workflow
  } from 'lucide-vue-next'
  import {
    createEnvironmentApi,
    getEnvironmentApi,
    createServiceApi,
    createWorkflowApi,
    getBuildConfigApi,
    getWorkflowApi,
    getProjectApi,
    listAllServicesApi,
    listBranchesApi,
    listBuildRunsApi,
    listClustersApi,
    listCodeSourcesApi,
    listEnvironmentsApi,
    listWorkflowsApi,
    listWorkflowRunsApi,
    runWorkflowApi,
    updateProjectApi,
    type AxiOpsBuildRun,
    type AxiOpsBranch,
    type AxiOpsCluster,
    type AxiOpsCodeSource,
    type AxiOpsEnvironment,
    type AxiOpsProject,
    type AxiOpsService,
    type AxiOpsWorkflow,
    type AxiOpsWorkflowRun,
    type WorkflowNode
  } from '@/api/axiops'
  import {
    buildConfigurationIssues,
    isBuildConfigMissing,
    errorMessage,
    supportsServiceOverride,
    workflowServiceSelection
  } from '@/utils/cicd-workspace'
  import {
    hasWorkflowDeploymentSequence,
    workflowDeploymentOrder
  } from '@/utils/workflow-service-order'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'
  import ProjectRunTrend from './components/ProjectRunTrend.vue'
  import BuildConfigDialog from './components/BuildConfigDialog.vue'
  import BulkServiceImportDialog from './components/BulkServiceImportDialog.vue'
  import EnvironmentConfigDialog from './components/EnvironmentConfigDialog.vue'
  import ServiceCreationWizard from './components/ServiceCreationWizard.vue'
  import ServiceTemplatePicker from './components/ServiceTemplatePicker.vue'
  import ServiceLifecycleDialog from './components/ServiceLifecycleDialog.vue'
  import ServiceConfigurationPanel from './components/ServiceConfigurationPanel.vue'
  import HelmEnvironmentServices from './components/HelmEnvironmentServices.vue'
  import HelmEnvironmentReview from './components/HelmEnvironmentReview.vue'
  import type { InitialHelmService } from '@/api/axiops'
  import { mapLimited } from '@/utils/service-environment-config'
  import { getServiceEnvironmentConfigApi } from '@/api/axiops'
  import WorkflowRunDrawer from './components/WorkflowRunDrawer.vue'
  import ProjectNavBar from './components/ProjectNavBar.vue'
  import ArchivedProjectHistory from './components/ArchivedProjectHistory.vue'
  import WorkflowCreationDialog from './components/WorkflowCreationDialog.vue'
  import { isDeploymentNode } from '@/utils/workflow-template'
  import EnvironmentRuntimePanel from './components/EnvironmentRuntimePanel.vue'

  defineOptions({ name: 'AxiOpsCICDProjectWorkspace' })
  const route = useRoute(),
    router = useRouter(),
    access = useAxiOpsAccess()
  let projectId = Number(route.params.projectId)
  const loading = ref(false),
    loadError = ref(''),
    project = ref<AxiOpsProject | null>(null)
  const services = ref<AxiOpsService[]>([]),
    environments = ref<AxiOpsEnvironment[]>([]),
    clusters = ref<AxiOpsCluster[]>([]),
    workflows = ref<AxiOpsWorkflow[]>([]),
    runs = ref<AxiOpsBuildRun[]>([]),
    codeSources = ref<AxiOpsCodeSource[]>([])
  const workflowRuns = reactive<Record<number, AxiOpsWorkflowRun[]>>({})
  const selectedWorkflow = ref<AxiOpsWorkflow | null>(null)
  const workflowCreationVisible = ref(false)
  async function workflowCreated(id: number) {
    workflows.value = await listWorkflowsApi(projectId, { silent: true })
    await router.push(`/cicd/projects/${projectId}/workflows/${id}`)
  }
  const workflowJobDrawerVisible = ref(false)
  const workflowRunsLoading = ref(false)
  const configuredBuildServiceIds = ref<Set<number>>(new Set())
  const buildIssues = ref<Record<number, string[]>>({})
  const workspaceWarnings = ref<string[]>([])
  let workspaceRequest = 0,
    executeRequest = 0
  let workspaceLoaded = false
  const initializationSaving = ref(false)
  const workspaceTabs = new Set([
    'overview',
    'services',
    'environments',
    'workflows',
    'tests',
    'versions',
    'build'
  ])
  const requestedTab = String(route.query.tab || ''),
    initializationMode = ref(route.query.init === '1')
  // 已完成初始化的项目直接落在「工作流」：这是唯一能执行构建与查看运行记录的页面，
  // 概览只有统计数字，作为默认落地页会多一次点击。未初始化时仍由 initSteps 接管。
  const activeTab = ref(
    workspaceTabs.has(requestedTab)
      ? requestedTab
      : initializationMode.value
        ? 'services'
        : 'workflows'
  )
  const selectedService = ref<AxiOpsService | null>(null),
    buildService = ref<AxiOpsService | null>(null),
    selectedEnvironment = ref<AxiOpsEnvironment | null>(null),
    selectedEnvironmentId = ref(0)
  const serviceContextMenu = ref<{ x: number; y: number; service: AxiOpsService } | null>(null)
  const serviceConfiguration = ref<InstanceType<typeof ServiceConfigurationPanel>>()
  const runtimeEnvironmentId = ref(0)
  const runtimeEnvironment = computed(() =>
    environments.value.find((item) => item.id === runtimeEnvironmentId.value)
  )
  const environmentConfigTab = ref<'variables' | 'resources'>('variables')
  function selectRuntimeEnvironment(environment: AxiOpsEnvironment) {
    runtimeEnvironmentId.value = environment.id
    try {
      sessionStorage.setItem('axiops-runtime-environment-' + projectId, String(environment.id))
    } catch {
      /* Storage can be unavailable. */
    }
  }

  const expandedServices = ref<Set<number>>(new Set()),
    helmFilePaths = reactive<Record<number, string[]>>({}),
    serviceActiveFiles = reactive<Record<number, string>>({}),
    keyword = ref('')
  const serviceWizardVisible = ref(false),
    templatePickerVisible = ref(false),
    bulkImportVisible = ref(false),
    buildConfigVisible = ref(false),
    environmentConfigVisible = ref(false),
    environmentCreatorVisible = ref(false),
    runDrawerVisible = ref(false)
  const lifecycleVisible = ref(false),
    lifecycleMode = ref<'delete' | 'rename' | 'trash'>('trash'),
    lifecycleService = ref<AxiOpsService | null>(null)
  const canManageService = (service: AxiOpsService, action: string) =>
    access.can('service', action, { type: 'service', id: service.id, project_id: projectId })
  const openServiceLifecycle = (mode: 'delete' | 'rename' | 'trash', service?: AxiOpsService) => {
    lifecycleService.value = service || null
    lifecycleMode.value = mode
    lifecycleVisible.value = true
  }
  const closeServiceContextMenu = () => {
    serviceContextMenu.value = null
  }
  const openServiceContextMenu = (event: MouseEvent, service: AxiOpsService) => {
    if (!canManageService(service, 'update') && !canImport.value) return
    selectService(service)
    const width = 158
    const height = service.delivery_type === 'k8s_yaml' ? 176 : 140
    serviceContextMenu.value = {
      service,
      x: Math.min(event.clientX, window.innerWidth - width - 12),
      y: Math.min(event.clientY, window.innerHeight - height - 12)
    }
  }
  const openServiceYamlFromContext = async () => {
    const service = serviceContextMenu.value?.service
    closeServiceContextMenu()
    if (!service) return
    activeTab.value = 'services'
    await nextTick()
    await serviceConfiguration.value?.openNewYaml()
  }
  const openServiceTemplateFromContext = () => {
    const service = serviceContextMenu.value?.service
    closeServiceContextMenu()
    if (!service) return
    void openTemplateReplacement(service)
  }
  const openServiceRepositoryFromContext = async () => {
    const service = serviceContextMenu.value?.service
    closeServiceContextMenu()
    if (!service) return
    if (serviceConfiguration.value && !(await serviceConfiguration.value.confirmLeave())) return
    selectService(service)
    if (service.delivery_type === 'helm') {
      activeTab.value = 'services'
      await nextTick()
      await serviceConfiguration.value?.openRepositorySync()
      return
    }
    openImport()
  }
  const selectedRunId = ref<number | null>(null),
    wizardSource = ref<'manual' | 'template' | 'git'>('template'),
    manualDirectoryOpen = ref(false),
    manualDirectoryName = ref('')
  const executeDialogVisible = ref(false),
    executeWorkflowTarget = ref<AxiOpsWorkflow | null>(null),
    executeEnvironmentId = ref(0),
    executeScope = ref<'all' | 'selected'>('selected'),
    executeServiceIds = ref<string[]>([]),
    executeServiceBranches = ref<Record<number, AxiOpsBranch[]>>({}),
    executeBranchValues = ref<Record<number, string>>({}),
    executeBranchLoading = ref<Record<number, boolean>>({}),
    executingWorkflow = ref(false)
  const executeLoading = ref(false),
    executeLoadError = ref(''),
    executeSubmitMessage = ref('')
  const executeSubmitted = ref<Array<{ id: number; serviceId: number; name: string }>>([])
  const executeUncertain = ref(false)
  const executeAdvanced = ref(false)
  const executeBranchErrors = ref<Record<number, string>>({})
  const changeExecuteServices = (value: string | string[]) => {
    if (
      executingWorkflow.value ||
      executeLoading.value ||
      executeSubmitted.value.length ||
      executeUncertain.value
    )
      return
    const ids = Array.isArray(value) ? value : value ? [value] : []
    const allowed = new Set(executeSelectableServices.value.map((service) => String(service.id)))
    const selected = [...new Set(ids)].filter((id) => allowed.has(id))
    if (!executeMultiService.value && selected.length)
      selectExecuteService(Number(selected[0]), true)
    else executeServiceIds.value = selected
  }
  const selectExecuteService = (id: number, selected: boolean) => {
    if (executingWorkflow.value || executeLoading.value || executeSubmitted.value.length) return
    if (selected && !executeMultiService.value) {
      executeServiceIds.value = [String(id)]
      return
    }
    executeServiceIds.value = executeServiceIds.value.filter((item) => item !== String(id))
    if (selected) executeServiceIds.value.push(String(id))
  }
  const moveExecuteService = (id: number, direction: -1 | 1) => {
    if (executingWorkflow.value || executeLoading.value || executeSubmitted.value.length) return
    const key = String(id)
    const index = executeServiceIds.value.indexOf(key)
    const target = index + direction
    if (index < 0 || target < 0 || target >= executeServiceIds.value.length) return
    const next = [...executeServiceIds.value]
    ;[next[index], next[target]] = [next[target], next[index]]
    executeServiceIds.value = next
  }
  const executeNeedsEnvironment = computed(
    () => !!executeWorkflowTarget.value?.nodes?.some((n) => isDeploymentNode(n.type))
  )
  const executeMode = ref<'build' | 'existing_image'>('build')
  const productionVisible = ref(false),
    productionWorkflow = ref<AxiOpsWorkflow | null>(null),
    productionInitialServiceIds = ref<number[]>([])
  const changeExecuteMode = () => {
    if (!executeProductionBuild.value || executeMode.value !== 'existing_image') return
    if (
      executingWorkflow.value ||
      executeLoading.value ||
      executeSubmitted.value.length ||
      executeUncertain.value ||
      !executeWorkflowTarget.value
    ) {
      executeMode.value = 'build'
      return
    }
    productionWorkflow.value = executeWorkflowTarget.value
    productionInitialServiceIds.value = executeServiceIds.value.map(Number)
    executeDialogVisible.value = false
    productionVisible.value = true
  }
  const productionSubmitted = (id: number) => {
    if (productionWorkflow.value) openWorkflowRun(id, productionWorkflow.value)
  }
  const executeProductionBuild = ref(false)
  const openProductionConfiguration = async (sid: number, eid: number) => {
    const service = services.value.find((s) => s.id === sid)
    if (!service || productionWorkflow.value?.target_environment_id !== eid) return
    productionVisible.value = false
    await selectTab('services')
    if (activeTab.value !== 'services') return
    selectService(service)
    await nextTick()
    await serviceConfiguration.value?.switchEnvironment(eid)
  }
  const executeProductionBuildTarget = ref(0)
  const openProductionBuild = async (ids: number[]) => {
    const workflow = productionWorkflow.value
    if (!workflow) return
    productionVisible.value = false
    await openWorkflowExecute(workflow, true)
    if (executeWorkflowTarget.value?.id === workflow.id && !executeLoadError.value) {
      if (!ids.every((id) => executeSelectableServices.value.some((s) => s.id === id))) {
        executeLoadError.value = '所选服务已变化，请重新选择服务'
        return
      }
      executeServiceIds.value = ids.map(String)
      executeScope.value = 'selected'
    }
  }
  const executeSourceBuildIds = ref<Record<number, number>>({})
  const promotionBuilds = reactive<Record<number, AxiOpsBuildRun[]>>({})
  const promotionLoading = reactive<Record<number, boolean>>({})
  const promotionErrors = reactive<Record<number, string>>({})
  const promotionPages = reactive<Record<number, number>>({})
  const promotionHasMore = reactive<Record<number, boolean>>({})
  const executeCanPromote = computed(
    () =>
      executeCanOverride.value &&
      !executeEnvironment.value?.is_production &&
      ['k8s_yaml', 'helm'].includes(projectDeliveryType.value) &&
      !!executeWorkflowTarget.value?.nodes?.some((node) =>
        ['yaml_deploy', 'helm_deploy'].includes(node.type)
      ) &&
      !executeWorkflowTarget.value?.nodes?.some((node) =>
        ['test', 'host_deploy'].includes(node.type)
      )
  )
  const selectedPromotionBuild = (sid: number) =>
    promotionBuilds[sid]?.find((build) => build.id === executeSourceBuildIds.value[sid])
  const loadPromotionBuilds = async (sid: number, more = false) => {
    if (promotionLoading[sid]) return
    const request = executeRequest
    promotionLoading[sid] = true
    promotionErrors[sid] = ''
    const page = more ? (promotionPages[sid] || 1) + 1 : 1
    try {
      const result = await listBuildRunsApi(projectId, {
        silent: true,
        service_id: sid,
        status: 'success',
        page,
        page_size: 20
      })
      if (request !== executeRequest) return
      const items = (result.list || []).filter(
        (build) =>
          build.service_id === sid &&
          build.status === 'success' &&
          build.image_url &&
          /^sha256:[a-f0-9]{64}$/.test(build.image_digest || '')
      )
      promotionBuilds[sid] = more ? [...(promotionBuilds[sid] || []), ...items] : items
      promotionPages[sid] = page
      promotionHasMore[sid] = page * 20 < result.total
      if (!promotionBuilds[sid].length)
        promotionErrors[sid] = '暂无可复用镜像，请先成功构建；不会自动改为重新构建。'
    } catch (error) {
      if (request === executeRequest)
        promotionErrors[sid] = errorMessage(error, '镜像列表读取失败，请重新展开重试')
    } finally {
      promotionLoading[sid] = false
    }
  }
  watch(executeMode, (mode) => {
    if (mode === 'existing_image') executeScope.value = 'selected'
  })
  const executeCanOverride = computed(() =>
    supportsServiceOverride(executeWorkflowTarget.value?.nodes || [])
  )
  const executeWorkflowServiceIds = computed(() =>
    workflowServiceSelection(executeWorkflowTarget.value?.nodes || [])
  )
  const executeMultiService = computed(() => executeWorkflowServiceIds.value.length > 1)
  const executeSelectableServices = computed(() => {
    const source = executeMultiService.value
      ? executeWorkflowServiceIds.value
          .map((id) => services.value.find((service) => service.id === id))
          .filter((service): service is AxiOpsService => !!service)
      : services.value
    const order = new Map(executeServiceIds.value.map((id, index) => [Number(id), index]))
    return [...source].sort(
      (left, right) =>
        (order.get(left.id) ?? Number.MAX_SAFE_INTEGER) -
        (order.get(right.id) ?? Number.MAX_SAFE_INTEGER)
    )
  })
  const executeOrderedServiceNames = computed(() => {
    const nodes = executeWorkflowTarget.value?.nodes || []
    return hasWorkflowDeploymentSequence(nodes)
      ? workflowDeploymentOrder(nodes).map(
          (id) => services.value.find((service) => service.id === id)?.name || `服务 #${id}`
        )
      : []
  })
  const executeEnvironment = computed(() =>
    environments.value.find((item) => item.id === executeEnvironmentId.value)
  )
  const isExecutableEnvironment = (environment: AxiOpsEnvironment) =>
    environment.status === 'active' &&
    environment.binding_status === 'validated' &&
    (projectDeliveryType.value === 'host'
      ? environment.environment_type === 'host_group'
      : environment.environment_type === 'k8s_namespace')
  const executeTargetDescription = computed(() => {
    const environment = executeEnvironment.value
    if (!environment) return ''
    return environment.environment_type === 'host_group'
      ? `主机组：${environment.host_group_ref}`
      : `集群：${clusterName(environment.cluster_id)}；命名空间：${environment.namespace}${environment.is_production || environment.deploy_approval ? '；工作流必须包含审批节点' : ''}`
  })
  const manualDirectorySaving = ref(false),
    environmentSaving = ref(false),
    workflowGenerating = ref(false),
    runningAll = ref(false)
  const environmentForm = reactive({
    name: '',
    code: '',
    cluster_id: 0,
    namespace: '',
    host_group_ref: '',
    is_production: false,
    deploy_approval: false
  })
  const canEdit = computed(
    () =>
      access.can('service', 'update', { type: 'project', project_id: projectId }) ||
      access.can('project', 'update', {
        type: 'project',
        id: projectId,
        project_id: projectId
      })
  )
  const selectedServiceCanEdit = computed(
    () =>
      canEdit.value ||
      (!!selectedService.value && canManageService(selectedService.value, 'update'))
  )
  const canCreateService = computed(() =>
    access.can('service', 'create', { type: 'project', project_id: projectId })
  )
  const canImport = computed(() => canCreateService.value),
    canViewBuild = computed(() =>
      access.can('build', 'view', { type: 'project', project_id: projectId })
    ),
    canUpdateBuild = computed(() =>
      access.can('build', 'update', { type: 'project', project_id: projectId })
    )
  const navItems = computed(() => [
    { value: 'overview', label: '概览', icon: LayoutDashboard },
    {
      value: 'services',
      label: '服务配置',
      icon: Boxes,
      badge: services.value.length || undefined
    },
    {
      value: 'environments',
      label: '环境',
      icon: Server,
      badge: environments.value.length || undefined
    },
    {
      value: 'workflows',
      label: '工作流',
      icon: Workflow,
      badge: workflows.value.length || undefined
    },
    { value: 'tests', label: '测试', icon: TestTube2 },
    { value: 'versions', label: '版本管理', icon: GitCommit }
  ])
  const navTab = computed(() => (activeTab.value === 'build' ? 'workflows' : activeTab.value))
  const navCounts = computed(() => ({
    services: services.value.length,
    environments: environments.value.length,
    workflows: workflows.value.length
  }))
  const filteredServices = computed(() =>
    services.value.filter((item) =>
      `${item.name} ${item.code}`.toLowerCase().includes(keyword.value.toLowerCase())
    )
  )
  const tabTitle = computed(
      () => navItems.value.find((item) => item.value === activeTab.value)?.label || '概览'
    ),
    tabDescription = computed(() =>
      activeTab.value === 'overview' ? '项目交付概况与最近活动' : `${tabTitle.value}管理`
    )
  const allBuildsReady = computed(
    () =>
      services.value.length > 0 &&
      services.value.every((service) => configuredBuildServiceIds.value.has(service.id))
  )
  const servicesReady = computed(
    () =>
      services.value.length > 0 && services.value.every((service) => !!service.current_revision_id)
  )
  const environmentsReady = computed(
    () => environments.value.length > 0 && environments.value.every(isExecutableEnvironment)
  )
  const unboundDeploymentWorkflows = computed(() =>
    workflows.value.filter(
      (workflow) =>
        (workflow.requires_environment ||
          workflow.nodes?.some((node) => isDeploymentNode(node.type))) &&
        !environments.value.some(
          (environment) =>
            environment.id === workflow.target_environment_id &&
            isExecutableEnvironment(environment)
        )
    )
  )
  const initializationProgress = computed(
    () => `${initSteps.value.filter((step) => step.complete).length}/4 已完成`
  )
  const workflowsReady = computed(
    () =>
      workflows.value.length > 0 &&
      !unboundDeploymentWorkflows.value.length &&
      environments.value.every((environment) =>
        workflows.value.some(
          (workflow) =>
            workflow.target_environment_id === environment.id &&
            (workflow.requires_environment ||
              workflow.nodes?.some((node) => isDeploymentNode(node.type)))
        )
      )
  )
  const initSteps = computed(() => [
    {
      index: 1,
      value: 'environments',
      label: '环境',
      description: '集群与 Namespace',
      complete: environmentsReady.value
    },
    {
      index: 2,
      value: 'services',
      label: '服务配置',
      description: '当前环境的完整配置',
      complete: servicesReady.value
    },
    {
      index: 3,
      value: 'build',
      label: '构建配置',
      description: '仓库、脚本与 Agent',
      complete: allBuildsReady.value
    },
    {
      index: 4,
      value: 'workflows',
      label: '工作流与目标',
      description: '确认所选环境的工作流',
      complete: workflowsReady.value
    }
  ])
  const currentInitStepLabel = computed(
    () => initSteps.value.find((item) => item.value === activeTab.value)?.label || '项目初始化'
  )
  const firstIncompleteStep = () =>
    initSteps.value.find((step) => !step.complete) || initSteps.value[3]
  const persistInitializationStep = async (tab: string) => {
    const step = initSteps.value.find((item) => item.value === tab)
    if (!initializationMode.value || !step || !project.value) return true
    if (initializationSaving.value || runningAll.value) return false
    if (
      project.value.initialization_status === 'in_progress' &&
      project.value.initialization_step === step.index
    )
      return true
    initializationSaving.value = true
    const targetProject = projectId
    try {
      await updateProjectApi(targetProject, {
        initialization_status: 'in_progress',
        initialization_step: step.index
      })
      if (projectId !== targetProject || !project.value) return false
      project.value.initialization_status = 'in_progress'
      project.value.initialization_step = step.index
      return true
    } catch (error) {
      ElMessage.error(errorMessage(error, '初始化进度保存失败，请重试；已保存的服务和环境仍然保留'))
      return false
    } finally {
      initializationSaving.value = false
    }
  }
  const projectDeliveryType = computed(() => project.value?.delivery_type || 'k8s_yaml')
  const environmentHelmServices = ref<InitialHelmService[]>([]),
    environmentServicesBusy = ref(false)
  const environmentReview = ref<InstanceType<typeof HelmEnvironmentReview>>()
  const serviceFiles = (service: AxiOpsService) =>
    service.delivery_type === 'helm'
      ? [
          '@environment-values',
          ...(helmFilePaths[service.id]?.length
            ? helmFilePaths[service.id]
            : ['Chart.yaml', 'values.yaml'])
        ]
      : service.delivery_type === 'host'
        ? ['service.yaml']
        : ['deployment.yaml', 'service.yaml']
  const activeFile = computed({
    get: () => {
      const service = selectedService.value
      return service ? serviceActiveFiles[service.id] || serviceFiles(service)[0] : ''
    },
    set: (file: string) => {
      if (selectedService.value) serviceActiveFiles[selectedService.value.id] = file
    }
  })
  const isServiceExpanded = (id: number) => expandedServices.value.has(id),
    isBuildConfigured = (id: number) => configuredBuildServiceIds.value.has(id)
  const clusterName = (id: number) =>
    clusters.value.find((item) => item.id === id)?.name || `集群 #${id}`
  const currentEnvironment = computed(
    () =>
      environments.value.find((item) => item.id === selectedEnvironmentId.value) ||
      environments.value[0]
  )
  const selectEnvironment = (environment: AxiOpsEnvironment) => {
    selectedEnvironmentId.value = environment.id
    runtimeEnvironmentId.value = environment.id
    expandedServices.value = new Set()
  }
  const selectService = (service: AxiOpsService) => {
    selectedService.value = service
  }
  const toggleService = async (service: AxiOpsService) => {
    if (
      service.id !== selectedService.value?.id &&
      serviceConfiguration.value &&
      !(await serviceConfiguration.value.confirmLeave())
    )
      return
    selectService(service)
    const next = new Set(expandedServices.value)
    if (next.has(service.id)) next.delete(service.id)
    else next.add(service.id)
    expandedServices.value = next
  }
  const selectFile = async (file: string, service = selectedService.value) => {
    if (!service) return
    if (
      service.id !== selectedService.value?.id &&
      serviceConfiguration.value &&
      !(await serviceConfiguration.value.confirmLeave())
    )
      return
    selectService(service)
    activeFile.value = file
  }
  const openEnvironmentServiceConfiguration = async (serviceId: number) => {
    const service = services.value.find((s) => s.id === serviceId),
      eid = initializationMode.value
        ? currentEnvironment.value?.id || selectedEnvironmentId.value
        : runtimeEnvironmentId.value
    if (!service || !eid) return
    await selectTab('services')
    if (activeTab.value !== 'services') return
    selectService(service)
    await nextTick()
    await serviceConfiguration.value?.switchEnvironment(eid)
  }
  const selectTab = async (tab: string) => {
    if (!workspaceTabs.has(tab)) return
    if (initializationMode.value) {
      const target = initSteps.value.find((step) => step.value === tab)
      if (!target) tab = firstIncompleteStep().value
      else if (target.index > firstIncompleteStep().index)
        return void ElMessage.warning(`请先完成${firstIncompleteStep().label}`)
    }
    if (
      activeTab.value === 'services' &&
      tab !== 'services' &&
      serviceConfiguration.value &&
      !(await serviceConfiguration.value.confirmLeave())
    )
      return
    if (initializationMode.value && tab === 'services' && !environmentsReady.value)
      return void ElMessage.warning('请先完成环境配置')
    if (initializationMode.value && tab === 'build' && !environments.value.length)
      return void ElMessage.warning('请先完成环境配置')
    if (initializationMode.value && tab === 'workflows' && !allBuildsReady.value)
      return void ElMessage.warning('请先完成所有服务的构建配置')
    if (!(await persistInitializationStep(tab))) return
    activeTab.value = tab
    const query: Record<string, any> = { ...route.query, tab }
    delete query.setup
    await router.replace({ query })
  }
  const selectInitStep = (tab: string) => selectTab(tab)
  const nextInitStep = async () => {
    if (initializationSaving.value || workflowGenerating.value || loading.value) return
    const order = ['environments', 'services', 'build', 'workflows'],
      index = order.indexOf(activeTab.value)
    if (index < 0 || index >= order.length - 1) return
    if (activeTab.value === 'services' && !servicesReady.value)
      return void ElMessage.warning('请先创建服务并保存服务定义')
    if (activeTab.value === 'environments' && !environments.value.length)
      return openEnvironmentCreator()
    if (activeTab.value === 'environments' && !environmentsReady.value)
      return void ElMessage.warning('请先完成环境的运行目标配置')
    if (activeTab.value === 'build' && !allBuildsReady.value)
      return void ElMessage.warning('请先完成所有服务的构建配置')
    // Entering the final workflow step always materializes one workflow per
    // environment, so the user can immediately run the initial deployment.
    if (activeTab.value === 'build') {
      if (!(await generateWorkflows())) return
    }
    await selectTab(order[index + 1])
  }
  const previousInitStep = () => {
    const order = ['services', 'environments', 'build', 'workflows'],
      index = order.indexOf(activeTab.value)
    if (index > 0) {
      void selectTab(order[index - 1])
    }
  }
  const beginManualDirectory = () => {
      if (projectDeliveryType.value === 'helm') {
        void openTemplateImport()
        return
      }
      if (projectDeliveryType.value === 'host') {
        wizardSource.value = 'manual'
        serviceWizardVisible.value = true
        return
      }
      manualDirectoryName.value = ''
      manualDirectoryOpen.value = true
    },
    cancelManualDirectory = () => {
      manualDirectoryOpen.value = false
      manualDirectoryName.value = ''
    }
  const createManualDirectory = async () => {
    if (projectDeliveryType.value === 'helm') return
    const name = manualDirectoryName.value.trim()
    const nameError = serviceNameError(name)
    if (nameError) return void ElMessage.warning(nameError)
    const code = serviceCodeFromName(name)
    if (services.value.some((item) => item.code === code))
      return void ElMessage.warning('服务目录已存在')
    manualDirectorySaving.value = true
    try {
      const created = await createServiceApi(projectId, {
        name,
        code,
        description: '',
        source_type: 'manual',
        repo_url: '',
        repo_branch: 'main',
        language: '',
        framework: '',
        image_name: ''
      })
      cancelManualDirectory()
      await loadWorkspace()
      await persistInitializationStep('services')
      const fresh = services.value.find((item) => item.id === created.id)
      if (fresh) {
        selectService(fresh)
        expandedServices.value = new Set(expandedServices.value).add(fresh.id)
      }
      ElMessage.success('服务目录已创建')
    } catch (error: any) {
      ElMessage.error(error?.message || '服务目录创建失败')
    } finally {
      manualDirectorySaving.value = false
    }
  }
  const openImport = () => {
      if (!canImport.value) return
      if (projectDeliveryType.value === 'helm') {
        wizardSource.value = 'git'
        serviceWizardVisible.value = true
        return
      }
      bulkImportVisible.value = true
    },
    openTemplateImport = async () => {
      if (canImport.value) {
        if (projectDeliveryType.value === 'host')
          return void ElMessage.info('主机服务请使用手动创建或仓库导入')
        if (serviceConfiguration.value && !(await serviceConfiguration.value.confirmLeave())) return
        templateTarget.value = null
        templatePickerVisible.value = true
      }
    }
  const templateTarget = ref<AxiOpsService | null>(null)
  const openTemplateReplacement = async (service: AxiOpsService) => {
    if (!canManageService(service, 'update') || service.delivery_type === 'host') return
    if (serviceConfiguration.value && !(await serviceConfiguration.value.confirmLeave())) return
    selectService(service)
    templateTarget.value = service
    templatePickerVisible.value = true
  }
  const handleTemplateServiceSaved = async (
    serviceId: number,
    replaced = false,
    environmentId?: number
  ) => {
    templatePickerVisible.value = false
    if (environmentId) selectedEnvironmentId.value = environmentId
    await loadWorkspace()
    await persistInitializationStep('services')
    const created = services.value.find((item) => item.id === serviceId)
    if (created) {
      selectService(created)
      expandedServices.value = new Set(expandedServices.value).add(created.id)
      if (replaced) {
        await nextTick()
        await serviceConfiguration.value?.reloadSharedTemplate()
      }
    }
  }
  const openBuildConfig = (service?: AxiOpsService) => {
    buildService.value = service || selectedService.value
    if (!buildService.value) return void ElMessage.warning('请先选择服务')
    buildConfigVisible.value = true
  }
  const handleServiceSaved = async () => {
      serviceWizardVisible.value = false
      bulkImportVisible.value = false
      await loadWorkspace()
      await persistInitializationStep('services')
    },
    handleBuildSaved = async () => {
      buildConfigVisible.value = false
      // 构建配置里会把 code_source_id/repo_url/repo_branch 回写到服务，
      // 必须刷新服务列表，执行构建时才能按最新绑定动态拉取分支。
      await loadWorkspace()
      await persistInitializationStep('build')
    }
  const openEnvironmentCreator = () => {
    environmentHelmServices.value = []
    Object.assign(environmentForm, {
      name: '',
      code: '',
      cluster_id: clusters.value[0]?.id || 0,
      namespace: '',
      host_group_ref: '',
      is_production: false,
      deploy_approval: false
    })
    if (initializationMode.value && !environments.value.length) applyEnvironmentPreset('qa')
    environmentCreatorVisible.value = true
  }
  const environmentPresets = [
    { name: '开发环境', code: 'dev' },
    { name: '测试环境', code: 'qa' },
    { name: '预发布环境', code: 'staging' },
    { name: '生产环境', code: 'prod' }
  ]
  const applyEnvironmentPreset = (code: string) => {
    if (code === 'custom') {
      Object.assign(environmentForm, {
        name: '',
        code: '',
        namespace: '',
        is_production: false,
        deploy_approval: false
      })
      return
    }
    if (!code) code = 'qa'
    const preset = environmentPresets.find((e) => e.code === code)
    if (!preset) return
    Object.assign(environmentForm, preset, {
      namespace: `${project.value?.code}-${code}`,
      is_production: code === 'prod',
      deploy_approval: code === 'prod'
    })
  }
  const environmentPresetCode = computed({
    get: () =>
      environmentPresets.find(
        (preset) => preset.code === environmentForm.code && preset.name === environmentForm.name
      )?.code || 'custom',
    set: applyEnvironmentPreset
  })
  const syncNamespace = () => {
    const code = environmentForm.code
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9-]+/g, '-')
    environmentForm.namespace = code ? `${project.value?.code || 'project'}-${code}` : ''
    if (productionEnvironmentCode.value) environmentForm.is_production = true
  }
  const productionEnvironmentCode = computed(() =>
    ['prod', 'production'].includes(environmentForm.code.trim().toLowerCase())
  )
  const createEnvironment = async () => {
    if (environmentSaving.value || environmentServicesBusy.value) return
    if (!environmentForm.name.trim() && !environmentForm.code.trim()) applyEnvironmentPreset('qa')
    const code = environmentForm.code
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9-]+/g, '-')
    if (
      !environmentForm.name.trim() ||
      !code ||
      (projectDeliveryType.value !== 'host' &&
        (!environmentForm.cluster_id || !environmentForm.namespace.trim()))
    )
      return void ElMessage.warning('请完整填写环境名称、编码、集群和 Namespace')
    if (projectDeliveryType.value === 'host' && !environmentForm.host_group_ref.trim())
      return void ElMessage.warning('请输入主机组标识')
    if (environments.value.some((environment) => environment.code === code))
      return void ElMessage.warning('该环境编码已存在，请使用已有环境或更换编码')
    environmentSaving.value = true
    try {
      const environmentInput = {
        ...(projectDeliveryType.value === 'helm'
          ? { helm_services: environmentHelmServices.value }
          : {}),
        name: environmentForm.name.trim(),
        code,
        cluster_id: projectDeliveryType.value === 'host' ? 0 : environmentForm.cluster_id,
        namespace: projectDeliveryType.value === 'host' ? '' : environmentForm.namespace.trim(),
        host_group_ref:
          projectDeliveryType.value === 'host' ? environmentForm.host_group_ref.trim() : '',
        is_production: environmentForm.is_production || productionEnvironmentCode.value,
        deploy_approval: environmentForm.deploy_approval
      }
      if (environmentInput.helm_services?.length) {
        const snapshot = JSON.stringify(environmentInput)
        if (!environmentReview.value || !(await environmentReview.value.review(environmentInput)))
          return
        if (
          snapshot !==
          JSON.stringify({ ...environmentInput, helm_services: environmentHelmServices.value })
        )
          throw Error('环境配置已变化，请重新预览')
      }
      const created = await createEnvironmentApi(projectId, environmentInput)
      environmentCreatorVisible.value = false
      await loadWorkspace()
      if (created?.id) {
        selectedEnvironmentId.value = created.id
        runtimeEnvironmentId.value = created.id
        try {
          sessionStorage.setItem('axiops-runtime-environment-' + projectId, String(created.id))
        } catch {
          /* Storage can be unavailable. */
        }
      }
      await persistInitializationStep('environments')
      ElMessage.success(
        environmentHelmServices.value.length
          ? '环境和服务配置已创建，发布后生效'
          : '环境已创建，可继续配置服务'
      )
    } catch (error: any) {
      ElMessage.error(error?.message || '环境创建失败')
    } finally {
      environmentSaving.value = false
    }
  }
  const createDefaultEnvironments = () => {
    openEnvironmentCreator()
    applyEnvironmentPreset('qa')
  }
  const openEnvironmentConfig = (
    environment?: AxiOpsEnvironment,
    tab: 'variables' | 'resources' = 'variables'
  ) => {
    environmentConfigTab.value = tab
    selectedEnvironment.value = environment || environments.value[0] || null
    if (!selectedEnvironment.value) return void ElMessage.warning('请先创建环境')
    environmentConfigVisible.value = true
  }
  const workflowEnvironment = (workflow: AxiOpsWorkflow) =>
      environments.value.find((environment) => workflow.target_environment_id === environment.id),
    workflowEnvironmentName = (workflow: AxiOpsWorkflow) =>
      workflowEnvironment(workflow)?.name ||
      (workflow.requires_environment || workflow.nodes?.some((n) => isDeploymentNode(n.type))
        ? '未绑定环境（请先编排）'
        : '不部署到环境')
  const workflowRunList = (workflowId: number) => workflowRuns[workflowId] || []
  const terminalRunStatuses = new Set(['succeeded', 'success', 'completed', 'failed', 'cancelled'])
  const isSuccessfulRun = (status: string) => ['succeeded', 'success', 'completed'].includes(status)
  const workflowSuccessRate = (workflowId: number) => {
    const completed = workflowRunList(workflowId).filter((run) =>
      terminalRunStatuses.has(run.status)
    )
    if (!completed.length) return '-'
    return `${Math.round((completed.filter((run) => isSuccessfulRun(run.status)).length / completed.length) * 100)}%`
  }
  const formatRunDate = (value?: string) =>
    value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
  const workflowLastRunTime = (workflowId: number) =>
    formatRunDate(workflowRunList(workflowId)[0]?.created_at)
  const workflowLastRunner = (workflowId: number) => {
    return workflowActorLabel(workflowRunList(workflowId)[0])
  }
  const runStatusType = (status: string) =>
    isSuccessfulRun(status)
      ? 'success'
      : ['failed', 'cancelled'].includes(status)
        ? 'danger'
        : 'warning'
  const runStatusLabel = (status: string) =>
    (
      ({
        succeeded: '成功',
        success: '成功',
        completed: '成功',
        failed: '失败',
        cancelled: '已取消',
        running: '执行中',
        pending: '排队中'
      }) as Record<string, string>
    )[status] ||
    status ||
    '未知'
  const workflowStatusType = (workflowId: number) =>
    workflowRunList(workflowId)[0] ? runStatusType(workflowRunList(workflowId)[0].status) : 'info'
  const workflowStatusLabel = (workflowId: number) =>
    workflowRunList(workflowId)[0]
      ? runStatusLabel(workflowRunList(workflowId)[0].status)
      : '未执行'
  const environmentLabel = (id?: number) =>
    environments.value.find((item) => item.id === id)?.name || '未绑定环境'
  const allOverviewRuns = computed(() => {
    const rows: { run: AxiOpsWorkflowRun; workflow: AxiOpsWorkflow }[] = []
    workflows.value.forEach((workflow) => {
      ;(workflowRuns[workflow.id] || []).forEach((run) => rows.push({ run, workflow }))
    })
    return rows.sort((a, b) =>
      String(b.run.created_at || '').localeCompare(String(a.run.created_at || ''))
    )
  })
  const overviewRuns = computed(() => allOverviewRuns.value.slice(0, 8))
  const workflowAvgDuration = (workflowId: number) => {
    const runs = workflowRunList(workflowId).filter((run) => run.started_at && run.finished_at)
    if (!runs.length) return '-'
    const total = runs.reduce(
      (sum, run) =>
        sum + (new Date(run.finished_at!).getTime() - new Date(run.started_at!).getTime()) / 1000,
      0
    )
    const avg = Math.round(total / runs.length)
    const m = Math.floor(avg / 60)
    const s = avg % 60
    return m ? `${m} 分 ${s} 秒` : `${s} 秒`
  }
  const formatRunDuration = (run: AxiOpsWorkflowRun) => {
    if (!run.started_at) return formatRunDate(run.created_at)
    const end = run.finished_at ? new Date(run.finished_at).getTime() : Date.now()
    const seconds = Math.max(0, Math.round((end - new Date(run.started_at).getTime()) / 1000))
    return `${Math.floor(seconds / 60)}分${seconds % 60}秒`
  }
  const openWorkflowJob = (workflow: AxiOpsWorkflow) => {
    void router.push(`/cicd/projects/${projectId}/workflows/${workflow.id}`)
  }
  const repositoryPathFromUrl = (value: string) => {
    const source = String(value || '').trim()
    if (!source) return ''
    try {
      return new URL(source).pathname.replace(/^\/+|\/+$/g, '').replace(/\.git$/i, '')
    } catch {
      return source.replace(/^\/+|\/+$/g, '').replace(/\.git$/i, '')
    }
  }
  const loadExecuteServiceBranches = async (service: AxiOpsService) => {
    if (executeBranchLoading.value[service.id] || executeServiceBranches.value[service.id]?.length)
      return
    executeBranchLoading.value[service.id] = true
    const request = executeRequest
    executeBranchErrors.value[service.id] = ''
    try {
      if (!service.code_source_id || !service.repo_url)
        throw new Error('branch source is not configured')
      const branches = await listBranchesApi(
        service.code_source_id,
        repositoryPathFromUrl(service.repo_url)
      )
      if (request === executeRequest) executeServiceBranches.value[service.id] = branches || []
    } catch (error) {
      if (request === executeRequest)
        executeBranchErrors.value[service.id] =
          `${errorMessage(error, '分支读取失败')}；当前分支未验证，可重新展开列表重试。`
    } finally {
      if (request === executeRequest) executeBranchLoading.value[service.id] = false
    }
  }
  const reloadExecuteWorkflow = async () => {
    const workflow = executeWorkflowTarget.value
    if (!workflow) return
    const request = ++executeRequest
    executeLoading.value = true
    executeLoadError.value = ''
    try {
      const detail = await getWorkflowApi(workflow.id)
      if (request !== executeRequest) return
      if (detail.target_environment_id) {
        const environment = await getEnvironmentApi(detail.target_environment_id)
        if (request !== executeRequest) return
        if (
          !environment ||
          environment.id !== detail.target_environment_id ||
          environment.project_id !== projectId
        )
          throw new Error('工作流目标环境不可用，请重新加载环境配置')
        environments.value = environments.value.some((item) => item.id === environment.id)
          ? environments.value.map((item) => (item.id === environment.id ? environment : item))
          : [...environments.value, environment]
      }
      executeWorkflowTarget.value = detail
      if (
        executeProductionBuild.value &&
        detail.target_environment_id !== executeProductionBuildTarget.value
      )
        throw new Error('工作流目标环境已变化，请重新选择发布方式并核对目标环境')
      executeEnvironmentId.value = detail.target_environment_id || 0
      if (
        workflowEnvironment(detail)?.is_production &&
        projectDeliveryType.value === 'k8s_yaml' &&
        !executeProductionBuild.value
      ) {
        executeDialogVisible.value = false
        executeWorkflowTarget.value = null
        productionWorkflow.value = detail
        productionVisible.value = true
        return
      }
      executeScope.value = executeCanOverride.value ? 'selected' : 'all'
      const serviceNodes = (detail.nodes || []).filter((node) =>
        ['build', 'test', 'yaml_deploy', 'helm_deploy', 'host_deploy'].includes(node.type)
      )
      // Prefer the workflow's service over the workspace's unrelated selection.
      const boundServiceId = serviceNodes
        .map((node) => Number(node.config?.service_id || 0))
        .find(Boolean)
      const defaultServiceId =
        (executeMultiService.value ? executeWorkflowServiceIds.value[0] : boundServiceId) ||
        selectedService.value?.id
      executeServiceIds.value =
        executeCanOverride.value &&
        executeSelectableServices.value.some((service) => service.id === defaultServiceId)
          ? [String(defaultServiceId)]
          : []
      for (const service of services.value) {
        const savedBranches = [
          ...new Set(
            serviceNodes
              .filter(
                (node) =>
                  ['build', 'test'].includes(node.type) &&
                  Number(node.config?.service_id) === service.id
              )
              .map((node) => String(node.config?.branch || '').trim())
              .filter(Boolean)
          )
        ]
        // A one-off execution starts from the service's current configured branch.
        // The workflow node branch is historical and must not silently override it.
        executeBranchValues.value[service.id] =
          service.repo_branch || (savedBranches.length === 1 ? savedBranches[0] : '')
      }
    } catch (error) {
      if (request === executeRequest)
        executeLoadError.value = errorMessage(error, '工作流读取失败，请重试')
    } finally {
      if (request === executeRequest) executeLoading.value = false
    }
  }
  const openWorkflowExecute = async (
    workflow: AxiOpsWorkflow,
    buildProduction: boolean = false
  ) => {
    if (executingWorkflow.value) return
    executeProductionBuild.value = buildProduction === true
    productionInitialServiceIds.value = []
    executeAdvanced.value = false
    executeProductionBuildTarget.value =
      buildProduction === true ? workflow.target_environment_id || 0 : 0
    const environment = workflowEnvironment(workflow)
    if (
      environment?.is_production &&
      ['k8s_yaml', 'helm'].includes(projectDeliveryType.value) &&
      !executeProductionBuild.value
    ) {
      executeRequest++
      executeLoading.value = false
      executeDialogVisible.value = false
      executeWorkflowTarget.value = null
      productionWorkflow.value = workflow
      productionVisible.value = true
      return
    }
    executeWorkflowTarget.value = workflow
    executeEnvironmentId.value = environment?.id || 0
    executeScope.value = 'selected'
    executeServiceIds.value = []
    executeSubmitted.value = []
    executeUncertain.value = false
    executeSubmitMessage.value = ''
    executeServiceBranches.value = {}
    executeBranchLoading.value = {}
    executeBranchErrors.value = {}
    executeBranchValues.value = {}
    executeSourceBuildIds.value = {}
    executeMode.value = 'build'
    services.value.forEach((service) => {
      executeBranchValues.value[service.id] = service.repo_branch || ''
    })
    executeDialogVisible.value = true
    await reloadExecuteWorkflow()
  }
  const openWorkflowBinding = () => {
    const workflow = executeWorkflowTarget.value
    if (!workflow) return
    executeDialogVisible.value = false
    void router.push(`/cicd/projects/${projectId}/workflows/${workflow.id}?configure=1`)
  }
  const confirmWorkflowExecute = async () => {
    const workflow = executeWorkflowTarget.value
    const environment = executeEnvironment.value
    if (
      executingWorkflow.value ||
      executeLoading.value ||
      executeLoadError.value ||
      executeUncertain.value ||
      !workflow
    )
      return
    if (
      environment?.is_production &&
      projectDeliveryType.value === 'k8s_yaml' &&
      (!executeProductionBuild.value || executeMode.value !== 'build')
    ) {
      executeDialogVisible.value = false
      executeWorkflowTarget.value = null
      productionWorkflow.value = workflow
      productionVisible.value = true
      return
    }
    const needsEnvironment =
      workflow.nodes?.some((node) => isDeploymentNode(node.type)) ||
      !!workflow.target_environment_id
    if (
      needsEnvironment &&
      (!environment ||
        !isExecutableEnvironment(environment) ||
        environment.id !== workflow.target_environment_id)
    )
      return void ElMessage.warning('工作流绑定的目标环境不可用或不匹配，请检查编排与环境状态')
    const targets =
      executeScope.value === 'all'
        ? [0]
        : executeServiceIds.value
            .map(Number)
            .filter((id) => !executeSubmitted.value.some((item) => item.serviceId === id))
    if (executeScope.value === 'all' && executeSubmitted.value.length) return
    if (
      executeMode.value === 'existing_image' &&
      (!executeCanPromote.value || executeScope.value !== 'selected')
    )
      return void ElMessage.warning('复用镜像请选择支持的服务与对应成功构建')
    if (executeScope.value === 'selected') {
      if (!executeCanOverride.value)
        return void ElMessage.warning('此工作流包含多个服务，请按工作流配置执行')
      if (!targets.length) return void ElMessage.warning('请至少选择一个尚未提交的服务')
      if (targets.length > 1 && !executeMultiService.value)
        return void ElMessage.warning('此工作流只配置了一个服务，请单独执行')
      for (const id of targets) {
        const service = executeSelectableServices.value.find((item) => item.id === id)
        if (!service) return void ElMessage.warning('所选服务不属于当前工作流，请重新加载')
        if (
          !service?.current_revision_id ||
          (executeMode.value === 'build' && !executeBranchValues.value[id]?.trim())
        )
          return void ElMessage.warning(`请先为「${service?.name || id}」保存服务版本并选择分支`)
        if (executeMode.value === 'existing_image' && !selectedPromotionBuild(id))
          return void ElMessage.warning(`请为「${service?.name || id}」选择已有成功镜像`)
      }
    }
    if (
      environment &&
      (environment.is_production || environment.deploy_approval) &&
      !workflow.nodes?.some((node) => node.type === 'approval')
    )
      return void ElMessage.warning('目标环境要求审批，请先在工作流中配置审批节点')
    executingWorkflow.value = true
    executeSubmitMessage.value = ''
    // Snapshot inputs once; edits must never redirect a partially submitted batch.
    const branches = { ...executeBranchValues.value }
    const mode = executeMode.value,
      sourceBuildIds = { ...executeSourceBuildIds.value }
    let awaitingSubmission = false
    try {
      let latestRun = 0
      if (executeScope.value === 'selected' && targets.length > 1) {
        const selections = []
        for (const serviceId of targets) {
          const config = await getServiceEnvironmentConfigApi(environment!.id, serviceId)
          if (
            !config ||
            config.environment_id !== environment!.id ||
            config.service_id !== serviceId ||
            !config.revision_id
          )
            throw new Error('目标环境的服务配置无效，请保存后重试')
          selections.push({
            service_id: serviceId,
            service_revision_id: config.revision_id,
            ...(mode === 'existing_image'
              ? { source_build_id: sourceBuildIds[serviceId] }
              : { branch: branches[serviceId].trim() })
          })
        }
        awaitingSubmission = true
        const result = await runWorkflowApi(workflow.id, {
          environment_id: environment!.id,
          trigger_type: 'manual',
          execution_mode: mode,
          use_environment_config: true,
          services: selections,
          deployment_strategy: 'rolling'
        })
        awaitingSubmission = false
        executeSubmitted.value.push({ id: result.id, serviceId: 0, name: workflow.name })
        executeServiceIds.value = []
        executeDialogVisible.value = false
        ElMessage.success('已提交一次多服务运行，可查看构建和部署进度')
        await router.push(`/cicd/projects/${projectId}/workflows/${workflow.id}/runs/${result.id}`)
        return
      }
      for (const serviceId of targets) {
        const service = services.value.find((item) => item.id === serviceId)
        const config = service
          ? await getServiceEnvironmentConfigApi(environment!.id, service.id)
          : null
        if (
          service &&
          (!config ||
            config.environment_id !== environment!.id ||
            config.service_id !== service.id ||
            !config.revision_id)
        )
          throw new Error(`「${service.name}」的目标环境配置无效，请在服务配置中保存后重试`)
        awaitingSubmission = true
        const result = await runWorkflowApi(workflow.id, {
          environment_id: environment?.id || 0,
          trigger_type: 'manual',
          execution_mode: mode,
          ...(environment?.is_production ? { use_environment_config: true } : {}),
          ...(service
            ? {
                service_id: service.id,
                service_revision_id: config!.revision_id,
                ...(executeMultiService.value ? { select_workflow_service: true } : {}),
                use_environment_config: true,
                ...(mode === 'existing_image'
                  ? { execution_mode: mode, source_build_id: sourceBuildIds[service.id] }
                  : { branch: branches[service.id].trim() })
              }
            : {}),
          deployment_strategy: 'rolling'
        })
        awaitingSubmission = false
        latestRun = result.id
        executeSubmitted.value.push({
          id: result.id,
          serviceId,
          name: service?.name || workflow.name
        })
        executeServiceIds.value = executeServiceIds.value.filter((id) => Number(id) !== serviceId)
      }
      executeDialogVisible.value = false
      ElMessage.success(`已提交 ${executeSubmitted.value.length} 个运行，可查看执行进度和日志`)
      await router.push(`/cicd/projects/${projectId}/workflows/${workflow.id}/runs/${latestRun}`)
    } catch (error: any) {
      executeUncertain.value = awaitingSubmission && !(error?.code >= 400 && error?.code < 500)
      executeSubmitMessage.value = `已确认提交 ${executeSubmitted.value.length} 个运行；${errorMessage(error, '提交失败')}。${executeUncertain.value ? '本次提交结果尚未确认，已暂停继续提交。请先查看运行记录，避免重复发布。' : '已提交的服务不会再次提交，修正错误后可提交剩余服务。'}`
    } finally {
      executingWorkflow.value = false
    }
  }
  const refreshWorkflowJob = async () => {
    if (!selectedWorkflow.value) return
    workflowRunsLoading.value = true
    try {
      const result = await listWorkflowRunsApi(selectedWorkflow.value.id, {
        page: 1,
        page_size: 50
      })
      workflowRuns[selectedWorkflow.value.id] = result.list || []
    } finally {
      workflowRunsLoading.value = false
    }
  }
  const openWorkflowRun = (runId: number, workflow?: AxiOpsWorkflow) => {
    const target = workflow || selectedWorkflow.value
    if (target) void router.push(`/cicd/projects/${projectId}/workflows/${target.id}/runs/${runId}`)
    else {
      selectedRunId.value = runId
      runDrawerVisible.value = true
    }
  }
  const loadWorkflowRuns = async () => {
    const request = workspaceRequest,
      items = [...workflows.value]
    const results = await mapLimited(items, (workflow) =>
      listWorkflowRunsApi(workflow.id, { page: 1, page_size: 50 })
    )
    if (request !== workspaceRequest) return
    results.forEach((result, index) => {
      if (result.status === 'fulfilled') workflowRuns[items[index].id] = result.value.list || []
      else workspaceWarnings.value.push(`「${items[index].name}」运行记录读取失败，请刷新重试`)
    })
  }
  const generateWorkflows = async () => {
    if (workflowGenerating.value) return
    const deliveryType = projectDeliveryType.value
    if (deliveryType === 'unclassified') return void ElMessage.warning('请先设置项目交付类型')
    if (!services.value.length || !environments.value.length)
      return void ElMessage.warning('请先完成服务和环境配置')
    if (!allBuildsReady.value) return void ElMessage.warning('请先完成所有服务的构建配置')
    workflowGenerating.value = true
    try {
      // Always use persisted environments and workflows, including successful
      // creations from an earlier partially failed attempt. Never add presets here.
      const [savedEnvironments, savedWorkflows] = await Promise.all([
        listEnvironmentsApi(projectId, { silent: true }),
        listWorkflowsApi(projectId, { silent: true })
      ])
      environments.value = savedEnvironments || []
      workflows.value = savedWorkflows || []
      if (!environmentsReady.value) throw new Error('请先完成环境的运行目标配置')
      for (const environment of environments.value) {
        const code = `${project.value?.code}-${environment.code}`.slice(0, 64)
        if (
          workflows.value.some(
            (item) =>
              item.target_environment_id === environment.id &&
              (item.requires_environment || item.nodes?.some((node) => isDeploymentNode(node.type)))
          )
        )
          continue
        if (workflows.value.some((item) => item.code === code))
          throw new Error(`工作流编码 ${code} 已被其他配置使用，请先检查其目标环境`)
        const nodes: WorkflowNode[] = []
        if (environment.is_production || environment.deploy_approval)
          nodes.push({
            key: 'approval',
            name: '发布审批',
            type: 'approval',
            config: {},
            timeout_seconds: 86400,
            retry_limit: 0
          })
        for (const service of services.value) {
          const buildKey = `build-${service.code}`.slice(0, 64)
          nodes.push(
            {
              key: buildKey,
              name: `构建 ${service.name}`,
              type: 'build',
              depends_on:
                environment.is_production || environment.deploy_approval ? ['approval'] : [],
              config: { service_id: service.id, branch: service.repo_branch || 'main' },
              timeout_seconds: 3600,
              retry_limit: 1
            },
            {
              key: `deploy-${service.code}`.slice(0, 64),
              name: `部署 ${service.name}`,
              type:
                projectDeliveryType.value === 'helm'
                  ? 'helm_deploy'
                  : projectDeliveryType.value === 'host'
                    ? 'host_deploy'
                    : 'yaml_deploy',
              depends_on: [buildKey],
              config: { service_id: service.id, service_revision_id: service.current_revision_id },
              timeout_seconds: 1800,
              retry_limit: 0
            }
          )
        }
        await createWorkflowApi(projectId, {
          target_environment_id: environment.id,
          name: `${project.value?.name}-${environment.name}`,
          code,
          description: `${environment.name}的标准构建与部署工作流`,
          definition: { delivery_type: deliveryType, nodes }
        })
      }
      workflows.value = await listWorkflowsApi(projectId, { silent: true })
      await loadWorkflowRuns()
      ElMessage.success('环境工作流已生成')
      return true
    } catch (error: any) {
      ElMessage.error(error?.message || '工作流生成失败')
    } finally {
      workflowGenerating.value = false
    }
  }
  const completeInitialization = async () => {
    if (runningAll.value || initializationSaving.value || loading.value || loadError.value) return
    runningAll.value = true
    try {
      if (!servicesReady.value || !environmentsReady.value)
        return void ElMessage.warning('请先完成服务定义和环境运行目标配置')
      await loadBuildStates()
      if (!allBuildsReady.value) return void ElMessage.warning('请先补齐构建配置')
      if (unboundDeploymentWorkflows.value.length) {
        activeTab.value = 'workflows'
        return void ElMessage.warning('请先在工作流编排中绑定所有部署工作流的目标环境')
      }
      if (!workflowsReady.value && !(await generateWorkflows())) return
      if (!workflowsReady.value) return void ElMessage.warning('请先确认各环境对应的工作流')
      await updateProjectApi(projectId, {
        initialization_status: 'completed',
        initialization_step: 4
      })
      initializationMode.value = false
      if (project.value) {
        project.value.initialization_status = 'completed'
        project.value.initialization_step = 4
      }
      activeTab.value = 'workflows'
      await router.replace({ query: { tab: 'workflows' } })
      ElMessage.success('项目配置完成。执行时请核对工作流已绑定的目标环境。')
    } catch (error: any) {
      ElMessage.error(error?.message || '完成配置失败，请重试')
    } finally {
      runningAll.value = false
    }
  }
  const loadBuildStates = async (request = workspaceRequest) => {
    const items = [...services.value]
    const results = await mapLimited(items, (service) => getBuildConfigApi(service.id)),
      ids = new Set<number>()
    if (request !== workspaceRequest) return
    const issues: Record<number, string[]> = {}
    results.forEach((result, index) => {
      const service = items[index]
      issues[service.id] =
        result.status === 'fulfilled'
          ? buildConfigurationIssues(service, result.value)
          : [
              isBuildConfigMissing(result.reason)
                ? '尚未配置构建'
                : `读取失败：${errorMessage(result.reason, '请刷新重试')}`
            ]
      if (!issues[service.id].length) ids.add(service.id)
    })
    configuredBuildServiceIds.value = ids
    buildIssues.value = issues
  }
  watch(activeTab, (tab) => {
    if (loading.value || !project.value || project.value.status === 'deleted') return
    if (tab === 'build' || tab === 'overview') void loadBuildStates()
    if (tab === 'workflows' || tab === 'overview') void loadWorkflowRuns()
  })
  const loadWorkspace = async () => {
    if (!Number.isInteger(projectId) || projectId <= 0) {
      loadError.value = '无效的项目编号'
      return
    }
    loading.value = true
    loadError.value = ''
    workspaceWarnings.value = []
    const request = ++workspaceRequest,
      targetProject = projectId
    try {
      const result = await getProjectApi(targetProject, { silent: true })
      if (request !== workspaceRequest) return
      project.value = result
      if (result.status === 'deleted') {
        initializationMode.value = false
        return
      }
      initializationMode.value = project.value.initialization_status !== 'completed'
      if (!initializationMode.value && route.query.init === '1') {
        const cleanQuery: Record<string, any> = { ...route.query }
        delete cleanQuery.init
        await router.replace({ query: cleanQuery })
      }
      // 初始化已完成、且 URL 上没有明确指定 tab 时，直接落到工作流
      if (!initializationMode.value && !workspaceLoaded && !workspaceTabs.has(requestedTab))
        activeTab.value = 'workflows'
      const [
        serviceResult,
        environmentResult,
        clusterResult,
        workflowResult,
        runResult,
        sourceResult
      ] = await Promise.allSettled([
        listAllServicesApi(targetProject, {}, { silent: true }),
        listEnvironmentsApi(targetProject, { silent: true }),
        listClustersApi(targetProject, { silent: true }),
        listWorkflowsApi(targetProject, { silent: true }),
        listBuildRunsApi(targetProject, { silent: true }),
        listCodeSourcesApi(targetProject, { silent: true })
      ])
      if (request !== workspaceRequest) return
      const required = [
        [serviceResult, '服务列表'],
        [environmentResult, '环境列表'],
        [workflowResult, '工作流列表']
      ] as const
      for (const [result, label] of required) {
        if (result.status === 'rejected')
          throw new Error(`${label}读取失败：${errorMessage(result.reason, '请检查权限或连接')}`)
      }
      for (const [result, label] of [
        [clusterResult, '集群列表'],
        [runResult, '构建记录'],
        [sourceResult, '代码源列表']
      ] as const) {
        if (result.status === 'rejected')
          workspaceWarnings.value.push(
            `${label}读取失败：${errorMessage(result.reason, '请刷新重试')}`
          )
      }
      services.value = serviceResult.status === 'fulfilled' ? serviceResult.value || [] : []
      environments.value =
        environmentResult.status === 'fulfilled' ? environmentResult.value || [] : []
      if (
        !currentEnvironment.value ||
        !environments.value.some((item) => item.id === currentEnvironment.value?.id)
      )
        selectedEnvironmentId.value = environments.value[0]?.id || 0
      if (
        !runtimeEnvironmentId.value ||
        !environments.value.some((item) => item.id === runtimeEnvironmentId.value)
      ) {
        let remembered = 0
        try {
          remembered = Number(sessionStorage.getItem('axiops-runtime-environment-' + projectId))
        } catch {
          /* No browser storage. */
        }
        runtimeEnvironmentId.value =
          environments.value.find((item) => item.id === remembered)?.id ||
          environments.value[0]?.id ||
          0
      }
      clusters.value = clusterResult.status === 'fulfilled' ? clusterResult.value || [] : []
      workflows.value = workflowResult.status === 'fulfilled' ? workflowResult.value || [] : []
      runs.value = runResult.status === 'fulfilled' ? runResult.value.list || [] : []
      codeSources.value = sourceResult.status === 'fulfilled' ? sourceResult.value || [] : []
      selectedService.value =
        services.value.find((item) => item.id === selectedService.value?.id) ||
        services.value[0] ||
        null
      if (selectedService.value)
        expandedServices.value = new Set(expandedServices.value).add(selectedService.value.id)
      // Resume from saved resources. Merely opening a project never creates
      // environments/workflows or marks initialization completed.
      if (initializationMode.value && !workspaceLoaded) {
        if (servicesReady.value && environmentsReady.value) await loadBuildStates(request)
        else void loadBuildStates(request)
        if (request !== workspaceRequest) return
        const requested = initSteps.value.find(
          (step) => step.value === String(route.query.tab || '')
        )
        activeTab.value =
          requested && requested.index <= firstIncompleteStep().index
            ? requested.value
            : firstIncompleteStep().value
        await router.replace({ query: { ...route.query, init: '1', tab: activeTab.value } })
      } else if (initializationMode.value || activeTab.value !== 'services')
        void loadBuildStates(request)
      workspaceLoaded = true
      if (activeTab.value === 'workflows' || activeTab.value === 'overview') void loadWorkflowRuns()
    } catch (error: any) {
      if (request !== workspaceRequest) return
      loadError.value =
        error?.code === 404 || error?.code === 50005
          ? '项目不存在或当前账号无权访问'
          : error?.code === 403 || error?.code === 10003
            ? '当前账号无权访问此项目'
            : errorMessage(error, '项目服务暂时不可用，请稍后重试')
    } finally {
      if (request === workspaceRequest) loading.value = false
    }
  }
  const openRequestedExecution = async () => {
    if (project.value?.status === 'deleted') return
    if (loading.value || loadError.value || project.value?.id !== projectId) return
    const target = workflows.value.find((item) => item.id === Number(route.query.execute))
    if (target) {
      await openWorkflowExecute(target, route.query.execution_mode === 'build')
      if (route.query.build_services && executeWorkflowTarget.value && !executeLoadError.value) {
        const raw = String(route.query.build_services).split(',')
        const ids = raw.map(Number)
        const detail = executeWorkflowTarget.value
        const sourceId = Number(route.query.build_environment)
        const expected = executeEnvironment.value
        const valid =
          raw.every((id) => /^[1-9]\d*$/.test(id)) &&
          ids.length > 0 &&
          ids.length <= 50 &&
          new Set(ids).size === ids.length &&
          expected?.id === sourceId &&
          detail.target_environment_id === sourceId &&
          executeCanOverride.value &&
          (ids.length === 1 || executeMultiService.value) &&
          ids.every(
            (sid) =>
              executeSelectableServices.value.some((s) => s.id === sid) &&
              detail.nodes?.some(
                (n) => n.type === 'build' && Number(n.config?.service_id) === sid
              ) &&
              detail.nodes?.some(
                (n) => n.type === 'yaml_deploy' && Number(n.config?.service_id) === sid
              )
          )
        if (valid) {
          executeMode.value = 'build'
          executeScope.value = 'selected'
          executeServiceIds.value = ids.map(String)
          executeSubmitMessage.value =
            '为生产准备新镜像：本次先在来源环境构建、部署；测试验收通过后，再发布同一镜像到生产。'
        } else {
          executeServiceIds.value = []
          executeLoadError.value =
            '来源工作流、环境或所选服务已变化，请返回生产发布重新选择构建流程。'
        }
      }
      const query = { ...route.query }
      delete query.execute
      delete query.build_services
      delete query.build_environment
      delete query.execution_mode
      await router.replace({ query })
    }
  }
  onMounted(async () => {
    await access.load()
    await loadWorkspace()
    await openRequestedExecution()
  })
  onBeforeRouteLeave(async () => {
    if (activeTab.value === 'services' && serviceConfiguration.value)
      return serviceConfiguration.value.confirmLeave()
    return true
  })
  onBeforeUnmount(() => {
    workspaceRequest++
    executeRequest++
  })
  watch(
    () => route.query.execute,
    () => {
      void openRequestedExecution()
    }
  )
  watch(
    () => route.query.tab,
    (tab) => {
      const next = String(tab || '')
      if (loading.value || !workspaceTabs.has(next)) return
      if (initializationMode.value) {
        const requested = initSteps.value.find((step) => step.value === next)
        activeTab.value =
          requested && requested.index <= firstIncompleteStep().index
            ? next
            : firstIncompleteStep().value
      } else activeTab.value = next
    }
  )
  watch(
    () => route.params.projectId,
    async (value) => {
      const nextId = Number(value)
      if (nextId === projectId) return
      projectId = nextId
      workspaceLoaded = false
      configuredBuildServiceIds.value = new Set()
      buildIssues.value = {}
      project.value = null
      services.value = []
      environments.value = []
      workflows.value = []
      selectedService.value = null
      await loadWorkspace()
    }
  )
</script>

<style scoped lang="scss">
  @use '@/views/cicd/styles/cicd-block.scss' as *;
  .cicd-workspace {
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding-bottom: 68px;
    background: #fff;
    color: #1e293b;
  }
  .heading-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .initialization-bar {
    display: grid;
    grid-template-columns: 190px minmax(620px, 1fr) auto;
    align-items: center;
    gap: 18px;
    padding: 12px 24px;
    border-bottom: 1px solid #d1fae5;
    background: #f0fdf4;
  }
  .initialization-title {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .initialization-title strong {
    color: #047857;
    font-size: 13px;
  }
  .initialization-title span {
    color: #64748b;
    font-size: 11px;
  }
  .initialization-steps {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 7px;
  }
  .init-step {
    min-width: 0;
    display: grid;
    grid-template-columns: 25px minmax(0, 1fr) auto;
    align-items: center;
    gap: 8px;
    padding: 7px 9px;
    border: 1px solid #dbe5e1;
    border-radius: 5px;
    background: #fff;
    color: #64748b;
    text-align: left;
    cursor: pointer;
  }
  .init-step > span {
    width: 23px;
    height: 23px;
    display: grid;
    place-items: center;
    border-radius: 50%;
    background: #f1f5f9;
    font-size: 11px;
  }
  .init-step div {
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .init-step strong,
  .init-step small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .init-step strong {
    color: #334155;
    font-size: 12px;
  }
  .init-step small {
    margin-top: 2px;
    font-size: 10px;
  }
  .init-step.is-active {
    border-color: #10b981;
    box-shadow: 0 0 0 1px #10b981;
  }
  .init-step.is-active > span,
  .init-step.is-complete > span {
    background: #10b981;
    color: #fff;
  }
  .init-step.is-complete > svg {
    color: #059669;
  }
  .workspace-main {
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
    background: #fff;
  }
  .services-layout {
    height: 100%;
    min-height: 0;
    display: grid;
    grid-template-columns: clamp(236px, 20vw, 300px) minmax(0, 1fr);
    overflow: hidden;
    background: #fff;
  }
  .service-catalog {
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 14px 10px;
    overflow: hidden;
    border-right: 1px solid #e2e8f0;
    background: #fff;
  }
  .catalog-actions {
    min-height: 32px;
    display: flex;
    align-items: center;
    gap: 4px;
    margin: 0 2px 10px;
  }
  .catalog-actions :deep(.el-button) {
    flex: 1 1 0;
    min-width: 0;
    padding: 5px 6px;
    color: #334155;
    font-size: 12px;
  }
  .manual-directory-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    gap: 5px;
    align-items: center;
    margin: 0 2px 8px;
  }
  .manual-directory-row.is-inline {
    grid-template-columns: minmax(0, 1fr) 28px auto;
    margin: 8px 2px 2px 24px;
  }
  .manual-directory-row.is-inline :deep(.el-input__wrapper) {
    box-shadow: 0 0 0 1px #3b82f6 inset;
  }
  .manual-directory-row.is-inline :deep(.el-button) {
    min-width: 24px;
    padding: 4px;
  }
  .service-tree {
    min-height: 0;
    flex: 1 1 auto;
    margin-top: 12px;
    overflow: auto;
  }
  .service-node-button,
  .file-item {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 7px;
    padding: 8px 9px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: #334155;
    text-align: left;
    cursor: pointer;
  }
  .service-node-button:hover,
  .service-node-button.is-active,
  .file-item:hover,
  .file-item.is-active {
    background: #ecfdf5;
    color: #047857;
  }
  .file-item span {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .file-list {
    margin: 2px 0 7px 22px;
    border-left: 1px solid #d1fae5;
  }
  .catalog-secondary {
    display: flex;
    justify-content: flex-end;
    margin-top: 5px;
  }
  .catalog-secondary .el-button {
    gap: 5px;
    color: #8793a5;
    font-size: 11px;
  }
  .service-row {
    display: flex;
    align-items: center;
    border: 1px solid transparent;
    border-radius: 8px;
    margin-bottom: 4px;
  }
  .service-row:hover {
    background: #f5f7fb;
  }
  .service-row.selected {
    border-color: #dce4fa;
    background: #eff4ff;
  }
  .service-row .service-node-button {
    flex: 1;
    min-width: 0;
    width: auto;
    padding: 10px 5px;
    gap: 5px;
  }
  .service-row .service-node-button:hover,
  .service-row .service-node-button.is-active {
    background: transparent;
    color: #334155;
  }
  .service-node-button > svg {
    flex-shrink: 0;
    color: #94a3b8;
  }
  .service-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 22px;
    height: 22px;
    border-radius: 5px;
    color: #5578cb;
    background: #e7eefc;
  }
  .service-identity {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
    gap: 3px;
  }
  .service-identity strong {
    font-size: 12px;
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .service-identity small {
    color: #94a3b8;
    font-size: 10px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .service-type {
    color: #8492a9;
    font-size: 9px;
    flex-shrink: 0;
  }
  .service-more {
    padding: 3px !important;
    margin-right: 3px;
    color: #8492a9;
  }
  .service-context-menu {
    position: fixed;
    z-index: 3000;
    width: 158px;
    padding: 6px;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    background: #fff;
    box-shadow: 0 12px 30px rgb(15 23 42 / 16%);
  }
  .context-menu-heading {
    padding: 7px 9px 6px;
    overflow: hidden;
    color: #64748b;
    font-size: 11px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .service-context-menu button {
    display: flex;
    align-items: center;
    width: 100%;
    gap: 6px;
    padding: 7px 8px;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: #334155;
    font-size: 12px;
    text-align: left;
    cursor: pointer;
  }
  .service-context-menu button:hover {
    background: #eff6ff;
    color: #2563eb;
  }
  .context-menu-divider {
    height: 1px;
    margin: 5px 3px;
    background: #eef2f7;
  }
  .file-item {
    padding: 6px 9px;
    color: #64748b;
    font-size: 12px;
  }
  .content-panel {
    padding: 28px;
    background: #fff;
  }
  .panel-heading {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 20px;
    margin-bottom: 24px;
  }
  .panel-heading h2 {
    margin: 5px 0;
    font-size: 20px;
  }
  .panel-heading p {
    margin: 0;
    color: #64748b;
    font-size: 13px;
  }
  .view-eyebrow {
    color: #047857;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.08em;
  }
  .default-env-hint,
  .initial-run-box {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    margin-bottom: 18px;
    padding: 15px 16px;
    border: 1px solid #a7f3d0;
    border-radius: 6px;
    background: #f0fdf4;
  }
  .default-env-hint > div,
  .initial-run-box > div {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .default-env-hint span,
  .initial-run-box span {
    color: #64748b;
    font-size: 12px;
  }
  .environment-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(310px, 1fr));
    gap: 14px;
  }
  .environment-workspace {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .form-help {
    display: block;
    margin-top: 5px;
    color: #64748b;
    font-size: 12px;
    line-height: 1.5;
  }
  .environment-tabs {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    overflow-x: auto;
  }
  .runtime-config-actions {
    display: flex;
    justify-content: flex-end;
    margin-top: -4px;
  }
  .environment-tab {
    min-width: 112px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 8px 12px;
    border: 1px solid transparent;
    background: transparent;
    color: #64748b;
    cursor: pointer;
  }
  .environment-tab span {
    font-weight: 600;
  }
  .environment-tab small {
    color: #94a3b8;
  }
  .environment-tab.is-active {
    border-color: #bfdbfe;
    background: #fff;
    color: #2563eb;
    box-shadow: 0 1px 3px rgb(15 23 42 / 6%);
  }
  .environment-tab.environment-add {
    color: #16a34a;
  }
  .environment-binding-panel {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr)) auto;
    align-items: end;
    gap: 14px;
    padding: 16px;
    border: 1px solid #e2e8f0;
    background: #fff;
  }
  .binding-field {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .binding-field label,
  .environment-service-fields label {
    color: #94a3b8;
    font-size: 11px;
  }
  .binding-field strong {
    overflow: hidden;
    color: #334155;
    font-size: 13px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .binding-actions {
    display: flex;
    gap: 8px;
    white-space: nowrap;
  }
  .environment-service-list {
    padding: 16px;
    border: 1px solid #e2e8f0;
    background: #fff;
  }
  .section-heading {
    display: flex;
    justify-content: space-between;
    margin-bottom: 12px;
  }
  .section-heading div {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .section-heading strong {
    color: #1e293b;
    font-size: 14px;
  }
  .section-heading span {
    color: #94a3b8;
    font-size: 12px;
  }
  .environment-service-card {
    margin-top: 10px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
  }
  .environment-service-head {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 14px;
    border-bottom: 1px solid #e2e8f0;
    background: #fff;
  }
  .service-title {
    min-width: 150px;
    display: flex;
    align-items: center;
    gap: 7px;
  }
  .service-title strong {
    color: #1e293b;
  }
  .service-title small {
    color: #94a3b8;
  }
  .service-dot {
    display: inline-flex;
    color: #2563eb;
  }
  .resource-links {
    display: flex;
    flex: 1;
    gap: 6px;
  }
  .resource-pill {
    padding: 3px 7px;
    border: 1px solid #dbeafe;
    border-radius: 3px;
    color: #2563eb;
    background: #eff6ff;
    font-size: 11px;
  }
  .environment-service-fields {
    display: grid;
    grid-template-columns: 2fr repeat(3, 1fr);
    gap: 12px;
    padding: 14px;
  }

  .runtime-environment-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
    gap: 16px;
  }
  .runtime-environment-card {
    border: 1px solid #e5e7eb;
    border-radius: 10px;
    background: #fff;
    padding: 18px;
  }
  .runtime-card-head,
  .runtime-meta,
  .runtime-service-row,
  .runtime-actions {
    display: flex;
    align-items: center;
  }
  .runtime-card-head {
    justify-content: space-between;
    margin-bottom: 16px;
  }
  .runtime-card-head strong {
    display: block;
    font-size: 16px;
    color: #1f2937;
  }
  .runtime-card-head small,
  .runtime-service-row small {
    display: block;
    color: #94a3b8;
    margin-top: 3px;
  }
  .runtime-meta {
    justify-content: space-between;
    padding: 7px 0;
    border-bottom: 1px solid #f1f5f9;
    color: #64748b;
    font-size: 13px;
  }
  .runtime-meta strong {
    color: #334155;
    font-weight: 500;
  }
  .runtime-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    margin: 16px 0;
  }
  .runtime-stats div {
    background: #f8fafc;
    border-radius: 7px;
    padding: 10px;
    text-align: center;
  }
  .runtime-stats b {
    display: block;
    color: #1f2937;
    font-size: 18px;
  }
  .runtime-stats span {
    color: #64748b;
    font-size: 12px;
  }
  .runtime-service-list {
    border-top: 1px solid #e5e7eb;
    padding-top: 5px;
  }
  .runtime-service-row {
    gap: 9px;
    padding: 10px 0;
  }
  .runtime-service-row > div {
    flex: 1;
  }
  .runtime-service-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #22c55e;
  }
  .runtime-service-dot.pending {
    background: #94a3b8;
  }
  .runtime-actions {
    gap: 8px;
    border-top: 1px solid #e5e7eb;
    margin-top: 8px;
    padding-top: 14px;
  }
  .runtime-environment-card {
    cursor: pointer;
    transition:
      border-color 0.15s ease,
      box-shadow 0.15s ease;
  }
  .runtime-environment-card:hover {
    border-color: #99f6e4;
    box-shadow: 0 4px 14px rgb(15 118 110 / 8%);
  }
  .runtime-resource-panel {
    margin-top: 18px;
    border: 1px solid #e5e7eb;
    border-radius: 10px;
    background: #fff;
    overflow: hidden;
  }
  .runtime-resource-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 18px 12px;
  }
  .runtime-resource-head h3 {
    margin: 3px 0 0;
    color: #1f2937;
    font-size: 16px;
  }
  .runtime-resource-tabs {
    display: flex;
    gap: 4px;
    padding: 0 18px;
    border-bottom: 1px solid #e5e7eb;
  }
  .runtime-resource-tabs button {
    border: 0;
    border-bottom: 2px solid transparent;
    padding: 10px 12px;
    background: transparent;
    color: #64748b;
    cursor: pointer;
    font-size: 13px;
  }
  .runtime-resource-tabs button:hover,
  .runtime-resource-tabs button.active {
    color: #0f766e;
    border-bottom-color: #0f766e;
  }
  .runtime-resource-body {
    padding: 12px 18px 18px;
  }
  .resource-kind-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-bottom: 8px;
  }
  .resource-kind-tabs span {
    padding: 5px 10px;
    border: 1px solid #e2e8f0;
    border-radius: 5px;
    color: #64748b;
    background: #fff;
    cursor: pointer;
    font-size: 12px;
  }
  .resource-kind-tabs span.active {
    border-color: #99f6e4;
    color: #0f766e;
    background: #f0fdfa;
  }
  .resource-row {
    display: grid;
    grid-template-columns: minmax(180px, 1fr) auto auto auto;
    gap: 12px;
    align-items: center;
    min-height: 52px;
    border-bottom: 1px solid #f1f5f9;
  }
  .resource-row > div {
    min-width: 0;
  }
  .resource-row strong,
  .resource-row small {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .resource-row strong {
    color: #334155;
    font-size: 13px;
  }
  .resource-row small {
    margin-top: 3px;
    color: #94a3b8;
    font-size: 11px;
  }
  .resource-time {
    color: #64748b;
    font-size: 12px;
    white-space: nowrap;
  }
  .resource-empty-hint {
    padding: 28px 8px;
    color: #94a3b8;
    text-align: center;
    font-size: 13px;
  }
  @media (max-width: 760px) {
    .resource-row {
      grid-template-columns: 1fr auto;
      gap: 6px;
      padding: 8px 0;
    }
    .resource-row .el-button {
      justify-self: start;
    }
  }
  .environment-service-fields > div {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 5px;
  }
  .environment-service-fields span {
    overflow: hidden;
    color: #475569;
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .environment-service-fields .image-value {
    color: #0f766e;
    font-family: var(--ax-font-mono);
  }
  @media (max-width: 1000px) {
    .environment-binding-panel {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .binding-actions {
      grid-column: 1 / -1;
    }
  }
  @media (max-width: 700px) {
    .environment-service-head,
    .environment-service-fields {
      grid-template-columns: 1fr;
      flex-wrap: wrap;
    }
    .environment-service-fields {
      display: grid;
    }
    .resource-links {
      flex-basis: 100%;
    }
  }
  .env-card,
  .build-card,
  .workflow-card {
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    background: #fff;
  }
  .env-card-head {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 15px;
    border-bottom: 1px solid #f1f5f9;
  }
  .env-icon,
  .build-icon,
  .workflow-icon {
    width: 34px;
    height: 34px;
    display: grid;
    place-items: center;
    border-radius: 5px;
    background: #ecfdf5;
    color: #047857;
  }
  .env-card-head > div,
  .build-service > div,
  .workflow-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .env-card-head small,
  .build-service small,
  .workflow-info small {
    margin-top: 3px;
    color: #64748b;
  }
  .env-card dl {
    margin: 0;
    padding: 12px 15px;
  }
  .env-card dl > div {
    display: grid;
    grid-template-columns: 90px minmax(0, 1fr);
    padding: 5px 0;
    font-size: 12px;
  }
  .env-card dt {
    color: #64748b;
  }
  .env-card dd {
    margin: 0;
    overflow: hidden;
    color: #334155;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .card-actions {
    padding: 6px 10px;
    border-top: 1px solid #f1f5f9;
  }
  .build-list,
  .workflow-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .build-card,
  .workflow-card {
    display: grid;
    grid-template-columns: minmax(180px, 1fr) minmax(240px, 1.5fr) auto auto;
    align-items: center;
    gap: 18px;
    padding: 14px 16px;
  }
  .build-service {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .build-source,
  .workflow-target {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .build-readiness {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
    max-width: 260px;
  }
  .build-readiness small {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    overflow-wrap: anywhere;
  }
  .execute-branch-error {
    flex-basis: 100%;
    color: var(--el-color-warning);
    font-size: 12px;
  }
  .build-source span,
  .workflow-target span {
    color: #94a3b8;
    font-size: 10px;
  }
  .build-source strong {
    overflow: hidden;
    color: #334155;
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .build-source small,
  .workflow-target strong {
    color: #64748b;
    font-size: 11px;
  }
  .workflow-card {
    grid-template-columns: auto minmax(180px, 1fr) minmax(130px, 0.55fr) auto minmax(130px, auto);
    row-gap: 12px;
    cursor: pointer;
  }
  .workflow-card-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
  }
  .execute-dialog-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin: 8px 0 18px;
  }
  .execute-dialog {
    max-height: 69vh;
    overflow-y: auto;
    padding: 0 4px;
  }
  .execute-dialog-heading > div {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 14px;
  }
  .execute-dialog-heading strong {
    color: var(--el-text-color-primary);
    font-size: 18px;
  }
  .execute-dialog-heading small,
  .execute-target {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    overflow-wrap: anywhere;
  }
  .execute-service-picker {
    flex: 1;
    min-width: 0;
  }
  .execute-tools {
    display: flex;
    justify-content: flex-end;
    margin-top: 12px;
  }
  .execute-advanced {
    padding: 12px;
    background: var(--el-fill-color-light);
    border-radius: 6px;
  }
  .execute-dialog-heading span,
  .execute-section-title span,
  .execute-selected-hint {
    color: #94a3b8;
    font-size: 12px;
  }
  .execute-section {
    padding: 18px 0;
    border-bottom: 1px solid #eef2f7;
  }
  .execute-section:last-child {
    border-bottom: 0;
  }
  .execute-section-title {
    display: flex;
    align-items: baseline;
    gap: 10px;
    margin-bottom: 12px;
  }
  .execute-section-title strong {
    color: #1e293b;
    font-size: 14px;
  }
  .execute-service-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .execute-service-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    row-gap: 6px;
    min-height: 38px;
    padding: 18px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }
  .execute-service-name {
    flex: 1 1 160px;
    min-width: 160px;
  }
  .execute-order-actions {
    display: inline-flex;
    flex: 0 0 auto;
    gap: 2px;
    margin-left: auto;
  }
  .execute-branch-select {
    flex: 0 1 360px;
    width: 360px;
  }
  .execute-image-reference {
    flex-basis: 100%;
    min-width: 0;
    color: var(--el-text-color-secondary);
    font-size: 11px;
    line-height: 1.5;
    overflow-wrap: anywhere;
  }
  .execute-form-row {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-top: 18px;
  }
  .execute-form-row > label {
    flex: 0 0 80px;
    color: var(--el-text-color-regular);
    font-size: 13px;
  }
  .execute-selected-hint {
    margin: 10px 0 0;
    line-height: 1.6;
  }
  @media (max-width: 640px) {
    .execute-form-row {
      flex-wrap: wrap;
      gap: 10px;
    }
    .execute-form-row > label {
      flex-basis: 100%;
    }
    .execute-branch-select {
      flex-basis: 100%;
      width: 100%;
    }
  }
  .workflow-metrics {
    display: flex;
    gap: 18px;
    min-width: 0;
  }
  .workflow-metrics span {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 72px;
  }
  .workflow-metrics small {
    color: #94a3b8;
    font-size: 10px;
  }
  .workflow-metrics strong {
    color: #334155;
    font-size: 12px;
    font-weight: 600;
    white-space: nowrap;
  }
  .workflow-run-history {
    grid-column: 1 / -1;
    border-top: 1px solid #e2e8f0;
    padding-top: 12px;
  }
  .workflow-run-history-head {
    display: flex;
    justify-content: space-between;
    color: #334155;
    font-size: 12px;
    margin-bottom: 8px;
  }
  .workflow-run-history-head span {
    color: #94a3b8;
  }
  .workflow-run-table {
    display: flex;
    flex-direction: column;
    border: 1px solid #e2e8f0;
  }
  .workflow-run-row {
    display: grid;
    grid-template-columns: 70px 100px 1.5fr 1fr 90px;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 9px 12px;
    border: 0;
    border-bottom: 1px solid #f1f5f9;
    background: #fff;
    color: #475569;
    text-align: left;
    font-size: 12px;
  }
  button.workflow-run-row {
    cursor: pointer;
  }
  button.workflow-run-row:hover {
    background: #f8fafc;
  }
  .workflow-run-row:last-child {
    border-bottom: 0;
  }
  .workflow-run-header {
    background: #f8fafc;
    color: #94a3b8;
    font-size: 11px;
  }
  .workflow-run-link {
    color: #2563eb;
  }
  .job-detail-panel {
    padding: 4px 2px 24px;
  }
  .job-detail-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding-bottom: 20px;
    border-bottom: 1px solid #e2e8f0;
  }
  .job-detail-header h2 {
    margin: 5px 0 4px;
    color: #1e293b;
    font-size: 20px;
  }
  .job-detail-header p {
    margin: 0;
    color: #64748b;
    font-size: 13px;
  }
  .job-detail-metrics {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 10px;
    margin: 18px 0 26px;
  }
  .job-detail-metrics div {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 13px 14px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
  }
  .job-detail-metrics span {
    color: #94a3b8;
    font-size: 11px;
  }
  .job-detail-metrics strong {
    color: #1e293b;
    font-size: 15px;
    font-weight: 600;
  }
  .job-detail-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
    color: #334155;
  }
  .initial-run-box {
    margin-top: 20px;
  }
  .mini-metrics {
    display: flex;
    gap: 18px;
    margin: -2px 2px 0;
    color: #94a3b8;
    font-size: 12px;
  }
  .mini-metrics button {
    padding: 0;
    border: 0;
    background: transparent;
    color: #94a3b8;
    font-family: inherit;
    font-size: 12px;
    cursor: pointer;
  }
  .mini-metrics button:hover {
    color: #0f766e;
  }
  .mini-metrics b {
    margin-left: 3px;
    color: #475569;
    font-weight: 600;
  }
  .overview-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .overview-block {
    @include cicd-block;
  }
  .block-head {
    @include cicd-block-head;
  }
  .overview-list {
    display: flex;
    flex-direction: column;
    padding: 6px;
  }
  .workflow-table {
    display: flex;
    flex-direction: column;
  }
  .wf-row {
    display: grid;
    grid-template-columns: 180px minmax(120px, 180px) 1fr auto;
    align-items: center;
    gap: 14px;
    padding: 10px 16px;
    border-bottom: 1px solid #f1f5f9;
  }
  .wf-row:last-child {
    border-bottom: 0;
  }
  .wf-row:hover {
    background: #f8fafc;
  }
  .wf-name {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .wf-name .row-link {
    justify-content: flex-start;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .wf-name small {
    color: #94a3b8;
    font-size: 12px;
  }
  .wf-trend {
    display: flex;
    align-items: center;
    min-width: 120px;
  }
  .trend-empty {
    color: #cbd5e1;
    font-size: 13px;
  }
  .wf-metrics {
    display: flex;
    gap: 24px;
    justify-content: flex-end;
    min-width: 0;
  }
  .metric {
    @include cicd-metric;
  }
  .wf-action {
    display: flex;
    justify-content: flex-end;
    min-width: 84px;
  }
  .list-row {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 9px 10px;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: #334155;
    font-size: 13px;
    text-align: left;
  }
  .list-row.is-button {
    cursor: pointer;
  }
  .list-row.is-button:hover {
    background: #f8fafc;
  }
  .list-row strong {
    color: #1e293b;
    font-family: var(--ax-font-mono);
    font-size: 12px;
    white-space: nowrap;
  }
  .list-row small {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    color: #64748b;
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row-link {
    flex: 0 0 auto;
    padding: 0;
    border: 0;
    background: transparent;
    color: #2563eb;
    font-family: inherit;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
  }
  .row-link:hover {
    text-decoration: underline;
  }
  .row-name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .init-footer {
    position: fixed;
    right: 0;
    bottom: 0;
    left: var(--art-sidebar-width, 0);
    z-index: 20;
    min-height: 60px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 28px;
    border-top: 1px solid #dbe5e1;
    background: rgb(255 255 255 / 96%);
    box-shadow: 0 -4px 16px rgb(15 23 42 / 6%);
    backdrop-filter: blur(8px);
  }
  .init-footer > span {
    color: #64748b;
    font-size: 12px;
  }
  .init-footer > div {
    display: flex;
    gap: 8px;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  .switch-row {
    min-height: 58px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-top: 1px solid #f1f5f9;
  }
  .switch-row span {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .switch-row small {
    color: #64748b;
  }
  .project-load-result {
    min-height: 180px;
    padding: 48px 20px;
  }
  .project-load-result :deep(.el-result__icon) {
    display: none;
  }
  @media (max-width: 1100px) {
    .initialization-bar {
      grid-template-columns: 1fr;
    }
    .initialization-steps {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .build-card,
    .workflow-card {
      grid-template-columns: 1fr;
    }
    .initialization-steps,
    .form-grid {
      grid-template-columns: 1fr;
    }
  }
  @media (max-width: 760px) {
    .services-layout {
      height: auto;
      min-height: 0;
      grid-template-columns: 1fr;
      overflow: visible;
    }
    .service-catalog {
      max-height: 300px;
      border-right: 0;
      border-bottom: 1px solid #e2e8f0;
    }
    .initialization-steps,
    .form-grid {
      grid-template-columns: 1fr;
    }
  }
</style>

<template>
  <div class="update-management-container">
    <div class="config-layout">
      <!-- 左侧 Tabs -->
      <div class="tabs-sidebar">
        <ElMenu :default-active="activeTab" @select="handleTabChange" class="config-menu">
          <ElMenuItem index="image">
            <Box :size="16" />
            <span>更新镜像</span>
          </ElMenuItem>
          <ElMenuItem index="strategy">
            <Settings :size="16" />
            <span>更新策略</span>
          </ElMenuItem>
          <ElMenuItem index="history">
            <History :size="16" />
            <span>历史版本</span>
          </ElMenuItem>
        </ElMenu>
      </div>

      <!-- 右侧内容区 -->
      <div class="config-content">
        <!-- 更新镜像 -->
        <div v-show="activeTab === 'image'" v-loading="loadingContainers" class="config-section">
          <!-- 待更新提示 -->
          <div v-if="hasPendingChanges" class="pending-changes-banner">
            <div class="banner-content">
              <AlertCircle :size="18" />
              <span
                >有 <strong>{{ pendingChangesCount }}</strong> 个容器镜像待更新</span
              >
            </div>
            <div class="banner-actions">
              <ElButton size="small" @click="clearAllPendingChanges">
                <X :size="14" />
                清除所有
              </ElButton>
              <ElButton size="small" type="primary" @click="handleBatchUpdate">
                <Upload :size="14" />
                立即更新
              </ElButton>
            </div>
          </div>

          <!-- Init 容器区域 -->
          <div class="container-section">
            <div class="section-header">
              <div class="header-left">
                <PlayCircle :size="16" />
                <span>Init 容器</span>
                <ElTag size="small" type="info">{{ initContainers.length }}</ElTag>
              </div>
            </div>

            <div v-if="initContainers.length === 0" class="empty-container-tip">
              <Inbox :size="32" />
              <span>无 Init 容器</span>
            </div>

            <div v-else class="container-list">
              <div
                v-for="container in initContainers"
                :key="container.containerName"
                class="container-card"
                :class="{ 'has-pending': hasPendingChange(container.containerName) }"
              >
                <div class="container-info">
                  <div class="container-name">
                    <Box :size="14" />
                    <span>{{ container.containerName }}</span>
                  </div>
                  <div class="container-image">
                    <code class="current-image">{{ container.image }}</code>
                    <template v-if="hasPendingChange(container.containerName)">
                      <ArrowRight :size="14" class="arrow-icon" />
                      <code class="new-image">{{ getPendingImage(container.containerName) }}</code>
                    </template>
                  </div>
                </div>
                <div class="container-actions">
                  <ElButton
                    v-if="hasPendingChange(container.containerName)"
                    size="small"
                    text
                    type="danger"
                    @click="clearPendingChange(container.containerName)"
                  >
                    <X :size="14" />
                    撤销
                  </ElButton>
                  <ElButton size="small" type="primary" @click="openUpdateDialog(container)">
                    <RefreshCw :size="14" />
                    {{ hasPendingChange(container.containerName) ? '修改' : '更新' }}
                  </ElButton>
                </div>
              </div>
            </div>
          </div>

          <!-- Main 容器区域 -->
          <div class="container-section">
            <div class="section-header">
              <div class="header-left">
                <Container :size="16" />
                <span>主容器</span>
                <ElTag size="small" type="success">{{ mainContainers.length }}</ElTag>
              </div>
            </div>

            <div v-if="mainContainers.length === 0" class="empty-container-tip">
              <Inbox :size="32" />
              <span>无主容器</span>
            </div>

            <div v-else class="container-list">
              <div
                v-for="container in mainContainers"
                :key="container.containerName"
                class="container-card"
                :class="{ 'has-pending': hasPendingChange(container.containerName) }"
              >
                <div class="container-info">
                  <div class="container-name">
                    <Box :size="14" />
                    <span>{{ container.containerName }}</span>
                  </div>
                  <div class="container-image">
                    <code class="current-image">{{ container.image }}</code>
                    <template v-if="hasPendingChange(container.containerName)">
                      <ArrowRight :size="14" class="arrow-icon" />
                      <code class="new-image">{{ getPendingImage(container.containerName) }}</code>
                    </template>
                  </div>
                </div>
                <div class="container-actions">
                  <ElButton
                    v-if="hasPendingChange(container.containerName)"
                    size="small"
                    text
                    type="danger"
                    @click="clearPendingChange(container.containerName)"
                  >
                    <X :size="14" />
                    撤销
                  </ElButton>
                  <ElButton size="small" type="primary" @click="openUpdateDialog(container)">
                    <RefreshCw :size="14" />
                    {{ hasPendingChange(container.containerName) ? '修改' : '更新' }}
                  </ElButton>
                </div>
              </div>
            </div>
          </div>

          <!-- Ephemeral 容器区域（如果有） -->
          <div v-if="ephemeralContainers.length > 0" class="container-section">
            <div class="section-header">
              <div class="header-left">
                <Zap :size="16" />
                <span>临时容器</span>
                <ElTag size="small" type="warning">{{ ephemeralContainers.length }}</ElTag>
              </div>
            </div>

            <div class="container-list">
              <div
                v-for="container in ephemeralContainers"
                :key="container.containerName"
                class="container-card"
                :class="{ 'has-pending': hasPendingChange(container.containerName) }"
              >
                <div class="container-info">
                  <div class="container-name">
                    <Box :size="14" />
                    <span>{{ container.containerName }}</span>
                  </div>
                  <div class="container-image">
                    <code class="current-image">{{ container.image }}</code>
                    <template v-if="hasPendingChange(container.containerName)">
                      <ArrowRight :size="14" class="arrow-icon" />
                      <code class="new-image">{{ getPendingImage(container.containerName) }}</code>
                    </template>
                  </div>
                </div>
                <div class="container-actions">
                  <ElButton
                    v-if="hasPendingChange(container.containerName)"
                    size="small"
                    text
                    type="danger"
                    @click="clearPendingChange(container.containerName)"
                  >
                    <X :size="14" />
                    撤销
                  </ElButton>
                  <ElButton size="small" type="primary" @click="openUpdateDialog(container)">
                    <RefreshCw :size="14" />
                    {{ hasPendingChange(container.containerName) ? '修改' : '更新' }}
                  </ElButton>
                </div>
              </div>
            </div>
          </div>

          <!-- 底部操作区 -->
          <div class="bottom-actions" v-if="allContainers.length > 0">
            <ElButton @click="loadContainers" :loading="loadingContainers">
              <RefreshCw :size="14" />
              刷新列表
            </ElButton>
            <ElButton
              type="primary"
              :disabled="!hasPendingChanges"
              :loading="updatingImages"
              @click="handleBatchUpdate"
            >
              <Upload :size="14" />
              批量更新 {{ hasPendingChanges ? `(${pendingChangesCount})` : '' }}
            </ElButton>
          </div>
        </div>

        <!-- 更新策略 -->
        <div v-show="activeTab === 'strategy'" v-loading="loadingStrategy" class="config-section">
          <ElForm :model="strategyForm" label-width="140px" label-position="left">
            <!-- 策略类型选择 -->
            <div class="section-header">
              <Settings :size="16" />
              <span>策略类型</span>
            </div>

            <ElFormItem label="更新策略" required>
              <template #label>
                <span>更新策略</span>
                <ElTooltip :content="getStrategyTypeTooltip()" placement="top">
                  <HelpCircle :size="14" class="label-help-icon" />
                </ElTooltip>
              </template>
              <ElRadioGroup v-model="strategyForm.type" @change="handleStrategyTypeChange">
                <ElRadio
                  v-for="strategy in getAvailableStrategies()"
                  :key="strategy.value"
                  :value="strategy.value"
                >
                  {{ strategy.label }}
                  <ElTooltip :content="strategy.description" placement="top">
                    <HelpCircle :size="12" class="inline-help-icon" />
                  </ElTooltip>
                </ElRadio>
              </ElRadioGroup>
            </ElFormItem>

            <!-- RollingUpdate 配置 -->
            <template v-if="strategyForm.type === 'RollingUpdate'">
              <div class="section-header">
                <Settings :size="16" />
                <span>滚动更新配置</span>
              </div>

              <!-- DaemonSet 特殊警告提示 -->
              <ElAlert
                v-if="isDaemonSet()"
                type="warning"
                :closable="false"
                show-icon
                style="margin-bottom: 20px"
              >
                <template #title>
                  <strong>DaemonSet 更新策略限制</strong>
                </template>
                <div style="line-height: 1.6; font-size: 13px">
                  <div style="margin-bottom: 8px">
                    <strong style="color: #e6a23c">⚠️ 重要：</strong>
                    <span style="color: #606266">
                      <code
                        style="
                          background: #fff;
                          padding: 2px 6px;
                          border-radius: 3px;
                          color: #e6a23c;
                        "
                        >maxSurge</code
                      >
                      和
                      <code
                        style="
                          background: #fff;
                          padding: 2px 6px;
                          border-radius: 3px;
                          color: #e6a23c;
                        "
                        >maxUnavailable</code
                      >
                      不能同时大于 0
                    </span>
                  </div>
                  <div style="color: #909399; font-size: 12px">
                    • 使用 <code>maxSurge > 0</code> 时，<code>maxUnavailable</code> 必须为
                    0（先创建新 Pod 再删除旧 Pod）
                    <br />
                    • 使用 <code>maxUnavailable > 0</code> 时，<code>maxSurge</code> 必须为
                    0（先删除旧 Pod 再创建新 Pod）
                  </div>
                </div>
              </ElAlert>

              <!-- maxSurge - Deployment 和 DaemonSet -->
              <ElFormItem
                v-if="supportedFields.maxSurge"
                label="最大额外副本"
                :error="getMaxSurgeError()"
              >
                <template #label>
                  <span>最大额外副本</span>
                  <ElTooltip
                    :content="
                      isDaemonSet()
                        ? '更新期间可创建的额外 Pod 数（K8s 1.22+ 支持）。设置 > 0 时，最大不可用副本会自动设为 0'
                        : '更新期间可创建的额外 Pod 数，默认为 25%'
                    "
                    placement="top"
                  >
                    <HelpCircle :size="14" class="label-help-icon" />
                  </ElTooltip>
                </template>
                <ElInput
                  v-model="strategyForm.rollingUpdate.maxSurge"
                  :placeholder="isDaemonSet() ? '如: 0 或 10%' : '如: 1 或 25%'"
                  style="max-width: 200px"
                  @input="handleMaxSurgeChange"
                  :class="{ 'has-conflict': isDaemonSet() && hasConflict() }"
                >
                  <template #append>Pod</template>
                </ElInput>
                <div
                  class="form-tip"
                  :class="{ 'tip-warning': isDaemonSet() && isMaxSurgeNonZero() }"
                >
                  <template v-if="isDaemonSet()">
                    <span v-if="isMaxSurgeNonZero()" style="color: #e6a23c">
                      ⚠️ 已设置 > 0，最大不可用副本将自动设为 0
                    </span>
                    <span v-else> 可以是数字或百分比，默认为 0 </span>
                  </template>
                  <template v-else> 可以是数字或百分比，默认为 25% </template>
                </div>
              </ElFormItem>

              <!-- maxUnavailable - 所有类型 -->
              <ElFormItem
                v-if="supportedFields.maxUnavailable"
                label="最大不可用副本"
                :error="getMaxUnavailableError()"
              >
                <template #label>
                  <span>最大不可用副本</span>
                  <ElTooltip :content="getMaxUnavailableTooltip()" placement="top">
                    <HelpCircle :size="14" class="label-help-icon" />
                  </ElTooltip>
                </template>
                <ElInput
                  v-model="strategyForm.rollingUpdate.maxUnavailable"
                  :placeholder="getMaxUnavailablePlaceholder()"
                  style="max-width: 200px"
                  @input="handleMaxUnavailableChange"
                  :class="{ 'has-conflict': isDaemonSet() && hasConflict() }"
                >
                  <template #append>Pod</template>
                </ElInput>
                <div
                  class="form-tip"
                  :class="{ 'tip-warning': isDaemonSet() && isMaxUnavailableNonZero() }"
                >
                  <template v-if="isDaemonSet()">
                    <span v-if="isMaxUnavailableNonZero()" style="color: #e6a23c">
                      ⚠️ 已设置 > 0，最大额外副本将自动设为 0
                    </span>
                    <span v-else> 可以是数字或百分比，默认为 1 </span>
                  </template>
                  <template v-else>
                    {{ getMaxUnavailableTip() }}
                  </template>
                </div>
              </ElFormItem>

              <!-- partition - StatefulSet 专用 -->
              <ElFormItem v-if="supportedFields.partition" label="分区">
                <template #label>
                  <span>分区</span>
                  <ElTooltip
                    content="只更新序号 >= partition 的 Pod，用于金丝雀发布。设置为 0 表示更新所有 Pod"
                    placement="top"
                  >
                    <HelpCircle :size="14" class="label-help-icon" />
                  </ElTooltip>
                </template>
                <ElInputNumber
                  v-model="strategyForm.rollingUpdate.partition"
                  :min="0"
                  :max="1000"
                  controls-position="right"
                  style="max-width: 200px"
                />
                <div class="form-tip">默认为 0（更新所有 Pod）。设置 > 0 可实现分阶段更新</div>
              </ElFormItem>

              <!-- 配置冲突提示 -->
              <ElAlert
                v-if="isDaemonSet() && hasConflict()"
                type="error"
                :closable="false"
                show-icon
                style="margin-top: 16px"
              >
                <template #title>
                  <strong>配置冲突</strong>
                </template>
                <div style="font-size: 13px">
                  检测到 <code>maxSurge</code> 和 <code>maxUnavailable</code>
                  同时大于 0，这是不允许的。
                  <br />
                  请将其中一个设置为 0，或点击下方按钮自动修复。
                </div>
                <div style="margin-top: 12px; display: flex; gap: 8px">
                  <ElButton size="small" type="warning" @click="autoFixConflict('surge')">
                    保留 maxSurge，将 maxUnavailable 设为 0
                  </ElButton>
                  <ElButton size="small" type="warning" @click="autoFixConflict('unavailable')">
                    保留 maxUnavailable，将 maxSurge 设为 0
                  </ElButton>
                </div>
              </ElAlert>
            </template>

            <!-- 策略说明提示 -->
            <template v-if="strategyForm.type !== 'RollingUpdate'">
              <!-- OnDelete 提示 -->
              <ElAlert
                v-if="(isStatefulSet() || isDaemonSet()) && strategyForm.type === 'OnDelete'"
                type="info"
                :closable="false"
                style="margin-top: 16px"
              >
                <template #title>
                  <div style="display: flex; align-items: center; gap: 8px">
                    <Info :size="16" />
                    <span>OnDelete 策略说明</span>
                  </div>
                </template>
                <div style="margin-top: 8px">
                  使用 OnDelete 策略时，Pod 不会自动更新。您需要手动删除
                  Pod，系统才会创建使用新配置的 Pod。
                  <br />
                  这种策略适用于需要精确控制更新时机的场景。
                </div>
              </ElAlert>

              <!-- Recreate 提示 -->
              <ElAlert
                v-if="isDeployment() && strategyForm.type === 'Recreate'"
                type="warning"
                :closable="false"
                style="margin-top: 16px"
              >
                <template #title>
                  <div style="display: flex; align-items: center; gap: 8px">
                    <AlertTriangle :size="16" />
                    <span>Recreate 策略警告</span>
                  </div>
                </template>
                <div style="margin-top: 8px">
                  使用 Recreate 策略时，会先删除所有旧 Pod，然后再创建新 Pod。
                  <br />
                  <strong>这会导致服务短暂不可用</strong>，请谨慎使用。
                </div>
              </ElAlert>
            </template>

            <!-- 操作按钮 -->
            <div class="form-actions">
              <ElTooltip
                :content="getSubmitButtonTooltip()"
                placement="top"
                :disabled="isStrategyFormValid()"
              >
                <span>
                  <ElButton
                    type="primary"
                    @click="handleUpdateStrategy"
                    :loading="updatingStrategy"
                    :disabled="!isStrategyFormValid()"
                  >
                    <Save :size="16" />
                    保存策略
                  </ElButton>
                </span>
              </ElTooltip>
              <ElButton @click="loadStrategy">
                <RotateCcw :size="16" />
                重置
              </ElButton>
            </div>
          </ElForm>
        </div>

        <!-- 历史版本 -->
        <div v-show="activeTab === 'history'" class="config-section">
          <div class="section-header">
            <History :size="16" />
            <span>版本列表</span>
          </div>

          <ElTable :data="revisions" v-loading="loadingRevisions" stripe class="history-table">
            <ElTableColumn prop="revision" label="版本号" width="100" align="center" />
            <ElTableColumn prop="images" label="镜像" min-width="250">
              <template #default="{ row }">
                <div class="image-list">
                  <div v-for="(image, index) in row.images" :key="index" class="image-item">
                    {{ image }}
                  </div>
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="replicas" label="副本数" width="100" align="center" />
            <ElTableColumn prop="reason" label="原因" min-width="150">
              <template #default="{ row }">
                <span>{{ row.reason || '-' }}</span>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="creationTimestamp" label="创建时间" width="160" align="center">
              <template #default="{ row }">{{ formatTime(row.creationTimestamp) }}</template>
            </ElTableColumn>
            <ElTableColumn label="操作" width="100" align="center" fixed="right">
              <template #default="{ row }">
                <ElButton
                  size="small"
                  type="primary"
                  :loading="rollingBack && rollbackTarget === row.revision"
                  :disabled="rollingBack"
                  @click="handleRollback(row)"
                >
                  回滚
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
        </div>
      </div>
    </div>

    <!-- 更新镜像弹窗 -->
    <ElDialog
      v-model="updateDialogVisible"
      :title="`更新镜像 - ${currentEditContainer?.containerName || ''}`"
      width="680px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <div class="update-dialog-content">
        <!-- 当前镜像信息 -->
        <div class="current-image-info">
          <span class="label">当前镜像：</span>
          <code>{{ currentEditContainer?.image || '-' }}</code>
        </div>

        <!-- 输入方式选择 -->
        <div class="input-mode-selector">
          <ElRadioGroup v-model="inputMode" size="default">
            <ElRadioButton value="manual">
              <Edit3 :size="14" />
              手动输入
            </ElRadioButton>
            <ElRadioButton value="search">
              <Search :size="14" />
              从仓库选择
            </ElRadioButton>
          </ElRadioGroup>
        </div>

        <!-- 手动输入模式 -->
        <div v-if="inputMode === 'manual'" class="manual-input-section">
          <ElForm label-width="100px" label-position="left">
            <ElFormItem label="新镜像地址" required>
              <ElInput
                v-model="manualImageInput"
                placeholder="请输入完整镜像地址，如: nginx:latest 或 registry.example.com/project/image:tag"
                clearable
              >
                <template #prefix>
                  <Package :size="16" />
                </template>
              </ElInput>
              <div class="form-tip">
                <Info :size="12" />
                <span>输入完整的镜像地址，包含仓库地址、项目、镜像名和标签</span>
              </div>
            </ElFormItem>
          </ElForm>
        </div>

        <!-- 仓库搜索模式 -->
        <div v-else class="search-input-section">
          <ElForm label-width="100px" label-position="left">
            <!-- 镜像搜索 -->
            <ElFormItem label="镜像搜索">
              <ElInput
                v-model="imageSearch"
                placeholder="搜索镜像名称，如: nginx, redis..."
                clearable
                @keyup.enter="handleImageSearchClick"
              >
                <template #prefix>
                  <Search :size="16" />
                </template>
                <template #append>
                  <ElButton @click="handleImageSearchClick" :loading="searchingImages">
                    搜索
                  </ElButton>
                </template>
              </ElInput>
              <div v-if="searchApplied" class="search-hint">
                <Info :size="12" />
                <span>已应用搜索过滤，找到 {{ getTotalFilteredCount() }} 个结果</span>
                <ElButton text type="primary" size="small" @click="clearSearch">
                  清除过滤
                </ElButton>
              </div>
            </ElFormItem>

            <!-- 镜像仓库 -->
            <ElFormItem label="镜像仓库" required>
              <ElSelect
                v-model="imageConfig.registryUuid"
                placeholder="选择仓库"
                filterable
                :loading="loadingRegistries"
                @change="handleRegistryChange"
              >
                <ElOption
                  v-for="registry in displayRegistries"
                  :key="registry.registryUuid"
                  :label="registry.registryName"
                  :value="registry.registryUuid"
                >
                  <div class="registry-option">
                    <span>{{ registry.registryName }}</span>
                    <ElTag size="small" type="info">{{ registry.registryType }}</ElTag>
                  </div>
                </ElOption>
                <template #footer>
                  <div
                    v-if="registryPagination.hasMore"
                    v-loading="loadingRegistries"
                    class="select-loading-more"
                    @click="loadMoreRegistries"
                  >
                    <span v-if="!loadingRegistries">点击加载更多</span>
                  </div>
                </template>
              </ElSelect>
            </ElFormItem>

            <!-- 项目/命名空间 -->
            <ElFormItem label="项目">
              <ElSelect
                v-model="imageConfig.project"
                placeholder="选择项目"
                filterable
                :loading="loadingProjects"
                :disabled="!imageConfig.registryUuid"
                @change="handleProjectChange"
              >
                <ElOption
                  v-for="project in displayProjects"
                  :key="project.name"
                  :label="project.name"
                  :value="project.name"
                />
                <template #footer>
                  <div
                    v-if="projectPagination.hasMore"
                    v-loading="loadingProjects"
                    class="select-loading-more"
                    @click="loadMoreProjects"
                  >
                    <span v-if="!loadingProjects">点击加载更多</span>
                  </div>
                </template>
              </ElSelect>
            </ElFormItem>

            <!-- 镜像名称 -->
            <ElFormItem label="镜像名称">
              <ElSelect
                v-model="imageConfig.image"
                placeholder="选择镜像"
                filterable
                :loading="loadingImages"
                :disabled="!imageConfig.project"
                @change="handleImageChange"
              >
                <ElOption
                  v-for="img in displayImages"
                  :key="img.name"
                  :label="img.displayName"
                  :value="img.name"
                />
                <template #footer>
                  <div
                    v-if="imagePagination.hasMore"
                    v-loading="loadingImages"
                    class="select-loading-more"
                    @click="loadMoreImages"
                  >
                    <span v-if="!loadingImages">点击加载更多</span>
                  </div>
                </template>
              </ElSelect>
            </ElFormItem>

            <!-- 版本标签 -->
            <ElFormItem label="版本标签">
              <ElSelect
                v-model="imageConfig.tag"
                placeholder="选择标签"
                filterable
                :loading="loadingTags"
                :disabled="!imageConfig.image"
                @change="handleTagChange"
              >
                <ElOption
                  v-for="tag in displayTags"
                  :key="tag.name"
                  :label="tag.name"
                  :value="tag.name"
                >
                  <div class="tag-option">
                    <span>{{ tag.name }} - </span>
                    <span class="tag-size"> {{ tag.size }}</span>
                  </div>
                </ElOption>
                <template #footer>
                  <div
                    v-if="tagPagination.hasMore"
                    v-loading="loadingTags"
                    class="select-loading-more"
                    @click="loadMoreTags"
                  >
                    <span v-if="!loadingTags">点击加载更多</span>
                  </div>
                </template>
              </ElSelect>
            </ElFormItem>
          </ElForm>
        </div>

        <!-- 新镜像预览 -->
        <div v-if="dialogNewImage" class="new-image-preview">
          <div class="preview-label">
            <Check :size="16" />
            <span>新镜像地址：</span>
          </div>
          <div class="preview-value">
            <code>{{ dialogNewImage }}</code>
            <ElButton text type="primary" size="small" @click="copyDialogImageUrl">
              <Copy :size="14" />
              复制
            </ElButton>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="closeUpdateDialog">取消</ElButton>
          <ElButton type="primary" :disabled="!dialogNewImage" @click="confirmUpdateImage">
            确定
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <!-- 批量更新确认弹窗 -->
    <ElDialog
      v-model="batchConfirmDialogVisible"
      title="确认批量更新"
      width="600px"
      :close-on-click-modal="false"
    >
      <div class="batch-confirm-content">
        <ElAlert type="warning" :closable="false" show-icon style="margin-bottom: 16px">
          <template #title>即将更新 {{ pendingChangesCount }} 个容器的镜像</template>
        </ElAlert>

        <div class="changes-list">
          <div
            v-for="(newImage, containerName) in pendingChanges"
            :key="containerName"
            class="change-item"
          >
            <div class="change-container">
              <Box :size="14" />
              <span>{{ containerName }}</span>
            </div>
            <div class="change-images">
              <code class="old">{{ getOriginalImage(containerName as string) }}</code>
              <ArrowRight :size="14" />
              <code class="new">{{ newImage }}</code>
            </div>
          </div>
        </div>

        <!-- 更新原因 -->
        <div class="update-reason-section">
          <ElForm label-width="80px">
            <ElFormItem label="更新原因">
              <ElInput
                v-model="batchUpdateReason"
                type="textarea"
                placeholder="请输入更新原因，如：修复安全漏洞、功能升级、性能优化等（可选）"
                :rows="2"
                :maxlength="500"
                show-word-limit
                resize="none"
              />
            </ElFormItem>
          </ElForm>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="batchConfirmDialogVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="updatingImages" @click="executeBatchUpdate">
            确认更新
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    Box,
    Copy,
    Search,
    Info,
    Check,
    RotateCcw,
    Settings,
    History,
    HelpCircle,
    Package,
    Save,
    AlertTriangle,
    RefreshCw,
    Upload,
    X,
    ArrowRight,
    PlayCircle,
    Container,
    Inbox,
    Zap,
    Edit3,
    AlertCircle
  } from 'lucide-vue-next'
  import {
    getResourceImagesApi,
    updateImagesApi,
    getUpdateStrategyApi,
    updateUpdateStrategyApi,
    getResourceHistoryApi,
    rollbackToRevisionApi,
    listClusterRegistriesApi,
    listProjectsByAppApi,
    listRepositoriesApi,
    listArtifactsApi,
    searchImagesGlobalByProjectApi,
    type ContainerImageInfo,
    type UpdateImagesRequest,
    type UpdateStrategyRequest,
    type RevisionInfo,
    type OnecProjectApplication,
    type OnecProjectVersion,
    type ProjectCluster,
    type ProjectWorkspace
  } from '@/api'
  import dayjs from 'dayjs'

  // ========== 类型定义 ==========
  interface ImageItem {
    name: string
    displayName: string
    originalName: string
    artifactCount: number
    pullCount: number
  }

  interface TagItem {
    name: string
    size: string
    pushTime: number
    artifactId: number
    digest: string
  }

  interface RegistryItem {
    registryId: number
    registryUuid: string
    registryName: string
    registryUrl: string
    registryType: string
  }

  interface ProjectItem {
    name: string
    projectId: number
  }

  interface SearchResultRegistry {
    registryUuid: string
    registryUrl?: string
    images?: Array<{
      projectName: string
      repoName: string
      tags?: string[]
    }>
  }

  interface SearchResult {
    data: SearchResultRegistry[]
    total?: number
  }

  // ========== Props ==========
  interface Props {
    version: OnecProjectVersion
    application: OnecProjectApplication
    cluster: ProjectCluster | null
    workspace: ProjectWorkspace | null
    refreshTrigger?: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ success: [] }>()

  const projectId = computed(() => props.cluster?.projectId)
  const clusterUuid = computed(
    () => props.cluster?.clusterUuid || props.cluster?.uuid || `cluster-${props.cluster?.id}`
  )

  const activeTab = ref('image')

  // ==================== 资源类型判断 ====================
  const getResourceType = (): string => {
    return (props.application?.resourceType || '').toUpperCase()
  }

  const isDeployment = (): boolean => {
    return getResourceType() === 'DEPLOYMENT'
  }

  const isStatefulSet = (): boolean => {
    return getResourceType() === 'STATEFULSET'
  }

  const isDaemonSet = (): boolean => {
    return getResourceType() === 'DAEMONSET'
  }

  const getAvailableStrategies = () => {
    const resourceType = getResourceType()

    switch (resourceType) {
      case 'DEPLOYMENT':
        return [
          {
            value: 'RollingUpdate',
            label: '滚动更新',
            description: '逐步更新 Pod，确保服务持续可用（推荐）'
          },
          {
            value: 'Recreate',
            label: '重建',
            description: '先删除所有旧 Pod，再创建新 Pod（会导致服务中断）'
          }
        ]

      case 'STATEFULSET':
        return [
          {
            value: 'RollingUpdate',
            label: '滚动更新',
            description: '按 Pod 序号逆序更新，支持分区更新（推荐）'
          },
          {
            value: 'OnDelete',
            label: '手动删除',
            description: '需要手动删除 Pod 才会更新'
          }
        ]

      case 'DAEMONSET':
        return [
          {
            value: 'RollingUpdate',
            label: '滚动更新',
            description: '逐节点更新 Pod（推荐）'
          },
          {
            value: 'OnDelete',
            label: '手动删除',
            description: '需要手动删除 Pod 才会更新'
          }
        ]

      default:
        return []
    }
  }

  const getStrategyTypeTooltip = (): string => {
    const resourceType = getResourceType()

    switch (resourceType) {
      case 'DEPLOYMENT':
        return '选择 Deployment 的更新策略：滚动更新（推荐）或重建'
      case 'STATEFULSET':
        return '选择 StatefulSet 的更新策略：滚动更新（推荐）或手动删除'
      case 'DAEMONSET':
        return '选择 DaemonSet 的更新策略：滚动更新（推荐）或手动删除'
      default:
        return '选择资源的更新策略'
    }
  }

  // ==================== 更新镜像相关 ====================
  const loadingContainers = ref(false)
  const updatingImages = ref(false)
  const allContainers = ref<ContainerImageInfo[]>([])

  // 分类容器
  const initContainers = computed(() =>
    allContainers.value.filter((c) => c.containerType === 'init')
  )
  const mainContainers = computed(() =>
    allContainers.value.filter((c) => c.containerType === 'main' || c.containerType === 'container')
  )
  const ephemeralContainers = computed(() =>
    allContainers.value.filter((c) => c.containerType === 'ephemeral')
  )

  // 使用 ref 替代 reactive 来管理 pendingChanges，确保响应式更新
  const pendingChanges = ref<Record<string, string>>({})

  const hasPendingChanges = computed(() => Object.keys(pendingChanges.value).length > 0)
  const pendingChangesCount = computed(() => Object.keys(pendingChanges.value).length)

  const hasPendingChange = (containerName: string) => containerName in pendingChanges.value
  const getPendingImage = (containerName: string) => pendingChanges.value[containerName] || ''

  const getOriginalImage = (containerName: string): string => {
    const container = allContainers.value.find((c) => c.containerName === containerName)
    return container?.image || '-'
  }

  // 使用正确的方式清除响应式对象中的属性
  const clearPendingChange = (containerName: string) => {
    const newChanges = { ...pendingChanges.value }
    delete newChanges[containerName]
    pendingChanges.value = newChanges
  }

  const clearAllPendingChanges = () => {
    pendingChanges.value = {}
  }

  // 添加待更新变更
  const addPendingChange = (containerName: string, newImage: string) => {
    pendingChanges.value = {
      ...pendingChanges.value,
      [containerName]: newImage
    }
  }

  // 更新弹窗相关
  const updateDialogVisible = ref(false)
  const currentEditContainer = ref<ContainerImageInfo | null>(null)
  const inputMode = ref<'manual' | 'search'>('manual')
  const manualImageInput = ref('')

  // 批量更新确认弹窗
  const batchConfirmDialogVisible = ref(false)
  const batchUpdateReason = ref('')

  // 镜像搜索相关
  const imageSearch = ref('')
  const searchingImages = ref(false)
  const searchApplied = ref(false)
  const searchResults = ref<SearchResult | null>(null)
  const searchPagination = ref({ page: 1, pageSize: 50, total: 0, hasMore: false })

  const imageConfig = ref({
    registryUuid: '',
    registryUrl: '',
    project: '',
    image: '',
    tag: ''
  })

  const loadingRegistries = ref(false)
  const loadingProjects = ref(false)
  const loadingImages = ref(false)
  const loadingTags = ref(false)

  const registryPagination = ref({ page: 1, pageSize: 20, total: 0, hasMore: false })
  const projectPagination = ref({ page: 1, pageSize: 20, total: 0, hasMore: false })
  const imagePagination = ref({ page: 1, pageSize: 20, total: 0, hasMore: false })
  const tagPagination = ref({ page: 1, pageSize: 20, total: 0, hasMore: false })

  const registries = ref<RegistryItem[]>([])
  const projects = ref<ProjectItem[]>([])
  const images = ref<ImageItem[]>([])
  const tags = ref<TagItem[]>([])

  // 计算弹窗中的新镜像地址
  const dialogNewImage = computed(() => {
    if (inputMode.value === 'manual') {
      return manualImageInput.value.trim()
    } else {
      return fullImageUrl.value
    }
  })

  const fullImageUrl = computed(() => {
    const { registryUrl, project, image, tag } = imageConfig.value
    if (!image) return ''

    let url = ''
    if (registryUrl && registryUrl !== 'docker.io') {
      url = registryUrl + '/'
    }
    if (project) {
      url += project + '/'
    }
    url += image
    if (tag) {
      url += ':' + tag
    }
    return url
  })

  function removeProjectPrefix(repoName: string, projectName: string): string {
    if (!repoName || !projectName) return repoName || ''
    const projectPrefix = `${projectName}/`
    if (repoName.startsWith(projectPrefix)) {
      return repoName.substring(projectPrefix.length)
    }
    return repoName
  }

  const displayRegistries = computed(() => {
    if (
      !searchApplied.value ||
      !searchResults.value?.data ||
      !Array.isArray(searchResults.value.data)
    ) {
      return registries.value
    }
    const searchRegistryUuids = new Set(
      searchResults.value.data.map((r) => r?.registryUuid).filter(Boolean)
    )
    return registries.value.filter((r) => searchRegistryUuids.has(r.registryUuid))
  })

  const displayProjects = computed(() => {
    if (!searchApplied.value || !searchResults.value?.data || !imageConfig.value.registryUuid) {
      return projects.value
    }

    const registryResult = searchResults.value.data.find(
      (r) => r?.registryUuid === imageConfig.value.registryUuid
    )

    if (!registryResult?.images || !Array.isArray(registryResult.images)) {
      return []
    }

    const searchProjectNames = new Set(
      registryResult.images.map((img) => img?.projectName).filter(Boolean)
    )
    return projects.value.filter((p) => searchProjectNames.has(p.name))
  })

  const displayImages = computed(() => {
    return images.value
  })

  const displayTags = computed(() => {
    return tags.value
  })

  function formatSize(bytes: number): string {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
  }

  function cleanRegistryUrl(url: string): string {
    if (!url) return url
    return url.replace(/^https?:\/\//, '')
  }

  /**
   * 加载容器镜像列表
   * 添加防重复加载机制：如果正在加载中则直接返回
   */
  const loadContainers = async () => {
    // 防止重复加载
    if (loadingContainers.value) return

    loadingContainers.value = true
    try {
      const response = await getResourceImagesApi(props.version.id)
      allContainers.value = response.containers || []
    } catch (error) {
      console.error('加载容器镜像列表失败:', error)
      allContainers.value = []
    } finally {
      loadingContainers.value = false
    }
  }

  async function fetchRegistries(page: number = 1, append: boolean = false) {
    if (!clusterUuid.value) return

    loadingRegistries.value = true
    try {
      const res = await listClusterRegistriesApi({ clusterUuid: clusterUuid.value })

      if (!res?.data) {
        registries.value = []
        registryPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }
        return
      }

      const newData = res.data
        .filter((item) => item?.registry?.uuid)
        .map((item) => ({
          registryId: item.registryId,
          registryUuid: item.registry.uuid,
          registryName: item.registry.name || '未命名仓库',
          registryUrl: cleanRegistryUrl(item.registry.url || ''),
          registryType: item.registry.type || 'Harbor'
        }))

      if (append) {
        registries.value = [...registries.value, ...newData]
      } else {
        registries.value = newData
      }

      registryPagination.value.page = page
      registryPagination.value.total = newData.length
      registryPagination.value.hasMore = false
    } catch (error) {
      console.error('加载镜像仓库列表失败:', error)
      registries.value = []
    } finally {
      loadingRegistries.value = false
    }
  }

  async function loadMoreRegistries() {
    if (loadingRegistries.value || !registryPagination.value.hasMore) return
    await fetchRegistries(registryPagination.value.page + 1, true)
  }

  async function fetchProjects(registryUuid: string, page: number = 1, append: boolean = false) {
    if (!projectId.value || !clusterUuid.value || !registryUuid) return

    loadingProjects.value = true
    try {
      const res = await listProjectsByAppApi({
        appProjectId: projectId.value,
        clusterUuid: clusterUuid.value,
        registryUuid: registryUuid,
        page: page,
        pageSize: projectPagination.value.pageSize
      })

      if (!res?.items) {
        projects.value = []
        projectPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }
        return
      }

      const newData = res.items.map((project) => ({
        name: project.name,
        projectId: project.projectId
      }))

      if (append) {
        projects.value = [...projects.value, ...newData]
      } else {
        projects.value = newData
      }

      projectPagination.value.page = page
      projectPagination.value.total = res.total
      projectPagination.value.hasMore = projects.value.length < res.total
    } catch (error) {
      console.error('加载项目列表失败:', error)
      projects.value = []
    } finally {
      loadingProjects.value = false
    }
  }

  async function loadMoreProjects() {
    if (
      loadingProjects.value ||
      !projectPagination.value.hasMore ||
      !imageConfig.value.registryUuid
    )
      return
    await fetchProjects(imageConfig.value.registryUuid, projectPagination.value.page + 1, true)
  }

  async function fetchImages(
    registryUuid: string,
    projectName: string,
    page: number = 1,
    append: boolean = false
  ) {
    if (!registryUuid || !projectName) return

    loadingImages.value = true
    try {
      const res = await listRepositoriesApi({
        registryUuid: registryUuid,
        projectName: projectName,
        page: page,
        pageSize: imagePagination.value.pageSize,
        sortBy: 'update_time',
        sortDesc: true
      })

      if (!res?.items) {
        images.value = []
        imagePagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }
        return
      }

      const newData = res.items.map((repo) => {
        const originalName = repo.name
        const cleanedName = removeProjectPrefix(repo.name, projectName)

        return {
          name: cleanedName,
          displayName: cleanedName,
          originalName: originalName,
          artifactCount: repo.artifactCount,
          pullCount: repo.pullCount
        }
      })

      if (append) {
        images.value = [...images.value, ...newData]
      } else {
        images.value = newData
      }

      imagePagination.value.page = page
      imagePagination.value.total = res.total
      imagePagination.value.hasMore = images.value.length < res.total
    } catch (error) {
      console.error('加载镜像列表失败:', error)
      images.value = []
    } finally {
      loadingImages.value = false
    }
  }

  async function loadMoreImages() {
    if (searchApplied.value) {
      return
    }

    if (
      loadingImages.value ||
      !imagePagination.value.hasMore ||
      !imageConfig.value.registryUuid ||
      !imageConfig.value.project
    )
      return
    await fetchImages(
      imageConfig.value.registryUuid,
      imageConfig.value.project,
      imagePagination.value.page + 1,
      true
    )
  }

  function handleImageSearchClick() {
    handleImageSearch(1, false)
  }

  async function handleImageSearch(page: number = 1, append: boolean = false) {
    if (!imageSearch.value.trim()) {
      return
    }

    if (!projectId.value || !clusterUuid.value) {
      return
    }

    searchingImages.value = true
    try {
      const res = await searchImagesGlobalByProjectApi({
        appProjectId: projectId.value,
        clusterUuid: clusterUuid.value,
        imageName: imageSearch.value.trim(),
        page: page,
        pageSize: searchPagination.value.pageSize
      })

      if (res?.data && Array.isArray(res.data)) {
        res.data = res.data.map((registry: SearchResultRegistry) => ({
          ...registry,
          registryUrl: cleanRegistryUrl(registry?.registryUrl || '')
        }))
      }

      if (append && searchResults.value) {
        searchResults.value.data = [...searchResults.value.data, ...res.data]
      } else {
        searchResults.value = res
      }

      searchPagination.value.page = page
      searchPagination.value.total = res.total || 0
      searchPagination.value.hasMore = searchResults.value.data?.length < (res.total || 0)

      if ((!searchResults.value.data || searchResults.value.data.length === 0) && page === 1) {
        ElMessage.info('未找到匹配的镜像')
        searchApplied.value = false
        return
      }

      searchApplied.value = true

      const totalImages = searchResults.value.data.reduce((sum: number, r) => {
        return sum + (r?.images && Array.isArray(r.images) ? r.images.length : 0)
      }, 0)

      if (page === 1) {
        ElMessage.success(`找到 ${totalImages} 个匹配的镜像，已自动过滤下拉列表`)
        imageConfig.value.registryUuid = ''
        imageConfig.value.registryUrl = ''
        imageConfig.value.project = ''
        imageConfig.value.image = ''
        imageConfig.value.tag = ''
      }
    } catch (error) {
      console.error('搜索镜像失败:', error)
      searchApplied.value = false
    } finally {
      searchingImages.value = false
    }
  }

  function clearSearch() {
    imageSearch.value = ''
    searchResults.value = null
    searchApplied.value = false
    searchPagination.value = { page: 1, pageSize: 50, total: 0, hasMore: false }
    imageConfig.value.image = ''
    imageConfig.value.tag = ''
  }

  function getTotalFilteredCount() {
    if (!searchResults.value?.data || !Array.isArray(searchResults.value.data)) {
      return 0
    }
    return searchResults.value.data.reduce((sum: number, r) => {
      return sum + (r?.images && Array.isArray(r.images) ? r.images.length : 0)
    }, 0)
  }

  async function fetchTags(
    registryUuid: string,
    projectName: string,
    repoName: string,
    page: number = 1,
    append: boolean = false
  ) {
    if (!registryUuid || !projectName || !repoName) return

    loadingTags.value = true
    try {
      const res = await listArtifactsApi({
        registryUuid: registryUuid,
        projectName: projectName,
        repoName: repoName,
        page: page,
        pageSize: tagPagination.value.pageSize,
        sortBy: 'push_time',
        sortDesc: true
      })

      const newTags: TagItem[] = []
      res.items.forEach((artifact) => {
        artifact.tags.forEach((tag) => {
          newTags.push({
            name: tag.name,
            size: formatSize(artifact.size),
            pushTime: tag.pushTime,
            artifactId: artifact.id,
            digest: artifact.digest
          })
        })
      })

      if (append) {
        tags.value = [...tags.value, ...newTags]
      } else {
        tags.value = newTags
      }

      tagPagination.value.page = page
      tagPagination.value.total = res.total
      tagPagination.value.hasMore = tags.value.length < res.total
    } catch (error) {
      console.error('加载标签列表失败:', error)
      tags.value = []
    } finally {
      loadingTags.value = false
    }
  }

  async function loadMoreTags() {
    if (
      loadingTags.value ||
      !tagPagination.value.hasMore ||
      !imageConfig.value.registryUuid ||
      !imageConfig.value.project ||
      !imageConfig.value.image
    )
      return

    const selectedImage = images.value.find((img) => img.name === imageConfig.value.image)
    const repoName = selectedImage?.originalName || imageConfig.value.image

    await fetchTags(
      imageConfig.value.registryUuid,
      imageConfig.value.project,
      repoName,
      tagPagination.value.page + 1,
      true
    )
  }

  async function handleRegistryChange(registryUuid: string) {
    imageConfig.value.project = ''
    imageConfig.value.image = ''
    imageConfig.value.tag = ''
    projects.value = []
    images.value = []
    tags.value = []
    projectPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }
    imagePagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }
    tagPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }

    if (!registryUuid) return

    const selectedRegistry = registries.value.find((r) => r.registryUuid === registryUuid)

    if (selectedRegistry) {
      imageConfig.value.registryUrl = selectedRegistry.registryUrl
      await fetchProjects(selectedRegistry.registryUuid)
    }
  }

  async function handleProjectChange() {
    imageConfig.value.image = ''
    imageConfig.value.tag = ''
    images.value = []
    tags.value = []
    imagePagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }
    tagPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }

    if (!imageConfig.value.project || !imageConfig.value.registryUuid) return

    if (searchApplied.value && searchResults.value?.data) {
      const registryResult = searchResults.value.data.find(
        (r) => r?.registryUuid === imageConfig.value.registryUuid
      )

      if (registryResult?.images && Array.isArray(registryResult.images)) {
        const projectImages = registryResult.images.filter(
          (img) => img?.projectName === imageConfig.value.project
        )

        images.value = projectImages.map((img) => {
          const originalName = img.repoName || ''
          const cleanedName = removeProjectPrefix(originalName, imageConfig.value.project)

          return {
            name: cleanedName,
            displayName: cleanedName,
            originalName: originalName,
            artifactCount: 0,
            pullCount: 0
          }
        })

        imagePagination.value.total = images.value.length
        imagePagination.value.hasMore = false

        if (images.value.length === 0) {
          ElMessage.warning({
            message: `项目 "${imageConfig.value.project}" 下没有匹配 "${imageSearch.value}" 的镜像`,
            duration: 3000
          })
        }
      }
    } else {
      await fetchImages(imageConfig.value.registryUuid, imageConfig.value.project)
    }
  }

  async function handleImageChange() {
    imageConfig.value.tag = ''
    tags.value = []
    tagPagination.value = { page: 1, pageSize: 20, total: 0, hasMore: false }

    if (!imageConfig.value.image || !imageConfig.value.registryUuid || !imageConfig.value.project)
      return

    if (searchApplied.value && searchResults.value?.data) {
      const registryResult = searchResults.value.data.find(
        (r) => r?.registryUuid === imageConfig.value.registryUuid
      )

      if (registryResult?.images && Array.isArray(registryResult.images)) {
        const imageResult = registryResult.images.find((img) => {
          if (!img?.repoName || !img?.projectName) return false
          const cleanedName = removeProjectPrefix(img.repoName, img.projectName)
          return (
            img.projectName === imageConfig.value.project && cleanedName === imageConfig.value.image
          )
        })

        if (imageResult?.tags && Array.isArray(imageResult.tags)) {
          tags.value = imageResult.tags.map((tagName: string) => ({
            name: tagName,
            size: '-',
            pushTime: 0,
            artifactId: 0,
            digest: ''
          }))

          tagPagination.value.total = tags.value.length
          tagPagination.value.hasMore = false
          return
        }
      }
    }

    const selectedImage = images.value.find((img) => img.name === imageConfig.value.image)
    const repoName = selectedImage?.originalName || imageConfig.value.image
    await fetchTags(imageConfig.value.registryUuid, imageConfig.value.project, repoName)
  }

  // 标签选择变化处理
  function handleTagChange(selectedTag: string) {
    console.log('选择了标签:', selectedTag)
  }

  // 打开更新弹窗
  const openUpdateDialog = async (container: ContainerImageInfo) => {
    currentEditContainer.value = container
    inputMode.value = 'manual'
    manualImageInput.value = pendingChanges.value[container.containerName] || ''

    // 重置搜索选择相关状态
    imageSearch.value = ''
    searchApplied.value = false
    searchResults.value = null
    imageConfig.value = {
      registryUuid: '',
      registryUrl: '',
      project: '',
      image: '',
      tag: ''
    }
    projects.value = []
    images.value = []
    tags.value = []

    updateDialogVisible.value = true

    // 加载仓库列表供搜索使用
    if (clusterUuid.value && registries.value.length === 0) {
      await fetchRegistries()
    }
  }

  const closeUpdateDialog = () => {
    updateDialogVisible.value = false
    currentEditContainer.value = null
    manualImageInput.value = ''
  }

  const confirmUpdateImage = () => {
    if (!currentEditContainer.value || !dialogNewImage.value) return

    // 检查新镜像是否与当前镜像相同
    if (dialogNewImage.value === currentEditContainer.value.image) {
      ElMessage.warning('新镜像与当前镜像相同，无需更新')
      return
    }

    // 使用专门的函数添加待更新变更
    addPendingChange(currentEditContainer.value.containerName, dialogNewImage.value)
    ElMessage.success(`已添加 ${currentEditContainer.value.containerName} 到待更新列表`)
    closeUpdateDialog()
  }

  async function copyDialogImageUrl() {
    try {
      await navigator.clipboard.writeText(dialogNewImage.value)
      ElMessage.success('镜像地址已复制到剪贴板')
    } catch (error) {
      console.error('复制失败:', error)
    }
  }

  // 批量更新
  const handleBatchUpdate = () => {
    if (!hasPendingChanges.value) return
    batchUpdateReason.value = ''
    batchConfirmDialogVisible.value = true
  }

  const executeBatchUpdate = async () => {
    if (!hasPendingChanges.value) return

    try {
      updatingImages.value = true

      // 构建请求数据
      const containersToUpdate: ContainerImageInfo[] = Object.entries(pendingChanges.value).map(
        ([containerName, newImage]) => {
          const originalContainer = allContainers.value.find(
            (c) => c.containerName === containerName
          )
          return {
            containerName,
            containerType: originalContainer?.containerType || 'main',
            image: newImage
          }
        }
      )

      const requestData: UpdateImagesRequest = {
        containers: {
          containers: containersToUpdate
        },
        reason: batchUpdateReason.value || undefined
      }

      await updateImagesApi(props.version.id, requestData)

      ElMessage.success(`成功更新 ${containersToUpdate.length} 个容器的镜像`)
      batchConfirmDialogVisible.value = false
      clearAllPendingChanges()
      await loadContainers()
      emit('success')
    } catch (error) {
      console.error('批量更新镜像失败:', error)
    } finally {
      updatingImages.value = false
    }
  }

  // ==================== 更新策略相关 ====================
  const loadingStrategy = ref(false)
  const updatingStrategy = ref(false)
  const strategyForm = reactive({
    type: 'RollingUpdate' as string,
    rollingUpdate: {
      maxSurge: undefined as string | undefined,
      maxUnavailable: undefined as string | undefined,
      partition: undefined as number | undefined
    }
  })

  const supportedFields = computed(() => {
    const resourceType = getResourceType()

    switch (resourceType) {
      case 'DEPLOYMENT':
        return {
          maxSurge: true,
          maxUnavailable: true,
          partition: false
        }
      case 'STATEFULSET':
        return {
          maxSurge: false,
          maxUnavailable: true,
          partition: true
        }
      case 'DAEMONSET':
        return {
          maxSurge: true,
          maxUnavailable: true,
          partition: false
        }
      default:
        return {
          maxSurge: false,
          maxUnavailable: false,
          partition: false
        }
    }
  })

  const resetRollingUpdateConfig = () => {
    const fields = supportedFields.value

    strategyForm.rollingUpdate = {
      maxSurge: fields.maxSurge ? '' : undefined,
      maxUnavailable: fields.maxUnavailable ? '' : undefined,
      partition: fields.partition ? 0 : undefined
    }
  }

  // 解析数值（支持百分比和数字）
  const parseValue = (value: string | undefined): number => {
    if (!value) return 0
    const trimmed = value.trim()
    if (!trimmed) return 0
    return parseFloat(trimmed.replace('%', '')) || 0
  }

  // 检查 maxSurge 是否非零
  const isMaxSurgeNonZero = (): boolean => {
    return parseValue(strategyForm.rollingUpdate.maxSurge) > 0
  }

  // 检查 maxUnavailable 是否非零
  const isMaxUnavailableNonZero = (): boolean => {
    return parseValue(strategyForm.rollingUpdate.maxUnavailable) > 0
  }

  // 检查是否有配置冲突（DaemonSet 专用）
  const hasConflict = (): boolean => {
    if (!isDaemonSet()) return false
    return isMaxSurgeNonZero() && isMaxUnavailableNonZero()
  }

  // 获取 maxSurge 错误提示
  const getMaxSurgeError = (): string => {
    if (isDaemonSet() && hasConflict()) {
      return '与"最大不可用副本"冲突，两者不能同时大于 0'
    }
    return ''
  }

  // 获取 maxUnavailable 错误提示
  const getMaxUnavailableError = (): string => {
    if (isDaemonSet() && hasConflict()) {
      return '与"最大额外副本"冲突，两者不能同时大于 0'
    }
    return ''
  }

  // 获取提交按钮提示
  const getSubmitButtonTooltip = (): string => {
    if (!strategyForm.type) {
      return '请选择更新策略类型'
    }

    if (strategyForm.type === 'RollingUpdate') {
      if (isDeployment()) {
        const surge = parseValue(strategyForm.rollingUpdate.maxSurge)
        const unavailable = parseValue(strategyForm.rollingUpdate.maxUnavailable)
        if (surge === 0 && unavailable === 0) {
          return 'maxSurge 和 maxUnavailable 不能同时为 0'
        }
      }

      if (isDaemonSet() && hasConflict()) {
        return 'maxSurge 和 maxUnavailable 不能同时大于 0'
      }

      if (isStatefulSet()) {
        const partition = strategyForm.rollingUpdate.partition
        if (typeof partition !== 'number' || partition < 0) {
          return 'partition 必须是大于等于 0 的数字'
        }
      }
    }

    return '保存更新策略配置'
  }

  // maxSurge 变化处理
  const handleMaxSurgeChange = (value: string) => {
    if (!isDaemonSet()) return
    console.log('maxSurge 变化:', parseValue(value))
  }

  // maxUnavailable 变化处理
  const handleMaxUnavailableChange = (value: string) => {
    if (!isDaemonSet()) return
    console.log('maxUnavailable 变化:', parseValue(value))
  }

  // 自动修复冲突
  const autoFixConflict = (keepField: 'surge' | 'unavailable') => {
    if (keepField === 'surge') {
      strategyForm.rollingUpdate.maxUnavailable = '0'
      ElMessage.success('已将"最大不可用副本"设为 0')
    } else {
      strategyForm.rollingUpdate.maxSurge = '0'
      ElMessage.success('已将"最大额外副本"设为 0')
    }
  }

  const getMaxUnavailableTooltip = (): string => {
    if (isStatefulSet()) {
      return '更新期间最多可以有多少个 Pod 不可用（K8s 1.24+ 支持）'
    } else if (isDaemonSet()) {
      return '更新期间最多可以有多少个 Pod 不可用。设置此值时，maxSurge 必须为 0'
    } else {
      return '更新期间最多可以有多少个 Pod 不可用，可以是绝对数字或百分比'
    }
  }

  const getMaxUnavailablePlaceholder = (): string => {
    if (isStatefulSet()) {
      return '如: 1（留空使用默认值）'
    } else if (isDaemonSet()) {
      return '如: 1 或 10%'
    } else {
      return '如: 0 或 25%'
    }
  }

  const getMaxUnavailableTip = (): string => {
    if (isStatefulSet()) {
      return '可以是数字或百分比，留空表示每次只更新 1 个 Pod'
    } else if (isDaemonSet()) {
      return '可以是数字或百分比，默认为 1'
    } else {
      return '可以是数字或百分比，默认为 25%'
    }
  }

  /**
   * 加载更新策略
   * 添加防重复加载机制：如果正在加载中则直接返回
   */
  const loadStrategy = async () => {
    // 防止重复加载
    if (loadingStrategy.value) return

    if (!props.version?.id) {
      return
    }

    try {
      loadingStrategy.value = true
      const res = await getUpdateStrategyApi(props.version.id)

      if (res) {
        strategyForm.type = res.type || 'RollingUpdate'

        resetRollingUpdateConfig()

        if (res.rollingUpdate) {
          const fields = supportedFields.value
          const ru = res.rollingUpdate

          if (fields.maxSurge && ru.maxSurge !== undefined) {
            strategyForm.rollingUpdate.maxSurge = ru.maxSurge
          }
          if (fields.maxUnavailable && ru.maxUnavailable !== undefined) {
            strategyForm.rollingUpdate.maxUnavailable = ru.maxUnavailable
          }
          if (fields.partition && ru.partition !== undefined) {
            strategyForm.rollingUpdate.partition = ru.partition
          }
        }
      }
    } catch (error) {
      console.error('加载更新策略失败:', error)
    } finally {
      loadingStrategy.value = false
    }
  }

  const handleStrategyTypeChange = () => {
    if (strategyForm.type !== 'RollingUpdate') {
      resetRollingUpdateConfig()
    }
  }

  const isStrategyFormValid = (): boolean => {
    if (!strategyForm.type) return false

    if (strategyForm.type === 'RollingUpdate') {
      const fields = supportedFields.value
      const ru = strategyForm.rollingUpdate

      if (isDeployment()) {
        if (fields.maxSurge && fields.maxUnavailable) {
          const surge = parseValue(ru.maxSurge)
          const unavailable = parseValue(ru.maxUnavailable)
          if (surge === 0 && unavailable === 0) {
            return false
          }
        }
      }

      if (isDaemonSet()) {
        if (hasConflict()) {
          return false
        }
      }

      if (isStatefulSet()) {
        if (fields.partition) {
          if (typeof ru.partition !== 'number' || ru.partition < 0) {
            return false
          }
        }
      }
    }

    return true
  }

  const handleUpdateStrategy = async () => {
    if (!props.version?.id) {
      return
    }

    if (!isStrategyFormValid()) {
      if (isDaemonSet() && hasConflict()) {
        ElMessage.error({
          message: 'DaemonSet 的"最大额外副本"和"最大不可用副本"不能同时大于 0，请修改配置',
          duration: 5000
        })
        return
      }

      if (isDeployment()) {
        const surge = parseValue(strategyForm.rollingUpdate.maxSurge)
        const unavailable = parseValue(strategyForm.rollingUpdate.maxUnavailable)
        if (surge === 0 && unavailable === 0) {
          ElMessage.error({
            message: 'Deployment 的"最大额外副本"和"最大不可用副本"不能同时为 0',
            duration: 5000
          })
          return
        }
      }

      return
    }

    try {
      updatingStrategy.value = true

      const requestData: UpdateStrategyRequest = {
        type: strategyForm.type
      }

      if (strategyForm.type === 'RollingUpdate') {
        const fields = supportedFields.value
        const ru = strategyForm.rollingUpdate
        const rollingUpdate: Record<string, string | number> = {}

        if (fields.maxSurge && ru.maxSurge) {
          rollingUpdate.maxSurge = ru.maxSurge
        }
        if (fields.maxUnavailable && ru.maxUnavailable) {
          rollingUpdate.maxUnavailable = ru.maxUnavailable
        }
        if (fields.partition && typeof ru.partition === 'number') {
          rollingUpdate.partition = ru.partition
        }

        if (Object.keys(rollingUpdate).length > 0) {
          requestData.rollingUpdate = rollingUpdate
        }
      }

      await updateUpdateStrategyApi(props.version.id, requestData)
      ElMessage.success('更新策略已保存')
      emit('success')
      await loadStrategy()
    } catch (error) {
      console.error('更新策略失败:', error)
    } finally {
      updatingStrategy.value = false
    }
  }

  // ==================== 历史版本相关 ====================
  const loadingRevisions = ref(false)
  const rollingBack = ref(false)
  const rollbackTarget = ref<number | null>(null)
  const revisions = ref<RevisionInfo[]>([])

  const formatTime = (timestamp: number) =>
    !timestamp ? '-' : dayjs(timestamp * 1000).format('MM-DD HH:mm')

  /**
   * 加载历史版本列表
   * 添加防重复加载机制：如果正在加载中则直接返回
   */
  const loadRevisions = async () => {
    // 防止重复加载
    if (loadingRevisions.value) return

    loadingRevisions.value = true
    try {
      const response = await getResourceHistoryApi(props.version.id)
      revisions.value = response || []
    } catch (error) {
      console.error('加载历史版本失败:', error)
      revisions.value = []
    } finally {
      loadingRevisions.value = false
    }
  }

  const handleRollback = async (revision: RevisionInfo) => {
    try {
      await ElMessageBox.confirm(`确定要回滚到版本 ${revision.revision} 吗？`, '确认回滚', {
        type: 'warning'
      })
      rollingBack.value = true
      rollbackTarget.value = revision.revision
      await rollbackToRevisionApi(props.version.id, { revision: revision.revision })
      ElMessage.success(`已回滚到版本 ${revision.revision}`)
      emit('success')
      await loadRevisions()
    } catch (error) {
      // 检查是否为用户取消操作
      if (error === 'cancel' || (error instanceof Error && error.message === 'cancel')) {
        return
      }
      console.error('回滚失败:', error)
    } finally {
      rollingBack.value = false
      rollbackTarget.value = null
    }
  }

  // ==================== Tab 切换 ====================
  const handleTabChange = (index: string) => {
    activeTab.value = index
    loadCurrentTabData()
  }

  // ==================== 生命周期 ====================

  /**
   * 根据当前激活的 tab 加载对应数据
   * 每个加载函数内部都有防重复机制
   */
  const loadCurrentTabData = async () => {
    switch (activeTab.value) {
      case 'image':
        await loadContainers()
        break
      case 'strategy':
        await loadStrategy()
        break
      case 'history':
        await loadRevisions()
        break
    }
  }

  /**
   * 监听 refreshTrigger 变化
   * 只在 refreshTrigger 真正变化时才触发刷新，避免与 onMounted 重复
   */
  watch(
    () => props.refreshTrigger,
    (newVal, oldVal) => {
      // 只有当值真正变化且新值大于0时才刷新
      if (newVal && newVal > 0 && newVal !== oldVal) {
        clearAllPendingChanges()
        loadCurrentTabData()
      }
    }
  )

  /**
   * 组件挂载时加载数据
   * 由于各加载函数内部有防重复机制，即使 watch 也触发也不会重复请求
   */
  onMounted(loadCurrentTabData)
</script>

<style lang="scss" scoped>
  .update-management-container {
    height: 100%;
    min-height: 600px;

    .config-layout {
      display: flex;
      height: 100%;
      background: #fff;
      border-radius: 8px;

      .tabs-sidebar {
        width: 160px;
        border-right: 1px solid var(--el-border-color);
        padding: 20px 16px 20px 20px;
        flex-shrink: 0;

        .config-menu {
          border: none;

          :deep(.el-menu-item) {
            display: flex;
            align-items: center;
            gap: 8px;
            border-radius: 6px;
            margin-bottom: 4px;
            height: 40px;

            &.is-active {
              background-color: var(--el-color-primary-light-9);
              color: var(--el-color-primary);
            }
          }
        }
      }

      .config-content {
        flex: 1;
        padding: 20px 24px;
        overflow-y: auto;

        .config-section {
          width: 100%;
        }

        .section-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          font-size: 14px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin: 16px 0 12px 0;
          padding-bottom: 8px;
          border-bottom: 1px solid var(--el-border-color-lighter);

          &:first-child {
            margin-top: 0;
          }

          .header-left {
            display: flex;
            align-items: center;
            gap: 8px;
          }
        }

        :deep(.el-form-item .el-input),
        :deep(.el-form-item .el-select) {
          max-width: 500px;
        }

        :deep(.el-form-item .el-textarea) {
          max-width: 500px;
        }

        :deep(.el-radio-group) {
          display: flex;
          flex-direction: row;
          gap: 16px;
          flex-wrap: wrap;

          .el-radio {
            margin-right: 0;
            display: flex;
            align-items: center;
          }
        }

        .label-help-icon {
          margin-left: 6px;
          color: var(--el-text-color-secondary);
          cursor: help;
          vertical-align: middle;
        }

        .inline-help-icon {
          margin-left: 4px;
          color: var(--el-text-color-secondary);
          cursor: help;
          vertical-align: middle;
        }

        .form-actions {
          display: flex;
          gap: 12px;
          padding-top: 8px;

          :deep(.el-button) {
            display: inline-flex;
            align-items: center;
            gap: 6px;
          }
        }

        .form-tip {
          font-size: 12px;
          color: var(--el-text-color-secondary);
          margin-top: 4px;
          display: flex;
          align-items: center;
          gap: 4px;
        }

        .tip-warning {
          color: #e6a23c !important;
          font-weight: 500;
        }

        :deep(.el-form-item) {
          margin-bottom: 18px;

          .el-form-item__label {
            font-weight: 500;
            display: flex;
            align-items: center;
          }
        }

        // 待更新提示横幅
        .pending-changes-banner {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 12px 16px;
          background: linear-gradient(135deg, #fff7e6 0%, #fff3cd 100%);
          border: 1px solid #ffd591;
          border-radius: 8px;
          margin-bottom: 16px;

          .banner-content {
            display: flex;
            align-items: center;
            gap: 8px;
            color: #d48806;
            font-size: 14px;

            strong {
              color: #ad6800;
            }
          }

          .banner-actions {
            display: flex;
            gap: 8px;
          }
        }

        // 容器区域
        .container-section {
          margin-bottom: 24px;
        }

        .empty-container-tip {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          padding: 32px;
          color: var(--el-text-color-secondary);
          background: var(--el-fill-color-lighter);
          border-radius: 8px;
          gap: 8px;

          span {
            font-size: 13px;
          }
        }

        .container-list {
          display: flex;
          flex-direction: column;
          gap: 10px;
        }

        .container-card {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 14px 16px;
          background: var(--el-fill-color-lighter);
          border: 1px solid var(--el-border-color-light);
          border-radius: 8px;
          transition: all 0.2s ease;

          &:hover {
            border-color: var(--el-color-primary-light-5);
            background: var(--el-color-primary-light-9);
          }

          &.has-pending {
            border-color: var(--el-color-warning);
            background: linear-gradient(135deg, #fffbe6 0%, #fff8e6 100%);

            &:hover {
              border-color: var(--el-color-warning-light-3);
            }
          }

          .container-info {
            flex: 1;
            min-width: 0;

            .container-name {
              display: flex;
              align-items: center;
              gap: 6px;
              font-size: 14px;
              font-weight: 500;
              color: var(--el-text-color-primary);
              margin-bottom: 6px;
            }

            .container-image {
              display: flex;
              align-items: center;
              gap: 8px;
              flex-wrap: wrap;

              code {
                font-family: var(--ax-font-mono);
                font-size: 12px;
                padding: 2px 8px;
                border-radius: 4px;
                word-break: break-all;
              }

              .current-image {
                background: var(--el-fill-color);
                color: var(--el-text-color-regular);
              }

              .new-image {
                background: #e6f7ff;
                color: #1890ff;
                border: 1px solid #91d5ff;
              }

              .arrow-icon {
                color: var(--el-color-warning);
                flex-shrink: 0;
              }
            }
          }

          .container-actions {
            display: flex;
            align-items: center;
            gap: 8px;
            flex-shrink: 0;
            margin-left: 16px;

            :deep(.el-button) {
              display: inline-flex;
              align-items: center;
              gap: 4px;
            }
          }
        }

        // 底部操作区
        .bottom-actions {
          display: flex;
          justify-content: flex-end;
          gap: 12px;
          padding-top: 16px;
          border-top: 1px solid var(--el-border-color-lighter);
          margin-top: 8px;

          :deep(.el-button) {
            display: inline-flex;
            align-items: center;
            gap: 6px;
          }
        }

        .history-table {
          margin-top: 16px;

          .image-list .image-item {
            padding: 4px 0;
            font-size: 13px;

            &:not(:last-child) {
              border-bottom: 1px solid #f0f0f0;
            }
          }
        }

        .has-conflict {
          :deep(.el-input__wrapper) {
            border-color: #f56c6c !important;
            box-shadow: 0 0 0 1px #f56c6c inset !important;
          }
        }

        :deep(.el-alert) {
          code {
            background: rgba(0, 0, 0, 0.05);
            padding: 2px 6px;
            border-radius: 3px;
            font-family: var(--ax-font-mono);
            font-size: 12px;
          }
        }

        :deep(.el-form-item.is-error) {
          .el-input__wrapper {
            box-shadow: 0 0 0 1px var(--el-color-danger) inset !important;
          }
        }
      }
    }
  }

  .select-loading-more {
    padding: 8px;
    text-align: center;
    color: var(--el-color-primary);
    font-size: 12px;
    cursor: pointer;
    border-top: 1px solid var(--el-border-color);
    transition: background-color 0.2s;

    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }

  // 更新弹窗样式
  .update-dialog-content {
    .current-image-info {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 12px 16px;
      background: var(--el-fill-color-lighter);
      border-radius: 6px;
      margin-bottom: 20px;

      .label {
        font-size: 13px;
        color: var(--el-text-color-secondary);
        flex-shrink: 0;
      }

      code {
        font-family: var(--ax-font-mono);
        font-size: 12px;
        color: var(--el-text-color-primary);
        word-break: break-all;
      }
    }

    .input-mode-selector {
      margin-bottom: 20px;

      :deep(.el-radio-button__inner) {
        display: flex;
        align-items: center;
        gap: 6px;
      }
    }

    .manual-input-section,
    .search-input-section {
      :deep(.el-form-item) {
        margin-bottom: 16px;
      }

      :deep(.el-input),
      :deep(.el-select) {
        width: 100%;
        max-width: none;
      }

      .form-tip {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        margin-top: 4px;
        display: flex;
        align-items: center;
        gap: 4px;
      }

      .search-hint {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-top: 8px;
        padding: 8px 12px;
        background: #e6f7ff;
        border-left: 3px solid #1890ff;
        border-radius: 4px;
        font-size: 12px;
        color: var(--el-text-color-regular);

        svg {
          color: #1890ff;
          flex-shrink: 0;
        }
      }

      .registry-option {
        display: flex;
        justify-content: space-between;
        align-items: center;
        width: 100%;
      }

      .tag-option {
        display: flex;
        justify-content: space-between;
        align-items: center;
        width: 100%;

        .tag-size {
          font-size: 11px;
          color: var(--el-text-color-secondary);
          margin-left: 8px;
        }
      }
    }

    .new-image-preview {
      margin-top: 20px;
      padding: 12px 16px;
      background: linear-gradient(135deg, #f6ffed 0%, #e6f7ff 100%);
      border: 1px solid #b7eb8f;
      border-radius: 8px;

      .preview-label {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 13px;
        color: #52c41a;
        margin-bottom: 8px;
      }

      .preview-value {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        padding: 8px 12px;
        background: white;
        border-radius: 6px;

        code {
          flex: 1;
          font-family: var(--ax-font-mono);
          font-size: 12px;
          color: var(--el-text-color-primary);
          word-break: break-all;
        }
      }
    }
  }

  // 批量更新确认弹窗
  .batch-confirm-content {
    .changes-list {
      max-height: 300px;
      overflow-y: auto;
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      margin-bottom: 16px;

      .change-item {
        padding: 12px 16px;
        border-bottom: 1px solid var(--el-border-color-lighter);

        &:last-child {
          border-bottom: none;
        }

        .change-container {
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 14px;
          font-weight: 500;
          color: var(--el-text-color-primary);
          margin-bottom: 8px;
        }

        .change-images {
          display: flex;
          align-items: center;
          gap: 8px;
          flex-wrap: wrap;

          code {
            font-family: var(--ax-font-mono);
            font-size: 11px;
            padding: 3px 8px;
            border-radius: 4px;
            word-break: break-all;
            max-width: 220px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;

            &.old {
              background: #fef0f0;
              color: #f56c6c;
            }

            &.new {
              background: #f0f9eb;
              color: #67c23a;
            }
          }

          svg {
            color: var(--el-text-color-secondary);
            flex-shrink: 0;
          }
        }
      }
    }

    .update-reason-section {
      :deep(.el-form-item) {
        margin-bottom: 0;
      }

      :deep(.el-textarea) {
        width: 100%;
        max-width: none;
      }
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
</style>

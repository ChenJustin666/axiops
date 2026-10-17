<template>
  <ElDialog
    :model-value="modelValue"
    title="审计日志详情"
    width="800px"
    align-center
    :close-on-click-modal="false"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="detail-content" v-if="data">
      <!-- 顶部状态栏 -->
      <div class="detail-status-bar">
        <div class="status-left">
          <span class="log-id">ID: {{ data.id }}</span>
        </div>
        <ElTag :type="getAuditStatusInfo(data.status).type" size="default">
          {{ getAuditStatusInfo(data.status).label }}
        </ElTag>
      </div>

      <!-- 操作信息 -->
      <div class="detail-section">
        <div class="section-title">
          <Activity :size="18" />
          <span>操作信息</span>
        </div>
        <div class="section-body">
          <div class="info-row">
            <div class="info-item">
              <label>操作简称</label>
              <div class="info-value highlight">{{ data.title }}</div>
            </div>
          </div>
          <div class="info-row">
            <div class="info-item full-width">
              <label>操作详情</label>
              <div class="info-value detail-text">
                {{ data.actionDetail }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 资源信息 -->
      <div class="detail-section">
        <div class="section-title">
          <Server :size="18" />
          <span>资源信息</span>
        </div>
        <div class="section-body">
          <div class="info-row">
            <div class="info-item">
              <label>
                <Database :size="14" />
                集群名称
              </label>
              <div class="info-value">{{ data.clusterName }}</div>
            </div>
            <div class="info-item">
              <label>
                <Hash :size="14" />
                集群 UUID
              </label>
              <div class="info-value code">{{ data.clusterUuid }}</div>
            </div>
          </div>
          <div class="info-row">
            <div class="info-item">
              <label>
                <FolderKanban :size="14" />
                项目名称
              </label>
              <div class="info-value">{{ data.projectName || '-' }}</div>
            </div>
            <div class="info-item">
              <label>
                <Hash :size="14" />
                项目 ID
              </label>
              <div class="info-value code">{{ data.projectId || '-' }}</div>
            </div>
          </div>
          <div class="info-row">
            <div class="info-item">
              <label>
                <Layers :size="14" />
                工作空间
              </label>
              <div class="info-value">
                <template v-if="data.workspaceName || data.workspaceId">
                  {{ getWorkspaceDisplayName(data.workspaceName, data.workspaceId) }}
                  <span class="info-badge">ID: {{ data.workspaceId }}</span>
                </template>
                <span v-else class="empty-value">-</span>
              </div>
            </div>
            <div class="info-item">
              <label>
                <Box :size="14" />
                应用名称
              </label>
              <div class="info-value">
                <template v-if="data.applicationName">
                  {{ data.applicationName }}
                  <span class="info-badge">ID: {{ data.applicationId }}</span>
                </template>
                <span v-else class="empty-value">-</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 操作人信息 -->
      <div class="detail-section">
        <div class="section-title">
          <User :size="18" />
          <span>操作人信息</span>
        </div>
        <div class="section-body">
          <div class="info-row">
            <div class="info-item">
              <label>
                <UserCircle :size="14" />
                操作人
              </label>
              <div class="info-value">
                <span class="operator-name">{{ data.operatorName }}</span>
              </div>
            </div>
            <div class="info-item">
              <label>
                <Hash :size="14" />
                操作人 ID
              </label>
              <div class="info-value code">{{ data.operatorId }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 时间信息 -->
      <div class="detail-section">
        <div class="section-title">
          <Clock :size="18" />
          <span>时间信息</span>
        </div>
        <div class="section-body">
          <div class="info-row">
            <div class="info-item">
              <label>
                <Calendar :size="14" />
                操作时间
              </label>
              <div class="info-value time">
                {{ formatTimestamp(data.createdAt) }}
              </div>
            </div>
            <div class="info-item">
              <label>
                <RefreshCw :size="14" />
                更新时间
              </label>
              <div class="info-value time">
                {{ formatTimestamp(data.updatedAt) }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <ElButton @click="$emit('update:modelValue', false)">关闭</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import {
    Activity,
    Server,
    Database,
    Hash,
    FolderKanban,
    Layers,
    Box,
    User,
    UserCircle,
    Clock,
    Calendar,
    RefreshCw
  } from 'lucide-vue-next'
  import {
    getAuditStatusInfo,
    getWorkspaceDisplayName,
    type ProjectAuditLog
  } from '@/api/manager/audit'
  import { formatTimestamp } from '@/utils/format'

  interface Props {
    modelValue: boolean
    data: ProjectAuditLog | null
  }

  defineProps<Props>()
  defineEmits(['update:modelValue'])
</script>

<style lang="scss" scoped>
  // 详情内容区域
  .detail-content {
    max-height: 65vh;
    overflow-y: auto;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #dcdfe6;
      border-radius: 3px;
    }
  }

  // 顶部状态栏
  .detail-status-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    margin-bottom: 20px;
    background: #f8fafc;
    border-radius: 8px;
    border: 1px solid #e4e7ed;

    .status-left {
      .log-id {
        font-size: 13px;
        color: #64748b;
        font-family: var(--ax-font-mono);
      }
    }
  }

  // 详情分组
  .detail-section {
    margin-bottom: 20px;
    background: #fff;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid #e4e7ed;

    &:last-child {
      margin-bottom: 0;
    }

    .section-title {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 12px 16px;
      background: #f8fafc;
      border-bottom: 1px solid #e4e7ed;
      font-size: 14px;
      font-weight: 600;
      color: #303133;

      svg {
        color: #409eff;
      }
    }

    .section-body {
      padding: 16px;
    }
  }

  // 信息行
  .info-row {
    display: flex;
    gap: 20px;
    margin-bottom: 14px;

    &:last-child {
      margin-bottom: 0;
    }
  }

  // 信息项
  .info-item {
    flex: 1;
    min-width: 0;

    &.full-width {
      flex: 0 0 100%;
    }

    label {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 12px;
      color: #909399;
      margin-bottom: 6px;

      svg {
        color: #c0c4cc;
      }
    }

    .info-value {
      font-size: 14px;
      color: #303133;
      line-height: 1.5;
      word-break: break-all;

      &.highlight {
        font-size: 15px;
        font-weight: 500;
        color: #409eff;
        padding: 10px 12px;
        background: #ecf5ff;
        border-radius: 6px;
        border-left: 3px solid #409eff;
      }

      &.detail-text {
        padding: 12px;
        background: #f5f7fa;
        border-radius: 6px;
        white-space: pre-wrap;
        max-height: 180px;
        overflow-y: auto;
        line-height: 1.7;
        font-size: 13px;

        &::-webkit-scrollbar {
          width: 4px;
        }

        &::-webkit-scrollbar-thumb {
          background: #dcdfe6;
          border-radius: 2px;
        }
      }

      &.code {
        font-family: var(--ax-font-mono);
        font-size: 13px;
        color: #606266;
        background: #f5f7fa;
        padding: 4px 8px;
        border-radius: 4px;
        display: inline-block;
      }

      &.time {
        font-family: var(--ax-font-mono);
        font-size: 13px;
        color: #67c23a;
      }

      .info-badge {
        display: inline-block;
        margin-left: 8px;
        padding: 2px 6px;
        background: #f0f2f5;
        color: #606266;
        border-radius: 4px;
        font-size: 12px;
        font-family: var(--ax-font-mono);
      }

      .operator-name {
        font-weight: 500;
        color: #303133;
      }

      .empty-value {
        color: #c0c4cc;
      }
    }
  }
</style>

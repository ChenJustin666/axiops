<template>
  <div class="repository-card">
    <div class="card-header">
      <div class="repo-icon" :class="`type-${registry.type}`">
        <Server v-if="registry.type === 'harbor'" :size="24" />
        <Package v-else-if="registry.type === 'docker-registry'" :size="24" />
        <Database v-else :size="24" />
      </div>
      <div class="repo-info">
        <h3 class="repo-name" @click="$emit('manage', registry)">
          {{ registry.name }}
        </h3>
        <div class="repo-url-wrapper">
          <ElTooltip :content="registry.url" placement="top" :show-after="300">
            <div class="repo-url">
              <Globe :size="12" />
              <span>{{ registry.url }}</span>
            </div>
          </ElTooltip>
          <ElTooltip :content="copyTooltip" placement="top" :hide-after="1500">
            <button class="copy-btn" @click="copyUrl">
              <Check v-if="copied" :size="12" />
              <Copy v-else :size="12" />
            </button>
          </ElTooltip>
        </div>
      </div>
      <div class="repo-status">
        <span
          class="status-badge"
          :class="registry.status === 1 ? 'status-enabled' : 'status-disabled'"
        >
          <span class="status-dot"></span>
          {{ registry.status === 1 ? '启用' : '禁用' }}
        </span>
      </div>
    </div>

    <div class="card-body">
      <div class="repo-meta">
        <span class="meta-tag type-tag">{{ getTypeText(registry.type) }}</span>
        <span class="meta-tag env-tag" :class="`env-${registry.env}`">
          {{ getEnvText(registry.env) }}
        </span>
        <span v-if="registry.insecure" class="meta-tag warning-tag">
          <ShieldAlert :size="11" />
          跳过验证
        </span>
      </div>

      <div class="repo-stats">
        <div class="stat-item">
          <div class="stat-icon">
            <FolderGit2 :size="16" />
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ registry.totalProjects || 0 }}</div>
            <div class="stat-label">项目</div>
          </div>
        </div>

        <div class="stat-item">
          <div class="stat-icon">
            <Package :size="16" />
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ registry.totalRepositories || 0 }}</div>
            <div class="stat-label">仓库</div>
          </div>
        </div>

        <div class="stat-item">
          <div class="stat-icon">
            <HardDrive :size="16" />
          </div>
          <div class="stat-content">
            <div class="stat-value" :title="formatSizeFull(registry.storageTotal || 0)">
              {{ formatSizeCompact(registry.storageTotal || 0) }}
            </div>
            <div class="stat-label">存储</div>
          </div>
        </div>
      </div>

      <div v-if="registry.description" class="repo-description">
        <FileText :size="14" />
        <span>{{ registry.description }}</span>
      </div>
    </div>

    <div class="card-footer">
      <div class="footer-info">
        <div class="info-item">
          <User :size="13" />
          <span>{{ registry.username }}</span>
        </div>
        <div class="info-item">
          <Calendar :size="13" />
          <span>{{ formatDate(registry.createdAt) }}</span>
        </div>
      </div>

      <div class="card-actions">
        <ElButton type="primary" size="small" :icon="Settings" @click="$emit('manage', registry)">
          管理
        </ElButton>
        <ElButton size="small" :icon="Link2" @click="$emit('bind-cluster', registry)">
          绑定
        </ElButton>
        <ElDropdown @command="handleCommand" trigger="click">
          <ElButton size="small" :icon="MoreVertical" circle />
          <template #dropdown>
            <ElDropdownMenu>
              <ElDropdownItem command="edit" :icon="Edit">编辑</ElDropdownItem>
              <ElDropdownItem command="delete" :icon="Trash2" divided>删除</ElDropdownItem>
            </ElDropdownMenu>
          </template>
        </ElDropdown>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed } from 'vue'
  import {
    Server,
    Package,
    Database,
    FolderGit2,
    HardDrive,
    User,
    Calendar,
    Settings,
    Link2,
    MoreVertical,
    Edit,
    Trash2,
    Globe,
    ShieldAlert,
    FileText,
    Copy,
    Check
  } from 'lucide-vue-next'
  import type { ContainerRegistry } from '@/api'

  interface Props {
    registry: ContainerRegistry
  }

  interface Emits {
    (e: 'manage', registry: ContainerRegistry): void
    (e: 'bind-cluster', registry: ContainerRegistry): void
    (e: 'edit', registry: ContainerRegistry): void
    (e: 'delete', registry: ContainerRegistry): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  // 复制状态
  const copied = ref(false)
  const copyTooltip = computed(() => (copied.value ? '已复制' : '复制地址'))

  // 复制 URL
  const copyUrl = async () => {
    try {
      await navigator.clipboard.writeText(props.registry.url)
      copied.value = true
      setTimeout(() => {
        copied.value = false
      }, 2000)
    } catch (err) {
      console.error('复制失败:', err)
    }
  }

  const getTypeText = (type: string) => {
    const texts: Record<string, string> = {
      harbor: 'Harbor',
      'docker-registry': 'Docker Registry',
      nexus: 'Nexus'
    }
    return texts[type] || type
  }

  const getEnvText = (env: string) => {
    const texts: Record<string, string> = {
      dev: '开发',
      test: '测试',
      staging: '预发',
      prod: '生产'
    }
    return texts[env] || env
  }

  // 紧凑格式：用于卡片显示，保证不会太长
  const formatSizeCompact = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    const value = bytes / Math.pow(k, i)
    // 如果数值大于100，不显示小数；10-100显示1位小数；小于10显示2位小数
    let formatted: string
    if (value >= 100) {
      formatted = Math.round(value).toString()
    } else if (value >= 10) {
      formatted = value.toFixed(1)
    } else {
      formatted = value.toFixed(2)
    }
    // 移除末尾的 .0 或 .00
    formatted = formatted.replace(/\.0+$/, '')
    return `${formatted} ${sizes[i]}`
  }

  // 完整格式：用于 tooltip 显示精确值
  const formatSizeFull = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    const value = bytes / Math.pow(k, i)
    return `${value.toFixed(2)} ${sizes[i]} (${bytes.toLocaleString()} bytes)`
  }

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleDateString('zh-CN')
  }

  const handleCommand = (command: string) => {
    if (command === 'edit') {
      emit('edit', props.registry)
    } else if (command === 'delete') {
      emit('delete', props.registry)
    }
  }
</script>

<style lang="scss" scoped>
  .repository-card {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 16px;
    padding: 24px;
    transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
    background: var(--el-bg-color);
    position: relative;
    overflow: hidden;

    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 3px;
      background: linear-gradient(90deg, var(--el-color-primary), var(--el-color-success));
      transform: scaleX(0);
      transform-origin: left;
      transition: transform 0.4s cubic-bezier(0.4, 0, 0.2, 1);
    }

    &:hover {
      border-color: var(--el-color-primary-light-5);
      box-shadow:
        0 4px 6px -1px rgba(0, 0, 0, 0.05),
        0 10px 20px -5px rgba(0, 0, 0, 0.08),
        0 20px 40px -10px rgba(var(--el-color-primary-rgb), 0.1);
      transform: translateY(-2px);

      &::before {
        transform: scaleX(1);
      }

      .repo-name {
        color: var(--el-color-primary);
      }
    }

    .card-header {
      display: flex;
      gap: 16px;
      margin-bottom: 20px;
      align-items: flex-start;

      .repo-icon {
        width: 52px;
        height: 52px;
        border-radius: 14px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        flex-shrink: 0;
        box-shadow:
          0 4px 8px rgba(0, 0, 0, 0.12),
          inset 0 1px 0 rgba(255, 255, 255, 0.2);
        position: relative;

        &::after {
          content: '';
          position: absolute;
          inset: 0;
          border-radius: inherit;
          background: linear-gradient(180deg, rgba(255, 255, 255, 0.15) 0%, transparent 50%);
        }

        &.type-harbor {
          background: linear-gradient(145deg, #7c3aed 0%, #5b21b6 100%);
        }

        &.type-docker-registry {
          background: linear-gradient(145deg, #0ea5e9 0%, #0369a1 100%);
        }

        &.type-nexus {
          background: linear-gradient(145deg, #f59e0b 0%, #d97706 100%);
        }
      }

      .repo-info {
        flex: 1;
        min-width: 0;

        .repo-name {
          font-size: 17px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin: 0 0 10px 0;
          cursor: pointer;
          transition: color 0.25s;
          display: -webkit-box;
          -webkit-line-clamp: 1;
          -webkit-box-orient: vertical;
          overflow: hidden;
          letter-spacing: -0.01em;
        }

        .repo-url-wrapper {
          display: inline-flex;
          align-items: center;
          gap: 0;
          max-width: 100%;
        }

        .repo-url {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          font-size: 12px;
          color: var(--el-text-color-secondary);
          font-family: var(--ax-font-mono);
          background: var(--el-fill-color-light);
          padding: 5px 10px;
          border-radius: 6px 0 0 6px;
          border: 1px solid var(--el-border-color-extra-light);
          border-right: none;
          cursor: default;

          svg {
            flex-shrink: 0;
            opacity: 0.5;
          }

          span {
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            max-width: 180px;
          }
        }

        .copy-btn {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          width: 28px;
          height: 26px;
          border: 1px solid var(--el-border-color-extra-light);
          border-left: 1px solid var(--el-border-color-lighter);
          border-radius: 0 6px 6px 0;
          background: var(--el-fill-color-light);
          color: var(--el-text-color-secondary);
          cursor: pointer;
          transition: all 0.2s;
          flex-shrink: 0;

          &:hover {
            background: var(--el-color-primary-light-9);
            color: var(--el-color-primary);
            border-color: var(--el-color-primary-light-5);
          }

          &:active {
            background: var(--el-color-primary-light-8);
          }

          svg {
            flex-shrink: 0;
          }
        }
      }

      .repo-status {
        flex-shrink: 0;

        .status-badge {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: 6px 12px;
          border-radius: 20px;
          font-size: 12px;
          font-weight: 500;
          transition: all 0.25s;

          .status-dot {
            width: 6px;
            height: 6px;
            border-radius: 50%;
            position: relative;
          }

          &.status-enabled {
            background: linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%);
            color: #059669;
            border: 1px solid #a7f3d0;

            .status-dot {
              background: #10b981;
              box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
              animation: pulse-green 2s infinite;
            }
          }

          &.status-disabled {
            background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
            color: #64748b;
            border: 1px solid #e2e8f0;

            .status-dot {
              background: #94a3b8;
            }
          }
        }
      }
    }

    .card-body {
      margin-bottom: 20px;

      .repo-meta {
        display: flex;
        gap: 8px;
        margin-bottom: 16px;
        flex-wrap: wrap;

        .meta-tag {
          display: inline-flex;
          align-items: center;
          gap: 4px;
          padding: 4px 10px;
          border-radius: 6px;
          font-size: 12px;
          font-weight: 500;
          transition: all 0.2s;

          svg {
            flex-shrink: 0;
          }
        }

        .type-tag {
          background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
          color: #2563eb;
          border: 1px solid #bfdbfe;
        }

        .env-tag {
          &.env-dev {
            background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
            color: #0284c7;
            border: 1px solid #bae6fd;
          }
          &.env-test {
            background: linear-gradient(135deg, #fefce8 0%, #fef9c3 100%);
            color: #ca8a04;
            border: 1px solid #fde047;
          }
          &.env-staging {
            background: linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%);
            color: #059669;
            border: 1px solid #a7f3d0;
          }
          &.env-prod {
            background: linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%);
            color: #dc2626;
            border: 1px solid #fecaca;
          }
        }

        .warning-tag {
          background: linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%);
          color: #d97706;
          border: 1px solid #fde68a;

          svg {
            opacity: 0.8;
          }
        }
      }

      .repo-stats {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 12px;
        margin-bottom: 16px;

        .stat-item {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: 8px;
          padding: 16px 8px;
          background: linear-gradient(
            135deg,
            var(--el-fill-color-lighter) 0%,
            var(--el-fill-color-light) 100%
          );
          border-radius: 12px;
          border: 1px solid var(--el-border-color-extra-light);
          transition: all 0.25s;
          text-align: center;
          min-width: 0;

          &:hover {
            background: var(--el-fill-color-light);
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
          }

          .stat-icon {
            width: 36px;
            height: 36px;
            border-radius: 10px;
            background: linear-gradient(
              135deg,
              var(--el-color-primary-light-8),
              var(--el-color-primary-light-9)
            );
            display: flex;
            align-items: center;
            justify-content: center;
            color: var(--el-color-primary);
            flex-shrink: 0;
            box-shadow: 0 2px 6px rgba(var(--el-color-primary-rgb), 0.15);
          }

          .stat-content {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 2px;
            width: 100%;
            min-width: 0;

            .stat-value {
              font-size: 16px;
              font-weight: 700;
              color: var(--el-text-color-primary);
              line-height: 1.2;
              letter-spacing: -0.02em;
              white-space: nowrap;
              overflow: hidden;
              text-overflow: ellipsis;
              max-width: 100%;
              cursor: default;
            }

            .stat-label {
              font-size: 12px;
              color: var(--el-text-color-secondary);
              font-weight: 400;
            }
          }
        }
      }

      .repo-description {
        display: flex;
        align-items: flex-start;
        gap: 10px;
        font-size: 13px;
        color: var(--el-text-color-regular);
        line-height: 1.6;
        padding: 14px 16px;
        background: linear-gradient(
          135deg,
          var(--el-fill-color-lighter) 0%,
          var(--el-fill-color-extra-light) 100%
        );
        border-radius: 10px;
        border-left: 3px solid var(--el-color-primary-light-3);

        svg {
          flex-shrink: 0;
          margin-top: 2px;
          opacity: 0.5;
          color: var(--el-color-primary);
        }

        span {
          flex: 1;
          overflow: hidden;
          text-overflow: ellipsis;
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
        }
      }
    }

    .card-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding-top: 18px;
      border-top: 1px solid var(--el-border-color-extra-light);

      .footer-info {
        display: flex;
        gap: 12px;
        flex-wrap: wrap;

        .info-item {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          font-size: 13px;
          color: var(--el-text-color-regular);
          padding: 6px 12px;
          background: linear-gradient(
            135deg,
            var(--el-fill-color-lighter) 0%,
            var(--el-fill-color-extra-light) 100%
          );
          border-radius: 8px;
          border: 1px solid var(--el-border-color-extra-light);
          transition: all 0.2s;

          &:hover {
            background: var(--el-fill-color-light);
            border-color: var(--el-border-color-lighter);
          }

          svg {
            flex-shrink: 0;
            opacity: 0.6;
            color: var(--el-color-primary-light-3);
          }

          span {
            font-weight: 450;
          }
        }
      }

      .card-actions {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-shrink: 0;

        :deep(.el-button) {
          border-radius: 8px;
          font-weight: 500;

          &.is-circle {
            border-radius: 8px;
          }
        }
      }
    }
  }

  @keyframes pulse-green {
    0%,
    100% {
      box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
    }
    50% {
      box-shadow: 0 0 0 5px rgba(16, 185, 129, 0.1);
    }
  }
</style>

<template>
  <div class="workload-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-container">
        <!-- 总览卡片 -->
        <div class="overview-card">
          <div class="overview-header">
            <div class="header-left">
              <div class="header-icon">
                <svg viewBox="0 0 24 24" width="24" height="24">
                  <path
                    fill="currentColor"
                    d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"
                  />
                </svg>
              </div>
              <div>
                <h2>工作负载</h2>
                <p class="subtitle">Pod、Deployment、StatefulSet、DaemonSet、Job、Service</p>
              </div>
            </div>
            <div class="health-badge" :class="workloadHealthStatus.status">
              <span class="status-dot"></span>
              <span class="status-text">{{ workloadHealthStatus.text }}</span>
            </div>
          </div>
        </div>

        <!-- 工作负载网格 -->
        <div class="workload-grid">
          <!-- Pod 卡片 -->
          <div class="workload-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge pod">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"
                    />
                  </svg>
                </div>
                <div>
                  <h3>Pod</h3>
                  <p>容器实例</p>
                </div>
              </div>
              <div class="status-tag" :class="getPodStatus.class">{{ getPodStatus.text }}</div>
            </div>
            <div class="metrics-grid four-cols">
              <div class="metric-box primary">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                    />
                  </svg>
                  <span>总计</span>
                </div>
                <div class="box-value">{{ workloadData?.pods?.total || 0 }}</div>
              </div>
              <div class="metric-box success">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path fill="currentColor" d="M8 5v14l11-7z" />
                  </svg>
                  <span>运行中</span>
                </div>
                <div class="box-value">{{ workloadData?.pods?.running || 0 }}</div>
              </div>
              <div class="metric-box warning">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                    />
                  </svg>
                  <span>待调度</span>
                </div>
                <div class="box-value">{{ workloadData?.pods?.pending || 0 }}</div>
              </div>
              <div class="metric-box" :class="{ danger: (workloadData?.pods?.failed || 0) > 0 }">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                    />
                  </svg>
                  <span>失败</span>
                </div>
                <div
                  class="box-value"
                  :class="{ 'text-danger': (workloadData?.pods?.failed || 0) > 0 }"
                  >{{ workloadData?.pods?.failed || 0 }}</div
                >
              </div>
            </div>
          </div>

          <!-- Deployment 卡片 -->
          <div class="workload-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge deployment">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>Deployment</h3>
                  <p>无状态应用</p>
                </div>
              </div>
              <div class="status-tag success">正常</div>
            </div>
            <div class="metrics-grid four-cols">
              <div class="metric-box primary">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                    />
                  </svg>
                  <span>总计</span>
                </div>
                <div class="box-value">{{ workloadData?.deployments?.total || 0 }}</div>
              </div>
              <div class="metric-box success">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                    />
                  </svg>
                  <span>可用副本</span>
                </div>
                <div class="box-value">{{ workloadData?.deployments?.availableReplicas || 0 }}</div>
              </div>
              <div
                class="metric-box"
                :class="{ danger: (workloadData?.deployments?.unavailableReplicas || 0) > 0 }"
              >
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                    />
                  </svg>
                  <span>不可用</span>
                </div>
                <div
                  class="box-value"
                  :class="{
                    'text-danger': (workloadData?.deployments?.unavailableReplicas || 0) > 0
                  }"
                  >{{ workloadData?.deployments?.unavailableReplicas || 0 }}</div
                >
              </div>
              <div class="metric-box warning">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"
                    />
                  </svg>
                  <span>更新中</span>
                </div>
                <div class="box-value">{{ workloadData?.deployments?.updating || 0 }}</div>
              </div>
            </div>
          </div>

          <!-- StatefulSet 卡片 -->
          <div class="workload-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge statefulset">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>StatefulSet</h3>
                  <p>有状态应用</p>
                </div>
              </div>
              <div class="status-tag success">正常</div>
            </div>
            <div class="metrics-grid three-cols">
              <div class="metric-box primary">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                    />
                  </svg>
                  <span>总计</span>
                </div>
                <div class="box-value">{{ workloadData?.statefulSets?.total || 0 }}</div>
              </div>
              <div class="metric-box success">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                    />
                  </svg>
                  <span>就绪副本</span>
                </div>
                <div class="box-value">{{ workloadData?.statefulSets?.readyReplicas || 0 }}</div>
              </div>
              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M13 2.05v2.02c3.95.49 7 3.85 7 7.93s-3.05 7.44-7 7.93v2.02c5.05-.5 9-4.76 9-9.95s-3.95-9.45-9-9.95zm-2 0c-5.05.5-9 4.76-9 9.95s3.95 9.45 9 9.95v-2.02c-3.95-.49-7-3.85-7-7.93s3.05-7.44 7-7.93V2.05z"
                    />
                  </svg>
                  <span>当前副本</span>
                </div>
                <div class="box-value">{{ workloadData?.statefulSets?.currentReplicas || 0 }}</div>
              </div>
            </div>
          </div>

          <!-- DaemonSet 卡片 -->
          <div class="workload-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge daemonset">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>DaemonSet</h3>
                  <p>节点守护进程</p>
                </div>
              </div>
              <div class="status-tag" :class="getDaemonSetStatus.class">{{
                getDaemonSetStatus.text
              }}</div>
            </div>
            <div class="metrics-grid four-cols">
              <div class="metric-box primary">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                    />
                  </svg>
                  <span>总计</span>
                </div>
                <div class="box-value">{{ workloadData?.daemonSets?.total || 0 }}</div>
              </div>
              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"
                    />
                  </svg>
                  <span>期望 Pod</span>
                </div>
                <div class="box-value">{{ workloadData?.daemonSets?.desiredPods || 0 }}</div>
              </div>
              <div class="metric-box success">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                    />
                  </svg>
                  <span>可用 Pod</span>
                </div>
                <div class="box-value">{{ workloadData?.daemonSets?.availablePods || 0 }}</div>
              </div>
              <div
                class="metric-box"
                :class="{ danger: (workloadData?.daemonSets?.unavailablePods || 0) > 0 }"
              >
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                    />
                  </svg>
                  <span>不可用</span>
                </div>
                <div
                  class="box-value"
                  :class="{ 'text-danger': (workloadData?.daemonSets?.unavailablePods || 0) > 0 }"
                  >{{ workloadData?.daemonSets?.unavailablePods || 0 }}</div
                >
              </div>
            </div>
          </div>

          <!-- Job 卡片 -->
          <div class="workload-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge job">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>Job & CronJob</h3>
                  <p>批处理任务</p>
                </div>
              </div>
              <div class="status-tag" :class="getJobStatus.class">{{ getJobStatus.text }}</div>
            </div>
            <div class="metrics-grid four-cols">
              <div class="metric-box warning">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path fill="currentColor" d="M8 5v14l11-7z" />
                  </svg>
                  <span>活跃 Job</span>
                </div>
                <div class="box-value">{{ workloadData?.jobs?.active || 0 }}</div>
              </div>
              <div class="metric-box success">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                    />
                  </svg>
                  <span>成功</span>
                </div>
                <div class="box-value">{{ workloadData?.jobs?.succeeded || 0 }}</div>
              </div>
              <div class="metric-box" :class="{ danger: (workloadData?.jobs?.failed || 0) > 0 }">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                    />
                  </svg>
                  <span>失败</span>
                </div>
                <div
                  class="box-value"
                  :class="{ 'text-danger': (workloadData?.jobs?.failed || 0) > 0 }"
                  >{{ workloadData?.jobs?.failed || 0 }}</div
                >
              </div>
              <div class="metric-box primary">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path
                      fill="currentColor"
                      d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                    />
                  </svg>
                  <span>CronJob</span>
                </div>
                <div class="box-value">{{ workloadData?.jobs?.cronJobsTotal || 0 }}</div>
              </div>
            </div>
          </div>

          <!-- Service 卡片 -->
          <div class="workload-card">
            <div class="card-header">
              <div class="header-info">
                <div class="icon-badge service">
                  <svg viewBox="0 0 24 24" width="22" height="22">
                    <path
                      fill="currentColor"
                      d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H7c-2.76 0-5 2.24-5 5s2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm9-6h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1s-1.39 3.1-3.1 3.1h-4V17h4c2.76 0 5-2.24 5-5s-2.24-5-5-5z"
                    />
                  </svg>
                </div>
                <div>
                  <h3>Service</h3>
                  <p>服务发现与负载均衡</p>
                </div>
              </div>
              <div class="status-tag success">正常</div>
            </div>
            <div class="service-content">
              <div class="service-total">
                <span class="total-label">总计</span>
                <span class="total-value">{{ workloadData?.services?.total || 0 }}</span>
              </div>
              <div
                class="service-types"
                v-if="workloadData?.services?.byType && workloadData.services.byType.length > 0"
              >
                <div
                  v-for="typeCount in workloadData.services.byType"
                  :key="typeCount.type"
                  class="type-badge"
                  :class="getServiceTypeClass(typeCount.type)"
                >
                  <span class="type-name">{{ typeCount.type }}</span>
                  <span class="type-count">{{ typeCount.count }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 高频重启 Pod -->
        <div
          v-if="workloadData?.pods?.highRestartPods && workloadData.pods.highRestartPods.length > 0"
          class="high-restart-card"
        >
          <div class="card-header">
            <div class="header-info">
              <div class="icon-badge warning">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"
                  />
                </svg>
              </div>
              <div>
                <h3>高频重启 Pod</h3>
                <p>频繁重启的 Pod 可能存在问题</p>
              </div>
            </div>
            <div class="status-tag warning">{{ workloadData.pods.highRestartPods.length }} 个</div>
          </div>

          <div class="restart-table">
            <div class="table-header">
              <div class="col namespace">命名空间</div>
              <div class="col pod-name">Pod 名称</div>
              <div class="col restart-count">重启次数</div>
              <div class="col restart-rate">重启率</div>
            </div>
            <div class="table-body">
              <div
                v-for="pod in workloadData.pods.highRestartPods"
                :key="`${pod.namespace}-${pod.podName}`"
                class="table-row"
              >
                <div class="col namespace">{{ pod.namespace }}</div>
                <div class="col pod-name">{{ pod.podName }}</div>
                <div class="col restart-count">
                  <span class="count-badge">{{ pod.restartCount }}</span>
                </div>
                <div class="col restart-rate">{{ pod.restartRate?.toFixed(2) || '0' }} 次/小时</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path fill="currentColor" d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
          </svg>
        </div>
        <h3>暂无工作负载监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { ClusterWorkloadMetrics } from '@/api/console/cluster-monitor'

  interface Props {
    workloadData?: ClusterWorkloadMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    workloadData: undefined,
    loading: false
  })

  const podChartRef = ref<HTMLElement>()
  let podChart: echarts.ECharts | null = null
  let resizeTimer: NodeJS.Timeout | null = null

  const hasData = computed(() => {
    return !!(props.workloadData && Object.keys(props.workloadData).length > 0)
  })

  // 工作负载整体健康状态
  const workloadHealthStatus = computed(() => {
    if (!props.workloadData) return { status: 'unknown', text: '未知' }

    const podsFailed = props.workloadData.pods?.failed || 0
    const jobsFailed = props.workloadData.jobs?.failed || 0
    const daemonSetUnavailable = props.workloadData.daemonSets?.unavailablePods || 0

    if (podsFailed > 0 || jobsFailed > 0 || daemonSetUnavailable > 0) {
      return { status: 'critical', text: '有异常' }
    }

    const podsPending = props.workloadData.pods?.pending || 0
    if (podsPending > 5) {
      return { status: 'warning', text: '待调度' }
    }

    return { status: 'healthy', text: '运行正常' }
  })

  // Pod 状态
  const getPodStatus = computed(() => {
    const failed = props.workloadData?.pods?.failed || 0
    const pending = props.workloadData?.pods?.pending || 0

    if (failed > 0) return { class: 'danger', text: '有失败' }
    if (pending > 5) return { class: 'warning', text: '待调度' }
    return { class: 'success', text: '正常' }
  })

  // DaemonSet 状态
  const getDaemonSetStatus = computed(() => {
    const unavailable = props.workloadData?.daemonSets?.unavailablePods || 0
    if (unavailable > 0) return { class: 'danger', text: '有异常' }
    return { class: 'success', text: '正常' }
  })

  // Job 状态
  const getJobStatus = computed(() => {
    const failed = props.workloadData?.jobs?.failed || 0
    if (failed > 0) return { class: 'danger', text: '有失败' }
    return { class: 'success', text: '正常' }
  })

  // Service 类型样式
  const getServiceTypeClass = (type: string): string => {
    const typeMap: Record<string, string> = {
      ClusterIP: 'cluster-ip',
      NodePort: 'node-port',
      LoadBalancer: 'load-balancer',
      ExternalName: 'external-name'
    }
    return typeMap[type] || 'default'
  }

  const initPodChart = () => {
    if (!podChartRef.value) return
    podChart = echarts.init(podChartRef.value)

    const option: echarts.EChartsOption = {
      grid: { left: 10, right: 10, top: 10, bottom: 10 },
      xAxis: { type: 'time', show: false },
      yAxis: { type: 'value', show: false },
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => {
          if (!Array.isArray(params) || params.length === 0) return ''
          const date = new Date(params[0].value[0])
          const timeStr = date.toLocaleString('zh-CN')
          return `${timeStr}<br/>运行中: ${params[0].value[1]}`
        }
      },
      series: [
        {
          type: 'line',
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#10b981' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(16, 185, 129, 0.25)' },
              { offset: 1, color: 'rgba(16, 185, 129, 0.05)' }
            ])
          },
          data: []
        }
      ]
    }

    podChart.setOption(option)
  }

  const updatePodChart = () => {
    if (!podChart || !props.workloadData?.pods?.trend) return

    const data = props.workloadData.pods.trend.map((item) => [item.timestamp * 1000, item.running])
    podChart.setOption({ series: [{ data }] })
  }

  const handleResize = () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      podChart?.resize()
    }, 300)
  }

  watch(
    () => props.workloadData,
    () => {
      nextTick(() => {
        if (hasData.value) {
          if (!podChart) initPodChart()
          updatePodChart()
        }
      })
    },
    { deep: true }
  )

  onMounted(() => {
    if (hasData.value) {
      initPodChart()
      updatePodChart()
    }
    window.addEventListener('resize', handleResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    if (resizeTimer) clearTimeout(resizeTimer)
    podChart?.dispose()
  })
</script>

<style lang="scss" scoped>
  $primary: #3b82f6;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;
  $gray-50: #f9fafb;
  $gray-100: #f3f4f6;
  $gray-200: #e5e7eb;
  $gray-300: #d1d5db;
  $gray-400: #9ca3af;
  $gray-500: #6b7280;
  $gray-600: #4b5563;
  $gray-700: #374151;
  $gray-800: #1f2937;
  $radius: 14px;
  $radius-sm: 10px;
  $radius-xs: 6px;

  .workload-monitor {
    background: $gray-50;
    font-family: var(--ax-font-sans);

    .monitor-container {
      max-width: 1800px;
      margin: 0 auto;
      display: flex;
      flex-direction: column;
      gap: 24px;
    }

    .overview-card {
      background: #fff;
      border-radius: $radius;
      padding: 24px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);

      .overview-header {
        display: flex;
        justify-content: space-between;
        align-items: center;

        .header-left {
          display: flex;
          align-items: center;
          gap: 14px;

          .header-icon {
            width: 44px;
            height: 44px;
            border-radius: $radius-sm;
            background: linear-gradient(135deg, #d1fae5, #a7f3d0);
            display: flex;
            align-items: center;
            justify-content: center;
            color: $success;
          }

          h2 {
            margin: 0;
            font-size: 18px;
            font-weight: 600;
            color: $gray-800;
          }

          .subtitle {
            margin: 4px 0 0;
            font-size: 13px;
            color: $gray-500;
          }
        }

        .health-badge {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 10px 18px;
          border-radius: 24px;
          font-weight: 500;
          font-size: 14px;

          .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            animation: pulse 2s infinite;
          }

          &.healthy {
            background: rgba($success, 0.1);
            color: #059669;
            .status-dot {
              background: $success;
            }
          }

          &.warning {
            background: rgba($warning, 0.1);
            color: #d97706;
            .status-dot {
              background: $warning;
            }
          }

          &.critical {
            background: rgba($danger, 0.1);
            color: #dc2626;
            .status-dot {
              background: $danger;
            }
          }

          &.unknown {
            background: $gray-100;
            color: $gray-500;
            .status-dot {
              background: $gray-400;
            }
          }
        }
      }
    }

    .workload-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 20px;

      @media (max-width: 1400px) {
        grid-template-columns: repeat(2, 1fr);
      }

      @media (max-width: 900px) {
        grid-template-columns: 1fr;
      }
    }

    .workload-card {
      background: #fff;
      border-radius: $radius;
      padding: 20px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);

      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 18px;
        padding-bottom: 14px;
        border-bottom: 1px solid $gray-200;

        .header-info {
          display: flex;
          align-items: center;
          gap: 12px;

          .icon-badge {
            width: 46px;
            height: 46px;
            border-radius: $radius-sm;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fff;
            flex-shrink: 0;

            &.pod {
              background: linear-gradient(135deg, #43e97b, #38f9d7);
            }

            &.deployment {
              background: linear-gradient(135deg, #4facfe, #00f2fe);
            }

            &.statefulset {
              background: linear-gradient(135deg, #667eea, #764ba2);
            }

            &.daemonset {
              background: linear-gradient(135deg, #a8c0ff, #3f2b96);
            }

            &.job {
              background: linear-gradient(135deg, #f093fb, #f5576c);
            }

            &.service {
              background: linear-gradient(135deg, $gray-500, $gray-600);
            }

            &.warning {
              background: linear-gradient(135deg, $warning, #fbbf24);
            }
          }

          h3 {
            margin: 0;
            font-size: 15px;
            font-weight: 600;
            color: $gray-800;
          }

          p {
            margin: 2px 0 0;
            font-size: 12px;
            color: $gray-500;
          }
        }

        .status-tag {
          padding: 5px 12px;
          border-radius: 14px;
          font-size: 11px;
          font-weight: 600;

          &.success {
            background: rgba($success, 0.1);
            color: #059669;
          }

          &.warning {
            background: rgba($warning, 0.1);
            color: #d97706;
          }

          &.danger {
            background: rgba($danger, 0.1);
            color: #dc2626;
          }
        }
      }

      .metrics-grid {
        display: grid;
        gap: 10px;

        &.four-cols {
          grid-template-columns: repeat(4, 1fr);

          @media (max-width: 1600px) {
            grid-template-columns: repeat(2, 1fr);
          }
        }

        &.three-cols {
          grid-template-columns: repeat(3, 1fr);
        }
      }

      .metric-box {
        padding: 14px;
        border-radius: $radius-xs;
        background: $gray-50;
        border: 1px solid $gray-200;
        transition: all 0.2s ease;

        &:hover {
          border-color: $gray-300;
        }

        &.primary {
          background: rgba($primary, 0.04);
          border-color: rgba($primary, 0.15);

          .box-value {
            color: $primary;
          }
        }

        &.success {
          background: rgba($success, 0.04);
          border-color: rgba($success, 0.15);

          .box-value {
            color: #059669;
          }
        }

        &.warning {
          background: rgba($warning, 0.04);
          border-color: rgba($warning, 0.15);

          .box-value {
            color: #d97706;
          }
        }

        &.danger {
          background: rgba($danger, 0.04);
          border-color: rgba($danger, 0.2);
        }

        .box-header {
          display: flex;
          align-items: center;
          gap: 6px;
          margin-bottom: 8px;
          color: $gray-500;

          span {
            font-size: 11px;
            font-weight: 500;
          }
        }

        .box-value {
          font-size: 20px;
          font-weight: 700;
          color: $gray-800;

          &.text-danger {
            color: $danger;
          }
        }
      }

      .service-content {
        .service-total {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 14px 16px;
          background: rgba($primary, 0.04);
          border: 1px solid rgba($primary, 0.15);
          border-radius: $radius-xs;
          margin-bottom: 14px;

          .total-label {
            font-size: 13px;
            color: $gray-600;
          }

          .total-value {
            font-size: 24px;
            font-weight: 700;
            color: $primary;
          }
        }

        .service-types {
          display: flex;
          flex-wrap: wrap;
          gap: 8px;

          .type-badge {
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 6px 12px;
            border-radius: 14px;
            font-size: 12px;

            .type-name {
              font-weight: 500;
            }

            .type-count {
              font-weight: 700;
            }

            &.cluster-ip {
              background: rgba($primary, 0.1);
              color: $primary;
            }

            &.node-port {
              background: rgba($success, 0.1);
              color: #059669;
            }

            &.load-balancer {
              background: rgba(#a855f7, 0.1);
              color: #7c3aed;
            }

            &.external-name {
              background: rgba($warning, 0.1);
              color: #d97706;
            }

            &.default {
              background: $gray-100;
              color: $gray-600;
            }
          }
        }
      }
    }

    .high-restart-card {
      background: #fff;
      border-radius: $radius;
      padding: 20px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      border: 1px solid rgba($warning, 0.2);

      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 18px;
        padding-bottom: 14px;
        border-bottom: 1px solid $gray-200;

        .header-info {
          display: flex;
          align-items: center;
          gap: 12px;

          .icon-badge {
            width: 46px;
            height: 46px;
            border-radius: $radius-sm;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fff;
            background: linear-gradient(135deg, $warning, #fbbf24);
          }

          h3 {
            margin: 0;
            font-size: 15px;
            font-weight: 600;
            color: $gray-800;
          }

          p {
            margin: 2px 0 0;
            font-size: 12px;
            color: $gray-500;
          }
        }

        .status-tag {
          padding: 5px 12px;
          border-radius: 14px;
          font-size: 11px;
          font-weight: 600;
          background: rgba($warning, 0.1);
          color: #d97706;
        }
      }

      .restart-table {
        border: 1px solid $gray-200;
        border-radius: $radius-xs;
        overflow: hidden;

        .table-header {
          display: flex;
          background: $gray-50;
          border-bottom: 1px solid $gray-200;
          padding: 12px 16px;
          font-size: 12px;
          font-weight: 600;
          color: $gray-600;
        }

        .table-body {
          .table-row {
            display: flex;
            padding: 12px 16px;
            border-bottom: 1px solid $gray-100;
            font-size: 13px;
            color: $gray-700;
            transition: background 0.2s ease;

            &:last-child {
              border-bottom: none;
            }

            &:hover {
              background: $gray-50;
            }
          }
        }

        .col {
          &.namespace {
            width: 20%;
          }

          &.pod-name {
            flex: 1;
            font-weight: 500;
          }

          &.restart-count {
            width: 15%;
            text-align: center;
          }

          &.restart-rate {
            width: 20%;
            text-align: right;
            color: $gray-500;
          }
        }

        .count-badge {
          display: inline-block;
          padding: 2px 10px;
          border-radius: 10px;
          font-size: 12px;
          font-weight: 600;
          background: rgba($danger, 0.1);
          color: #dc2626;
        }
      }
    }

    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 400px;
      background: #fff;
      border-radius: $radius;
      padding: 60px 40px;

      .empty-icon {
        color: $gray-300;
        margin-bottom: 20px;
      }

      h3 {
        margin: 0 0 10px;
        font-size: 16px;
        color: $gray-600;
        font-weight: 500;
      }

      p {
        margin: 0;
        font-size: 13px;
        color: $gray-400;
      }
    }
  }

  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
  }

  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }
</style>

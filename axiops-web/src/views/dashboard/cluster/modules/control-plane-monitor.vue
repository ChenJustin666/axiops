<template>
  <div class="control-plane-monitor" v-loading="loading">
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
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
                  />
                </svg>
              </div>
              <div>
                <h2>控制平面概览</h2>
                <p class="subtitle">集群控制组件实时监控与健康分析</p>
              </div>
            </div>
            <div class="health-badge" :class="overallHealth.status">
              <span class="status-dot"></span>
              <span class="status-text">{{ overallHealth.text }}</span>
              <span class="health-score">{{ overallHealth.score }}/100</span>
            </div>
          </div>

          <!-- 问题告警 -->
          <div v-if="healthIssues.length > 0" class="issues-alert">
            <div class="alert-header">
              <svg viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                />
              </svg>
              <span>检测到 {{ healthIssues.length }} 个问题</span>
            </div>
            <div class="issues-list">
              <div v-for="(issue, index) in healthIssues" :key="index" class="issue-item">
                <span class="issue-dot"></span>
                <span>{{ issue }}</span>
              </div>
            </div>
          </div>

          <!-- 快速指标 -->
          <div class="quick-metrics">
            <div class="quick-metric">
              <div class="metric-icon api">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">API QPS</span>
                <span class="metric-value">{{ formatNumber(apiServer?.requestsPerSecond) }}</span>
                <span class="metric-sub"
                  >错误率 {{ formatPercent((apiServer?.errorRate || 0) * 100) }}%</span
                >
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon scheduler">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">调度成功率</span>
                <span class="metric-value"
                  >{{ formatPercent((scheduler?.scheduleSuccessRate || 0) * 100) }}%</span
                >
                <span class="metric-sub">待调度 {{ formatInteger(scheduler?.pendingPods) }}</span>
              </div>
            </div>
            <div class="quick-metric">
              <div class="metric-icon controller">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                  />
                </svg>
              </div>
              <div class="metric-info">
                <span class="metric-label">队列总数</span>
                <span class="metric-value">{{ formatInteger(getTotalQueueDepth()) }}</span>
                <span class="metric-sub">{{
                  controllerManager?.isLeader ? 'Leader' : '非Leader'
                }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- API Server -->
        <div class="monitor-card">
          <div class="card-header">
            <div class="header-info">
              <div class="icon-badge api-server">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"
                  />
                </svg>
              </div>
              <div>
                <h3>API Server</h3>
                <p>集群网关与认证中心</p>
              </div>
            </div>
            <div class="status-tag" :class="apiServerStatus.class">{{ apiServerStatus.text }}</div>
          </div>

          <div class="metrics-grid">
            <div class="metric-box">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M13 2.05v2.02c3.95.49 7 3.85 7 7.93s-3.05 7.44-7 7.93v2.02c5.05-.5 9-4.76 9-9.95s-3.95-9.45-9-9.95zm-2 0c-5.05.5-9 4.76-9 9.95s3.95 9.45 9 9.95v-2.02c-3.95-.49-7-3.85-7-7.93s3.05-7.44 7-7.93V2.05z"
                  />
                </svg>
                <span>请求速率</span>
              </div>
              <div class="box-value">
                <span class="value">{{ formatNumber(apiServer?.requestsPerSecond) }}</span>
                <span class="unit">req/s</span>
              </div>
              <div class="box-badge" :class="getQPSLevel(apiServer?.requestsPerSecond)">{{
                getQPSLevelText(apiServer?.requestsPerSecond)
              }}</div>
            </div>

            <div class="metric-box" :class="{ warning: (apiServer?.errorRate || 0) > 0.01 }">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
                  />
                </svg>
                <span>错误率</span>
              </div>
              <div class="box-value">
                <span class="value">{{ formatPercent((apiServer?.errorRate || 0) * 100) }}</span>
                <span class="unit">%</span>
              </div>
            </div>

            <div class="metric-box">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                  />
                </svg>
                <span>P95 延迟</span>
              </div>
              <div class="box-value">
                <span class="value">{{ formatLatency(apiServer?.p95Latency) }}</span>
              </div>
            </div>

            <div
              class="metric-box"
              :class="{ warning: (apiServer?.currentInflightRequests || 0) > 600 }"
            >
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"
                  />
                </svg>
                <span>并发请求</span>
              </div>
              <div class="box-value">
                <span class="value">{{ formatInteger(apiServer?.currentInflightRequests) }}</span>
                <span class="unit">/600</span>
              </div>
              <div class="box-progress"
                ><div
                  class="progress-fill"
                  :style="{
                    width: getConcurrencyPercent(apiServer?.currentInflightRequests) + '%'
                  }"
                ></div
              ></div>
            </div>
          </div>

          <div class="latency-bar">
            <span class="bar-title">延迟分布</span>
            <div class="bar-items">
              <div class="bar-item"
                ><span class="label">P50</span
                ><span class="value">{{ formatLatency(apiServer?.p50Latency) }}</span></div
              >
              <div class="bar-item"
                ><span class="label">P95</span
                ><span class="value">{{ formatLatency(apiServer?.p95Latency) }}</span></div
              >
              <div class="bar-item"
                ><span class="label">P99</span
                ><span class="value">{{ formatLatency(apiServer?.p99Latency) }}</span></div
              >
            </div>
          </div>

          <div class="chart-area">
            <div class="chart-header">
              <span class="chart-title">请求与延迟趋势</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot qps"></i>QPS</span>
                <span class="legend-item"><i class="dot error"></i>错误率</span>
                <span class="legend-item"><i class="dot latency"></i>P95延迟</span>
              </div>
            </div>
            <div ref="apiServerChartRef" class="chart-container"></div>
          </div>

          <div class="detail-panel">
            <div class="panel-toggle" @click="toggleApiServerDetail">
              <span>详细指标</span>
              <svg
                :class="{ rotated: showApiServerDetail }"
                viewBox="0 0 24 24"
                width="16"
                height="16"
              >
                <path
                  fill="currentColor"
                  d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z"
                />
              </svg>
            </div>
            <Transition name="slide">
              <div v-if="showApiServerDetail" class="panel-content">
                <div class="detail-section">
                  <div class="section-title">并发与连接</div>
                  <div class="detail-grid">
                    <div class="detail-item"
                      ><span class="label">读请求</span
                      ><span class="value">{{
                        formatInteger(apiServer?.inflightReadRequests)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">写请求</span
                      ><span class="value">{{
                        formatInteger(apiServer?.inflightMutatingRequests)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">长连接</span
                      ><span class="value">{{
                        formatInteger(apiServer?.longRunningRequests)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">Watch连接</span
                      ><span class="value">{{ formatInteger(apiServer?.watchCount) }}</span></div
                    >
                  </div>
                </div>
                <div class="detail-section">
                  <div class="section-title">性能与异常</div>
                  <div class="detail-grid">
                    <div class="detail-item"
                      ><span class="label">请求丢弃</span
                      ><span
                        class="value"
                        :class="{ danger: (apiServer?.requestDropped || 0) > 0 }"
                        >{{ formatInteger(apiServer?.requestDropped) }}</span
                      ></div
                    >
                    <div class="detail-item"
                      ><span class="label">请求超时</span
                      ><span
                        class="value"
                        :class="{ danger: (apiServer?.requestTimeout || 0) > 0 }"
                        >{{ formatInteger(apiServer?.requestTimeout) }}</span
                      ></div
                    >
                    <div class="detail-item"
                      ><span class="label">响应大小</span
                      ><span class="value">{{
                        formatBytes(apiServer?.responseSizeBytes)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">Webhook延迟</span
                      ><span class="value">{{
                        formatLatency(apiServer?.webhookDurationSeconds)
                      }}</span></div
                    >
                  </div>
                </div>
              </div>
            </Transition>
          </div>
        </div>

        <!-- Scheduler -->
        <div class="monitor-card">
          <div class="card-header">
            <div class="header-info">
              <div class="icon-badge scheduler">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2z"
                  />
                </svg>
              </div>
              <div>
                <h3>Scheduler</h3>
                <p>Pod 调度与资源分配</p>
              </div>
            </div>
            <div class="status-tag" :class="schedulerStatus.class">{{ schedulerStatus.text }}</div>
          </div>

          <div class="metrics-grid">
            <div class="metric-box primary">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                </svg>
                <span>调度成功率</span>
              </div>
              <div class="box-value">
                <span class="value">{{
                  formatPercent((scheduler?.scheduleSuccessRate || 0) * 100)
                }}</span>
                <span class="unit">%</span>
              </div>
              <div class="box-badge" :class="getSuccessRateLevel(scheduler?.scheduleSuccessRate)">{{
                getSuccessRateText(scheduler?.scheduleSuccessRate)
              }}</div>
              <div class="box-progress success"
                ><div
                  class="progress-fill"
                  :style="{ width: (scheduler?.scheduleSuccessRate || 0) * 100 + '%' }"
                ></div
              ></div>
            </div>

            <div class="metric-box" :class="{ warning: (scheduler?.pendingPods || 0) > 10 }">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                  />
                </svg>
                <span>待调度</span>
              </div>
              <div class="box-value">
                <span class="value">{{ formatInteger(scheduler?.pendingPods) }}</span>
                <span class="unit">个</span>
              </div>
            </div>

            <div class="metric-box" :class="{ danger: (scheduler?.unschedulablePods || 0) > 0 }">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"
                  />
                </svg>
                <span>无法调度</span>
              </div>
              <div class="box-value">
                <span class="value danger">{{ formatInteger(scheduler?.unschedulablePods) }}</span>
                <span class="unit">个</span>
              </div>
            </div>

            <div class="metric-box">
              <div class="box-header">
                <svg viewBox="0 0 24 24" width="16" height="16">
                  <path
                    fill="currentColor"
                    d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z"
                  />
                </svg>
                <span>P95 延迟</span>
              </div>
              <div class="box-value">
                <span class="value">{{ formatLatency(scheduler?.p95ScheduleLatency) }}</span>
              </div>
            </div>
          </div>

          <div class="stats-bar">
            <div class="stats-item success"
              ><span class="stats-label">已调度</span
              ><span class="stats-value">{{ formatInteger(scheduler?.scheduledPods) }}</span></div
            >
            <div class="stats-item danger"
              ><span class="stats-label">调度失败</span
              ><span class="stats-value">{{
                formatInteger(scheduler?.failedScheduling)
              }}</span></div
            >
            <div class="stats-item warning"
              ><span class="stats-label">抢占尝试</span
              ><span class="stats-value">{{
                formatInteger(scheduler?.preemptionAttempts)
              }}</span></div
            >
            <div class="stats-item info"
              ><span class="stats-label">队列长度</span
              ><span class="stats-value">{{
                formatInteger(scheduler?.schedulingQueueLength)
              }}</span></div
            >
          </div>

          <div class="chart-area">
            <div class="chart-header">
              <span class="chart-title">调度趋势</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot success"></i>成功率</span>
                <span class="legend-item"><i class="dot pending"></i>待调度</span>
                <span class="legend-item"><i class="dot latency"></i>P95延迟</span>
              </div>
            </div>
            <div ref="schedulerChartRef" class="chart-container"></div>
          </div>

          <div class="detail-panel">
            <div class="panel-toggle" @click="toggleSchedulerDetail">
              <span>失败原因与队列分析</span>
              <svg
                :class="{ rotated: showSchedulerDetail }"
                viewBox="0 0 24 24"
                width="16"
                height="16"
              >
                <path
                  fill="currentColor"
                  d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z"
                />
              </svg>
            </div>
            <Transition name="slide">
              <div v-if="showSchedulerDetail" class="panel-content">
                <div class="detail-section">
                  <div class="section-title">延迟分布</div>
                  <div class="detail-grid">
                    <div class="detail-item"
                      ><span class="label">P50</span
                      ><span class="value">{{
                        formatLatency(scheduler?.p50ScheduleLatency)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">P95</span
                      ><span class="value">{{
                        formatLatency(scheduler?.p95ScheduleLatency)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">P99</span
                      ><span class="value">{{
                        formatLatency(scheduler?.p99ScheduleLatency)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">绑定延迟</span
                      ><span class="value">{{
                        formatLatency(scheduler?.bindingLatency)
                      }}</span></div
                    >
                  </div>
                </div>
                <div class="detail-section">
                  <div class="section-title">调度队列</div>
                  <div class="detail-grid">
                    <div class="detail-item"
                      ><span class="label">总队列</span
                      ><span class="value">{{
                        formatInteger(scheduler?.schedulingQueueLength)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">活跃队列</span
                      ><span class="value">{{
                        formatInteger(scheduler?.activeQueueLength)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">退避队列</span
                      ><span class="value">{{
                        formatInteger(scheduler?.backoffQueueLength)
                      }}</span></div
                    >
                  </div>
                </div>
              </div>
            </Transition>
          </div>
        </div>

        <!-- Controller Manager -->
        <div class="monitor-card">
          <div class="card-header">
            <div class="header-info">
              <div class="icon-badge controller">
                <svg viewBox="0 0 24 24" width="22" height="22">
                  <path
                    fill="currentColor"
                    d="M4 8h4V4H4v4zm6 12h4v-4h-4v4zm-6 0h4v-4H4v4zm0-6h4v-4H4v4zm6 0h4v-4h-4v4zm6-10v4h4V4h-4zm-6 4h4V4h-4v4zm6 6h4v-4h-4v4zm0 6h4v-4h-4v4z"
                  />
                </svg>
              </div>
              <div>
                <h3>Controller Manager</h3>
                <p>资源控制器管理</p>
              </div>
            </div>
            <div class="status-tag" :class="controllerStatus.class">{{
              controllerStatus.text
            }}</div>
          </div>

          <div class="leader-status" :class="{ active: controllerManager?.isLeader }">
            <div class="leader-icon">
              <svg v-if="controllerManager?.isLeader" viewBox="0 0 24 24" width="28" height="28">
                <path
                  fill="currentColor"
                  d="M5 16L3 5l5.5 5L12 4l3.5 6L21 5l-2 11H5m14 3c0 .6-.4 1-1 1H6c-.6 0-1-.4-1-1v-1h14v1z"
                />
              </svg>
              <svg v-else viewBox="0 0 24 24" width="28" height="28">
                <path
                  fill="currentColor"
                  d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z"
                />
              </svg>
            </div>
            <div class="leader-info">
              <span class="leader-title">{{
                controllerManager?.isLeader ? 'Leader 状态' : '非 Leader'
              }}</span>
              <span class="leader-sub"
                >Leader 变更: {{ formatInteger(controllerManager?.leaderChanges) }} 次</span
              >
            </div>
          </div>

          <div class="queue-section">
            <div class="queue-title">控制器队列深度</div>
            <div class="queue-grid">
              <div class="queue-item">
                <div class="queue-icon"
                  ><svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
                    /></svg
                ></div>
                <div class="queue-info"
                  ><span class="queue-name">Deployment</span
                  ><span class="queue-value">{{
                    formatInteger(controllerManager?.deploymentQueueDepth)
                  }}</span></div
                >
              </div>
              <div class="queue-item">
                <div class="queue-icon"
                  ><svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"
                    /></svg
                ></div>
                <div class="queue-info"
                  ><span class="queue-name">ReplicaSet</span
                  ><span class="queue-value">{{
                    formatInteger(controllerManager?.replicaSetQueueDepth)
                  }}</span></div
                >
              </div>
              <div class="queue-item">
                <div class="queue-icon"
                  ><svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"
                    /></svg
                ></div>
                <div class="queue-info"
                  ><span class="queue-name">StatefulSet</span
                  ><span class="queue-value">{{
                    formatInteger(controllerManager?.statefulSetQueueDepth)
                  }}</span></div
                >
              </div>
              <div class="queue-item">
                <div class="queue-icon"
                  ><svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93s3.05-7.44 7-7.93V2.05c-5.05.5-9 4.76-9 9.95s3.95 9.45 9 9.95v-2.02z"
                    /></svg
                ></div>
                <div class="queue-info"
                  ><span class="queue-name">DaemonSet</span
                  ><span class="queue-value">{{
                    formatInteger(controllerManager?.daemonSetQueueDepth)
                  }}</span></div
                >
              </div>
              <div class="queue-item">
                <div class="queue-icon"
                  ><svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"
                    /></svg
                ></div>
                <div class="queue-info"
                  ><span class="queue-name">Job</span
                  ><span class="queue-value">{{
                    formatInteger(controllerManager?.jobQueueDepth)
                  }}</span></div
                >
              </div>
              <div class="queue-item">
                <div class="queue-icon"
                  ><svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"
                    /></svg
                ></div>
                <div class="queue-info"
                  ><span class="queue-name">Node</span
                  ><span class="queue-value">{{
                    formatInteger(controllerManager?.nodeQueueDepth)
                  }}</span></div
                >
              </div>
              <div class="queue-item">
                <div class="queue-icon"
                  ><svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M21 3H3c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H3V5h18v14zM9 8h2v8H9V8zm4 2h2v6h-2v-6z"
                    /></svg
                ></div>
                <div class="queue-info"
                  ><span class="queue-name">Service</span
                  ><span class="queue-value">{{
                    formatInteger(controllerManager?.serviceQueueDepth)
                  }}</span></div
                >
              </div>
              <div class="queue-item">
                <div class="queue-icon"
                  ><svg viewBox="0 0 24 24" width="18" height="18">
                    <path
                      fill="currentColor"
                      d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H7c-2.76 0-5 2.24-5 5s2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm9-6h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1s-1.39 3.1-3.1 3.1h-4V17h4c2.76 0 5-2.24 5-5s-2.24-5-5-5z"
                    /></svg
                ></div>
                <div class="queue-info"
                  ><span class="queue-name">Endpoint</span
                  ><span class="queue-value">{{
                    formatInteger(controllerManager?.endpointQueueDepth)
                  }}</span></div
                >
              </div>
            </div>
          </div>

          <div class="chart-area">
            <div class="chart-header">
              <span class="chart-title">队列趋势</span>
              <div class="chart-legend">
                <span class="legend-item"><i class="dot deployment"></i>Deployment</span>
                <span class="legend-item"><i class="dot replicaset"></i>ReplicaSet</span>
                <span class="legend-item"><i class="dot statefulset"></i>StatefulSet</span>
                <span class="legend-item"><i class="dot total"></i>总深度</span>
              </div>
            </div>
            <div ref="controllerChartRef" class="chart-container"></div>
          </div>

          <div class="detail-panel">
            <div class="panel-toggle" @click="toggleControllerDetail">
              <span>工作队列详细统计</span>
              <svg
                :class="{ rotated: showControllerDetail }"
                viewBox="0 0 24 24"
                width="16"
                height="16"
              >
                <path
                  fill="currentColor"
                  d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z"
                />
              </svg>
            </div>
            <Transition name="slide">
              <div v-if="showControllerDetail" class="panel-content">
                <div class="detail-section" v-if="controllerManager?.workQueueMetrics">
                  <div class="section-title">工作队列统计</div>
                  <div class="detail-grid">
                    <div class="detail-item"
                      ><span class="label">添加速率</span
                      ><span class="value"
                        >{{ formatNumber(controllerManager.workQueueMetrics.addsRate) }} /s</span
                      ></div
                    >
                    <div class="detail-item"
                      ><span class="label">队列总深度</span
                      ><span class="value">{{
                        formatInteger(controllerManager.workQueueMetrics.depthTotal)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">未完成工作</span
                      ><span class="value">{{
                        formatInteger(controllerManager.workQueueMetrics.unfinishedWork)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">重试率</span
                      ><span class="value"
                        >{{ formatNumber(controllerManager.workQueueMetrics.retriesRate) }} /s</span
                      ></div
                    >
                  </div>
                </div>
                <div class="detail-section">
                  <div class="section-title">错误与重试</div>
                  <div class="detail-grid">
                    <div class="detail-item"
                      ><span class="label">同步错误总数</span
                      ><span class="value danger">{{
                        formatInteger(controllerManager?.totalSyncErrors)
                      }}</span></div
                    >
                    <div class="detail-item"
                      ><span class="label">重试率</span
                      ><span class="value"
                        >{{ formatNumber(controllerManager?.retryRate) }} /s</span
                      ></div
                    >
                  </div>
                </div>
              </div>
            </Transition>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon"
          ><svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2V9h2v8zm0-10h-2V5h2v2z"
            /></svg
        ></div>
        <h3>暂无控制平面数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { ClusterControlPlaneMetrics } from '@/api/console/cluster-monitor'

  interface Props {
    controlPlaneData?: ClusterControlPlaneMetrics
    loading?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    controlPlaneData: undefined,
    loading: false
  })

  const apiServerChartRef = ref<HTMLElement>()
  const schedulerChartRef = ref<HTMLElement>()
  const controllerChartRef = ref<HTMLElement>()

  let apiServerChart: echarts.ECharts | null = null
  let schedulerChart: echarts.ECharts | null = null
  let controllerChart: echarts.ECharts | null = null

  const showApiServerDetail = ref(false)
  const showSchedulerDetail = ref(false)
  const showControllerDetail = ref(false)

  const toggleApiServerDetail = () => (showApiServerDetail.value = !showApiServerDetail.value)
  const toggleSchedulerDetail = () => (showSchedulerDetail.value = !showSchedulerDetail.value)
  const toggleControllerDetail = () => (showControllerDetail.value = !showControllerDetail.value)

  const apiServer = computed(() => props.controlPlaneData?.apiServer)
  const scheduler = computed(() => props.controlPlaneData?.scheduler)
  const controllerManager = computed(() => props.controlPlaneData?.controllerManager)

  const hasData = computed(
    () => !!(props.controlPlaneData && Object.keys(props.controlPlaneData).length > 0)
  )

  const formatNumber = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0'
    return value.toFixed(2)
  }

  const formatPercent = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0.00'
    return value.toFixed(2)
  }

  const formatInteger = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0'
    return Math.round(value).toString()
  }

  const formatLatency = (value?: number): string => {
    if (value === undefined || value === null || isNaN(value)) return '0 ms'
    const ms = value * 1000
    if (ms < 1) return `${(ms * 1000).toFixed(2)} µs`
    if (ms < 1000) return `${ms.toFixed(2)} ms`
    return `${(ms / 1000).toFixed(2)} s`
  }

  const formatBytes = (bytes?: number): string => {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  const overallHealth = computed(() => {
    if (!props.controlPlaneData) return { status: 'unknown', text: '未知', score: 0 }
    let score = 100
    if (apiServer.value) {
      if (apiServer.value.errorRate > 0.05) score -= 20
      else if (apiServer.value.errorRate > 0.01) score -= 10
      if (apiServer.value.requestDropped > 0) score -= 15
      if (apiServer.value.currentInflightRequests > 600) score -= 10
    }
    if (scheduler.value) {
      if (scheduler.value.scheduleSuccessRate < 0.8) score -= 25
      else if (scheduler.value.scheduleSuccessRate < 0.95) score -= 10
      if (scheduler.value.unschedulablePods > 10) score -= 15
      else if (scheduler.value.unschedulablePods > 0) score -= 5
    }
    if (controllerManager.value) {
      if (controllerManager.value.totalSyncErrors > 10) score -= 15
      if (controllerManager.value.workQueueMetrics?.depthTotal > 1000) score -= 10
    }
    score = Math.max(0, score)
    let status: 'healthy' | 'warning' | 'critical' | 'unknown' = 'healthy'
    let text = '健康'
    if (score < 60) {
      status = 'critical'
      text = '严重异常'
    } else if (score < 80) {
      status = 'warning'
      text = '需要关注'
    }
    return { status, text, score: Math.round(score) }
  })

  const healthIssues = computed(() => {
    const issues: string[] = []
    if (apiServer.value) {
      if (apiServer.value.errorRate > 0.05) issues.push('API Server 错误率超过 5%')
      if (apiServer.value.requestDropped > 0)
        issues.push(`API Server 丢弃了 ${apiServer.value.requestDropped} 个请求`)
      if (apiServer.value.requestTimeout > 0)
        issues.push(`API Server 有 ${apiServer.value.requestTimeout} 个超时请求`)
    }
    if (scheduler.value) {
      if (scheduler.value.scheduleSuccessRate < 0.8) issues.push('调度成功率低于 80%')
      if (scheduler.value.unschedulablePods > 10)
        issues.push(`有 ${scheduler.value.unschedulablePods} 个 Pod 无法调度`)
    }
    if (controllerManager.value) {
      if (controllerManager.value.totalSyncErrors > 10)
        issues.push(`Controller Manager 同步错误: ${controllerManager.value.totalSyncErrors}`)
    }
    return issues
  })

  const getTotalQueueDepth = (): number => {
    const cm = controllerManager.value
    if (!cm) return 0
    return (
      (cm.deploymentQueueDepth || 0) +
      (cm.replicaSetQueueDepth || 0) +
      (cm.statefulSetQueueDepth || 0) +
      (cm.daemonSetQueueDepth || 0) +
      (cm.jobQueueDepth || 0) +
      (cm.nodeQueueDepth || 0) +
      (cm.serviceQueueDepth || 0) +
      (cm.endpointQueueDepth || 0)
    )
  }

  const apiServerStatus = computed(() => {
    const data = apiServer.value
    if (!data) return { class: 'unknown', text: '未知' }
    if (data.errorRate > 0.05 || data.requestDropped > 0 || data.requestTimeout > 0)
      return { class: 'danger', text: '异常' }
    if (data.errorRate > 0.01 || data.p95Latency > 1 || data.currentInflightRequests > 600)
      return { class: 'warning', text: '需关注' }
    return { class: 'success', text: '正常' }
  })

  const schedulerStatus = computed(() => {
    const data = scheduler.value
    if (!data) return { class: 'unknown', text: '未知' }
    if (data.scheduleSuccessRate < 0.8 || data.unschedulablePods > 10)
      return { class: 'danger', text: '异常' }
    if (data.scheduleSuccessRate < 0.95 || data.pendingPods > 10)
      return { class: 'warning', text: '需关注' }
    return { class: 'success', text: '正常' }
  })

  const controllerStatus = computed(() => {
    const data = controllerManager.value
    if (!data) return { class: 'unknown', text: '未知' }
    if (data.totalSyncErrors > 10) return { class: 'danger', text: '异常' }
    if (data.workQueueMetrics?.depthTotal > 1000 || data.retryRate > 10)
      return { class: 'warning', text: '需关注' }
    return { class: 'success', text: data.isLeader ? '正常 (Leader)' : '正常 (Standby)' }
  })

  const getQPSLevel = (qps?: number) => {
    if (!qps) return 'low'
    if (qps < 100) return 'low'
    if (qps < 500) return 'normal'
    if (qps < 1000) return 'high'
    return 'very-high'
  }
  const getQPSLevelText = (qps?: number) => {
    if (!qps) return '低负载'
    if (qps < 100) return '低负载'
    if (qps < 500) return '正常'
    if (qps < 1000) return '高负载'
    return '极高负载'
  }
  const getSuccessRateLevel = (rate?: number) => {
    if (!rate) return 'critical'
    if (rate > 0.95) return 'excellent'
    if (rate > 0.9) return 'good'
    if (rate > 0.8) return 'warning'
    return 'critical'
  }
  const getSuccessRateText = (rate?: number) => {
    if (!rate) return '异常'
    if (rate > 0.95) return '优秀'
    if (rate > 0.9) return '良好'
    if (rate > 0.8) return '需改进'
    return '异常'
  }
  const getConcurrencyPercent = (current?: number): number => {
    if (!current) return 0
    return Math.min((current / 600) * 100, 100)
  }

  const initAPIServerChart = () => {
    if (!apiServerChartRef.value) return
    apiServerChart = echarts.init(apiServerChartRef.value)
    apiServerChart.setOption({
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 }
      },
      grid: { left: '3%', right: '4%', bottom: '8%', top: '12%', containLabel: true },
      xAxis: {
        type: 'time',
        boundaryGap: false,
        axisLine: { lineStyle: { color: '#e5e7eb' } },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (v: number) =>
            new Date(v).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
        },
        splitLine: { show: true, lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      yAxis: [
        {
          type: 'value',
          name: 'QPS',
          position: 'left',
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: { color: '#6b7280', fontSize: 11 },
          splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
        },
        {
          type: 'value',
          name: '错误率 % / 延迟 ms',
          position: 'right',
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: { color: '#6b7280', fontSize: 11 },
          splitLine: { show: false }
        }
      ],
      series: []
    })
  }

  const updateAPIServerChart = () => {
    if (!apiServerChart || !apiServer.value?.trend) return
    const trend = apiServer.value.trend
    if (trend.length === 0) return
    apiServerChart.setOption({
      series: [
        {
          name: 'QPS',
          type: 'line',
          yAxisIndex: 0,
          data: trend.map((i) => [i.timestamp, i.requestsPerSecond || 0]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#3b82f6' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(59,130,246,0.2)' },
              { offset: 1, color: 'rgba(59,130,246,0.02)' }
            ])
          }
        },
        {
          name: '错误率',
          type: 'line',
          yAxisIndex: 1,
          data: trend.map((i) => [i.timestamp, (i.errorRate || 0) * 100]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#ef4444' }
        },
        {
          name: 'P95延迟',
          type: 'line',
          yAxisIndex: 1,
          data: trend.map((i) => [i.timestamp, (i.p95Latency || 0) * 1000]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#10b981' }
        }
      ]
    })
  }

  const initSchedulerChart = () => {
    if (!schedulerChartRef.value) return
    schedulerChart = echarts.init(schedulerChartRef.value)
    schedulerChart.setOption({
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1
      },
      grid: { left: '3%', right: '4%', bottom: '8%', top: '12%', containLabel: true },
      xAxis: {
        type: 'time',
        boundaryGap: false,
        axisLine: { lineStyle: { color: '#e5e7eb' } },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (v: number) =>
            new Date(v).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
        },
        splitLine: { show: true, lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      yAxis: [
        {
          type: 'value',
          name: '成功率 %',
          position: 'left',
          max: 100,
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: { color: '#6b7280', fontSize: 11 },
          splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
        },
        {
          type: 'value',
          name: '数量 / 延迟 ms',
          position: 'right',
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: { color: '#6b7280', fontSize: 11 },
          splitLine: { show: false }
        }
      ],
      series: []
    })
  }

  const updateSchedulerChart = () => {
    if (!schedulerChart || !scheduler.value?.trend) return
    const trend = scheduler.value.trend
    if (trend.length === 0) return
    schedulerChart.setOption({
      series: [
        {
          name: '成功率',
          type: 'line',
          yAxisIndex: 0,
          data: trend.map((i) => [i.timestamp, (i.scheduleSuccessRate || 0) * 100]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#10b981' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(16,185,129,0.2)' },
              { offset: 1, color: 'rgba(16,185,129,0.02)' }
            ])
          }
        },
        {
          name: '待调度',
          type: 'line',
          yAxisIndex: 1,
          data: trend.map((i) => [i.timestamp, i.pendingPods || 0]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#f59e0b' }
        },
        {
          name: 'P95延迟',
          type: 'line',
          yAxisIndex: 1,
          data: trend.map((i) => [i.timestamp, (i.p95ScheduleLatency || 0) * 1000]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#3b82f6' }
        }
      ]
    })
  }

  const initControllerChart = () => {
    if (!controllerChartRef.value) return
    controllerChart = echarts.init(controllerChartRef.value)
    controllerChart.setOption({
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1
      },
      grid: { left: '3%', right: '4%', bottom: '8%', top: '12%', containLabel: true },
      xAxis: {
        type: 'time',
        boundaryGap: false,
        axisLine: { lineStyle: { color: '#e5e7eb' } },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (v: number) =>
            new Date(v).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
        },
        splitLine: { show: true, lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      yAxis: {
        type: 'value',
        name: '队列深度',
        nameTextStyle: { color: '#6b7280', fontSize: 11 },
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: '#6b7280', fontSize: 11 },
        splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      series: []
    })
  }

  const updateControllerChart = () => {
    if (!controllerChart || !controllerManager.value?.trend) return
    const trend = controllerManager.value.trend
    if (trend.length === 0) return
    controllerChart.setOption({
      series: [
        {
          name: 'Deployment',
          type: 'line',
          data: trend.map((i) => [i.timestamp, i.deploymentQueueDepth || 0]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#3b82f6' }
        },
        {
          name: 'ReplicaSet',
          type: 'line',
          data: trend.map((i) => [i.timestamp, i.replicaSetQueueDepth || 0]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#10b981' }
        },
        {
          name: 'StatefulSet',
          type: 'line',
          data: trend.map((i) => [i.timestamp, i.statefulSetQueueDepth || 0]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#f59e0b' }
        },
        {
          name: '总深度',
          type: 'line',
          data: trend.map((i) => [i.timestamp, i.totalQueueDepth || 0]),
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#ef4444', type: 'dashed' }
        }
      ]
    })
  }

  const handleResize = () => {
    apiServerChart?.resize()
    schedulerChart?.resize()
    controllerChart?.resize()
  }

  watch(
    () => props.controlPlaneData,
    () => {
      nextTick(() => {
        if (hasData.value) {
          if (!apiServerChart) initAPIServerChart()
          if (!schedulerChart) initSchedulerChart()
          if (!controllerChart) initControllerChart()
          updateAPIServerChart()
          updateSchedulerChart()
          updateControllerChart()
        }
      })
    },
    { deep: true, immediate: true }
  )

  onMounted(() => {
    window.addEventListener('resize', handleResize)
  })
  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    apiServerChart?.dispose()
    schedulerChart?.dispose()
    controllerChart?.dispose()
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

  .control-plane-monitor {
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
        margin-bottom: 20px;
        padding-bottom: 16px;
        border-bottom: 1px solid $gray-200;
        .header-left {
          display: flex;
          align-items: center;
          gap: 14px;
          .header-icon {
            width: 44px;
            height: 44px;
            border-radius: $radius-sm;
            background: linear-gradient(135deg, #dbeafe, #bfdbfe);
            display: flex;
            align-items: center;
            justify-content: center;
            color: $primary;
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
          .health-score {
            margin-left: 6px;
            font-weight: 700;
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
        }
      }
      .issues-alert {
        background: rgba($danger, 0.05);
        border: 1px solid rgba($danger, 0.2);
        border-radius: $radius-sm;
        padding: 14px 18px;
        margin-bottom: 20px;
        .alert-header {
          display: flex;
          align-items: center;
          gap: 10px;
          margin-bottom: 10px;
          font-weight: 600;
          font-size: 14px;
          color: $danger;
          svg {
            color: $danger;
          }
        }
        .issues-list {
          display: flex;
          flex-direction: column;
          gap: 6px;
          .issue-item {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 13px;
            color: $gray-600;
            .issue-dot {
              width: 4px;
              height: 4px;
              border-radius: 50%;
              background: $danger;
            }
          }
        }
      }
      .quick-metrics {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 16px;
        @media (max-width: 768px) {
          grid-template-columns: 1fr;
        }
        .quick-metric {
          display: flex;
          align-items: center;
          gap: 14px;
          padding: 18px;
          border-radius: $radius-sm;
          background: $gray-50;
          border: 1px solid $gray-200;
          transition: all 0.2s ease;
          &:hover {
            border-color: $gray-300;
          }
          .metric-icon {
            width: 48px;
            height: 48px;
            border-radius: $radius-xs;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;
            &.api {
              background: linear-gradient(135deg, #dbeafe, #bfdbfe);
              color: $primary;
            }
            &.scheduler {
              background: linear-gradient(135deg, #fef3c7, #fde68a);
              color: $warning;
            }
            &.controller {
              background: linear-gradient(135deg, #e5e7eb, #d1d5db);
              color: $gray-600;
            }
          }
          .metric-info {
            .metric-label {
              font-size: 12px;
              color: $gray-500;
              margin-bottom: 4px;
              display: block;
            }
            .metric-value {
              font-size: 20px;
              font-weight: 700;
              color: $gray-800;
              display: block;
            }
            .metric-sub {
              font-size: 11px;
              color: $gray-400;
              margin-top: 2px;
              display: block;
            }
          }
        }
      }
    }

    .monitor-card {
      background: #fff;
      border-radius: $radius;
      padding: 24px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 24px;
        padding-bottom: 16px;
        border-bottom: 1px solid $gray-200;
        .header-info {
          display: flex;
          align-items: center;
          gap: 14px;
          .icon-badge {
            width: 46px;
            height: 46px;
            border-radius: $radius-sm;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fff;
            &.api-server {
              background: linear-gradient(135deg, $primary, #2563eb);
            }
            &.scheduler {
              background: linear-gradient(135deg, $warning, #d97706);
            }
            &.controller {
              background: linear-gradient(135deg, $gray-500, $gray-600);
            }
          }
          h3 {
            margin: 0;
            font-size: 16px;
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
          padding: 6px 14px;
          border-radius: 16px;
          font-size: 12px;
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
          &.unknown {
            background: $gray-100;
            color: $gray-500;
          }
        }
      }

      .metrics-grid {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 14px;
        margin-bottom: 20px;
        @media (max-width: 1200px) {
          grid-template-columns: repeat(2, 1fr);
        }
        @media (max-width: 640px) {
          grid-template-columns: 1fr;
        }
      }

      .metric-box {
        padding: 16px;
        border-radius: $radius-sm;
        background: $gray-50;
        border: 1px solid $gray-200;
        transition: all 0.2s ease;
        &:hover {
          border-color: $gray-300;
        }
        &.primary {
          background: rgba($primary, 0.04);
          border-color: rgba($primary, 0.15);
        }
        &.warning {
          background: rgba($warning, 0.06);
          border-color: rgba($warning, 0.25);
        }
        &.danger {
          background: rgba($danger, 0.06);
          border-color: rgba($danger, 0.25);
        }
        .box-header {
          display: flex;
          align-items: center;
          gap: 6px;
          margin-bottom: 10px;
          color: $gray-500;
          span {
            font-size: 12px;
            font-weight: 500;
          }
        }
        .box-value {
          display: flex;
          align-items: baseline;
          gap: 4px;
          .value {
            font-size: 22px;
            font-weight: 700;
            color: $gray-800;
            &.danger {
              color: $danger;
            }
          }
          .unit {
            font-size: 12px;
            color: $gray-400;
          }
        }
        .box-badge {
          display: inline-block;
          padding: 4px 10px;
          border-radius: 12px;
          font-size: 11px;
          font-weight: 600;
          margin-top: 8px;
          &.low {
            background: rgba($primary, 0.1);
            color: #2563eb;
          }
          &.normal {
            background: rgba($success, 0.1);
            color: #059669;
          }
          &.high {
            background: rgba($warning, 0.1);
            color: #d97706;
          }
          &.very-high {
            background: rgba($danger, 0.1);
            color: #dc2626;
          }
          &.excellent {
            background: rgba($success, 0.1);
            color: #059669;
          }
          &.good {
            background: rgba($primary, 0.1);
            color: #2563eb;
          }
          &.warning {
            background: rgba($warning, 0.1);
            color: #d97706;
          }
          &.critical {
            background: rgba($danger, 0.1);
            color: #dc2626;
          }
        }
        .box-progress {
          margin-top: 10px;
          height: 4px;
          background: $gray-200;
          border-radius: 2px;
          overflow: hidden;
          .progress-fill {
            height: 100%;
            background: $primary;
            transition: width 0.4s ease;
          }
          &.success .progress-fill {
            background: $success;
          }
        }
      }

      .latency-bar {
        padding: 14px 18px;
        background: $gray-50;
        border-radius: $radius-sm;
        margin-bottom: 20px;
        .bar-title {
          font-size: 12px;
          font-weight: 500;
          color: $gray-600;
          margin-bottom: 12px;
        }
        .bar-items {
          display: flex;
          gap: 24px;
          .bar-item {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 4px;
            .label {
              font-size: 11px;
              color: $gray-400;
            }
            .value {
              font-size: 14px;
              font-weight: 600;
              color: $gray-700;
            }
          }
        }
      }

      .stats-bar {
        display: flex;
        gap: 12px;
        margin-bottom: 20px;
        .stats-item {
          flex: 1;
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 4px;
          padding: 12px;
          background: $gray-50;
          border-radius: $radius-xs;
          border: 1px solid $gray-200;
          &.success {
            border-color: rgba($success, 0.3);
          }
          &.danger {
            border-color: rgba($danger, 0.3);
          }
          &.warning {
            border-color: rgba($warning, 0.3);
          }
          &.info {
            border-color: rgba($primary, 0.3);
          }
          .stats-label {
            font-size: 11px;
            color: $gray-500;
          }
          .stats-value {
            font-size: 16px;
            font-weight: 700;
            color: $gray-800;
          }
        }
      }

      .leader-status {
        display: flex;
        align-items: center;
        gap: 14px;
        padding: 16px 20px;
        background: $gray-50;
        border-radius: $radius-sm;
        border: 1px solid $gray-200;
        margin-bottom: 20px;
        &.active {
          background: rgba($warning, 0.06);
          border-color: rgba($warning, 0.25);
          .leader-icon svg {
            color: $warning;
          }
        }
        .leader-icon svg {
          color: $gray-400;
        }
        .leader-info {
          .leader-title {
            font-size: 14px;
            font-weight: 600;
            color: $gray-800;
            margin-bottom: 2px;
            display: block;
          }
          .leader-sub {
            font-size: 12px;
            color: $gray-500;
            display: block;
          }
        }
      }

      .queue-section {
        margin-bottom: 20px;
        .queue-title {
          font-size: 13px;
          font-weight: 600;
          color: $gray-600;
          margin-bottom: 14px;
        }
        .queue-grid {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
          gap: 12px;
          @media (max-width: 1200px) {
            grid-template-columns: repeat(2, 1fr);
          }
          .queue-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 14px;
            background: $gray-50;
            border-radius: $radius-xs;
            border: 1px solid $gray-200;
            transition: all 0.2s ease;
            &:hover {
              border-color: $gray-300;
            }
            .queue-icon {
              width: 36px;
              height: 36px;
              border-radius: $radius-xs;
              background: #fff;
              display: flex;
              align-items: center;
              justify-content: center;
              color: $gray-500;
            }
            .queue-info {
              .queue-name {
                font-size: 11px;
                color: $gray-500;
                margin-bottom: 2px;
                display: block;
              }
              .queue-value {
                font-size: 16px;
                font-weight: 700;
                color: $gray-800;
              }
            }
          }
        }
      }

      .chart-area {
        padding: 18px;
        background: $gray-50;
        border-radius: $radius-sm;
        margin-bottom: 16px;
        .chart-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 14px;
          .chart-title {
            font-size: 13px;
            font-weight: 600;
            color: $gray-600;
          }
          .chart-legend {
            display: flex;
            gap: 14px;
            .legend-item {
              display: flex;
              align-items: center;
              gap: 5px;
              font-size: 11px;
              color: $gray-500;
              .dot {
                width: 8px;
                height: 3px;
                border-radius: 2px;
                &.qps,
                &.deployment {
                  background: $primary;
                }
                &.error,
                &.total {
                  background: $danger;
                }
                &.latency,
                &.replicaset {
                  background: $success;
                }
                &.success {
                  background: $success;
                }
                &.pending,
                &.statefulset {
                  background: $warning;
                }
              }
            }
          }
        }
        .chart-container {
          height: 240px;
          width: 100%;
        }
      }

      .detail-panel {
        border-top: 1px solid $gray-200;
        padding-top: 14px;
        .panel-toggle {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 12px 16px;
          background: $gray-50;
          border-radius: $radius-xs;
          cursor: pointer;
          transition: all 0.2s ease;
          user-select: none;
          &:hover {
            background: $gray-100;
          }
          span {
            font-size: 12px;
            font-weight: 600;
            color: $gray-600;
          }
          svg {
            color: $gray-400;
            transition: transform 0.3s ease;
            &.rotated {
              transform: rotate(180deg);
            }
          }
        }
        .panel-content {
          padding: 16px 0;
          .detail-section {
            margin-bottom: 18px;
            &:last-child {
              margin-bottom: 0;
            }
            .section-title {
              font-size: 12px;
              font-weight: 600;
              color: $gray-600;
              margin-bottom: 12px;
              padding-bottom: 8px;
              border-bottom: 1px solid $gray-200;
            }
            .detail-grid {
              display: grid;
              grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
              gap: 10px;
              .detail-item {
                display: flex;
                justify-content: space-between;
                align-items: center;
                padding: 10px 14px;
                background: $gray-50;
                border-radius: $radius-xs;
                .label {
                  font-size: 11px;
                  color: $gray-500;
                }
                .value {
                  font-size: 13px;
                  font-weight: 600;
                  color: $gray-700;
                  &.danger {
                    color: $danger;
                  }
                  &.warning {
                    color: $warning;
                  }
                }
              }
            }
          }
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
  .slide-enter-active,
  .slide-leave-active {
    transition: all 0.3s ease;
    max-height: 2000px;
    overflow: hidden;
  }
  .slide-enter-from,
  .slide-leave-to {
    max-height: 0;
    opacity: 0;
  }
</style>

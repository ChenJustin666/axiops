<template>
  <div class="hpa-management-container">
    <!-- 加载状态 -->
    <div v-if="initialLoading" v-loading="initialLoading" class="loading-wrapper">
      <div style="height: 400px"></div>
    </div>

    <!-- HPA 不存在 - 创建表单 -->
    <div v-else-if="!hpaDetail" class="content-layout">
      <!-- 左侧：表单配置 -->
      <div class="form-section">
        <!-- 模式切换 -->
        <div class="mode-switch">
          <ElRadioGroup v-model="editMode" size="default">
            <ElRadioButton label="form">
              <Edit :size="14" />
              表单配置
            </ElRadioButton>
            <ElRadioButton label="yaml">
              <FileText :size="14" />
              YAML 配置
            </ElRadioButton>
          </ElRadioGroup>
        </div>

        <!-- 表单模式 -->
        <div v-show="editMode === 'form'" class="form-content">
          <ElForm
            ref="formRef"
            :model="formData"
            :rules="formRules"
            label-width="140px"
            label-position="left"
          >
            <!-- 基础配置 -->
            <div class="section-header">
              <Database :size="16" />
              <span>基础配置</span>
            </div>

            <ElFormItem label="HPA 名称" prop="name">
              <template #label>
                <span>HPA 名称</span>
                <ElTooltip content="HPA 资源的名称，创建后不可修改" placement="top">
                  <HelpCircle :size="14" class="label-help-icon" />
                </ElTooltip>
              </template>
              <ElInput v-model="formData.name" placeholder="自动生成，可自定义" clearable />
            </ElFormItem>

            <ElFormItem label="命名空间">
              <template #label>
                <span>命名空间</span>
                <ElTooltip content="HPA 所在的命名空间，不可修改" placement="top">
                  <HelpCircle :size="14" class="label-help-icon" />
                </ElTooltip>
              </template>
              <ElInput :value="workspace?.namespace" disabled />
            </ElFormItem>

            <ElFormItem label="目标资源">
              <template #label>
                <span>目标资源</span>
                <ElTooltip content="HPA 控制的工作负载资源，不可修改" placement="top">
                  <HelpCircle :size="14" class="label-help-icon" />
                </ElTooltip>
              </template>
              <ElInput :value="targetResourceDisplay" disabled>
                <template #prepend>{{ formData.targetRef.kind }}</template>
              </ElInput>
            </ElFormItem>

            <ElFormItem label="副本数范围" required>
              <template #label>
                <span>副本数范围</span>
                <ElTooltip
                  content="HPA 自动扩缩容的副本数范围。最小值至少为 1，最大值不超过 100"
                  placement="top"
                >
                  <HelpCircle :size="14" class="label-help-icon" />
                </ElTooltip>
              </template>
              <div class="range-inputs">
                <ElFormItem prop="minReplicas" style="margin-bottom: 0">
                  <ElInputNumber
                    v-model="formData.minReplicas"
                    :min="1"
                    :max="99"
                    placeholder="最小"
                  />
                </ElFormItem>
                <span class="range-separator">-</span>
                <ElFormItem prop="maxReplicas" style="margin-bottom: 0">
                  <ElInputNumber
                    v-model="formData.maxReplicas"
                    :min="2"
                    :max="100"
                    placeholder="最大"
                  />
                </ElFormItem>
              </div>
            </ElFormItem>

            <!-- 指标配置 -->
            <div class="section-header">
              <BarChart2 :size="16" />
              <span>指标配置</span>
            </div>

            <!-- CPU 指标 -->
            <ElFormItem label="CPU 指标">
              <template #label>
                <span>CPU 指标</span>
                <ElTooltip content="基于 CPU 使用率或使用量进行扩缩容" placement="top">
                  <HelpCircle :size="14" class="label-help-icon" />
                </ElTooltip>
              </template>
              <div class="metric-config">
                <ElSwitch v-model="formData.enableCpu" />
                <template v-if="formData.enableCpu">
                  <ElSelect
                    v-model="formData.cpuTargetType"
                    placeholder="类型"
                    style="width: 140px"
                  >
                    <ElOption label="使用率 (%)" value="Utilization" />
                    <ElOption label="平均值" value="AverageValue" />
                  </ElSelect>
                  <ElInputNumber
                    v-if="formData.cpuTargetType === 'Utilization'"
                    v-model="formData.cpuUtilization"
                    :min="1"
                    :max="100"
                    placeholder="目标值"
                    style="width: 150px"
                  />
                  <template v-else>
                    <ElInputNumber
                      v-model="formData.cpuAverageValueNum"
                      :min="0"
                      placeholder="数值"
                      style="width: 150px"
                    />
                    <ElSelect v-model="formData.cpuAverageValueUnit" style="width: 80px">
                      <ElOption label="核" value="" />
                      <ElOption label="毫核" value="m" />
                    </ElSelect>
                  </template>
                  <span v-if="formData.cpuTargetType === 'Utilization'" class="unit-text">%</span>
                </template>
              </div>
            </ElFormItem>

            <!-- 内存指标 -->
            <ElFormItem label="内存指标">
              <template #label>
                <span>内存指标</span>
                <ElTooltip content="基于内存使用率或使用量进行扩缩容" placement="top">
                  <HelpCircle :size="14" class="label-help-icon" />
                </ElTooltip>
              </template>
              <div class="metric-config">
                <ElSwitch v-model="formData.enableMemory" />
                <template v-if="formData.enableMemory">
                  <ElSelect
                    v-model="formData.memoryTargetType"
                    placeholder="类型"
                    style="width: 140px"
                  >
                    <ElOption label="使用率 (%)" value="Utilization" />
                    <ElOption label="平均值" value="AverageValue" />
                  </ElSelect>
                  <ElInputNumber
                    v-if="formData.memoryTargetType === 'Utilization'"
                    v-model="formData.memoryUtilization"
                    :min="1"
                    :max="100"
                    placeholder="目标值"
                    style="width: 150px"
                  />
                  <template v-else>
                    <ElInputNumber
                      v-model="formData.memoryAverageValueNum"
                      :min="0"
                      placeholder="数值"
                      style="width: 150px"
                    />
                    <ElSelect v-model="formData.memoryAverageValueUnit" style="width: 80px">
                      <ElOption label="Ki" value="Ki" />
                      <ElOption label="Mi" value="Mi" />
                      <ElOption label="Gi" value="Gi" />
                    </ElSelect>
                  </template>
                  <span v-if="formData.memoryTargetType === 'Utilization'" class="unit-text"
                    >%</span
                  >
                </template>
              </div>
            </ElFormItem>

            <!-- 扩缩容行为 -->
            <div class="section-header">
              <Settings :size="16" />
              <span>扩缩容行为（可选）</span>
            </div>

            <ElFormItem label="启用行为控制">
              <template #label>
                <span>启用行为控制</span>
                <ElTooltip content="控制扩缩容的速度和策略，避免频繁扩缩容" placement="top">
                  <HelpCircle :size="14" class="label-help-icon" />
                </ElTooltip>
              </template>
              <ElSwitch v-model="formData.enableBehavior" />
            </ElFormItem>

            <template v-if="formData.enableBehavior">
              <ElFormItem label="扩容稳定窗口">
                <template #label>
                  <span>扩容稳定窗口</span>
                  <ElTooltip
                    content="扩容前等待的时间（秒）。在此期间，如果指标恢复正常，则不会扩容"
                    placement="top"
                  >
                    <HelpCircle :size="14" class="label-help-icon" />
                  </ElTooltip>
                </template>
                <ElInputNumber
                  v-model="formData.scaleUpStabilizationWindowSeconds"
                  :min="0"
                  :max="3600"
                  :step="10"
                  style="width: 150px"
                >
                  <template #append>秒</template>
                </ElInputNumber>
              </ElFormItem>

              <ElFormItem label="缩容稳定窗口">
                <template #label>
                  <span>缩容稳定窗口</span>
                  <ElTooltip
                    content="缩容前等待的时间（秒）。建议设置较大值（如 300 秒）避免频繁缩容"
                    placement="top"
                  >
                    <HelpCircle :size="14" class="label-help-icon" />
                  </ElTooltip>
                </template>
                <ElInputNumber
                  v-model="formData.scaleDownStabilizationWindowSeconds"
                  :min="0"
                  :max="3600"
                  :step="10"
                  style="width: 150px"
                >
                  <template #append>秒</template>
                </ElInputNumber>
              </ElFormItem>
            </template>

            <!-- 操作按钮 -->
            <ElFormItem>
              <div class="form-actions">
                <ElButton type="primary" size="large" :loading="submitting" @click="handleSubmit">
                  <Check :size="16" v-if="!submitting" />
                  {{ submitting ? '创建中...' : '创建 HPA' }}
                </ElButton>
                <ElButton size="large" @click="handleReset">
                  <RotateCcw :size="16" />
                  重置
                </ElButton>
              </div>
            </ElFormItem>
          </ElForm>
        </div>

        <!-- YAML 模式 -->
        <div v-show="editMode === 'yaml'" class="yaml-content">
          <YamlEditorPro
            v-model="yamlContent"
            height="500px"
            :filename="`${formData.name}.yaml`"
            :readonly="false"
            :show-toolbar="true"
            :show-line-numbers="true"
            :show-status-bar="true"
            @change="handleYamlChange"
          />

          <div class="yaml-actions">
            <ElButton type="primary" size="large" :loading="submitting" @click="handleSubmit">
              <Check :size="16" v-if="!submitting" />
              {{ submitting ? '创建中...' : '创建 HPA' }}
            </ElButton>
            <ElButton size="large" @click="syncYamlToForm">
              <Upload :size="16" />
              导入到表单
            </ElButton>
          </div>
        </div>
      </div>

      <!-- 右侧：配置建议 -->
      <div class="info-section">
        <!-- 创建提示 -->
        <ElAlert type="info" :closable="false" show-icon class="create-alert">
          <template #title>
            <div class="alert-content">
              <TrendingUp :size="16" />
              <span>暂未配置 HPA</span>
            </div>
          </template>
          <template #default>
            <p>HPA 可根据 CPU/内存等指标自动调整 Pod 副本数，实现水平自动扩缩容。</p>
          </template>
        </ElAlert>

        <div class="info-card">
          <div class="info-title">
            <Info :size="16" />
            <span>配置建议</span>
          </div>
          <div class="info-content">
            <div class="info-item">
              <div class="info-item-header">
                <CheckCircle :size="14" class="icon-success" />
                <strong>前置要求</strong>
              </div>
              <ul>
                <li>确保集群已安装 <strong>Metrics Server</strong></li>
                <li>确保 Pod 已设置 <strong>resources.requests</strong></li>
              </ul>
            </div>

            <div class="info-item">
              <div class="info-item-header">
                <CheckCircle :size="14" class="icon-success" />
                <strong>指标设置</strong>
              </div>
              <ul>
                <li>CPU 利用率建议设置为 <strong>50%-80%</strong></li>
                <li>内存利用率建议设置为 <strong>60%-80%</strong></li>
                <li>CPU 平均值单位：核（如 0.5）或毫核（如 500m）</li>
                <li>内存平均值单位：Ki、Mi、Gi（如 1Gi）</li>
                <li>至少启用一个指标（CPU 或内存）</li>
              </ul>
            </div>

            <div class="info-item">
              <div class="info-item-header">
                <CheckCircle :size="14" class="icon-info" />
                <strong>副本数范围</strong>
              </div>
              <ul>
                <li>最小副本数至少为 <strong>1</strong></li>
                <li>最大副本数根据实际资源情况设置</li>
                <li>生产环境建议最小 <strong>2 个副本</strong>保证高可用</li>
              </ul>
            </div>

            <div class="info-item warning">
              <div class="info-item-header">
                <AlertTriangle :size="14" class="icon-warning" />
                <strong>重要提示</strong>
              </div>
              <ul>
                <li><strong>HPA 会覆盖手动设置的副本数</strong>，请谨慎配置</li>
                <li>不要同时使用 <strong>HPA 和 VPA 的 Auto 模式</strong></li>
                <li>建议设置合理的<strong>稳定窗口</strong>避免频繁扩缩容</li>
                <li>扩缩容会有一定延迟（默认 15 秒采集一次指标）</li>
              </ul>
            </div>

            <div class="info-item">
              <div class="info-item-header">
                <Lightbulb :size="14" class="icon-info" />
                <strong>最佳实践</strong>
              </div>
              <ul>
                <li>优先使用 <strong>CPU 利用率</strong>作为扩缩容指标</li>
                <li>缩容稳定窗口建议设置为 <strong>300 秒</strong></li>
                <li>扩容稳定窗口可以设置为 <strong>0 秒</strong>快速响应</li>
                <li>定期检查 HPA 状态和推荐值</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- HPA 已存在 - 详情和编辑 -->
    <div v-else class="content-layout">
      <!-- 左侧：详情或编辑表单 -->
      <div class="form-section">
        <!-- 状态标题栏 -->
        <div class="status-bar">
          <div class="status-left">
            <ElTag type="success" size="large" effect="dark">
              <Activity :size="14" />
              HPA 已启用
            </ElTag>
          </div>
          <div class="status-right">
            <ElButton v-if="!editing" :icon="Edit" size="default" @click="startEdit">
              编辑配置
            </ElButton>
            <ElButton :icon="FileText" size="default" @click="viewYaml"> 查看 YAML </ElButton>
            <ElButton
              type="danger"
              :icon="Trash2"
              size="default"
              :loading="deleting"
              @click="handleDelete"
            >
              删除 HPA
            </ElButton>
          </div>
        </div>

        <!-- 编辑模式 -->
        <template v-if="editing">
          <!-- 模式切换 -->
          <div class="mode-switch">
            <ElRadioGroup v-model="editMode" size="default">
              <ElRadioButton label="form">
                <Edit :size="14" />
                表单编辑
              </ElRadioButton>
              <ElRadioButton label="yaml">
                <FileText :size="14" />
                YAML 编辑
              </ElRadioButton>
            </ElRadioGroup>
          </div>

          <!-- 表单编辑 -->
          <div v-show="editMode === 'form'" class="form-content">
            <ElForm
              ref="formRef"
              :model="formData"
              :rules="formRules"
              label-width="140px"
              label-position="left"
            >
              <!-- 基础配置 -->
              <div class="section-header">
                <Database :size="16" />
                <span>基础配置</span>
              </div>

              <ElFormItem label="HPA 名称" prop="name">
                <ElInput v-model="formData.name" disabled />
              </ElFormItem>

              <ElFormItem label="命名空间">
                <ElInput :value="workspace?.namespace" disabled />
              </ElFormItem>

              <ElFormItem label="目标资源">
                <ElInput :value="targetResourceDisplay" disabled>
                  <template #prepend>{{ formData.targetRef.kind }}</template>
                </ElInput>
              </ElFormItem>

              <ElFormItem label="副本数范围" required>
                <div class="range-inputs">
                  <ElFormItem prop="minReplicas" style="margin-bottom: 0">
                    <ElInputNumber v-model="formData.minReplicas" :min="1" :max="99" />
                  </ElFormItem>
                  <span class="range-separator">-</span>
                  <ElFormItem prop="maxReplicas" style="margin-bottom: 0">
                    <ElInputNumber v-model="formData.maxReplicas" :min="2" :max="100" />
                  </ElFormItem>
                </div>
              </ElFormItem>

              <!-- 指标配置 -->
              <div class="section-header">
                <BarChart2 :size="16" />
                <span>指标配置</span>
              </div>

              <ElFormItem label="CPU 指标">
                <div class="metric-config">
                  <ElSwitch v-model="formData.enableCpu" />
                  <template v-if="formData.enableCpu">
                    <ElSelect v-model="formData.cpuTargetType" style="width: 140px">
                      <ElOption label="使用率 (%)" value="Utilization" />
                      <ElOption label="平均值" value="AverageValue" />
                    </ElSelect>
                    <ElInputNumber
                      v-if="formData.cpuTargetType === 'Utilization'"
                      v-model="formData.cpuUtilization"
                      :min="1"
                      :max="100"
                      style="width: 150px"
                    />
                    <template v-else>
                      <ElInputNumber
                        v-model="formData.cpuAverageValueNum"
                        :min="0"
                        style="width: 150px"
                      />
                      <ElSelect v-model="formData.cpuAverageValueUnit" style="width: 80px">
                        <ElOption label="核" value="" />
                        <ElOption label="毫核" value="m" />
                      </ElSelect>
                    </template>
                    <span v-if="formData.cpuTargetType === 'Utilization'" class="unit-text">%</span>
                  </template>
                </div>
              </ElFormItem>

              <ElFormItem label="内存指标">
                <div class="metric-config">
                  <ElSwitch v-model="formData.enableMemory" />
                  <template v-if="formData.enableMemory">
                    <ElSelect v-model="formData.memoryTargetType" style="width: 140px">
                      <ElOption label="使用率 (%)" value="Utilization" />
                      <ElOption label="平均值" value="AverageValue" />
                    </ElSelect>
                    <ElInputNumber
                      v-if="formData.memoryTargetType === 'Utilization'"
                      v-model="formData.memoryUtilization"
                      :min="1"
                      :max="100"
                      style="width: 150px"
                    />
                    <template v-else>
                      <ElInputNumber
                        v-model="formData.memoryAverageValueNum"
                        :min="0"
                        style="width: 150px"
                      />
                      <ElSelect v-model="formData.memoryAverageValueUnit" style="width: 80px">
                        <ElOption label="Ki" value="Ki" />
                        <ElOption label="Mi" value="Mi" />
                        <ElOption label="Gi" value="Gi" />
                      </ElSelect>
                    </template>
                    <span v-if="formData.memoryTargetType === 'Utilization'" class="unit-text"
                      >%</span
                    >
                  </template>
                </div>
              </ElFormItem>

              <!-- 扩缩容行为 -->
              <div class="section-header">
                <Settings :size="16" />
                <span>扩缩容行为（可选）</span>
              </div>

              <ElFormItem label="启用行为控制">
                <ElSwitch v-model="formData.enableBehavior" />
              </ElFormItem>

              <template v-if="formData.enableBehavior">
                <ElFormItem label="扩容稳定窗口">
                  <ElInputNumber
                    v-model="formData.scaleUpStabilizationWindowSeconds"
                    :min="0"
                    :max="3600"
                    :step="10"
                    style="width: 150px"
                  >
                    <template #append>秒</template>
                  </ElInputNumber>
                </ElFormItem>

                <ElFormItem label="缩容稳定窗口">
                  <ElInputNumber
                    v-model="formData.scaleDownStabilizationWindowSeconds"
                    :min="0"
                    :max="3600"
                    :step="10"
                    style="width: 150px"
                  >
                    <template #append>秒</template>
                  </ElInputNumber>
                </ElFormItem>
              </template>

              <!-- 操作按钮 -->
              <ElFormItem>
                <div class="form-actions">
                  <ElButton type="primary" size="large" :loading="submitting" @click="handleSubmit">
                    <Check :size="16" v-if="!submitting" />
                    {{ submitting ? '保存中...' : '保存更改' }}
                  </ElButton>
                  <ElButton size="large" @click="cancelEdit">
                    <X :size="16" />
                    取消
                  </ElButton>
                </div>
              </ElFormItem>
            </ElForm>
          </div>

          <!-- YAML 编辑 -->
          <div v-show="editMode === 'yaml'" class="yaml-content">
            <YamlEditorPro
              v-model="yamlContent"
              height="500px"
              :filename="`${formData.name}.yaml`"
              :readonly="false"
              :show-toolbar="true"
              :show-line-numbers="true"
              :show-status-bar="true"
              @change="handleYamlChange"
            />

            <div class="yaml-actions">
              <ElButton type="primary" size="large" :loading="submitting" @click="handleSubmit">
                <Check :size="16" v-if="!submitting" />
                {{ submitting ? '保存中...' : '保存更改' }}
              </ElButton>
              <ElButton size="large" @click="syncYamlToForm">
                <Upload :size="16" />
                导入到表单
              </ElButton>
              <ElButton size="large" @click="cancelEdit">
                <X :size="16" />
                取消
              </ElButton>
            </div>
          </div>
        </template>

        <!-- 查看模式 - 显示详情 -->
        <template v-else>
          <div class="detail-card">
            <div class="card-header">
              <Database :size="16" />
              <span>基础信息</span>
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">名称</span>
                <span class="info-value">{{ hpaDetail.name }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">命名空间</span>
                <span class="info-value">{{ hpaDetail.namespace }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">目标资源</span>
                <span class="info-value">
                  {{ hpaDetail.targetRef?.kind }}/{{ hpaDetail.targetRef?.name }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">副本数范围</span>
                <span class="info-value">
                  {{ hpaDetail.minReplicas || 1 }} - {{ hpaDetail.maxReplicas }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">当前副本数</span>
                <span class="info-value replica-count">
                  {{ hpaDetail.currentReplicas || 0 }}
                  <ArrowRight :size="14" />
                  {{ hpaDetail.desiredReplicas || 0 }}
                </span>
              </div>
            </div>
          </div>

          <div class="detail-card">
            <div class="card-header">
              <BarChart2 :size="16" />
              <span>指标配置</span>
            </div>
            <div class="metrics-list">
              <div v-for="(metric, index) in hpaDetail.metrics" :key="index" class="metric-item">
                <ElTag :type="getMetricTypeTag(metric.type)" size="small">
                  {{ metric.type }}
                </ElTag>
                <span class="metric-name">{{ getMetricName(metric) }}</span>
                <span class="metric-target">{{ getMetricTarget(metric) }}</span>
              </div>
            </div>
          </div>

          <div
            v-if="hpaDetail.currentMetrics && hpaDetail.currentMetrics.length > 0"
            class="detail-card"
          >
            <div class="card-header">
              <Activity :size="16" />
              <span>当前状态</span>
            </div>
            <div class="status-grid">
              <div
                v-for="(current, index) in hpaDetail.currentMetrics"
                :key="index"
                class="status-item"
              >
                <div class="status-label">{{ current.type }}</div>
                <div class="status-value">
                  <span v-if="current.current?.averageUtilization">
                    {{ current.current.averageUtilization }}%
                  </span>
                  <span v-else-if="current.current?.averageValue">
                    {{ current.current.averageValue }}
                  </span>
                  <span v-else>-</span>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>

      <!-- 右侧：配置建议（查看模式下也显示） -->
      <div class="info-section">
        <div class="info-card">
          <div class="info-title">
            <Info :size="16" />
            <span>{{ editing ? '配置建议' : 'HPA 说明' }}</span>
          </div>
          <div class="info-content">
            <div v-if="!editing" class="info-item">
              <div class="info-item-header">
                <Activity :size="14" class="icon-success" />
                <strong>工作原理</strong>
              </div>
              <p>
                HPA 每 15
                秒采集一次指标数据，根据当前指标值与目标值的比率计算期望副本数。如果期望副本数与当前副本数不同，且超过容忍度（默认
                10%），则触发扩缩容。
              </p>
            </div>

            <div class="info-item">
              <div class="info-item-header">
                <CheckCircle :size="14" class="icon-success" />
                <strong>{{ editing ? '指标设置' : '扩缩容算法' }}</strong>
              </div>
              <p v-if="!editing">
                期望副本数 = 当前副本数 × (当前指标值 / 目标指标值)。例如：当前 3 个副本，CPU 使用率
                90%，目标 60%，则期望副本数 = 3 × (90/60) ≈ 5 个。
              </p>
              <ul v-else>
                <li>CPU 利用率建议设置为 <strong>50%-80%</strong></li>
                <li>内存利用率建议设置为 <strong>60%-80%</strong></li>
                <li>至少启用一个指标（CPU 或内存）</li>
              </ul>
            </div>

            <div class="info-item warning">
              <div class="info-item-header">
                <AlertTriangle :size="14" class="icon-warning" />
                <strong>注意事项</strong>
              </div>
              <ul>
                <li v-if="!editing">HPA 会覆盖手动设置的副本数</li>
                <li>不要同时使用 HPA 和 VPA 的 Auto 模式</li>
                <li v-if="editing">建议设置合理的稳定窗口避免频繁扩缩容</li>
                <li v-if="!editing">扩容较快，缩容较慢（保护服务稳定性）</li>
              </ul>
            </div>

            <div v-if="editing" class="info-item">
              <div class="info-item-header">
                <Lightbulb :size="14" class="icon-info" />
                <strong>最佳实践</strong>
              </div>
              <ul>
                <li>优先使用 CPU 利用率作为扩缩容指标</li>
                <li>缩容稳定窗口建议设置为 300 秒</li>
                <li>扩容稳定窗口可以设置为 0 秒快速响应</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>

    <ElDialog v-model="yamlViewVisible" title="HPA YAML" width="900px">
      <YamlEditorPro
        v-model="yamlViewContent"
        height="600px"
        :filename="`${hpaDetail?.name || 'hpa'}.yaml`"
        :readonly="true"
        :show-toolbar="true"
        :show-line-numbers="true"
        :show-status-bar="true"
      />
      <template #footer>
        <ElButton @click="yamlViewVisible = false">关闭</ElButton>
        <ElButton type="primary" @click="copyYaml">
          <Copy :size="16" />
          复制 YAML
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted, onActivated } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import {
    Edit,
    FileText,
    TrendingUp,
    Activity,
    Database,
    BarChart2,
    Settings,
    Info,
    CheckCircle,
    AlertTriangle,
    Check,
    RotateCcw,
    Trash2,
    Upload,
    Copy,
    X,
    ArrowRight,
    HelpCircle,
    Lightbulb
  } from 'lucide-vue-next'
  import type { V2HorizontalPodAutoscaler } from '@kubernetes/client-node'
  import * as yaml from 'js-yaml'
  import {
    getHPADetailApi,
    getHPAYamlApi,
    createHPAApi,
    updateHPAApi,
    deleteHPAApi,
    type OnecProjectVersion,
    type OnecProjectApplication,
    type ProjectWorkspace
  } from '@/api'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'

  defineOptions({ name: 'HPAManagement' })

  interface Props {
    version: OnecProjectVersion
    application: OnecProjectApplication
    workspace: ProjectWorkspace
    resourceType: string
    refreshTrigger?: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ success: [] }>()

  // 表单数据结构
  interface HPAFormData {
    name: string
    targetRef: {
      apiVersion: string
      kind: string
      name: string
    }
    minReplicas: number
    maxReplicas: number
    enableCpu: boolean
    cpuTargetType: 'Utilization' | 'AverageValue'
    cpuUtilization: number
    cpuAverageValueNum: number
    cpuAverageValueUnit: string
    enableMemory: boolean
    memoryTargetType: 'Utilization' | 'AverageValue'
    memoryUtilization: number
    memoryAverageValueNum: number
    memoryAverageValueUnit: string
    enableBehavior: boolean
    scaleUpStabilizationWindowSeconds: number
    scaleDownStabilizationWindowSeconds: number
  }

  // 状态管理
  const initialLoading = ref(false)
  const deleting = ref(false)
  const submitting = ref(false)
  const editing = ref(false)
  const hpaDetail = ref<V2HorizontalPodAutoscaler | null>(null)

  const editMode = ref<'form' | 'yaml'>('form')
  const yamlContent = ref('')

  const yamlViewVisible = ref(false)
  const yamlViewContent = ref('')

  // 表单相关
  const formRef = ref<FormInstance>()
  const formData = ref<HPAFormData>({
    name: '',
    targetRef: {
      apiVersion: 'apps/v1',
      kind: '',
      name: ''
    },
    minReplicas: 2,
    maxReplicas: 10,
    enableCpu: true,
    cpuTargetType: 'Utilization',
    cpuUtilization: 80,
    cpuAverageValueNum: 500,
    cpuAverageValueUnit: 'm',
    enableMemory: false,
    memoryTargetType: 'Utilization',
    memoryUtilization: 80,
    memoryAverageValueNum: 1,
    memoryAverageValueUnit: 'Gi',
    enableBehavior: false,
    scaleUpStabilizationWindowSeconds: 0,
    scaleDownStabilizationWindowSeconds: 300
  })

  const formRules: FormRules = {
    name: [{ required: true, message: '请输入 HPA 名称', trigger: 'blur' }],
    minReplicas: [
      { required: true, message: '请输入最小副本数', trigger: 'blur' },
      { type: 'number', min: 1, message: '最小副本数不能小于 1', trigger: 'blur' }
    ],
    maxReplicas: [
      { required: true, message: '请输入最大副本数', trigger: 'blur' },
      { type: 'number', min: 2, message: '最大副本数不能小于 2', trigger: 'blur' }
    ]
  }

  // 计算属性
  const targetResourceDisplay = computed(() => {
    return formData.value.targetRef.name || props.version?.resourceName || '未知'
  })

  const parseResourceValue = (
    value: string | undefined,
    type: 'cpu' | 'memory'
  ): { value: number; unit: string } => {
    if (!value) {
      return {
        value: type === 'cpu' ? 500 : 1,
        unit: type === 'cpu' ? 'm' : 'Mi'
      }
    }

    const str = String(value).trim()

    if (type === 'cpu') {
      // CPU: 支持 m（毫核）或空（核）
      if (str.endsWith('m')) {
        const num = parseFloat(str.slice(0, -1))
        return { value: isNaN(num) ? 500 : num, unit: 'm' }
      } else {
        const num = parseFloat(str)
        return { value: isNaN(num) ? 0.5 : num, unit: '' }
      }
    } else {
      // 内存: 支持 Ki, Mi, Gi
      if (str.endsWith('Gi')) {
        const num = parseFloat(str.slice(0, -2))
        return { value: isNaN(num) ? 1 : num, unit: 'Gi' }
      } else if (str.endsWith('Mi')) {
        const num = parseFloat(str.slice(0, -2))
        return { value: isNaN(num) ? 512 : num, unit: 'Mi' }
      } else if (str.endsWith('Ki')) {
        const num = parseFloat(str.slice(0, -2))
        return { value: isNaN(num) ? 512 : num, unit: 'Ki' }
      } else {
        // 纯数字，默认按 Mi 处理
        const num = parseFloat(str)
        return { value: isNaN(num) ? 512 : num, unit: 'Mi' }
      }
    }
  }

  const formatResourceValue = (value: number, unit: string): string => {
    if (!value && value !== 0) return ''
    return `${value}${unit}`
  }

  // 初始化表单数据
  const initFormData = () => {
    const resourceKind =
      props.application?.resourceType?.charAt(0).toUpperCase() +
      props.application?.resourceType?.slice(1)

    formData.value = {
      name: `${props.version?.resourceName || 'app'}-hpa`,
      targetRef: {
        apiVersion: 'apps/v1',
        kind: resourceKind || 'Deployment',
        name: props.version?.resourceName || ''
      },
      minReplicas: 2,
      maxReplicas: 10,
      enableCpu: true,
      cpuTargetType: 'Utilization',
      cpuUtilization: 80,
      cpuAverageValueNum: 500,
      cpuAverageValueUnit: 'm',
      enableMemory: false,
      memoryTargetType: 'Utilization',
      memoryUtilization: 80,
      memoryAverageValueNum: 1,
      memoryAverageValueUnit: 'Gi',
      enableBehavior: false,
      scaleUpStabilizationWindowSeconds: 0,
      scaleDownStabilizationWindowSeconds: 300
    }
  }

  // 表单转 HPA 对象
  const formToHPA = (): V2HorizontalPodAutoscaler => {
    const metrics: any[] = []

    if (formData.value.enableCpu) {
      const cpuMetric: any = {
        type: 'Resource',
        resource: {
          name: 'cpu',
          target: {
            type: formData.value.cpuTargetType
          }
        }
      }

      if (formData.value.cpuTargetType === 'Utilization') {
        cpuMetric.resource.target.averageUtilization = formData.value.cpuUtilization
      } else {
        cpuMetric.resource.target.averageValue = formatResourceValue(
          formData.value.cpuAverageValueNum,
          formData.value.cpuAverageValueUnit
        )
      }

      metrics.push(cpuMetric)
    }

    if (formData.value.enableMemory) {
      const memoryMetric: any = {
        type: 'Resource',
        resource: {
          name: 'memory',
          target: {
            type: formData.value.memoryTargetType
          }
        }
      }

      if (formData.value.memoryTargetType === 'Utilization') {
        memoryMetric.resource.target.averageUtilization = formData.value.memoryUtilization
      } else {
        memoryMetric.resource.target.averageValue = formatResourceValue(
          formData.value.memoryAverageValueNum,
          formData.value.memoryAverageValueUnit
        )
      }

      metrics.push(memoryMetric)
    }

    const hpa: V2HorizontalPodAutoscaler = {
      apiVersion: 'autoscaling/v2',
      kind: 'HorizontalPodAutoscaler',
      metadata: {
        name: formData.value.name,
        namespace: props.workspace?.namespace
      },
      spec: {
        scaleTargetRef: formData.value.targetRef,
        minReplicas: formData.value.minReplicas,
        maxReplicas: formData.value.maxReplicas,
        metrics
      }
    }

    if (formData.value.enableBehavior) {
      hpa.spec!.behavior = {
        scaleUp: {
          stabilizationWindowSeconds: formData.value.scaleUpStabilizationWindowSeconds
        },
        scaleDown: {
          stabilizationWindowSeconds: formData.value.scaleDownStabilizationWindowSeconds
        }
      }
    }

    return hpa
  }

  // HPA 对象转表单
  const hpaToForm = (hpa: V2HorizontalPodAutoscaler) => {
    const cpuMetric = hpa.spec?.metrics?.find(
      (m) => m.type === 'Resource' && m.resource?.name === 'cpu'
    )
    const memoryMetric = hpa.spec?.metrics?.find(
      (m) => m.type === 'Resource' && m.resource?.name === 'memory'
    )

    const cpuAverage = parseResourceValue(cpuMetric?.resource?.target?.averageValue, 'cpu')
    const memoryAverage = parseResourceValue(memoryMetric?.resource?.target?.averageValue, 'memory')

    formData.value = {
      name: hpa.metadata?.name || '',
      targetRef: {
        apiVersion: hpa.spec?.scaleTargetRef?.apiVersion || 'apps/v1',
        kind: hpa.spec?.scaleTargetRef?.kind || '',
        name: hpa.spec?.scaleTargetRef?.name || ''
      },
      minReplicas: hpa.spec?.minReplicas || 1,
      maxReplicas: hpa.spec?.maxReplicas || 10,
      enableCpu: !!cpuMetric,
      cpuTargetType:
        (cpuMetric?.resource?.target?.type as 'Utilization' | 'AverageValue') || 'Utilization',
      cpuUtilization: cpuMetric?.resource?.target?.averageUtilization || 80,
      cpuAverageValueNum: cpuAverage.value,
      cpuAverageValueUnit: cpuAverage.unit,
      enableMemory: !!memoryMetric,
      memoryTargetType:
        (memoryMetric?.resource?.target?.type as 'Utilization' | 'AverageValue') || 'Utilization',
      memoryUtilization: memoryMetric?.resource?.target?.averageUtilization || 80,
      memoryAverageValueNum: memoryAverage.value,
      memoryAverageValueUnit: memoryAverage.unit,
      enableBehavior: !!hpa.spec?.behavior,
      scaleUpStabilizationWindowSeconds:
        hpa.spec?.behavior?.scaleUp?.stabilizationWindowSeconds || 0,
      scaleDownStabilizationWindowSeconds:
        hpa.spec?.behavior?.scaleDown?.stabilizationWindowSeconds || 300
    }
  }

  // 同步表单到 YAML
  const syncFormToYaml = () => {
    const hpa = formToHPA()
    yamlContent.value = yaml.dump(hpa, {
      indent: 2,
      lineWidth: -1,
      noRefs: true
    })
  }

  // 同步 YAML 到表单 - 修复 namespace 和 targetRef
  const syncYamlToForm = () => {
    try {
      let parsed = yaml.load(yamlContent.value) as V2HorizontalPodAutoscaler

      if (!parsed.metadata) {
        parsed.metadata = {}
      }
      parsed.metadata.namespace = props.workspace?.namespace

      if (!parsed.spec) {
        parsed.spec = {} as any
      }
      parsed.spec.scaleTargetRef = {
        apiVersion: 'apps/v1',
        kind:
          props.application?.resourceType?.charAt(0).toUpperCase() +
          props.application?.resourceType?.slice(1),
        name: props.version?.resourceName || ''
      }

      // 更新 YAML 内容
      yamlContent.value = yaml.dump(parsed, {
        indent: 2,
        lineWidth: -1,
        noRefs: true
      })

      hpaToForm(parsed)
      editMode.value = 'form'
      ElMessage.success('✅ 已导入到表单（namespace 和 targetRef 已自动修复）')
    } catch (error: any) {}
  }

  // 处理 YAML 变化
  const handleYamlChange = (value: string) => {
    yamlContent.value = value
  }

  // 加载 HPA 详情
  const loadHPADetail = async () => {
    if (!props.version?.id) {
      console.warn('[HPA 管理] 版本ID不存在，跳过加载')
      return
    }

    initialLoading.value = true
    try {
      const detail = await getHPADetailApi({ versionId: props.version.id })
      hpaDetail.value = detail
    } catch (error: any) {
      console.error('[HPA 管理] 加载失败:', error)
      if (error?.response?.status !== 404) {
      }
      hpaDetail.value = null
    } finally {
      initialLoading.value = false
    }
  }

  // 开始编辑
  const startEdit = async () => {
    if (!props.version?.id || !hpaDetail.value) return

    try {
      const yamlStr = await getHPAYamlApi({ versionId: props.version.id })
      const parsed = yaml.load(yamlStr) as V2HorizontalPodAutoscaler

      hpaToForm(parsed)
      syncFormToYaml()
      editing.value = true
      editMode.value = 'form'
    } catch (error) {
      console.error('[HPA 管理] 获取 YAML 失败:', error)
    }
  }

  // 取消编辑
  const cancelEdit = () => {
    editing.value = false
    editMode.value = 'form'
    formRef.value?.resetFields()
  }

  // 查看 YAML
  const viewYaml = async () => {
    if (!props.version?.id) return

    try {
      const yamlStr = await getHPAYamlApi({ versionId: props.version.id })
      yamlViewContent.value = yamlStr
      yamlViewVisible.value = true
    } catch (error) {
      console.error('[HPA 管理] 获取 YAML 失败:', error)
    }
  }

  // 复制 YAML
  const copyYaml = async () => {
    try {
      await navigator.clipboard.writeText(yamlViewContent.value)
      ElMessage.success('📋 已复制到剪贴板')
    } catch {}
  }

  // 重置表单
  const handleReset = () => {
    if (hpaDetail.value) {
      hpaToForm(hpaDetail.value)
      syncFormToYaml()
    } else {
      initFormData()
      syncFormToYaml()
    }
    ElMessage.info('已重置')
  }

  // 提交
  const handleSubmit = async () => {
    if (!props.version?.id) {
      return
    }

    // 验证至少启用一个指标
    if (!formData.value.enableCpu && !formData.value.enableMemory) {
      return
    }

    let yamlStr = ''

    if (editMode.value === 'form') {
      if (!formRef.value) return

      try {
        await formRef.value.validate()
        syncFormToYaml()
        yamlStr = yamlContent.value
      } catch {
        return
      }
    } else {
      if (!yamlContent.value.trim()) {
        return
      }

      try {
        let parsed = yaml.load(yamlContent.value) as V2HorizontalPodAutoscaler

        if (!parsed.metadata) {
          parsed.metadata = {}
        }
        parsed.metadata.namespace = props.workspace?.namespace

        if (!parsed.spec) {
          parsed.spec = {} as any
        }
        parsed.spec.scaleTargetRef = {
          apiVersion: 'apps/v1',
          kind:
            props.application?.resourceType?.charAt(0).toUpperCase() +
            props.application?.resourceType?.slice(1),
          name: props.version?.resourceName || ''
        }

        yamlStr = yaml.dump(parsed, {
          indent: 2,
          lineWidth: -1,
          noRefs: true
        })
      } catch (error: any) {
        return
      }
    }

    submitting.value = true
    try {
      if (!hpaDetail.value) {
        await createHPAApi({
          versionId: props.version.id,
          hpaYamlStr: yamlStr
        })
        ElMessage.success('✅ HPA 创建成功')
      } else {
        await updateHPAApi({
          versionId: props.version.id,
          hpaYamlStr: yamlStr
        })
        ElMessage.success('✅ HPA 更新成功')
      }

      editing.value = false
      emit('success')
      await loadHPADetail()
    } catch (error) {
      console.error('[HPA 管理] 提交失败:', error)
    } finally {
      submitting.value = false
    }
  }

  // 删除
  const handleDelete = async () => {
    try {
      await ElMessageBox.confirm('确定要删除 HPA 配置吗？删除后将停止自动扩缩容。', '删除确认', {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      })

      if (!props.version?.id) {
        return
      }

      deleting.value = true
      await deleteHPAApi({ versionId: props.version.id })
      ElMessage.success('✅ HPA 删除成功')
      hpaDetail.value = null
      emit('success')
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('[HPA 管理] 删除失败:', error)
      }
    } finally {
      deleting.value = false
    }
  }

  // 获取指标类型标签
  const getMetricTypeTag = (type?: string) => {
    const tagMap: Record<string, string> = {
      Resource: 'primary',
      Pods: 'success',
      Object: 'warning',
      External: 'danger'
    }
    return tagMap[type || ''] || 'info'
  }

  const getMetricName = (metric: any): string => {
    if (metric.resource) {
      return metric.resource.name || 'Unknown'
    } else if (metric.pods) {
      return metric.pods.metric?.name || 'Unknown'
    } else if (metric.object) {
      return metric.object.metric?.name || 'Unknown'
    } else if (metric.external) {
      return metric.external.metric?.name || 'Unknown'
    }
    return 'Unknown'
  }

  const getMetricTarget = (metric: any): string => {
    let target
    if (metric.resource) {
      target = metric.resource.target
    } else if (metric.pods) {
      target = metric.pods.target
    } else if (metric.object) {
      target = metric.object.target
    } else if (metric.external) {
      target = metric.external.target
    }

    if (!target) return ''

    if (target.type === 'Utilization' && target.averageUtilization) {
      return `目标: ${target.averageUtilization}%`
    } else if (target.type === 'AverageValue' && target.averageValue) {
      return `目标: ${target.averageValue}`
    }
    return ''
  }

  // 监听编辑模式切换
  watch(editMode, (newMode) => {
    if (newMode === 'yaml') {
      syncFormToYaml()
    }
  })

  // 监听刷新触发器
  watch(
    () => props.refreshTrigger,
    (newVal) => {
      if (newVal && newVal > 0) {
        loadHPADetail()
      }
    }
  )

  onMounted(() => {
    initFormData()
    syncFormToYaml()
    loadHPADetail()
  })

  onActivated(() => {
    loadHPADetail()
  })

  defineExpose({
    hasData: () => hpaDetail.value !== null,
    hasUnsavedChanges: () => editing.value,
    refresh: loadHPADetail
  })
</script>

<style scoped lang="scss">
  .hpa-management-container {
    height: 100%;
    background: #fff;

    .loading-wrapper {
      display: flex;
      justify-content: center;
      align-items: center;
      height: 100%;
    }

    // 左右布局
    .content-layout {
      display: flex;
      height: 100%;
      gap: 24px;

      // 左侧表单区域
      .form-section {
        flex: 1;
        padding: 20px 24px;
        overflow-y: auto;
        min-width: 0;

        // 状态栏
        .status-bar {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 20px;
          padding-bottom: 16px;
          border-bottom: 1px solid var(--el-border-color-lighter);

          .status-left {
            :deep(.el-tag) {
              padding: 8px 16px;
              font-size: 14px;
              display: inline-flex;
              align-items: center;
              gap: 6px;
            }
          }

          .status-right {
            display: flex;
            gap: 8px;
          }
        }

        // Section 标题
        .section-header {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 14px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin: 20px 0 12px 0;
          padding-bottom: 8px;
          border-bottom: 1px solid var(--el-border-color-lighter);

          &:first-child {
            margin-top: 0;
          }
        }

        // 模式切换
        .mode-switch {
          margin-bottom: 20px;
          display: flex;
          justify-content: center;

          :deep(.el-radio-button__inner) {
            display: flex;
            align-items: center;
            gap: 6px;
          }
        }

        // 表单内容
        .form-content {
          // Label 帮助图标
          .label-help-icon {
            margin-left: 6px;
            color: var(--el-text-color-secondary);
            cursor: help;
            vertical-align: middle;

            &:hover {
              color: var(--el-color-primary);
            }
          }

          .range-inputs {
            display: flex;
            align-items: center;
            gap: 12px;

            .range-separator {
              color: var(--el-text-color-secondary);
            }

            :deep(.el-form-item) {
              flex: 1;
            }
          }

          .metric-config {
            display: flex;
            align-items: center;
            gap: 12px;
            width: 100%;
            flex-wrap: wrap;

            .unit-text {
              color: var(--el-text-color-secondary);
              font-weight: 500;
            }
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
        }

        // YAML 内容
        .yaml-content {
          .yaml-actions {
            display: flex;
            gap: 12px;
            margin-top: 16px;

            :deep(.el-button) {
              display: inline-flex;
              align-items: center;
              gap: 6px;
            }
          }
        }

        // 详情卡片
        .detail-card {
          margin-bottom: 16px;
          padding: 20px;
          background: var(--el-fill-color-lighter);
          border-radius: 8px;
          border: 1px solid var(--el-border-color);

          .card-header {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 14px;
            font-weight: 600;
            margin-bottom: 16px;
            padding-bottom: 12px;
            border-bottom: 1px solid var(--el-border-color-lighter);
          }

          .info-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 16px;

            .info-item {
              display: flex;
              flex-direction: column;
              gap: 6px;

              .info-label {
                font-size: 12px;
                color: var(--el-text-color-secondary);
              }

              .info-value {
                font-size: 14px;
                font-weight: 500;
                color: var(--el-text-color-primary);

                &.replica-count {
                  display: flex;
                  align-items: center;
                  gap: 6px;
                  color: var(--el-color-primary);
                }
              }
            }
          }

          .metrics-list {
            display: flex;
            flex-direction: column;
            gap: 12px;

            .metric-item {
              display: flex;
              align-items: center;
              gap: 12px;
              padding: 12px;
              background: white;
              border-radius: 6px;

              .metric-name {
                font-weight: 600;
                flex: 1;
              }

              .metric-target {
                font-size: 13px;
                color: var(--el-text-color-secondary);
              }
            }
          }

          .status-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
            gap: 16px;

            .status-item {
              text-align: center;
              padding: 16px;
              background: white;
              border-radius: 6px;

              .status-label {
                font-size: 12px;
                color: var(--el-text-color-secondary);
                margin-bottom: 8px;
              }

              .status-value {
                font-size: 24px;
                font-weight: 700;
                color: var(--el-color-primary);
              }
            }
          }
        }
      }

      // 右侧信息区域
      .info-section {
        width: 360px;
        padding: 20px 24px 20px 0;
        flex-shrink: 0;
        overflow-y: auto;

        .create-alert {
          margin-bottom: 20px;

          .alert-content {
            display: flex;
            align-items: center;
            gap: 8px;
            font-weight: 600;
          }

          p {
            margin: 4px 0;
            line-height: 1.6;
          }
        }

        .info-card {
          background: var(--el-fill-color-lighter);
          border-radius: 8px;
          border: 1px solid var(--el-border-color);
          padding: 20px;

          .info-title {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 14px;
            font-weight: 600;
            color: var(--el-text-color-primary);
            margin-bottom: 16px;
            padding-bottom: 12px;
            border-bottom: 1px solid var(--el-border-color-lighter);
          }

          .info-content {
            display: flex;
            flex-direction: column;
            gap: 16px;

            .info-item {
              padding: 12px;
              background: white;
              border-radius: 6px;
              border: 1px solid var(--el-border-color-lighter);

              &.warning {
                background: var(--el-color-warning-light-9);
                border-color: var(--el-color-warning-light-5);
              }

              .info-item-header {
                display: flex;
                align-items: center;
                gap: 8px;
                margin-bottom: 8px;

                .icon-success {
                  color: var(--el-color-success);
                }

                .icon-warning {
                  color: var(--el-color-warning);
                }

                .icon-info {
                  color: var(--el-color-primary);
                }

                strong {
                  font-size: 13px;
                  color: var(--el-text-color-primary);
                }
              }

              p {
                margin: 0;
                font-size: 12px;
                line-height: 1.6;
                color: var(--el-text-color-regular);
              }

              ul {
                margin: 0;
                padding-left: 20px;
                font-size: 12px;
                line-height: 1.6;
                color: var(--el-text-color-regular);

                li {
                  margin: 4px 0;
                }
              }
            }
          }
        }
      }
    }
  }
</style>

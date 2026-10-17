<template>
  <div class="resource-dashboard">
    <!-- 页面头部 -->
    <div class="dashboard-header">
      <div>
        <h1 class="page-title">资源仪表盘</h1>
        <p class="page-subtitle">集群资源分配与使用情况总览</p>
      </div>
      <div class="header-actions">
        <ElSelect
          v-model="filterForm.clusterUuid"
          placeholder="全部集群"
          clearable
          style="width: 160px"
          @change="handleFilterChange"
        >
          <ElOption label="全部集群" value="" />
          <ElOption
            v-for="cluster in clusterList"
            :key="cluster.uuid"
            :label="cluster.name"
            :value="cluster.uuid"
          />
        </ElSelect>
        <ElSelect
          v-model="filterForm.projectId"
          placeholder="全部项目"
          clearable
          style="width: 160px"
          @change="handleFilterChange"
        >
          <ElOption label="全部项目" :value="0" />
          <ElOption
            v-for="project in projectList"
            :key="project.id"
            :label="project.name"
            :value="project.id"
          />
        </ElSelect>
        <ElButton type="primary" @click="refreshAllData">
          <ArtSvgIcon icon="ri:refresh-line" class="mr-1" />
          刷新
        </ElButton>
      </div>
    </div>

    <!-- 顶部统计卡片 - 8个卡片 -->
    <div class="top-stats-grid">
      <div class="stat-card stat-card-blue">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:server-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">集群总数</span>
          <ArtCountTo
            class="stat-card-value"
            :target="summaryData.summaryCards.clusterTotalCount"
            :duration="1000"
          />
        </div>
      </div>
      <div class="stat-card stat-card-purple">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:folder-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">项目总数</span>
          <ArtCountTo
            class="stat-card-value"
            :target="summaryData.summaryCards.projectTotalCount"
            :duration="1000"
          />
        </div>
      </div>
      <div class="stat-card stat-card-cyan">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:layout-grid-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">工作空间</span>
          <ArtCountTo
            class="stat-card-value"
            :target="summaryData.summaryCards.workspaceTotalCount"
            :duration="1000"
          />
        </div>
      </div>
      <div class="stat-card stat-card-pink">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:cpu-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">物理CPU分配率</span>
          <div class="stat-card-value"
            >{{ summaryData.summaryCards.cpuPhysicalRate.toFixed(1) }}%</div
          >
          <span class="stat-card-sub"
            >{{ summaryData.summaryCards.cpuLimit.toFixed(0) }} /
            {{ summaryData.summaryCards.cpuPhysical.toFixed(0) }} 核</span
          >
        </div>
      </div>
      <div class="stat-card stat-card-green">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:database-2-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">物理内存分配率</span>
          <div class="stat-card-value"
            >{{ summaryData.summaryCards.memPhysicalRate.toFixed(1) }}%</div
          >
          <span class="stat-card-sub"
            >{{ summaryData.summaryCards.memLimit.toFixed(0) }} /
            {{ summaryData.summaryCards.memPhysical.toFixed(0) }} GiB</span
          >
        </div>
      </div>
      <div class="stat-card stat-card-violet">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:cpu-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">物理GPU分配率</span>
          <div class="stat-card-value"
            >{{ summaryData.summaryCards.gpuPhysicalRate.toFixed(1) }}%</div
          >
          <span class="stat-card-sub"
            >{{ summaryData.summaryCards.gpuLimit.toFixed(0) }} /
            {{ summaryData.summaryCards.gpuPhysical.toFixed(0) }} 卡</span
          >
        </div>
      </div>
      <div class="stat-card stat-card-indigo">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:hard-drive-2-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">存储分配率</span>
          <div class="stat-card-value"
            >{{ summaryData.summaryCards.storagePhysicalRate.toFixed(1) }}%</div
          >
          <span class="stat-card-sub"
            >{{ formatStorage(summaryData.summaryCards.storageLimit) }} /
            {{ formatStorage(summaryData.summaryCards.storagePhysical) }}</span
          >
        </div>
      </div>
      <div class="stat-card stat-card-amber">
        <div class="stat-card-icon"><ArtSvgIcon icon="ri:instance-line" /></div>
        <div class="stat-card-content">
          <span class="stat-card-label">Pod分配率</span>
          <div class="stat-card-value"
            >{{ summaryData.summaryCards.podPhysicalRate.toFixed(1) }}%</div
          >
          <span class="stat-card-sub"
            >{{ summaryData.summaryCards.podLimit }} /
            {{ summaryData.summaryCards.podPhysical }} 个</span
          >
        </div>
      </div>
    </div>

    <!-- 中间两个卡片 - 超分收益 + 资源详情 -->
    <ElRow :gutter="20" class="mb-5">
      <ElCol :span="10">
        <div class="art-card saving-card">
          <div class="card-header">
            <ArtSvgIcon icon="ri:leaf-line" class="header-icon text-green-500" />
            <span class="header-title">超分收益</span>
          </div>
          <div class="saving-list">
            <div class="saving-row saving-row-green">
              <div class="saving-icon-wrapper saving-icon-green"
                ><ArtSvgIcon icon="ri:cpu-line"
              /></div>
              <div class="saving-info">
                <div class="saving-label">CPU 节约</div>
                <div class="saving-value"
                  >{{ formatSavingValue(summaryData.oversellSavings.cpu.savingAmount) }} 核</div
                >
                <div class="saving-detail"
                  >配额 {{ summaryData.oversellSavings.cpu.limitTotal.toFixed(0) }} → 超分
                  {{ summaryData.oversellSavings.cpu.capacityTotal.toFixed(0) }} 核</div
                >
              </div>
              <div class="saving-rate-tag saving-rate-green"
                ><span>↑ {{ summaryData.oversellSavings.cpu.savingRate.toFixed(1) }}%</span></div
              >
            </div>
            <div class="saving-row saving-row-blue">
              <div class="saving-icon-wrapper saving-icon-blue"
                ><ArtSvgIcon icon="ri:database-2-line"
              /></div>
              <div class="saving-info">
                <div class="saving-label">内存 节约</div>
                <div class="saving-value"
                  >{{ formatSavingValue(summaryData.oversellSavings.mem.savingAmount) }} GiB</div
                >
                <div class="saving-detail"
                  >配额 {{ summaryData.oversellSavings.mem.limitTotal.toFixed(0) }} → 超分
                  {{ summaryData.oversellSavings.mem.capacityTotal.toFixed(0) }} GiB</div
                >
              </div>
              <div class="saving-rate-tag saving-rate-blue"
                ><span>↑ {{ summaryData.oversellSavings.mem.savingRate.toFixed(1) }}%</span></div
              >
            </div>
            <div class="saving-row saving-row-purple">
              <div class="saving-icon-wrapper saving-icon-purple"
                ><ArtSvgIcon icon="ri:cpu-line"
              /></div>
              <div class="saving-info">
                <div class="saving-label">GPU 节约</div>
                <div class="saving-value"
                  >{{ formatSavingValue(summaryData.oversellSavings.gpu.savingAmount) }} 卡</div
                >
                <div class="saving-detail"
                  >配额 {{ summaryData.oversellSavings.gpu.limitTotal.toFixed(0) }} → 超分
                  {{ summaryData.oversellSavings.gpu.capacityTotal.toFixed(0) }} 卡</div
                >
              </div>
              <div class="saving-rate-tag saving-rate-purple"
                ><span>↑ {{ summaryData.oversellSavings.gpu.savingRate.toFixed(1) }}%</span></div
              >
            </div>
          </div>
        </div>
      </ElCol>
      <ElCol :span="14">
        <div class="art-card detail-card">
          <div class="card-header">
            <ArtSvgIcon icon="ri:file-list-3-line" class="header-icon text-blue-500" />
            <span class="header-title">资源详情</span>
          </div>
          <div class="detail-table">
            <div class="detail-row header-row">
              <div class="detail-cell cell-name">资源</div>
              <div class="detail-cell cell-value">物理总量</div>
              <div class="detail-cell cell-value">分配总数</div>
              <div class="detail-cell cell-value">超分总数</div>
            </div>
            <div class="detail-row">
              <div class="detail-cell cell-name"
                ><span class="resource-dot bg-orange-400"></span>CPU</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.cpu.physical.toFixed(0) }} 核</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.cpu.limit.toFixed(0) }} 核</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.cpu.capacity.toFixed(0) }} 核</div
              >
            </div>
            <div class="detail-row">
              <div class="detail-cell cell-name"
                ><span class="resource-dot bg-green-400"></span>内存</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.mem.physical.toFixed(0) }} GiB</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.mem.limit.toFixed(0) }} GiB</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.mem.capacity.toFixed(0) }} GiB</div
              >
            </div>
            <div class="detail-row">
              <div class="detail-cell cell-name"
                ><span class="resource-dot bg-amber-400"></span>GPU</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.gpu.physical.toFixed(0) }} 卡</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.gpu.limit.toFixed(0) }} 卡</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.gpu.capacity.toFixed(0) }} 卡</div
              >
            </div>
            <div class="detail-row">
              <div class="detail-cell cell-name"
                ><span class="resource-dot bg-indigo-400"></span>存储</div
              >
              <div class="detail-cell cell-value">{{
                formatStorage(summaryData.allocationOverview.storage.physical)
              }}</div>
              <div class="detail-cell cell-value">{{
                formatStorage(summaryData.allocationOverview.storage.limit)
              }}</div>
              <div class="detail-cell cell-value text-gray-400">-</div>
            </div>
            <div class="detail-row">
              <div class="detail-cell cell-name"
                ><span class="resource-dot bg-cyan-400"></span>Pod</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.pod.physical.toFixed(0) }} 个</div
              >
              <div class="detail-cell cell-value"
                >{{ summaryData.allocationOverview.pod.limit.toFixed(0) }} 个</div
              >
              <div class="detail-cell cell-value text-gray-400">-</div>
            </div>
          </div>
        </div>
      </ElCol>
    </ElRow>

    <!-- 资源排行榜 -->
    <div class="art-card ranking-card">
      <div class="ranking-header">
        <div class="ranking-title">
          <ArtSvgIcon icon="ri:bar-chart-grouped-line" class="header-icon text-indigo-500" />
          <span
            >资源使用 TOP{{ topN }}
            {{
              currentDimension === 'cluster'
                ? '集群'
                : currentDimension === 'project'
                  ? '项目'
                  : '工作空间'
            }}</span
          >
        </div>
        <div class="ranking-controls">
          <ElSelect v-model="topN" style="width: 110px" @change="handleTopNChange">
            <ElOption label="TOP 1" :value="1" />
            <ElOption label="TOP 3" :value="3" />
            <ElOption label="TOP 5" :value="5" />
            <ElOption label="TOP 10" :value="10" />
            <ElOption label="TOP 20" :value="20" />
            <ElOption label="TOP 50" :value="50" />
          </ElSelect>
          <div class="btn-group dimension-group">
            <button
              v-for="dim in dimensionOptions"
              :key="dim.value"
              class="btn-item"
              :class="{ active: currentDimension === dim.value }"
              @click="changeDimension(dim.value)"
              >{{ dim.label }}</button
            >
          </div>
        </div>
      </div>

      <!-- 表格区域 -->
      <div class="ranking-table-wrapper">
        <!-- 集群排行表格 -->
        <table v-if="currentDimension === 'cluster'" class="ranking-table ranking-table-cluster">
          <thead>
            <tr>
              <th class="col-rank">排名</th>
              <th class="col-name-wide">集群名称</th>
              <th
                class="col-data sortable"
                :class="getSortClass('cpuCapacity')"
                @click="handleSort('cpuCapacity')"
              >
                <span>物理CPU</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('cpuCapacity')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('cpuLimit')"
                @click="handleSort('cpuLimit')"
              >
                <span>超分CPU</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('cpuLimit')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('cpuAllocated')"
                @click="handleSort('cpuAllocated')"
              >
                <span>分配CPU</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('cpuAllocated')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('memCapacityGib')"
                @click="handleSort('memCapacityGib')"
              >
                <span>物理内存</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('memCapacityGib')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('memLimitGib')"
                @click="handleSort('memLimitGib')"
              >
                <span>超分内存</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('memLimitGib')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('memAllocatedGib')"
                @click="handleSort('memAllocatedGib')"
              >
                <span>分配内存</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('memAllocatedGib')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('gpuCapacity')"
                @click="handleSort('gpuCapacity')"
              >
                <span>物理GPU</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('gpuCapacity')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('gpuLimit')"
                @click="handleSort('gpuLimit')"
              >
                <span>超分GPU</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('gpuLimit')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('gpuAllocated')"
                @click="handleSort('gpuAllocated')"
              >
                <span>分配GPU</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('gpuAllocated')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('storageLimitGib')"
                @click="handleSort('storageLimitGib')"
              >
                <span>存储总量</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('storageLimitGib')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('storageAllocatedGib')"
                @click="handleSort('storageAllocatedGib')"
              >
                <span>存储分配</span>
                <span class="sort-icon"
                  ><ArtSvgIcon :icon="getSortIcon('storageAllocatedGib')"
                /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('podsLimit')"
                @click="handleSort('podsLimit')"
              >
                <span>Pod总数</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('podsLimit')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('podsAllocated')"
                @click="handleSort('podsAllocated')"
              >
                <span>Pod分配</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('podsAllocated')" /></span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, index) in clusterRankingData.items" :key="item.clusterUuid">
              <td class="col-rank">
                <div class="rank-badge" :class="getRankClass(index + 1)">{{ index + 1 }}</div>
              </td>
              <td class="col-name-wide">
                <div class="item-name">{{ item.clusterName }}</div>
                <span class="env-tag" :class="getEnvClass(item.environment)">{{
                  getEnvLabel(item.environment)
                }}</span>
              </td>
              <td class="col-data">{{ formatCpu(item.cpuCapacity) }}</td>
              <td class="col-data text-primary">{{ formatCpu(item.cpuLimit) }}</td>
              <td class="col-data text-success">{{ formatCpu(item.cpuAllocated) }}</td>
              <td class="col-data">{{ formatMem(item.memCapacityGib) }}</td>
              <td class="col-data text-primary">{{ formatMem(item.memLimitGib) }}</td>
              <td class="col-data text-success">{{ formatMem(item.memAllocatedGib) }}</td>
              <td class="col-data">{{ formatGpu(item.gpuCapacity) }}</td>
              <td class="col-data text-primary">{{ formatGpu(item.gpuLimit) }}</td>
              <td class="col-data text-success">{{ formatGpu(item.gpuAllocated) }}</td>
              <td class="col-data">{{ formatStorage(item.storageLimitGib) }}</td>
              <td class="col-data text-success">{{ formatStorage(item.storageAllocatedGib) }}</td>
              <td class="col-data">{{ item.podsLimit }} 个</td>
              <td class="col-data text-success">{{ item.podsAllocated }} 个</td>
            </tr>
            <tr v-if="clusterRankingData.items.length === 0">
              <td colspan="15" class="empty-cell">暂无数据</td>
            </tr>
          </tbody>
        </table>

        <!-- 项目排行表格 -->
        <table
          v-else-if="currentDimension === 'project'"
          class="ranking-table ranking-table-project"
        >
          <thead>
            <tr>
              <th class="col-rank">排名</th>
              <th class="col-name">项目名称</th>
              <th
                class="col-belong sortable"
                :class="getSortClass('primaryClusterName')"
                @click="handleSort('primaryClusterName')"
              >
                <span>所属集群</span>
                <span class="sort-icon"
                  ><ArtSvgIcon :icon="getSortIcon('primaryClusterName')"
                /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('cpuAllocated')"
                @click="handleSort('cpuAllocated')"
              >
                <span>分配CPU</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('cpuAllocated')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('cpuCapacity')"
                @click="handleSort('cpuCapacity')"
              >
                <span>CPU总数</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('cpuCapacity')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('memAllocatedGib')"
                @click="handleSort('memAllocatedGib')"
              >
                <span>分配内存</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('memAllocatedGib')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('memCapacityGib')"
                @click="handleSort('memCapacityGib')"
              >
                <span>内存总数</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('memCapacityGib')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('gpuAllocated')"
                @click="handleSort('gpuAllocated')"
              >
                <span>分配GPU</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('gpuAllocated')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('gpuCapacity')"
                @click="handleSort('gpuCapacity')"
              >
                <span>GPU总数</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('gpuCapacity')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('storageLimitGib')"
                @click="handleSort('storageLimitGib')"
              >
                <span>存储总量</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('storageLimitGib')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('podsLimit')"
                @click="handleSort('podsLimit')"
              >
                <span>Pod总量</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('podsLimit')" /></span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(item, index) in projectRankingData.items"
              :key="`${item.projectId}-${item.primaryClusterUuid}`"
            >
              <td class="col-rank">
                <div class="rank-badge" :class="getRankClass(index + 1)">{{ index + 1 }}</div>
              </td>
              <td class="col-name">
                <div class="item-name">{{ item.projectName }}</div>
              </td>
              <td class="col-belong">
                <span v-if="item.primaryClusterName" class="cluster-tag">{{
                  item.primaryClusterName
                }}</span>
                <span v-else class="text-gray-400">-</span>
              </td>
              <td class="col-data text-success">{{ formatCpu(item.cpuAllocated) }}</td>
              <td class="col-data">{{ formatCpu(item.cpuCapacity) }}</td>
              <td class="col-data text-success">{{ formatMem(item.memAllocatedGib) }}</td>
              <td class="col-data">{{ formatMem(item.memCapacityGib) }}</td>
              <td class="col-data text-success">{{ formatGpu(item.gpuAllocated) }}</td>
              <td class="col-data">{{ formatGpu(item.gpuCapacity) }}</td>
              <td class="col-data">{{ formatStorage(item.storageLimitGib) }}</td>
              <td class="col-data">{{ item.podsLimit }} 个</td>
            </tr>
            <tr v-if="projectRankingData.items.length === 0">
              <td colspan="11" class="empty-cell">暂无数据</td>
            </tr>
          </tbody>
        </table>

        <!-- 工作空间排行表格 -->
        <table v-else class="ranking-table ranking-table-workspace">
          <thead>
            <tr>
              <th class="col-rank">排名</th>
              <th class="col-belong">所属集群</th>
              <th
                class="col-belong sortable"
                :class="getSortClass('projectName')"
                @click="handleSort('projectName')"
              >
                <span>所属项目</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('projectName')" /></span>
              </th>
              <th
                class="col-name sortable"
                :class="getSortClass('workspaceName')"
                @click="handleSort('workspaceName')"
              >
                <span>工作空间</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('workspaceName')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('cpuAllocated')"
                @click="handleSort('cpuAllocated')"
              >
                <span>CPU配额</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('cpuAllocated')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('memAllocatedGib')"
                @click="handleSort('memAllocatedGib')"
              >
                <span>内存配额</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('memAllocatedGib')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('gpuAllocated')"
                @click="handleSort('gpuAllocated')"
              >
                <span>GPU配额</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('gpuAllocated')" /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('storageAllocatedGib')"
                @click="handleSort('storageAllocatedGib')"
              >
                <span>存储配额</span>
                <span class="sort-icon"
                  ><ArtSvgIcon :icon="getSortIcon('storageAllocatedGib')"
                /></span>
              </th>
              <th
                class="col-data sortable"
                :class="getSortClass('podsAllocated')"
                @click="handleSort('podsAllocated')"
              >
                <span>Pod配额</span>
                <span class="sort-icon"><ArtSvgIcon :icon="getSortIcon('podsAllocated')" /></span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, index) in workspaceRankingData.items" :key="item.workspaceId">
              <td class="col-rank">
                <div class="rank-badge" :class="getRankClass(index + 1)">{{ index + 1 }}</div>
              </td>
              <td class="col-belong">
                <span v-if="item.clusterName" class="cluster-tag">{{ item.clusterName }}</span>
                <span v-else class="text-gray-400">-</span>
              </td>
              <td class="col-belong">{{ item.projectName || '-' }}</td>
              <td class="col-name">
                <div class="item-name">{{ item.workspaceName }}</div>
                <div class="item-sub">{{ item.namespace }}</div>
              </td>
              <td class="col-data">{{ formatCpu(item.cpuAllocated) }}</td>
              <td class="col-data">{{ formatMem(item.memAllocatedGib) }}</td>
              <td class="col-data">{{ formatGpu(item.gpuAllocated) }}</td>
              <td class="col-data">{{ formatStorage(item.storageAllocatedGib) }}</td>
              <td class="col-data">{{ item.podsAllocated }} 个</td>
            </tr>
            <tr v-if="workspaceRankingData.items.length === 0">
              <td colspan="9" class="empty-cell">暂无数据</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, onMounted } from 'vue'
  import {
    getResourceDashboardSummaryApi,
    getClusterResourceRankingApi,
    getProjectResourceRankingApi,
    getWorkspaceResourceRankingApi,
    type GetResourceDashboardSummaryResponse,
    type GetClusterResourceRankingResponse,
    type GetProjectResourceRankingResponse,
    type GetWorkspaceResourceRankingResponse
  } from '@/api/manager/resource-dashboard'
  import { searchClusterApi, type Cluster } from '@/api/manager/cluster'
  import { getProjectsByUserApi, type Project } from '@/api/manager/project'

  defineOptions({ name: 'ResourceDashboard' })

  type DimensionType = 'cluster' | 'project' | 'workspace'
  type SortDirection = 'asc' | 'desc'

  const filterForm = reactive({ clusterUuid: '', projectId: 0 })
  const clusterList = ref<Cluster[]>([])
  const projectList = ref<Project[]>([])
  const currentDimension = ref<DimensionType>('project')
  const topN = ref(10)

  // 排序状态 - 每个维度独立
  const clusterSortKey = ref<string>('cpuCapacity')
  const clusterSortDirection = ref<SortDirection>('desc')
  const projectSortKey = ref<string>('cpuCapacity')
  const projectSortDirection = ref<SortDirection>('desc')
  const workspaceSortKey = ref<string>('cpuAllocated')
  const workspaceSortDirection = ref<SortDirection>('desc')

  // 当前排序状态的 getter
  const getSortKey = () => {
    switch (currentDimension.value) {
      case 'cluster':
        return clusterSortKey.value
      case 'project':
        return projectSortKey.value
      case 'workspace':
        return workspaceSortKey.value
      default:
        return 'cpuCapacity'
    }
  }

  const getSortDirection = () => {
    switch (currentDimension.value) {
      case 'cluster':
        return clusterSortDirection.value
      case 'project':
        return projectSortDirection.value
      case 'workspace':
        return workspaceSortDirection.value
      default:
        return 'desc'
    }
  }

  const setSortState = (key: string, direction: SortDirection) => {
    switch (currentDimension.value) {
      case 'cluster':
        clusterSortKey.value = key
        clusterSortDirection.value = direction
        break
      case 'project':
        projectSortKey.value = key
        projectSortDirection.value = direction
        break
      case 'workspace':
        workspaceSortKey.value = key
        workspaceSortDirection.value = direction
        break
    }
  }

  const dimensionOptions = [
    { label: '集群', value: 'cluster' as DimensionType },
    { label: '项目', value: 'project' as DimensionType },
    { label: '工作空间', value: 'workspace' as DimensionType }
  ]

  const defaultOverviewItem = { physical: 0, limit: 0, capacity: 0, unit: '' }
  const defaultSavingItem = {
    limitTotal: 0,
    capacityTotal: 0,
    savingAmount: 0,
    savingRate: 0,
    unit: ''
  }

  const getDefaultSummaryData = (): GetResourceDashboardSummaryResponse => ({
    currentFilter: {
      clusterUuid: '',
      clusterName: '全部集群',
      projectId: 0,
      projectName: '全部项目',
      filteredClusterCount: 0,
      filteredProjectCount: 0,
      filteredWorkspaceCount: 0
    },
    summaryCards: {
      clusterTotalCount: 0,
      projectTotalCount: 0,
      workspaceTotalCount: 0,
      cpuPhysical: 0,
      cpuLimit: 0,
      cpuPhysicalRate: 0,
      memPhysical: 0,
      memLimit: 0,
      memPhysicalRate: 0,
      gpuPhysical: 0,
      gpuLimit: 0,
      gpuPhysicalRate: 0,
      storagePhysical: 0,
      storageLimit: 0,
      storagePhysicalRate: 0,
      podPhysical: 0,
      podLimit: 0,
      podPhysicalRate: 0
    },
    allocationOverview: {
      cpu: { ...defaultOverviewItem, unit: '核' },
      mem: { ...defaultOverviewItem, unit: 'GiB' },
      gpu: { ...defaultOverviewItem, unit: '卡' },
      storage: { ...defaultOverviewItem, unit: 'GiB' },
      pod: { ...defaultOverviewItem, unit: '个' }
    },
    oversellSavings: {
      cpu: { ...defaultSavingItem, unit: '核' },
      mem: { ...defaultSavingItem, unit: 'GiB' },
      gpu: { ...defaultSavingItem, unit: '卡' }
    }
  })

  const summaryData = ref<GetResourceDashboardSummaryResponse>(getDefaultSummaryData())
  const clusterRankingData = ref<GetClusterResourceRankingResponse>({ items: [], total: 0 })
  const projectRankingData = ref<GetProjectResourceRankingResponse>({ items: [], total: 0 })
  const workspaceRankingData = ref<GetWorkspaceResourceRankingResponse>({ items: [], total: 0 })

  // 格式化函数 - 保留单位
  const formatSavingValue = (val: number) => (val >= 0 ? '+' : '') + val.toFixed(0)
  const formatStorage = (gib: number) =>
    gib >= 1024 ? (gib / 1024).toFixed(1) + ' TiB' : gib.toFixed(1) + ' GiB'
  const formatCpu = (val: number | undefined) =>
    val !== undefined && val !== null ? val.toFixed(1) + ' 核' : '0 核'
  const formatMem = (val: number | undefined) =>
    val !== undefined && val !== null ? val.toFixed(1) + ' GiB' : '0 GiB'
  const formatGpu = (val: number | undefined) =>
    val !== undefined && val !== null ? val.toFixed(1) + ' 卡' : '0 卡'

  // 排名样式
  const getRankClass = (rank: number) =>
    rank === 1
      ? 'rank-gold'
      : rank === 2
        ? 'rank-silver'
        : rank === 3
          ? 'rank-bronze'
          : 'rank-default'
  const getEnvClass = (env: string) =>
    ({
      production: 'env-prod',
      staging: 'env-staging',
      testing: 'env-test',
      development: 'env-dev'
    })[env] || 'env-default'
  const getEnvLabel = (env: string) =>
    ({ production: '生产', staging: '预发', testing: '测试', development: '开发' })[env] || env

  // 排序相关
  const getSortIcon = (key: string) => {
    const currentKey = getSortKey()
    const currentDirection = getSortDirection()
    if (currentKey !== key) return 'ri:arrow-up-down-line'
    return currentDirection === 'desc' ? 'ri:arrow-down-s-fill' : 'ri:arrow-up-s-fill'
  }

  const getSortClass = (key: string) => (getSortKey() === key ? 'sort-active' : '')

  // 点击表头排序 - 调用后端
  const handleSort = async (key: string) => {
    const currentKey = getSortKey()
    const currentDirection = getSortDirection()

    let newDirection: SortDirection = 'desc'
    if (currentKey === key) {
      newDirection = currentDirection === 'desc' ? 'asc' : 'desc'
    }

    setSortState(key, newDirection)

    // 调用后端重新获取数据
    await loadRankingData()
  }

  // 清空排行数据
  const clearRankingData = () => {
    clusterRankingData.value = { items: [], total: 0 }
    projectRankingData.value = { items: [], total: 0 }
    workspaceRankingData.value = { items: [], total: 0 }
  }

  // 加载集群列表
  const loadClusterList = async () => {
    try {
      const res = await searchClusterApi({ pageSize: 100 })
      clusterList.value = res.items || []
    } catch (e) {}
  }

  // 加载项目列表
  const loadProjectList = async () => {
    try {
      const res = await getProjectsByUserApi()
      projectList.value = res || []
    } catch (e) {}
  }

  // 加载汇总数据
  const loadSummaryData = async () => {
    try {
      const res = await getResourceDashboardSummaryApi({
        clusterUuid: filterForm.clusterUuid || undefined,
        projectId: filterForm.projectId || undefined
      })
      summaryData.value = res
    } catch (e) {}
  }

  // 加载排行数据 - 使用后端排序
  const loadRankingData = async () => {
    // 先清空当前维度的数据
    switch (currentDimension.value) {
      case 'cluster':
        clusterRankingData.value = { items: [], total: 0 }
        break
      case 'project':
        projectRankingData.value = { items: [], total: 0 }
        break
      case 'workspace':
        workspaceRankingData.value = { items: [], total: 0 }
        break
    }

    try {
      const sortKey = getSortKey()
      const sortDirection = getSortDirection()
      // 组合排序字段和方向，格式: "fieldName:direction"
      const sortBy = `${sortKey}:${sortDirection}`

      if (currentDimension.value === 'cluster') {
        const res = await getClusterResourceRankingApi({
          projectId: filterForm.projectId || undefined,
          sortBy: sortBy,
          topN: topN.value
        })
        clusterRankingData.value = res
      } else if (currentDimension.value === 'project') {
        const res = await getProjectResourceRankingApi({
          clusterUuid: filterForm.clusterUuid || undefined,
          sortBy: sortBy,
          topN: topN.value
        })
        projectRankingData.value = res
      } else {
        const res = await getWorkspaceResourceRankingApi({
          clusterUuid: filterForm.clusterUuid || undefined,
          projectId: filterForm.projectId || undefined,
          sortBy: sortBy,
          topN: topN.value
        })
        workspaceRankingData.value = res
      }
    } catch (e) {}
  }

  // 切换维度
  const changeDimension = async (dim: DimensionType) => {
    if (currentDimension.value === dim) return
    currentDimension.value = dim
    // 切换维度时加载对应数据
    await loadRankingData()
  }

  // TopN变化 - 需要重新请求后端
  const handleTopNChange = async () => {
    await loadRankingData()
  }

  // 筛选变化
  const handleFilterChange = () => refreshAllData()

  // 刷新所有数据
  const refreshAllData = async () => {
    // 清空所有排行数据
    clearRankingData()
    await Promise.all([loadSummaryData(), loadRankingData()])
  }

  onMounted(async () => {
    await Promise.all([loadClusterList(), loadProjectList()])
    await refreshAllData()
  })
</script>

<style scoped>
  .resource-dashboard {
    padding: 0;
  }
  .dashboard-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 24px;
  }
  .page-title {
    font-size: 24px;
    font-weight: 600;
    color: #1f2937;
    margin: 0;
  }
  .page-subtitle {
    font-size: 14px;
    color: #6b7280;
    margin-top: 4px;
  }
  .header-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .top-stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 24px;
  }
  .stat-card {
    position: relative;
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 20px;
    border-radius: 16px;
    color: white;
    overflow: hidden;
  }
  .stat-card::before {
    content: '';
    position: absolute;
    top: 0;
    right: 0;
    width: 100px;
    height: 100px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 50%;
    transform: translate(30%, -30%);
  }
  .stat-card-blue {
    background: linear-gradient(135deg, #3b82f6, #60a5fa);
  }
  .stat-card-purple {
    background: linear-gradient(135deg, #8b5cf6, #a78bfa);
  }
  .stat-card-cyan {
    background: linear-gradient(135deg, #06b6d4, #22d3ee);
  }
  .stat-card-pink {
    background: linear-gradient(135deg, #ec4899, #f472b6);
  }
  .stat-card-green {
    background: linear-gradient(135deg, #10b981, #34d399);
  }
  .stat-card-violet {
    background: linear-gradient(135deg, #7c3aed, #a78bfa);
  }
  .stat-card-indigo {
    background: linear-gradient(135deg, #6366f1, #818cf8);
  }
  .stat-card-amber {
    background: linear-gradient(135deg, #f59e0b, #fbbf24);
  }
  .stat-card-icon {
    width: 48px;
    height: 48px;
    background: rgba(255, 255, 255, 0.2);
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 22px;
    flex-shrink: 0;
  }
  .stat-card-content {
    position: relative;
    z-index: 1;
    min-width: 0;
  }
  .stat-card-label {
    font-size: 13px;
    opacity: 0.9;
    margin-bottom: 4px;
    display: block;
  }
  .stat-card-value {
    font-size: 28px;
    font-weight: 700;
    line-height: 1.2;
    color: white;
  }
  .stat-card-sub {
    font-size: 12px;
    opacity: 0.8;
    margin-top: 4px;
    display: block;
  }

  .saving-card,
  .detail-card {
    padding: 24px;
    height: 450px;
    display: flex;
    flex-direction: column;
  }
  .card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 20px;
    flex-shrink: 0;
  }
  .header-icon {
    font-size: 18px;
    color: #6b7280;
  }
  .header-title {
    font-size: 16px;
    font-weight: 600;
    color: #374151;
  }
  .saving-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
    flex: 1;
  }
  .saving-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px 20px;
    border-radius: 16px;
    transition: transform 0.2s;
    flex: 1;
  }
  .saving-row:hover {
    transform: translateX(4px);
  }
  .saving-row-green {
    background: linear-gradient(135deg, rgba(16, 185, 129, 0.12), rgba(52, 211, 153, 0.06));
    border: 1px solid rgba(16, 185, 129, 0.2);
  }
  .saving-row-blue {
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.12), rgba(96, 165, 250, 0.06));
    border: 1px solid rgba(59, 130, 246, 0.2);
  }
  .saving-row-purple {
    background: linear-gradient(135deg, rgba(139, 92, 246, 0.12), rgba(167, 139, 250, 0.06));
    border: 1px solid rgba(139, 92, 246, 0.2);
  }
  .saving-icon-wrapper {
    width: 44px;
    height: 44px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    flex-shrink: 0;
  }
  .saving-icon-green {
    background: linear-gradient(135deg, #10b981, #34d399);
    color: white;
  }
  .saving-icon-blue {
    background: linear-gradient(135deg, #3b82f6, #60a5fa);
    color: white;
  }
  .saving-icon-purple {
    background: linear-gradient(135deg, #8b5cf6, #a78bfa);
    color: white;
  }
  .saving-info {
    flex: 1;
    min-width: 0;
  }
  .saving-label {
    font-size: 13px;
    color: #6b7280;
    margin-bottom: 4px;
  }
  .saving-value {
    font-size: 24px;
    font-weight: 700;
    color: #1f2937;
    line-height: 1.2;
  }
  .saving-detail {
    font-size: 12px;
    color: #9ca3af;
    margin-top: 4px;
  }
  .saving-rate-tag {
    padding: 6px 14px;
    border-radius: 20px;
    font-size: 13px;
    font-weight: 600;
    white-space: nowrap;
  }
  .saving-rate-green {
    background: rgba(16, 185, 129, 0.15);
    color: #059669;
  }
  .saving-rate-blue {
    background: rgba(59, 130, 246, 0.15);
    color: #2563eb;
  }
  .saving-rate-purple {
    background: rgba(139, 92, 246, 0.15);
    color: #7c3aed;
  }

  .detail-table {
    font-size: 14px;
    flex: 1;
    display: flex;
    flex-direction: column;
  }
  .detail-row {
    display: grid;
    grid-template-columns: 100px 1fr 1fr 1fr;
    padding: 14px 16px;
    border-bottom: 1px solid #f3f4f6;
    align-items: center;
  }
  .detail-row:last-child {
    border-bottom: none;
  }
  .detail-row.header-row {
    font-weight: 500;
    color: #6b7280;
    font-size: 13px;
    background: #f9fafb;
    border-radius: 10px;
    margin-bottom: 8px;
    border-bottom: none;
  }
  .detail-cell {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .cell-name {
    font-weight: 500;
    color: #374151;
  }
  .cell-value {
    justify-content: flex-end;
    font-family: var(--ax-font-mono);
    color: #4b5563;
  }
  .resource-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .ranking-card {
    padding: 24px;
  }
  .ranking-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }
  .ranking-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 16px;
    font-weight: 600;
    color: #374151;
  }
  .ranking-controls {
    display: flex;
    gap: 12px;
    align-items: center;
  }
  .btn-group {
    display: flex;
    background: #f3f4f6;
    border-radius: 8px;
    padding: 4px;
  }
  .btn-item {
    padding: 6px 14px;
    font-size: 13px;
    border: none;
    background: transparent;
    color: #6b7280;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;
  }
  .btn-item:hover {
    color: #374151;
  }
  .btn-item.active {
    background: white;
    color: var(--el-color-primary);
    font-weight: 500;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  /* 表格容器 - 固定高度防止切换抖动，超出可滚动 */
  .ranking-table-wrapper {
    height: 480px; /* 固定高度，约可显示10行数据 */
    overflow: auto;
    border: 1px solid #e5e7eb;
    border-radius: 12px;
  }
  .ranking-table {
    width: 100%;
    border-collapse: collapse;
  }
  .ranking-table-cluster {
    min-width: 1500px;
  }
  .ranking-table-project {
    min-width: 1100px;
  }
  .ranking-table-workspace {
    min-width: 950px;
  }
  .ranking-table thead {
    position: sticky;
    top: 0;
    z-index: 10;
  }
  .ranking-table th {
    padding: 12px 8px;
    font-size: 12px;
    font-weight: 500;
    color: #6b7280;
    background: #f9fafb;
    border-bottom: 1px solid #e5e7eb;
    text-align: center;
    white-space: nowrap;
  }
  .ranking-table td {
    padding: 12px 8px;
    border-bottom: 1px solid #f3f4f6;
    font-size: 13px;
    text-align: center;
    font-family: var(--ax-font-mono);
  }
  .ranking-table tbody tr:hover {
    background: #fafafa;
  }
  .ranking-table tbody tr:last-child td {
    border-bottom: none;
  }

  .sortable {
    cursor: pointer;
    user-select: none;
    transition: background 0.2s;
  }
  .sortable:hover {
    background: #f3f4f6;
  }
  .sortable .sort-icon {
    margin-left: 2px;
    font-size: 14px;
    color: #9ca3af;
    vertical-align: middle;
    display: inline-flex;
  }
  .sort-active {
    background: #e0e7ff !important;
    color: var(--el-color-primary) !important;
  }
  .sort-active .sort-icon {
    color: var(--el-color-primary);
  }

  .col-rank {
    width: 60px;
    text-align: center !important;
  }
  .col-name {
    width: 140px;
    text-align: left !important;
  }
  .col-name-wide {
    width: 180px;
    text-align: left !important;
  }
  .col-belong {
    width: 120px;
    text-align: center !important;
  }
  .col-data {
    width: 85px;
    text-align: center !important;
  }
  .ranking-table th.col-name,
  .ranking-table td.col-name,
  .ranking-table th.col-name-wide,
  .ranking-table td.col-name-wide {
    text-align: left;
  }
  .ranking-table td.col-name,
  .ranking-table td.col-name-wide,
  .ranking-table td.col-belong {
    font-family: inherit;
  }

  .item-name {
    font-weight: 500;
    color: #374151;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .item-sub {
    font-size: 11px;
    color: #9ca3af;
    margin-top: 2px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .cluster-tag {
    display: inline-block;
    padding: 3px 10px;
    background: linear-gradient(135deg, #dbeafe, #e0e7ff);
    color: #3b82f6;
    border-radius: 20px;
    font-size: 11px;
    font-weight: 500;
    max-width: 100px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .rank-badge {
    width: 26px;
    height: 26px;
    border-radius: 50%;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 600;
    font-family: inherit;
  }
  .rank-gold {
    background: linear-gradient(135deg, #fbbf24, #f59e0b);
    color: white;
  }
  .rank-silver {
    background: linear-gradient(135deg, #9ca3af, #6b7280);
    color: white;
  }
  .rank-bronze {
    background: linear-gradient(135deg, #d97706, #b45309);
    color: white;
  }
  .rank-default {
    background: #f3f4f6;
    color: #6b7280;
  }
  .env-tag {
    padding: 2px 8px;
    border-radius: 12px;
    font-size: 10px;
    font-weight: 500;
    margin-left: 6px;
  }
  .env-prod {
    background: #fee2e2;
    color: #dc2626;
  }
  .env-staging {
    background: #fef3c7;
    color: #d97706;
  }
  .env-test {
    background: #dbeafe;
    color: #2563eb;
  }
  .env-dev {
    background: #d1fae5;
    color: #059669;
  }

  .text-primary {
    color: var(--el-color-primary) !important;
  }
  .text-success {
    color: #059669 !important;
  }
  .text-gray-400 {
    color: #9ca3af;
  }
  .empty-cell {
    text-align: center;
    padding: 40px;
    color: #9ca3af;
    font-family: inherit;
  }

  .bg-orange-400 {
    background-color: #fb923c;
  }
  .bg-green-400 {
    background-color: #4ade80;
  }
  .bg-amber-400 {
    background-color: #fbbf24;
  }
  .bg-indigo-400 {
    background-color: #818cf8;
  }
  .bg-cyan-400 {
    background-color: #22d3ee;
  }
  .text-green-500 {
    color: #10b981;
  }
  .text-blue-500 {
    color: #3b82f6;
  }
</style>

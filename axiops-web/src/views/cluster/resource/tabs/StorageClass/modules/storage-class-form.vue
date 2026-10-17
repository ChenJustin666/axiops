<template>
  <ElForm
    ref="formRef"
    :model="formData"
    :rules="rules"
    label-width="130px"
    :disabled="disabled"
    class="storage-class-form"
  >
    <ElCollapse v-model="activeCollapse" class="form-collapse">
      <!-- 基本信息 -->
      <ElCollapseItem name="basic" class="collapse-section-item">
        <template #title>
          <div class="collapse-title">
            <ElIcon class="title-icon"><InfoFilled /></ElIcon>
            <span>基本信息</span>
          </div>
        </template>
        <div class="collapse-content">
          <ElRow :gutter="24">
            <ElCol :span="12">
              <ElFormItem label="名称" prop="name">
                <ElInput
                  v-model="formData.name"
                  placeholder="请输入 StorageClass 名称"
                  :disabled="isEdit"
                />
                <div class="form-item-tip">
                  只能包含小写字母、数字和连字符，且必须以字母数字开头和结尾
                </div>
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="供应商" prop="provisioner">
                <ElSelect
                  v-model="formData.provisioner"
                  placeholder="选择存储供应商"
                  filterable
                  :allow-create="!isEdit"
                  :disabled="isEdit"
                  style="width: 100%"
                  @change="handleProvisionerChange"
                >
                  <ElOptionGroup
                    v-for="group in provisionerGroups"
                    :key="group.label"
                    :label="group.label"
                  >
                    <ElOption
                      v-for="item in group.options"
                      :key="item.value"
                      :label="item.label"
                      :value="item.value"
                    />
                  </ElOptionGroup>
                </ElSelect>
                <div v-if="isEdit" class="form-item-tip warning">编辑模式下不允许修改供应商</div>
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElRow :gutter="24">
            <ElCol :span="12">
              <ElFormItem label="回收策略" prop="reclaimPolicy">
                <ElRadioGroup v-model="formData.reclaimPolicy">
                  <ElRadio value="Delete">Delete（删除卷）</ElRadio>
                  <ElRadio value="Retain">Retain（保留卷）</ElRadio>
                </ElRadioGroup>
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="绑定模式" prop="volumeBindingMode">
                <ElRadioGroup v-model="formData.volumeBindingMode">
                  <ElRadio value="Immediate">Immediate（立即绑定）</ElRadio>
                  <ElRadio value="WaitForFirstConsumer">WaitForFirstConsumer（延迟绑定）</ElRadio>
                </ElRadioGroup>
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElRow :gutter="24">
            <ElCol :span="12">
              <ElFormItem label="允许扩容">
                <ElSwitch
                  v-model="formData.allowVolumeExpansion"
                  active-text="是"
                  inactive-text="否"
                  inline-prompt
                />
                <span class="switch-tip">开启后可在不重建 PVC 的情况下扩容</span>
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="设为默认">
                <ElSwitch
                  v-model="formData.isDefault"
                  active-text="是"
                  inactive-text="否"
                  inline-prompt
                />
                <span class="switch-tip">未指定 StorageClass 的 PVC 将使用此存储类</span>
              </ElFormItem>
            </ElCol>
          </ElRow>
        </div>
      </ElCollapseItem>

      <!-- 自定义参数 -->
      <ElCollapseItem name="custom" class="collapse-section-item">
        <template #title>
          <div class="collapse-title">
            <ElIcon class="title-icon"><Setting /></ElIcon>
            <span>自定义参数</span>
          </div>
        </template>
        <div class="collapse-content">
          <div v-if="currentProvisionerConfig" class="form-section">
            <div class="section-header">{{ currentProvisionerConfig.name }} 参数配置</div>

            <!-- NFS 参数 -->
            <template v-if="provisionerType === 'nfs'">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="NFS 服务器">
                    <ElInput
                      v-model="provisionerParams.nfsServer"
                      placeholder="NFS 服务器地址，如: 192.168.1.100"
                    />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="共享路径">
                    <ElInput
                      v-model="provisionerParams.nfsPath"
                      placeholder="NFS 共享路径，如: /exports/data"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </template>

            <!-- OpenEBS Local PV 参数 -->
            <template v-else-if="provisionerType === 'openebs-local'">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="存储类型">
                    <ElSelect v-model="provisionerParams.openebsLocalType" style="width: 100%">
                      <ElOption label="HostPath (主机路径)" value="hostpath" />
                      <ElOption label="Device (块设备)" value="device" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem
                    label="基础路径"
                    v-if="provisionerParams.openebsLocalType === 'hostpath'"
                  >
                    <ElInput
                      v-model="provisionerParams.openebsLocalBasePath"
                      placeholder="如: /var/openebs/local"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </template>

            <!-- OpenEBS cStor 参数 -->
            <template v-else-if="provisionerType === 'openebs-cstor'">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="cStor Pool 集群">
                    <ElInput
                      v-model="provisionerParams.cstorPoolCluster"
                      placeholder="cStor Pool Cluster 名称"
                    />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="副本数">
                    <ElInputNumber
                      v-model="provisionerParams.cstorReplicaCount"
                      :min="1"
                      :max="5"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </template>

            <!-- OpenEBS LVM 参数 -->
            <template v-else-if="provisionerType === 'openebs-lvm'">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="卷组名称">
                    <ElInput v-model="provisionerParams.lvmVolgroup" placeholder="LVM 卷组名称" />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="文件系统类型">
                    <ElSelect v-model="provisionerParams.lvmFsType" style="width: 100%">
                      <ElOption label="ext4" value="ext4" />
                      <ElOption label="xfs" value="xfs" />
                      <ElOption label="btrfs" value="btrfs" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="共享卷">
                    <ElSelect v-model="provisionerParams.lvmShared" style="width: 100%">
                      <ElOption label="否" value="no" />
                      <ElOption label="是" value="yes" />
                    </ElSelect>
                    <div class="form-item-tip">是否允许多个 Pod 同时访问</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </template>

            <!-- OpenEBS ZFS 参数 -->
            <template v-else-if="provisionerType === 'openebs-zfs'">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="ZFS 池名称">
                    <ElInput v-model="provisionerParams.zfsPoolname" placeholder="ZFS 存储池名称" />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="文件系统类型">
                    <ElSelect v-model="provisionerParams.zfsFsType" style="width: 100%">
                      <ElOption label="zfs" value="zfs" />
                      <ElOption label="ext4" value="ext4" />
                      <ElOption label="xfs" value="xfs" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="压缩">
                    <ElSelect v-model="provisionerParams.zfsCompression" style="width: 100%">
                      <ElOption label="关闭" value="off" />
                      <ElOption label="lz4" value="lz4" />
                      <ElOption label="gzip" value="gzip" />
                      <ElOption label="zstd" value="zstd" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="去重">
                    <ElSwitch v-model="provisionerParams.zfsDedup" />
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="记录大小">
                    <ElSelect v-model="provisionerParams.zfsRecordsize" style="width: 100%">
                      <ElOption label="4k" value="4k" />
                      <ElOption label="8k" value="8k" />
                      <ElOption label="16k" value="16k" />
                      <ElOption label="32k" value="32k" />
                      <ElOption label="64k" value="64k" />
                      <ElOption label="128k (默认)" value="128k" />
                      <ElOption label="256k" value="256k" />
                      <ElOption label="512k" value="512k" />
                      <ElOption label="1m" value="1m" />
                    </ElSelect>
                    <div class="form-item-tip">ZFS 数据块大小</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </template>

            <!-- Longhorn 参数 -->
            <template v-else-if="provisionerType === 'longhorn'">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="副本数">
                    <ElInputNumber
                      v-model="provisionerParams.longhornNumberOfReplicas"
                      :min="1"
                      :max="5"
                    />
                    <div class="form-item-tip">数据副本数量，建议生产环境至少 2 个</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="文件系统类型">
                    <ElSelect v-model="provisionerParams.longhornFsType" style="width: 100%">
                      <ElOption label="ext4" value="ext4" />
                      <ElOption label="xfs" value="xfs" />
                    </ElSelect>
                    <div class="form-item-tip">卷的文件系统类型</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="数据本地性">
                    <ElSelect v-model="provisionerParams.longhornDataLocality" style="width: 100%">
                      <ElOption label="disabled (禁用)" value="disabled" />
                      <ElOption label="best-effort (尽力而为)" value="best-effort" />
                      <ElOption label="strict-local (严格本地)" value="strict-local" />
                    </ElSelect>
                    <div class="form-item-tip">控制数据副本是否优先放置在工作负载所在节点</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="副本自动平衡">
                    <ElSelect
                      v-model="provisionerParams.longhornReplicaAutoBalance"
                      style="width: 100%"
                    >
                      <ElOption label="ignored (使用全局设置)" value="ignored" />
                      <ElOption label="disabled (禁用)" value="disabled" />
                      <ElOption label="least-effort (最小努力)" value="least-effort" />
                      <ElOption label="best-effort (尽力而为)" value="best-effort" />
                    </ElSelect>
                    <div class="form-item-tip">自动在节点间平衡副本分布</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="Stale 副本超时">
                    <div class="input-with-suffix">
                      <ElInputNumber
                        v-model="provisionerParams.longhornStaleReplicaTimeout"
                        :min="1"
                        :max="2880"
                      />
                      <span class="input-suffix">分钟</span>
                    </div>
                    <div class="form-item-tip">副本被标记为过期前的等待时间</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="数据引擎">
                    <ElSelect v-model="provisionerParams.longhornDataEngine" style="width: 100%">
                      <ElOption label="v1 (默认)" value="v1" />
                      <ElOption label="v2 (实验性)" value="v2" />
                    </ElSelect>
                    <div class="form-item-tip">Longhorn 数据引擎版本</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="磁盘选择器">
                    <ElInput
                      v-model="provisionerParams.longhornDiskSelector"
                      placeholder="例如：ssd,fast"
                    />
                    <div class="form-item-tip">逗号分隔的磁盘标签，用于选择特定磁盘</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="节点选择器">
                    <ElInput
                      v-model="provisionerParams.longhornNodeSelector"
                      placeholder="例如：storage,fast"
                    />
                    <div class="form-item-tip">逗号分隔的节点标签，用于选择特定节点</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="副本软反亲和">
                    <ElSelect
                      v-model="provisionerParams.longhornReplicaSoftAntiAffinity"
                      style="width: 100%"
                    >
                      <ElOption label="ignored (使用全局设置)" value="ignored" />
                      <ElOption label="enabled (启用)" value="enabled" />
                      <ElOption label="disabled (禁用)" value="disabled" />
                    </ElSelect>
                    <div class="form-item-tip">允许副本调度到同一节点</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="副本区域软反亲和">
                    <ElSelect
                      v-model="provisionerParams.longhornReplicaZoneSoftAntiAffinity"
                      style="width: 100%"
                    >
                      <ElOption label="ignored (使用全局设置)" value="ignored" />
                      <ElOption label="enabled (启用)" value="enabled" />
                      <ElOption label="disabled (禁用)" value="disabled" />
                    </ElSelect>
                    <div class="form-item-tip">允许副本调度到同一区域</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="禁用修订计数器">
                    <ElSelect
                      v-model="provisionerParams.longhornDisableRevisionCounter"
                      style="width: 100%"
                    >
                      <ElOption label="ignored (使用全局设置)" value="ignored" />
                      <ElOption label="true (禁用)" value="true" />
                      <ElOption label="false (启用)" value="false" />
                    </ElSelect>
                    <div class="form-item-tip">禁用可提高性能但降低数据完整性保护</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="可迁移">
                    <ElSelect v-model="provisionerParams.longhornMigratable" style="width: 100%">
                      <ElOption label="false (不可迁移)" value="false" />
                      <ElOption label="true (可迁移)" value="true" />
                    </ElSelect>
                    <div class="form-item-tip">是否允许卷在节点间实时迁移</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="从备份恢复">
                    <ElInput
                      v-model="provisionerParams.longhornFromBackup"
                      placeholder="备份 URL（可选）"
                    />
                    <div class="form-item-tip">从指定备份创建卷</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="后备镜像">
                    <ElInput
                      v-model="provisionerParams.longhornBackingImage"
                      placeholder="后备镜像名称（可选）"
                    />
                    <div class="form-item-tip">使用预先创建的后备镜像</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="24">
                  <ElFormItem label="定期任务选择器">
                    <ElInput
                      v-model="provisionerParams.longhornRecurringJobSelector"
                      type="textarea"
                      :rows="2"
                      placeholder='[{"name":"snapshot-job","isGroup":true}]'
                    />
                    <div class="form-item-tip">JSON 格式的定期快照/备份任务选择器</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </template>

            <!-- Ceph RBD 参数 -->
            <template v-else-if="provisionerType === 'ceph-rbd'">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="集群 ID">
                    <ElInput v-model="provisionerParams.cephClusterID" placeholder="Ceph 集群 ID" />
                    <div class="form-item-tip">Ceph 集群的 fsid</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="存储池">
                    <ElInput v-model="provisionerParams.cephPool" placeholder="RBD 存储池名称" />
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="镜像特性">
                    <ElSelect v-model="provisionerParams.cephImageFeatures" style="width: 100%">
                      <ElOption label="layering" value="layering" />
                      <ElOption label="layering,exclusive-lock" value="layering,exclusive-lock" />
                      <ElOption
                        label="layering,exclusive-lock,object-map,fast-diff"
                        value="layering,exclusive-lock,object-map,fast-diff"
                      />
                    </ElSelect>
                    <div class="form-item-tip">RBD 镜像特性，影响性能和兼容性</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="文件系统类型">
                    <ElSelect v-model="provisionerParams.cephFsType" style="width: 100%">
                      <ElOption label="ext4" value="ext4" />
                      <ElOption label="xfs" value="xfs" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="Provisioner Secret">
                    <ElInput
                      v-model="provisionerParams.cephProvisionerSecretName"
                      placeholder="Secret 名称"
                    />
                    <div class="form-item-tip">用于创建卷的 Ceph 用户凭证</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="Provisioner Secret NS">
                    <ElInput
                      v-model="provisionerParams.cephProvisionerSecretNamespace"
                      placeholder="Secret 命名空间"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="Node Stage Secret">
                    <ElInput
                      v-model="provisionerParams.cephNodeStageSecretName"
                      placeholder="Secret 名称"
                    />
                    <div class="form-item-tip">用于挂载卷的 Ceph 用户凭证</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="Node Stage Secret NS">
                    <ElInput
                      v-model="provisionerParams.cephNodeStageSecretNamespace"
                      placeholder="Secret 命名空间"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="Expand Secret">
                    <ElInput
                      v-model="provisionerParams.cephControllerExpandSecretName"
                      placeholder="Secret 名称（可选）"
                    />
                    <div class="form-item-tip">用于扩容卷的 Ceph 用户凭证</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="Expand Secret NS">
                    <ElInput
                      v-model="provisionerParams.cephControllerExpandSecretNamespace"
                      placeholder="Secret 命名空间"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </template>

            <!-- AWS EBS 参数 -->
            <template v-else-if="provisionerType === 'aws-ebs'">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="卷类型">
                    <ElSelect v-model="provisionerParams.ebsType" style="width: 100%">
                      <ElOption label="gp3 (通用 SSD)" value="gp3" />
                      <ElOption label="gp2 (通用 SSD 旧版)" value="gp2" />
                      <ElOption label="io1 (预置 IOPS SSD)" value="io1" />
                      <ElOption label="io2 (预置 IOPS SSD)" value="io2" />
                      <ElOption label="sc1 (冷 HDD)" value="sc1" />
                      <ElOption label="st1 (吞吐优化 HDD)" value="st1" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="文件系统类型">
                    <ElSelect v-model="provisionerParams.ebsFsType" style="width: 100%">
                      <ElOption label="ext4" value="ext4" />
                      <ElOption label="xfs" value="xfs" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24" v-if="provisionerParams.ebsType === 'gp3'">
                <ElCol :span="12">
                  <ElFormItem label="IOPS">
                    <ElInputNumber v-model="provisionerParams.ebsIops" :min="3000" :max="16000" />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="吞吐量 (MiB/s)">
                    <ElInputNumber
                      v-model="provisionerParams.ebsThroughput"
                      :min="125"
                      :max="1000"
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="加密">
                    <ElSwitch v-model="provisionerParams.ebsEncrypted" />
                    <div class="form-item-tip">启用 EBS 卷加密</div>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12" v-if="provisionerParams.ebsEncrypted">
                  <ElFormItem label="KMS Key ID">
                    <ElInput
                      v-model="provisionerParams.ebsKmsKeyId"
                      placeholder="KMS 密钥 ID（可选）"
                    />
                    <div class="form-item-tip">留空使用默认 AWS 托管密钥</div>
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </template>

            <!-- Local Path Provisioner 参数 -->
            <template v-else-if="provisionerType === 'local-path'">
              <ElFormItem label="节点路径映射">
                <ElTable :data="provisionerParams.localPathNodePaths" border style="width: 100%">
                  <ElTableColumn label="节点名" width="200">
                    <template #default="{ row }">
                      <ElInput v-model="row.node" placeholder="留空表示所有节点" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="存储路径">
                    <template #default="{ row }">
                      <ElInput v-model="row.path" placeholder="如: /opt/local-path" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="操作" width="60" align="center">
                    <template #default="{ $index }">
                      <ElButton type="danger" link @click="removeNodePath($index)"> 删除 </ElButton>
                    </template>
                  </ElTableColumn>
                </ElTable>
                <ElButton
                  type="primary"
                  @click="addNodePath"
                  plain
                  size="small"
                  style="margin-top: 8px"
                >
                  添加节点路径
                </ElButton>
              </ElFormItem>
            </template>

            <!-- 通用/自定义参数 -->
            <template v-else>
              <ElAlert
                title="此供应商使用通用参数配置，请在下方自定义参数中添加"
                type="info"
                :closable="false"
                style="margin-bottom: 16px"
              />
            </template>
          </div>

          <!-- 自定义参数（所有供应商通用） -->
          <ElFormItem label="存储参数">
            <ElTable :data="formData.parameters" border style="width: 100%">
              <ElTableColumn label="参数名" width="200">
                <template #default="{ row }">
                  <ElInput v-model="row.key" placeholder="参数名" />
                </template>
              </ElTableColumn>
              <ElTableColumn label="参数值">
                <template #default="{ row }">
                  <ElInput v-model="row.value" placeholder="参数值" />
                </template>
              </ElTableColumn>
              <ElTableColumn label="操作" width="60" align="center">
                <template #default="{ $index }">
                  <ElButton type="danger" link @click="removeParameter($index)"> 删除 </ElButton>
                </template>
              </ElTableColumn>
            </ElTable>
            <ElButton
              type="primary"
              @click="addParameter"
              plain
              size="small"
              style="margin-top: 8px"
            >
              添加参数
            </ElButton>
          </ElFormItem>

          <ElFormItem label="挂载选项">
            <div class="mount-options">
              <ElTag
                v-for="(option, index) in formData.mountOptions"
                :key="index"
                closable
                @close="removeMountOption(index)"
                type="info"
              >
                {{ option }}
              </ElTag>
              <ElInput
                v-model="newMountOption"
                placeholder="输入挂载选项后按回车"
                style="width: 200px"
                @keyup.enter="addMountOption"
                size="small"
              >
                <template #append>
                  <ElButton @click="addMountOption">添加</ElButton>
                </template>
              </ElInput>
            </div>
            <div class="form-item-tip">
              常用挂载选项：hard, soft, nfsvers=4, rsize=1048576, wsize=1048576
            </div>
          </ElFormItem>
        </div>
      </ElCollapseItem>

      <!-- 标签和注解（可选） -->
      <ElCollapseItem name="metadata" class="collapse-section-item">
        <template #title>
          <div class="collapse-title">
            <ElIcon class="title-icon"><PriceTag /></ElIcon>
            <span>标签和注解（可选）</span>
          </div>
        </template>
        <div class="collapse-content">
          <ElFormItem label="标签 (Labels)">
            <ElTable :data="formData.labels" border style="width: 100%">
              <ElTableColumn label="标签名" width="200">
                <template #default="{ row }">
                  <ElInput v-model="row.key" placeholder="标签名" />
                </template>
              </ElTableColumn>
              <ElTableColumn label="标签值">
                <template #default="{ row }">
                  <ElInput v-model="row.value" placeholder="标签值" />
                </template>
              </ElTableColumn>
              <ElTableColumn label="操作" width="60" align="center">
                <template #default="{ $index }">
                  <ElButton type="danger" link @click="removeLabel($index)"> 删除 </ElButton>
                </template>
              </ElTableColumn>
            </ElTable>
            <ElButton type="primary" @click="addLabel" plain size="small" style="margin-top: 8px">
              添加标签
            </ElButton>
          </ElFormItem>

          <ElFormItem label="注解 (Annotations)">
            <ElTable :data="formData.annotations" border style="width: 100%">
              <ElTableColumn label="注解名" width="200">
                <template #default="{ row }">
                  <ElInput v-model="row.key" placeholder="注解名" />
                </template>
              </ElTableColumn>
              <ElTableColumn label="注解值">
                <template #default="{ row }">
                  <ElInput v-model="row.value" placeholder="注解值" />
                </template>
              </ElTableColumn>
              <ElTableColumn label="操作" width="60" align="center">
                <template #default="{ $index }">
                  <ElButton type="danger" link @click="removeAnnotation($index)"> 删除 </ElButton>
                </template>
              </ElTableColumn>
            </ElTable>
            <ElButton
              type="primary"
              @click="addAnnotation"
              plain
              size="small"
              style="margin-top: 8px"
            >
              添加注解
            </ElButton>
          </ElFormItem>
        </div>
      </ElCollapseItem>
    </ElCollapse>
  </ElForm>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, nextTick } from 'vue'
  import { InfoFilled, Setting, PriceTag } from '@element-plus/icons-vue'
  import type { FormInstance, FormRules } from 'element-plus'
  import type { StorageClassFormData, KeyValuePair } from './type.ts'

  interface Props {
    modelValue: StorageClassFormData
    disabled?: boolean
    isEdit?: boolean
  }

  interface Emits {
    (e: 'update:modelValue', value: StorageClassFormData): void
  }

  const props = withDefaults(defineProps<Props>(), {
    disabled: false,
    isEdit: false
  })

  const emit = defineEmits<Emits>()

  // 表单引用
  const formRef = ref<FormInstance>()

  // 防止循环更新的标志
  let isInternalUpdate = false

  // 本地表单数据 - 使用 ref 而不是 reactive 来避免深度监听问题
  const formData = ref<StorageClassFormData>({
    name: '',
    provisioner: '',
    reclaimPolicy: 'Delete',
    volumeBindingMode: 'Immediate',
    allowVolumeExpansion: false,
    isDefault: false,
    parameters: [],
    mountOptions: [],
    labels: [],
    annotations: []
  })

  // 供应商特定参数 - 独立管理，不参与双向绑定
  // 注意：为避免字段冲突，不同供应商的同名参数使用前缀区分
  const provisionerParams = reactive<Record<string, any>>({
    // NFS
    nfsServer: '',
    nfsPath: '',
    // OpenEBS Local
    openebsLocalType: 'hostpath',
    openebsLocalBasePath: '/var/openebs/local',
    // OpenEBS cStor
    cstorPoolCluster: '',
    cstorReplicaCount: 3,
    // OpenEBS LVM
    lvmVolgroup: '',
    lvmFsType: 'ext4',
    lvmShared: 'no',
    // OpenEBS ZFS
    zfsPoolname: '',
    zfsFsType: 'zfs',
    zfsCompression: 'off',
    zfsDedup: false,
    zfsRecordsize: '128k',
    // Longhorn
    longhornNumberOfReplicas: 3,
    longhornStaleReplicaTimeout: 30,
    longhornFromBackup: '',
    longhornFsType: 'ext4',
    longhornDataLocality: 'disabled',
    longhornReplicaAutoBalance: 'ignored',
    longhornDiskSelector: '',
    longhornNodeSelector: '',
    longhornRecurringJobSelector: '',
    longhornBackingImage: '',
    longhornDataEngine: 'v1',
    longhornReplicaSoftAntiAffinity: 'ignored',
    longhornReplicaZoneSoftAntiAffinity: 'ignored',
    longhornDisableRevisionCounter: 'ignored',
    longhornMigratable: 'false',
    // Ceph RBD
    cephClusterID: '',
    cephPool: '',
    cephImageFeatures: 'layering',
    cephFsType: 'ext4',
    cephProvisionerSecretName: '',
    cephProvisionerSecretNamespace: '',
    cephNodeStageSecretName: '',
    cephNodeStageSecretNamespace: '',
    cephControllerExpandSecretName: '',
    cephControllerExpandSecretNamespace: '',
    // AWS EBS
    ebsType: 'gp3',
    ebsFsType: 'ext4',
    ebsIops: 3000,
    ebsThroughput: 125,
    ebsEncrypted: false,
    ebsKmsKeyId: '',
    // Local Path
    localPathNodePaths: [{ node: '', path: '/opt/local-path-provisioner' }]
  })

  // 新挂载选项
  const newMountOption = ref('')

  // 折叠面板激活状态
  const activeCollapse = ref(['basic', 'custom', 'metadata'])

  // 供应商分组配置
  const provisionerGroups = [
    {
      label: 'OpenEBS',
      options: [
        { label: 'OpenEBS Local PV', value: 'local.csi.openebs.io', color: '#F15A29' },
        { label: 'OpenEBS cStor CSI', value: 'cstor.csi.openebs.io', color: '#F15A29' },
        { label: 'OpenEBS LVM CSI', value: 'lvm.csi.openebs.io', color: '#F15A29' },
        { label: 'OpenEBS ZFS CSI', value: 'zfs.csi.openebs.io', color: '#F15A29' },
        { label: 'OpenEBS Mayastor', value: 'io.openebs.csi-mayastor', color: '#F15A29' }
      ]
    },
    {
      label: 'NFS',
      options: [
        { label: 'NFS CSI Driver', value: 'nfs.csi.k8s.io', color: '#326CE5' },
        { label: 'NFS Subdir External', value: 'nfs-subdir-external-provisioner', color: '#326CE5' }
      ]
    },
    {
      label: 'Local Storage',
      options: [
        { label: 'Local Path Provisioner', value: 'rancher.io/local-path', color: '#0075A8' },
        {
          label: 'No Provisioner (Static)',
          value: 'kubernetes.io/no-provisioner',
          color: '#909399'
        }
      ]
    },
    {
      label: 'Longhorn',
      options: [{ label: 'Longhorn', value: 'driver.longhorn.io', color: '#5F259F' }]
    },
    {
      label: 'Ceph',
      options: [
        { label: 'Ceph RBD CSI', value: 'rbd.csi.ceph.com', color: '#EF5C55' },
        { label: 'CephFS CSI', value: 'cephfs.csi.ceph.com', color: '#EF5C55' }
      ]
    },
    {
      label: 'Cloud Providers',
      options: [
        { label: 'AWS EBS CSI', value: 'ebs.csi.aws.com', color: '#FF9900' },
        { label: 'GCP PD CSI', value: 'pd.csi.storage.gke.io', color: '#4285F4' },
        { label: 'Azure Disk CSI', value: 'disk.csi.azure.com', color: '#0078D4' },
        { label: 'Azure File CSI', value: 'file.csi.azure.com', color: '#0078D4' }
      ]
    },
    {
      label: 'vSphere',
      options: [{ label: 'vSphere CSI', value: 'csi.vsphere.vmware.com', color: '#717D7E' }]
    }
  ]

  // 供应商配置映射
  const provisionerConfigMap: Record<string, { name: string; type: string }> = {
    'nfs.csi.k8s.io': { name: 'NFS CSI', type: 'nfs' },
    'nfs-subdir-external-provisioner': { name: 'NFS Subdir', type: 'nfs' },
    'local.csi.openebs.io': { name: 'OpenEBS Local PV', type: 'openebs-local' },
    'cstor.csi.openebs.io': { name: 'OpenEBS cStor', type: 'openebs-cstor' },
    'lvm.csi.openebs.io': { name: 'OpenEBS LVM', type: 'openebs-lvm' },
    'zfs.csi.openebs.io': { name: 'OpenEBS ZFS', type: 'openebs-zfs' },
    'driver.longhorn.io': { name: 'Longhorn', type: 'longhorn' },
    'rbd.csi.ceph.com': { name: 'Ceph RBD', type: 'ceph-rbd' },
    'cephfs.csi.ceph.com': { name: 'CephFS', type: 'ceph-fs' },
    'ebs.csi.aws.com': { name: 'AWS EBS', type: 'aws-ebs' },
    'rancher.io/local-path': { name: 'Local Path', type: 'local-path' }
  }

  // 供应商参数键名映射 - 定义哪些参数属于供应商特定参数（使用 K8s StorageClass 中的实际键名）
  const provisionerParamKeys: Record<string, string[]> = {
    nfs: ['server', 'share'],
    'openebs-local': ['openebs.io/cas-type', 'BasePath'],
    'openebs-cstor': ['cstorPoolCluster', 'replicaCount'],
    'openebs-lvm': ['volgroup', 'fsType', 'shared'],
    'openebs-zfs': ['poolname', 'fstype', 'compression', 'dedup', 'recordsize'],
    longhorn: [
      'numberOfReplicas',
      'staleReplicaTimeout',
      'fromBackup',
      'fsType',
      'dataLocality',
      'replicaAutoBalance',
      'diskSelector',
      'nodeSelector',
      'recurringJobSelector',
      'backingImage',
      'dataEngine',
      'replicaSoftAntiAffinity',
      'replicaZoneSoftAntiAffinity',
      'disableRevisionCounter',
      'migratable'
    ],
    'ceph-rbd': [
      'clusterID',
      'pool',
      'imageFeatures',
      'fsType',
      'csi.storage.k8s.io/provisioner-secret-name',
      'csi.storage.k8s.io/provisioner-secret-namespace',
      'csi.storage.k8s.io/node-stage-secret-name',
      'csi.storage.k8s.io/node-stage-secret-namespace',
      'csi.storage.k8s.io/controller-expand-secret-name',
      'csi.storage.k8s.io/controller-expand-secret-namespace'
    ],
    'aws-ebs': ['type', 'fsType', 'iops', 'throughput', 'encrypted', 'kmsKeyId'],
    'local-path': []
  }

  // 当前供应商配置
  const currentProvisionerConfig = computed(() => {
    return provisionerConfigMap[formData.value.provisioner]
  })

  // 当前供应商类型
  const provisionerType = computed(() => {
    return currentProvisionerConfig.value?.type || 'generic'
  })

  // 表单验证规则
  const rules: FormRules = {
    name: [
      { required: true, message: '请输入名称', trigger: 'blur' },
      {
        pattern: /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/,
        message: '名称只能包含小写字母、数字和连字符，且必须以字母数字开头和结尾',
        trigger: 'blur'
      }
    ],
    provisioner: [{ required: true, message: '请选择或输入供应商', trigger: 'change' }]
  }

  // 从 parameters 中提取供应商特定参数
  const extractProvisionerParams = () => {
    const params = formData.value.parameters.reduce(
      (acc, p) => {
        if (p.key) {
          acc[p.key] = p.value
        }
        return acc
      },
      {} as Record<string, string>
    )

    const type = provisionerType.value

    switch (type) {
      case 'nfs':
        provisionerParams.nfsServer = params.server || ''
        provisionerParams.nfsPath = params.share || ''
        break
      case 'openebs-local':
        provisionerParams.openebsLocalType = params['openebs.io/cas-type'] || 'hostpath'
        provisionerParams.openebsLocalBasePath = params.BasePath || '/var/openebs/local'
        break
      case 'openebs-cstor':
        provisionerParams.cstorPoolCluster = params.cstorPoolCluster || ''
        provisionerParams.cstorReplicaCount = parseInt(params.replicaCount) || 3
        break
      case 'openebs-lvm':
        provisionerParams.lvmVolgroup = params.volgroup || ''
        provisionerParams.lvmFsType = params.fsType || 'ext4'
        provisionerParams.lvmShared = params.shared || 'no'
        break
      case 'openebs-zfs':
        provisionerParams.zfsPoolname = params.poolname || ''
        provisionerParams.zfsFsType = params.fstype || 'zfs'
        provisionerParams.zfsCompression = params.compression || 'off'
        provisionerParams.zfsDedup = params.dedup === 'on'
        provisionerParams.zfsRecordsize = params.recordsize || '128k'
        break
      case 'longhorn':
        provisionerParams.longhornNumberOfReplicas = parseInt(params.numberOfReplicas) || 3
        provisionerParams.longhornStaleReplicaTimeout = parseInt(params.staleReplicaTimeout) || 30
        provisionerParams.longhornFromBackup = params.fromBackup || ''
        provisionerParams.longhornFsType = params.fsType || 'ext4'
        provisionerParams.longhornDataLocality = params.dataLocality || 'disabled'
        provisionerParams.longhornReplicaAutoBalance = params.replicaAutoBalance || 'ignored'
        provisionerParams.longhornDiskSelector = params.diskSelector || ''
        provisionerParams.longhornNodeSelector = params.nodeSelector || ''
        provisionerParams.longhornRecurringJobSelector = params.recurringJobSelector || ''
        provisionerParams.longhornBackingImage = params.backingImage || ''
        provisionerParams.longhornDataEngine = params.dataEngine || 'v1'
        provisionerParams.longhornReplicaSoftAntiAffinity =
          params.replicaSoftAntiAffinity || 'ignored'
        provisionerParams.longhornReplicaZoneSoftAntiAffinity =
          params.replicaZoneSoftAntiAffinity || 'ignored'
        provisionerParams.longhornDisableRevisionCounter =
          params.disableRevisionCounter || 'ignored'
        provisionerParams.longhornMigratable = params.migratable || 'false'
        break
      case 'ceph-rbd':
        provisionerParams.cephClusterID = params.clusterID || ''
        provisionerParams.cephPool = params.pool || ''
        provisionerParams.cephImageFeatures = params.imageFeatures || 'layering'
        provisionerParams.cephFsType = params.fsType || 'ext4'
        provisionerParams.cephProvisionerSecretName =
          params['csi.storage.k8s.io/provisioner-secret-name'] || ''
        provisionerParams.cephProvisionerSecretNamespace =
          params['csi.storage.k8s.io/provisioner-secret-namespace'] || ''
        provisionerParams.cephNodeStageSecretName =
          params['csi.storage.k8s.io/node-stage-secret-name'] || ''
        provisionerParams.cephNodeStageSecretNamespace =
          params['csi.storage.k8s.io/node-stage-secret-namespace'] || ''
        provisionerParams.cephControllerExpandSecretName =
          params['csi.storage.k8s.io/controller-expand-secret-name'] || ''
        provisionerParams.cephControllerExpandSecretNamespace =
          params['csi.storage.k8s.io/controller-expand-secret-namespace'] || ''
        break
      case 'aws-ebs':
        provisionerParams.ebsType = params.type || 'gp3'
        provisionerParams.ebsFsType = params.fsType || 'ext4'
        provisionerParams.ebsIops = parseInt(params.iops) || 3000
        provisionerParams.ebsThroughput = parseInt(params.throughput) || 125
        provisionerParams.ebsEncrypted = params.encrypted === 'true'
        provisionerParams.ebsKmsKeyId = params.kmsKeyId || ''
        break
    }
  }

  // 获取供应商参数生成的 parameters
  const getProvisionerGeneratedParams = (): KeyValuePair[] => {
    const result: KeyValuePair[] = []
    const type = provisionerType.value

    switch (type) {
      case 'nfs':
        if (provisionerParams.nfsServer) {
          result.push({ key: 'server', value: provisionerParams.nfsServer })
        }
        if (provisionerParams.nfsPath) {
          result.push({ key: 'share', value: provisionerParams.nfsPath })
        }
        break
      case 'openebs-local':
        if (provisionerParams.openebsLocalType) {
          result.push({ key: 'openebs.io/cas-type', value: provisionerParams.openebsLocalType })
        }
        if (
          provisionerParams.openebsLocalBasePath &&
          provisionerParams.openebsLocalType === 'hostpath'
        ) {
          result.push({ key: 'BasePath', value: provisionerParams.openebsLocalBasePath })
        }
        break
      case 'openebs-cstor':
        if (provisionerParams.cstorPoolCluster) {
          result.push({ key: 'cstorPoolCluster', value: provisionerParams.cstorPoolCluster })
        }
        result.push({
          key: 'replicaCount',
          value: String(provisionerParams.cstorReplicaCount || 3)
        })
        break
      case 'openebs-lvm':
        if (provisionerParams.lvmVolgroup) {
          result.push({ key: 'volgroup', value: provisionerParams.lvmVolgroup })
        }
        if (provisionerParams.lvmFsType) {
          result.push({ key: 'fsType', value: provisionerParams.lvmFsType })
        }
        if (provisionerParams.lvmShared && provisionerParams.lvmShared === 'yes') {
          result.push({ key: 'shared', value: 'yes' })
        }
        break
      case 'openebs-zfs':
        if (provisionerParams.zfsPoolname) {
          result.push({ key: 'poolname', value: provisionerParams.zfsPoolname })
        }
        if (provisionerParams.zfsFsType) {
          result.push({ key: 'fstype', value: provisionerParams.zfsFsType })
        }
        if (provisionerParams.zfsCompression && provisionerParams.zfsCompression !== 'off') {
          result.push({ key: 'compression', value: provisionerParams.zfsCompression })
        }
        if (provisionerParams.zfsDedup) {
          result.push({ key: 'dedup', value: 'on' })
        }
        if (provisionerParams.zfsRecordsize && provisionerParams.zfsRecordsize !== '128k') {
          result.push({ key: 'recordsize', value: provisionerParams.zfsRecordsize })
        }
        break
      case 'longhorn':
        result.push({
          key: 'numberOfReplicas',
          value: String(provisionerParams.longhornNumberOfReplicas || 3)
        })
        if (provisionerParams.longhornStaleReplicaTimeout) {
          result.push({
            key: 'staleReplicaTimeout',
            value: String(provisionerParams.longhornStaleReplicaTimeout)
          })
        }
        if (provisionerParams.longhornFromBackup) {
          result.push({ key: 'fromBackup', value: provisionerParams.longhornFromBackup })
        }
        if (provisionerParams.longhornFsType && provisionerParams.longhornFsType !== 'ext4') {
          result.push({ key: 'fsType', value: provisionerParams.longhornFsType })
        }
        if (
          provisionerParams.longhornDataLocality &&
          provisionerParams.longhornDataLocality !== 'disabled'
        ) {
          result.push({ key: 'dataLocality', value: provisionerParams.longhornDataLocality })
        }
        if (
          provisionerParams.longhornReplicaAutoBalance &&
          provisionerParams.longhornReplicaAutoBalance !== 'ignored'
        ) {
          result.push({
            key: 'replicaAutoBalance',
            value: provisionerParams.longhornReplicaAutoBalance
          })
        }
        if (provisionerParams.longhornDiskSelector) {
          result.push({ key: 'diskSelector', value: provisionerParams.longhornDiskSelector })
        }
        if (provisionerParams.longhornNodeSelector) {
          result.push({ key: 'nodeSelector', value: provisionerParams.longhornNodeSelector })
        }
        if (provisionerParams.longhornRecurringJobSelector) {
          result.push({
            key: 'recurringJobSelector',
            value: provisionerParams.longhornRecurringJobSelector
          })
        }
        if (provisionerParams.longhornBackingImage) {
          result.push({ key: 'backingImage', value: provisionerParams.longhornBackingImage })
        }
        if (provisionerParams.longhornDataEngine && provisionerParams.longhornDataEngine !== 'v1') {
          result.push({ key: 'dataEngine', value: provisionerParams.longhornDataEngine })
        }
        if (
          provisionerParams.longhornReplicaSoftAntiAffinity &&
          provisionerParams.longhornReplicaSoftAntiAffinity !== 'ignored'
        ) {
          result.push({
            key: 'replicaSoftAntiAffinity',
            value: provisionerParams.longhornReplicaSoftAntiAffinity
          })
        }
        if (
          provisionerParams.longhornReplicaZoneSoftAntiAffinity &&
          provisionerParams.longhornReplicaZoneSoftAntiAffinity !== 'ignored'
        ) {
          result.push({
            key: 'replicaZoneSoftAntiAffinity',
            value: provisionerParams.longhornReplicaZoneSoftAntiAffinity
          })
        }
        if (
          provisionerParams.longhornDisableRevisionCounter &&
          provisionerParams.longhornDisableRevisionCounter !== 'ignored'
        ) {
          result.push({
            key: 'disableRevisionCounter',
            value: provisionerParams.longhornDisableRevisionCounter
          })
        }
        if (
          provisionerParams.longhornMigratable &&
          provisionerParams.longhornMigratable === 'true'
        ) {
          result.push({ key: 'migratable', value: 'true' })
        }
        break
      case 'ceph-rbd':
        if (provisionerParams.cephClusterID) {
          result.push({ key: 'clusterID', value: provisionerParams.cephClusterID })
        }
        if (provisionerParams.cephPool) {
          result.push({ key: 'pool', value: provisionerParams.cephPool })
        }
        if (provisionerParams.cephImageFeatures) {
          result.push({ key: 'imageFeatures', value: provisionerParams.cephImageFeatures })
        }
        if (provisionerParams.cephFsType && provisionerParams.cephFsType !== 'ext4') {
          result.push({ key: 'fsType', value: provisionerParams.cephFsType })
        }
        if (provisionerParams.cephProvisionerSecretName) {
          result.push({
            key: 'csi.storage.k8s.io/provisioner-secret-name',
            value: provisionerParams.cephProvisionerSecretName
          })
        }
        if (provisionerParams.cephProvisionerSecretNamespace) {
          result.push({
            key: 'csi.storage.k8s.io/provisioner-secret-namespace',
            value: provisionerParams.cephProvisionerSecretNamespace
          })
        }
        if (provisionerParams.cephNodeStageSecretName) {
          result.push({
            key: 'csi.storage.k8s.io/node-stage-secret-name',
            value: provisionerParams.cephNodeStageSecretName
          })
        }
        if (provisionerParams.cephNodeStageSecretNamespace) {
          result.push({
            key: 'csi.storage.k8s.io/node-stage-secret-namespace',
            value: provisionerParams.cephNodeStageSecretNamespace
          })
        }
        if (provisionerParams.cephControllerExpandSecretName) {
          result.push({
            key: 'csi.storage.k8s.io/controller-expand-secret-name',
            value: provisionerParams.cephControllerExpandSecretName
          })
        }
        if (provisionerParams.cephControllerExpandSecretNamespace) {
          result.push({
            key: 'csi.storage.k8s.io/controller-expand-secret-namespace',
            value: provisionerParams.cephControllerExpandSecretNamespace
          })
        }
        break
      case 'aws-ebs':
        if (provisionerParams.ebsType) {
          result.push({ key: 'type', value: provisionerParams.ebsType })
        }
        if (provisionerParams.ebsFsType) {
          result.push({ key: 'fsType', value: provisionerParams.ebsFsType })
        }
        if (provisionerParams.ebsType === 'gp3') {
          if (provisionerParams.ebsIops) {
            result.push({ key: 'iops', value: String(provisionerParams.ebsIops) })
          }
          if (provisionerParams.ebsThroughput) {
            result.push({ key: 'throughput', value: String(provisionerParams.ebsThroughput) })
          }
        }
        if (provisionerParams.ebsEncrypted) {
          result.push({ key: 'encrypted', value: 'true' })
        }
        if (provisionerParams.ebsKmsKeyId) {
          result.push({ key: 'kmsKeyId', value: provisionerParams.ebsKmsKeyId })
        }
        break
    }

    return result
  }

  // 过滤掉供应商特定参数，只保留自定义参数
  const filterCustomParameters = (parameters: KeyValuePair[]): KeyValuePair[] => {
    const type = provisionerType.value
    const provisionerKeys = provisionerParamKeys[type] || []

    return parameters.filter((p) => {
      // 保留空键的行（用户新添加的）
      if (!p.key) return true
      // 过滤掉属于供应商特定的参数
      return !provisionerKeys.includes(p.key)
    })
  }

  // 监听外部数据变化 - 只在真正需要时更新
  watch(
    () => props.modelValue,
    (newVal) => {
      if (isInternalUpdate) {
        return
      }

      // 深拷贝避免引用问题
      formData.value = {
        name: newVal.name || '',
        provisioner: newVal.provisioner || '',
        reclaimPolicy: newVal.reclaimPolicy || 'Delete',
        volumeBindingMode: newVal.volumeBindingMode || 'Immediate',
        allowVolumeExpansion: newVal.allowVolumeExpansion || false,
        isDefault: newVal.isDefault || false,
        parameters: [...(newVal.parameters || []).map((p) => ({ ...p }))],
        mountOptions: [...(newVal.mountOptions || [])],
        labels: [...(newVal.labels || []).map((l) => ({ ...l }))],
        annotations: [...(newVal.annotations || []).map((a) => ({ ...a }))]
      }

      // 提取供应商参数
      nextTick(() => {
        extractProvisionerParams()
      })
    },
    { immediate: true, deep: true }
  )

  // 监听本地数据变化 - 使用防抖和标志位避免循环
  watch(
    formData,
    (newVal) => {
      if (isInternalUpdate) return

      isInternalUpdate = true

      // 获取供应商生成的参数
      const provisionerGeneratedParams = getProvisionerGeneratedParams()

      // 过滤自定义参数（排除供应商参数）
      const customParams = filterCustomParameters(newVal.parameters)

      // 合并参数：供应商参数 + 自定义参数
      const existingKeys = new Set(provisionerGeneratedParams.map((p) => p.key))
      const finalParams = [
        ...provisionerGeneratedParams,
        ...customParams.filter((p) => p.key && !existingKeys.has(p.key))
      ]

      const emitData: StorageClassFormData = {
        name: newVal.name,
        provisioner: newVal.provisioner,
        reclaimPolicy: newVal.reclaimPolicy,
        volumeBindingMode: newVal.volumeBindingMode,
        allowVolumeExpansion: newVal.allowVolumeExpansion,
        isDefault: newVal.isDefault,
        parameters: finalParams,
        mountOptions: [...newVal.mountOptions],
        labels: [...newVal.labels.map((l) => ({ ...l }))],
        annotations: [...newVal.annotations.map((a) => ({ ...a }))]
      }

      emit('update:modelValue', emitData)

      nextTick(() => {
        isInternalUpdate = false
      })
    },
    { deep: true }
  )

  // 监听供应商参数变化 - 触发表单更新
  watch(
    provisionerParams,
    () => {
      if (isInternalUpdate) return

      // 触发表单数据更新以重新计算 parameters
      const current = formData.value
      formData.value = { ...current }
    },
    { deep: true }
  )

  // 处理供应商变化
  const handleProvisionerChange = (newProvisioner: string) => {
    // 重置供应商特定参数为默认值
    resetProvisionerParams()

    // 清空之前供应商的参数，只保留自定义参数
    const type = provisionerConfigMap[newProvisioner]?.type || 'generic'
    const oldType = provisionerType.value

    if (oldType && oldType !== type) {
      // 清空旧的供应商参数
      const oldKeys = provisionerParamKeys[oldType] || []
      formData.value.parameters = formData.value.parameters.filter((p) => !oldKeys.includes(p.key))
    }
  }

  // 重置供应商参数
  const resetProvisionerParams = () => {
    const type = provisionerType.value

    switch (type) {
      case 'nfs':
        provisionerParams.nfsServer = ''
        provisionerParams.nfsPath = ''
        break
      case 'openebs-local':
        provisionerParams.openebsLocalType = 'hostpath'
        provisionerParams.openebsLocalBasePath = '/var/openebs/local'
        break
      case 'openebs-cstor':
        provisionerParams.cstorPoolCluster = ''
        provisionerParams.cstorReplicaCount = 3
        break
      case 'openebs-lvm':
        provisionerParams.lvmVolgroup = ''
        provisionerParams.lvmFsType = 'ext4'
        provisionerParams.lvmShared = 'no'
        break
      case 'openebs-zfs':
        provisionerParams.zfsPoolname = ''
        provisionerParams.zfsFsType = 'zfs'
        provisionerParams.zfsCompression = 'off'
        provisionerParams.zfsDedup = false
        provisionerParams.zfsRecordsize = '128k'
        break
      case 'longhorn':
        provisionerParams.longhornNumberOfReplicas = 3
        provisionerParams.longhornStaleReplicaTimeout = 30
        provisionerParams.longhornFromBackup = ''
        provisionerParams.longhornFsType = 'ext4'
        provisionerParams.longhornDataLocality = 'disabled'
        provisionerParams.longhornReplicaAutoBalance = 'ignored'
        provisionerParams.longhornDiskSelector = ''
        provisionerParams.longhornNodeSelector = ''
        provisionerParams.longhornRecurringJobSelector = ''
        provisionerParams.longhornBackingImage = ''
        provisionerParams.longhornDataEngine = 'v1'
        provisionerParams.longhornReplicaSoftAntiAffinity = 'ignored'
        provisionerParams.longhornReplicaZoneSoftAntiAffinity = 'ignored'
        provisionerParams.longhornDisableRevisionCounter = 'ignored'
        provisionerParams.longhornMigratable = 'false'
        break
      case 'ceph-rbd':
        provisionerParams.cephClusterID = ''
        provisionerParams.cephPool = ''
        provisionerParams.cephImageFeatures = 'layering'
        provisionerParams.cephFsType = 'ext4'
        provisionerParams.cephProvisionerSecretName = ''
        provisionerParams.cephProvisionerSecretNamespace = ''
        provisionerParams.cephNodeStageSecretName = ''
        provisionerParams.cephNodeStageSecretNamespace = ''
        provisionerParams.cephControllerExpandSecretName = ''
        provisionerParams.cephControllerExpandSecretNamespace = ''
        break
      case 'aws-ebs':
        provisionerParams.ebsType = 'gp3'
        provisionerParams.ebsFsType = 'ext4'
        provisionerParams.ebsIops = 3000
        provisionerParams.ebsThroughput = 125
        provisionerParams.ebsEncrypted = false
        provisionerParams.ebsKmsKeyId = ''
        break
      case 'local-path':
        provisionerParams.localPathNodePaths = [{ node: '', path: '/opt/local-path-provisioner' }]
        break
    }
  }

  // 添加参数
  const addParameter = () => {
    formData.value.parameters.push({ key: '', value: '' })
  }

  // 移除参数
  const removeParameter = (index: number) => {
    formData.value.parameters.splice(index, 1)
  }

  // 添加挂载选项
  const addMountOption = () => {
    if (newMountOption.value.trim()) {
      formData.value.mountOptions.push(newMountOption.value.trim())
      newMountOption.value = ''
    }
  }

  // 移除挂载选项
  const removeMountOption = (index: number) => {
    formData.value.mountOptions.splice(index, 1)
  }

  // 添加标签
  const addLabel = () => {
    formData.value.labels.push({ key: '', value: '' })
  }

  // 移除标签
  const removeLabel = (index: number) => {
    formData.value.labels.splice(index, 1)
  }

  // 添加注解
  const addAnnotation = () => {
    formData.value.annotations.push({ key: '', value: '' })
  }

  // 移除注解
  const removeAnnotation = (index: number) => {
    formData.value.annotations.splice(index, 1)
  }

  // Local Path 节点路径操作
  const addNodePath = () => {
    if (!provisionerParams.localPathNodePaths) {
      provisionerParams.localPathNodePaths = []
    }
    provisionerParams.localPathNodePaths.push({ node: '', path: '' })
  }

  const removeNodePath = (index: number) => {
    provisionerParams.localPathNodePaths?.splice(index, 1)
  }

  // 验证表单
  const validate = () => {
    return formRef.value?.validate()
  }

  // 重置表单
  const resetFields = () => {
    formRef.value?.resetFields()
  }

  // 暴露方法
  defineExpose({
    validate,
    resetFields,
    formRef
  })
</script>

<style lang="scss" scoped>
  @use '@/views/cluster/resource/styles/resource-form.scss' as *;

  .storage-class-form {
    :deep(.form-collapse) {
      border: none;
      background: transparent;

      .el-collapse-item {
        background: #fff;
        border: 1px solid #dcdfe6;
        border-radius: 4px;
        margin-bottom: 12px;
        overflow: hidden;

        &.is-active {
          border-color: #409eff;
        }

        &:last-child {
          margin-bottom: 0;
        }
      }

      .el-collapse-item__header {
        height: 48px;
        line-height: 48px;
        background: #f5f7fa;
        border-bottom: 1px solid #e4e7ed;
        padding: 0 16px;
        font-size: 14px;
        font-weight: 500;
        color: #303133;
        transition: all 0.3s;

        &:hover {
          background: #ecf5ff;
        }

        .el-collapse-item__arrow {
          margin-right: 8px;
          color: #909399;
          transition: transform 0.3s;
        }

        &.is-active {
          .el-collapse-item__arrow {
            transform: rotate(90deg);
            color: #409eff;
          }
        }
      }

      .el-collapse-item__wrap {
        background: #fff;
        border-bottom: none;
      }

      .el-collapse-item__content {
        padding: 20px 24px;
      }
    }

    .collapse-section-item {
      .collapse-title {
        display: flex;
        align-items: center;
        width: 100%;

        .title-icon {
          margin-right: 8px;
          font-size: 16px;
          color: #409eff;
        }

        span {
          flex: 1;
        }
      }
    }

    .collapse-content {
      .form-section {
        @include form-section;

        .section-header {
          @include section-header;
        }
      }
    }

    .form-section {
      @include form-section;

      .section-header {
        @include section-header;
      }
    }

    .form-item-tip {
      @include form-item-tip;

      &.warning {
        color: #e6a23c;
      }
    }

    .switch-tip {
      margin-left: 12px;
      font-size: 12px;
      color: #909399;
    }

    .input-suffix {
      margin-left: 8px;
      font-size: 12px;
      color: #909399;
    }

    .input-with-suffix {
      display: flex;
      align-items: center;
    }

    .mount-options {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      align-items: center;
    }
  }
</style>

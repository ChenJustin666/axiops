export const compositePlatformParamTypes = [
  'gitRepositoryUrl',
  'nexusRepositoryUrl',
  'nexusArtifactUrl',
  'nexusArtifactVersionUrl',
  'harborProjectUrl',
  'harborImageUrl',
  'harborImageTagUrl',
  'registryRepositoryUrl',
  'registryImageUrl',
  'registryImageTagUrl',
  'jfrogRepositoryUrl',
  'jfrogArtifactUrl',
  'jfrogArtifactVersionUrl',
  'sonarProjectUrl',
  'sonarProjectKeyUrl',
  'hostConfig',
  'hostGroupConfig'
]

export interface PlatformAddressSpec {
  title: string
  channelTypeLabel?: string
  channelTypeSelectable?: boolean
  channelLabel: string
  channelParamType: string
  channelGroupCode?: string
  channelType?: string
  allowedChannelTypes?: string[]
  protocolLabel?: string
  protocolOptions?: Array<{ label: string; value: string }>
  outputFormatLabel?: string
  outputFormatOptions?: Array<{ label: string; value: string }>
  projectLabel?: string
  projectParamType?: string
  repositoryLabel?: string
  repositoryParamType?: string
  imageLabel?: string
  imageParamType?: string
  tagLabel?: string
  tagParamType?: string
  resultLabel: string
  placeholder: string
  fallbackPath: string[]
  trailingSlash?: boolean
}

export const platformAddressSpecs: Record<string, PlatformAddressSpec> = {
  gitRepositoryUrl: {
    title: '选择 Git 仓库地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '代码仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'code_repo',
    repositoryLabel: 'Git 项目',
    repositoryParamType: 'gitRepositoryUrl',
    protocolLabel: '克隆协议',
    protocolOptions: [
      { label: 'HTTP', value: 'http' },
      { label: 'SSH', value: 'ssh' }
    ],
    resultLabel: '仓库地址',
    placeholder: '请输入或选择 Git 仓库地址',
    fallbackPath: [],
    trailingSlash: false
  },
  nexusRepositoryUrl: {
    title: '选择制品仓库地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '制品仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'artifact_repo',
    channelType: 'nexus',
    repositoryLabel: '仓库',
    repositoryParamType: 'nexusRepository',
    resultLabel: '仓库地址',
    placeholder: '请输入或选择制品仓库地址',
    fallbackPath: ['repository'],
    trailingSlash: true
  },
  nexusArtifactUrl: {
    title: '选择制品地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '制品仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'artifact_repo',
    channelType: 'nexus',
    projectLabel: '仓库',
    projectParamType: 'nexusRepository',
    imageLabel: '制品',
    imageParamType: 'nexusArtifactName',
    resultLabel: '制品地址',
    placeholder: '请输入或选择制品地址',
    fallbackPath: ['repository'],
    trailingSlash: false
  },
  nexusArtifactVersionUrl: {
    title: '选择制品版本地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '制品仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'artifact_repo',
    channelType: 'nexus',
    projectLabel: '仓库',
    projectParamType: 'nexusRepository',
    imageLabel: '制品',
    imageParamType: 'nexusArtifactName',
    tagLabel: '版本',
    tagParamType: 'nexusArtifactVersion',
    resultLabel: '制品版本地址',
    placeholder: '请输入或选择制品版本地址',
    fallbackPath: ['repository'],
    trailingSlash: false
  },
  harborProjectUrl: {
    title: '选择镜像项目地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '镜像仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'image_registry',
    channelType: 'harbor',
    projectLabel: '镜像项目',
    projectParamType: 'harborProject',
    resultLabel: '镜像项目地址',
    placeholder: '请输入或选择镜像项目地址',
    fallbackPath: []
  },
  harborImageUrl: {
    title: '选择镜像地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '镜像仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'image_registry',
    channelType: 'harbor',
    projectLabel: '镜像项目',
    projectParamType: 'harborProject',
    imageLabel: '镜像',
    imageParamType: 'harborImage',
    resultLabel: '镜像地址',
    placeholder: '请输入或选择镜像地址',
    fallbackPath: []
  },
  harborImageTagUrl: {
    title: '选择镜像地址带 Tag',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '镜像仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'image_registry',
    channelType: 'harbor',
    projectLabel: '镜像项目',
    projectParamType: 'harborProject',
    imageLabel: '镜像',
    imageParamType: 'harborImage',
    tagLabel: 'Tag',
    tagParamType: 'harborImageTag',
    resultLabel: '镜像地址带 Tag',
    placeholder: '请输入或选择镜像地址带 Tag',
    fallbackPath: []
  },
  registryRepositoryUrl: {
    title: '选择镜像项目地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '镜像仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'image_registry',
    projectLabel: '镜像项目',
    projectParamType: 'registryRepository',
    resultLabel: '镜像项目地址',
    placeholder: '请输入或选择镜像项目地址',
    fallbackPath: []
  },
  registryImageUrl: {
    title: '选择镜像地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '镜像仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'image_registry',
    projectLabel: '镜像项目',
    projectParamType: 'registryRepository',
    imageLabel: '镜像',
    imageParamType: 'registryImage',
    resultLabel: '镜像地址',
    placeholder: '请输入或选择镜像地址',
    fallbackPath: []
  },
  registryImageTagUrl: {
    title: '选择镜像地址带 Tag',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '镜像仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'image_registry',
    projectLabel: '镜像项目',
    projectParamType: 'registryRepository',
    imageLabel: '镜像',
    imageParamType: 'registryImage',
    tagLabel: 'Tag',
    tagParamType: 'registryImageTag',
    resultLabel: '镜像地址带 Tag',
    placeholder: '请输入或选择镜像地址带 Tag',
    fallbackPath: []
  },
  jfrogRepositoryUrl: {
    title: '选择制品仓库地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '制品仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'artifact_repo',
    channelType: 'jfrog',
    repositoryLabel: '仓库',
    repositoryParamType: 'jfrogRepository',
    resultLabel: '仓库地址',
    placeholder: '请输入或选择制品仓库地址',
    fallbackPath: ['artifactory'],
    trailingSlash: true
  },
  jfrogArtifactUrl: {
    title: '选择制品地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '制品仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'artifact_repo',
    channelType: 'jfrog',
    projectLabel: '仓库',
    projectParamType: 'jfrogRepository',
    imageLabel: '制品',
    imageParamType: 'jfrogArtifactName',
    resultLabel: '制品地址',
    placeholder: '请输入或选择制品地址',
    fallbackPath: ['artifactory'],
    trailingSlash: false
  },
  jfrogArtifactVersionUrl: {
    title: '选择制品版本地址',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '制品仓库实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'artifact_repo',
    channelType: 'jfrog',
    projectLabel: '仓库',
    projectParamType: 'jfrogRepository',
    imageLabel: '制品',
    imageParamType: 'jfrogArtifactName',
    tagLabel: '版本',
    tagParamType: 'jfrogArtifactVersion',
    resultLabel: '制品版本地址',
    placeholder: '请输入或选择制品版本地址',
    fallbackPath: ['artifactory'],
    trailingSlash: false
  },
  sonarProjectUrl: {
    title: '选择 Sonar 项目地址',
    channelLabel: 'SonarQube 实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'code_scan',
    channelType: 'sonarqube',
    projectLabel: '项目',
    projectParamType: 'sonarProjectKey',
    resultLabel: '项目地址',
    placeholder: '请输入或选择 Sonar 项目地址',
    fallbackPath: []
  },
  sonarProjectKeyUrl: {
    title: '选择 Sonar 项目 Key 地址',
    channelLabel: 'SonarQube 实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'code_scan',
    channelType: 'sonarqube',
    projectLabel: '项目 Key',
    projectParamType: 'sonarProjectKey',
    resultLabel: '项目 Key 地址',
    placeholder: '请输入或选择 Sonar 项目 Key 地址',
    fallbackPath: []
  },
  hostConfig: {
    title: '选择主机配置',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '主机渠道实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'deploy_target',
    channelType: 'host',
    allowedChannelTypes: ['host'],
    repositoryLabel: '主机',
    repositoryParamType: 'hostConfig',
    outputFormatLabel: '输出格式',
    outputFormatOptions: [
      { label: 'JSON', value: 'json' },
      { label: 'JSON Base64', value: 'jsonBase64' },
      { label: 'Ansible', value: 'ansible' },
      { label: 'Ansible Base64', value: 'ansibleBase64' }
    ],
    resultLabel: '主机配置',
    placeholder: '请输入或选择主机配置',
    fallbackPath: []
  },
  hostGroupConfig: {
    title: '选择主机组配置',
    channelTypeLabel: '渠道类型',
    channelTypeSelectable: true,
    channelLabel: '主机组渠道实例',
    channelParamType: 'channelGroupCode',
    channelGroupCode: 'deploy_target',
    channelType: 'host_group',
    allowedChannelTypes: ['host_group'],
    repositoryLabel: '主机组',
    repositoryParamType: 'hostGroupConfig',
    outputFormatLabel: '输出格式',
    outputFormatOptions: [
      { label: 'JSON', value: 'json' },
      { label: 'JSON Base64', value: 'jsonBase64' },
      { label: 'Ansible', value: 'ansible' },
      { label: 'Ansible Base64', value: 'ansibleBase64' }
    ],
    resultLabel: '主机组配置',
    placeholder: '请输入或选择主机组配置',
    fallbackPath: []
  }
}

export const isCompositePlatformParamType = (paramType?: string) =>
  compositePlatformParamTypes.includes(String(paramType || ''))

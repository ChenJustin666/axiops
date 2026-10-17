# CICD 项目初始化向导实施方案

## 一、需求概述

### 当前问题
创建项目后直接进入编辑区域（project-workspace），没有初始化引导流程。

### 目标状态
1. **创建项目时**: 进入4步初始化向导
   - Step 1: 项目配置（确认基本信息）
   - Step 2: 新建服务（配置服务、镜像、变量）
   - Step 3: 创建环境（关联集群、命名空间）
   - Step 4: 执行工作流（展示自动生成的流水线）

2. **初始化完成后**: 进入日常编辑模式
   - 顶部导航Tab：项目配置 / 服务 / 环境 / 工作流
   - 可自由编辑，无需按顺序
   - 复用向导中的表单组件

## 二、数据库改造

### 1. 新增字段（已创建迁移文件）
```sql
-- 文件: migrations/000XXX_add_project_initialization_status.up.sql
ALTER TABLE project ADD COLUMN initialization_status VARCHAR(20) NOT NULL DEFAULT 'pending';
ALTER TABLE project ADD COLUMN initialization_step INT NOT NULL DEFAULT 0;
ALTER TABLE project ADD COLUMN initialized_at DATETIME NULL;
```

### 2. 字段说明
- `initialization_status`: pending | in_progress | completed
- `initialization_step`: 0-4 (当前步骤)
- `initialized_at`: 完成时间

## 三、前端改造

### 1. 类型定义更新（已完成）
文件: `src/api/axiops/cicd.ts`
- 添加 `InitializationStatus` 类型
- 更新 `AxiOpsProject` 接口

### 2. 路由配置
需要添加向导路由：
```typescript
// src/router/modules/axiops.ts 或 devops.ts
{
  path: '/cicd/projects/:projectId/wizard',
  name: 'ProjectWizard',
  component: '/cicd/project-wizard/index',
  meta: { title: '项目初始化向导', isHideTab: true }
}
```

### 3. 路由守卫逻辑
在 project-workspace/index.vue 中添加检查：
```typescript
onMounted(async () => {
  const project = await getProjectApi(projectId)
  
  // 未初始化则跳转到向导
  if (project.initialization_status !== 'completed') {
    await router.replace(`/cicd/projects/${projectId}/wizard`)
    return
  }
  
  // 已初始化，正常加载工作区
  await loadWorkspace()
})
```

### 4. 创建项目后跳转
在 projects/index.vue 的 submitCreate 方法中：
```typescript
const submitCreate = async () => {
  const result = await createProjectApi(form)
  ElMessage.success('项目已创建')
  createVisible.value = false
  
  // 跳转到初始化向导
  await router.push(`/cicd/projects/${result.id}/wizard`)
}
```

### 5. 核心组件结构

```
src/views/cicd/
├── project-wizard/           # 新增：初始化向导
│   └── index.vue            # 4步向导主页面
├── project-workspace/        # 现有：日常工作区
│   ├── index.vue            # 添加初始化状态检查
│   └── components/
│       ├── ServiceCreationWizard.vue     # 复用
│       ├── EnvironmentConfigDialog.vue   # 复用
│       └── ...
└── projects/                 # 项目列表
    └── index.vue            # 修改创建后的跳转逻辑
```

## 四、核心实现要点

### 1. 向导页面特点
- 全屏展示，顶部进度条显示4步
- 每步完成后才能进入下一步
- 支持上一步回退（保留已填数据）
- 底部操作栏：上一步 / 下一步 / 完成向导

### 2. 步骤验证逻辑
```typescript
const canProceed = computed(() => {
  if (currentStep.value === 1) return true  // Step1直接通过
  if (currentStep.value === 2) return services.value.length > 0  // 至少1个服务
  if (currentStep.value === 3) return environments.value.length > 0  // 至少1个环境
  return false
})
```

### 3. 步骤切换保存
```typescript
const nextStep = async () => {
  // 保存当前步骤到后端
  await updateProjectApi(projectId, {
    initialization_step: currentStep.value + 1
  })
  currentStep.value++
}
```

### 4. 完成初始化
```typescript
const completeWizard = async () => {
  await updateProjectApi(projectId, {
    initialization_status: 'completed',
    initialization_step: 4
  })
  ElMessage.success('项目初始化完成！')
  await router.push(`/cicd/projects/${projectId}`)  // 跳转到工作区
}
```

## 五、后端 API 需求

### 1. 更新项目接口
```go
// PATCH /api/v1/projects/:id
type UpdateProjectRequest struct {
    InitializationStatus string `json:"initialization_status,omitempty"`
    InitializationStep   int    `json:"initialization_step,omitempty"`
}
```

### 2. 获取项目接口（已有）
返回中需包含新增的初始化字段

## 六、实施步骤

### 阶段一：数据库和类型（已完成 ✓）
1. ✓ 创建数据库迁移文件
2. ✓ 更新前端类型定义

### 阶段二：后端API（需要实施）
1. 执行数据库迁移
2. 更新 Go 项目模型结构体
3. 修改创建项目接口（设置初始状态为 pending）
4. 修改更新项目接口（支持更新初始化字段）
5. 修改查询项目接口（返回初始化字段）

### 阶段三：前端实现（需要实施）
1. 创建向导页面组件 `/views/cicd/project-wizard/index.vue`
2. 添加路由配置
3. 修改 projects/index.vue 创建后的跳转逻辑
4. 修改 project-workspace/index.vue 添加初始化检查
5. 测试完整流程

## 七、关键技术点

### 1. 组件复用
向导中的服务创建、环境创建对话框直接复用现有组件：
- ServiceCreationWizard
- EnvironmentConfigDialog

### 2. 状态管理
向导步骤状态存储在后端，前端仅做展示和交互：
- 刷新页面时从后端恢复当前步骤
- 每步切换时保存到后端

### 3. 路由控制
- 未初始化项目：强制跳转到向导
- 已初始化项目：正常进入工作区
- 支持中途退出，下次继续

### 4. 用户体验
- 进度条清晰显示当前进度
- 已完成步骤显示绿色对勾
- 支持跳过工作流运行，直接完成
- 完成后自动跳转到工作区

## 八、测试要点

1. **创建新项目** → 自动跳转到向导 Step 1
2. **Step 1** → 点击下一步进入 Step 2
3. **Step 2** → 至少创建1个服务才能下一步
4. **Step 3** → 至少创建1个环境才能下一步
5. **Step 4** → 可选运行工作流，点击完成
6. **完成初始化** → 跳转到工作区
7. **再次访问** → 直接进入工作区（不再进向导）
8. **中途退出** → 下次继续从当前步骤开始

## 九、可选优化

1. **自动保存**：表单数据自动保存到 localStorage
2. **跳过向导**：管理员可直接跳过向导
3. **引导提示**：每步添加操作提示或视频教程
4. **模板快速初始化**：提供快速初始化模板（预设服务+环境）
5. **进度恢复提示**：中途退出后再进入显示恢复提示

## 十、注意事项

1. **老项目兼容**：迁移时将所有现有项目标记为 completed
2. **权限控制**：向导中的操作需检查用户权限
3. **异常处理**：API 失败时保留当前步骤数据
4. **浏览器刷新**：刷新后恢复到当前步骤
5. **并发编辑**：同一项目多人初始化时的冲突处理

---

**实施优先级**: 高
**预计工期**: 3-5天（后端2天 + 前端3天）
**影响范围**: 项目创建流程、项目访问流程

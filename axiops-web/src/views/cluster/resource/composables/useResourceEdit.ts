/**
 * 集群资源编辑通用 Hook
 * 统一处理创建/编辑模式切换、YAML 合并策略、表单验证
 */

import { ref, computed, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { parseAndWrap, mergeFormToYaml } from '@/utils/kubernetes/yaml-merge'
import type { RawResourceWrapper } from '@/utils/kubernetes/yaml-merge'
import * as yaml from 'js-yaml'

/**
 * 资源编辑 Hook 配置
 */
export interface UseResourceEditOptions<T> {
  /** 对话框类型 */
  dialogType: 'create' | 'edit'
  /** 表单数据转 YAML 函数 */
  formToYaml: (formData: T, isEdit?: boolean) => string
  /** YAML 转表单数据函数 */
  yamlToForm: (yamlStr: string) => T
  /** 创建空表单数据函数 */
  createEmptyForm: () => T
  /** 用户可编辑的字段路径列表 */
  editableFields: string[]
}

/**
 * 资源编辑 Hook 返回值
 */
export interface UseResourceEditReturn<T> {
  /** 表单数据 */
  formData: ReturnType<typeof ref<T>>
  /** YAML 内容 */
  yamlContent: ReturnType<typeof ref<string>>
  /** 原始 YAML（编辑模式下暂存） */
  rawSourceYaml: ReturnType<typeof ref<string>>
  /** 是否正在初始化 */
  isInitializing: ReturnType<typeof ref<boolean>>
  /** 转换错误信息 */
  convertError: ReturnType<typeof ref<string>>
  /** 编辑模式 */
  editMode: ReturnType<typeof ref<'form' | 'yaml'>>
  /** 是否为编辑模式 */
  isEditMode: ReturnType<typeof computed<boolean>>
  /** 初始化创建模式 */
  initCreateMode: () => Promise<void>
  /** 加载编辑模式数据 */
  loadEditMode: (yamlStr: string) => Promise<void>
  /** 获取最终提交的 YAML */
  getFinalYaml: () => string
  /** 表单转 YAML（手动触发） */
  syncFormToYaml: () => void
  /** YAML 转表单（手动触发） */
  syncYamlToForm: () => void
}

/**
 * 集群资源编辑通用 Hook
 * @param options 配置选项
 * @returns Hook 返回值
 */
export function useResourceEdit<T>(options: UseResourceEditOptions<T>): UseResourceEditReturn<T> {
  const { dialogType, formToYaml, yamlToForm, createEmptyForm, editableFields } = options

  // 状态
  const formData = ref<T>(createEmptyForm()) as ReturnType<typeof ref<T>>
  const yamlContent = ref('')
  const rawSourceYaml = ref('') // 编辑模式下暂存原始 YAML
  const isInitializing = ref(false)
  const convertError = ref('')
  const editMode = ref<'form' | 'yaml'>('form')

  // 计算属性
  const isEditMode = computed(() => dialogType === 'edit')

  /**
   * 表单转 YAML（自动同步）
   */
  const syncFormToYaml = () => {
    if (isInitializing.value) return

    try {
      const isEdit = dialogType === 'edit'
      yamlContent.value = formToYaml(formData.value, isEdit)
      convertError.value = ''
    } catch (error: any) {
      console.error('表单转 YAML 失败:', error)
      convertError.value = `表单转 YAML 失败: ${error.message}`
    }
  }

  /**
   * YAML 转表单（手动触发）
   */
  const syncYamlToForm = () => {
    try {
      const parsed = yamlToForm(yamlContent.value)
      formData.value = parsed
      convertError.value = ''
    } catch (error: any) {
      console.error('YAML 转表单失败:', error)
      convertError.value = `YAML 转表单失败: ${error.message}`
      throw error
    }
  }

  /**
   * 初始化创建模式
   */
  const initCreateMode = async () => {
    isInitializing.value = true
    try {
      formData.value = createEmptyForm()
      rawSourceYaml.value = ''
      await nextTick()
      syncFormToYaml()
    } finally {
      isInitializing.value = false
    }
  }

  /**
   * 加载编辑模式数据
   * @param yamlStr 原始 YAML 字符串
   */
  const loadEditMode = async (yamlStr: string) => {
    isInitializing.value = true
    try {
      // 暂存原始 YAML
      rawSourceYaml.value = yamlStr
      yamlContent.value = yamlStr

      // 解析到表单
      const parsed = yamlToForm(yamlStr)
      formData.value = parsed

      convertError.value = ''
    } catch (error: any) {
      console.error('加载编辑数据失败:', error)
      convertError.value = `加载失败: ${error.message}`
      throw error
    } finally {
      isInitializing.value = false
    }
  }

  /**
   * 获取最终提交的 YAML
   * 编辑模式使用 merge 策略，创建模式直接生成
   */
  const getFinalYaml = (): string => {
    if (dialogType === 'edit' && rawSourceYaml.value) {
      // 编辑模式：使用 merge 策略
      try {
        const wrapper = parseAndWrap(rawSourceYaml.value, editableFields)

        // 将表单数据转为对象
        const formYaml = formToYaml(formData.value, true)
        const formObj = yaml.load(formYaml) as any

        // 构建变更映射
        const changes: Record<string, any> = {}
        for (const field of editableFields) {
          const value = getNestedValue(formObj, field)
          if (value !== undefined) {
            changes[field] = value
          }
        }

        // 合并并返回
        return mergeFormToYaml(wrapper, changes)
      } catch (error: any) {
        console.error('Merge 策略失败:', error)
        throw new Error(`合并 YAML 失败: ${error.message}`)
      }
    } else {
      // 创建模式：直接生成 YAML
      return formToYaml(formData.value, false)
    }
  }

  /**
   * 监听表单数据变化，自动同步到 YAML
   */
  watch(
    formData,
    () => {
      if (editMode.value === 'form') {
        syncFormToYaml()
      }
    },
    { deep: true }
  )

  return {
    formData,
    yamlContent,
    rawSourceYaml,
    isInitializing,
    convertError,
    editMode,
    isEditMode,
    initCreateMode,
    loadEditMode,
    getFinalYaml,
    syncFormToYaml,
    syncYamlToForm
  }
}

/**
 * 获取嵌套对象的值
 * @param obj 对象
 * @param path 路径（如 'metadata.name'）
 * @returns 值
 */
function getNestedValue(obj: any, path: string): any {
  const keys = path.split('.')
  let current = obj

  for (const key of keys) {
    if (current === null || current === undefined) {
      return undefined
    }
    current = current[key]
  }

  return current
}

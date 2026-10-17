import {
  getControllerHealthApi,
  getIngressTrafficByHostApi,
  getIngressPerformanceApi,
  getIngressErrorsApi,
  getIngressBackendsApi,
  getIngressCertificatesApi,
  getIngressLatencyByHostApi,
  type IngressControllerHealth,
  type IngressTrafficMetrics,
  type IngressPerformanceMetrics,
  type IngressErrorMetrics,
  type IngressBackendMetrics,
  type IngressCertificateMetrics,
  type GetIngressByHostParams,
  type GetControllerHealthParams,
  type GetIngressCertificatesParams
} from '@/api'

// 临时类型定义，用于错误和后端指标（当前没有 host-specific API）
interface IngressErrorMetrics {
  overall: {
    totalErrorRate: number
    error4xxRate: number
    error5xxRate: number
  }
}

export function useIngressMonitor() {
  // ==================== 状态 ====================
  const controllerData = ref<IngressControllerHealth>()
  const controllerLoading = ref(false)

  const trafficData = ref<IngressTrafficMetrics>()
  const trafficLoading = ref(false)

  const performanceData = ref<IngressPerformanceMetrics>()
  const performanceLoading = ref(false)

  const errorData = ref<IngressErrorMetrics>()
  const errorLoading = ref(false)

  const backendData = ref<IngressBackendMetrics>()
  const backendLoading = ref(false)

  const certificateData = ref<IngressCertificateMetrics>()
  const certificateLoading = ref(false)

  // ==================== 加载方法 ====================

  /**
   * 加载 Controller 健康状态
   */
  const loadControllerHealth = async (params: GetControllerHealthParams) => {
    controllerLoading.value = true
    try {
      const response = await getControllerHealthApi(params)
      controllerData.value = response.data
    } catch (error) {
      console.error('[Ingress监控] 加载 Controller 健康状态失败:', error)
      controllerData.value = undefined
    } finally {
      controllerLoading.value = false
    }
  }

  /**
   * 加载流量指标（按 Host）
   */
  const loadTrafficMetrics = async (params: GetIngressByHostParams) => {
    trafficLoading.value = true
    try {
      const response = await getIngressTrafficByHostApi(params)
      trafficData.value = response.data
    } catch (error) {
      console.error('[Ingress监控] 加载流量指标失败:', error)
      trafficData.value = undefined
    } finally {
      trafficLoading.value = false
    }
  }

  /**
   * 加载性能指标（按 Host）
   */
  const loadPerformanceMetrics = async (params: GetIngressByHostParams) => {
    performanceLoading.value = true
    try {
      // 注意：这里可能需要使用 getIngressLatencyByHostApi
      // 暂时使用通用接口，后续可以调整
      const response = await getIngressPerformanceApi(params as any)
      performanceData.value = response.data
    } catch (error) {
      console.error('[Ingress监控] 加载性能指标失败:', error)
      performanceData.value = undefined
    } finally {
      performanceLoading.value = false
    }
  }

  /**
   * 加载错误指标（按 Host）
   */
  const loadErrorMetrics = async (params: GetIngressByHostParams) => {
    errorLoading.value = true
    try {
      const response = await getIngressErrorsApi(params as any)
      errorData.value = response.data
    } catch (error) {
      console.error('[Ingress监控] 加载错误指标失败:', error)
      errorData.value = undefined
    } finally {
      errorLoading.value = false
    }
  }

  /**
   * 加载后端健康状态（按 Host）
   */
  const loadBackendMetrics = async (params: GetIngressByHostParams) => {
    backendLoading.value = true
    try {
      const response = await getIngressBackendsApi(params as any)
      backendData.value = response.data
    } catch (error) {
      console.error('[Ingress监控] 加载后端健康状态失败:', error)
      backendData.value = undefined
    } finally {
      backendLoading.value = false
    }
  }

  /**
   * 加载证书信息
   */
  const loadCertificateMetrics = async (params: GetIngressCertificatesParams) => {
    certificateLoading.value = true
    try {
      const response = await getIngressCertificatesApi(params)
      certificateData.value = response.data
    } catch (error) {
      console.error('[Ingress监控] 加载证书信息失败:', error)
      certificateData.value = undefined
    } finally {
      certificateLoading.value = false
    }
  }

  return {
    // 状态
    controllerData,
    controllerLoading,
    trafficData,
    trafficLoading,
    performanceData,
    performanceLoading,
    errorData,
    errorLoading,
    backendData,
    backendLoading,
    certificateData,
    certificateLoading,

    // 方法
    loadControllerHealth,
    loadTrafficMetrics,
    loadPerformanceMetrics,
    loadErrorMetrics,
    loadBackendMetrics,
    loadCertificateMetrics
  }
}

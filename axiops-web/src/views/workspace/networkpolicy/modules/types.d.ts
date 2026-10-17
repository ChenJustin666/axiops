declare module '@/api/workload/selector' {
  export function getWorkloadResourceListApi(params: any): Promise<any>
  export function getWorkloadPodSelectorApi(params: any): Promise<any>
  export function getNamespaceLabelsApi(params: any): Promise<any>
}

declare module '@/api/manager/cluster' {
  export function getClusterNamespaceListApi(clusterUuid: string): Promise<string[]>
}

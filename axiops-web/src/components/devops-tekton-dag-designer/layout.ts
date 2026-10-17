import type { TektonDagConfig, TektonDagNode } from './types'

export function layoutTektonDag(config: TektonDagConfig): TektonDagConfig {
  const normalNodes = config.nodes.filter((node) => node.type !== 'finally' && node.type !== 'note')
  const finallyNodes = config.nodes.filter((node) => node.type === 'finally')
  const depth = nodeDepthMap(config)
  const rows = new Map<number, TektonDagNode[]>()
  normalNodes.forEach((node) => {
    const level = depth.get(node.id) || 0
    if (!rows.has(level)) rows.set(level, [])
    rows.get(level)?.push(node)
  })

  const positioned = config.nodes.map((node) => {
    if (node.type === 'finally') {
      const index = finallyNodes.findIndex((item) => item.id === node.id)
      return { ...node, position: { x: 280 + index * 300, y: 560 } }
    }
    if (node.type === 'note') return node
    const level = depth.get(node.id) || 0
    const siblings = rows.get(level) || []
    const index = siblings.findIndex((item) => item.id === node.id)
    return {
      ...node,
      position: {
        x: 100 + level * 320,
        y: 150 + Math.max(index, 0) * 150
      }
    }
  })
  return { ...config, nodes: positioned }
}

function nodeDepthMap(config: TektonDagConfig) {
  const incoming = new Map<string, string[]>()
  config.nodes.forEach((node) => incoming.set(node.id, []))
  config.edges
    .filter((edge) => ['runAfter', 'result', 'when'].includes(edge.type))
    .forEach((edge) => {
      incoming.get(edge.target)?.push(edge.source)
    })

  const memo = new Map<string, number>()
  const calc = (nodeId: string, stack = new Set<string>()): number => {
    if (memo.has(nodeId)) return memo.get(nodeId) || 0
    if (stack.has(nodeId)) return 0
    stack.add(nodeId)
    const parents = incoming.get(nodeId) || []
    const value = parents.length ? Math.max(...parents.map((parent) => calc(parent, stack))) + 1 : 0
    stack.delete(nodeId)
    memo.set(nodeId, value)
    return value
  }

  config.nodes.forEach((node) => calc(node.id))
  return memo
}

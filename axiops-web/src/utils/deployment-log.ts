type LogEntry = { time: string; text: string; order: number; outcome: boolean }

// A read-only projection of the saved diagnostics. Unknown lines are retained;
// the original log remains available in the viewer's detailed mode.
export function summarizeDeploymentLog(text: string, status = '', error = ''): string {
  if (!text.trim() && !error.trim()) return ''
  const entries = new Map<string, LogEntry>()
  let time = '',
    order = 0
  const terminal = ['succeeded', 'success', 'failed', 'timeout', 'cancelled', 'skipped'].includes(
    status
  )
  const add = (key: string, message: string, outcome = false, once = false) => {
    if (once && entries.has(key)) return
    entries.set(key, { time, text: message, order: order++, outcome })
  }
  const value = (line: string, key: string) =>
    new RegExp(`(?:^|\\s)${key}=([^\\s;]+)`).exec(line)?.[1] || ''
  const routineEvents = new Set([
    'Scheduled',
    'AllocIPSucceed',
    'Pulling',
    'Pulled',
    'Created',
    'Started',
    'ScalingReplicaSet',
    'SuccessfulCreate',
    'SuccessfulDelete',
    'Killing'
  ])
  // error_message is a shortened copy of the terminal diagnostics. Appending
  // it when its first line is already saved duplicates failures and may turn
  // a partially truncated event timestamp into an apparent event reason.
  const firstError = error.split(/\r?\n/)[0].trim()
  const source = firstError && !text.includes(firstError) ? `${text}\n${error}` : text
  for (const raw of source.split(/\r?\n/)) {
    const line = raw.replace(/\s*<nil>/g, '').trim()
    if (!line) continue
    if (/^\[\d{4}-\d{2}-\d{2}T[^\]]+\]$/.test(line)) {
      time = line.slice(1, -1).replace('T', ' ')
      continue
    }
    if (/^\[prepare\] YAML rendered and validated/.test(line)) {
      add('prepare', '[准备] YAML 已校验，部署快照已保存', false, true)
      continue
    }
    const target = /^\[(?:apply|check)\] cluster=(\d+) namespace=([^;\s]+)/.exec(line)
    if (target) {
      add(
        `target:${target[1]}:${target[2]}`,
        `[目标] 集群 #${target[1]} · 命名空间 ${target[2]}`,
        false,
        true
      )
      continue
    }
    const applied = /^\[apply\] (\S+\/\S+) submitted$/.exec(line)
    if (applied) {
      add(`apply:${applied[1]}`, `[提交] ${applied[1]}`, false, true)
      continue
    }
    if (line === '[apply] resources submitted; waiting for Kubernetes rollout checks') continue
    if (line === '[apply] resources submitted; readiness checks are not configured') {
      add('unchecked', '[注意] 资源已提交，但未配置就绪检查', false, true)
      continue
    }
    if (/^\[resource\] \S+ submitted; no rollout check for this kind$/.test(line)) continue
    const rollout = /^\[rollout\] (\S+\/\S+) /.exec(line)
    if (rollout) {
      const resource = rollout[1],
        desired = value(line, 'desired')
      let message = ''
      if (resource.startsWith('Job/')) {
        message = `成功 ${value(line, 'succeeded') || '0'} · 失败 ${value(line, 'failed') || '0'}`
      } else if (desired) {
        const ready = resource.startsWith('DaemonSet/')
          ? value(line, 'availableScheduled')
          : resource.startsWith('StatefulSet/')
            ? value(line, 'readyReplicas')
            : value(line, 'available') || value(line, 'readyReplicas')
        message = `就绪 ${ready || '0'}/${desired}`
        const updated = value(
          line,
          resource.startsWith('DaemonSet/') ? 'updatedScheduled' : 'updated'
        )
        if (updated) message += ` · 已更新 ${updated}/${desired}`
        if (desired === '0') message += ' · 已缩容为 0，不代表对外可用'
      } else {
        message = value(line, 'ready') === 'true' ? '就绪检查通过' : '等待就绪'
      }
      if (Number(value(line, 'observed')) < Number(value(line, 'generation')))
        message += ' · 等待控制器确认新版本'
      add(`rollout:${resource}`, `[进度] ${resource} · ${message}`)
      continue
    }
    if (line.startsWith('[rollout] desired replicas=0:')) continue // covered by its resource summary
    const pod = /^\[pod\] (\S+) phase=(\S+)\s*(.*)$/.exec(line)
    if (pod) {
      const [, name, phase, containers] = pod
      if (!containers.includes('container=')) {
        add(`pod:${name}`, `[Pod] ${name} · ${phase}`)
        continue
      }
      for (const container of containers.split(';')) {
        const nameOfContainer = value(container, 'container'),
          ready = value(container, 'ready') === 'true'
        const waiting = value(container, 'waiting'),
          terminated = value(container, 'terminated'),
          previous = value(container, 'previous')
        const failure =
          [waiting, terminated, !ready ? previous : ''].find(
            (reason) =>
              reason && !['ContainerCreating', 'PodInitializing', 'Completed'].includes(reason)
          ) || ''
        const key = `pod:${name}:${nameOfContainer}`
        const restarts = value(container, 'restarts')
        let message = `Pod/${name} · 容器 ${nameOfContainer}`
        if (failure) {
          message += ` · ${failure}`
          if (waiting && previous) message += ` · 上次退出 ${previous}`
          const codes = [...container.matchAll(/\bexitCode=(\S+)/g)].map((match) => match[1])
          if (codes.length) message += ` · 退出码 ${[...new Set(codes)].join(' / ')}`
          if (restarts) message += ` · 重启 ${restarts} 次`
          add(`${key}:error:${failure}`, `[异常记录] ${message}`)
        } else {
          message += ready ? ' · 已就绪' : ` · ${waiting || phase}，等待就绪`
          if (restarts && restarts !== '0') message += ` · 重启 ${restarts} 次`
          add(`${key}:state`, `[Pod] ${message}`)
        }
      }
      continue
    }
    // Routine false readiness conditions duplicate the replica/Pod summary.
    // Scheduling, deadline, image and unknown conditions remain visible.
    if (/^\[(?:pod|condition)\]/.test(line)) {
      if (
        /\b(?:Ready|ContainersReady|PodReadyToStartContainers)=False\b/.test(line) &&
        (!value(line, 'reason') || value(line, 'reason') === 'ContainersNotReady')
      )
        continue
      if (line.includes('Available=False reason=MinimumReplicasUnavailable')) continue
      if (
        line.includes('Progressing=True') &&
        [
          'ReplicaSetUpdated',
          'NewReplicaSetCreated',
          'NewReplicaSetAvailable',
          'FoundNewReplicaSet'
        ].includes(value(line, 'reason'))
      )
        continue
      const deadline =
        /^\[condition\] (\S+) Progressing=False reason=ProgressDeadlineExceeded\b/.exec(line)
      if (deadline) {
        add(
          `condition:${deadline[1]}:deadline`,
          `[条件] ${deadline[1]} · ProgressDeadlineExceeded（发布进度超时）`
        )
        continue
      }
      add(`condition:${line}`, line)
      continue
    }
    if (line.startsWith('[event]') && /\[truncated\]$/.test(line)) {
      add('truncated-event', '[提示] 部分事件信息已截断，可切换详细诊断查看已保存内容', false, true)
      continue
    }
    const event = /^\[event\] (\S+)\s+(?:(\d{4}-\S+)\s+)?([^:\s]+):\s*(.*)$/.exec(line)
    if (event) {
      const [, resource, , reason, message] = event
      if (routineEvents.has(reason)) continue
      const shown = reason === 'BackOff' ? '容器反复退出，Kubernetes 正在重试' : message
      // Do not count polls as occurrences. Keep the latest observation of each
      // distinct warning, including the original probe/scheduling error text.
      add(`event:${resource}:${reason}:${shown}`, `[事件] ${resource} · ${reason}: ${shown}`)
      continue
    }
    if (line === '[events] additional events omitted') {
      add('event-limit', '[提示] 事件采集有数量限制，必要时到环境资源页查看全部事件', false, true)
      continue
    }
    if (line.startsWith('[waiting]')) {
      if (!terminal) add('waiting', '[等待] 资源尚未就绪，继续检查；状态每 3 秒刷新')
      continue
    }
    if (line.startsWith('[succeeded]')) {
      const message = line.includes('health not verified')
        ? '资源已提交；未包含支持的就绪检查目标，应用健康未验证'
        : line.includes('Kubernetes rollout checks passed')
          ? 'Kubernetes 就绪检查通过；未执行外部访问或业务检查'
          : line.slice('[succeeded]'.length).trim()
      add('result', `[成功] ${message}`, true)
      continue
    }
    if (line.startsWith('[failed]')) {
      add('result', `[失败] ${line.slice('[failed]'.length).trim()}`, true)
      continue
    }
    if (line.startsWith('[system] earlier checks truncated')) {
      add('truncated', '[提示] 部分早期诊断已达到保存上限；当前记录不包含全部历史', false, true)
      continue
    }
    add(`other:${line}`, line)
  }
  if (!entries.has('result') && terminal) {
    const labels: Record<string, string> = {
      succeeded: '成功',
      success: '成功',
      failed: '失败',
      timeout: '超时',
      cancelled: '已取消',
      skipped: '已跳过'
    }
    add('result', `[结果] 任务${labels[status]}（以已保存的执行状态为准）`, true)
  }
  return [...entries.values()]
    .sort((a, b) => Number(a.outcome) - Number(b.outcome) || a.order - b.order)
    .map((entry) => `${entry.time ? `[${entry.time}] ` : ''}${entry.text}`)
    .join('\n')
}

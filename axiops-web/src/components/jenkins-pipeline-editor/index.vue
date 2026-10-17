<template>
  <div class="jenkins-pipeline-editor" :class="{ 'is-invalid': syntaxError }" :style="{ height }">
    <div class="editor-head">
      <div class="editor-title">
        <strong>{{ title }}</strong>
        <span>{{ lineCount }} 行</span>
      </div>
      <div class="editor-actions">
        <ElButton v-if="contentType === 'steps'" size="small" @click="insertStepsSnippet">
          插入 Steps
        </ElButton>
        <ElButton v-if="contentType === 'post'" size="small" @click="insertPostSnippet">
          插入 Post
        </ElButton>
        <ElButton size="small" @click="insertScriptSnippet">插入 Script</ElButton>
        <ElButton size="small" @click="formatCode">格式整理</ElButton>
        <ElButton size="small" type="primary" plain @click="$emit('extract-params')">
          提取参数
        </ElButton>
      </div>
    </div>

    <div class="editor-body">
      <pre ref="lineNumberRef" class="line-numbers">{{ lineNumbers }}</pre>
      <textarea
        ref="textareaRef"
        class="code-textarea"
        :value="modelValue"
        :readonly="readonly"
        spellcheck="false"
        placeholder="请输入 Jenkins steps 内容"
        @input="handleInput"
        @blur="validateCode(modelValue, false)"
        @scroll="handleScroll"
        @keydown.tab.prevent="insertTab"
      />
    </div>

    <div v-if="syntaxError" class="editor-error">{{ syntaxError }}</div>

    <div class="editor-foot">
      <span>Jenkins Pipeline</span>
      <span>支持 parameters、environment、params/env 读取和 ${变量} 提取参数</span>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'

  const props = withDefaults(
    defineProps<{
      modelValue: string
      title?: string
      height?: string
      readonly?: boolean
      contentType?: 'steps' | 'post'
    }>(),
    {
      title: 'Jenkins Steps 代码片段',
      height: '420px',
      readonly: false,
      contentType: 'steps'
    }
  )

  const emit = defineEmits<{
    'update:modelValue': [value: string]
    'extract-params': []
  }>()

  const textareaRef = ref<HTMLTextAreaElement>()
  const lineNumberRef = ref<HTMLPreElement>()
  const syntaxError = ref('')

  const lineCount = computed(() => Math.max(props.modelValue.split('\n').length, 1))
  const lineNumbers = computed(() =>
    Array.from({ length: lineCount.value }, (_, index) => index + 1).join('\n')
  )

  const updateValue = (value: string) => {
    emit('update:modelValue', value)
  }

  const handleInput = (event: Event) => {
    const value = (event.target as HTMLTextAreaElement).value
    updateValue(value)
    if (syntaxError.value) validateCode(value, false)
  }

  const insertAtCursor = (snippet: string) => {
    if (props.readonly) return
    const textarea = textareaRef.value
    if (!textarea) {
      updateValue(`${props.modelValue}\n${snippet}`)
      return
    }
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const nextValue = `${props.modelValue.slice(0, start)}${snippet}${props.modelValue.slice(end)}`
    updateValue(nextValue)
    requestAnimationFrame(() => {
      const nextPosition = start + snippet.length
      textarea.focus()
      textarea.setSelectionRange(nextPosition, nextPosition)
    })
  }

  const insertStepsSnippet = () => {
    insertAtCursor(`steps {
  script {
    echo '执行步骤'
  }
}
`)
  }

  const insertPostSnippet = () => {
    insertAtCursor(`always {
  echo 'post 步骤'
}
`)
  }

  const insertScriptSnippet = () => {
    insertAtCursor(`script {
  echo '执行脚本'
}
`)
  }

  const insertTab = () => {
    insertAtCursor('  ')
  }

  const handleScroll = () => {
    if (!textareaRef.value || !lineNumberRef.value) return
    lineNumberRef.value.scrollTop = textareaRef.value.scrollTop
  }

  type SyntaxCheckResult = {
    valid: boolean
    message?: string
    line?: number
    column?: number
  }

  const postConditions = ['always', 'success', 'failure', 'changed', 'aborted', 'unstable']

  const findMatchingPair = (code: string, start: number, open: string, close: string) => {
    let depth = 0
    let quote = ''
    let escaped = false

    for (let index = start; index < code.length; index += 1) {
      const char = code[index]
      if (quote) {
        if (escaped) {
          escaped = false
          continue
        }
        if (char === '\\') {
          escaped = true
          continue
        }
        if (char === quote) quote = ''
        continue
      }

      if (char === '"' || char === "'") {
        quote = char
        continue
      }
      if (char === open) depth += 1
      if (char === close) {
        depth -= 1
        if (depth === 0) return index
      }
    }

    return -1
  }

  const formatSyntaxMessage = (result: SyntaxCheckResult) => {
    if (!result.message) return ''
    if (!result.line || !result.column) return result.message
    return `第 ${result.line} 行第 ${result.column} 列：${result.message}`
  }

  const checkPipelineSyntax = (value: string): SyntaxCheckResult => {
    const source = (value || '').replace(/\r\n?/g, '\n')
    const openMap: Record<string, string> = { '{': '}', '(': ')', '[': ']' }
    const closeMap: Record<string, string> = { '}': '{', ')': '(', ']': '[' }
    const stack: Array<{ char: string; line: number; column: number }> = []
    let index = 0
    let line = 1
    let column = 1
    let quote = ''
    let escaped = false
    let blockComment = false
    let lineComment = false
    let blockCommentLine = 1
    let blockCommentColumn = 1
    let quoteLine = 1
    let quoteColumn = 1

    const move = (count = 1) => {
      for (let step = 0; step < count; step += 1) {
        if (source[index] === '\n') {
          line += 1
          column = 1
        } else {
          column += 1
        }
        index += 1
      }
    }

    while (index < source.length) {
      const char = source[index]
      const next = source[index + 1]

      if (lineComment) {
        if (char === '\n') lineComment = false
        move()
        continue
      }

      if (blockComment) {
        if (char === '*' && next === '/') {
          blockComment = false
          move(2)
          continue
        }
        move()
        continue
      }

      if (quote) {
        if (quote.length === 3) {
          if (source.slice(index, index + 3) === quote) {
            quote = ''
            move(3)
            continue
          }
          move()
          continue
        }

        if (escaped) {
          escaped = false
          move()
          continue
        }
        if (char === '\\') {
          escaped = true
          move()
          continue
        }
        if (char === quote) {
          quote = ''
          move()
          continue
        }
        move()
        continue
      }

      if (char === '/' && next === '/') {
        lineComment = true
        move(2)
        continue
      }
      if (char === '/' && next === '*') {
        blockComment = true
        blockCommentLine = line
        blockCommentColumn = column
        move(2)
        continue
      }
      if (source.slice(index, index + 3) === "'''" || source.slice(index, index + 3) === '"""') {
        quote = source.slice(index, index + 3)
        quoteLine = line
        quoteColumn = column
        move(3)
        continue
      }
      if (char === "'" || char === '"') {
        quote = char
        quoteLine = line
        quoteColumn = column
        move()
        continue
      }

      if (openMap[char]) {
        stack.push({ char, line, column })
        move()
        continue
      }

      if (closeMap[char]) {
        const last = stack.pop()
        if (!last) {
          return { valid: false, line, column, message: `多余的 ${char}` }
        }
        if (last.char !== closeMap[char]) {
          return {
            valid: false,
            line,
            column,
            message: `${last.char} 需要以 ${openMap[last.char]} 结束，当前遇到 ${char}`
          }
        }
      }

      move()
    }

    if (blockComment) {
      return {
        valid: false,
        line: blockCommentLine,
        column: blockCommentColumn,
        message: '块注释未闭合'
      }
    }
    if (quote) {
      return {
        valid: false,
        line: quoteLine,
        column: quoteColumn,
        message: '字符串未闭合'
      }
    }
    const last = stack[stack.length - 1]
    if (last) {
      return {
        valid: false,
        line: last.line,
        column: last.column,
        message: `${last.char} 缺少对应的 ${openMap[last.char]}`
      }
    }

    return { valid: true }
  }

  const extractWholeBlock = (source: string, keyword: string) => {
    const pattern = new RegExp(`^\\s*${keyword}\\s*\\{`)
    const match = source.match(pattern)
    if (!match || match.index !== 0) return null
    const openIndex = source.indexOf('{', match[0].length - 1)
    if (openIndex < 0) return null
    const closeIndex = findMatchingPair(source, openIndex, '{', '}')
    if (closeIndex < 0 || source.slice(closeIndex + 1).trim()) return null
    return source.slice(openIndex + 1, closeIndex)
  }

  const validateContentRule = (value: string): SyntaxCheckResult => {
    const source = (value || '').trim()
    if (!source) return { valid: false, message: '请输入步骤代码' }

    if (props.contentType === 'post') {
      if (
        /\bstage\s*\(/.test(source) ||
        /\bparallel\s*\{/.test(source) ||
        /\bsteps\s*\{/.test(source)
      ) {
        return { valid: false, message: 'post 步骤不能包含 stage、parallel 或 steps' }
      }
      const postBody = extractWholeBlock(source, 'post') ?? source
      let rest = postBody.trim()
      while (rest) {
        const matched = postConditions.find((name) => new RegExp(`^${name}\\s*\\{`).test(rest))
        if (!matched) {
          return {
            valid: false,
            message: 'post 步骤只允许 always、success、failure、changed、aborted、unstable 条件块'
          }
        }
        const openIndex = rest.indexOf('{')
        const closeIndex = findMatchingPair(rest, openIndex, '{', '}')
        if (closeIndex < 0) return { valid: false, message: 'post 条件代码块不完整' }
        rest = rest.slice(closeIndex + 1).trim()
      }
      return { valid: true }
    }

    if (
      /\bstage\s*\(/.test(source) ||
      /\bparallel\s*\{/.test(source) ||
      /\bpost\s*\{/.test(source)
    ) {
      return {
        valid: false,
        message: '步骤代码只能保存 steps 块，不能包含 stage、parallel 或 post'
      }
    }
    if (!extractWholeBlock(source, 'steps')) {
      return { valid: false, message: '步骤代码必须是完整 steps 块' }
    }
    if (postConditions.some((name) => new RegExp(`^\\s*${name}\\s*\\{`).test(source))) {
      return { valid: false, message: 'post 状态条件只能用于 post 步骤分类' }
    }
    return { valid: true }
  }

  const validateCode = (value = props.modelValue, showMessage = true) => {
    const syntaxResult = checkPipelineSyntax(value)
    const result = syntaxResult.valid ? validateContentRule(value) : syntaxResult
    syntaxError.value = result.valid ? '' : formatSyntaxMessage(result)
    if (!result.valid && showMessage && syntaxError.value) {
      ElMessage.error(syntaxError.value)
    }
    return result.valid
  }

  const collectBraceInfos = (lines: string[]) => {
    let quote = ''
    let escaped = false
    let blockComment = false

    return lines.map((line) => {
      let openCount = 0
      let closeCount = 0
      let startsWithClose = false
      let firstTokenFound = false
      let index = 0

      while (index < line.length) {
        const char = line[index]
        const next = line[index + 1]

        if (!firstTokenFound && /\S/.test(char)) {
          firstTokenFound = true
          startsWithClose = char === '}'
        }

        if (blockComment) {
          if (char === '*' && next === '/') {
            blockComment = false
            index += 2
            continue
          }
          index += 1
          continue
        }

        if (quote) {
          if (quote.length === 3) {
            if (line.slice(index, index + 3) === quote) {
              quote = ''
              index += 3
              continue
            }
            index += 1
            continue
          }

          if (escaped) {
            escaped = false
            index += 1
            continue
          }
          if (char === '\\') {
            escaped = true
            index += 1
            continue
          }
          if (char === quote) {
            quote = ''
            index += 1
            continue
          }
          index += 1
          continue
        }

        if (char === '/' && next === '/') break
        if (char === '/' && next === '*') {
          blockComment = true
          index += 2
          continue
        }
        if (line.slice(index, index + 3) === "'''" || line.slice(index, index + 3) === '"""') {
          quote = line.slice(index, index + 3)
          index += 3
          continue
        }
        if (char === "'" || char === '"') {
          quote = char
          index += 1
          continue
        }

        if (char === '{') openCount += 1
        if (char === '}') closeCount += 1
        index += 1
      }

      return { startsWithClose, openCount, closeCount }
    })
  }

  const formatCode = () => {
    if (!validateCode()) return
    const lines = props.modelValue.replace(/\t/g, '  ').split('\n')
    const braceInfos = collectBraceInfos(lines)
    let indent = 0
    const formatted = lines.map((line, index) => {
      const content = line.trim()
      if (!content) return ''
      const braceInfo = braceInfos[index]
      if (braceInfo.startsWithClose) indent = Math.max(indent - 1, 0)
      const nextLine = `${'  '.repeat(indent)}${content}`
      indent = Math.max(indent + braceInfo.openCount - braceInfo.closeCount, 0)
      return nextLine
    })
    updateValue(
      formatted
        .join('\n')
        .replace(/\n{3,}/g, '\n\n')
        .trim()
    )
  }

  watch(
    () => props.modelValue,
    (value) => {
      if (syntaxError.value) validateCode(value, false)
    }
  )

  defineExpose({ formatCode, validateCode })
</script>

<style scoped lang="scss">
  .jenkins-pipeline-editor {
    display: flex;
    flex-direction: column;
    width: 100%;
    min-height: 320px;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;

    &.is-invalid {
      border-color: var(--el-color-danger);
      box-shadow: 0 0 0 1px var(--el-color-danger-light-5);
    }
  }

  .editor-head,
  .editor-foot {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    background: var(--el-fill-color-lighter);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .editor-title {
    display: flex;
    flex: 1;
    gap: 10px;
    align-items: center;
    min-width: 0;

    strong,
    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      color: var(--el-text-color-secondary);
    }
  }

  .editor-actions {
    display: flex;
    flex: 0 0 auto;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
    justify-content: flex-end;
  }

  .editor-body {
    display: grid;
    grid-template-columns: 52px minmax(0, 1fr);
    flex: 1;
    min-height: 0;
    overflow: hidden;
    font-family: var(--ax-font-mono);
    background: #111827;
  }

  .line-numbers,
  .code-textarea {
    padding: 14px 12px;
    margin: 0;
    font-family: inherit;
    font-size: 13px;
    line-height: 1.7;
  }

  .line-numbers {
    overflow: hidden;
    color: #94a3b8;
    text-align: right;
    user-select: none;
    background: #0f172a;
    border-right: 1px solid rgba(148, 163, 184, 0.24);
  }

  .code-textarea {
    width: 100%;
    height: 100%;
    min-height: 0;
    color: #e5e7eb;
    resize: none;
    outline: none;
    background: transparent;
    border: 0;
    caret-color: #60a5fa;
    tab-size: 2;
  }

  .code-textarea::placeholder {
    color: #64748b;
  }

  .editor-foot {
    flex-wrap: wrap;
    padding: 8px 12px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    border-top: 1px solid var(--el-border-color-lighter);
    border-bottom: 0;
  }

  .editor-error {
    padding: 8px 12px;
    font-size: 12px;
    color: var(--el-color-danger);
    background: var(--el-color-danger-light-9);
    border-top: 1px solid var(--el-color-danger-light-7);
  }
</style>

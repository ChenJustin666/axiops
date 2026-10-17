<template>
  <div class="xml-editor" :class="{ 'xml-editor--dark': theme === 'dark' }" :style="{ height }">
    <div class="xml-editor__toolbar">
      <div class="editor-title">
        <ElIcon><Document /></ElIcon>
        <span>{{ filename }}</span>
      </div>
      <div class="toolbar-actions">
        <button
          type="button"
          class="tool-button"
          :disabled="readonly || !localValue"
          @click="formatXml"
        >
          <ElIcon><MagicStick /></ElIcon>
          <span>格式化</span>
        </button>
        <button type="button" class="tool-button" :disabled="!localValue" @click="copyContent">
          <ElIcon><CopyDocument /></ElIcon>
          <span>复制</span>
        </button>
        <button
          type="button"
          class="tool-button"
          :disabled="readonly || !localValue"
          @click="clearContent"
        >
          <ElIcon><Delete /></ElIcon>
          <span>清空</span>
        </button>
        <button type="button" class="tool-button" @click="showLineNumbers = !showLineNumbers">
          <ElIcon><Document /></ElIcon>
          <span>{{ showLineNumbers ? '隐藏行号' : '显示行号' }}</span>
        </button>
      </div>
      <div
        class="editor-status"
        :class="{ error: !!errorMessage, success: isValid && !!localValue }"
      >
        {{ statusText }}
      </div>
    </div>

    <div class="xml-editor__body">
      <div v-show="showLineNumbers" ref="gutterRef" class="line-gutter">
        <div
          v-for="line in lineCount"
          :key="line"
          class="line-number"
          :class="{ error: errorLine === line }"
        >
          {{ line }}
        </div>
      </div>
      <textarea
        ref="textareaRef"
        v-model="localValue"
        class="xml-textarea"
        :readonly="readonly"
        :placeholder="placeholder"
        spellcheck="false"
        @input="handleInput"
        @scroll="syncScroll"
        @keydown="handleKeydown"
        @click="updateCursor"
        @keyup="updateCursor"
        @blur="validateXml"
      />
    </div>

    <div class="xml-editor__footer">
      <span>Ln {{ currentLine }}, Col {{ currentColumn }}</span>
      <span>{{ localValue.length }} 字符</span>
      <span>XML</span>
      <span v-if="errorMessage" class="error-message" :title="errorMessage">
        {{ errorMessage }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue'
  import { ElIcon, ElMessage, ElMessageBox } from 'element-plus'
  import { CopyDocument, Delete, Document, MagicStick } from '@element-plus/icons-vue'

  interface XmlError {
    line: number
    column: number
    message: string
  }

  const props = withDefaults(
    defineProps<{
      modelValue?: string
      filename?: string
      height?: string
      readonly?: boolean
      placeholder?: string
      theme?: 'light' | 'dark'
    }>(),
    {
      modelValue: '',
      filename: 'settings.xml',
      height: '420px',
      readonly: false,
      placeholder: '请输入 settings.xml 内容',
      theme: 'light'
    }
  )

  const emit = defineEmits<{
    'update:modelValue': [value: string]
    change: [value: string]
    'valid-change': [valid: boolean]
    error: [error: XmlError | null]
  }>()

  const localValue = ref(props.modelValue)
  const textareaRef = ref<HTMLTextAreaElement>()
  const gutterRef = ref<HTMLElement>()
  const showLineNumbers = ref(true)
  const currentLine = ref(1)
  const currentColumn = ref(1)
  const isValid = ref(true)
  const errorLine = ref(0)
  const errorMessage = ref('')
  let validateTimer: ReturnType<typeof setTimeout> | undefined

  const lineCount = computed(() => Math.max(localValue.value.split('\n').length, 1))
  const statusText = computed(() => {
    if (!localValue.value) return '就绪'
    return isValid.value ? '结构正常' : '结构错误'
  })

  const handleInput = () => {
    emit('update:modelValue', localValue.value)
    emit('change', localValue.value)
    updateCursor()
    scheduleValidate()
  }

  const scheduleValidate = () => {
    if (validateTimer) clearTimeout(validateTimer)
    validateTimer = setTimeout(() => validateXml(), 300)
  }

  const validateXml = () => {
    const error = getXmlError(localValue.value)
    isValid.value = !error
    errorLine.value = error?.line || 0
    errorMessage.value = error?.message || ''
    emit('valid-change', !error)
    emit('error', error)
    return !error
  }

  const getXmlError = (value: string): XmlError | null => {
    if (!value.trim()) return null

    const parser = new DOMParser()
    const doc = parser.parseFromString(sanitizeCommentsForValidation(value), 'application/xml')
    const parserError = doc.getElementsByTagName('parsererror')[0]
    if (!parserError) return null

    const message = parserError.textContent || 'XML 语法错误'
    const position = extractPosition(message)
    return {
      line: position.line,
      column: position.column,
      message: simplifyXmlError(message)
    }
  }

  const sanitizeCommentsForValidation = (value: string) => {
    return value.replace(/<!--[\s\S]*?-->/g, '<!-- comment -->')
  }

  const extractPosition = (message: string) => {
    const match =
      message.match(/line\s+(\d+)\s+(?:at\s+)?column\s+(\d+)/i) ||
      message.match(/第\s*(\d+)\s*行.*第\s*(\d+)\s*列/)

    return {
      line: match ? Number(match[1]) : 1,
      column: match ? Number(match[2]) : 1
    }
  }

  const simplifyXmlError = (message: string) => {
    return message
      .split('\n')
      .map((item) => item.trim())
      .filter(Boolean)
      .slice(0, 2)
      .join('，')
  }

  const formatXml = () => {
    if (!localValue.value.trim()) {
      ElMessage.warning('没有内容需要格式化')
      return
    }

    const formatted = formatXmlContent(localValue.value)
    localValue.value = formatted
    emit('update:modelValue', formatted)
    emit('change', formatted)
    nextTick(() => {
      syncScroll()
      validateXml()
      updateCursor()
    })
    ElMessage.success('格式化成功')
  }

  const formatXmlContent = (value: string) => {
    const tokens =
      value.trim().match(/<!--[\s\S]*?-->|<!\[CDATA\[[\s\S]*?\]\]>|<[^>]+>|[^<]+/g) || []
    const result: string[] = []
    let indent = 0

    for (let i = 0; i < tokens.length; i++) {
      const token = normalizeToken(tokens[i])
      if (!token) continue

      const nextToken = normalizeToken(tokens[i + 1] || '')
      const afterNextToken = normalizeToken(tokens[i + 2] || '')
      const openingName = getOpeningName(token)
      const closingName = getClosingName(afterNextToken)

      if (openingName && nextToken && closingName === openingName && !nextToken.startsWith('<')) {
        result.push(`${'  '.repeat(indent)}${token}${nextToken}${afterNextToken}`)
        i += 2
        continue
      }

      const isClosingTag = /^<\//.test(token)
      const isDeclaration = /^<\?/.test(token)
      const isCommentOrDoctype = /^<!/.test(token)
      const isSelfClosing = /\/>$/.test(token)

      if (isClosingTag) indent = Math.max(indent - 1, 0)
      result.push(`${'  '.repeat(indent)}${token}`)
      if (openingName && !isSelfClosing && !isDeclaration && !isCommentOrDoctype) indent += 1
    }

    return result.join('\n')
  }

  const normalizeToken = (token: string) => {
    const value = token.trim()
    if (!value) return ''
    if (!value.startsWith('<') || value.startsWith('<![CDATA[') || value.startsWith('<!--')) {
      return value
    }
    return value.replace(/\s+/g, ' ').replace(/\s+([/>])/g, '$1')
  }

  const getOpeningName = (token: string) => token.match(/^<([\w:.-]+)/)?.[1] || ''
  const getClosingName = (token: string) => token.match(/^<\/([\w:.-]+)/)?.[1] || ''

  const copyContent = async () => {
    if (!localValue.value) {
      ElMessage.warning('没有内容可复制')
      return
    }
    await navigator.clipboard.writeText(localValue.value)
    ElMessage.success('已复制')
  }

  const clearContent = async () => {
    await ElMessageBox.confirm('确定清空配置内容吗？', '清空确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    localValue.value = ''
    handleInput()
  }

  const handleKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Tab' && !props.readonly) {
      event.preventDefault()
      const target = event.target as HTMLTextAreaElement
      const start = target.selectionStart
      const end = target.selectionEnd
      localValue.value = `${localValue.value.slice(0, start)}  ${localValue.value.slice(end)}`
      emit('update:modelValue', localValue.value)
      emit('change', localValue.value)
      nextTick(() => {
        target.selectionStart = target.selectionEnd = start + 2
      })
    }

    if ((event.metaKey || event.ctrlKey) && event.shiftKey && event.key.toLowerCase() === 'f') {
      event.preventDefault()
      formatXml()
    }
  }

  const syncScroll = () => {
    if (gutterRef.value && textareaRef.value) {
      gutterRef.value.scrollTop = textareaRef.value.scrollTop
    }
  }

  const updateCursor = () => {
    if (!textareaRef.value) return
    const text = localValue.value.slice(0, textareaRef.value.selectionStart)
    const lines = text.split('\n')
    currentLine.value = lines.length
    currentColumn.value = lines[lines.length - 1].length + 1
  }

  watch(
    () => props.modelValue,
    (value) => {
      if (value !== localValue.value) {
        localValue.value = value || ''
        nextTick(() => {
          validateXml()
          updateCursor()
        })
      }
    },
    { immediate: true }
  )
</script>

<style scoped lang="scss">
  .xml-editor {
    display: flex;
    flex-direction: column;
    width: 100%;
    overflow: hidden;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    background: var(--el-bg-color);
  }

  .editor-title {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  .editor-status {
    overflow: hidden;
    color: var(--el-text-color-secondary);
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;

    &.success {
      color: #73c991;
    }

    &.error {
      color: var(--el-color-danger);
    }
  }

  .xml-editor__toolbar {
    display: grid;
    grid-template-columns: 180px 1fr 120px;
    align-items: center;
    gap: 12px;
    min-height: 42px;
    padding: 0 12px;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .toolbar-actions {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .tool-button {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 30px;
    padding: 0 10px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 4px;

    &:hover:not(:disabled) {
      color: var(--el-color-primary);
      background: var(--el-fill-color);
    }

    &:disabled {
      cursor: not-allowed;
      opacity: 0.45;
    }
  }

  .xml-editor__body {
    display: flex;
    flex: 1;
    min-height: 0;
    background: #ffffff;
  }

  .line-gutter {
    width: 56px;
    padding: 8px 0;
    overflow: hidden;
    color: var(--el-text-color-placeholder);
    background: var(--el-fill-color-lighter);
    border-right: 1px solid var(--el-border-color-lighter);
  }

  .line-number {
    height: 21px;
    padding-right: 12px;
    font-family: var(--ax-font-mono);
    font-size: 13px;
    line-height: 21px;
    text-align: right;

    &.error {
      color: var(--el-color-danger);
      background: var(--el-color-danger-light-9);
    }
  }

  .xml-textarea {
    flex: 1;
    min-width: 0;
    height: 100%;
    padding: 8px 12px;
    overflow: auto;
    font-family: var(--ax-font-mono);
    font-size: 13px;
    line-height: 21px;
    color: var(--el-text-color-primary);
    white-space: pre;
    resize: none;
    background: #ffffff;
    border: 0;
    outline: none;
    tab-size: 2;

    &::placeholder {
      color: var(--el-text-color-placeholder);
    }
  }

  .xml-editor__footer {
    display: flex;
    align-items: center;
    gap: 16px;
    min-height: 28px;
    padding: 0 12px;
    overflow: hidden;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .error-message {
    min-width: 0;
    overflow: hidden;
    color: var(--el-color-danger);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .xml-editor--dark {
    color: #dbeafe;
    background: #0f172a;
    border-color: #27364d;
    box-shadow: 0 14px 30px rgb(15 23 42 / 12%);

    .xml-editor__toolbar,
    .xml-editor__footer {
      color: #94a3b8;
      background: #0b1220;
      border-color: #243047;
    }

    .editor-title {
      color: #e5e7eb;
    }

    .editor-status {
      color: #94a3b8;

      &.success {
        color: #34d399;
      }

      &.error {
        color: #f87171;
      }
    }

    .tool-button {
      color: #cbd5e1;

      &:hover:not(:disabled) {
        color: #ffffff;
        background: #1e293b;
      }
    }

    .xml-editor__body {
      background: #111827;
    }

    .line-gutter {
      color: #64748b;
      background: #0b1220;
      border-right-color: #243047;
    }

    .line-number.error {
      color: #fecaca;
      background: rgb(248 113 113 / 16%);
    }

    .xml-textarea {
      color: #e5e7eb;
      caret-color: #60a5fa;
      background: #111827;

      &::placeholder {
        color: #64748b;
      }
    }

    .error-message {
      color: #fca5a5;
    }
  }
</style>

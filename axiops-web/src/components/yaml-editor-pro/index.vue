<template>
  <div
    class="yaml-editor-pro"
    :class="{ 'light-editor': !isDarkTheme }"
    :style="{ width: width, height: height }"
  >
    <!-- Mac 风格标题栏 -->
    <div v-if="showTitleBar" class="editor-titlebar">
      <div class="window-controls">
        <div class="control close" @click="handleClose">
          <svg width="12" height="12" viewBox="0 0 12 12">
            <path stroke="currentColor" stroke-width="1.5" d="M3 3l6 6m0-6L3 9" />
          </svg>
        </div>
        <div class="control minimize" @click="handleMinimize">
          <svg width="12" height="12" viewBox="0 0 12 12">
            <path stroke="currentColor" stroke-width="1.5" d="M2 6h8" />
          </svg>
        </div>
        <div class="control maximize" @click="handleMaximize">
          <svg width="12" height="12" viewBox="0 0 12 12">
            <rect
              x="2"
              y="2"
              width="8"
              height="8"
              stroke="currentColor"
              stroke-width="1.5"
              fill="none"
            />
          </svg>
        </div>
      </div>
      <div class="title-text">
        <el-icon :size="14">
          <Document />
        </el-icon>
        {{ filename || 'untitled.yaml' }}
        <span v-if="readonly" class="readonly-badge">
          <el-icon :size="12"><Lock /></el-icon>
          只读
        </span>
      </div>
      <div class="title-status">
        <span v-if="!readonly" class="status-dot" :class="statusClass"></span>
        <span class="status-text">{{ statusText }}</span>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="editor-toolbar" v-if="showToolbar">
      <div class="toolbar-group">
        <!-- 格式化按钮 - 只读模式下也可用 -->
        <el-tooltip content="格式化 (Ctrl+Shift+F)" placement="bottom">
          <button
            class="tool-button"
            :class="{ disabled: !localContent || readonly }"
            @click="formatYaml"
            :disabled="!localContent || readonly"
          >
            <el-icon>
              <MagicStick />
            </el-icon>
            <span>格式化</span>
          </button>
        </el-tooltip>

        <div class="toolbar-divider"></div>

        <!-- 复制按钮 - 始终可用 -->
        <el-tooltip content="复制到剪贴板" placement="bottom">
          <button
            class="tool-button"
            :class="{ disabled: !localContent }"
            @click="copyContent"
            :disabled="!localContent"
          >
            <el-icon>
              <CopyDocument />
            </el-icon>
            <span>复制</span>
          </button>
        </el-tooltip>

        <!-- 清空按钮 - 只读模式下隐藏 -->
        <el-tooltip content="清空内容" placement="bottom" v-if="!readonly">
          <button class="tool-button" @click="clearContent">
            <el-icon>
              <Delete />
            </el-icon>
            <span>清空</span>
          </button>
        </el-tooltip>

        <!-- 导出按钮 - 始终可用 -->
        <el-tooltip content="导出文件" placement="bottom">
          <button class="tool-button" @click="exportYaml" :disabled="!localContent">
            <el-icon>
              <Download />
            </el-icon>
            <span>导出</span>
          </button>
        </el-tooltip>
      </div>

      <div class="toolbar-group">
        <!-- 切换行号 - 始终可用 -->
        <button class="tool-button" @click="toggleLineNumbers">
          <el-icon>
            <Document />
          </el-icon>
          <span>{{ showLineNumbersLocal ? '隐藏' : '显示' }}行号</span>
        </button>

        <!-- 切换高亮 - 始终可用 -->
        <button class="tool-button" @click="toggleHighlight">
          <el-icon>
            <Brush />
          </el-icon>
          <span>{{ enableHighlight ? '关闭' : '开启' }}高亮</span>
        </button>

        <!-- 切换主题 - 始终可用 -->
        <button class="tool-button" @click="toggleTheme">
          <el-icon>
            <component :is="isDarkTheme ? Sunny : Moon" />
          </el-icon>
          <span>{{ isDarkTheme ? '亮色' : '暗色' }}主题</span>
        </button>
      </div>
    </div>

    <!-- 编辑器主体 -->
    <div class="editor-body" :class="{ 'light-theme': !isDarkTheme }">
      <!-- 行号区域 -->
      <div class="gutter" v-show="showLineNumbersLocal" ref="gutterRef">
        <div v-for="(line, index) in lines" :key="index" class="gutter-line">
          <span
            class="line-number"
            :class="{
              'has-error': hasError(index + 1),
              'current-line': currentLine === index + 1
            }"
            @click="gotoLine(index + 1)"
          >
            {{ index + 1 }}
            <span v-if="hasError(index + 1) && showErrorMarkers" class="error-marker">
              <svg width="6" height="6" viewBox="0 0 6 6">
                <circle cx="3" cy="3" r="3" fill="#ff5f57" />
              </svg>
            </span>
          </span>
        </div>
      </div>

      <!-- 编辑器包装 -->
      <div class="editor-wrapper" :class="{ 'no-gutter': !showLineNumbersLocal }">
        <!-- 高亮层 -->
        <pre
          v-if="enableHighlight"
          class="highlight-layer"
          ref="highlightLayerRef"
          :style="{ fontSize: fontSize + 'px', fontFamily: fontFamily }"
        ><code v-html="highlightedContent"></code></pre>

        <!-- 文本输入层 -->
        <textarea
          ref="editorRef"
          v-model="localContent"
          :class="textareaClasses"
          :style="{
            fontSize: fontSize + 'px',
            fontFamily: fontFamily,
            tabSize: tabSize
          }"
          :readonly="readonly"
          :disabled="readonly"
          :placeholder="placeholder"
          @beforeinput="handleBeforeInput"
          @input="handleInput"
          @scroll="handleScroll"
          @keydown="handleKeydown"
          @keypress="handleKeypress"
          @paste="handlePaste"
          @cut="handleCut"
          @click="updateCursor"
          @keyup="updateCursor"
          @focus="handleFocus"
          @blur="handleBlur"
          @compositionstart="onCompositionStart"
          @compositionend="onCompositionEnd"
          spellcheck="false"
        />

        <!-- 错误波浪线层 -->
        <svg
          v-if="showErrorWave && errorLinePositions.length > 0"
          class="error-wave-svg"
          :style="{ transform: `translateY(-${scrollTop}px)` }"
        >
          <defs>
            <pattern id="wave" patternUnits="userSpaceOnUse" width="6" height="3">
              <path
                d="M0 3c1.5-2 2.5-2 3 0s1.5 2 3 0"
                stroke="#ff0000"
                fill="none"
                stroke-width="1.2"
                opacity="0.8"
              />
            </pattern>
          </defs>
          <g v-for="(pos, index) in errorLinePositions" :key="`error-${index}`">
            <rect
              :x="pos.x - 2"
              :y="pos.y - 1"
              :width="pos.width + 4"
              height="5"
              fill="rgba(255, 0, 0, 0.1)"
            />
            <rect :x="pos.x" :y="pos.y" :width="pos.width" height="3" fill="url(#wave)" />
          </g>
        </svg>

        <!-- 只读模式覆盖层 -->
        <div v-if="readonly" class="readonly-overlay">
          <el-icon :size="48" class="readonly-icon">
            <Lock />
          </el-icon>
        </div>
      </div>
    </div>

    <!-- 状态栏 -->
    <div class="editor-statusbar" v-if="showStatusBar">
      <div class="status-left">
        <span class="status-item clickable" @click="showGotoLineDialog">
          <el-icon><Position /></el-icon>
          Ln {{ currentLine }}, Col {{ currentColumn }}
        </span>
        <span class="status-item">
          <el-icon><Files /></el-icon>
          {{ localContent.length }} 字符
        </span>
        <span class="status-item error" v-if="errorCount > 0">
          <el-icon><WarningFilled /></el-icon>
          {{ errorCount }} 个错误
        </span>
        <span class="status-item success" v-else-if="localContent && errorCount === 0">
          <el-icon><CircleCheckFilled /></el-icon>
          语法正确
        </span>
      </div>
      <div class="status-right">
        <span class="status-item">
          <el-icon><Ticket /></el-icon>
          YAML
        </span>
        <span class="status-item" v-if="readonly">
          <el-icon><Lock /></el-icon>
          只读
        </span>
        <span class="status-item">
          {{ isDarkTheme ? '🌙' : '☀️' }}
          {{ isDarkTheme ? '暗色' : '亮色' }}主题
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
  import { ElMessage, ElMessageBox, ElIcon, ElTooltip } from 'element-plus'
  import * as yaml from 'js-yaml'
  import {
    formatYamlSource,
    indentYamlSelection,
    yamlNewline,
    yamlSyntaxSource,
    type YamlTextEdit
  } from '@/utils/yaml-editing'
  import {
    Document,
    MagicStick,
    CopyDocument,
    Delete,
    Download,
    Brush,
    Sunny,
    Moon,
    WarningFilled,
    CircleCheckFilled,
    Lock,
    Position,
    Files,
    Ticket
  } from '@element-plus/icons-vue'

  // Props
  interface Props {
    modelValue?: string
    width?: string
    height?: string
    filename?: string
    placeholder?: string
    readonly?: boolean
    showToolbar?: boolean
    showTitleBar?: boolean
    showLineNumbers?: boolean
    showStatusBar?: boolean
    theme?: 'dark' | 'light'
    validateOnChange?: boolean
    enableHighlight?: boolean
    autoFormat?: boolean
    tabSize?: number
    wordWrap?: boolean
    fontSize?: number
    fontFamily?: string
    showErrorMarkers?: boolean
    showErrorWave?: boolean
    errorTolerant?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    modelValue: '',
    width: '100%',
    height: '600px',
    filename: 'config.yaml',
    placeholder:
      '# 🎯 在此输入 YAML 内容\n# ✨ 支持语法高亮、实时验证\n# 💡 使用 Tab 键进行缩进，Ctrl+S 保存',
    readonly: false,
    showToolbar: true,
    showTitleBar: true,
    showLineNumbers: true,
    showStatusBar: true,
    theme: 'dark',
    validateOnChange: true,
    enableHighlight: true,
    autoFormat: false,
    tabSize: 2,
    wordWrap: false,
    fontSize: 14,
    fontFamily: 'var(--ax-font-mono)',
    showErrorMarkers: true,
    showErrorWave: true,
    errorTolerant: false
  })

  const emit = defineEmits<{
    'update:modelValue': [value: string]
    change: [value: string]
    'valid-change': [valid: boolean]
    error: [error: any]
    save: [value: string]
    format: [value: string]
  }>()

  // 类型定义
  interface ErrorInfo {
    line: number
    column: number
    message: string
  }

  // ============ 新增：多行字符串块状态跟踪接口 ============
  interface MultilineBlockState {
    inBlock: boolean // 是否在多行块内
    blockIndent: number // 块的基准缩进
    blockStartLine: number // 块开始的行号
  }

  // 响应式状态
  const editorRef = ref<HTMLTextAreaElement>()
  const gutterRef = ref<HTMLElement>()
  const highlightLayerRef = ref<HTMLElement>()
  const localContent = ref('')
  const currentLine = ref(1)
  const currentColumn = ref(1)
  const errorLines = ref<ErrorInfo[]>([])
  const errorCount = ref(0)
  const showLineNumbersLocal = ref(props.showLineNumbers)
  const isDarkTheme = ref(props.theme === 'dark')
  const focused = ref(false)
  const enableHighlight = ref(props.enableHighlight)
  const scrollTop = ref(0)
  const isComposing = ref(false)
  const validateTimer = ref<ReturnType<typeof setTimeout> | null>(null)

  // 计算属性
  const lines = computed(() => localContent.value.split('\n'))

  const errorLineSet = computed(() => {
    const set = new Set<number>()
    for (const error of errorLines.value) {
      set.add(error.line)
    }
    return set
  })

  const statusClass = computed(() => {
    if (errorCount.value > 0) return 'error'
    if (localContent.value && errorCount.value === 0) return 'success'
    return 'normal'
  })

  const statusText = computed(() => {
    if (!props.validateOnChange) return ''
    if (props.readonly) return '📖 只读'
    if (errorCount.value > 0) return `❌ ${errorCount.value} 个语法错误`
    if (localContent.value && errorCount.value === 0) return '✅ 语法正确'
    return '📝 就绪'
  })

  const textareaClasses = computed(() => {
    const classes = ['editor-textarea']
    if (enableHighlight.value) {
      classes.push('with-highlight')
    }
    if (props.wordWrap) {
      classes.push('word-wrap')
    }
    if (props.readonly) {
      classes.push('readonly')
    }
    return classes.join(' ')
  })

  const errorLinePositions = computed(() => {
    const positions: any[] = []
    const lines = localContent.value.split('\n')

    errorLineSet.value.forEach((lineNumber) => {
      const lineIndex = lineNumber - 1
      if (lineIndex < 0 || lineIndex >= lines.length) return

      const lineText = lines[lineIndex]
      const trimmedText = lineText.trim()
      const minWidth = 40

      const indent = lineText.length - lineText.trimStart().length

      const charWidth = 7.2
      const lineHeight = 21
      const paddingLeft = showLineNumbersLocal.value ? 12 : 20

      const width =
        trimmedText.length > 0 ? Math.max(trimmedText.length * charWidth, minWidth) : minWidth

      positions.push({
        x: paddingLeft + indent * charWidth,
        y: lineIndex * lineHeight + 18,
        width: width
      })
    })

    return positions
  })

  const isYamlDocumentSeparator = (text: string): boolean => {
    return /^---(?:\s+#.*)?$/.test(text) || /^\.\.\.(?:\s+#.*)?$/.test(text)
  }

  const highlightedContent = computed(() => {
    if (!enableHighlight.value) return escapeHtml(localContent.value)

    const lines = localContent.value.split('\n')
    return lines
      .map((line) => {
        if (!line.trim()) {
          return '<span class="line">&nbsp;</span>'
        }

        if (isYamlDocumentSeparator(line.trim())) {
          return `<span class="line"><span style="color: ${isDarkTheme.value ? '#c586c0' : '#8250df'}; font-weight: 600;">${escapeHtml(line)}</span></span>`
        }

        if (line.trim().startsWith('#')) {
          return `<span class="line"><span style="color: ${isDarkTheme.value ? '#6a9955' : '#008000'}; font-style: italic;">${escapeHtml(line)}</span></span>`
        }

        if (line.trim().startsWith('-')) {
          const match = line.match(/^(\s*)(-)(\s*)(.*)$/)
          if (match) {
            const [, indent, dash, space, content] = match
            let result = `<span class="line">${escapeHtml(indent)}<span style="color: ${isDarkTheme.value ? '#c586c0' : '#0000ff'}; font-weight: bold;">${escapeHtml(dash)}</span>${escapeHtml(space)}`

            if (content.includes(':')) {
              const kvMatch = content.match(/^([^:]+)(:)(.*)$/)
              if (kvMatch) {
                const [, key, colon, value] = kvMatch
                result += `<span style="color: ${isDarkTheme.value ? '#f14c4c' : '#cf222e'}; font-weight: 600;">${escapeHtml(key)}</span>`
                result += `<span style="color: ${isDarkTheme.value ? '#cccccc' : '#000000'};">${escapeHtml(colon)}</span>`
                if (value.trim()) {
                  result += getValueHighlight(value, isDarkTheme.value)
                }
              }
            } else if (content.trim()) {
              result += `<span style="color: ${isDarkTheme.value ? '#73c991' : '#0a3069'};">${escapeHtml(content)}</span>`
            }

            result += '</span>'
            return result
          }
        }

        const keyMatch = line.match(/^(\s*)([^:]+)(:)(.*)$/)
        if (keyMatch) {
          const [, indent, key, colon, value] = keyMatch
          let result = `<span class="line">`
          result += escapeHtml(indent)
          result += `<span style="color: ${isDarkTheme.value ? '#f14c4c' : '#cf222e'}; font-weight: 600;">${escapeHtml(key)}</span>`
          result += `<span style="color: ${isDarkTheme.value ? '#cccccc' : '#000000'};">${escapeHtml(colon)}</span>`

          if (value.trim()) {
            result += getValueHighlight(value, isDarkTheme.value)
          } else {
            result += escapeHtml(value)
          }

          result += '</span>'
          return result
        }

        return `<span class="line"><span style="color: ${isDarkTheme.value ? '#73c991' : '#0a3069'};">${escapeHtml(line)}</span></span>`
      })
      .join('\n')
  })

  const getValueHighlight = (value: string, isDark: boolean): string => {
    const trimmedValue = value.trim()

    if (trimmedValue.match(/^(true|false|null|yes|no)$/i)) {
      return `<span style="color: ${isDark ? '#569cd6' : '#0550ae'};">${escapeHtml(value)}</span>`
    }

    if (trimmedValue.match(/^-?\d+(\.\d+)?$/)) {
      return `<span style="color: ${isDark ? '#b5cea8' : '#0550ae'};">${escapeHtml(value)}</span>`
    }

    if (
      (trimmedValue.startsWith('"') && trimmedValue.endsWith('"')) ||
      (trimmedValue.startsWith("'") && trimmedValue.endsWith("'"))
    ) {
      return `<span style="color: ${isDark ? '#ce9178' : '#a31515'};">${escapeHtml(value)}</span>`
    }

    if (
      trimmedValue.includes(':') &&
      !trimmedValue.startsWith('"') &&
      !trimmedValue.startsWith("'")
    ) {
      return `<span style="color: ${isDark ? '#f14c4c' : '#cf222e'};">${escapeHtml(value)}</span>`
    }

    return `<span style="color: ${isDark ? '#73c991' : '#0a3069'};">${escapeHtml(value)}</span>`
  }

  const htmlEscapeMap: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  }

  const escapeHtml = (text: string): string => {
    return text.replace(/[&<>"']/g, (char) => htmlEscapeMap[char])
  }

  const hasError = (lineNumber: number): boolean => {
    return errorLineSet.value.has(lineNumber)
  }

  const validateYaml = (immediate = false) => {
    if (validateTimer.value) {
      clearTimeout(validateTimer.value)
    }

    if (immediate) {
      doValidate()
    } else {
      validateTimer.value = setTimeout(() => {
        doValidate()
      }, 500)
    }
  }

  // ============ 核心修复：改进的 doValidate 函数 ============
  const doValidate = () => {
    if (!props.validateOnChange || !localContent.value) {
      errorLines.value = []
      errorCount.value = 0
      emit('valid-change', true)
      return true
    }

    const errors: ErrorInfo[] = []
    const lines = localContent.value.split('\n')

    // 首先使用 js-yaml 进行基础验证（支持多文档）
    try {
      yaml.loadAll(yamlSyntaxSource(localContent.value), () => {})
    } catch (error: any) {
      if (error.mark) {
        errors.push({
          line: error.mark.line + 1,
          column: error.mark.column + 1,
          message: error.message || 'YAML 语法错误'
        })
      }
    }

    // ============ 新增：多行字符串块检测 ============
    // 检测哪些行属于多行字符串块（> 或 |）
    const multilineBlockLines = new Set<number>()
    let blockState: MultilineBlockState = {
      inBlock: false,
      blockIndent: 0,
      blockStartLine: 0
    }

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]
      const trimmedLine = line.trim()
      const currentIndent = line.length - line.trimStart().length

      // 检测多行字符串块的开始（以 > 或 | 结尾，可能带有修饰符如 >- 或 |+）
      if (trimmedLine.match(/:\s*[>|][-+]?\s*$/) || trimmedLine.match(/^[>|][-+]?\s*$/)) {
        blockState = {
          inBlock: true,
          blockIndent: currentIndent,
          blockStartLine: i
        }
        continue
      }

      // 如果在块内
      if (blockState.inBlock) {
        // 空行在块内是允许的
        if (!trimmedLine) {
          multilineBlockLines.add(i + 1)
          continue
        }

        // 如果当前行的缩进大于块开始行的缩进，则仍在块内
        if (currentIndent > blockState.blockIndent) {
          multilineBlockLines.add(i + 1)
          continue
        }

        // 如果缩进小于等于块开始行，则块结束
        blockState.inBlock = false
      }
    }

    // ============ 改进的自定义验证（跳过多行块内容）============
    let lastValidIndent = 0
    let inLabelSection = false
    let inAnnotationSection = false

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]
      const trimmedLine = line.trim()
      const currentIndent = line.length - line.trimStart().length
      const lineNumber = i + 1

      // 跳过空行和注释
      if (!trimmedLine || trimmedLine.startsWith('#')) continue

      // 跳过 YAML 多文档分隔符
      if (isYamlDocumentSeparator(trimmedLine)) {
        inLabelSection = false
        inAnnotationSection = false
        continue
      }

      // ============ 关键修复：跳过多行字符串块内的行 ============
      if (multilineBlockLines.has(lineNumber)) {
        continue
      }

      // 检测 labels/annotations 节
      if (trimmedLine === 'labels:') {
        inLabelSection = true
        inAnnotationSection = false
        lastValidIndent = currentIndent
        continue
      } else if (trimmedLine === 'annotations:') {
        inLabelSection = false
        inAnnotationSection = true
        lastValidIndent = currentIndent
        continue
      } else if (currentIndent <= lastValidIndent && (inLabelSection || inAnnotationSection)) {
        inLabelSection = false
        inAnnotationSection = false
      }

      // labels/annotations 节内的验证
      if ((inLabelSection || inAnnotationSection) && currentIndent > lastValidIndent) {
        if (!trimmedLine.includes(':')) {
          if (!errors.some((e) => e.line === lineNumber)) {
            errors.push({
              line: lineNumber,
              column: currentIndent + 1,
              message: '期望键值对格式 (key: value)'
            })
          }
        }
      }

      // 检查缩进（2的倍数）
      if (currentIndent % 2 !== 0 && trimmedLine) {
        if (!errors.some((e) => e.line === lineNumber)) {
          errors.push({
            line: lineNumber,
            column: currentIndent + 1,
            message: '缩进错误：应使用2个空格的倍数'
          })
        }
      }

      // ============ 改进：冒号后空格检查（排除特殊情况）============
      // 只检查看起来像 YAML 键值对的行，排除多行块标识符
      if (trimmedLine.includes(':') && !trimmedLine.endsWith(':')) {
        // 排除多行块标识符（如 expr: > 或 description: |）
        if (trimmedLine.match(/:\s*[>|][-+]?\s*$/)) {
          continue
        }

        // 查找第一个冒号的位置（排除引号内的冒号）
        let colonIndex = -1
        let inQuote = false
        let quoteChar = ''

        for (let j = 0; j < trimmedLine.length; j++) {
          const char = trimmedLine[j]

          if (!inQuote && (char === '"' || char === "'")) {
            inQuote = true
            quoteChar = char
          } else if (inQuote && char === quoteChar) {
            inQuote = false
            quoteChar = ''
          } else if (!inQuote && char === ':') {
            colonIndex = j
            break
          }
        }

        // 如果找到了非引号内的冒号，检查其后是否有空格
        if (colonIndex !== -1 && colonIndex < trimmedLine.length - 1) {
          const afterColon = trimmedLine[colonIndex + 1]
          // 允许的情况：空格、引号开始的值
          if (afterColon !== ' ' && afterColon !== '"' && afterColon !== "'") {
            // 再次检查这是否是一个实际的 YAML 键（不是 URL 或其他包含冒号的值）
            const beforeColon = trimmedLine.substring(0, colonIndex)
            // 如果冒号前面看起来像一个有效的 YAML 键（字母数字下划线横杠）
            if (beforeColon.match(/^[a-zA-Z_][a-zA-Z0-9_-]*$/)) {
              if (!errors.some((e) => e.line === lineNumber)) {
                errors.push({
                  line: lineNumber,
                  column: line.indexOf(':') + 2,
                  message: '冒号后需要空格'
                })
              }
            }
          }
        }
      }

      // A folded YAML scalar can contain lines beginning with -XX: or -D.
      // The YAML parser above owns sequence syntax validation; treating every
      // leading hyphen as a list item marks valid JVM options as errors.
    }

    // 去重并排序
    const uniqueErrors = errors.reduce((acc: ErrorInfo[], curr) => {
      if (!acc.some((e) => e.line === curr.line)) {
        acc.push(curr)
      }
      return acc
    }, [])

    uniqueErrors.sort((a, b) => a.line - b.line)

    errorLines.value = uniqueErrors
    errorCount.value = uniqueErrors.length

    emit('valid-change', uniqueErrors.length === 0)
    emit('error', uniqueErrors.length > 0 ? uniqueErrors[0] : null)
    return uniqueErrors.length === 0
  }

  const applyTextEdit = (edit: YamlTextEdit) => {
    const textarea = editorRef.value
    if (!textarea || props.readonly) return
    textarea.focus()
    textarea.setSelectionRange(edit.start, edit.end)
    // Native insertText keeps keyboard edits and formatting in the undo history.
    if (!document.execCommand('insertText', false, edit.text)) {
      textarea.setRangeText(edit.text, edit.start, edit.end, 'end')
    }
    localContent.value = textarea.value
    emit('update:modelValue', localContent.value)
    emit('change', localContent.value)
    nextTick(() => {
      textarea.setSelectionRange(edit.selectionStart, edit.selectionEnd)
      updateCursor()
      handleScroll()
    })
  }

  const formatYaml = async () => {
    if (props.readonly) return
    if (!localContent.value) {
      ElMessage.warning('没有内容需要格式化')
      return
    }

    try {
      const original = localContent.value
      const formatted = await formatYamlSource(original, props.tabSize)
      if (props.readonly || localContent.value !== original) return
      if (formatted !== original) {
        const position = Math.min(editorRef.value?.selectionStart || 0, formatted.length)
        applyTextEdit({
          start: 0,
          end: original.length,
          text: formatted,
          selectionStart: position,
          selectionEnd: position
        })
      }
      emit('format', formatted)
      ElMessage.success('格式化完成，保存后生效')
      validateYaml(true)
    } catch (error: any) {
      ElMessage.error('格式化失败，原内容未修改：' + error.message)
    }
  }

  const copyContent = async () => {
    if (!localContent.value) {
      ElMessage.warning('没有内容可复制')
      return
    }

    try {
      await navigator.clipboard.writeText(localContent.value)
      ElMessage.success('📋 已复制到剪贴板')
    } catch {
      const textarea = document.createElement('textarea')
      textarea.value = localContent.value
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
      ElMessage.success('📋 已复制到剪贴板')
    }
  }

  const clearContent = () => {
    if (props.readonly) {
      ElMessage.warning('只读模式下无法清空内容')
      return
    }

    ElMessageBox.confirm('确定要清空所有内容吗？此操作不可恢复。', '⚠️ 清空确认', {
      confirmButtonText: '确定清空',
      cancelButtonText: '取消',
      type: 'warning',
      dangerouslyUseHTMLString: true,
      beforeClose: (action, instance, done) => {
        if (action === 'confirm') {
          localContent.value = ''
          errorLines.value = []
          errorCount.value = 0
          emit('update:modelValue', '')
          emit('change', '')
          ElMessage.success('✅ 已清空内容')
          done()
        } else {
          done()
        }
      }
    }).catch(() => {
      ElMessage.info('已取消清空')
    })
  }

  const exportYaml = () => {
    if (!localContent.value) {
      ElMessage.warning('没有内容可导出')
      return
    }

    const blob = new Blob([localContent.value], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = props.filename || 'export.yaml'
    a.click()
    URL.revokeObjectURL(url)

    ElMessage.success('💾 文件已导出')
  }

  const toggleLineNumbers = () => {
    showLineNumbersLocal.value = !showLineNumbersLocal.value
  }

  const toggleHighlight = () => {
    enableHighlight.value = !enableHighlight.value
  }

  const toggleTheme = () => {
    isDarkTheme.value = !isDarkTheme.value
  }

  const handleBeforeInput = (e: Event) => {
    if (props.readonly) {
      e.preventDefault()
      e.stopPropagation()
      return false
    }
  }

  const handleKeypress = (e: KeyboardEvent) => {
    if (props.readonly) {
      e.preventDefault()
      return false
    }
  }

  const handlePaste = (e: ClipboardEvent) => {
    if (props.readonly) {
      e.preventDefault()
      ElMessage.warning('只读模式下无法粘贴')
      return false
    }
  }

  const handleCut = (e: ClipboardEvent) => {
    if (props.readonly) {
      e.preventDefault()
      ElMessage.warning('只读模式下无法剪切')
      return false
    }
  }

  const handleInput = (e: Event) => {
    if (props.readonly) {
      e.preventDefault()
      e.stopPropagation()
      localContent.value = props.modelValue
      if (editorRef.value) {
        editorRef.value.value = props.modelValue
      }
      return false
    }

    if (isComposing.value) return

    emit('update:modelValue', localContent.value)
    emit('change', localContent.value)
    validateYaml(false)
    updateCursor()
  }

  const handleScroll = () => {
    if (!editorRef.value) return

    scrollTop.value = editorRef.value.scrollTop

    if (gutterRef.value) {
      gutterRef.value.scrollTop = scrollTop.value
    }

    if (highlightLayerRef.value) {
      highlightLayerRef.value.scrollTop = scrollTop.value
      highlightLayerRef.value.scrollLeft = editorRef.value.scrollLeft
    }
  }

  const handleKeydown = (e: KeyboardEvent) => {
    if (isComposing.value || e.isComposing) return
    if (props.readonly) {
      const allowedKeys = [
        'ArrowUp',
        'ArrowDown',
        'ArrowLeft',
        'ArrowRight',
        'Home',
        'End',
        'PageUp',
        'PageDown'
      ]
      const allowedCombos = (e.metaKey || e.ctrlKey) && ['c', 'a'].includes(e.key.toLowerCase())

      if (!allowedKeys.includes(e.key) && !allowedCombos) {
        e.preventDefault()
        ElMessage.warning('当前为只读模式，无法编辑')
        return
      }
    }

    if (e.key === 'Tab' && !props.readonly) {
      e.preventDefault()
      const textarea = e.target as HTMLTextAreaElement
      applyTextEdit(
        indentYamlSelection(
          localContent.value,
          textarea.selectionStart,
          textarea.selectionEnd,
          e.shiftKey,
          props.tabSize
        )
      )
      return
    }

    if (e.key === 'Enter' && !props.readonly && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault()
      const textarea = e.target as HTMLTextAreaElement
      applyTextEdit(
        yamlNewline(
          localContent.value,
          textarea.selectionStart,
          textarea.selectionEnd,
          props.tabSize
        )
      )
      return
    }

    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault()
      if (!props.readonly) {
        emit('save', localContent.value)
      } else {
        ElMessage.warning('只读模式下无法保存')
      }
    }

    if (
      ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'f') ||
      (e.altKey && e.shiftKey && e.key.toLowerCase() === 'f')
    ) {
      e.preventDefault()
      formatYaml()
    }
  }

  const updateCursor = () => {
    if (!editorRef.value) return

    const textarea = editorRef.value
    const text = localContent.value.substring(0, textarea.selectionStart)
    const lines = text.split('\n')
    currentLine.value = lines.length
    currentColumn.value = lines[lines.length - 1].length + 1
  }

  const gotoLine = (line: number, options: { focus?: boolean } = {}) => {
    if (!editorRef.value) return

    const lines = localContent.value.split('\n')
    let position = 0

    for (let i = 0; i < line - 1 && i < lines.length; i++) {
      position += lines[i].length + 1
    }

    if (options.focus !== false) {
      editorRef.value.setSelectionRange(position, position)
      editorRef.value.focus()
    }
    editorRef.value.scrollTop = (line - 1) * 21
  }

  // Expose viewport navigation so combined YAML panels can focus a file section.
  defineExpose({ gotoLine, formatYaml })

  const showGotoLineDialog = () => {
    ElMessageBox.prompt('请输入要跳转到的行号', '跳转到行', {
      confirmButtonText: '跳转',
      cancelButtonText: '取消',
      inputPattern: /^\d+$/,
      inputErrorMessage: '请输入有效的行号'
    })
      .then(({ value }) => {
        const lineNumber = parseInt(value)
        const maxLine = localContent.value.split('\n').length
        if (lineNumber >= 1 && lineNumber <= maxLine) {
          gotoLine(lineNumber)
        } else {
          ElMessage.warning(`行号必须在 1 到 ${maxLine} 之间`)
        }
      })
      .catch(() => {})
  }

  const handleFocus = () => {
    focused.value = true
  }

  const handleBlur = () => {
    focused.value = false
    validateYaml(true)
  }

  const onCompositionStart = () => {
    isComposing.value = true
  }

  const onCompositionEnd = () => {
    isComposing.value = false
    handleInput(new Event('input'))
  }

  const handleClose = () => {
    if (props.readonly) {
      emit('update:modelValue', '')
      return
    }

    ElMessageBox.confirm('确定要关闭编辑器吗？未保存的更改将会丢失。', '关闭确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
      .then(() => {
        emit('update:modelValue', '')
      })
      .catch(() => {})
  }

  const handleMinimize = () => {
    ElMessage.info('最小化功能需要在容器中实现')
  }

  const handleMaximize = () => {
    ElMessage.info('最大化功能需要在容器中实现')
  }

  watch(
    () => props.modelValue,
    (newValue) => {
      if (newValue !== localContent.value) {
        localContent.value = newValue
        validateYaml(true)
      }
    },
    { immediate: true }
  )

  watch(
    () => props.showLineNumbers,
    (newValue) => {
      showLineNumbersLocal.value = newValue
    }
  )

  watch(
    () => props.theme,
    (newValue) => {
      isDarkTheme.value = newValue === 'dark'
    }
  )

  watch(localContent, (newValue) => {
    if (props.readonly && newValue !== props.modelValue) {
      nextTick(() => {
        localContent.value = props.modelValue
      })
    }
  })

  watch(
    () => props.readonly,
    (newValue) => {
      if (newValue && editorRef.value) {
        editorRef.value.blur()
      }
    }
  )

  onMounted(() => {
    validateYaml(true)
    updateCursor()

    if (props.readonly) {
      localContent.value = props.modelValue
      if (editorRef.value) {
        editorRef.value.value = props.modelValue
        editorRef.value.readOnly = true
        editorRef.value.disabled = true
      }
    } else if (props.autoFormat && localContent.value) {
      formatYaml()
    }
  })

  onUnmounted(() => {
    if (validateTimer.value) {
      clearTimeout(validateTimer.value)
    }
  })
</script>

<style lang="scss" scoped>
  .yaml-editor-pro {
    display: flex;
    flex-direction: column;
    border-radius: 4px;
    overflow: hidden;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
    background: #1e1e1e;
    position: relative;

    .editor-titlebar {
      display: flex;
      align-items: center;
      height: 40px;
      padding: 0 16px;
      background: #3c3c3c;
      border-bottom: 1px solid #191919;
      user-select: none;

      .window-controls {
        display: flex;
        gap: 8px;

        .control {
          width: 12px;
          height: 12px;
          border-radius: 50%;
          cursor: pointer;
          transition: all 0.3s ease;
          position: relative;

          svg {
            display: none;
            width: 8px;
            height: 8px;
            position: absolute;
            top: 2px;
            left: 2px;
          }

          &:hover svg {
            display: block;
          }

          &.close {
            background: #ff5f57;

            &:hover {
              background: #ff3b30;
              transform: scale(1.1);
            }
          }

          &.minimize {
            background: #ffbd2e;

            &:hover {
              background: #ffac00;
              transform: scale(1.1);
            }
          }

          &.maximize {
            background: #28ca42;

            &:hover {
              background: #00d600;
              transform: scale(1.1);
            }
          }
        }
      }

      .title-text {
        flex: 1;
        text-align: center;
        font-size: 13px;
        color: #cccccc;
        font-weight: 500;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 6px;

        .readonly-badge {
          display: inline-flex;
          align-items: center;
          gap: 4px;
          padding: 2px 8px;
          background: rgba(255, 152, 0, 0.2);
          color: #ff9800;
          border-radius: 4px;
          font-size: 11px;
          margin-left: 8px;
          font-weight: 600;
        }
      }

      .title-status {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
        color: #969696;

        .status-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
          animation: pulse 2s infinite;

          &.normal {
            background: #969696;
          }

          &.success {
            background: #28ca42;
            box-shadow: 0 0 10px rgba(40, 202, 66, 0.5);
          }

          &.error {
            background: #ff5f57;
            animation: pulse-error 1s infinite;
            box-shadow: 0 0 10px rgba(255, 95, 87, 0.5);
          }
        }
      }
    }

    &.light-editor {
      background: #ffffff;
      border: 1px solid #dbe3ee;

      .editor-titlebar {
        background: #ffffff;
        border-bottom-color: #e5e7eb;

        .title-text {
          color: #334155;
        }

        .title-status {
          color: #64748b;
        }
      }

      .editor-toolbar {
        background: #f8fafc;
        border-bottom-color: #e5e7eb;

        .tool-button {
          color: #475569;

          &:hover:not(:disabled) {
            background: #e2e8f0;
            color: #0f172a;
          }
        }
      }
    }

    .editor-toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      height: 42px;
      padding: 0 16px;
      background: #2d2d30;
      border-bottom: 1px solid #191919;

      .toolbar-group {
        display: flex;
        align-items: center;
        gap: 4px;
      }

      .tool-button {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 6px 12px;
        font-size: 12px;
        color: #cccccc;
        background: transparent;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        transition: background 0.2s;

        &:hover:not(:disabled) {
          background: rgba(255, 255, 255, 0.1);
          color: #ffffff;
        }

        &:disabled,
        &.disabled {
          opacity: 0.4;
          cursor: not-allowed;
          pointer-events: none;
        }

        .el-icon {
          font-size: 14px;
        }
      }

      .toolbar-divider {
        width: 1px;
        height: 24px;
        background: linear-gradient(to bottom, transparent, #464647, transparent);
        margin: 0 8px;
      }
    }

    .editor-body {
      flex: 1;
      display: flex;
      overflow: hidden;
      background: #1e1e1e;
      position: relative;
      min-height: 0;
      max-height: 100%;

      &.light-theme {
        background: linear-gradient(135deg, #ffffff 0%, #f5f5f5 100%);

        .gutter {
          background: linear-gradient(to right, #f8f8f8, #f3f3f3);
          border-right-color: #e8e8e8;

          .line-number {
            color: #858585;

            &.current-line {
              background: linear-gradient(90deg, transparent, rgba(0, 122, 204, 0.1));
              color: #0969da;
              font-weight: bold;
            }

            &.has-error {
              color: #cf222e;

              .error-marker {
                color: #cf222e;
              }
            }
          }
        }

        .editor-wrapper {
          .editor-textarea {
            &.with-highlight {
              color: transparent !important;
            }

            &:not(.with-highlight) {
              color: #000000 !important;
            }

            caret-color: #0969da;

            &::selection {
              background: rgba(9, 105, 218, 0.3);
            }

            &::placeholder {
              color: #999;
            }

            &::-webkit-scrollbar-track {
              background: #f5f5f5;
            }

            &::-webkit-scrollbar-thumb {
              background: #c0c0c0;
              border: 2px solid #f5f5f5;

              &:hover {
                background: #a0a0a0;
              }

              &:active {
                background: #808080;
              }
            }

            scrollbar-color: #c0c0c0 #f5f5f5;
          }
        }
      }

      .gutter {
        width: 65px;
        background: #1e1e1e;
        border-right: 1px solid #2d2d30;
        overflow-y: auto;
        overflow-x: hidden;
        padding: 0;

        &::-webkit-scrollbar {
          width: 0;
          height: 0;
        }

        scrollbar-width: none;

        .gutter-line {
          display: flex;
          align-items: center;
          height: 21px;
          position: relative;

          .line-number {
            flex: 1;
            display: flex;
            align-items: center;
            justify-content: flex-end;
            padding-right: 12px;
            padding-left: 8px;
            color: #858585;
            font-size: 12px;
            font-family: v-bind(fontFamily);
            cursor: pointer;
            position: relative;
            transition: all 0.2s;
            height: 21px;

            &:hover {
              color: #cccccc;
              background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.05));
            }

            &.current-line {
              background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.08));
              color: #cccccc;
              font-weight: bold;
            }

            &.has-error {
              color: #f48771;
              background: rgba(255, 95, 87, 0.1);

              .error-marker {
                position: absolute;
                left: 6px;
                top: 50%;
                transform: translateY(-50%);
                display: inline-flex;
                align-items: center;
                justify-content: center;

                svg {
                  filter: drop-shadow(0 0 3px rgba(255, 95, 87, 0.8));
                }
              }
            }
          }
        }
      }

      .editor-wrapper {
        flex: 1;
        position: relative;
        overflow: visible !important;
        min-height: 0;

        &.no-gutter {
          .editor-textarea,
          .highlight-layer {
            padding-left: 20px;
          }
        }

        .highlight-layer {
          position: absolute;
          top: 0;
          left: 0;
          right: 14px;
          bottom: 0;
          padding: 0 20px 0 12px;
          line-height: 21px;
          overflow: hidden;
          white-space: pre;
          pointer-events: none;
          color: transparent;
          z-index: 1;

          code {
            font-family: inherit;
            font-size: inherit;
          }

          .line {
            min-height: 21px;
            display: inline-block;
            width: 100%;
          }
        }

        .editor-textarea {
          width: 100%;
          height: 100%;
          padding: 0 20px 0 12px;
          line-height: 21px;
          background: transparent;
          border: none;
          outline: none;
          resize: none;
          overflow: auto !important;
          caret-color: #aeafad;
          font-family: v-bind(fontFamily);
          font-size: v-bind(fontSize + 'px');
          tab-size: v-bind(tabSize);
          z-index: 2;
          position: relative;
          white-space: pre;

          &.with-highlight {
            color: transparent !important;

            &::selection {
              background: rgba(51, 153, 255, 0.4) !important;
              color: transparent !important;
            }

            &::-moz-selection {
              background: rgba(51, 153, 255, 0.4) !important;
              color: transparent !important;
            }
          }

          &:not(.with-highlight) {
            color: #ffffff;

            &::selection {
              background: #264f78;
            }
          }

          &.readonly,
          &[readonly],
          &[disabled] {
            cursor: default !important;
            user-select: text !important;
            background: rgba(0, 0, 0, 0.1);
            opacity: 0.9;
          }

          &.word-wrap {
            word-wrap: break-word;
            white-space: pre-wrap;
          }

          &::placeholder {
            color: #5a5a5a;
            opacity: 0.8;
          }

          &::-webkit-scrollbar {
            width: 12px !important;
            height: 12px !important;
          }

          &::-webkit-scrollbar-track {
            background: #2d2d30;
            border-radius: 0;
          }

          &::-webkit-scrollbar-thumb {
            background: #424242;
            border-radius: 6px;
            border: 2px solid #2d2d30;

            &:hover {
              background: #525252;
            }

            &:active {
              background: #626262;
            }
          }

          &::-webkit-scrollbar-corner {
            background: #2d2d30;
          }

          scrollbar-width: thin;
          scrollbar-color: #424242 #2d2d30;
        }

        .error-wave-svg {
          position: absolute;
          top: 0;
          left: 0;
          right: 14px;
          height: 2000px;
          pointer-events: none;
          z-index: 3;
          padding: 0 20px 0 12px;
        }

        .readonly-overlay {
          position: absolute;
          top: 0;
          left: 0;
          right: 14px;
          bottom: 0;
          background: rgba(0, 0, 0, 0.05);
          display: flex;
          align-items: center;
          justify-content: center;
          pointer-events: none;
          z-index: 10;

          .readonly-icon {
            color: rgba(255, 255, 255, 0.1);
          }
        }
      }
    }

    .editor-statusbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      height: 28px;
      padding: 0 16px;
      background: #007acc;
      color: white;
      font-size: 12px;
      user-select: none;

      .status-left,
      .status-right {
        display: flex;
        align-items: center;
        gap: 16px;
      }

      .status-item {
        display: flex;
        align-items: center;
        gap: 4px;
        transition: color 0.2s;

        &.clickable {
          cursor: pointer;
          padding: 2px 6px;
          border-radius: 4px;
        }

        &.error {
          color: #ffcccc;
          animation: pulse 2s infinite;
        }

        &.success {
          color: #90ee90;
        }

        .el-icon {
          font-size: 14px;
        }
      }
    }

    @keyframes pulse {
      0%,
      100% {
        opacity: 1;
      }
      50% {
        opacity: 0.7;
      }
    }

    @keyframes pulse-error {
      0%,
      100% {
        transform: scale(1);
        opacity: 1;
      }
      50% {
        transform: scale(1.2);
        opacity: 0.8;
      }
    }
  }
</style>

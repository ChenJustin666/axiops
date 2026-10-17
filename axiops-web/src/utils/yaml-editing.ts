export interface YamlTextEdit {
  start: number
  end: number
  text: string
  selectionStart: number
  selectionEnd: number
}

// Validate the YAML structure before runtime values are known. Keep offsets so
// parser diagnostics still point to the original editor line and column.
export function yamlSyntaxSource(source: string): string {
  return source.replace(/\{\{\s*\.?[A-Za-z][A-Za-z0-9_]*\s*\}\}/g, (value) =>
    'A'.repeat(value.length)
  )
}

export function indentYamlSelection(
  source: string,
  start: number,
  end: number,
  outdent = false,
  size = 2
): YamlTextEdit {
  const lineStart = source.slice(0, start).lastIndexOf('\n') + 1
  if (start === end && !outdent) {
    const text = ' '.repeat(size - ((start - lineStart) % size))
    return {
      start,
      end,
      text,
      selectionStart: start + text.length,
      selectionEnd: start + text.length
    }
  }
  const last = end > start && source[end - 1] === '\n' ? end - 1 : end
  const nextLine = source.indexOf('\n', last)
  const lineEnd = nextLine < 0 ? source.length : nextLine
  const block = source.slice(lineStart, lineEnd)
  const lines = block.split('\n')
  const text = lines
    .map((line) =>
      outdent ? line.replace(new RegExp(`^ {1,${size}}`), '') : ' '.repeat(size) + line
    )
    .join('\n')
  if (start === end) {
    const position = Math.max(lineStart, start - (block.length - text.length))
    return {
      start: lineStart,
      end: lineEnd,
      text,
      selectionStart: position,
      selectionEnd: position
    }
  }
  return {
    start: lineStart,
    end: lineEnd,
    text,
    selectionStart: lineStart,
    selectionEnd: lineStart + text.length
  }
}

export function yamlNewline(source: string, start: number, end: number, size = 2): YamlTextEdit {
  const line = source.slice(source.slice(0, start).lastIndexOf('\n') + 1, start)
  const indent = line.match(/^ */)?.[0] || ''
  const nested = /:\s*(?:[>|][+-]?[1-9]?)?\s*(?:#.*)?$/.test(line)
  const text = '\n' + indent + (nested ? ' '.repeat(size) : '')
  return {
    start,
    end,
    text,
    selectionStart: start + text.length,
    selectionEnd: start + text.length
  }
}

export async function formatYamlSource(source: string, tabWidth = 2): Promise<string> {
  // Protect AxiOps placeholders from being parsed as YAML flow mappings.
  let prefix = 'AXIOPSFORMATPLACEHOLDER'
  while (source.includes(prefix)) prefix += 'X'
  const placeholders: string[] = []
  const protectedSource = source.replace(/\{\{\s*\.?[A-Za-z][A-Za-z0-9_]*\s*\}\}/g, (value) => {
    const marker = `${prefix}${placeholders.length}END`
    placeholders.push(value)
    return marker
  })
  if (protectedSource.includes('{{') || protectedSource.includes('}}')) {
    throw new Error('仅支持普通 YAML 和 AxiOps 变量；Helm 模板请在 Chart 中编辑')
  }
  const [{ format }, plugin] = await Promise.all([
    import('prettier/standalone'),
    import('prettier/plugins/yaml')
  ])
  let formatted = await format(protectedSource, {
    parser: 'yaml',
    plugins: [plugin],
    tabWidth,
    printWidth: 100
  })
  placeholders.forEach((value, index) => {
    formatted = formatted.replaceAll(`${prefix}${index}END`, value)
  })
  return formatted
}

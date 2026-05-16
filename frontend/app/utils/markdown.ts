function isFenceBoundary(line: string) {
  return /^\s*(```|~~~)/.test(line)
}

function isBlankLine(line: string) {
  return line.trim().length === 0
}

function isMarkdownTableRow(line: string) {
  const trimmed = line.trim()
  if (!trimmed.startsWith('|') || !trimmed.endsWith('|')) return false

  return trimmed.slice(1, -1).includes('|')
}

function isMarkdownTableSeparator(line: string) {
  const cells = line
    .trim()
    .replace(/^\|/, '')
    .replace(/\|$/, '')
    .split('|')
    .map(cell => cell.trim())

  return cells.length > 1 && cells.every(cell => /^:?-{3,}:?$/.test(cell))
}

export function normalizeLooseMarkdownTables(markdown: string) {
  const lines = markdown.split('\n')
  const normalized: string[] = []
  let inFence = false

  for (let index = 0; index < lines.length;) {
    const line = lines[index] || ''

    if (isFenceBoundary(line)) {
      inFence = !inFence
      normalized.push(line)
      index += 1
      continue
    }

    if (inFence || !isMarkdownTableRow(line)) {
      normalized.push(line)
      index += 1
      continue
    }

    const originalBlock: string[] = []
    const tableRows: string[] = []
    let cursor = index

    while (cursor < lines.length) {
      const current = lines[cursor] || ''

      if (isMarkdownTableRow(current)) {
        originalBlock.push(current)
        tableRows.push(current)
        cursor += 1
        continue
      }

      if (isBlankLine(current) && cursor + 1 < lines.length && isMarkdownTableRow(lines[cursor + 1] || '')) {
        originalBlock.push(current)
        cursor += 1
        continue
      }

      break
    }

    if (tableRows.length >= 2 && isMarkdownTableSeparator(tableRows[1] || '')) {
      normalized.push(...tableRows)
    } else {
      normalized.push(...originalBlock)
    }

    index = cursor
  }

  return normalized.join('\n')
}

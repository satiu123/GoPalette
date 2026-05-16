const CJK_CHARS_PER_MINUTE = 500
const ENGLISH_WORDS_PER_MINUTE = 225

export function estimateReadingMinutes(content = '', fallback = '') {
  const text = (content.trim() || fallback.trim())
  if (!text) return 1

  const cjkChars = Array.from(text.matchAll(/[\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff]/g)).length
  const latinWords = Array.from(text.matchAll(/[A-Za-z0-9]+(?:[-'][A-Za-z0-9]+)*/g)).length
  const minutes = (cjkChars / CJK_CHARS_PER_MINUTE) + (latinWords / ENGLISH_WORDS_PER_MINUTE)
  return Math.max(1, Math.ceil(minutes))
}

export function resolveReadingMinutes(input: unknown, content = '', fallback = '') {
  const value = Number(input)
  if (Number.isFinite(value) && value > 0) {
    return Math.max(1, Math.ceil(value))
  }
  return estimateReadingMinutes(content, fallback)
}

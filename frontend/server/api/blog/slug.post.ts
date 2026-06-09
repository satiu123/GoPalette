import { generateText } from 'ai'
import { createDeepSeek } from '@ai-sdk/deepseek'
import type { DeepSeekLanguageModelOptions } from '@ai-sdk/deepseek'

function toText(input: unknown) {
  return typeof input === 'string' ? input.trim() : ''
}

function toErrorMessage(error: unknown) {
  if (error instanceof Error) return error.message
  return String(error || 'Unknown error')
}

function normalizeSlug(input: string) {
  return input
    .toLowerCase()
    .replace(/^["'`“”‘’\s]+|["'`“”‘’\s]+$/g, '')
    .replace(/&/g, ' and ')
    .replace(/[^a-z0-9\s-]/g, ' ')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 80)
    .replace(/-+$/g, '')
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env || {}
  const deepseekApiKey = config.deepseekApiKey || env.DEEPSEEK_API_KEY || ''
  const body = await readBody<Record<string, unknown>>(event)
  const title = toText(body.title)
  const content = toText(body.content)
  const startedAt = Date.now()
  const requestId = Math.random().toString(36).slice(2, 10)
  const model = 'deepseek-v4-flash'

  if (!title) {
    throw createError({ statusCode: 400, message: '标题不能为空' })
  }

  if (!deepseekApiKey) {
    throw createError({
      statusCode: 500,
      message: 'AI 服务未配置，请联系管理员'
    })
  }

  const deepseek = createDeepSeek({ apiKey: deepseekApiKey })
  const prompt = [
    `Title: ${title}`,
    content ? `Article excerpt:\n${content.slice(0, 2000)}` : ''
  ].filter(Boolean).join('\n\n')

  console.info('[ai:slug] start', {
    requestId,
    model,
    titleChars: title.length,
    contentChars: content.length
  })

  try {
    const result = await generateText({
      model: deepseek(model),
      system: [
        'You generate URL slugs for technical blog posts.',
        'Return exactly one slug.',
        'Use concise English words only.',
        'Allowed characters: lowercase a-z, numbers, and hyphens.',
        'Do not use Chinese characters, punctuation, quotes, markdown, or explanations.',
        'Keep it under 80 characters.'
      ].join('\n'),
      prompt,
      maxOutputTokens: 80,
      providerOptions: {
        deepseek: {
          thinking: { type: 'disabled' }
        } satisfies DeepSeekLanguageModelOptions
      }
    })

    const slug = normalizeSlug(result.text)
    if (!slug) {
      throw createError({ statusCode: 502, message: '未能生成有效的文章路径，请手动填写' })
    }

    console.info('[ai:slug] finish', {
      requestId,
      elapsedMs: Date.now() - startedAt,
      inputTokens: result.totalUsage.inputTokens,
      outputTokens: result.totalUsage.outputTokens,
      totalTokens: result.totalUsage.totalTokens
    })

    return { slug }
  } catch (error) {
    console.error('[ai:slug] failed', {
      requestId,
      elapsedMs: Date.now() - startedAt,
      error: toErrorMessage(error)
    })
    throw error
  }
})

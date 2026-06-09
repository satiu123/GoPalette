import { generateText } from 'ai'
import { createDeepSeek } from '@ai-sdk/deepseek'
import type { DeepSeekLanguageModelOptions } from '@ai-sdk/deepseek'

function toErrorMessage(error: unknown) {
  if (error instanceof Error) return error.message
  return String(error || 'Unknown error')
}

function toText(input: unknown) {
  return typeof input === 'string' ? input.trim() : ''
}

function normalizeSummary(input: string) {
  return input
    .replace(/^["'“”‘’\s]+|["'“”‘’\s]+$/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 140)
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

  if (!content) {
    console.warn('[ai:summary] rejected', {
      requestId,
      reason: 'missing_content'
    })
    throw createError({ statusCode: 400, message: '正文不能为空' })
  }

  if (!deepseekApiKey) {
    console.error('[ai:summary] rejected', {
      requestId,
      reason: 'missing_deepseek_api_key'
    })
    throw createError({
      statusCode: 500,
      message: 'AI 服务未配置，请联系管理员'
    })
  }

  const deepseek = createDeepSeek({
    apiKey: deepseekApiKey
  })

  const prompt = [
    title ? `标题：${title}` : '',
    `正文：\n${content.slice(0, 12000)}`
  ].filter(Boolean).join('\n\n')

  console.info('[ai:summary] start', {
    requestId,
    model,
    titleChars: title.length,
    contentChars: content.length
  })

  try {
    const result = await generateText({
      model: deepseek(model),
      system: [
        '你是博客编辑助手，请根据文章标题和正文生成发布用摘要。',
        '要求：只输出摘要正文，不要标题、解释、引号或列表符号。',
        '摘要必须使用中文，保留核心观点，适合展示在文章列表中。',
        '长度不超过 140 个中文字符。'
      ].join('\n'),
      prompt,
      maxOutputTokens: 220,
      providerOptions: {
        deepseek: {
          thinking: { type: 'disabled' }
        } satisfies DeepSeekLanguageModelOptions
      }
    })

    const summary = normalizeSummary(result.text)

    if (!summary) {
      console.warn('[ai:summary] empty_result', {
        requestId,
        elapsedMs: Date.now() - startedAt
      })
      throw createError({ statusCode: 502, message: '未能生成摘要，请手动填写' })
    }

    console.info('[ai:summary] finish', {
      requestId,
      elapsedMs: Date.now() - startedAt,
      inputTokens: result.totalUsage.inputTokens,
      outputTokens: result.totalUsage.outputTokens,
      totalTokens: result.totalUsage.totalTokens
    })

    return { summary }
  } catch (error) {
    console.error('[ai:summary] failed', {
      requestId,
      elapsedMs: Date.now() - startedAt,
      error: toErrorMessage(error)
    })
    throw error
  }
})

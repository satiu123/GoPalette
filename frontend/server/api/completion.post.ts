import { streamText } from 'ai'
import { createDeepSeek } from '@ai-sdk/deepseek'
import type { DeepSeekLanguageModelOptions } from '@ai-sdk/deepseek'

function toErrorMessage(error: unknown) {
  if (error instanceof Error) return error.message
  return String(error || 'Unknown error')
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env || {}
  const deepseekApiKey = config.deepseekApiKey || env.DEEPSEEK_API_KEY || ''
  const { prompt, mode, language } = await readBody(event)
  const startedAt = Date.now()
  const requestId = Math.random().toString(36).slice(2, 10)
  const model = 'deepseek-v4-flash'

  if (!prompt) {
    console.warn('[ai:completion] rejected', {
      requestId,
      reason: 'missing_prompt'
    })
    throw createError({ statusCode: 400, message: 'Prompt is required' })
  }
  if (!deepseekApiKey) {
    console.error('[ai:completion] rejected', {
      requestId,
      reason: 'missing_deepseek_api_key'
    })
    throw createError({
      statusCode: 500,
      message: 'DeepSeek API key is not configured'
    })
  }

  let system: string
  let maxOutputTokens: number

  const preserveMarkdown = 'IMPORTANT: Preserve all markdown formatting (bold, italic, links, etc.) exactly as in the original.'

  switch (mode) {
    case 'fix':
      system = `You are a writing assistant. Fix all spelling and grammar errors in the given text. ${preserveMarkdown} Only output the corrected text, nothing else.`
      maxOutputTokens = 800
      break
    case 'extend':
      system = `You are a writing assistant. Extend the given text with more details, examples, and explanations while maintaining the same style. ${preserveMarkdown} Only output the extended text, nothing else.`
      maxOutputTokens = 1400
      break
    case 'reduce':
      system = `You are a writing assistant. Make the given text more concise by removing unnecessary words while keeping the meaning. ${preserveMarkdown} Only output the reduced text, nothing else.`
      maxOutputTokens = 700
      break
    case 'simplify':
      system = `You are a writing assistant. Simplify the given text to make it easier to understand, using simpler words and shorter sentences. ${preserveMarkdown} Only output the simplified text, nothing else.`
      maxOutputTokens = 900
      break
    case 'summarize':
      system = 'You are a writing assistant. Summarize the given text concisely while keeping the key points. Only output the summary, nothing else.'
      maxOutputTokens = 600
      break
    case 'translate':
      system = `You are a writing assistant. Translate the given text to ${language || 'English'}. ${preserveMarkdown} Only output the translated text, nothing else.`
      maxOutputTokens = 1200
      break
    case 'continue':
    default:
      system = `You are a writing assistant helping continue a draft.
CRITICAL RULES:
- Output ONLY the NEW text that should come AFTER the cursor
- NEVER repeat any words from the end of the user's text
- Keep completions concise: 1-3 sentences or one short paragraph
- Match the tone and style of the existing text
- Preserve the user's language and markdown style
- Do not add explanations, labels, or quotation marks
- ${preserveMarkdown}`
      maxOutputTokens = 180
      break
  }

  const deepseek = createDeepSeek({
    apiKey: deepseekApiKey
  })

  console.info('[ai:completion] start', {
    requestId,
    model,
    mode: mode || 'continue',
    language: language || '',
    promptChars: String(prompt).length,
    maxOutputTokens
  })

  let firstChunkLogged = false
  let firstChunkTypeLogged = false
  let textChunkCount = 0

  try {
    return streamText({
      model: deepseek(model),
      system,
      prompt,
      maxOutputTokens,
      providerOptions: {
        deepseek: {
          thinking: { type: 'disabled' }
        } satisfies DeepSeekLanguageModelOptions
      },
      onChunk: ({ chunk }) => {
        if (!firstChunkTypeLogged) {
          firstChunkTypeLogged = true
          console.info('[ai:completion] first_chunk_type', {
            requestId,
            type: chunk.type,
            elapsedMs: Date.now() - startedAt
          })
        }
        if (chunk.type !== 'text-delta') return
        textChunkCount += 1
        if (firstChunkLogged) return
        firstChunkLogged = true
        console.info('[ai:completion] first_chunk', {
          requestId,
          elapsedMs: Date.now() - startedAt
        })
      },
      onFinish: ({ finishReason, totalUsage }) => {
        console.info('[ai:completion] finish', {
          requestId,
          finishReason,
          elapsedMs: Date.now() - startedAt,
          inputTokens: totalUsage.inputTokens,
          outputTokens: totalUsage.outputTokens,
          totalTokens: totalUsage.totalTokens
        })
        if (textChunkCount === 0) {
          console.warn('[ai:completion] no_text_chunks', {
            requestId,
            model,
            finishReason
          })
        }
      },
      onError: ({ error }) => {
        console.error('[ai:completion] stream_error', {
          requestId,
          elapsedMs: Date.now() - startedAt,
          error: toErrorMessage(error)
        })
      }
    }).toTextStreamResponse()
  } catch (error) {
    console.error('[ai:completion] failed', {
      requestId,
      elapsedMs: Date.now() - startedAt,
      error: toErrorMessage(error)
    })
    throw error
  }
})

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
  const { prompt, mode, language, source } = await readBody(event)
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

  const markdownEditorRules = [
    'You are editing content for a Markdown rich-text editor.',
    'Output valid GitHub-Flavored Markdown only, with no wrapper fences unless the content itself is a code block.',
    'Use Markdown structure when it improves the result: headings, lists, blockquotes, bold/italic text, inline code, fenced code blocks with language tags, tables, and descriptive links.',
    'Preserve existing Markdown semantics, links, tables, code blocks, and formatting unless the requested edit requires changing them.',
    'When adding links, write descriptive linked text like [label](https://example.com); do not expose raw URLs unless they are the content.',
    'Do not add explanations, labels, prefaces, or quotation marks around the answer.'
  ].join('\n')

  const transformInputFormat = [
    'The user input is Markdown selected from the editor.',
    'Return the complete replacement Markdown for that selection.'
  ].join('\n')

  switch (mode) {
    case 'fix':
      system = `${markdownEditorRules}
${transformInputFormat}
Task: Fix spelling, grammar, punctuation, and awkward wording. Keep the original structure and formatting unless a small Markdown cleanup is needed.`
      maxOutputTokens = 800
      break
    case 'extend':
      system = `${markdownEditorRules}
${transformInputFormat}
Task: Extend the selection with useful detail, examples, explanation, or supporting material while matching the original language, tone, and structure. Add Markdown-rich elements such as lists, tables, links, or fenced code blocks when they genuinely make the expanded content clearer.`
      maxOutputTokens = 2200
      break
    case 'reduce':
      system = `${markdownEditorRules}
${transformInputFormat}
Task: Make the selection more concise while preserving the essential meaning and useful Markdown structure. Keep code, tables, and links intact unless they are redundant.`
      maxOutputTokens = 700
      break
    case 'simplify':
      system = `${markdownEditorRules}
${transformInputFormat}
Task: Simplify the selection so it is easier to understand. Prefer clearer wording, shorter paragraphs, and helpful Markdown structure. Keep technical accuracy, code blocks, tables, and links when they help comprehension.`
      maxOutputTokens = 1200
      break
    case 'summarize':
      system = `${markdownEditorRules}
${transformInputFormat}
Task: Summarize the selection concisely while keeping the key points. Use bullets, a compact table, or short sections when that is the clearest format.`
      maxOutputTokens = 900
      break
    case 'translate':
      system = `${markdownEditorRules}
${transformInputFormat}
Task: Translate the selection to ${language || 'English'}. Preserve Markdown structure, code blocks, tables, links, and technical identifiers. Translate human-readable prose and link labels when appropriate, but do not translate code.`
      maxOutputTokens = 1800
      break
    case 'continue':
    default:
      if (source === 'auto') {
        system = `You are an inline autocomplete engine for a Markdown editor.
Output only the missing text after the cursor.
Keep it to one short line, usually 4-18 words.
Match the user's language and tone.
Do not repeat the text already provided.
Do not use headings, lists, tables, code fences, labels, quotes, or explanations.`
        maxOutputTokens = 48
        break
      }

      system = `${markdownEditorRules}
You are helping continue a Markdown draft from the cursor position.
CRITICAL RULES:
- Output only the new Markdown that should come after the cursor.
- Never repeat the end of the user's draft.
- Match the language, tone, topic, and current Markdown structure.
- For inline prose, keep it concise: 1-3 sentences or one short paragraph.
- If the context naturally calls for it, continue with Markdown-rich content such as a list item, table row, fenced code block, or descriptive link.`
      maxOutputTokens = 500
      break
  }

  const deepseek = createDeepSeek({
    apiKey: deepseekApiKey
  })

  console.info('[ai:completion] start', {
    requestId,
    model,
    mode: mode || 'continue',
    source: source || '',
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

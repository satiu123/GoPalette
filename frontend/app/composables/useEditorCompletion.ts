import { useCompletion } from '@ai-sdk/vue'
import type { Editor } from '@tiptap/vue-3'
import { Completion } from '~/components/editor/CompletionExtension'
import type { CompletionStorage } from '~/components/editor/CompletionExtension'

type CompletionMode = 'continue' | 'fix' | 'extend' | 'reduce' | 'simplify' | 'summarize' | 'translate'
type TransformMode = Exclude<CompletionMode, 'continue'>
type InlineContinueSource = 'auto' | 'manual' | 'toolbar-cursor' | 'toolbar-selection' | 'shortcut'

interface AiCandidate {
  id: number
  mode: TransformMode
  language?: string
  text: string
  stopped?: boolean
}

interface UseEditorCompletionOptions {
  api?: string
}

const transformModes: CompletionMode[] = ['fix', 'extend', 'reduce', 'simplify', 'summarize', 'translate']

export function useEditorCompletion(editorRef: Ref<{ editor: Editor | undefined } | null | undefined>, options: UseEditorCompletionOptions = {}) {
  // CSRF protection
  const { csrf, headerName } = useCsrf()

  // State for direct insertion/transform mode
  const insertState = ref<{
    pos: number
    deleteRange?: { from: number, to: number }
  }>()
  const mode = ref<CompletionMode>('continue')
  const language = ref<string>()
  const activeRequestId = ref(0)
  const inlineContinueSource = ref<InlineContinueSource>('manual')
  const candidateSeq = ref(0)
  const candidates = ref<AiCandidate[]>([])
  const activeCandidateIndex = ref(-1)
  const reviewState = ref<{
    pos: number
    deleteRange: { from: number, to: number }
    prompt: string
    mode: TransformMode
    language?: string
  }>()

  const isTransformMode = computed(() => transformModes.includes(mode.value))
  const activeCandidate = computed(() => candidates.value[activeCandidateIndex.value])
  const previewText = computed(() => {
    if (isTransformMode.value && isLoading.value && completion.value) {
      return completion.value
    }
    return activeCandidate.value?.text || ''
  })
  const isReviewOpen = computed(() => Boolean(reviewState.value && (isLoading.value || previewText.value || candidates.value.length)))

  // Helper to get completion storage
  function getCompletionStorage() {
    const storage = editorRef.value?.editor?.storage as Record<string, CompletionStorage> | undefined
    return storage?.completion
  }

  function shouldPrefixInlineSpace(textBefore: string, text: string) {
    if (!textBefore || /\s/.test(textBefore) || /^\s/.test(text)) return false
    return !/^(#{1,6}\s|[-*+]\s|\d+\.\s|>\s|```|~~~|\|)/.test(text)
  }

  function normalizeInlineSuggestion(text: string, source: InlineContinueSource) {
    if (source !== 'auto') return text

    return text
      .replace(/[\r\n]+[\s\S]*$/u, '')
      .replace(/\s+/g, ' ')
      .trimEnd()
  }

  function beginRequest(nextMode: CompletionMode) {
    activeRequestId.value += 1
    setCompletion('')
    mode.value = nextMode
    getCompletionStorage()?.clearSuggestion()
    return activeRequestId.value
  }

  function clearActiveRequest(options: { clearReview?: boolean } = {}) {
    insertState.value = undefined
    getCompletionStorage()?.clearSuggestion()
    if (options.clearReview) {
      reviewState.value = undefined
      candidates.value = []
      activeCandidateIndex.value = -1
    }
  }

  function addCandidate(text: string, stopped = false) {
    if (!reviewState.value) return
    const normalized = text.trim()
    if (!normalized) return

    candidateSeq.value += 1
    candidates.value.push({
      id: candidateSeq.value,
      mode: reviewState.value.mode,
      language: reviewState.value.language,
      text: normalized,
      stopped
    })
    activeCandidateIndex.value = candidates.value.length - 1
  }

  function stopGeneration() {
    activeRequestId.value += 1
    stop()
    if (isTransformMode.value && reviewState.value) {
      addCandidate(completion.value, true)
      setCompletion('')
      insertState.value = undefined
      return
    }
    setCompletion('')
    clearActiveRequest()
  }

  const { completion, complete, isLoading, stop, setCompletion } = useCompletion({
    api: options.api || '/api/completion',
    streamProtocol: 'text',
    headers: { [headerName]: csrf },
    body: computed(() => ({
      mode: mode.value,
      language: language.value,
      source: inlineContinueSource.value
    })),
    onFinish: (_prompt, completionText) => {
      // For inline suggestion mode, don't clear - let user accept with Tab
      const storage = getCompletionStorage()
      if (mode.value === 'continue' && storage?.visible) {
        console.info('[ai:editor] inline suggestion ready', {
          chars: completionText.length
        })
        return
      }

      if (transformModes.includes(mode.value) && reviewState.value && completionText) {
        addCandidate(completionText)
      }

      insertState.value = undefined
    },
    onError: (error) => {
      console.error('AI completion error:', error)
      clearActiveRequest({ clearReview: false })
    }
  })

  // Watch completion for inline suggestion updates
  watch(completion, (newCompletion, _oldCompletion) => {
    const editor = editorRef.value?.editor
    if (!editor || !newCompletion) return

    const storage = getCompletionStorage()
    if (storage?.visible) {
      // Update inline suggestion
      // Add space prefix if needed (so preview matches what will be inserted)
      let suggestionText = normalizeInlineSuggestion(newCompletion, inlineContinueSource.value)
      if (!suggestionText) return

      if (storage.consumedText) {
        if (!suggestionText.startsWith(storage.consumedText)) return
        suggestionText = suggestionText.slice(storage.consumedText.length)
        if (!suggestionText) return
      }

      if (storage.position !== undefined) {
        const textBefore = editor.state.doc.textBetween(Math.max(0, storage.position - 1), storage.position)
        if (shouldPrefixInlineSpace(textBefore, suggestionText)) {
          suggestionText = ' ' + suggestionText
        }
      }
      storage.setSuggestion(suggestionText)
      editor.view.dispatch(editor.state.tr.setMeta('completionUpdate', true))
    }
  })

  function getMarkdownBefore(editor: Editor, pos: number): string {
    const { state } = editor
    const serializer = (editor.storage.markdown as { serializer?: { serialize: (content: unknown) => string } })?.serializer
    if (serializer) {
      const slice = state.doc.slice(0, pos)
      return serializer.serialize(slice.content)
    }
    // Fallback to plain text
    return state.doc.textBetween(0, pos, '\n')
  }

  function getMarkdownBetween(editor: Editor, from: number, to: number): string {
    const { state } = editor
    const serializer = (editor.storage.markdown as { serializer?: { serialize: (content: unknown) => string } })?.serializer
    if (serializer) {
      const slice = state.doc.slice(from, to)
      return serializer.serialize(slice.content)
    }
    // Fallback to plain text
    return state.doc.textBetween(from, to, '\n\n')
  }

  function triggerTransform(editor: Editor, transformMode: Exclude<CompletionMode, 'continue'>, lang?: string) {
    if (isLoading.value) return

    const { state } = editor
    const { selection } = state

    if (selection.empty) return

    const requestId = beginRequest(transformMode)
    language.value = lang
    const selectedMarkdown = getMarkdownBetween(editor, selection.from, selection.to)

    insertState.value = { pos: selection.from, deleteRange: { from: selection.from, to: selection.to } }
    reviewState.value = {
      pos: selection.from,
      deleteRange: { from: selection.from, to: selection.to },
      prompt: selectedMarkdown,
      mode: transformMode,
      language: lang
    }
    candidates.value = []
    activeCandidateIndex.value = -1

    console.info('[ai:editor] transform start', {
      mode: transformMode,
      selectedChars: selectedMarkdown.length
    })
    void complete(selectedMarkdown).finally(() => {
      if (activeRequestId.value === requestId) {
        activeRequestId.value = 0
      }
    })
  }

  function buildContinuePrompt(editor: Editor, pos: number, options: { singleLine?: boolean } = {}) {
    const context = options.singleLine
      ? editor.state.doc.textBetween(Math.max(0, pos - 700), pos, '\n').trim()
      : getMarkdownBefore(editor, pos).slice(-2400).trim()
    const outputInstruction = options.singleLine
      ? 'Write a short single-line inline completion only. Usually complete the current sentence or phrase in 4-18 words. No headings, lists, tables, code fences, or line breaks.'
      : 'You may continue with prose, a list item, a table row, a fenced code block, or a descriptive Markdown link when the context calls for it.'

    if (options.singleLine) {
      return [
        'Complete the current Markdown line from the cursor.',
        'Output only the missing text after the cursor.',
        outputInstruction,
        '',
        context
      ].join('\n')
    }

    return [
      'Continue the following Markdown draft from the cursor position.',
      'Write only the next natural continuation as valid Markdown. Do not repeat existing text.',
      'Keep the same language, tone, structure, and Markdown style.',
      outputInstruction,
      '',
      context
    ].join('\n')
  }

  function startInlineContinue(editor: Editor, insertPos: number, source: InlineContinueSource) {
    const requestId = beginRequest('continue')
    language.value = undefined
    inlineContinueSource.value = source
    const prompt = buildContinuePrompt(editor, insertPos, { singleLine: source === 'auto' })
    const storage = getCompletionStorage()
    if (!storage) return

    storage.position = insertPos
    storage.visible = true
    storage.consumedText = ''
    storage.setSuggestion('')
    editor.view.dispatch(editor.state.tr.setMeta('completionUpdate', true))

    console.info('[ai:editor] inline continue start', {
      source,
      promptChars: prompt.length,
      position: insertPos
    })
    void complete(prompt).then((result) => {
      if (activeRequestId.value !== requestId || !result || !storage.visible || storage.suggestion) return
      const textBeforePosition = editor.state.doc.textBetween(Math.max(0, insertPos - 1), insertPos)
      let suggestion = normalizeInlineSuggestion(result, source)
      if (!suggestion) return
      suggestion = shouldPrefixInlineSpace(textBeforePosition, suggestion)
        ? ` ${suggestion}`
        : suggestion
      storage.setSuggestion(suggestion)
      editor.view.dispatch(editor.state.tr.setMeta('completionUpdate', true))
      console.info('[ai:editor] inline suggestion set from promise', {
        chars: suggestion.length
      })
    }).finally(() => {
      if (activeRequestId.value === requestId) {
        activeRequestId.value = 0
      }
    })
  }

  function triggerContinue(editor: Editor) {
    if (isLoading.value) return

    const { state } = editor
    const { selection } = state
    const insertPos = selection.empty ? selection.from : selection.to

    startInlineContinue(editor, insertPos, selection.empty ? 'toolbar-cursor' : 'toolbar-selection')
  }

  function acceptCandidate() {
    const editor = editorRef.value?.editor
    const state = reviewState.value
    const text = previewText.value
    if (!editor || !state || !text.trim()) return

    if (isLoading.value) {
      stop()
    }
    activeRequestId.value += 1
    editor.chain()
      .focus()
      .insertContentAt(state.deleteRange, text, { contentType: 'markdown' })
      .run()
    setCompletion('')
    clearActiveRequest({ clearReview: true })
  }

  function discardCandidates() {
    activeRequestId.value += 1
    stop()
    setCompletion('')
    clearActiveRequest({ clearReview: true })
  }

  function selectCandidate(index: number) {
    if (index < 0 || index >= candidates.value.length) return
    activeCandidateIndex.value = index
  }

  function rerollCandidate() {
    const state = reviewState.value
    if (!state || isLoading.value) return

    const requestId = beginRequest(state.mode)
    language.value = state.language
    insertState.value = { pos: state.pos, deleteRange: state.deleteRange }
    console.info('[ai:editor] transform reroll', {
      mode: state.mode,
      previousCandidates: candidates.value.length
    })
    void complete(state.prompt).finally(() => {
      if (activeRequestId.value === requestId) {
        activeRequestId.value = 0
      }
    })
  }

  // Configure Completion extension
  const extension = Completion.configure({
    autoTrigger: true,
    debounce: 500,
    onTrigger: (editor, source) => {
      if (isLoading.value) return
      const insertPos = editor.state.selection.from
      startInlineContinue(editor, insertPos, source === 'auto' ? 'auto' : 'shortcut')
    },
    onAccept: () => {
      setCompletion('')
    },
    onDismiss: () => {
      stopGeneration()
    }
  })

  // Create handlers for toolbar
  const handlers = {
    aiContinue: {
      canExecute: () => !isLoading.value,
      execute: (editor: Editor) => {
        triggerContinue(editor)
        return editor.chain()
      },
      isActive: () => !!(isLoading.value && mode.value === 'continue'),
      isDisabled: () => !!isLoading.value
    },
    aiStop: {
      canExecute: () => !!isLoading.value,
      execute: (editor: Editor) => {
        stopGeneration()
        return editor.chain()
      },
      isActive: () => !!isLoading.value,
      isDisabled: () => !isLoading.value
    },
    aiFix: {
      canExecute: (editor: Editor) => !editor.state.selection.empty && !isLoading.value,
      execute: (editor: Editor) => {
        triggerTransform(editor, 'fix')
        return editor.chain()
      },
      isActive: () => !!(isLoading.value && mode.value === 'fix'),
      isDisabled: (editor: Editor) => editor.state.selection.empty || !!isLoading.value
    },
    aiExtend: {
      canExecute: (editor: Editor) => !editor.state.selection.empty && !isLoading.value,
      execute: (editor: Editor) => {
        triggerTransform(editor, 'extend')
        return editor.chain()
      },
      isActive: () => !!(isLoading.value && mode.value === 'extend'),
      isDisabled: (editor: Editor) => editor.state.selection.empty || !!isLoading.value
    },
    aiReduce: {
      canExecute: (editor: Editor) => !editor.state.selection.empty && !isLoading.value,
      execute: (editor: Editor) => {
        triggerTransform(editor, 'reduce')
        return editor.chain()
      },
      isActive: () => !!(isLoading.value && mode.value === 'reduce'),
      isDisabled: (editor: Editor) => editor.state.selection.empty || !!isLoading.value
    },
    aiSimplify: {
      canExecute: (editor: Editor) => !editor.state.selection.empty && !isLoading.value,
      execute: (editor: Editor) => {
        triggerTransform(editor, 'simplify')
        return editor.chain()
      },
      isActive: () => !!(isLoading.value && mode.value === 'simplify'),
      isDisabled: (editor: Editor) => editor.state.selection.empty || !!isLoading.value
    },
    aiSummarize: {
      canExecute: (editor: Editor) => !editor.state.selection.empty && !isLoading.value,
      execute: (editor: Editor) => {
        triggerTransform(editor, 'summarize')
        return editor.chain()
      },
      isActive: () => !!(isLoading.value && mode.value === 'summarize'),
      isDisabled: (editor: Editor) => editor.state.selection.empty || !!isLoading.value
    },
    aiTranslate: {
      canExecute: (editor: Editor) => !editor.state.selection.empty && !isLoading.value,
      execute: (editor: Editor, cmd: { language?: string } | undefined) => {
        triggerTransform(editor, 'translate', cmd?.language)
        return editor.chain()
      },
      isActive: (_editor: Editor, cmd: { language?: string } | undefined) => !!(isLoading.value && mode.value === 'translate' && language.value === cmd?.language),
      isDisabled: (editor: Editor) => editor.state.selection.empty || !!isLoading.value
    }
  }

  return {
    extension,
    handlers,
    isLoading,
    mode,
    aiReview: {
      candidates,
      activeCandidateIndex,
      activeCandidate,
      previewText,
      isOpen: isReviewOpen,
      accept: acceptCandidate,
      discard: discardCandidates,
      reroll: rerollCandidate,
      select: selectCandidate,
      stop: stopGeneration
    }
  }
}

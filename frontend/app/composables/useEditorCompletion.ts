import { useCompletion } from '@ai-sdk/vue'
import type { Editor } from '@tiptap/vue-3'
import { Completion } from '~/components/editor/CompletionExtension'
import type { CompletionStorage } from '~/components/editor/CompletionExtension'

type CompletionMode = 'continue' | 'fix' | 'extend' | 'reduce' | 'simplify' | 'summarize' | 'translate'
type TransformMode = Exclude<CompletionMode, 'continue'>

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
  const insertedChars = ref(0)
  const activeRequestId = ref(0)
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

  function insertContinueText(editor: Editor, pos: number, text: string, source: string) {
    if (!text || insertedChars.value > 0) return

    const textBefore = editor.state.doc.textBetween(Math.max(0, pos - 1), pos)
    const textToInsert = textBefore && !/\s/.test(textBefore) && !text.startsWith(' ')
      ? ` ${text}`
      : text

    editor.chain()
      .focus()
      .insertContentAt(pos, textToInsert, { contentType: 'markdown' })
      .run()
    insertedChars.value = textToInsert.length
    console.info('[ai:editor] continue inserted', {
      source,
      chars: insertedChars.value
    })
  }

  function beginRequest(nextMode: CompletionMode) {
    activeRequestId.value += 1
    setCompletion('')
    insertedChars.value = 0
    mode.value = nextMode
    getCompletionStorage()?.clearSuggestion()
    return activeRequestId.value
  }

  function clearActiveRequest(options: { clearReview?: boolean } = {}) {
    insertState.value = undefined
    insertedChars.value = 0
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
      insertedChars.value = 0
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
      language: language.value
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

      if (mode.value === 'continue' && insertState.value && completionText && insertedChars.value === 0) {
        const editor = editorRef.value?.editor
        if (editor) {
          insertContinueText(editor, insertState.value.pos, completionText, 'finish')
        }
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
  watch(completion, (newCompletion, oldCompletion) => {
    const editor = editorRef.value?.editor
    if (!editor || !newCompletion) return

    const storage = getCompletionStorage()
    if (storage?.visible) {
      // Update inline suggestion
      // Add space prefix if needed (so preview matches what will be inserted)
      let suggestionText = newCompletion
      if (storage.position !== undefined) {
        const textBefore = editor.state.doc.textBetween(Math.max(0, storage.position - 1), storage.position)
        if (textBefore && !/\s/.test(textBefore) && !suggestionText.startsWith(' ')) {
          suggestionText = ' ' + suggestionText
        }
      }
      storage.setSuggestion(suggestionText)
      editor.view.dispatch(editor.state.tr.setMeta('completionUpdate', true))
    } else if (insertState.value) {
      // Direct insertion/transform mode (from toolbar actions)

      if (transformModes.includes(mode.value)) {
        return
      }

      // If this is the first chunk and we have a selection to replace, delete it first
      if (insertState.value.deleteRange && !oldCompletion) {
        editor.chain()
          .focus()
          .deleteRange(insertState.value.deleteRange)
          .run()
        insertState.value.deleteRange = undefined
      }

      let delta = newCompletion.slice(oldCompletion?.length || 0)
      if (delta) {
        // For single-paragraph transforms, replace all line breaks with spaces
        if (['fix', 'simplify', 'translate'].includes(mode.value)) {
          delta = delta.replace(/[\r\n]+/g, ' ').replace(/\s{2,}/g, ' ')
        }

        // For "continue" mode, add a space before if needed (first chunk only)
        if (mode.value === 'continue' && !oldCompletion) {
          const textBefore = editor.state.doc.textBetween(Math.max(0, insertState.value.pos - 1), insertState.value.pos)
          if (textBefore && !/\s/.test(textBefore)) {
            delta = ' ' + delta
          }
        }

        editor.chain().focus().command(({ tr }) => {
          tr.insertText(delta, insertState.value!.pos)
          return true
        }).run()
        insertState.value.pos += delta.length
        insertedChars.value += delta.length
        console.info('[ai:editor] continue chunk inserted', {
          chars: delta.length,
          totalChars: insertedChars.value
        })
      }
    }
  })

  function triggerTransform(editor: Editor, transformMode: Exclude<CompletionMode, 'continue'>, lang?: string) {
    if (isLoading.value) return

    const { state } = editor
    const { selection } = state

    if (selection.empty) return

    const requestId = beginRequest(transformMode)
    language.value = lang
    const selectedText = state.doc.textBetween(selection.from, selection.to, '\n\n')

    insertState.value = { pos: selection.from, deleteRange: { from: selection.from, to: selection.to } }
    reviewState.value = {
      pos: selection.from,
      deleteRange: { from: selection.from, to: selection.to },
      prompt: selectedText,
      mode: transformMode,
      language: lang
    }
    candidates.value = []
    activeCandidateIndex.value = -1

    console.info('[ai:editor] transform start', {
      mode: transformMode,
      selectedChars: selectedText.length
    })
    void complete(selectedText).finally(() => {
      if (activeRequestId.value === requestId) {
        activeRequestId.value = 0
      }
    })
  }

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

  function buildContinuePrompt(editor: Editor, pos: number) {
    const markdownBefore = getMarkdownBefore(editor, pos)
    const context = markdownBefore.slice(-2400).trim()
    return [
      'Continue the following draft from the cursor position.',
      'Write only the next natural continuation. Do not repeat existing text.',
      'Keep the same language, tone, and markdown style.',
      '',
      context
    ].join('\n')
  }

  function startContinue(editor: Editor, insertPos: number, source: string) {
    const requestId = beginRequest('continue')
    language.value = undefined
    const prompt = buildContinuePrompt(editor, insertPos)
    insertState.value = { pos: insertPos }
    console.info('[ai:editor] continue start', {
      source,
      promptChars: prompt.length,
      position: insertPos
    })
    void complete(prompt).then((result) => {
      if (activeRequestId.value !== requestId || !result) return
      insertContinueText(editor, insertPos, result, 'promise')
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

    startContinue(editor, insertPos, selection.empty ? 'cursor' : 'selection')
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
    onTrigger: (editor) => {
      if (isLoading.value) return
      const requestId = beginRequest('continue')
      const insertPos = editor.state.selection.from
      const prompt = buildContinuePrompt(editor, insertPos)
      console.info('[ai:editor] inline continue start', {
        promptChars: prompt.length,
        position: insertPos
      })
      void complete(prompt).then((result) => {
        if (activeRequestId.value !== requestId) return
        const storage = getCompletionStorage()
        if (!result || !storage?.visible || storage.suggestion) return
        const textBeforePosition = editor.state.doc.textBetween(Math.max(0, insertPos - 1), insertPos)
        const suggestion = textBeforePosition && !/\s/.test(textBeforePosition) && !result.startsWith(' ')
          ? ` ${result}`
          : result
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

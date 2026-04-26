import { useCompletion } from '@ai-sdk/vue'
import type { Editor } from '@tiptap/vue-3'
import { Completion } from '~/components/editor/CompletionExtension'
import type { CompletionStorage } from '~/components/editor/CompletionExtension'

type CompletionMode = 'continue' | 'fix' | 'extend' | 'reduce' | 'simplify' | 'summarize' | 'translate'

interface UseEditorCompletionOptions {
  api?: string
}

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

      // For transform modes, insert the full completion with markdown parsing
      const transformModes = ['fix', 'extend', 'reduce', 'simplify', 'summarize', 'translate']
      if (transformModes.includes(mode.value) && insertState.value && completionText) {
        const editor = editorRef.value?.editor
        if (editor) {
          // Delete the original selection if not already done
          if (insertState.value.deleteRange) {
            editor.chain()
              .focus()
              .deleteRange(insertState.value.deleteRange)
              .run()
          }
          // Insert with markdown parsing
          editor.chain()
            .focus()
            .insertContentAt(insertState.value.pos, completionText, { contentType: 'markdown' })
            .run()
        }
      }

      insertState.value = undefined
    },
    onError: (error) => {
      console.error('AI completion error:', error)
      insertState.value = undefined
      insertedChars.value = 0
      getCompletionStorage()?.clearSuggestion()
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

      // Transform modes use markdown insertion - wait for full completion
      const transformModes = ['fix', 'extend', 'reduce', 'simplify', 'summarize', 'translate']
      if (transformModes.includes(mode.value)) {
        // Don't stream - will be handled in onFinish
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

    setCompletion('')
    insertedChars.value = 0
    getCompletionStorage()?.clearSuggestion()

    const { state } = editor
    const { selection } = state

    if (selection.empty) return

    mode.value = transformMode
    language.value = lang
    const selectedText = state.doc.textBetween(selection.from, selection.to)

    // Replace the selected text with the transformed version
    insertState.value = { pos: selection.from, deleteRange: { from: selection.from, to: selection.to } }

    console.info('[ai:editor] transform start', {
      mode: transformMode,
      selectedChars: selectedText.length
    })
    complete(selectedText)
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

  function triggerContinue(editor: Editor) {
    if (isLoading.value) return

    setCompletion('')
    insertedChars.value = 0
    mode.value = 'continue'
    getCompletionStorage()?.clearSuggestion()
    const { state } = editor
    const { selection } = state

    if (selection.empty) {
      // No selection: continue from cursor position
      const textBefore = getMarkdownBefore(editor, selection.from)
      const insertPos = selection.from
      insertState.value = { pos: insertPos }
      console.info('[ai:editor] continue start', {
        promptChars: textBefore.length,
        position: insertPos
      })
      void complete(textBefore).then((result) => {
        if (result) insertContinueText(editor, insertPos, result, 'promise')
      })
    } else {
      // Text selected: append completion after the selection
      const textBefore = getMarkdownBefore(editor, selection.to)
      const insertPos = selection.to
      insertState.value = { pos: insertPos }
      console.info('[ai:editor] continue start', {
        promptChars: textBefore.length,
        position: insertPos
      })
      void complete(textBefore).then((result) => {
        if (result) insertContinueText(editor, insertPos, result, 'promise')
      })
    }
  }

  // Configure Completion extension
  const extension = Completion.configure({
    onTrigger: (editor) => {
      if (isLoading.value) return
      setCompletion('')
      insertedChars.value = 0
      mode.value = 'continue'
      const textBefore = getMarkdownBefore(editor, editor.state.selection.from)
      const insertPos = editor.state.selection.from
      console.info('[ai:editor] inline continue start', {
        promptChars: textBefore.length,
        position: insertPos
      })
      void complete(textBefore).then((result) => {
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
      })
    },
    onAccept: () => {
      setCompletion('')
    },
    onDismiss: () => {
      stop()
      setCompletion('')
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
    mode
  }
}

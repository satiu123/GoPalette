import { Extension } from '@tiptap/core'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import type { Editor } from '@tiptap/vue-3'
import { useDebounceFn } from '@vueuse/core'

type CompletionTriggerSource = 'auto' | 'manual'

export interface CompletionOptions {
  /**
   * Whether to automatically trigger completion while typing
   * @defaultValue false
   */
  autoTrigger?: boolean
  /**
   * Debounce delay in ms before triggering completion
   * @defaultValue 250
   */
  debounce?: number
  /**
   * Characters that should prevent completion from triggering
   * @defaultValue ['/', ':', '@']
   */
  triggerCharacters?: string[]
  /**
   * Called when completion should be triggered, receives the editor instance
   */
  onTrigger?: (editor: Editor, source: CompletionTriggerSource) => void
  /**
   * Called when suggestion is accepted
   */
  onAccept?: () => void
  /**
   * Called when suggestion is dismissed
   */
  onDismiss?: () => void
}

export interface CompletionStorage {
  suggestion: string
  consumedText: string
  position: number | undefined
  visible: boolean
  debouncedTrigger: ((editor: Editor, source?: CompletionTriggerSource) => void) | null
  setSuggestion: (text: string) => void
  clearSuggestion: () => void
}

export const completionPluginKey = new PluginKey('completion')

function shouldInsertAsMarkdown(text: string) {
  return /[\r\n]/.test(text) || /^(#{1,6}\s|[-*+]\s|\d+\.\s|>\s|```|~~~|\|)/.test(text.trimStart())
}

export const Completion = Extension.create<CompletionOptions, CompletionStorage>({
  name: 'completion',

  addOptions() {
    return {
      autoTrigger: false,
      debounce: 250,
      triggerCharacters: ['/', ':', '@'],
      onTrigger: undefined,
      onAccept: undefined,
      onDismiss: undefined
    }
  },

  addStorage() {
    return {
      suggestion: '',
      consumedText: '',
      position: undefined as number | undefined,
      visible: false,
      debouncedTrigger: null as ((editor: Editor, source?: CompletionTriggerSource) => void) | null,
      setSuggestion(text: string) {
        this.suggestion = text
      },
      clearSuggestion() {
        this.suggestion = ''
        this.consumedText = ''
        this.position = undefined
        this.visible = false
      }
    }
  },

  addProseMirrorPlugins() {
    const storage = this.storage

    return [
      new Plugin({
        key: completionPluginKey,
        props: {
          decorations(state) {
            if (!storage.visible || !storage.suggestion || storage.position === undefined) {
              return DecorationSet.empty
            }

            const widget = Decoration.widget(storage.position, () => {
              const span = document.createElement('span')
              span.className = 'completion-suggestion'
              span.textContent = storage.suggestion
              span.style.cssText = 'color: var(--ui-text-muted); opacity: 0.6; pointer-events: none; white-space: pre-wrap;'
              return span
            }, { side: 1 })

            return DecorationSet.create(state.doc, [widget])
          }
        }
      })
    ]
  },

  addKeyboardShortcuts() {
    const triggerCompletion = (editor: Editor) => {
      if (this.storage.visible) {
        this.storage.clearSuggestion()
        this.options.onDismiss?.()
      }
      this.storage.debouncedTrigger?.(editor, 'manual')
      return true
    }

    return {
      'Mod-j': ({ editor }) => {
        return triggerCompletion(editor as Editor)
      },
      'Alt-j': ({ editor }) => {
        return triggerCompletion(editor as Editor)
      },
      'Alt-J': ({ editor }) => {
        return triggerCompletion(editor as Editor)
      },
      'Tab': ({ editor }) => {
        if (!this.storage.visible || !this.storage.suggestion || this.storage.position === undefined) {
          return false
        }

        // Store values before clearing
        const suggestion = this.storage.suggestion
        const position = this.storage.position

        // Clear suggestion first
        this.storage.clearSuggestion()

        // Force decoration update
        editor.view.dispatch(editor.state.tr.setMeta('completionUpdate', true))

        if (shouldInsertAsMarkdown(suggestion)) {
          editor.chain().focus().insertContentAt(position, suggestion, { contentType: 'markdown' }).run()
        } else {
          editor.chain().focus().command(({ tr }) => {
            tr.insertText(suggestion, position)
            return true
          }).run()
        }

        this.options.onAccept?.()
        return true
      },
      'Escape': ({ editor }) => {
        if (this.storage.visible) {
          this.storage.clearSuggestion()
          // Force decoration update
          editor.view.dispatch(editor.state.tr.setMeta('completionUpdate', true))
          this.options.onDismiss?.()
          return true
        }
        return false
      }
    }
  },

  onUpdate({ editor, transaction }) {
    if (!transaction.docChanged) return

    if (this.storage.visible) {
      const { selection, doc } = editor.state
      const typedText = this.storage.position !== undefined && selection.empty && selection.from > this.storage.position
        ? doc.textBetween(this.storage.position, selection.from, '\n')
        : ''

      if (typedText && this.storage.suggestion.startsWith(typedText)) {
        this.storage.consumedText += typedText
        this.storage.position = selection.from
        this.storage.suggestion = this.storage.suggestion.slice(typedText.length)

        if (!this.storage.suggestion) {
          this.storage.clearSuggestion()
          this.options.onAccept?.()
        }

        editor.view.dispatch(editor.state.tr.setMeta('completionUpdate', true))
        return
      }

      this.storage.clearSuggestion()
      // Force decoration update
      editor.view.dispatch(editor.state.tr.setMeta('completionUpdate', true))
      this.options.onDismiss?.()
      return
    }

    // Debounced trigger check (only if autoTrigger is enabled)
    if (this.options.autoTrigger) {
      this.storage.debouncedTrigger?.(editor as Editor, 'auto')
    }
  },

  onSelectionUpdate({ editor, transaction }) {
    if (!transaction.selectionSet) return

    if (this.storage.visible) {
      this.storage.clearSuggestion()
      // Force decoration update
      editor.view.dispatch(editor.state.tr.setMeta('completionUpdate', true))
      this.options.onDismiss?.()
    }
  },

  onCreate() {
    const storage = this.storage
    const options = this.options

    // Create debounced trigger function for this instance
    this.storage.debouncedTrigger = useDebounceFn((editor: Editor, source: CompletionTriggerSource = 'auto') => {
      if (!options.onTrigger) return

      const { state } = editor
      const { selection } = state
      const { $from } = selection

      // Only suggest at end of block with content
      const hasEmptySelection = selection.empty
      const isAtEndOfBlock = $from.parentOffset === $from.parent.content.size
      const hasContent = $from.parent.textContent.trim().length >= 6
      const textContent = $from.parent.textContent

      // Don't trigger if sentence is complete (ends with punctuation)
      const endsWithPunctuation = /[.!?。！？]\s*$/.test(textContent)

      // Don't trigger if text ends with trigger characters
      const triggerChars = options.triggerCharacters || []
      const endsWithTrigger = triggerChars.some(char => textContent.endsWith(char))

      if (!hasEmptySelection || !isAtEndOfBlock || !hasContent || endsWithPunctuation || endsWithTrigger) {
        return
      }

      // Set position and mark as visible
      storage.position = selection.from
      storage.visible = true

      // Pass editor to let the handler extract content (e.g., as markdown)
      options.onTrigger(editor, source)
    }, options.debounce || 250)
  },

  onDestroy() {
    this.storage.debouncedTrigger = null
  }
})

export default Completion

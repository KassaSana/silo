import { useEffect } from 'react'

/*
 * useKeyboard — Global keyboard event handler.
 *
 * HOW IT WORKS:
 * You pass a map of key -> handler function. The hook attaches a single
 * keydown listener to the window. When a key is pressed, it looks up
 * the handler and calls it.
 *
 * WHY global listener?
 * In a TUI app, keyboard shortcuts work regardless of what's focused.
 * You don't click a button then press Enter — you just press Enter
 * from anywhere. This mimics how terminal apps like vim work.
 *
 * SMART FEATURE: It skips handlers when the user is typing in an input.
 * Otherwise, pressing 'j' to type the letter 'j' in a text field
 * would also trigger the "move down" action. Not good.
 *
 * Usage:
 *   useKeyboard({
 *     'j': () => moveDown(),
 *     'k': () => moveUp(),
 *     'Enter': () => selectItem(),
 *     'Escape': () => goBack(),
 *   })
 */

type KeyMap = Record<string, (e: KeyboardEvent) => void>

export function useKeyboard(keyMap: KeyMap) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Don't intercept keys when user is typing in an input/textarea
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
        // But still allow Escape and Enter — these are navigation keys
        // even inside inputs (Escape = cancel, Enter = submit)
        if (e.key !== 'Escape' && e.key !== 'Enter') {
          return
        }
      }

      // Build a key string that includes modifiers: "ctrl+s", "shift+Enter", etc.
      const parts: string[] = []
      if (e.ctrlKey || e.metaKey) parts.push('ctrl')
      if (e.shiftKey) parts.push('shift')
      if (e.altKey) parts.push('alt')
      parts.push(e.key)
      const combo = parts.join('+')

      // Try combo first (e.g. "ctrl+s"), then plain key (e.g. "s")
      const fn = keyMap[combo] || (!e.ctrlKey && !e.metaKey && !e.altKey ? keyMap[e.key] : undefined)
      if (fn) {
        e.preventDefault()
        fn(e)
      }
    }

    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [keyMap])
}

import { useState, useMemo } from 'react'
import { useKeyboard } from './useKeyboard'

/*
 * useNavigation — List navigation with j/k and arrow keys.
 *
 * Manages a selectedIndex that wraps around the list boundaries.
 * Returns the index and a setter (for when you need to reset it).
 *
 * WRAPPING: If you press 'k' at the top, it wraps to the bottom.
 * This is standard TUI behavior — no dead ends.
 *
 * WHY useMemo for the keyMap?
 * React re-creates objects on every render. If we pass a new keyMap
 * object each time, useKeyboard's useEffect would re-attach the
 * listener on every render (wasteful). useMemo keeps it stable.
 */

interface UseNavigationOptions {
  itemCount: number
  onSelect?: (index: number) => void
  extraKeys?: Record<string, (e: KeyboardEvent) => void>
}

export function useNavigation({ itemCount, onSelect, extraKeys = {} }: UseNavigationOptions) {
  const [selectedIndex, setSelectedIndex] = useState(0)

  const keyMap = useMemo(() => ({
    // j and ArrowDown move selection down
    'j': () => setSelectedIndex((i) => (i + 1) % itemCount),
    'ArrowDown': () => setSelectedIndex((i) => (i + 1) % itemCount),
    // k and ArrowUp move selection up
    'k': () => setSelectedIndex((i) => (i - 1 + itemCount) % itemCount),
    'ArrowUp': () => setSelectedIndex((i) => (i - 1 + itemCount) % itemCount),
    // Enter triggers select callback
    'Enter': () => onSelect?.(selectedIndex),
    // Spread any extra screen-specific keys
    ...extraKeys,
  }), [itemCount, selectedIndex, onSelect, extraKeys])

  useKeyboard(keyMap)

  return { selectedIndex, setSelectedIndex }
}

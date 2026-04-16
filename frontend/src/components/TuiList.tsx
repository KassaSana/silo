/*
 * TuiList — Keyboard-navigable list with selection indicator.
 *
 * This is the core navigation pattern in TUI apps:
 *   ▸ selected item     (▸ = blue arrow, background highlighted)
 *     unselected item   (dimmer, no indicator)
 *
 * The parent controls `selectedIndex` — this component just renders.
 * Keyboard handling (j/k/arrows) lives in useKeyboard hook, not here.
 *
 * WHY separate rendering from keyboard handling?
 * Because different screens might want different key bindings.
 * The list just needs to know "which item is selected" — it doesn't
 * care HOW that selection changes.
 */

interface TuiListProps<T> {
  items: T[]
  selectedIndex: number
  renderItem: (item: T, isSelected: boolean) => React.ReactNode
}

import React from 'react'

export function TuiList<T>({ items, selectedIndex, renderItem }: TuiListProps<T>) {
  return (
    <ul className="tui-list">
      {items.map((item, index) => {
        const isSelected = index === selectedIndex
        return (
          <li
            key={index}
            className={`tui-list__item ${isSelected ? 'tui-list__item--selected' : ''}`}
          >
            <span className="tui-list__indicator">
              {isSelected ? '▸' : ' '}
            </span>
            {renderItem(item, isSelected)}
          </li>
        )
      })}
    </ul>
  )
}

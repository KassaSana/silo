import React from 'react'

/*
 * TuiBox — The outermost container for every screen.
 *
 * It provides the 1px border, dark background, and flex column layout
 * that makes content flow top-to-bottom: Header → Content → Footer.
 *
 * Think of it as the terminal window frame itself.
 */

interface TuiBoxProps {
  children: React.ReactNode
}

export function TuiBox({ children }: TuiBoxProps) {
  return <div className="tui-box">{children}</div>
}

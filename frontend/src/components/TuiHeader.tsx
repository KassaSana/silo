import { useState, useEffect } from 'react'

/*
 * TuiHeader — Top bar with breadcrumb navigation and clock.
 *
 * Every TUI app has a status bar. Ours shows:
 *   Left:  ■ silo › screen-name › sub-screen
 *   Right: HH:MM AM/PM (live clock)
 *
 * The breadcrumb uses › as separator — mimics terminal path display.
 * The green ■ square is silo's identity marker (like a cursor block).
 */

interface TuiHeaderProps {
  breadcrumb: string[]  // e.g. ['silo', 'react-project', 'edit']
}

export function TuiHeader({ breadcrumb }: TuiHeaderProps) {
  const [time, setTime] = useState(formatTime())

  useEffect(() => {
    // Update clock every 30 seconds — doesn't need to be per-second
    const interval = setInterval(() => setTime(formatTime()), 30000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="tui-header">
      <div className="tui-header__title">
        <span className="tui-header__icon">■</span>
        <span>{breadcrumb.join(' › ')}</span>
      </div>
      <span className="tui-header__clock">{time}</span>
    </div>
  )
}

function formatTime(): string {
  const now = new Date()
  return now.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  })
}

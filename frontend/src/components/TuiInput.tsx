import React, { useRef, useEffect } from 'react'

/*
 * TuiInput — Terminal-styled text input.
 *
 * Looks like a command prompt: dark background, monospace, cursor blink.
 * Supports an optional "> " prefix to mimic terminal input style.
 *
 * autoFocus is important in TUI apps — when a screen with an input
 * appears, the cursor should already be in the input. No mouse needed.
 */

interface TuiInputProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  autoFocus?: boolean
  onSubmit?: () => void
}

export function TuiInput({ value, onChange, placeholder, autoFocus, onSubmit }: TuiInputProps) {
  const ref = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (autoFocus && ref.current) {
      ref.current.focus()
    }
  }, [autoFocus])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && onSubmit) {
      onSubmit()
    }
  }

  return (
    <input
      ref={ref}
      className="tui-input"
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      onKeyDown={handleKeyDown}
      placeholder={placeholder}
      autoComplete="off"
      spellCheck={false}
    />
  )
}

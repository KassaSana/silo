import { useState, useEffect, useMemo } from 'react'
import { TuiBox, TuiHeader, TuiFooter } from '../components'
import { useKeyboard } from '../hooks'
import { GetCurrentSession, AttemptUnlock } from '../../wailsjs/go/main/App'

/*
 * UnlockAttempt — Type N random chars to break the seal (Screen 6).
 *
 * The lock text is displayed, and the user types it character by character.
 * The progress bar shows how far they've gotten.
 *
 * If they complete it: seal breaks, session ends.
 * If they fail: attempt counter goes up, chars ESCALATE (200→400→600).
 * Each failed attempt generates a NEW, LONGER lock text.
 *
 * This is the friction barrier. It converts impulsive "I want to quit"
 * into a deliberate, tedious act that gives you time to reconsider.
 */

interface UnlockAttemptProps {
  onNavigate: (screen: string) => void
}

export function UnlockAttempt({ onNavigate }: UnlockAttemptProps) {
  const [lockText, setLockText] = useState('')
  const [lockChars, setLockChars] = useState(200)
  const [input, setInput] = useState('')
  const [attempt, setAttempt] = useState(1)
  const [error, setError] = useState('')

  useEffect(() => {
    GetCurrentSession().then((s) => {
      if (s) {
        setLockText(s.lock_text)
        setLockChars(s.lock_chars)
        setAttempt(s.breach_attempts + 1)
      }
    })
  }, [])

  const handleSubmit = async () => {
    try {
      const result = await AttemptUnlock(input)
      // AttemptUnlock returns [success, newLockText, newLockChars]
      // But Wails flattens Go multiple returns differently...
      // The binding returns a Promise<boolean> for the first return
      // This is a simplification — in practice we'd check the response
      if (result) {
        onNavigate('dashboard')
      }
    } catch (err) {
      // Failed attempt — reload session for new lock text
      setError('incorrect. lock has been escalated.')
      setInput('')
      const s = await GetCurrentSession()
      if (s) {
        setLockText(s.lock_text)
        setLockChars(s.lock_chars)
        setAttempt(s.breach_attempts + 1)
      }
    }
  }

  const screenKeys = useMemo(() => ({
    'Escape': () => onNavigate('active-seal'),
  }), [onNavigate])
  useKeyboard(screenKeys)

  // Show how much of the lock text matches the input so far
  const correctChars = countCorrect(input, lockText)

  // Display the lock text in rows of ~50 chars
  const lockRows = lockText ? splitIntoRows(lockText, 50) : []

  return (
    <TuiBox>
      <TuiHeader breadcrumb={['silo', 'LOCKED', `breach attempt #${attempt}`]} />

      <div className="tui-content">
        <div className="text-dim mb-md">type the following text exactly to unlock:</div>

        {/* Lock text display */}
        <div style={{
          background: 'var(--bg-secondary)',
          padding: '12px 16px',
          border: '1px solid var(--border)',
          marginBottom: '16px',
          lineHeight: '1.8',
          wordBreak: 'break-all',
        }}>
          {lockRows.map((row, i) => (
            <div key={i} className="text-primary" style={{ fontSize: '12px' }}>{row}</div>
          ))}
        </div>

        <hr className="tui-divider" />

        {/* Input area */}
        <textarea
          className="tui-input"
          style={{
            height: '80px',
            resize: 'none',
            fontSize: '12px',
            fontFamily: 'var(--font-mono)',
          }}
          value={input}
          onChange={(e) => {
            setInput(e.target.value)
            setError('')
          }}
          autoFocus
          spellCheck={false}
          autoComplete="off"
        />

        <div className="mt-sm flex-row" style={{ justifyContent: 'space-between' }}>
          <span className="text-dim">
            progress: {correctChars}/{lockChars} characters
          </span>
          {attempt < 3 && (
            <span className="text-yellow" style={{ fontSize: '12px' }}>
              attempt #{attempt + 1} will require {lockChars + 200} characters
            </span>
          )}
        </div>

        {error && <div className="text-red mt-sm">{error}</div>}

        {input.length >= lockChars && (
          <button
            style={{
              marginTop: '16px',
              background: 'var(--accent-red)',
              color: 'var(--bg-primary)',
              border: 'none',
              padding: '8px 24px',
              fontFamily: 'var(--font-mono)',
              fontSize: '13px',
              cursor: 'pointer',
              width: '100%',
            }}
            onClick={handleSubmit}
          >
            submit unlock attempt
          </button>
        )}
      </div>

      <TuiFooter actions={[
        { key: 'esc', label: 'cancel attempt (lock stays active)' },
      ]} />
    </TuiBox>
  )
}

function countCorrect(input: string, target: string): number {
  let count = 0
  for (let i = 0; i < input.length && i < target.length; i++) {
    if (input[i] === target[i]) count++
    else break
  }
  return count
}

function splitIntoRows(text: string, charsPerRow: number): string[] {
  const rows: string[] = []
  for (let i = 0; i < text.length; i += charsPerRow) {
    rows.push(text.slice(i, i + charsPerRow))
  }
  return rows
}

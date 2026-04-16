import { useState, useEffect, useMemo } from 'react'
import { TuiBox, TuiHeader, TuiFooter } from '../components'
import { useKeyboard } from '../hooks'
import { GetCurrentSession, GetBlockedAttempts, CompleteSession } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

/*
 * ActiveSeal — The locked screen with countdown timer (Screen 3).
 *
 * EVENTS: The Go timer sends "timer:tick" events every second.
 * We listen for them with Wails EventsOn(). When "timer:done" fires,
 * we show the completion prompt.
 *
 * The "blocked just now" log shows recently killed processes,
 * polled every 2 seconds from the Go backend.
 */

interface TimerState {
  remaining: number
  elapsed: number
  formatted: string
  done: boolean
}

interface BlockedAttempt {
  name: string
  timestamp: string
}

interface ActiveSealProps {
  onNavigate: (screen: string) => void
}

export function ActiveSeal({ onNavigate }: ActiveSealProps) {
  const [timer, setTimer] = useState<TimerState>({ remaining: 0, elapsed: 0, formatted: '00:00', done: false })
  const [session, setSession] = useState<any>(null)
  const [blocked, setBlocked] = useState<BlockedAttempt[]>([])
  const [phase, setPhase] = useState<'active' | 'completing'>('active')
  const [commitMsg, setCommitMsg] = useState('')

  // Load session info
  useEffect(() => {
    GetCurrentSession().then(setSession)
  }, [])

  // Listen for timer events from Go backend
  useEffect(() => {
    const cancelTick = EventsOn('timer:tick', (state: TimerState) => {
      setTimer(state)
    })
    const cancelDone = EventsOn('timer:done', () => {
      setPhase('completing')
    })
    return () => {
      cancelTick()
      cancelDone()
    }
  }, [])

  // Poll blocked attempts every 2 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      GetBlockedAttempts().then((b) => setBlocked(b || []))
    }, 2000)
    return () => clearInterval(interval)
  }, [])

  const screenKeys = useMemo(() => ({
    'x': () => {
      if (phase === 'active') onNavigate('quick-exception')
    },
    'u': () => {
      if (phase === 'active') onNavigate('unlock-attempt')
    },
    'Enter': () => {
      if (phase === 'completing' && commitMsg.trim()) {
        CompleteSession(commitMsg.trim())
          .then(() => onNavigate('dashboard'))
      }
    },
  }), [phase, commitMsg, onNavigate])
  useKeyboard(screenKeys)

  const elapsedFormatted = formatElapsed(timer.elapsed)
  const recentBlocked = blocked.slice(-5).reverse()

  return (
    <TuiBox>
      <TuiHeader breadcrumb={['silo', 'LOCKED']} />

      <div className="tui-content">
        {phase === 'completing' ? (
          <>
            <div style={{ textAlign: 'center', paddingTop: '24px' }}>
              <div className="text-green" style={{ fontSize: '24px', fontWeight: 700 }}>SESSION COMPLETE</div>
              <div className="text-dim mt-md">{formatElapsed(timer.elapsed)} focused</div>
            </div>
            <div className="mt-lg">
              <div className="tui-label">commit message — what did you accomplish?</div>
              <input
                className="tui-input"
                type="text"
                value={commitMsg}
                onChange={(e) => setCommitMsg(e.target.value)}
                placeholder="describe what you did"
                autoFocus
                autoComplete="off"
                spellCheck={false}
              />
            </div>
          </>
        ) : (
          <>
            {/* Big countdown */}
            <div style={{ textAlign: 'center', padding: '16px 0' }}>
              <div className="tui-timer">{timer.formatted}</div>
              <div className="tui-timer__label">remaining</div>
            </div>

            {/* Session info */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', marginBottom: '16px' }}>
              <div>
                <span className="text-dim" style={{ width: '90px', display: 'inline-block' }}>workspace</span>
                <span>{session?.workspace_name}</span>
              </div>
              <div>
                <span className="text-dim" style={{ width: '90px', display: 'inline-block' }}>task</span>
                <span className="text-primary">{session?.task_description}</span>
              </div>
              <div>
                <span className="text-dim" style={{ width: '90px', display: 'inline-block' }}>lock</span>
                <span>{session?.lock_type} ({session?.lock_chars} chars)</span>
              </div>
              <div>
                <span className="text-dim" style={{ width: '90px', display: 'inline-block' }}>elapsed</span>
                <span>{elapsedFormatted}</span>
              </div>
              {(session?.exceptions?.length ?? 0) > 0 && (
                <div>
                  <span className="text-dim" style={{ width: '90px', display: 'inline-block' }}>exceptions</span>
                  <span className="text-yellow">{session.exceptions.length} added this session</span>
                </div>
              )}
            </div>

            <hr className="tui-divider" />

            {/* Blocked log */}
            {recentBlocked.length > 0 && (
              <>
                <div className="tui-label">blocked just now</div>
                {recentBlocked.map((b, i) => (
                  <div key={i} className="blocked-entry">
                    <span className="blocked-entry__icon">✕</span>
                    <span>{b.name}</span>
                    <span className="blocked-entry__time">{timeAgo(b.timestamp)}</span>
                  </div>
                ))}
                <div className="text-dim mt-md" style={{ fontStyle: 'italic', fontSize: '12px' }}>
                  "that's not what you're doing right now."
                </div>
              </>
            )}
          </>
        )}
      </div>

      <TuiFooter
        actions={
          phase === 'completing'
            ? [{ key: 'enter', label: 'save & finish' }]
            : [
                { key: 'x', label: 'quick exception' },
                { key: 'u', label: `unlock (${session?.lock_chars || 200} chars)` },
              ]
        }
      />
    </TuiBox>
  )
}

function formatElapsed(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
}

function timeAgo(timestamp: string): string {
  const diff = Math.floor((Date.now() - new Date(timestamp).getTime()) / 1000)
  if (diff < 5) return 'just now'
  if (diff < 60) return `${diff} sec ago`
  return `${Math.floor(diff / 60)} min ago`
}

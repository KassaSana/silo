import { useState, useMemo } from 'react'
import { TuiBox, TuiHeader, TuiFooter, TuiInput } from '../components'
import { useKeyboard } from '../hooks'
import { AddException } from '../../wailsjs/go/main/App'

/*
 * QuickException — Friction-gated temporary allowance (Screen 7).
 *
 * FLOW:
 *   1. Choose: site or app
 *   2. Type the domain or app name
 *   3. Type "i need this" to confirm
 *
 * The confirmation phrase is the FRICTION BARRIER. It stops impulsive
 * "let me just check Twitter for a sec" but lets through genuine needs
 * like "I forgot to add docs.python.org to my workspace."
 *
 * The exception is session-scoped (gone when session ends) and logged.
 */

type ExceptionType = 'site' | 'app'
type Phase = 'type' | 'value' | 'confirm'

interface QuickExceptionProps {
  onNavigate: (screen: string) => void
}

export function QuickException({ onNavigate }: QuickExceptionProps) {
  const [exType, setExType] = useState<ExceptionType>('site')
  const [value, setValue] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [phase, setPhase] = useState<Phase>('type')
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    try {
      await AddException(exType, value.trim(), confirmation)
      onNavigate('active-seal')
    } catch (err) {
      setError(String(err))
    }
  }

  const screenKeys = useMemo(() => ({
    'Escape': () => {
      if (phase === 'confirm') setPhase('value')
      else if (phase === 'value') setPhase('type')
      else onNavigate('active-seal')
    },
    'Enter': () => {
      if (phase === 'type') {
        setPhase('value')
      } else if (phase === 'value' && value.trim()) {
        setPhase('confirm')
      } else if (phase === 'confirm') {
        handleSubmit()
      }
    },
  }), [phase, value, confirmation, exType])
  useKeyboard(screenKeys)

  return (
    <TuiBox>
      <TuiHeader breadcrumb={['silo', 'LOCKED', 'quick exception']} />

      <div className="tui-content">
        <div className="text-dim mb-md">need something not in your workspace?</div>

        {/* Step 1: Choose type */}
        <div className="tui-label">type</div>
        <div className="mb-md">
          {(['site', 'app'] as ExceptionType[]).map((t) => (
            <div
              key={t}
              style={{
                padding: '2px 8px',
                cursor: phase === 'type' ? 'pointer' : 'default',
                background: exType === t ? 'var(--bg-tertiary)' : undefined,
              }}
              onClick={() => phase === 'type' && setExType(t)}
            >
              <span className="text-blue">{exType === t ? '▸ ' : '  '}</span>
              <span className={exType === t ? 'text-primary' : ''}>{t}</span>
            </div>
          ))}
        </div>

        {/* Step 2: Enter value */}
        {phase !== 'type' && (
          <>
            <div className="tui-label">{exType === 'site' ? 'domain' : 'app name'}</div>
            <div className="mb-md">
              <TuiInput
                value={value}
                onChange={setValue}
                placeholder={exType === 'site' ? 'docs.python.org' : 'Slack'}
                autoFocus={phase === 'value'}
              />
            </div>
          </>
        )}

        {/* Step 3: Confirm */}
        {phase === 'confirm' && (
          <>
            <div className="tui-label">to confirm, type "i need this"</div>
            <div className="mb-md">
              <TuiInput
                value={confirmation}
                onChange={setConfirmation}
                placeholder='i need this'
                autoFocus
              />
            </div>
          </>
        )}

        <div className="text-yellow mt-md" style={{ fontSize: '12px' }}>
          this exception lasts for this session only
        </div>
        <div className="text-yellow" style={{ fontSize: '12px' }}>
          it will be logged in your session history
        </div>

        {error && <div className="text-red mt-sm">{error}</div>}
      </div>

      <TuiFooter actions={[
        { key: 'enter', label: phase === 'confirm' ? 'add exception' : 'next' },
        { key: 'esc', label: 'cancel' },
      ]} />
    </TuiBox>
  )
}

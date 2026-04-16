import { useState, useEffect, useMemo } from 'react'
import { TuiBox, TuiHeader, TuiFooter, TuiInput } from '../components'
import { useKeyboard } from '../hooks'
import { GetWorkspace, SealWorkspace } from '../../wailsjs/go/main/App'
import { workspace } from '../../wailsjs/go/models'

/*
 * SealConfig — Pre-seal confirmation screen (Screen 2 from design spec).
 *
 * Shows workspace summary, lets you set duration + lock type,
 * and asks the activation ramp questions:
 *   "What are you working on?"
 *   "First tiny step?"
 *
 * The activation ramp is an ADHD research-backed technique:
 * the hardest part is starting. By writing down the task AND
 * the very first tiny step, you lower the activation barrier.
 *
 * After Enter, there's a 3-second countdown, then the seal engages.
 */

type LockOption = 'random-text' | 'timer' | 'reboot'

interface SealConfigProps {
  workspaceId: string
  onNavigate: (screen: string) => void
}

export function SealConfig({ workspaceId, onNavigate }: SealConfigProps) {
  const [ws, setWs] = useState<workspace.Workspace | null>(null)
  const [duration, setDuration] = useState(90)
  const [lockType, setLockType] = useState<LockOption>('random-text')
  const [task, setTask] = useState('')
  const [firstStep, setFirstStep] = useState('')
  const [phase, setPhase] = useState<'config' | 'countdown'>('config')
  const [countdown, setCountdown] = useState(3)
  const [activeField, setActiveField] = useState<'task' | 'step'>('task')
  const [error, setError] = useState('')

  useEffect(() => {
    GetWorkspace(workspaceId).then(setWs)
  }, [workspaceId])

  // 3-second countdown before seal
  useEffect(() => {
    if (phase !== 'countdown') return
    if (countdown <= 0) {
      // SEAL!
      SealWorkspace(workspaceId, task, firstStep, lockType, 200, duration)
        .then(() => onNavigate('active-seal'))
        .catch((err) => {
          setError(String(err))
          setPhase('config')
        })
      return
    }
    const timer = setTimeout(() => setCountdown((c) => c - 1), 1000)
    return () => clearTimeout(timer)
  }, [phase, countdown])

  const lockOptions: { type: LockOption; desc: string }[] = [
    { type: 'random-text', desc: 'type 200 random chars to unlock' },
    { type: 'reboot', desc: 'must restart machine to unlock' },
    { type: 'timer', desc: 'cannot unlock until timer expires' },
  ]

  const screenKeys = useMemo(() => ({
    'Escape': () => {
      if (phase === 'countdown') return // no escape during countdown
      onNavigate('dashboard')
    },
    'Enter': () => {
      if (phase === 'countdown') return
      if (activeField === 'task' && task.trim()) {
        setActiveField('step')
      } else if (activeField === 'step' && firstStep.trim() && task.trim()) {
        setPhase('countdown')
      }
    },
  }), [phase, activeField, task, firstStep, onNavigate])
  useKeyboard(screenKeys)

  if (!ws) {
    return (
      <TuiBox>
        <TuiHeader breadcrumb={['silo', 'loading...']} />
        <div className="tui-content text-dim">loading...</div>
        <TuiFooter actions={[]} />
      </TuiBox>
    )
  }

  const siteSummary = (ws.allowed_sites || []).slice(0, 4).join(', ')
    + ((ws.allowed_sites || []).length > 4 ? ` +${(ws.allowed_sites || []).length - 4}` : '')

  return (
    <TuiBox>
      <TuiHeader breadcrumb={['silo', ws.name, 'seal']} />

      <div className="tui-content">
        {phase === 'countdown' ? (
          <div style={{ textAlign: 'center', paddingTop: '48px' }}>
            <div className="tui-timer" style={{ color: 'var(--accent-red)' }}>
              {countdown}
            </div>
            <div className="tui-timer__label mt-md">sealing in {countdown}...</div>
            <div className="text-dim mt-md">no going back</div>
          </div>
        ) : (
          <>
            {/* Workspace summary */}
            <div className="tui-label">workspace summary</div>
            <div className="mb-md" style={{ lineHeight: '1.8' }}>
              <div>
                <span className="text-dim">apps:  </span>
                <span className="text-green">{(ws.allowed_apps || []).join(', ') || 'none'}</span>
              </div>
              <div>
                <span className="text-dim">sites: </span>
                <span className="text-green">{siteSummary || 'none'}</span>
              </div>
              {ws.obsidian_vault && (
                <div>
                  <span className="text-dim">notes: </span>
                  <span>{ws.obsidian_vault}/{ws.obsidian_note}</span>
                </div>
              )}
            </div>

            {/* Duration */}
            <div className="tui-label">duration</div>
            <div className="mb-md flex-row gap-sm" style={{ alignItems: 'center' }}>
              <button
                style={{ background: 'none', border: 'none', color: 'var(--accent-blue)', cursor: 'pointer', fontFamily: 'var(--font-mono)', fontSize: '13px' }}
                onClick={() => setDuration((d) => Math.max(15, d - 15))}
              >◄</button>
              <span className="text-primary" style={{ minWidth: '60px', textAlign: 'center' }}>
                {duration} min
              </span>
              <button
                style={{ background: 'none', border: 'none', color: 'var(--accent-blue)', cursor: 'pointer', fontFamily: 'var(--font-mono)', fontSize: '13px' }}
                onClick={() => setDuration((d) => d + 15)}
              >►</button>
            </div>

            {/* Lock type */}
            <div className="tui-label">lock type</div>
            <div className="mb-md">
              {lockOptions.map((opt) => (
                <div
                  key={opt.type}
                  style={{
                    padding: '2px 8px',
                    cursor: 'pointer',
                    background: lockType === opt.type ? 'var(--bg-tertiary)' : undefined,
                  }}
                  onClick={() => setLockType(opt.type)}
                >
                  <span className="text-blue">{lockType === opt.type ? '▸ ' : '  '}</span>
                  <span className={lockType === opt.type ? 'text-primary' : ''}>{opt.type}</span>
                  <span className="text-dim">  {opt.desc}</span>
                </div>
              ))}
            </div>

            {/* Activation ramp */}
            <div className="tui-label">what are you working on?</div>
            <div className="mb-md">
              <TuiInput
                value={task}
                onChange={setTask}
                placeholder="describe your task"
                autoFocus={activeField === 'task'}
              />
            </div>

            <div className="tui-label">first tiny step?</div>
            <TuiInput
              value={firstStep}
              onChange={setFirstStep}
              placeholder="the very first thing you'll do"
              autoFocus={activeField === 'step'}
            />

            {error && <div className="text-red mt-sm">{error}</div>}
          </>
        )}
      </div>

      <TuiFooter
        actions={
          phase === 'countdown'
            ? []
            : [
                { key: 'enter', label: 'SEAL — no going back' },
                { key: 'esc', label: 'cancel' },
              ]
        }
      />
    </TuiBox>
  )
}

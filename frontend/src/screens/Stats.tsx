import { useState, useEffect, useMemo } from 'react'
import { TuiBox, TuiHeader, TuiFooter, ProgressBar } from '../components'
import { useKeyboard, useNavigation } from '../hooks'
import {
  GetStatsSummary,
  GetRecentSessions,
  ExportStatsJSON,
} from '../../wailsjs/go/main/App'
import { stats } from '../../wailsjs/go/models'

/*
 * Stats — Focus history and streak tracking (Screen 5).
 *
 * Wired to real SQLite data via three bindings:
 *   - GetStatsSummary()   → today/week minutes, streak, totals
 *   - GetRecentSessions() → joined session rows for the scrolling log
 *   - ExportStatsJSON()   → raw dump written to clipboard (MVP export)
 *
 * The streak is visualized as the classic TUI dot pattern: filled for
 * days you focused, empty for days you skipped.
 */

interface StatsProps {
  onNavigate: (screen: string) => void
}

const EMPTY_SUMMARY: stats.Summary = {
  today_minutes: 0,
  week_minutes: 0,
  streak_days: 0,
  total_sessions: 0,
  total_focus_minutes: 0,
}

export function Stats({ onNavigate }: StatsProps) {
  const [summary, setSummary] = useState<stats.Summary>(EMPTY_SUMMARY)
  const [sessions, setSessions] = useState<stats.Session[]>([])
  const [loading, setLoading] = useState(true)
  const [exportMsg, setExportMsg] = useState<string>('')

  useEffect(() => {
    Promise.all([GetStatsSummary(), GetRecentSessions(50)])
      .then(([sum, recent]) => {
        setSummary(sum || EMPTY_SUMMARY)
        setSessions(recent || [])
      })
      .catch((err) => console.error('stats load failed:', err))
      .finally(() => setLoading(false))
  }, [])

  const { selectedIndex } = useNavigation({
    itemCount: sessions.length,
    onSelect: () => {}, // details screen deferred
  })

  const handleExport = async () => {
    try {
      const json = await ExportStatsJSON()
      // MVP: write to clipboard. A save dialog belongs in a later polish pass.
      await navigator.clipboard.writeText(json)
      setExportMsg(`copied ${json.length} bytes to clipboard`)
      setTimeout(() => setExportMsg(''), 3000)
    } catch (err) {
      console.error('export failed:', err)
      setExportMsg('export failed — check console')
    }
  }

  const screenKeys = useMemo(
    () => ({
      Escape: () => onNavigate('dashboard'),
      x: handleExport,
    }),
    [onNavigate],
  )
  useKeyboard(screenKeys)

  return (
    <TuiBox>
      <TuiHeader breadcrumb={['silo', 'stats']} />

      <div className="tui-content">
        <div className="tui-label">focus summary</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginBottom: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            <span className="tui-label" style={{ width: '80px', marginBottom: 0 }}>today</span>
            <ProgressBar
              filled={Math.min(12, Math.round((summary.today_minutes / 480) * 12))}
              total={12}
              label={formatMinutes(summary.today_minutes)}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            <span className="tui-label" style={{ width: '80px', marginBottom: 0 }}>this week</span>
            <ProgressBar
              filled={Math.min(12, Math.round((summary.week_minutes / 2400) * 12))}
              total={12}
              label={formatMinutes(summary.week_minutes)}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            <span className="tui-label" style={{ width: '80px', marginBottom: 0 }}>streak</span>
            <span className="text-green">{streakDots(summary.streak_days)} {summary.streak_days} days</span>
          </div>
        </div>

        <hr className="tui-divider" />

        <div className="tui-label">recent sessions</div>
        {loading ? (
          <div className="text-dim">loading...</div>
        ) : sessions.length === 0 ? (
          <div className="text-dim" style={{ padding: '8px 0' }}>
            no sessions yet. seal a workspace to start tracking.
          </div>
        ) : (
          <div style={{ maxHeight: '180px', overflowY: 'auto' }}>
            {sessions.map((s, i) => (
              <div
                key={s.id}
                className={i === selectedIndex ? 'text-primary' : ''}
                style={{ display: 'flex', gap: '12px', padding: '2px 0' }}
              >
                <span className="text-dim" style={{ width: '100px' }}>
                  {formatWhen(s.started_at)}
                </span>
                <span style={{ width: '60px', textAlign: 'right' }}>
                  {formatDuration(s.duration_actual)}
                </span>
                <span className="text-dim" style={{ width: '140px' }}>
                  {s.workspace_name}
                </span>
                <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {s.commit_message || s.task_description}
                </span>
                {s.status !== 'completed' && (
                  <span className="text-yellow" style={{ fontSize: '11px' }}>
                    {s.status}
                  </span>
                )}
              </div>
            ))}
          </div>
        )}

        <hr className="tui-divider" />

        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <span className="text-dim" style={{ fontSize: '12px' }}>
            total sessions: {summary.total_sessions} &nbsp; total focus:{' '}
            {formatMinutes(summary.total_focus_minutes)}
          </span>
          {exportMsg && (
            <span className="text-green" style={{ fontSize: '12px' }}>
              {exportMsg}
            </span>
          )}
        </div>
      </div>

      <TuiFooter
        actions={[
          { key: 'j/k', label: 'scroll' },
          { key: 'x', label: 'export json' },
          { key: 'esc', label: 'back' },
        ]}
      />
    </TuiBox>
  )
}

function formatMinutes(min: number): string {
  const h = Math.floor(min / 60)
  const m = min % 60
  return `${h}h ${m.toString().padStart(2, '0')}m`
}

function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60)
  return `${mins}m`
}

// "today 10:30", "yesterday 08:00", "apr 13", etc.
function formatWhen(isoString: string): string {
  if (!isoString) return ''
  const d = new Date(isoString)
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  const yest = new Date(now)
  yest.setDate(yest.getDate() - 1)
  const sameYest = d.toDateString() === yest.toDateString()

  const hhmm = d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })
  if (sameDay) return `today ${hhmm}`
  if (sameYest) return `yest  ${hhmm}`
  return d.toLocaleDateString([], { month: 'short', day: '2-digit' }).toLowerCase()
}

// ●●●●●●●○○○ — filled dots for streak days, capped at 10 for display
function streakDots(days: number): string {
  const cap = 10
  const on = Math.min(cap, Math.max(0, days))
  return '●'.repeat(on) + '○'.repeat(cap - on)
}

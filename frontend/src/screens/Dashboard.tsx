import { useState, useEffect, useMemo, useRef } from 'react'
import { TuiBox, TuiHeader, TuiFooter, TuiList, ProgressBar } from '../components'
import { useNavigation, useKeyboard } from '../hooks'
import { ListWorkspaces, GetStatsSummary, HideWindow } from '../../wailsjs/go/main/App'
import { workspace, stats } from '../../wailsjs/go/models'

/*
 * Dashboard — Home screen, now wired to real SQLite data.
 *
 * On mount, it calls ListWorkspaces() which goes:
 *   React → Wails bridge → Go App.ListWorkspaces() → SQLite → back
 *
 * The empty state ("no workspaces yet") guides new users to create one.
 * Stats are still hardcoded — they'll be wired in Phase 5.
 */

interface DashboardProps {
  onNavigate: (screen: string, workspaceId?: string) => void
}

export function Dashboard({ onNavigate }: DashboardProps) {
  const [workspaces, setWorkspaces] = useState<workspace.Workspace[]>([])
  const [loading, setLoading] = useState(true)
  const [summary, setSummary] = useState<stats.Summary>({
    today_minutes: 0,
    week_minutes: 0,
    streak_days: 0,
    total_sessions: 0,
    total_focus_minutes: 0,
  })

  // Fetch workspaces AND stats summary on mount. Parallel because they're
  // independent queries — no reason to serialize.
  useEffect(() => {
    Promise.all([ListWorkspaces(), GetStatsSummary()])
      .then(([ws, sum]) => {
        setWorkspaces(ws || [])
        if (sum) setSummary(sum)
      })
      .catch((err) => console.error('dashboard load failed:', err))
      .finally(() => setLoading(false))
  }, [])

  const indexRef = useRef(0)

  const { selectedIndex } = useNavigation({
    itemCount: Math.max(workspaces.length, 1), // at least 1 to prevent divide-by-zero
    onSelect: (index) => {
      if (workspaces.length > 0) {
        onNavigate('seal-config', workspaces[index].id)
      }
    },
  })
  indexRef.current = selectedIndex

  const screenKeys = useMemo(() => ({
    'e': () => {
      if (workspaces.length > 0) {
        onNavigate('workspace-editor', workspaces[indexRef.current].id)
      }
    },
    'n': () => onNavigate('workspace-editor'),
    't': () => onNavigate('template-picker'),
    's': () => onNavigate('stats'),
    // Hide the window; silo keeps running in the background (dock/taskbar).
    // Active seals survive. Re-surface by clicking the dock icon.
    'h': () => { HideWindow() },
  }), [onNavigate, workspaces])
  useKeyboard(screenKeys)

  const todayMinutes = summary.today_minutes
  const weekMinutes = summary.week_minutes
  const streak = summary.streak_days

  return (
    <TuiBox>
      <TuiHeader breadcrumb={['silo', 'v0.1.0']} />

      <div className="tui-content">
        <div className="tui-label">workspaces</div>

        {loading ? (
          <div className="text-dim">loading...</div>
        ) : workspaces.length === 0 ? (
          <div className="text-dim" style={{ padding: '8px 0' }}>
            no workspaces yet. press [n] to create or [t] for templates.
          </div>
        ) : (
          <TuiList
            items={workspaces}
            selectedIndex={selectedIndex}
            renderItem={(ws, isSelected) => (
              <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                <span className={isSelected ? 'text-primary' : ''}>
                  {ws.name}
                </span>
                <span style={{ display: 'flex', gap: '16px' }}>
                  <span className="text-dim">
                    {pluralize((ws.allowed_sites || []).length, 'site')} · {pluralize((ws.allowed_apps || []).length, 'app')}
                  </span>
                  <span className="text-dim">○ idle</span>
                </span>
              </div>
            )}
          />
        )}

        <hr className="tui-divider" />

        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            <span className="tui-label" style={{ width: '80px', marginBottom: 0 }}>today</span>
            <ProgressBar
              filled={Math.min(12, Math.round((todayMinutes / 480) * 12))}
              total={12}
              label={formatMinutes(todayMinutes) + ' focused'}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            <span className="tui-label" style={{ width: '80px', marginBottom: 0 }}>this week</span>
            <ProgressBar
              filled={Math.min(12, Math.round((weekMinutes / 2400) * 12))}
              total={12}
              label={formatMinutes(weekMinutes) + ' focused'}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            <span className="tui-label" style={{ width: '80px', marginBottom: 0 }}>streak</span>
            <span className="text-green">{streak} days</span>
          </div>
        </div>
      </div>

      <TuiFooter
        actions={[
          // enter/edit only make sense with at least one workspace — hide them
          // on empty state so the footer doesn't advertise dead keys.
          ...(workspaces.length > 0
            ? [
                { key: 'enter', label: 'seal' },
                { key: 'e', label: 'edit' },
              ]
            : []),
          { key: 'n', label: 'new' },
          { key: 't', label: 'templates' },
          { key: 's', label: 'stats' },
          { key: 'h', label: 'hide' },
          { key: 'q', label: 'quit' },
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

function pluralize(n: number, noun: string): string {
  return `${n} ${noun}${n === 1 ? '' : 's'}`
}

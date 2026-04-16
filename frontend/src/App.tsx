import { useState, useCallback } from 'react'
import { Dashboard } from './screens/Dashboard'
import { TemplatePicker } from './screens/TemplatePicker'
import { WorkspaceEditor } from './screens/WorkspaceEditor'
import { SealConfig } from './screens/SealConfig'
import { ActiveSeal } from './screens/ActiveSeal'
import { UnlockAttempt } from './screens/UnlockAttempt'
import { QuickException } from './screens/QuickException'
import { Stats } from './screens/Stats'

/*
 * App — Root component and screen router.
 *
 * All 8 screens from the design spec are now wired up:
 *   1. Dashboard (home)
 *   2. SealConfig (pre-seal)
 *   3. ActiveSeal (locked with timer)
 *   4. WorkspaceEditor (add/edit workspace)
 *   5. Stats (focus history)
 *   6. UnlockAttempt (type chars to break seal)
 *   7. QuickException (friction-gated temporary allow)
 *   8. TemplatePicker (choose a template)
 */

type Screen =
  | 'dashboard'
  | 'seal-config'
  | 'active-seal'
  | 'workspace-editor'
  | 'template-picker'
  | 'stats'
  | 'unlock-attempt'
  | 'quick-exception'

export default function App() {
  const [screen, setScreen] = useState<Screen>('dashboard')
  const [selectedId, setSelectedId] = useState<string | undefined>()

  const handleNavigate = useCallback((target: string, id?: string) => {
    setSelectedId(id)
    setScreen(target as Screen)
  }, [])

  switch (screen) {
    case 'template-picker':
      return <TemplatePicker onNavigate={handleNavigate} />
    case 'workspace-editor':
      return <WorkspaceEditor workspaceId={selectedId} onNavigate={handleNavigate} />
    case 'seal-config':
      return <SealConfig workspaceId={selectedId!} onNavigate={handleNavigate} />
    case 'active-seal':
      return <ActiveSeal onNavigate={handleNavigate} />
    case 'unlock-attempt':
      return <UnlockAttempt onNavigate={handleNavigate} />
    case 'quick-exception':
      return <QuickException onNavigate={handleNavigate} />
    case 'stats':
      return <Stats onNavigate={handleNavigate} />
    case 'dashboard':
    default:
      return <Dashboard onNavigate={handleNavigate} />
  }
}

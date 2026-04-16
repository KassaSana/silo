import { useState, useEffect, useMemo } from 'react'
import { TuiBox, TuiHeader, TuiFooter, TuiInput } from '../components'
import { useKeyboard } from '../hooks'
import {
  GetWorkspace,
  CreateWorkspace,
  UpdateWorkspace,
  DeleteWorkspace,
} from '../../wailsjs/go/main/App'

/*
 * WorkspaceEditor — Add/edit apps, sites, and Obsidian config.
 *
 * DESIGN SPEC (Screen 4):
 *   ALLOWED APPS (3)
 *     checkmark VS Code    checkmark Terminal    checkmark Chrome
 *   ALLOWED SITES (5)
 *     checkmark localhost:*    checkmark react.dev
 *   OBSIDIAN
 *     vault: CS
 *     note:  projects/react-project-log
 *
 * MODES:
 *   - If workspaceId is provided → edit existing workspace
 *   - If not → create new (asks for name first)
 *
 * ADDING ITEMS: Press [a], type the app/site name, Enter to add.
 * DELETING: Navigate to item, press [d] to remove.
 *
 * Sections are navigated with Tab key.
 */

type Section = 'apps' | 'sites' | 'obsidian'

interface WorkspaceEditorProps {
  workspaceId?: string
  onNavigate: (screen: string) => void
}

export function WorkspaceEditor({ workspaceId, onNavigate }: WorkspaceEditorProps) {
  const [name, setName] = useState('')
  const [apps, setApps] = useState<string[]>([])
  const [sites, setSites] = useState<string[]>([])
  const [obsVault, setObsVault] = useState('')
  const [obsNote, setObsNote] = useState('')
  const [section, setSection] = useState<Section>('apps')
  const [adding, setAdding] = useState(false)
  const [addValue, setAddValue] = useState('')
  const [selectedItem, setSelectedItem] = useState(0)
  const [needsName, setNeedsName] = useState(!workspaceId)
  const [loading, setLoading] = useState(!!workspaceId)

  // Load existing workspace data
  useEffect(() => {
    if (workspaceId) {
      GetWorkspace(workspaceId)
        .then((ws) => {
          setName(ws.name)
          setApps(ws.allowed_apps || [])
          setSites(ws.allowed_sites || [])
          setObsVault(ws.obsidian_vault || '')
          setObsNote(ws.obsidian_note || '')
        })
        .finally(() => setLoading(false))
    }
  }, [workspaceId])

  const currentItems = section === 'apps' ? apps : section === 'sites' ? sites : []

  // Save to backend
  const save = async () => {
    if (workspaceId) {
      await UpdateWorkspace(workspaceId, name, apps, sites, obsVault, obsNote)
    } else {
      await CreateWorkspace(name, apps, sites, obsVault, obsNote, '')
    }
    onNavigate('dashboard')
  }

  const screenKeys = useMemo(() => ({
    'Escape': () => {
      if (adding) {
        setAdding(false)
        setAddValue('')
      } else {
        onNavigate('dashboard')
      }
    },
    'Tab': () => {
      if (!adding) {
        const sections: Section[] = ['apps', 'sites', 'obsidian']
        const idx = sections.indexOf(section)
        setSection(sections[(idx + 1) % sections.length])
        setSelectedItem(0)
      }
    },
    'a': () => {
      if (!adding && section !== 'obsidian') {
        setAdding(true)
        setAddValue('')
      }
    },
    'd': () => {
      if (!adding && section !== 'obsidian' && currentItems.length > 0) {
        if (section === 'apps') {
          setApps((prev) => prev.filter((_, i) => i !== selectedItem))
        } else {
          setSites((prev) => prev.filter((_, i) => i !== selectedItem))
        }
        setSelectedItem((prev) => Math.max(0, prev - 1))
      }
    },
    'j': () => {
      if (!adding && section !== 'obsidian') {
        setSelectedItem((i) => Math.min(i + 1, currentItems.length - 1))
      }
    },
    'k': () => {
      if (!adding && section !== 'obsidian') {
        setSelectedItem((i) => Math.max(0, i - 1))
      }
    },
    'ArrowDown': () => {
      if (!adding && section !== 'obsidian') {
        setSelectedItem((i) => Math.min(i + 1, currentItems.length - 1))
      }
    },
    'ArrowUp': () => {
      if (!adding && section !== 'obsidian') {
        setSelectedItem((i) => Math.max(0, i - 1))
      }
    },
    'Enter': () => {
      if (adding && addValue.trim()) {
        if (section === 'apps') {
          setApps((prev) => [...prev, addValue.trim()])
        } else if (section === 'sites') {
          setSites((prev) => [...prev, addValue.trim()])
        }
        setAdding(false)
        setAddValue('')
      }
    },
    'ctrl+s': () => { save() },
  }), [adding, section, selectedItem, currentItems, addValue, onNavigate, save])
  useKeyboard(screenKeys)

  if (loading) {
    return (
      <TuiBox>
        <TuiHeader breadcrumb={['silo', 'loading...']} />
        <div className="tui-content text-dim">loading workspace...</div>
        <TuiFooter actions={[]} />
      </TuiBox>
    )
  }

  // Name input phase for new workspaces
  if (needsName) {
    return (
      <TuiBox>
        <TuiHeader breadcrumb={['silo', 'new workspace']} />
        <div className="tui-content">
          <div className="tui-label">workspace name</div>
          <TuiInput
            value={name}
            onChange={setName}
            placeholder="my-workspace"
            autoFocus
            onSubmit={() => {
              if (name.trim()) setNeedsName(false)
            }}
          />
        </div>
        <TuiFooter actions={[
          { key: 'enter', label: 'continue' },
          { key: 'esc', label: 'cancel' },
        ]} />
      </TuiBox>
    )
  }

  return (
    <TuiBox>
      <TuiHeader breadcrumb={['silo', name, 'edit']} />

      <div className="tui-content">
        {/* Apps section */}
        <div
          className="tui-label"
          style={{ color: section === 'apps' ? 'var(--accent-blue)' : undefined }}
        >
          allowed apps ({apps.length})
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginBottom: '16px' }}>
          {apps.map((app, i) => (
            <span
              key={i}
              className={section === 'apps' && i === selectedItem ? 'text-primary' : 'text-green'}
              style={{
                background: section === 'apps' && i === selectedItem ? 'var(--bg-tertiary)' : undefined,
                padding: '2px 8px',
              }}
            >
              ✓ {app}
            </span>
          ))}
          {apps.length === 0 && <span className="text-dim">none — press [a] to add</span>}
        </div>

        {/* Sites section */}
        <div
          className="tui-label"
          style={{ color: section === 'sites' ? 'var(--accent-blue)' : undefined }}
        >
          allowed sites ({sites.length})
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginBottom: '16px' }}>
          {sites.map((site, i) => (
            <span
              key={i}
              className={section === 'sites' && i === selectedItem ? 'text-primary' : 'text-green'}
              style={{
                background: section === 'sites' && i === selectedItem ? 'var(--bg-tertiary)' : undefined,
                padding: '2px 8px',
              }}
            >
              ✓ {site}
            </span>
          ))}
          {sites.length === 0 && <span className="text-dim">none — press [a] to add</span>}
        </div>

        {/* Obsidian section */}
        <div
          className="tui-label"
          style={{ color: section === 'obsidian' ? 'var(--accent-blue)' : undefined }}
        >
          obsidian
        </div>
        {section === 'obsidian' ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span className="text-dim" style={{ width: '48px' }}>vault:</span>
              <TuiInput value={obsVault} onChange={setObsVault} placeholder="vault name" />
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span className="text-dim" style={{ width: '48px' }}>note:</span>
              <TuiInput value={obsNote} onChange={setObsNote} placeholder="path/to/note" />
            </div>
          </div>
        ) : (
          <div className="text-dim" style={{ marginBottom: '16px' }}>
            {obsVault ? `${obsVault}/${obsNote}` : 'not configured'}
          </div>
        )}

        <hr className="tui-divider" />
        <div className="text-dim" style={{ fontSize: '12px' }}>
          everything not listed above is BLOCKED during seal
        </div>

        {/* Add input overlay */}
        {adding && (
          <div style={{ marginTop: '16px' }}>
            <div className="tui-label">
              add {section === 'apps' ? 'app' : 'site'}
            </div>
            <TuiInput
              value={addValue}
              onChange={setAddValue}
              placeholder={section === 'apps' ? 'App Name' : 'domain.com'}
              autoFocus
            />
          </div>
        )}
      </div>

      <TuiFooter
        actions={
          adding
            ? [
                { key: 'enter', label: 'add' },
                { key: 'esc', label: 'cancel' },
              ]
            : [
                { key: 'a', label: 'add' },
                { key: 'd', label: 'delete' },
                { key: 'tab', label: 'section' },
                { key: 'ctrl+s', label: 'save' },
                { key: 'esc', label: 'back' },
              ]
        }
      />
    </TuiBox>
  )
}

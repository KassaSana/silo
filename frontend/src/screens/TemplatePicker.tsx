import { useState, useEffect, useMemo } from 'react'
import { TuiBox, TuiHeader, TuiFooter, TuiList, TuiInput } from '../components'
import { useNavigation, useKeyboard } from '../hooks'
import { GetTemplates, CreateFromTemplate } from '../../wailsjs/go/main/App'
import { workspace } from '../../wailsjs/go/models'

/*
 * TemplatePicker — Choose a template, name it, create workspace.
 *
 * FLOW:
 *   1. Pick a template from the list (j/k + Enter)
 *   2. Type a name for the new workspace
 *   3. Enter → creates workspace from template → navigates to editor
 *
 * Two-phase UI: first you see the template list, then after selecting
 * one, a name input appears. This keeps it simple — one decision at a time.
 */

interface TemplatePickerProps {
  onNavigate: (screen: string, workspaceId?: string) => void
}

export function TemplatePicker({ onNavigate }: TemplatePickerProps) {
  const [templates, setTemplates] = useState<workspace.Template[]>([])
  const [phase, setPhase] = useState<'pick' | 'name'>('pick')
  const [selectedTemplate, setSelectedTemplate] = useState<workspace.Template | null>(null)
  const [name, setName] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    GetTemplates().then(setTemplates)
  }, [])

  const { selectedIndex } = useNavigation({
    itemCount: templates.length || 1,
    onSelect: (index) => {
      if (phase === 'pick' && templates[index]) {
        setSelectedTemplate(templates[index])
        setName(templates[index].name)
        setPhase('name')
      }
    },
  })

  const screenKeys = useMemo(() => ({
    'Escape': () => {
      if (phase === 'name') {
        setPhase('pick')
        setError('')
      } else {
        onNavigate('dashboard')
      }
    },
    'Enter': () => {
      if (phase === 'name' && selectedTemplate && name.trim()) {
        CreateFromTemplate(selectedTemplate.name, name.trim())
          .then((ws) => onNavigate('workspace-editor', ws.id))
          .catch((err) => setError(String(err)))
      }
    },
  }), [phase, selectedTemplate, name, onNavigate])
  useKeyboard(screenKeys)

  return (
    <TuiBox>
      <TuiHeader breadcrumb={['silo', 'new workspace', 'templates']} />

      <div className="tui-content">
        {phase === 'pick' ? (
          <>
            <div className="tui-label">choose a template</div>
            <TuiList
              items={templates}
              selectedIndex={selectedIndex}
              renderItem={(t, isSelected) => (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                  <span className={isSelected ? 'text-primary' : ''}>
                    {t.name}
                  </span>
                  <span className="text-dim" style={{ fontSize: '12px', paddingLeft: '16px' }}>
                    {t.apps.length > 0 ? t.apps.join(', ') : '(nothing allowed)'}
                    {t.sites.length > 0 && (
                      <><br />&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;{t.sites.join(', ')}</>
                    )}
                  </span>
                </div>
              )}
            />
          </>
        ) : (
          <>
            <div className="tui-label">workspace name</div>
            <div className="mb-md">
              <span className="text-dim">template: </span>
              <span className="text-green">{selectedTemplate?.name}</span>
            </div>
            <TuiInput
              value={name}
              onChange={setName}
              placeholder="my-workspace"
              autoFocus
            />
            {error && (
              <div className="text-red mt-sm" style={{ fontSize: '12px' }}>
                {error}
              </div>
            )}
          </>
        )}
      </div>

      <TuiFooter
        actions={
          phase === 'pick'
            ? [
                { key: 'enter', label: 'select' },
                { key: 'esc', label: 'cancel' },
              ]
            : [
                { key: 'enter', label: 'create' },
                { key: 'esc', label: 'back' },
              ]
        }
      />
    </TuiBox>
  )
}

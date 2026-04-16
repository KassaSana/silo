/*
 * TuiFooter — Keyboard shortcut bar at the bottom of every screen.
 *
 * This is how TUI apps tell you what you can do. Instead of buttons,
 * you see: [enter] seal  [e] edit  [n] new  [q] quit
 *
 * The key is highlighted in blue, the action is dim gray.
 * This pattern comes from apps like htop, midnight commander, nano.
 */

interface FooterAction {
  key: string    // e.g. "enter", "e", "esc"
  label: string  // e.g. "seal", "edit", "back"
}

interface TuiFooterProps {
  actions: FooterAction[]
}

export function TuiFooter({ actions }: TuiFooterProps) {
  return (
    <div className="tui-footer">
      {actions.map((action) => (
        <div key={action.key} className="tui-footer__action">
          <span className="tui-footer__key">[{action.key}]</span>
          <span>{action.label}</span>
        </div>
      ))}
    </div>
  )
}

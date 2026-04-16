/*
 * ProgressBar — Block-character style progress indicator.
 *
 * The design spec shows: ████████░░░░  2h 14m focused
 * We render this with CSS segments instead of actual Unicode blocks,
 * because CSS gives us better sizing control across fonts.
 *
 * `filled` and `total` define the ratio. `label` is the text after.
 */

interface ProgressBarProps {
  filled: number
  total: number
  label: string
}

export function ProgressBar({ filled, total, label }: ProgressBarProps) {
  return (
    <div className="tui-progress">
      <div className="tui-progress__bar">
        {Array.from({ length: total }, (_, i) => (
          <div
            key={i}
            className={`tui-progress__segment ${
              i < filled ? 'tui-progress__segment--filled' : ''
            }`}
          />
        ))}
      </div>
      <span className="tui-progress__text">{label}</span>
    </div>
  )
}

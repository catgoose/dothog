// setup:feature:demo

package views

// NumTile holds the display state for a single numerical KPI tile.
type NumTile struct {
	ID         string
	Title      string
	Value      string // formatted primary value
	Delta      string // change indicator text
	Subtitle   string // optional context line
	Color      string // "success", "warning", "error", "info"
	Scale      string // "ms", "s", "min" — determines slider unit
	IntervalMs int    // update interval in milliseconds (canonical)
	DeltaUp    bool   // true = positive direction
	Neutral    bool   // delta is informational, not good/bad
	Pinned     bool   // tile is pinned (excluded from master override)
}

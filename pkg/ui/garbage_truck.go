package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Animation color palette
var (
	truckBodyColor    = lipgloss.Color("#2ed573") // Bright green body
	truckFrameColor   = lipgloss.Color("#1e8449") // Dark green fill
	truckCabColor     = lipgloss.Color("#87ceeb") // Light blue cab window
	truckWheelColor   = lipgloss.Color("#a0a0a0") // Gray wheels
	truckExhaustColor = lipgloss.Color("#555555") // Dark exhaust
	truckExhaustLight = lipgloss.Color("#777777") // Light exhaust
	truckBinColor     = lipgloss.Color("#27ae60") // Bin green
	truckBinFillColor = lipgloss.Color("#8b6914") // Garbage brown
	truckWarnColor    = lipgloss.Color("#ffdd57") // Warning yellow
	truckAlertColor   = lipgloss.Color("#ff6b6b") // Alert red
	truckRoadColor    = lipgloss.Color("#3a3a3a") // Road gray
	truckDoneColor    = lipgloss.Color("#5fd787") // Done green
)

// GarbageTruckAnimation garbage truck idle animation
type GarbageTruckAnimation struct {
	frame     int
	phase     int
	phaseTime int
}

func NewGarbageTruckAnimation() *GarbageTruckAnimation {
	return &GarbageTruckAnimation{}
}

func (g *GarbageTruckAnimation) Update() {
	g.frame++
	g.phaseTime++
	switch g.phase {
	case 0:
		if g.phaseTime > 40 {
			g.phase = 1
			g.phaseTime = 0
		}
	case 1:
		if g.phaseTime > 25 {
			g.phase = 2
			g.phaseTime = 0
		}
	case 2:
		if g.phaseTime > 50 {
			g.phase = 3
			g.phaseTime = 0
		}
	case 3:
		if g.phaseTime > 35 {
			g.phase = 0
			g.phaseTime = 0
		}
	}
}

func (g *GarbageTruckAnimation) wheelChar() string {
	if g.frame%4 < 2 {
		return "●"
	}
	return "○"
}

func (g *GarbageTruckAnimation) Draw(width int) string {
	if width < 50 {
		return ""
	}
	lines := make([]string, 8)
	for i := range lines {
		lines[i] = strings.Repeat(" ", width)
	}
	truckX := g.getTruckX(width)
	binX := width - 15

	// Road
	g.renderRoad(lines, width)

	// Bin (filling during dump/away phases)
	switch g.phase {
	case 2:
		fillLevel := int(float64(g.phaseTime) / 50.0 * 3)
		if fillLevel > 3 {
			fillLevel = 3
		}
		g.renderBinFilling(lines, binX, fillLevel)
	case 3:
		g.renderBinFilling(lines, binX, 3) // full
	default:
		g.renderBin(lines, binX)
	}

	// Truck by phase
	switch g.phase {
	case 0:
		g.renderTruckDrivingIn(lines, truckX)
	case 1:
		g.renderTruckReversing(lines, truckX)
	case 2:
		g.renderTruckDumping(lines, truckX, binX)
	case 3:
		g.renderTruckDrivingAway(lines, truckX)
	}

	return strings.Join(lines, "\n")
}

func (g *GarbageTruckAnimation) getTruckX(width int) int {
	switch g.phase {
	case 0:
		progress := float64(g.phaseTime) / 40
		return int(-15 + progress*float64(width-50))
	case 1:
		progress := float64(g.phaseTime) / 25
		baseX := width - 45
		return baseX + int(progress*5)
	case 2:
		return width - 45
	case 3:
		progress := float64(g.phaseTime) / 35
		baseX := width - 45
		return baseX + int(progress*float64(width+20))
	}
	return 0
}

// ─── Road ────────────────────────────────────────────────────

func (g *GarbageTruckAnimation) renderRoad(lines []string, width int) {
	rs := lipgloss.NewStyle().Foreground(truckRoadColor)
	var road strings.Builder
	for x := 0; x < width; x++ {
		if x%8 < 4 {
			road.WriteRune('─')
		} else {
			road.WriteRune(' ')
		}
	}
	lines[6] = rs.Render(road.String())
}

// ─── Bin ─────────────────────────────────────────────────────

func (g *GarbageTruckAnimation) renderBin(lines []string, x int) {
	bc := lipgloss.NewStyle().Foreground(truckBinColor)
	bin := []string{
		"┌─────────┐",
		"│  TRASH  │",
		"│         │",
		"│         │",
		"│         │",
		"└─────────┘",
	}
	for i, line := range bin {
		if y := 1 + i; y < len(lines) {
			lines[y] = overlayStringAt(lines[y], x, bc.Render(line))
		}
	}
}

func (g *GarbageTruckAnimation) renderBinFilling(lines []string, x int, fillLevel int) {
	bc := lipgloss.NewStyle().Foreground(truckBinColor)
	fc := lipgloss.NewStyle().Foreground(truckBinFillColor)

	binArt := make([]string, 6)
	binArt[0] = bc.Render("┌─────────┐")
	binArt[1] = bc.Render("│  TRASH  │")

	// 3 fill rows (indices 2, 3, 4) — fill from bottom up
	for i := 0; i < 3; i++ {
		if i >= 3-fillLevel {
			binArt[2+i] = bc.Render("│") + fc.Render("▓▓▓▓▓▓▓▓▓") + bc.Render("│")
		} else {
			binArt[2+i] = bc.Render("│         │")
		}
	}
	binArt[5] = bc.Render("└─────────┘")

	for i, line := range binArt {
		if y := 1 + i; y < len(lines) {
			lines[y] = overlayStringAt(lines[y], x, line)
		}
	}
}

// ─── Truck builder ───────────────────────────────────────────

// buildTruckLines creates multi-colored truck art with given label and label color
//
//	   ┌──────────────┬─────┐         (line 0, 25 chars)
//	   │  RECYCLE     │ ▒▒  │         (line 1, 25 chars)
//	   │              │     │         (line 2, 25 chars)
//	┌──┴──────────────┴─────┴──┐      (line 3, 28 chars)
//	│░░░░░░░░░░░░░░░░░░░░░░░░░░│      (line 4, 28 chars)
//	└──●──────────●──●─────●───┘      (line 5, 28 chars)
func (g *GarbageTruckAnimation) buildTruckLines(label string, labelColor lipgloss.Color) []string {
	bc := lipgloss.NewStyle().Foreground(truckBodyColor)
	fc := lipgloss.NewStyle().Foreground(truckFrameColor)
	wc := lipgloss.NewStyle().Foreground(truckCabColor)
	gc := lipgloss.NewStyle().Foreground(truckWheelColor)
	lc := lipgloss.NewStyle().Foreground(labelColor)

	w := g.wheelChar()

	return []string{
		bc.Render("   ┌──────────────┬─────┐"),
		bc.Render("   │") + lc.Render(padRight("  "+label, 14)) + bc.Render("│") + wc.Render(" ▒▒  ") + bc.Render("│"),
		bc.Render("   │              │     │"),
		bc.Render("┌──┴──────────────┴─────┴──┐"),
		bc.Render("│") + fc.Render("░░░░░░░░░░░░░░░░░░░░░░░░░░") + bc.Render("│"),
		bc.Render("└──") + gc.Render(w) + bc.Render("──────────") + gc.Render(w) + bc.Render("──") + gc.Render(w) + bc.Render("─────") + gc.Render(w) + bc.Render("───┘"),
	}
}

func (g *GarbageTruckAnimation) overlayTruck(lines []string, x int, truck []string) {
	for i, line := range truck {
		if i < len(lines)-1 {
			lines[i] = overlayStringAt(lines[i], x, line)
		}
	}
}

// ─── Phase renderers ─────────────────────────────────────────

func (g *GarbageTruckAnimation) renderTruckDrivingIn(lines []string, x int) {
	truck := g.buildTruckLines("RECYCLE", lipgloss.Color("#ffffff"))
	g.overlayTruck(lines, x, truck)

	// Exhaust smoke trailing behind the truck
	if x > 5 {
		es := lipgloss.NewStyle().Foreground(truckExhaustColor)
		el := lipgloss.NewStyle().Foreground(truckExhaustLight)
		switch g.frame % 8 {
		case 0, 1:
			lines[4] = overlayStringAt(lines[4], x-2, es.Render("░░"))
		case 2, 3:
			lines[4] = overlayStringAt(lines[4], x-3, el.Render("░░░"))
			lines[3] = overlayStringAt(lines[3], x-2, es.Render("░"))
		case 4, 5:
			lines[4] = overlayStringAt(lines[4], x-2, es.Render("░"))
			lines[3] = overlayStringAt(lines[3], x-3, el.Render("░"))
		case 6, 7:
			lines[4] = overlayStringAt(lines[4], x-4, es.Render("░░"))
			lines[3] = overlayStringAt(lines[3], x-2, el.Render("░░"))
		}
	}
}

func (g *GarbageTruckAnimation) renderTruckReversing(lines []string, x int) {
	var labelColor lipgloss.Color
	if g.frame%6 < 3 {
		labelColor = truckWarnColor
	} else {
		labelColor = truckAlertColor
	}
	truck := g.buildTruckLines("REVERSE!", labelColor)
	g.overlayTruck(lines, x, truck)

	// Blinking warning indicator at rear (left side)
	if g.frame%4 < 2 && x > 2 {
		wl := lipgloss.NewStyle().Foreground(truckWarnColor).Bold(true)
		lines[3] = overlayStringAt(lines[3], x-2, wl.Render("◀"))
		lines[4] = overlayStringAt(lines[4], x-2, wl.Render("◀"))
	}
}

func (g *GarbageTruckAnimation) renderTruckDumping(lines []string, truckX, binX int) {
	progress := float64(g.phaseTime) / 50.0

	var label string
	var labelColor lipgloss.Color
	if progress < 0.7 {
		dots := strings.Repeat(".", (g.frame/3)%4)
		label = "DUMPING" + dots
		labelColor = truckWarnColor
	} else {
		label = "DONE!"
		labelColor = truckDoneColor
	}

	// Shake effect while actively dumping
	shake := 0
	if progress > 0.1 && progress < 0.7 && g.frame%4 < 2 {
		shake = 1
	}

	truck := g.buildTruckLines(label, labelColor)
	g.overlayTruck(lines, truckX+shake, truck)

	// Garbage particles flying from truck toward bin
	if progress > 0.1 && progress < 0.75 {
		gc := lipgloss.NewStyle().Foreground(truckBinFillColor)
		rightEdge := truckX + 25
		numParticles := int((progress - 0.1) / 0.65 * 6)
		particleChars := []string{"▪", "▫", "▪", "▫", "▪", "▫"}

		for i := 0; i < numParticles && i < 6; i++ {
			px := rightEdge + (i % 3)
			py := 2 + (i+g.frame/2)%3
			if px > 0 && px < binX && py >= 1 && py <= 5 {
				lines[py] = overlayStringAt(lines[py], px, gc.Render(particleChars[i]))
			}
		}
	}
}

func (g *GarbageTruckAnimation) renderTruckDrivingAway(lines []string, x int) {
	truck := g.buildTruckLines("BYE~", truckDoneColor)
	g.overlayTruck(lines, x, truck)

	// Exhaust puffs while driving away
	if x > 5 && g.frame%3 == 0 {
		es := lipgloss.NewStyle().Foreground(truckExhaustColor)
		lines[4] = overlayStringAt(lines[4], x-3, es.Render("░░"))
	}
}

// ─── Helpers ─────────────────────────────────────────────────

func overlayStringAt(background string, x int, foreground string) string {
	if x < 0 {
		return background
	}
	bgRunes := []rune(background)
	fgRunes := []rune(stripAnsi(foreground))
	if x >= len(bgRunes) {
		return background
	}
	result := string(bgRunes[:x])
	result += foreground
	if x+len(fgRunes) < len(bgRunes) {
		result += string(bgRunes[x+len(fgRunes):])
	}
	return result
}

func garbageMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func garbageAbs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

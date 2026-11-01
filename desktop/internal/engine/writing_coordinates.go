package engine

import (
	"fmt"
	"math"
)

// Match Chixie's configured StartPointOfFrame: register the pen at the
// writing frame's top-left, inside the margins, without changing the layout.
// Positive UI offsets move ink right/down. Auto origin labels that position
// X0 Y<paper height>; None retains the printer's XY coordinates.
// Z always uses the operator's absolute calibration, independent of XY mode.
func machineTarget(p Point, s LayoutSettings, w WritingSettings) (int64, int64, error) {
	x := p.X - s.MarginLeft + w.OffsetX
	y := s.PaperHeight - (p.Y - s.MarginTop + w.OffsetY)
	if err := validateMachineTarget(x, y, s, w); err != nil {
		return 0, 0, err
	}
	return int64(math.Round(x * 1000)), int64(math.Round(y * 1000)), nil
}

func parkingTarget(s LayoutSettings, w WritingSettings) (int64, int64, error) {
	// Return to the operator-positioned origin, not the bottom of a paper
	// that can be taller than the machine's travel (for example, A4).
	y := 0.0
	if w.OriginMode == "auto" {
		y = s.PaperHeight
	}
	if err := validateMachineTarget(0, y, s, w); err != nil {
		return 0, 0, err
	}
	return 0, int64(math.Round(y * 1000)), nil
}

func validateMachineTarget(x, y float64, s LayoutSettings, w WritingSettings) error {
	minimum := 0.0
	if w.OriginMode == "auto" {
		minimum = -BedSize
		y -= s.PaperHeight // displacement from the operator-positioned pen
	}
	if !finite(x) || !finite(y) || x < minimum || y < minimum || x > BedSize || y > BedSize {
		return fmt.Errorf("书写范围超过 A1 的 256 mm 行程，请调整偏移或排版")
	}
	return nil
}

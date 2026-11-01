package engine

// F is mm/min. A1's per-axis caps still apply, notably Z's 30 mm/s cap.
const motionFeedRate = 16000

// Slow only pen-down table borders; handwriting, travel and pen cycles retain
// their existing feed. Keep the kind attached through cutting and sorting.
const tableGridFeedRate = motionFeedRate * 70 / 100
const strokeKindTableGrid = "table-grid"

// Initialise motion independently of the previous task or speed preset.
// The A1 machine profile specifies XY acceleration 12000 mm/s²,
// Z acceleration 1500 mm/s² and Z speed 30 mm/s. Use the XY limit for
// short handwriting strokes instead of the start template's 6000 reset.
// The inherited XY speed cap is 500 mm/s. E, motor current, bed compensation
// and endstops are untouched.
// Source: bambulab/BambuStudio resources/profiles/BBL/machine/
// Bambu Lab A1 0.4 nozzle.json and its machine_start/end_gcode templates.
const a1MotionSetup = "M220 S100\nM201.2 K1.0\nM201 X12000 Y12000 Z1500\nM203 X500 Y500 Z30\nM204 S12000\n"

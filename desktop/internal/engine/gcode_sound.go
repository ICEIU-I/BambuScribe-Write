package engine

// A1's motor-tone command. E here is a musical channel, not an extrusion axis.
// Queue a short completion cue only after the pen has lifted and parked.
const completionSoundGCode = "M1006 S1\nM1006 A0 B8 L0 C60 D8 M40 E60 F8 N40\nM1006 W\n"

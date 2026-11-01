package engine

import "strconv"

// addSupplementalSymbols supplies the compact, single-line symbols that are
// common in lab notes but are not part of the handwritten Chinese font.
// Coordinates use the same 11-unit cell as the embedded font.
func addSupplementalSymbols(f *Font) {
	put := func(chars string, width float64, strokes ...Stroke) {
		for _, r := range chars {
			if _, exists := f.Glyphs[r]; !exists {
				f.Glyphs[r] = Glyph{Width: width, Strokes: strokes}
			}
		}
	}
	line := func(a, b Point) Stroke { return Stroke{a, b} }
	circle := func(cx, cy, rx, ry float64) Stroke {
		const n = 16
		out := make(Stroke, n+1)
		for i := 0; i <= n; i++ {
			angle := 2 * 3.141592653589793 * float64(i) / n
			out[i] = Point{cx + rx*cosApprox(angle), cy + ry*sinApprox(angle)}
		}
		return out
	}

	addScienceSymbols(put)
	addGreekSymbols(put, line, circle)
	addPhysicsSymbols(put, line, circle)
	addNotationSymbols(put, line, circle)
	addArrowSymbols(put, line)
	addSuperscriptSymbols(f, put)
	addFractionSymbols(f, put, line)
	addCircledNumberSymbols(f, put)
}

// Small local trigonometry helpers keep the generated symbol table free of
// floating point tables while retaining smooth enough circles for a pen plotter.
func sinApprox(x float64) float64 {
	for x > 3.141592653589793 {
		x -= 2 * 3.141592653589793
	}
	for x < -3.141592653589793 {
		x += 2 * 3.141592653589793
	}
	// Five-term Taylor series is accurate to below 0.001 on [-pi, pi].
	x2 := x * x
	return x * (1 - x2/6 + x2*x2/120 - x2*x2*x2/5040 + x2*x2*x2*x2/362880)
}

func cosApprox(x float64) float64 { return sinApprox(x + 1.5707963267948966) }

func addGreekSymbols(
	put func(string, float64, ...Stroke),
	line func(Point, Point) Stroke,
	circle func(float64, float64, float64, float64) Stroke,
) {
	// Uppercase Greek, drawn with the same simple geometric vocabulary used by
	// the rest of the supplemental font.
	put("Α", 7, Stroke{{1, 9}, {1, 2}, {6, 9}}, line(Point{2, 6}, Point{5.2, 6}))
	put("Β", 7, Stroke{{1, 9}, {1, 1.5}, {4.8, 1.5}, {6, 2.5}, {6, 4.4}, {4.8, 5.3}, {1, 5.3}, {4.8, 5.3}, {6, 6.1}, {6, 8}, {4.8, 9}, {1, 9}})
	put("Γ", 7, Stroke{{1, 9}, {1, 1.5}, {6.5, 1.5}})
	put("Ε", 7, Stroke{{6, 1.5}, {1, 1.5}, {1, 9}, {6, 9}}, line(Point{1, 5.3}, Point{5.3, 5.3}))
	put("Ζ", 7, Stroke{{1, 1.5}, {6, 1.5}, {1, 9}, {6, 9}})
	put("Η", 7, Stroke{{1, 1.5}, {1, 9}}, line(Point{6, 1.5}, Point{6, 9}), line(Point{1, 5.3}, Point{6, 5.3}))
	put("Θ", 7, circle(3.5, 5.5, 2.7, 4), line(Point{1.1, 5.5}, Point{5.9, 5.5}))
	put("Ι", 5, line(Point{2.5, 1.5}, Point{2.5, 9}))
	put("Κ", 7, Stroke{{1, 1.5}, {1, 9}}, line(Point{6, 1.5}, Point{1, 5.3}), line(Point{1, 5.3}, Point{6, 9}))
	put("Λ", 7, Stroke{{0.7, 9}, {3.5, 1.2}, {6.3, 9}})
	put("Μ", 8, Stroke{{0.8, 9}, {0.8, 1.5}, {4, 5.2}, {7.2, 1.5}, {7.2, 9}})
	put("Ν", 7, Stroke{{1, 9}, {1, 1.5}, {6, 9}, {6, 1.5}})
	put("Ξ", 7, line(Point{0.8, 1.5}, Point{6.2, 1.5}), line(Point{1.5, 5.3}, Point{5.5, 5.3}), line(Point{0.8, 9}, Point{6.2, 9}))
	put("Ο", 7, circle(3.5, 5.5, 2.7, 4))
	put("Π", 7, Stroke{{1, 9}, {1, 1.5}, {6, 1.5}, {6, 9}})
	put("Ρ", 7, Stroke{{1, 9}, {1, 1.5}, {4.8, 1.5}, {6, 2.6}, {6, 4.5}, {4.8, 5.5}, {1, 5.5}})
	put("Σ", 7, Stroke{{6, 1.5}, {1, 1.5}, {4.8, 5.3}, {1, 9}, {6, 9}})
	put("Τ", 7, line(Point{0.8, 1.5}, Point{6.2, 1.5}), line(Point{3.5, 1.5}, Point{3.5, 9}))
	put("Υ", 7, Stroke{{0.8, 1.5}, {3.5, 5.3}, {6.2, 1.5}}, line(Point{3.5, 5.3}, Point{3.5, 9}))
	put("Φ", 7, circle(3.5, 5.5, 2.7, 3.8), line(Point{3.5, 1}, Point{3.5, 10}))
	put("Χ", 7, line(Point{1, 1.5}, Point{6, 9}), line(Point{6, 1.5}, Point{1, 9}))
	put("Ψ", 8, Stroke{{0.8, 1.5}, {3.8, 5.2}, {6.8, 1.5}}, line(Point{3.8, 5.2}, Point{3.8, 9}), line(Point{2.2, 1.5}, Point{5.4, 1.5}))
	put("℧", 8, circle(3.8, 5.5, 2.7, 4), line(Point{1.1, 8.8}, Point{6.5, 2.2}))

	// Lowercase Greek used in formulas and lab reports.
	put("δ", 7, Stroke{{5.4, 1.7}, {3.8, 1.1}, {2.1, 1.8}, {1.5, 3.3}}, circle(3.5, 6.3, 2.3, 2.6))
	put("εϵ", 6, Stroke{{5.3, 3}, {4.3, 2}, {2, 2}, {1.2, 3.2}, {2.2, 4.7}, {4.8, 4.7}, {2.2, 4.7}, {1.2, 6.2}, {2, 8.3}, {4.7, 8.3}, {5.5, 7.2}})
	put("ζ", 6, Stroke{{5.7, 2}, {1.2, 2}, {4.8, 6.1}, {4.2, 9.2}, {1.3, 9.2}})
	put("η", 7, Stroke{{1, 9}, {1, 3}, {2.2, 2}, {4.7, 2}, {5.8, 3}, {5.8, 9}}, line(Point{5.8, 3}, Point{6.4, 9}))
	put("θϑ", 7, circle(3.5, 5.5, 2.5, 3.7), line(Point{1.3, 5.5}, Point{5.7, 5.5}))
	put("ι", 4, line(Point{2, 3}, Point{2, 9}), line(Point{1.4, 3}, Point{2.6, 3}))
	put("κϰ", 7, line(Point{1.2, 2.5}, Point{1.2, 9}), line(Point{1.2, 6}, Point{5.8, 2.5}), line(Point{2.8, 6}, Point{6, 9}))
	// Lambda descends from an upper-left hook and forks midway down its stem;
	// nu has an upright left stem and a rounded, asymmetric turn into its arc.
	// Their advances and vertical extents stay fixed so existing formulas fit.
	put("λ", 7,
		Stroke{{1, 1.7}, {1.35, 1.3}, {1.8, 1.35}, {2.15, 1.75}, {2.45, 2.35}, {2.75, 3.1}, {3.1, 4}, {3.5, 5.05}, {4, 6.3}, {4.55, 7.55}, {5.05, 8.4}, {5.5, 8.85}, {5.95, 9}, {6.3, 8.75}},
		Stroke{{3.1, 4}, {2.7, 5}, {2.25, 6.05}, {1.75, 7.1}, {1.2, 8.15}, {0.8, 9}})
	put("ν", 7, Stroke{{0.8, 2.65}, {1.2, 2.4}, {1.55, 2.55}, {1.75, 3.15}, {1.8, 4.1}, {1.8, 5.3}, {1.85, 6.5}, {2, 7.6}, {2.25, 8.45}, {2.6, 8.95}, {3.05, 9}, {3.55, 8.6}, {4.1, 7.95}, {4.7, 6.95}, {5.25, 5.8}, {5.7, 4.6}, {6.05, 3.4}, {6.2, 2.4}})
	put("ξ", 7, Stroke{{5.8, 2}, {2, 2}, {1.2, 3.3}, {2.5, 4.6}, {4.8, 4.6}, {2.4, 6.1}, {1.6, 7.3}, {2.6, 9}, {5.8, 9}})
	put("ο", 6, circle(3, 5.5, 2.2, 3))
	put("π", 7, line(Point{0.8, 2.7}, Point{6.2, 2.7}), line(Point{1.8, 2.7}, Point{1.4, 8.8}), line(Point{4.8, 2.7}, Point{5.2, 8.8}))
	put("ρϱ", 7, Stroke{{1.5, 9.5}, {1.5, 3.5}, {2.6, 2.1}, {4.8, 2.1}, {5.8, 3.4}, {5.8, 5.4}, {4.8, 6.8}, {2.4, 6.8}}, line(Point{1.5, 6.5}, Point{1.5, 9.5}))
	put("σ", 7, Stroke{{5.9, 2.6}, {2.2, 2.6}, {1.1, 4}, {1.1, 6.7}, {2.4, 8.2}, {4.8, 8.2}, {5.9, 6.8}, {5.2, 5.2}, {3.4, 5.2}})
	put("ς", 6, Stroke{{5.6, 2.8}, {2.2, 2.8}, {1.1, 4.4}, {1.7, 6.7}, {4.4, 7.9}, {5.5, 7.2}, {3.3, 9.3}})
	put("τ", 7, line(Point{0.8, 2.7}, Point{6.2, 2.7}), Stroke{{3.5, 2.7}, {3.5, 8.8}, {4.7, 9.5}})
	put("υ", 7, Stroke{{1, 2.7}, {1, 6.4}, {2, 8.4}, {4.3, 8.4}, {5.6, 6.4}, {5.6, 2.7}})
	put("φϕ", 7, circle(3.5, 5.5, 2.5, 3.1), line(Point{3.5, 1}, Point{3.5, 10}))
	put("χ", 7, line(Point{1, 2.5}, Point{5.8, 9}), line(Point{5.8, 2.5}, Point{1, 9}))
	put("ψ", 8, Stroke{{0.8, 2.5}, {2.2, 7.3}, {3.8, 8.5}, {5.4, 7.3}, {6.8, 2.5}}, line(Point{3.8, 1.2}, Point{3.8, 10}))
	// Omega has two rounded bowls and a low central rise, with the same ink
	// bounds and advance as before so formulas keep their spacing.
	put("ω", 8, Stroke{{1.5, 3}, {1.18, 3.45}, {0.92, 4.1}, {0.75, 4.85}, {0.7, 5.65}, {0.78, 6.55}, {1, 7.35}, {1.36, 8}, {1.8, 8.37}, {2.3, 8.5}, {2.78, 8.37}, {3.15, 8.02}, {3.42, 7.5}, {3.57, 6.85}, {3.65, 6.15}, {3.75, 5.55}, {3.9, 5.3}, {4.03, 5.6}, {4.08, 6.2}, {4.12, 6.95}, {4.28, 7.6}, {4.6, 8.1}, {5.03, 8.4}, {5.5, 8.5}, {5.98, 8.34}, {6.4, 7.9}, {6.72, 7.15}, {6.88, 6.3}, {6.9, 5.4}, {6.8, 4.5}, {6.57, 3.7}, {6.3, 3.1}})
}

func addPhysicsSymbols(
	put func(string, float64, ...Stroke),
	line func(Point, Point) Stroke,
	circle func(float64, float64, float64, float64) Stroke,
) {
	put("ÅÅ", 7, Stroke{{0.6, 9}, {3.5, 1.4}, {6.4, 9}}, line(Point{1.8, 6.2}, Point{5.2, 6.2}), circle(3.5, 1.8, 1.3, 0.7))
	put("∇", 7, Stroke{{0.7, 2}, {3.5, 9}, {6.3, 2}})
	put("∂", 7, Stroke{{5.8, 2.2}, {4.8, 1.4}, {2.4, 1.7}, {1.2, 3.4}, {1.2, 6.6}, {2.4, 8.5}, {4.8, 8.8}, {5.8, 7.8}}, line(Point{1.2, 5.4}, Point{4.7, 5.4}))
	put("∑", 8, Stroke{{6.8, 1.5}, {1, 1.5}, {5.5, 5.3}, {1, 9}, {6.8, 9}})
	put("∏", 8, Stroke{{1, 1.5}, {1, 9}}, line(Point{1, 1.5}, Point{7, 1.5}), line(Point{7, 1.5}, Point{7, 9}))
	put("∫", 5, Stroke{{3.4, 1}, {2.1, 1.8}, {2.1, 3.5}, {3.5, 5.4}, {3.5, 7.2}, {2.1, 9.3}, {1, 10}})
	put("√", 8, Stroke{{0.6, 6.2}, {1.8, 8.7}, {3.1, 1.8}, {7, 1.8}})
	put("≡", 7, line(Point{0.8, 3}, Point{6.2, 3}), line(Point{0.8, 5.5}, Point{6.2, 5.5}), line(Point{0.8, 8}, Point{6.2, 8}))
	put("∝", 8, Stroke{{1, 3}, {2, 2.2}, {3.6, 3.1}, {5, 5.5}, {6.1, 7.9}, {7, 7}}, Stroke{{1, 8}, {2, 8.8}, {3.6, 7.9}, {5, 5.5}, {6.1, 3.1}, {7, 4}})
	put("∀", 7, Stroke{{0.8, 2}, {3.5, 9}, {6.2, 2}}, line(Point{1.8, 5.8}, Point{5.2, 5.8}))
	put("∃", 7, Stroke{{6.2, 2}, {1, 2}, {1, 9}, {6.2, 9}}, line(Point{1, 5.5}, Point{5.2, 5.5}))
	put("¬", 6, line(Point{0.8, 4}, Point{5.2, 4}), line(Point{0.8, 4}, Point{0.8, 7.5}))
	put("∧", 7, Stroke{{0.8, 8.5}, {3.5, 2}, {6.2, 8.5}})
	put("∨", 7, Stroke{{0.8, 2}, {3.5, 8.5}, {6.2, 2}})
	put("⊂", 7, Stroke{{6.2, 2.2}, {2.2, 2.2}, {1, 5.5}, {2.2, 8.8}, {6.2, 8.8}})
	put("⊃", 7, Stroke{{0.8, 2.2}, {4.8, 2.2}, {6, 5.5}, {4.8, 8.8}, {0.8, 8.8}})
	put("⊆", 7, Stroke{{6.2, 2.2}, {2.2, 2.2}, {1, 5.5}, {2.2, 8.8}, {6.2, 8.8}}, line(Point{1, 5.5}, Point{6.2, 5.5}))
	put("⊇", 7, Stroke{{0.8, 2.2}, {4.8, 2.2}, {6, 5.5}, {4.8, 8.8}, {0.8, 8.8}}, line(Point{0.8, 5.5}, Point{6, 5.5}))
	put("≪", 8, Stroke{{6.5, 2.5}, {1, 5.5}, {6.5, 8.5}}, Stroke{{3.5, 2.5}, {0.8, 5.5}, {3.5, 8.5}})
	put("≫", 8, Stroke{{1, 2.5}, {6.5, 5.5}, {1, 8.5}}, Stroke{{4, 2.5}, {6.7, 5.5}, {4, 8.5}})
	put("∠", 7, Stroke{{1, 8.5}, {1, 2.5}, {6.3, 8.5}}, line(Point{1, 8.5}, Point{3.2, 8.5}))
	put("⊥", 7, line(Point{0.8, 7.8}, Point{6.2, 7.8}), line(Point{3.5, 1.8}, Point{3.5, 7.8}))
	put("∥", 6, line(Point{1.5, 1.8}, Point{0.8, 9}), line(Point{5.2, 1.8}, Point{4.5, 9}))
	put("∼", 7, Stroke{{0.8, 5.8}, {2.1, 4.2}, {3.5, 5.8}, {4.9, 7.4}, {6.2, 5.8}})
	put("∈", 7, Stroke{{6.3, 2.2}, {2, 2.2}, {1, 5.5}, {2, 8.8}, {6.3, 8.8}}, line(Point{1, 5.5}, Point{5.2, 5.5}))
	put("∉", 7, Stroke{{6.3, 2.2}, {2, 2.2}, {1, 5.5}, {2, 8.8}, {6.3, 8.8}}, line(Point{0.8, 8.8}, Point{6.3, 2.2}))
	put("∅", 7, circle(3.5, 5.5, 2.7, 4), line(Point{1.1, 8.7}, Point{5.9, 2.3}))
	put("∪", 7, Stroke{{0.8, 2.2}, {0.8, 7.2}, {2, 8.8}, {5, 8.8}, {6.2, 7.2}, {6.2, 2.2}})
	put("∩", 7, Stroke{{0.8, 8.8}, {0.8, 4.2}, {2, 2.2}, {5, 2.2}, {6.2, 4.2}, {6.2, 8.8}})
	put("⊕", 7, circle(3.5, 5.5, 2.8, 3.8), line(Point{1.1, 5.5}, Point{5.9, 5.5}), line(Point{3.5, 1.8}, Point{3.5, 9.2}))
	put("⊗", 7, circle(3.5, 5.5, 2.8, 3.8), line(Point{1.5, 3}, Point{5.5, 8}), line(Point{5.5, 3}, Point{1.5, 8}))
	put("∴", 5, Stroke{{0.8, 2.5}, {1, 2.5}}, Stroke{{2.3, 5.5}, {2.5, 5.5}}, Stroke{{3.8, 8.5}, {4, 8.5}})
	put("∵", 5, Stroke{{0.8, 8.5}, {1, 8.5}}, Stroke{{2.3, 5.5}, {2.5, 5.5}}, Stroke{{3.8, 2.5}, {4, 2.5}})
	put("⋅•∙", 2, Stroke{{0.8, 5.5}, {1.3, 5.5}})
	put("⋯", 7, Stroke{{0.8, 5.5}, {1.2, 5.5}}, Stroke{{3.2, 5.5}, {3.6, 5.5}}, Stroke{{5.6, 5.5}, {6, 5.5}})
	put("ℏ", 7, Stroke{{1.2, 3}, {1.2, 8.5}, {2.2, 2}, {2.2, 8.5}, {5.8, 8.5}}, line(Point{0.8, 5.3}, Point{6.4, 5.3}))
	put("ℓ", 4, Stroke{{3.2, 1.5}, {2.1, 2.4}, {2.1, 7.8}, {1.1, 9}, {0.8, 8.3}})
	put("℉", 8, circle(1.2, 2.2, 1, 1), Stroke{{7, 2}, {3.3, 2}, {3.3, 9}, {7, 9}}, line(Point{3.3, 5.3}, Point{6.1, 5.3}))
	put("№", 9, Stroke{{0.8, 8.5}, {0.8, 2.2}, {3.5, 8.5}, {3.5, 2.2}}, circle(6.2, 5.5, 1.6, 2.2))
	put("§", 6, Stroke{{5.2, 2}, {3, 1.5}, {1.5, 2.5}, {1.5, 4.3}, {4.8, 6.8}, {4.8, 8.6}, {2.6, 9.2}, {1, 8.2}}, line(Point{1, 2}, Point{5, 9}))
	put("※", 8, line(Point{3.5, 1}, Point{3.5, 10}), line(Point{0.8, 3}, Point{6.2, 8}), line(Point{6.2, 3}, Point{0.8, 8}))
	put("†", 5, line(Point{2.5, 1.3}, Point{2.5, 9.5}), line(Point{0.8, 4}, Point{4.2, 4}))
	put("‡", 6, line(Point{2, 1.3}, Point{2, 9.5}), line(Point{4.2, 1.3}, Point{4.2, 9.5}), line(Point{0.8, 4}, Point{5.3, 4}), line(Point{0.8, 7}, Point{5.3, 7}))
}

func addNotationSymbols(
	put func(string, float64, ...Stroke),
	line func(Point, Point) Stroke,
	circle func(float64, float64, float64, float64) Stroke,
) {
	put("＋", 5, line(Point{0.8, 5.5}, Point{5.2, 5.5}), line(Point{3, 2.8}, Point{3, 8.2}))
	put("＝", 5, line(Point{0.8, 4}, Point{5.2, 4}), line(Point{0.8, 7}, Point{5.2, 7}))
	put("＜", 5, Stroke{{5.1, 2.8}, {1, 5.5}, {5.1, 8.2}})
	put("＞", 5, Stroke{{0.9, 2.8}, {5, 5.5}, {0.9, 8.2}})
	put("／", 5, line(Point{4.8, 1.5}, Point{0.8, 9.5}))
	put("＼", 5, line(Point{0.8, 1.5}, Point{4.8, 9.5}))
	put("～", 7, Stroke{{0.8, 5.8}, {2, 4.2}, {3.5, 5.8}, {5, 7.4}, {6.2, 5.8}})
	put("‰", 8, line(Point{0.8, 2}, Point{2.4, 4}), line(Point{3.5, 7.5}, Point{3.7, 7.5}), line(Point{5.2, 2}, Point{7.6, 4}), line(Point{6, 7.5}, Point{6.2, 7.5}))
	put("‱", 9, line(Point{0.8, 2}, Point{2.4, 4}), line(Point{3.3, 7.5}, Point{3.5, 7.5}), line(Point{4.8, 2}, Point{6.4, 4}), line(Point{7.8, 7.5}, Point{8, 7.5}))
	put("©", 7, circle(3.5, 5.5, 3, 4), Stroke{{2, 7.5}, {2, 3.8}, {5.2, 3.8}, {5.2, 7.5}})
	put("®", 7, circle(3.5, 5.5, 3, 4), Stroke{{2, 7.5}, {2, 3.8}, {4.5, 3.8}, {5.2, 4.7}, {4.5, 5.6}, {2, 5.6}}, line(Point{4, 5.6}, Point{5.3, 7.5}))
	put("™", 8, Stroke{{0.8, 2}, {0.8, 5.5}, {2.4, 2}, {2.4, 5.5}, {4, 2}, {4, 5.5}}, Stroke{{5.2, 2}, {5.2, 5.5}}, line(Point{4.6, 2}, Point{5.8, 2}), line(Point{6.8, 2}, Point{6.8, 5.5}), line(Point{6.2, 2}, Point{7.4, 2}))

	// Unicode's square unit symbols are frequently produced by lab software.
	// They share one compact unit mark so they remain writable and exportable
	// even when the user pastes them from a datasheet.
	unitMark := Stroke{{0.8, 2}, {0.8, 9}, {6.2, 9}, {6.2, 2}, {0.8, 2}}
	unitBar := line(Point{1.8, 5.5}, Point{5.2, 5.5})
	put("㎜㎝㎞㎏㎎㎍㎚㎛㎲㎳㎐㎑㎒㎓㎔㏀㏁㏂㏃㏄㏅㏈㏉㏋㏌㏏㏒㏓㏔㏕㏖㏗㏙㏚㏛㏜㏝", 7, unitMark, unitBar)
}

func addArrowSymbols(put func(string, float64, ...Stroke), line func(Point, Point) Stroke) {
	arrow := func(dx, dy float64) []Stroke {
		const cx, cy = 3.8, 5.5
		end := Point{cx + dx, cy + dy}
		start := Point{cx - dx, cy - dy}
		perp := Point{-dy * 0.28, dx * 0.28}
		return []Stroke{line(start, end), line(Point{end.X - dx*0.28 + perp.X, end.Y - dy*0.28 + perp.Y}, end), line(Point{end.X - dx*0.28 - perp.X, end.Y - dy*0.28 - perp.Y}, end)}
	}
	for chars, vector := range map[string][2]float64{
		"←": {-2.8, 0}, "↑": {0, -3.7}, "↓": {0, 3.7},
		"↗": {2.5, -3}, "↘": {2.5, 3}, "↙": {-2.5, 3}, "↖": {-2.5, -3},
	} {
		dx, dy := vector[0], vector[1]
		v := arrow(dx, dy)
		put(chars, 8, v...)
	}
	put("↔", 8, arrow(2.8, 0)...)
	put("↕", 8, arrow(0, 3.7)...)
	put("⇒", 8, line(Point{0.8, 5.5}, Point{7, 5.5}), Stroke{{4.8, 3.5}, {7, 5.5}, {4.8, 7.5}})
	put("⇐", 8, line(Point{1, 5.5}, Point{7.2, 5.5}), Stroke{{3.2, 3.5}, {1, 5.5}, {3.2, 7.5}})
	put("⇔", 8, line(Point{0.8, 5.5}, Point{7, 5.5}), Stroke{{2.7, 3.5}, {0.8, 5.5}, {2.7, 7.5}}, Stroke{{5.1, 3.5}, {7, 5.5}, {5.1, 7.5}})
}

// scaledGlyph reuses an existing handwritten glyph in a new box. This keeps
// composed symbols such as circled numbers consistent with the source font's
// stroke rhythm instead of replacing them with machine-like primitives.
//
// One scale fits both axes, so a glyph keeps the proportions it was written
// with. Fitting each axis separately stretch-fills the box, which the digits
// of this font cannot survive: the 1 is a 0.55-wide stroke next to a 3.0-wide
// 2, so filling a 3.3-wide box turned a superscript ¹ into a slash and the
// 1 of ½ with it.
func scaledGlyph(f *Font, r rune, x, y, width, height float64) []Stroke {
	return reusedGlyph(f, r, x, y, width, height, true)
}

// stretchedGlyph fills its box on both axes. Only shapes that are meant to be
// reshaped use it: the hand-drawn circle is a stretched 0, and the parentheses
// of ⑴ are stretched to stand the full height of the composed symbol.
func stretchedGlyph(f *Font, r rune, x, y, width, height float64) []Stroke {
	return reusedGlyph(f, r, x, y, width, height, false)
}

func reusedGlyph(f *Font, r rune, x, y, width, height float64, keepAspect bool) []Stroke {
	g, ok := f.Glyphs[r]
	if !ok || len(g.Strokes) == 0 {
		return nil
	}
	minX, minY, maxX, maxY, ok := inkBounds(g.Strokes)
	if !ok {
		return nil
	}
	sx, sy := width/(maxX-minX), height/(maxY-minY)
	if keepAspect {
		sx = min(sx, sy)
		sy = sx
	}
	// Centre what the scale leaves over, so a narrow glyph sits in the middle
	// of its cell instead of against the left edge.
	dx := x + (width-(maxX-minX)*sx)/2
	dy := y + (height-(maxY-minY)*sy)/2
	out := make([]Stroke, 0, len(g.Strokes))
	for _, stroke := range g.Strokes {
		copyStroke := make(Stroke, len(stroke))
		for i, p := range stroke {
			copyStroke[i] = Point{dx + (p.X-minX)*sx, dy + (p.Y-minY)*sy}
		}
		out = append(out, copyStroke)
	}
	return out
}

// shrunkDigit renders a body digit at the height of a superscript or subscript
// and reports the advance that digit needs. The cell follows the ink the way
// the body digits' own widths do: a superscript 1 is a narrow stroke, and
// giving it a wide digit's cell would space a run like ⁻¹⁹ apart.
func shrunkDigit(f *Font, r rune, x, y, height float64) ([]Stroke, float64) {
	g, ok := f.Glyphs[r]
	if !ok || len(g.Strokes) == 0 {
		return nil, 0
	}
	minX, minY, maxX, maxY, ok := inkBounds(g.Strokes)
	if !ok {
		return nil, 0
	}
	width := (maxX - minX) / (maxY - minY) * height
	return reusedGlyph(f, r, x, y, width, height, true), width
}

// inkBounds is the box the strokes of a glyph occupy.
func inkBounds(strokes []Stroke) (minX, minY, maxX, maxY float64, ok bool) {
	minX, minY, maxX, maxY = 1e9, 1e9, -1e9, -1e9
	for _, stroke := range strokes {
		for _, p := range stroke {
			minX, minY = min(minX, p.X), min(minY, p.Y)
			maxX, maxY = max(maxX, p.X), max(maxY, p.Y)
		}
	}
	if maxX-minX <= 1e-9 || maxY-minY <= 1e-9 {
		return 0, 0, 0, 0, false
	}
	return minX, minY, maxX, maxY, true
}

func addSuperscriptSymbols(f *Font, put func(string, float64, ...Stroke)) {
	for i, r := range []rune("⁰¹²³⁴⁵⁶⁷⁸⁹") {
		strokes, width := shrunkDigit(f, rune('0'+i), 0.5, 1.1, 4.2)
		put(string(r), width, strokes...)
	}
	for i, r := range []rune("₀₁₂₃₄₅₆₇₈₉") {
		strokes, width := shrunkDigit(f, rune('0'+i), 0.5, 5.4, 4.2)
		put(string(r), width, strokes...)
	}
	put("⁺", 5, Stroke{{0.8, 3.2}, {4.8, 3.2}}, Stroke{{2.8, 1.2}, {2.8, 5.2}})
	put("⁻", 5, lineForSymbols(Point{0.8, 3.2}, Point{4.8, 3.2}))
	put("⁼", 5, lineForSymbols(Point{0.8, 2.5}, Point{4.8, 2.5}), lineForSymbols(Point{0.8, 4.2}, Point{4.8, 4.2}))
	put("⁽", 3, Stroke{{3, 1}, {1.3, 2.2}, {1.3, 4.5}, {3, 6}})
	put("⁾", 3, Stroke{{1, 1}, {2.7, 2.2}, {2.7, 4.5}, {1, 6}})
	put("₊", 5, Stroke{{0.8, 7.3}, {4.8, 7.3}}, Stroke{{2.8, 5.3}, {2.8, 9.3}})
	put("₋", 5, lineForSymbols(Point{0.8, 7.3}, Point{4.8, 7.3}))
	put("₌", 5, lineForSymbols(Point{0.8, 6.5}, Point{4.8, 6.5}), lineForSymbols(Point{0.8, 8.2}, Point{4.8, 8.2}))
	put("₍", 3, Stroke{{3, 5}, {1.3, 6.2}, {1.3, 8.5}, {3, 10}})
	put("₎", 3, Stroke{{1, 5}, {2.7, 6.2}, {2.7, 8.5}, {1, 10}})
}

func lineForSymbols(a, b Point) Stroke { return Stroke{a, b} }

func addFractionSymbols(f *Font, put func(string, float64, ...Stroke), line func(Point, Point) Stroke) {
	fractions := map[rune][2]int{
		'½': {1, 2}, '⅓': {1, 3}, '⅔': {2, 3}, '¼': {1, 4}, '¾': {3, 4},
		'⅕': {1, 5}, '⅖': {2, 5}, '⅗': {3, 5}, '⅘': {4, 5}, '⅙': {1, 6},
		'⅚': {5, 6}, '⅐': {1, 7}, '⅛': {1, 8}, '⅜': {3, 8}, '⅝': {5, 8}, '⅞': {7, 8},
	}
	for r, pair := range fractions {
		strokes := []Stroke{line(Point{3.8, 1}, Point{2.8, 10})}
		strokes = append(strokes, scaledGlyph(f, rune('0'+pair[0]), 0.4, 1.1, 2.8, 3.2)...)
		strokes = append(strokes, scaledGlyph(f, rune('0'+pair[1]), 4, 6.1, 2.8, 3.2)...)
		put(string(r), 7, strokes...)
	}
	put("⁄", 4, line(Point{3.5, 1}, Point{0.8, 10}))
	put("∕", 4, line(Point{3.5, 1}, Point{0.8, 10}))
}

func addCircledNumberSymbols(f *Font, put func(string, float64, ...Stroke)) {
	// The circle is a 0 reshaped into the ring the source font never drew.
	handCircle := func(x, y, width, height float64) []Stroke {
		return stretchedGlyph(f, '0', x, y, width, height)
	}
	for n := 1; n <= 31; n++ {
		var r rune
		switch {
		case n <= 20:
			r = rune(0x245f + n)
		default:
			r = rune(0x3250 + n - 20)
		}
		strokes := handCircle(1, 1, 9, 9)
		digits := strconv.Itoa(n)
		if len(digits) == 1 {
			strokes = append(strokes, scaledGlyph(f, rune('0'+n), 3.4, 3.2, 3.2, 4.6)...)
		} else {
			strokes = append(strokes, scaledGlyph(f, rune(digits[0]), 2.0, 3.4, 2.4, 4.1)...)
			strokes = append(strokes, scaledGlyph(f, rune(digits[1]), 6.0, 3.4, 2.4, 4.1)...)
		}
		put(string(r), 11, strokes...)
	}
	// Circled zero is frequently used as an unnumbered option.
	zero := handCircle(1, 1, 9, 9)
	zero = append(zero, scaledGlyph(f, '0', 3.4, 3.2, 3.2, 4.6)...)
	put("⓪", 11, zero...)
	for n := 1; n <= 20; n++ {
		r := rune(0x2473 + n)
		strokes := stretchedGlyph(f, '(', 0.4, 1.2, 1.6, 8.7)
		strokes = append(strokes, stretchedGlyph(f, ')', 9.0, 1.2, 1.6, 8.7)...)
		digits := strconv.Itoa(n)
		if len(digits) == 1 {
			strokes = append(strokes, scaledGlyph(f, rune('0'+n), 3.4, 3.2, 3.2, 4.6)...)
		} else {
			strokes = append(strokes, scaledGlyph(f, rune(digits[0]), 2.0, 3.4, 2.4, 4.1)...)
			strokes = append(strokes, scaledGlyph(f, rune(digits[1]), 6.0, 3.4, 2.4, 4.1)...)
		}
		put(string(r), 11, strokes...)
	}
}

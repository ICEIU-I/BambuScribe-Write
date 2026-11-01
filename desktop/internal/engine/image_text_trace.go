package engine

// Keep the geometry of unverified text, including single-pixel decimal points.
// Thinning still makes a single pen centreline; no geometric fitting follows it.
func traceKeptImageText(mask []bool, width, height int) []Stroke {
	if len(mask) == 0 {
		return nil
	}
	small := smallTextComponents(mask, width, height)
	thinImageMask(mask, width, height)
	// Some compact antialiased dots lose every pixel in a simultaneous thinning
	// pass. Keep one original pixel nearest their centre if that happens.
	for _, pixels := range small {
		remaining, cx, cy := false, 0., 0.
		for _, p := range pixels {
			remaining = remaining || mask[p]
			cx += float64(p % width)
			cy += float64(p / width)
		}
		if remaining {
			continue
		}
		cx, cy = cx/float64(len(pixels)), cy/float64(len(pixels))
		anchor, best := pixels[0], float64(width*width+height*height)
		for _, p := range pixels {
			dx, dy := float64(p%width)-cx, float64(p/width)-cy
			if d := dx*dx + dy*dy; d < best {
				anchor, best = p, d
			}
		}
		mask[anchor] = true
	}
	strokes := traceImageSkeleton(mask, width, height, func(x, y int) Point {
		return Point{float64(x), float64(y)}
	})
	for i, ink := range mask {
		if !ink {
			continue
		}
		x, y := i%width, i/width
		isolated := true
		for _, d := range skeletonDirections {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && nx < width && ny >= 0 && ny < height && mask[ny*width+nx] {
				isolated = false
				break
			}
		}
		if isolated {
			end := float64(x) + .25
			if x == width-1 {
				end = float64(x) - .25
			}
			strokes = append(strokes, Stroke{{float64(x), float64(y)}, {end, float64(y)}})
		}
	}
	return strokes
}

func smallTextComponents(mask []bool, width, height int) [][]int {
	seen := make([]bool, len(mask))
	components := [][]int{}
	for i, ink := range mask {
		if !ink || seen[i] {
			continue
		}
		seen[i] = true
		pixels := []int{i}
		for head := 0; head < len(pixels); head++ {
			p := pixels[head]
			for _, d := range skeletonDirections {
				x, y := p%width+d[0], p/width+d[1]
				if x >= 0 && x < width && y >= 0 && y < height {
					n := y*width + x
					if mask[n] && !seen[n] {
						seen[n] = true
						pixels = append(pixels, n)
					}
				}
			}
		}
		if len(pixels) <= 64 {
			components = append(components, pixels)
		}
	}
	return components
}

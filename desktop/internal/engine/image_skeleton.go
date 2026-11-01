package engine

var skeletonDirections = [8][2]int{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}

func thinImageMask(mask []bool, width, height int) []bool {
	// A white border lets thinning handle ink touching an image edge normally.
	stride := width + 2
	padded := make([]bool, stride*(height+2))
	for y := 0; y < height; y++ {
		copy(padded[(y+1)*stride+1:(y+1)*stride+1+width], mask[y*width:(y+1)*width])
	}
	thinPaddedImage(padded, stride, height+2)
	for y := 0; y < height; y++ {
		copy(mask[y*width:(y+1)*width], padded[(y+1)*stride+1:(y+1)*stride+1+width])
	}
	return mask
}

func thinPaddedImage(mask []bool, width, height int) []bool {
	changed := true
	for pass := 0; changed && pass < 128; pass++ {
		changed = false
		for phase := 0; phase < 2; phase++ {
			remove := make([]int, 0)
			for y := 1; y < height-1; y++ {
				for x := 1; x < width-1; x++ {
					idx := y*width + x
					if !mask[idx] {
						continue
					}
					var neighbours [8]bool
					count := 0
					for i, d := range skeletonDirections {
						neighbours[i] = mask[(y+d[1])*width+x+d[0]]
						if neighbours[i] {
							count++
						}
					}
					if count < 2 || count > 6 {
						continue
					}
					transitions := 0
					for i := 0; i < 8; i++ {
						if !neighbours[i] && neighbours[(i+1)%8] {
							transitions++
						}
					}
					if transitions != 1 {
						continue
					}
					p2, p4, p6, p8 := neighbours[0], neighbours[2], neighbours[4], neighbours[6]
					if phase == 0 {
						if p2 && p4 && p6 || p4 && p6 && p8 {
							continue
						}
					} else if p2 && p4 && p8 || p2 && p6 && p8 {
						continue
					}
					remove = append(remove, idx)
				}
			}
			if len(remove) > 0 {
				changed = true
				for _, idx := range remove {
					mask[idx] = false
				}
			}
		}
	}
	return mask
}

func traceImageSkeleton(mask []bool, width, height int, point func(int, int) Point) []Stroke {
	neighbours := func(idx int) []int {
		out := []int{}
		x, y := idx%width, idx/width
		for _, d := range skeletonDirections {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || nx >= width || ny < 0 || ny >= height || !mask[ny*width+nx] {
				continue
			}
			// Don't add a diagonal shortcut across an existing orthogonal connection.
			// Such triangles create tiny branches and repeated pen lifts on curves.
			if d[0] != 0 && d[1] != 0 && (mask[y*width+nx] || mask[ny*width+x]) {
				continue
			}
			out = append(out, ny*width+nx)
		}
		return out
	}
	degree := make([]int, len(mask))
	for i, active := range mask {
		if active {
			degree[i] = len(neighbours(i))
		}
	}
	visited := map[uint64]bool{}
	key := func(a, b int) uint64 {
		if a > b {
			a, b = b, a
		}
		return uint64(a)<<32 | uint64(uint32(b))
	}
	strokes := []Stroke{}
	walk := func(start, next int) {
		indices := []int{start}
		prev, cur := start, next
		visited[key(prev, cur)] = true
		for {
			indices = append(indices, cur)
			if degree[cur] != 2 || cur == start {
				break
			}
			ns := neighbours(cur)
			n := ns[0]
			if n == prev {
				n = ns[1]
			}
			if visited[key(cur, n)] {
				break
			}
			visited[key(cur, n)] = true
			prev, cur = cur, n
		}
		s := make(Stroke, 0, len(indices))
		for _, idx := range indices {
			s = append(s, point(idx%width, idx/width))
		}
		if len(s) > 1 {
			strokes = append(strokes, s)
		}
	}
	for i, active := range mask {
		if active && degree[i] != 2 {
			for _, n := range neighbours(i) {
				if !visited[key(i, n)] {
					walk(i, n)
				}
			}
		}
	}
	for i, active := range mask {
		if active {
			for _, n := range neighbours(i) {
				if !visited[key(i, n)] {
					walk(i, n)
				}
			}
		}
	}
	return strokes
}

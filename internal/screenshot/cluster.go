package screenshot

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// Cluster represents a spatial region of differences.
type Cluster struct {
	ID         int     `json:"id"`
	BoundingBox Rect   `json:"bounding_box"`
	PixelCount int     `json:"pixel_count"`
	CenterX    float64 `json:"center_x"`
	CenterY    float64 `json:"center_y"`
	LocalSSIM  float64 `json:"local_ssim"`
}

// Rect is a simple rectangle.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// FindClusters identifies spatial clusters of differences using
// connected-component labeling with 8-connectivity.
// Pixels with SSIM below threshold are considered "different".
func FindClusters(result SSIMResult, threshold float64) []Cluster {
	w, h := result.Width, result.Height
	labels := make([]int, w*h)
	nextLabel := 1

	// Mark different pixels
	diff := make([]bool, w*h)
	for y := range h {
		for x := range w {
			if result.Heatmap[y][x] < threshold {
				diff[y*w+x] = true
			}
		}
	}

	// Connected-component labeling (two-pass)
	parent := make(map[int]int)

	find := func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]] // path compression
			x = parent[x]
		}
		return x
	}

	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}

	// First pass: assign labels
	for y := range h {
		for x := range w {
			if !diff[y*w+x] {
				continue
			}

			neighbors := []int{}
			// Check 8-connected neighbors already visited
			for _, d := range [][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= 0 && nx < w && ny >= 0 && ny < h && labels[ny*w+nx] > 0 {
					neighbors = append(neighbors, labels[ny*w+nx])
				}
			}

			if len(neighbors) == 0 {
				labels[y*w+x] = nextLabel
				parent[nextLabel] = nextLabel
				nextLabel++
			} else {
				minLabel := neighbors[0]
				for _, n := range neighbors[1:] {
					if n < minLabel {
						minLabel = n
					}
				}
				labels[y*w+x] = minLabel
				for _, n := range neighbors {
					union(minLabel, n)
				}
			}
		}
	}

	// Second pass: resolve labels
	for i := range labels {
		if labels[i] > 0 {
			labels[i] = find(labels[i])
		}
	}

	// Collect cluster data
	type clusterData struct {
		minX, minY, maxX, maxY int
		sumX, sumY             int
		count                  int
		ssimSum                float64
	}

	clusters := make(map[int]*clusterData)

	for y := range h {
		for x := range w {
			l := labels[y*w+x]
			if l == 0 {
				continue
			}

			cd, ok := clusters[l]
			if !ok {
				cd = &clusterData{minX: x, minY: y, maxX: x, maxY: y}
				clusters[l] = cd
			}

			if x < cd.minX {
				cd.minX = x
			}
			if x > cd.maxX {
				cd.maxX = x
			}
			if y < cd.minY {
				cd.minY = y
			}
			if y > cd.maxY {
				cd.maxY = y
			}

			cd.sumX += x
			cd.sumY += y
			cd.count++
			cd.ssimSum += result.Heatmap[y][x]
		}
	}

	// Build result
	var result2 []Cluster
	id := 1

	for _, cd := range clusters {
		result2 = append(result2, Cluster{
			ID: id,
			BoundingBox: Rect{
				X:      cd.minX,
				Y:      cd.minY,
				Width:  cd.maxX - cd.minX + 1,
				Height: cd.maxY - cd.minY + 1,
			},
			PixelCount: cd.count,
			CenterX:    float64(cd.sumX) / float64(cd.count),
			CenterY:    float64(cd.sumY) / float64(cd.count),
			LocalSSIM:  cd.ssimSum / float64(cd.count),
		})
		id++
	}

	// Sort by pixel count descending
	sort.Slice(result2, func(i, j int) bool {
		return result2[i].PixelCount > result2[j].PixelCount
	})

	// Re-assign sequential IDs after sorting
	for i := range result2 {
		result2[i].ID = i + 1
	}

	return result2
}

// RenderDiffWithBoxes creates a diff image with bounding boxes drawn around clusters.
func RenderDiffWithBoxes(img1, img2 image.Image, clusters []Cluster) *image.RGBA {
	b := img1.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))

	// Blend the two images (50/50)
	for y := range h {
		for x := range w {
			r1, g1, b1, _ := img1.At(b.Min.X+x, b.Min.Y+y).RGBA()
			r2, g2, b2, _ := img2.At(b.Min.X+x, b.Min.Y+y).RGBA()
			out.SetRGBA(x, y, color.RGBA{
				R: uint8((r1/256 + r2/256) / 2),
				G: uint8((g1/256 + g2/256) / 2),
				B: uint8((b1/256 + b2/256) / 2),
				A: 255,
			})
		}
	}

	// Draw bounding boxes in red
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	for _, c := range clusters {
		bb := c.BoundingBox
		drawRect(out, bb.X, bb.Y, bb.X+bb.Width-1, bb.Y+bb.Height-1, red)
	}

	return out
}

func drawRect(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	b := img.Bounds()
	x1 = clampInt(x1, 0, b.Dx()-1)
	x2 = clampInt(x2, 0, b.Dx()-1)
	y1 = clampInt(y1, 0, b.Dy()-1)
	y2 = clampInt(y2, 0, b.Dy()-1)

	for x := x1; x <= x2; x++ {
		img.SetRGBA(x, y1, c)
		img.SetRGBA(x, y2, c)
	}

	for y := y1; y <= y2; y++ {
		img.SetRGBA(x1, y, c)
		img.SetRGBA(x2, y, c)
	}
}

func clampInt(v, lo, hi int) int {
	return int(math.Max(float64(lo), math.Min(float64(hi), float64(v))))
}

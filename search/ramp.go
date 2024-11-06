package search

import (
	"math"

	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
)

type RampLocation struct {
	Center api.Point2D
}

func CalculateRampLocations(bot *botutil.Bot, pos api.Point2D, distance int32) []*api.DebugBox {
	heightMap := NewHeightMap(bot.GameInfo().StartRaw)
	xMin, yMin := int32(pos.X-float32(distance)/2), int32(pos.Y-float32(distance)/2)
	xMax, yMax := xMin+distance, yMin+distance
	if xMin < 1 {
		xMin = 1
	}
	if yMin < 1 {
		yMin = 1
	}
	if xMax >= heightMap.Width() {
		xMax = heightMap.Width() - 1
	}
	if yMax >= heightMap.Height() {
		xMax = heightMap.Height() - 1
	}

	var boxes []*api.DebugBox
	for y := yMin; y < yMax; y++ {
		for x := xMin; x < xMax; x++ {
			X := float32(x)
			Y := float32(y)
			h := heightMap.Interpolate(X, Y)
			if h < 8 || (math.Round(float64(h)) == float64(h) && (int(h)%2) == 0) {
				continue
			}

			oth := []float32{
				heightMap.Get(x+1, y+1),
				heightMap.Get(x+1, y),
				heightMap.Get(x+1, y-1),
				heightMap.Get(x-1, y),
				heightMap.Get(x-1, y+1),
				heightMap.Get(x, y+1),
				heightMap.Get(x-1, y-1),
				heightMap.Get(x, y-1),
			}

			elevationChange := false
			for _, o := range oth {
				if o < 2 {
					continue
				}
				if o != h && math.Abs(float64(o-h)) < 1 {
					elevationChange = true
				}
			}

			if elevationChange {
				z := float32(int(h)/4) * 4.0
				PrintBox(api.DebugBox{
					Color: green,
					Min:   &api.Point{X: X, Y: Y, Z: z},
					Max:   &api.Point{X: X + 1, Y: Y + 1, Z: z + 4},
				})
			}
		}
	}

	return boxes
}

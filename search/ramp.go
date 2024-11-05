package search

import (
	"log"
	"math"

	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
)

type RampLocation struct {
	Center api.Point2D
}

func CalculateRampLocations(bot *botutil.Bot, debug bool) []BaseLocation {
	var spheres []*api.DebugBox
	info := bot.GameInfo()
	heightMap := NewHeightMap(info.StartRaw)
	for y := int32(0); y < heightMap.Height(); y++ {
		for x := int32(0); x < heightMap.Width(); x++ {
			h := heightMap.Get(x, y)
			if h < 8 || math.Round(float64(h)) != float64(h) || (int(h)%2) != 0 {
				continue
			}

			oth := []float32{
				heightMap.Get(x+1, y+1),
				heightMap.Get(x+1, y-1),
				heightMap.Get(x-1, y+1),
				heightMap.Get(x-1, y-1),
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
				//log.Printf("x: %v y: %v h: %v\n", x, y, h)
				X := float32(x)
				Y := float32(y)
				z := h
				spheres = append(spheres, &api.DebugBox{
					Color: green,
					Min:   &api.Point{X: X, Y: Y, Z: z},
					Max:   &api.Point{X: X + 1, Y: Y + 1, Z: z},
				})
			}
		}
	}

	log.Printf("Num spheres: %v h: %v w: %v", len(spheres), heightMap.Height(), heightMap.Width())

	bot.SendDebugCommands([]*api.DebugCommand{
		{
			Command: &api.DebugCommand_Draw{
				Draw: &api.DebugDraw{
					Boxes: spheres,
				},
			},
		},
	})

	return nil
}

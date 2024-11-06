package search

import (
	"log"

	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

var (
	debugBoxes []*api.DebugBox
	debugBoxP  = float32(0.05)
)

func PrintBox(box api.DebugBox) {
	debugBoxes = append(debugBoxes, &api.DebugBox{
		Color: box.Color,
		Min: &api.Point{
			X: box.Min.X + debugBoxP,
			Y: box.Min.Y + debugBoxP,
			Z: box.Min.Z + debugBoxP*2,
		},
		Max: &api.Point{
			X: box.Max.X - debugBoxP,
			Y: box.Max.Y - debugBoxP,
			Z: box.Max.Z + debugBoxP*2,
		},
	})
}

func PrintPoint(p api.Point2D) {
	debugBoxes = append(debugBoxes, &api.DebugBox{
		Color: red,
		Min: &api.Point{
			X: p.X + debugBoxP,
			Y: p.Y + debugBoxP,
			Z: 0,
		},
		Max: &api.Point{
			X: p.X - debugBoxP,
			Y: p.Y - debugBoxP,
			Z: 20,
		},
	})
}

func ShowDebugBoxes(bot *botutil.Bot) {
	bot.SendDebugCommands([]*api.DebugCommand{
		{
			Command: &api.DebugCommand_Draw{
				Draw: &api.DebugDraw{
					Boxes: debugBoxes,
				},
			},
		},
	})
	//debugBoxes = make([]*api.DebugBox, 0)
}

func (pg *PlacementGrid) DebugBuildings() {
	heightMap := NewHeightMap(pg.bot.GameInfo().StartRaw)
	for _, v := range pg.structures {
		z := heightMap.Interpolate(v.point.X, v.point.Y)
		PrintBox(api.DebugBox{
			Min: &api.Point{X: v.point.X - float32(v.size.X)/2, Y: v.point.Y - float32(v.size.Y)/2, Z: z},
			Max: &api.Point{X: v.point.X + float32(v.size.X)/2, Y: v.point.Y + float32(v.size.Y)/2, Z: z},
		})
	}
}

func (pg *PlacementGrid) DebugLocationsNearPoint(pos api.Point2D, distance int32) {
	log.Printf("pos: %v", pos)
	heightMap := NewHeightMap(pg.bot.GameInfo().StartRaw)
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
		yMax = heightMap.Height() - 1
	}

	var req []*api.RequestQueryBuildingPlacement
	for y := yMin; y < yMax; y++ {
		for x := xMin; x < xMax; x++ {
			req = append(req, &api.RequestQueryBuildingPlacement{
				AbilityId: ability.Build_SensorTower,
				TargetPos: &api.Point2D{X: float32(x) + 0.5, Y: float32(y) + 0.5},
			})
		}
	}

	resp := pg.bot.Query(api.RequestQuery{
		Placements: req,
	})

	for i, r := range resp.GetPlacements() {

		x := int32(req[i].TargetPos.X)
		y := int32(req[i].TargetPos.Y)
		X := req[i].TargetPos.X
		Y := req[i].TargetPos.Y

		ok := r.GetResult() == api.ActionResult_Success
		valid := pg.grid.Get(x, y)
		if ok != valid {
			z := heightMap.Get(x, y)
			PrintBox(api.DebugBox{
				Color: red,
				Min:   &api.Point{X: X, Y: Y, Z: z},
				Max:   &api.Point{X: Y + 1, Y: Y + 1, Z: z},
			})
			log.Printf("Wrong! x: %v y: %v height: %v width: %v\n", X, Y, pg.grid.Height(), pg.grid.Width())
		} else {
		}
	}

	for y := yMin; y < yMax; y++ {
		for x := xMin; x < xMax; x++ {
			X := float32(x)
			Y := float32(y)
			z := heightMap.Get(x, y)
			if pg.grid.Get(x, y) == true {
				PrintBox(api.DebugBox{
					Color: green,
					Min:   &api.Point{X: X, Y: Y, Z: z},
					Max:   &api.Point{X: X + 1, Y: Y + 1, Z: z},
				})
			}
		}
	}
}

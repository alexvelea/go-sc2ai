package search

import (
	"log"

	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
	"github.com/chippydip/go-sc2ai/enums/unit"
)

var sizeCache = map[api.UnitTypeID]api.Size2DI{}

// UnitPlacementSize estimates building footprints based on unit radius.
func UnitPlacementSize(u botutil.Unit) api.Size2DI {
	if s, ok := sizeCache[u.UnitType]; ok {
		return s
	}

	// Round coordinate to the nearest half (not needed except for things like the KD8Charge)
	x, y := float32(int32(u.Pos.X*2+0.5))/2, float32(int32(u.Pos.Y*2+0.5))/2
	xEven, yEven := int(u.Pos.X*2+0.5)%2 == 0, int(u.Pos.Y*2+0.5)%2 == 0

	// Compute bounds based on the (bad) radius provided by the game
	xMin, yMin := int32(x-u.Radius+0.5), int32(y-u.Radius+0.5)
	xMax, yMax := int32(x+u.Radius+0.5), int32(y+u.Radius+0.5)

	// Get the real radius in all four directions as calculated above
	rxMin, ryMin := x-float32(xMin), y-float32(yMin)
	rxMax, ryMax := float32(xMax)-x, float32(yMax)-y

	// If the radii are not symetric, take the smaller value
	rx, ry := rxMin, ryMin
	if rxMax < rx {
		rx = rxMax
	}
	if ryMax < ry {
		ry = ryMax
	}

	// Re-compute bounds with the hopefully better radii
	xMin, yMin = int32(u.Pos.X-rx+0.5), int32(u.Pos.Y-ry+0.5)
	xMax, yMax = int32(u.Pos.X+rx+0.5), int32(u.Pos.Y+ry+0.5)

	// Adjust for non-square structures (TODO: should this just special-case Minerals?)
	if xEven != yEven {
		if yEven {
			xMin++
			xMax--
		} else {
			yMin++
			yMax--
		}
	}

	// Cache and return the computed size
	size := api.Size2DI{X: xMax - xMin, Y: yMax - yMin}
	sizeCache[u.UnitType] = size
	log.Printf("%v %v %v -> %v", unit.String(u.UnitType), u.Pos2D(), u.Radius, size)
	return size
}

// PlacementGrid ...
type PlacementGrid struct {
	bot        *botutil.Bot
	raw        api.ImageDataBits
	grid       api.ImageDataBits
	structures map[api.UnitTag]structureInfo
}

type structureInfo struct {
	point api.Point2D
	size  api.Size2DI
}

// NewPlacementGrid ...
func NewPlacementGrid(bot *botutil.Bot) *PlacementGrid {
	raw := bot.GameInfo().GetStartRaw().GetPlacementGrid().Bits()
	pg := &PlacementGrid{
		bot:        bot,
		raw:        raw,
		grid:       raw.Copy(),
		structures: map[api.UnitTag]structureInfo{},
	}
	pg.Update()
	return pg
}

func (pg *PlacementGrid) markGrid(pos api.Point2D, size api.Size2DI, value bool) {
	xMin, yMin := int32(pos.X-float32(size.X)/2), int32(pos.Y-float32(size.Y)/2)
	xMax, yMax := xMin+size.X, yMin+size.Y

	for y := yMin; y < yMax; y++ {
		for x := xMin; x < xMax; x++ {
			pg.grid.Set(x, y, value)
		}
	}
}

func (pg *PlacementGrid) checkGrid(pos api.Point2D, size api.Size2DI, value bool) bool {
	xMin, yMin := int32(pos.X-float32(size.X)/2), int32(pos.Y-float32(size.Y)/2)
	xMax, yMax := xMin+size.X, yMin+size.Y

	for y := yMin; y < yMax; y++ {
		for x := xMin; x < xMax; x++ {
			if pg.grid.Get(x, y) != value {
				return false
			}
		}
	}
	return true
}

// CanPlace checks if a structure of a certain type can currently be places at the given location.
func (pg *PlacementGrid) CanPlace(u botutil.Unit, pos api.Point2D) bool {
	return pg.checkGrid(pos, UnitPlacementSize(u), true)
}

func (pg *PlacementGrid) Update() {
	// Remove any units that are gone or have changed type or position
	for k, v := range pg.structures {
		if u := pg.bot.UnitByTag(k); u.IsNil() || !u.IsStructure() || u.Pos2D() != v.point || UnitPlacementSize(u) != v.size {
			pg.markGrid(v.point, v.size, true)
			delete(pg.structures, k)
		}
	}

	// (Re-)add new units or ones that have changed type or position
	pg.bot.AllUnits().Each(func(u botutil.Unit) {
		if _, ok := pg.structures[u.Tag]; !ok && u.IsStructure() {
			v := structureInfo{u.Pos2D(), UnitPlacementSize(u)}
			pg.markGrid(v.point, v.size, false)
			pg.structures[u.Tag] = v
		}
	})
}

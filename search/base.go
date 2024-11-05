package search

import (
	"log"
	"strings"

	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

// Base ...
type Base struct {
	m *Map
	i int

	ResourceCenter api.Point2D
	MineralCenter  api.Point2D
	Minerals       []botutil.Unit
	Geysers        []botutil.Unit
	Resources      map[api.UnitTag]botutil.Unit
	Workers        map[api.UnitTag]botutil.Unit

	Location     api.Point2D
	TownHall     botutil.Unit
	GasBuildings map[api.Point2D]botutil.Unit

	mining  map[api.UnitTag]api.UnitTag
	minedBy map[api.UnitTag]map[api.UnitTag]bool
}

func newBase(m *Map, i int, loc BaseLocation) *Base {
	// Re-compute center with 4x weight on geysers to better represent unbalanced gas bases
	cluster := UnitCluster{}
	minerals := UnitCluster{}
	for _, u := range loc.Resources.Units() {
		if u.HasVespene {
			cluster.Add(u)
			cluster.Add(u)
			cluster.Add(u)
		} else {
			minerals.Add(u)
		}
		cluster.Add(u)
	}

	return &Base{
		m:              m,
		i:              i,
		ResourceCenter: cluster.Center(),
		MineralCenter:  minerals.Center(),
		Minerals:       make([]botutil.Unit, 0, loc.Resources.Count()),
		Geysers:        make([]botutil.Unit, 0, 2),
		Resources:      make(map[api.UnitTag]botutil.Unit),
		Workers:        make(map[api.UnitTag]botutil.Unit),
		Location:       loc.Location,
		GasBuildings:   map[api.Point2D]botutil.Unit{},
		mining:         make(map[api.UnitTag]api.UnitTag),
		minedBy:        make(map[api.UnitTag]map[api.UnitTag]bool),
	}
}

func (base *Base) updateResource(u botutil.Unit) {
	switch {
	case u.HasMinerals:
		base.Minerals = base.updateOrAdd(base.Minerals, u)
	case u.HasVespene:
		base.Geysers = base.updateOrAdd(base.Geysers, u)
	default:
		log.Panicf("unknown resource: %v", u)
	}

	base.Resources[u.Tag] = u
	if base.minedBy[u.Tag] == nil {
		base.minedBy[u.Tag] = make(map[api.UnitTag]bool)
	}
}

func (base *Base) update(bot *botutil.Bot) {
	// Check for exhausted minerals
	for i := 0; i < len(base.Minerals); i++ {
		u := base.Minerals[i]
		if !bot.WasObserved(u.Tag) {
			for by := range base.minedBy[u.Tag] {
				delete(base.mining, by)
			}
			delete(base.minedBy, u.Tag)
			copy(base.Minerals[i:], base.Minerals[i+1:])
			base.Minerals = base.Minerals[:len(base.Minerals)-1]
			i--
		}
	}

	// Clear fields that are re-computed each loop
	base.TownHall = botutil.Unit{}
	for k := range base.GasBuildings {
		delete(base.GasBuildings, k)
	}
	for k := range base.Workers {
		delete(base.Workers, k)
	}
}

func (base *Base) step(bot *botutil.Bot) {
	// cross-reference all valid workers
	toDelete := make([]api.UnitTag, 0)
	for tag := range base.mining {
		_, ok := base.Workers[tag]
		if !ok {
			toDelete = append(toDelete, tag)
		}
	}
	for _, tag := range toDelete {
		patchTag := base.mining[tag]
		delete(base.mining, tag)
		delete(base.minedBy[patchTag], tag)
	}

	for workerTag, resourceTag := range base.mining {
		worker := base.Workers[workerTag]
		patch := base.Resources[resourceTag]

		if worker.IsCarryingResources() {
			worker.Order(ability.Harvest_Return)
		} else {
			bot.UnitOrderTarget(worker, ability.Harvest_Gather, patch)
		}
	}
}

func (base *Base) updateOrAdd(units []botutil.Unit, u botutil.Unit) []botutil.Unit {
	for i, u2 := range units {
		if u2.Pos2D().Distance2(u.Pos2D()) < 1 {
			if u2.Pos2D() != u.Pos2D() {
				log.Panicf("%v != %v", u2.Pos2D(), u.Pos2D())
			}

			units[i] = u
			if u.IsSnapshot() {
				u.MineralContents = u2.MineralContents
				u.VespeneContents = u2.VespeneContents
			}
			return units
		}
	}

	// Not found, append
	units = append(units, u)

	// Keep sorted
	uIsSmall, uDist := strings.HasSuffix(u.Name, "750"), u.Pos2D().Distance2(base.Location)
	for i := 0; i < len(units)-1; i++ {
		isSmall := strings.HasSuffix(units[i].Name, "750")
		if !isSmall && uIsSmall {
			continue // small patches after big ones, regardless of distance
		}

		dist := units[i].Pos2D().Distance2(base.Location)
		if uIsSmall != isSmall || uDist < dist {
			// found insertion point, shift back the rest and insert again
			copy(units[i+1:], units[i:])
			units[i] = u
			return units
		}
	}

	return units
}

func (base *Base) addWorker(worker botutil.Unit) {
	workerTag := worker.Tag
	base.Workers[workerTag] = worker

	log.Printf("adding worker: %v i: %v", workerTag, base.i)

	// prioritize minerals
	for _, mineral := range base.Minerals {
		if len(base.minedBy[mineral.Tag]) < 2 {
			base.minedBy[mineral.Tag][workerTag] = true
			base.mining[workerTag] = mineral.Tag
			return
		}
	}

	for _, vespene := range base.GasBuildings {
		if len(base.minedBy[vespene.Tag]) < 3 {
			base.minedBy[vespene.Tag][workerTag] = true
			base.mining[workerTag] = vespene.Tag
			return
		}
	}
}

func (base *Base) GetBuilder() botutil.Unit {
	log.Printf("fetching i: %v builder. workers: %v", base.i, len(base.Workers))
	for i := len(base.Minerals) - 1; i >= 0; i -= 1 {
		mineral := base.Minerals[i]
		mineralTag := mineral.Tag
		if len(base.minedBy[mineralTag]) > 0 {
			var unitTag api.UnitTag
			for unitTag = range base.minedBy[mineralTag] {
				break
			}
			builder := base.Workers[unitTag]

			delete(base.minedBy[mineralTag], unitTag)
			delete(base.mining, unitTag)
			delete(base.Workers, unitTag)
			return builder
		}
	}
	return botutil.Unit{}
}

// IsSelfOwned returns true if the current player owns the TownHall at this base.
func (base *Base) IsSelfOwned() bool {
	return !base.TownHall.IsNil() && base.TownHall.Alliance == api.Alliance_Self
}

// IsEnemyOwned returns true if the enemy player owns the TownHall at this base.
func (base *Base) IsEnemyOwned() bool {
	return !base.TownHall.IsNil() && base.TownHall.Alliance == api.Alliance_Enemy
}

// IsUnowned returns true if no player owns a TownHall at this base.
func (base *Base) IsUnowned() bool {
	return base.TownHall.IsNil()
}

// Natural returns the closest other base.
func (base *Base) Natural() *Base {
	best, minDist := (*Base)(nil), float32(256*256)
	for _, other := range base.m.Bases {
		if dist := base.WalkDistance(other); dist > 0 && dist < minDist {
			best, minDist = other, dist
		}
	}
	return best
}

// WalkDistance returns the ground pathfinding distances between the bases.
func (base *Base) WalkDistance(other *Base) float32 {
	return base.m.distance(base.i, other.i)
}

func (base *Base) HasWorker(tag api.UnitTag) bool {
	_, ok := base.mining[tag]
	return ok
}

func (base *Base) NeedsWorker() bool {
	if base.IsSelfOwned() == false {
		return false
	}
	needed := len(base.Minerals)*2 + len(base.GasBuildings)*3
	return needed > len(base.Workers)
}

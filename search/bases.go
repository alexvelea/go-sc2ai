package search

import (
	"log"

	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

type bases struct {
	Bases     []*Base
	distances []float32 // from i <-> j where i < j at index j*(j-1)/2 + i
	cache     map[api.Point2D]*Base
}

func (b *bases) distance(i, j int) float32 {
	if i == j {
		return 0
	}
	if i > j {
		i, j = j, i
	}
	return b.distances[j*(j-1)/2+i]
}

func newBases(m *Map, bot *botutil.Bot) bases {
	b := bases{cache: map[api.Point2D]*Base{}}

	locs := CalculateBaseLocations(bot, false)
	b.Bases = make([]*Base, len(locs))
	b.distances = make([]float32, len(locs)*(len(locs)-1)/2)

	query := make([]*api.RequestQueryPathing, 0, len(b.distances))
	for j, loc := range locs {
		b.Bases[j] = newBase(m, j, loc)

		// TODO: Compute distances via pathfinding rather than query
		for i := 0; i < j; i++ {
			// Sometimes path queries only work in one direction, or return different results (WHY?!?)
			// so we try both directions and take the max
			query = append(query, &api.RequestQueryPathing{
				Start: &api.RequestQueryPathing_StartPos{
					StartPos: &b.Bases[i].ResourceCenter,
				},
				EndPos: &b.Bases[j].ResourceCenter,
			}, &api.RequestQueryPathing{
				Start: &api.RequestQueryPathing_StartPos{
					StartPos: &b.Bases[j].ResourceCenter,
				},
				EndPos: &b.Bases[i].ResourceCenter,
			})

			// Should be at least as far as the crow flies, in case both queries fail this is better than nothing
			b.distances[j*(j-1)/2+i] = float32(b.Bases[i].ResourceCenter.Distance(b.Bases[j].ResourceCenter))
		}
	}

	resp := bot.Query(api.RequestQuery{Pathing: query})
	for k, r := range resp.Pathing {
		// Take the maximum computed distance
		if b.distances[k/2] < r.Distance {
			b.distances[k/2] = r.Distance
		}
	}

	return b
}

func (b *bases) update(bot *botutil.Bot) {
	bot.Neutral.Resources().Each(func(u botutil.Unit) {
		b.NearestBase(u.Pos2D()).updateResource(u)
	})

	for _, base := range b.Bases {
		base.update(bot)
	}

	bot.AllUnits().Each(func(u botutil.Unit) {
		if u.IsTownHall() {
			pos := u.Pos2D()
			base := b.NearestBase(pos)
			if base.TownHall.IsNil() || pos.Distance2(base.Location) < base.TownHall.Pos2D().Distance2(base.Location) {
				base.TownHall = u
			}
		} else if u.IsGasBuilding() {
			pos := u.Pos2D()
			b.NearestBase(pos).GasBuildings[pos] = u
		} else if u.IsWorker() {
			if u.Alliance != api.Alliance_Self {
				return
			}
			// update workers for all bases, if a worker is assigned
			for _, b := range b.Bases {
				if b.HasWorker(u.Tag) {
					b.Workers[u.Tag] = u
					return
				}
			}

			// check if worker needs to be assigned
			if !u.IsIdle() && !(u.Orders[0].AbilityId == ability.Harvest_Gather_SCV || u.Orders[0].AbilityId == ability.Harvest_Return_SCV) {
				//log.Printf("leaving SCV alone. tag: %v orders: %v", u.Tag, u.Orders[0])
				return
			}

			// assign worker to the nearest base which would want one
			base := b.NearestBaseIf(u.Pos2D(), func(b *Base) bool {
				return b.NeedsWorker(false)
			})
			if base != nil {
				log.Printf("assigned free worker to unsaturated mining tag: %v to %v workersInBase: %v", u.Tag, base.i, base.NumWorkers())
				base.addWorker(u)
				return
			}

			base = b.NearestBaseIf(u.Pos2D(), func(b *Base) bool {
				return b.NeedsOverSaturatedWorker(false)
			})
			if base != nil {
				log.Printf("assigned free worker to oversaturated mining tag: %v to %v workersInBase: %v", u.Tag, base.i, base.NumWorkers())
				base.addWorker(u)
				return
			}

			log.Printf("nothing to do with free worker tag: %v", u.Tag)
		}
	})

	// try to move workers from build bases to bases in construction
	// the number of steps required to move from one point to another, given SCV acceleration is
	// pos2.Distance(pos1)/builder.MovementSpeed*16+16
	// if this is lower than the StepsUntilFinished we should assign them
	for _, inConstruction := range b.Bases {
		if !inConstruction.IsUnderConstruction() {
			continue
		}

		for _, base := range b.Bases {
			if !base.IsFinished() {
				continue
			}
			//log.Printf("found inConstruction: %v and our own base: %v isOverSaturated: %v workers: %v", inConstruction.i, base.i, base.IsOverSaturated(), len(base.Workers))

			for {
				// has all the workers it needs
				if !inConstruction.NeedsWorker(true) {
					break
				}
				if !base.IsOverSaturated() {
					break
				}

				worker := base.PeakWorker()
				distance := worker.Pos2D().Distance(inConstruction.Location)
				travelTime := int(distance/worker.MovementSpeed*16 + 16)
				untilFinish := inConstruction.StepsUntilFinished()
				if travelTime < untilFinish {
					break
				}

				// move worker from one base to another
				log.Printf("moving over saturated worker: %v from base: %v to: %v travel: %v untilFinish: %v", worker.Tag, base.i, inConstruction.i, travelTime, untilFinish)
				worker = base.GetWorker()
				inConstruction.addWorker(worker)
			}
		}
	}

	// rebalance gas
	for _, base := range b.Bases {
		base.RebalanceToGas()
	}

	// step
	for _, base := range b.Bases {
		base.step(bot)
	}
}

// NearestBase ...
func (b *bases) NearestBase(pos api.Point2D) *Base {
	// Round to the nearest half tile
	pos.X, pos.Y = float32(int(pos.X*2))/2, float32(int(pos.Y*2))/2

	// Memoize resutls for faster repeated use
	base, ok := b.cache[pos]
	if !ok {
		base = b.NearestBaseIf(pos, func(*Base) bool { return true })
		b.cache[pos] = base
	}
	return base
}

// NearestBaseIf ...
func (b *bases) NearestBaseIf(pos api.Point2D, f func(*Base) bool) *Base {
	best, minDist := (*Base)(nil), float32(256*256)
	for _, e := range b.Bases {
		if dist := pos.Distance2(e.Location); dist < minDist && f(e) {
			best, minDist = e, dist
		}
	}
	return best
}

// NearestSelfBase ...
func (b *bases) NearestSelfBase(pos api.Point2D) *Base {
	return b.NearestBaseIf(pos, func(e *Base) bool {
		return !e.TownHall.IsNil() && e.TownHall.Alliance == api.Alliance_Self
	})
}

// NearestEnemyBase ...
func (b *bases) NearestEnemyBase(pos api.Point2D) *Base {
	return b.NearestBaseIf(pos, func(e *Base) bool {
		return !e.TownHall.IsNil() && e.TownHall.Alliance == api.Alliance_Enemy
	})
}

func (b *bases) MULEBase(pos api.Point2D) *Base {
	best, maxDist := (*Base)(nil), float32(-10)
	for _, e := range b.Bases {
		if dist := pos.Distance2(e.Location); dist > maxDist && e.IsFinished() && len(e.Minerals) > 0 {
			best, maxDist = e, dist
		}
	}
	return best
}

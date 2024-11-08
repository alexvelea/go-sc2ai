package main

import (
	"math"

	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

func closestToPos(points []api.Point2D, pos api.Point2D) api.Point2D {
	minDist := float32(math.Inf(1))
	var closest api.Point2D
	for _, p := range points {
		dist := pos.Distance2(p)
		if dist < minDist {
			closest = p
			minDist = dist
		}
	}
	return closest
}

type builderFetcher struct {
	done           bool
	initialRally   api.Point2D
	initialWorkers botutil.Units
}

func (r *builderFetcher) get(bot *bot, townHall botutil.Unit, dest api.Point2D) botutil.Unit {
	if r.done {
		return botutil.Unit{}
	}
	if r.initialWorkers.Len() == 0 {
		r.initialWorkers = bot.Self.All().IsWorker()
		r.initialRally = townHall.RallyTargets[0].Point.ToPoint2D()
		townHall.OrderPos(ability.Rally_CommandCenter, dest)
		return botutil.Unit{}
	}

	// figure out which SCV is new
	worker := bot.Self.All().IsWorker().Drop(func(u botutil.Unit) bool {
		return !r.initialWorkers.ByTag(u.Tag).IsNil()
	}).First()
	if worker.IsNil() {
		return worker
	}

	// set rally point back
	townHall.OrderPos(ability.Rally_CommandCenter, r.initialRally)
	r.done = true
	return worker
}

package main

import (
	"log"

	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
)

type raxFE struct {
	stepStrategy
	*bot

	positionForRefinery botutil.Unit

	builderFetcher builderFetcher
	builder        api.UnitTag

	refineryBuilderFetcher builderFetcher
	refineryBuilder        api.UnitTag
}

func (o *raxFE) Strategy() {
	if o.finished() {
		return
	}

	// try to fetch the newly created SCV and mark it as the builder
	if o.now() {
		builder := o.builderFetcher.get(o.bot, o.main.TownHall, o.positionsForSupplies[0])
		if builder.IsNil() {
			return
		}

		o.builder = builder.Tag
		o.advance()
	}

	builder := o.Self.All().ByTag(o.builder)
	o.mp.MarkWorkerAsUsed(builder)

	// move to depo location & build depo at 14
	if o.now() {
		builder.MoveTo(o.positionsForSupplies[0], 0)
		if o.FoodUsed == 14 && o.CanAfford(o.ProductionCost(terran.SupplyDepot, ability.Build_SupplyDepot)) {
			builder.BuildUnitAt(ability.Build_SupplyDepot, o.positionsForSupplies[0])
			o.advance()
		}
	}

	// wait for depo to finish
	if o.now() {
		if o.Self.Count(terran.SupplyDepot) == 1 && o.Self.CountInProduction(terran.SupplyDepot) == 0 {
			// set ramp depo & lower it
			ramp := o.Self.Structures().Choose(func(unit botutil.Unit) bool {
				return unit.UnitType == terran.SupplyDepot
			}).First()
			if ramp.IsNil() {
				return
			}
			o.rampSupply = ramp.Tag
			o.UnitOrder(ramp, ability.Morph_SupplyDepot_Lower)
			o.advance()
		}
	}

	// build rax
	if o.now() {
		if o.FoodUsed == 16 && o.CanAfford(o.ProductionCost(terran.Barracks, ability.Build_Barracks)) {
			log.Printf("building barracks")
			builder.BuildUnitAt(ability.Build_Barracks, o.positionsForProduction[0])
			o.advance()
		}
	}

	// == build first refinery ==

	// get refinery builder & location
	if o.now() {
		o.positionForRefinery = o.main.GetFreeGasGeyserPosition()
		builder := o.refineryBuilderFetcher.get(o.bot, o.main.TownHall, o.positionForRefinery.Pos2D())
		if builder.IsNil() {
			return
		}
		o.refineryBuilder = builder.Tag
		o.advance()
	}

	// try to build refinery
	if o.now() {
		builder := o.Self.All().ByTag(o.refineryBuilder)
		o.mp.MarkWorkerAsUsed(builder)

		if o.CanAfford(o.ProductionCost(terran.Refinery, ability.Build_Refinery)) {
			log.Printf("building refinery")
			builder.BuildUnitOn(ability.Build_Refinery, o.positionForRefinery)
			o.refineryBuilder = api.UnitTag(0)
			o.advance()

			// force next iteration now
			return
		} else {
			builder.MoveTo(o.positionForRefinery.Pos2D(), 1)
		}
	}

	// wait until rax is finished
	if o.now() {
		rax := o.bot.AllUnits().ByType(terran.Barracks).First()
		if rax.IsNil() || rax.BuildProgress != 1 {
			return
		}
		o.advance()
	}

	// go towards natural & expand
	if o.now() {
		builder.MoveTo(o.natural.Location, 0)
		if builder.Pos2D().Distance(o.natural.Location) > 3 {
			return
		}
		if o.CanAfford(o.ProductionCost(terran.CommandCenter, ability.Build_CommandCenter)) {
			log.Printf("building command center")
			builder.BuildUnitAt(ability.Build_CommandCenter, o.natural.Location)
			o.advance()
		}
	}
}

type firstRaxMarineThanReactor struct {
	stepStrategy
	*bot

	builder api.UnitTag
}

func (o *firstRaxMarineThanReactor) Strategy() {
	if o.finished() {
		return
	}

	if o.now() {
		rax := o.Self.All().ByPosition(o.positionsForProduction[0])
		if rax.IsNil() {
			return
		}
		if rax.BuildProgress != 1.0 {
			return
		}
		if o.CanAfford(o.ProductionCost(terran.Marine, ability.Train_Marine)) {
			rax.Order(ability.Train_Marine)
			o.advance()
		}
	}

	// build reactor
	if o.now() {
		rax := o.Self.All().ByPosition(o.positionsForProduction[0])
		if rax.IsIdle() && o.CanAfford(o.ProductionCost(terran.Reactor, ability.Build_Reactor)) {
			log.Printf("building reactor ...")
			rax.Order(ability.Build_Reactor)
			o.advance()
		}
	}
}

type secondRaxTechLabDepot struct {
	stepStrategy
	*bot

	builder api.UnitTag
}

func (o *secondRaxTechLabDepot) Strategy() {
	if o.finished() {
		return
	}

	// wait until natural is started
	if o.natural.TownHall.IsNil() {
		return
	}

	// move worker ahead of time towards barracks place
	if o.now() {
		if o.CanAfford(o.ProductionCost(terran.SupplyDepot, ability.Build_SupplyDepot)) {
			o.builder = o.main.GetWorker().Tag
			o.advance()
		}
	}

	builder := o.Self.All().ByTag(o.builder)
	o.mp.MarkWorkerAsUsed(builder)

	// build rax
	if o.now() {
		builder.MoveTo(o.positionsForProduction[1], 0)
		if o.CanAfford(o.ProductionCost(terran.Barracks, ability.Build_Barracks)) {
			log.Printf("building barracks")
			builder.BuildUnitAt(ability.Build_Barracks, o.positionsForProduction[1])
			o.builder = api.UnitTag(0)
			o.advance()
		}
	}

	// build supply depot
	if o.now() {
		o.builder = o.main.GetWorker().Tag
		o.advance()
	}

	builder = o.Self.All().ByTag(o.builder)
	o.mp.MarkWorkerAsUsed(builder)

	if o.now() {
		builder.MoveTo(o.positionsForSupplies[1], 0)
		if o.CanAfford(o.ProductionCost(terran.SupplyDepot, ability.Build_SupplyDepot)) {
			builder.BuildUnitAt(ability.Build_SupplyDepot, o.positionsForSupplies[1])
			o.advance()
		}
	}

	// build tech lab
	if o.now() {
		rax := o.Self.All().ByPosition(o.positionsForProduction[1])
		if rax.BuildProgress == 1.0 && rax.IsIdle() && o.CanAfford(o.ProductionCost(terran.TechLab, ability.Build_TechLab)) {
			log.Printf("building tech lab ...")
			rax.Order(ability.Build_TechLab)
			o.advance()
		}
	}
}

type factoryStarPortReactor struct {
	stepStrategy
	*bot

	builder         api.UnitTag
	refineryBuilder api.UnitTag
}

func (o *factoryStarPortReactor) Strategy() {
	if o.finished() {
		return
	}

	// wait until we have 2 rax
	if o.bot.Self.All().ByType(terran.Barracks).Len() < 2 {
		return
	}

	// move worker ahead of time towards barracks place
	if o.now() {
		if o.Vespene > 80 {
			o.builder = o.main.GetWorker().Tag
			o.advance()
		}
	}

	builder := o.Self.All().ByTag(o.builder)
	o.mp.MarkWorkerAsUsed(builder)

	// build factory
	if o.now() {
		builder.MoveTo(o.positionsForProduction[2], 0)
		if o.CanAfford(o.ProductionCost(terran.Factory, ability.Build_Factory)) {
			log.Printf("building factory")
			builder.BuildUnitAt(ability.Build_Factory, o.positionsForProduction[2])
			o.advance()
		}
	}

	// get worker for gas
	if o.now() {
		o.refineryBuilder = o.main.GetWorker().Tag
		o.advance()
	}

	refineryBuilder := o.Self.All().ByTag(o.refineryBuilder)
	o.mp.MarkWorkerAsUsed(refineryBuilder)

	// build refinery
	if o.now() {
		positionForRefinery := o.main.GetFreeGasGeyserPosition()
		if o.CanAfford(o.ProductionCost(terran.Refinery, ability.Build_Refinery)) {
			log.Printf("building refinery")
			refineryBuilder.BuildUnitOn(ability.Build_Refinery, positionForRefinery)
			o.refineryBuilder = api.UnitTag(0)
			o.advance()
		} else {
			refineryBuilder.MoveTo(positionForRefinery.Pos2D(), 1)
		}
	}

	// wait until factory is build
	if o.now() {
		factory := o.Self.All().ByType(terran.Factory).First()
		if !factory.IsNil() && factory.BuildProgress == 1.0 {
			o.advance()
		}
	}

	// build StarPort
	if o.now() {
		if o.CanAfford(o.ProductionCost(terran.Starport, ability.Build_Starport)) {
			log.Printf("building StarPort")
			builder.BuildUnitAt(ability.Build_Starport, o.positionsForProduction[3])
			o.builder = api.UnitTag(0)
			o.advance()
		} else {
			builder.MoveTo(o.positionsForProduction[3], 1)
		}
	}

	// build reactor
	if o.now() {
		factory := o.Self.All().ByType(terran.Factory).First()
		if factory.BuildProgress == 1.0 && factory.IsIdle() && o.CanAfford(o.ProductionCost(terran.Reactor, ability.Build_Reactor)) {
			log.Printf("building reactor on factory ...")
			factory.Order(ability.Build_Reactor)
			o.advance()
		}
	}

	// wait until factory & starport are finished, lift both of them and land on top of the other
	if o.now() {
		factory := o.Self.All().ByTypes([]api.UnitTypeID{terran.Factory, terran.FactoryFlying}).First()
		starport := o.Self.All().ByTypes([]api.UnitTypeID{terran.Starport, terran.StarportFlying}).First()
		cnt := 0

		if !factory.IsNil() && len(factory.Orders) == 0 {
			if factory.UnitType != terran.FactoryFlying {
				factory.Order(ability.Lift_Factory)
			} else {
				factory.MoveTo(o.positionsForProduction[3], 0)
				cnt += 1
			}
		}
		if !starport.IsNil() && starport.BuildProgress == 1.0 {
			if starport.UnitType != terran.StarportFlying {
				starport.Order(ability.Lift_Starport)
			} else {
				starport.MoveTo(o.positionsForProduction[2], 0)
			}
			cnt += 1
		}

		if cnt == 2 {
			o.advance()
		}
	}

	// land
	if o.now() {
		cnt := 0

		factory := o.Self.All().ByTypes([]api.UnitTypeID{terran.Factory, terran.FactoryFlying}).First()
		starport := o.Self.All().ByTypes([]api.UnitTypeID{terran.Starport, terran.StarportFlying}).First()

		if factory.UnitType == terran.FactoryFlying {
			factory.OrderPos(ability.Land_Factory, o.positionsForProduction[3])
		} else {
			cnt += 1
		}
		if starport.UnitType == terran.StarportFlying {
			starport.OrderPos(ability.Land_Starport, o.positionsForProduction[2])
		} else {
			cnt += 1
		}

		if cnt == 2 {
			o.advance()
		}
	}
}

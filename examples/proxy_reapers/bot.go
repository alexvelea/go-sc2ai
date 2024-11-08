package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
	"github.com/chippydip/go-sc2ai/client"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/search"
)

var (
	GameDuration = GameLoopMin * 5
	GameSpeed    = 100.0

	GameLoopMin uint32  = 224 * 6
	GameLoopPer         = time.Second * 60 / time.Duration(GameLoopMin)
	StepsPerSec float32 = float32(GameLoopMin) / 60
)

type bot struct {
	*botutil.Bot

	mp      *search.Map
	main    *search.Base
	natural *search.Base

	myStartLocation    api.Point2D
	enemyStartLocation api.Point2D

	positionsForSupplies   []api.Point2D
	positionsForProduction []api.Point2D
	defendPoint            api.Point2D

	rampSupply api.UnitTag

	strategies []Strategy
}

func runAgent(info client.AgentInfo) {
	bot := bot{Bot: botutil.NewBot(info)}
	bot.LogActionErrors()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)

	bot.init()
	for bot.IsInGame() {
		select {
		case <-sigCh:
			bot.SendDebugCommands([]*api.DebugCommand{
				{
					Command: &api.DebugCommand_EndGame{
						EndGame: &api.DebugEndGame{
							EndResult: api.DebugEndGame_Surrender,
						},
					},
				},
			})
			bot.LeaveGame()
			break
		default:
		}

		bot.Strategy()
		bot.tactics()

		if err := bot.Step(1); err != nil {
			log.Print(err)
			break
		}

		if bot.GameLoop > GameDuration {
			bot.SendDebugCommands([]*api.DebugCommand{
				{
					Command: &api.DebugCommand_EndGame{
						EndGame: &api.DebugEndGame{
							EndResult: api.DebugEndGame_Surrender,
						},
					},
				},
			})
			bot.LeaveGame()
		}
		<-time.After(GameLoopPer / time.Duration(GameSpeed))
	}
}

func (bot *bot) init() {
	bot.initLocations()

	bot.mp = search.NewMap(bot.Bot)
	bot.main = bot.mp.NearestBase(bot.myStartLocation)
	bot.natural = bot.main.Natural()

	bot.strategies = append(bot.strategies, &raxFE{bot: bot})
	bot.strategies = append(bot.strategies, &firstRaxMarineThanReactor{bot: bot})
	bot.strategies = append(bot.strategies, &secondRaxTechLabDepot{bot: bot})
	bot.strategies = append(bot.strategies, &factoryStarPortReactor{bot: bot})

	log.Printf("MyLocation: %v main: %v", bot.myStartLocation, bot.main.Location)

	//search.CalculateRampLocations(bot.Bot, true)

	bot.findBuildingsPositions()

	// Send a friendly hello
	bot.Chat("(glhf)")
}

func (bot *bot) initLocations() {
	// My CC is on start position
	bot.myStartLocation = bot.Self[terran.CommandCenter].First().Pos2D()
	bot.enemyStartLocation = *bot.GameInfo().StartRaw.StartLocations[0]
}

func (bot *bot) findBuildingsPositions() {
	if bot.myStartLocation.X < 100 {
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: -3, Y: -13}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: -6, Y: -16}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: -11, Y: -6}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: -11, Y: -8}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: -11, Y: -10}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: -11, Y: -12}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: -11, Y: -14}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: -11, Y: -16}))

		bot.positionsForProduction = append(bot.positionsForProduction, bot.myStartLocation.Add(api.Vec2D{X: -7, Y: -13}))
		bot.positionsForProduction = append(bot.positionsForProduction, bot.myStartLocation.Add(api.Vec2D{X: +5, Y: -9}))
		bot.positionsForProduction = append(bot.positionsForProduction, bot.myStartLocation.Add(api.Vec2D{X: +5, Y: -5}))
		bot.positionsForProduction = append(bot.positionsForProduction, bot.myStartLocation.Add(api.Vec2D{X: +5, Y: -1}))

		bot.defendPoint = bot.natural.Location.Add(api.Vec2D{X: +10, Y: 0})
	} else {
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: +2, Y: -13}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: +5, Y: -16}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: +10, Y: -6}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: +10, Y: -8}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: +10, Y: -10}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: +10, Y: -12}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: +10, Y: -14}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: +10, Y: -16}))
		bot.positionsForSupplies = append(bot.positionsForSupplies, bot.myStartLocation.Add(api.Vec2D{X: +10, Y: -18}))

		bot.positionsForProduction = append(bot.positionsForProduction, bot.myStartLocation.Add(api.Vec2D{X: +5, Y: -13}))
		bot.positionsForProduction = append(bot.positionsForProduction, bot.myStartLocation.Add(api.Vec2D{X: -7, Y: -9}))
		bot.positionsForProduction = append(bot.positionsForProduction, bot.myStartLocation.Add(api.Vec2D{X: -7, Y: -5}))
		bot.positionsForProduction = append(bot.positionsForProduction, bot.myStartLocation.Add(api.Vec2D{X: -7, Y: -1}))

		bot.defendPoint = bot.natural.Location.Add(api.Vec2D{X: -10, Y: 0})
	}

	// Pick locations for supply depots
	//pos := bot.myStartLocation.Offset(homeMinerals.Center(), -7)
	//neighbors8 := pos.Offset8By(2)
	//bot.positionsForSupplies = append(append(bot.positionsForSupplies, pos), neighbors8[:]...)

	//// Determine proxy location
	//pos = bot.enemyStartLocation.Offset(bot.myStartLocation, 25)
	//pos = closestToPos(bot.baseLocations, pos).Offset(bot.myStartLocation, 1)
	//bot.positionsForProduction = pos
	//
	//// Build a re-usable query to check if we can build barracks
	//bot.barracksQuery = botutil.NewQuery(bot)
	//bot.barracksQuery.IgnoreResourceRequirements()
	//
	//bot.barracksQuery.Placement(ability.Build_Barracks, pos)
	//for _, np := range pos.Offset4By(4) {
	//	bot.barracksQuery.Placement(ability.Build_Barracks, np)
	//}
}

func (bot *bot) buildSCVs() {
	ccs := bot.Self.Structures().Choose(func(unit botutil.Unit) bool {
		return unit.IsFlying == false && unit.BuildProgress == 1.0 &&
			(unit.UnitType == terran.OrbitalCommand || unit.UnitType == terran.CommandCenter || unit.UnitType == terran.PlanetaryFortress)
	}).All()

	for _, cc := range ccs.Slice() {
		if len(cc.Orders) != 0 {
			continue
		}

		// if it's a CC, and we have a rax, we could upgrade to orbital
		if cc.UnitType == terran.CommandCenter && bot.CanAfford(bot.ProductionCost(terran.CommandCenter, ability.Morph_OrbitalCommand)) {
			rax := bot.Self.Count(terran.Barracks)
			raxInConstruction := bot.Self.CountInProduction(terran.Barracks)
			if rax-raxInConstruction > 0 {
				bot.UnitOrder(cc, ability.Morph_OrbitalCommand)
				continue
			} else {
				rax := bot.Self.ByType(terran.Barracks).First()
				if !rax.IsNil() {
					steps := (1.0 - rax.BuildProgress) * rax.BuildTime
					// can wait 4 seconds to morph
					if steps/StepsPerSec < 4 {
						// wait for rax to finish
						continue
					}
				}
			}
		}

		if cc.UnitType == terran.OrbitalCommand && cc.CanOrder(ability.Effect_CalldownMULE) {
			cc.OrderTarget(ability.Effect_CalldownMULE, bot.mp.MULEBase(bot.main.Location).MULEPatch())
		}

		if bot.Self.CountAll(terran.SCV) < 30 {
			cc.Order(ability.Train_SCV)
		}
	}
}

func (bot *bot) Strategy() {
	defer func() { search.ShowDebugBoxes(bot.Bot) }()
	bot.mp.Update()
	for _, s := range bot.strategies {
		s.Strategy()
	}

	bot.buildSCVs()

	// check if we should build supply depots
	depotCount := bot.Self.Count(terran.SupplyDepot) + bot.Self.Count(terran.SupplyDepotLowered)
	if bot.FoodCap >= 46 && bot.FoodLeft() < 12 && bot.Self.CountInProduction(terran.SupplyDepot) == 0 && depotCount < len(bot.positionsForSupplies) {
		if bot.CanAfford(bot.ProductionCost(terran.SupplyDepot, ability.Build_SupplyDepot)) {
			worker := bot.main.GetWorker()
			worker.BuildUnitAt(ability.Build_SupplyDepot, bot.positionsForSupplies[depotCount])
		} else {
			return
		}
	}

	// build units
	bot.Self.ByType(terran.Barracks).Each(func(unit botutil.Unit) {
		// set rally point
		unit.RallyTo(bot.defendPoint, 0.5)

		addOn := bot.Self.All().ByTag(unit.AddOnTag)
		if !addOn.IsNil() && addOn.UnitType == terran.BarracksReactor {
			for i := len(unit.Orders); i < 2; i += 1 {
				unit.Order(ability.Train_Marine)
			}
		}
		if !addOn.IsNil() && addOn.UnitType == terran.BarracksTechLab {
			for i := len(unit.Orders); i < 1; i += 1 {
				unit.Order(ability.Train_Marine)
			}
		}
	})
	bot.Self.ByType(terran.Starport).Each(func(unit botutil.Unit) {
		unit.RallyTo(bot.defendPoint, 0.5)

		addOn := bot.Self.All().ByTag(unit.AddOnTag)
		if !addOn.IsNil() && addOn.UnitType == terran.StarportReactor {
			for i := len(unit.Orders); i < 2; i += 1 {
				unit.Order(ability.Train_Medivac)
			}
		}
	})

	//// expand to natural
	//if bot.natural.TownHall.IsNil() && bot.CanAfford(bot.ProductionCost(terran.CommandCenter, ability.Build_CommandCenter)) {
	//	worker := bot.main.GetWorker()
	//	if worker.IsNil() {
	//		log.Printf("worker is nil!")
	//		return
	//	}
	//	worker.BuildUnitAt(ability.Build_CommandCenter, bot.natural.Location)
	//	log.Printf("building command center! worker: %v location: %v", worker.Tag, bot.natural.Location)
	//}
}

func (bot *bot) tactics() {
}

//func (bot *bot) tactics() {
//	// Attack!
//	reapers := bot.Self[terran.Reaper]
//	if reapers.Len() == 0 {
//		return
//	}
//
//	targets := bot.getTargets()
//	if targets.Len() == 0 {
//		bot.UnitsOrderPos(reapers, ability.Attack, bot.enemyStartLocation)
//		return
//	}
//
//	reapers.Each(func(reaper botutil.Unit) {
//		// retreat
//		if bot.retreat[reaper.Tag] && reaper.Health > 50 {
//			delete(bot.retreat, reaper.Tag)
//		}
//		if reaper.Health < 21 || bot.retreat[reaper.Tag] {
//			bot.retreat[reaper.Tag] = true
//			reaper.OrderPos(ability.Move, bot.positionsForBarracks)
//			return
//		}
//
//		target := targets.ClosestTo(reaper.Pos2D())
//
//		// Keep range
//		// Weapon is recharging
//		if reaper.WeaponCooldown > 1 {
//			// Enemy is closer than shooting distance - 0.5
//			if reaper.IsInWeaponsRange(target, -0.5) {
//				// Retreat a little
//				reaper.OrderPos(ability.Move, bot.positionsForBarracks)
//				return
//			}
//		}
//
//		// Attack
//		if reaper.Pos2D().Distance2(target.Pos2D()) > 4*4 {
//			// If target is far, attack it as unit, ling will run ignoring everything else
//			reaper.OrderTarget(ability.Attack, target)
//		} else if target.UnitType == zerg.ChangelingMarine || target.UnitType == zerg.ChangelingMarineShield {
//			// Must specificially attack changelings, attack move is not enough
//			reaper.OrderTarget(ability.Attack, target)
//		} else {
//			// Attack as position, ling will choose best target around
//			reaper.OrderPos(ability.Attack, target.Pos2D())
//		}
//	})
//}

func (bot *bot) getTargets() botutil.Units {
	// Prioritize things that can fight back
	if targets := bot.Enemy.Ground().CanAttack().All(); targets.Len() > 0 {
		return targets
	}

	// Otherwise just kill all the buildings
	return bot.Enemy.Ground().Structures().All()
}

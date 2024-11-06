package search

import (
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/botutil"
)

// Map ...
type Map struct {
	bot *botutil.Bot
	bases

	StartLocation api.Point2D
	PlacementGrid *PlacementGrid
}

// NewMap ...
func NewMap(bot *botutil.Bot) *Map {
	m := &Map{
		bot:           bot,
		PlacementGrid: NewPlacementGrid(bot),
	}
	m.bases = newBases(m, bot)
	m.StartLocation = bot.Self.Structures().First().Pos2D()

	m.Update()

	return m
}

func (m *Map) Update() {
	m.bases.update(m.bot)
	m.PlacementGrid.Update()
}

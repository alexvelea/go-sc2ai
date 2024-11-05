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
}

// NewMap ...
func NewMap(bot *botutil.Bot) *Map {
	m := &Map{
		bot: bot,
	}
	m.bases = newBases(m, bot)
	m.StartLocation = bot.Self.Structures().First().Pos2D()

	return m
}

func (m *Map) Update() {
	m.bases.update(m.bot)
}

// Copyright 2026 Team 254. All Rights Reserved.

package field

import (
	"testing"

	"github.com/Team254/cheesy-arena-lite/partner"
	"github.com/stretchr/testify/assert"
)

const testSeasonBody = `{
      "schema": 2, "key": "xyz-2030-teste", "revision": 1, "program": "TESTE",
      "name": "Temporada de teste", "year": 2030, "format": "alliance", "robotsPerAlliance": 2,
      "periods": [{ "id": "teleop", "label": "Teleoperado", "durationSec": 100, "scoring": true }],
      "eventLevels": [{ "id": "EVENTO", "label": "Evento" }],
      "thresholds": { "meta": { "*": 10 } },
      "categories": [{ "id": "pontos", "label": "Pontos" }],
      "actions": [
        { "id": "cesta", "label": "Cesta", "category": "pontos", "unit": "count",
          "periods": ["teleop"], "points": { "teleop": 5 } }],
      "ranking": {
        "mode": "alliance", "rsDecimals": 2, "result": { "win": 3, "tie": 1, "loss": 0 },
        "rankingPoints": [], "tiebreakers": [{ "id": "total", "label": "Total", "expr": ["total"] }] },
      "tests": [
        { "name": "uma cesta dá 5", "level": "EVENTO",
          "state": { "actions": { "cesta@teleop": 1 } },
          "expect": { "total": 5, "rpEarned": [] } }]
    }`

func TestAdoptSeasonFromBootstrapWritesSettings(t *testing.T) {
	arena := setupTestArena(t)

	boot := &partner.ArenaBootstrap{}
	boot.Event.Season = &partner.ArenaSeason{
		Key: "xyz-2030-teste", Revision: 1, Hash: "abc123", Level: "EVENTO", Body: testSeasonBody,
	}

	arena.adoptSeasonFromBootstrap(boot)

	assert.Equal(t, "xyz-2030-teste", arena.EventSettings.SeasonKey)
	assert.Equal(t, 1, arena.EventSettings.SeasonRevision)
	assert.Equal(t, "abc123", arena.EventSettings.SeasonHash)
	assert.Equal(t, testSeasonBody, arena.EventSettings.SeasonBody)
	assert.Equal(t, "EVENTO", arena.EventSettings.EventLevel)

	reloaded, err := arena.Database.GetEventSettings()
	assert.Nil(t, err)
	assert.Equal(t, "xyz-2030-teste", reloaded.SeasonKey)
}

func TestAdoptSeasonFromBootstrapSkipsWhenHashUnchanged(t *testing.T) {
	arena := setupTestArena(t)
	arena.EventSettings.SeasonKey = "already-here"
	arena.EventSettings.SeasonHash = "same-hash"
	assert.Nil(t, arena.Database.UpdateEventSettings(arena.EventSettings))

	boot := &partner.ArenaBootstrap{}
	boot.Event.Season = &partner.ArenaSeason{
		Key: "xyz-2030-teste", Revision: 1, Hash: "same-hash", Body: testSeasonBody,
	}

	arena.adoptSeasonFromBootstrap(boot)

	assert.Equal(t, "already-here", arena.EventSettings.SeasonKey)
}

func TestAdoptSeasonFromBootstrapRejectsPackageThatFailsItsOwnTests(t *testing.T) {
	arena := setupTestArena(t)
	arena.EventSettings.SeasonKey = "already-here"
	assert.Nil(t, arena.Database.UpdateEventSettings(arena.EventSettings))

	broken := replaceOnceForTest(testSeasonBody, `"total": 5`, `"total": 6`)
	boot := &partner.ArenaBootstrap{}
	boot.Event.Season = &partner.ArenaSeason{
		Key: "xyz-2030-teste", Revision: 1, Hash: "some-hash", Body: broken,
	}

	arena.adoptSeasonFromBootstrap(boot)

	assert.Equal(t, "already-here", arena.EventSettings.SeasonKey)
}

func TestAdoptSeasonFromBootstrapDoesNothingWithoutASeason(t *testing.T) {
	arena := setupTestArena(t)
	arena.EventSettings.SeasonKey = "already-here"
	assert.Nil(t, arena.Database.UpdateEventSettings(arena.EventSettings))

	arena.adoptSeasonFromBootstrap(&partner.ArenaBootstrap{})

	assert.Equal(t, "already-here", arena.EventSettings.SeasonKey)
}

func replaceOnceForTest(text, old, replacement string) string {
	for i := 0; i+len(old) <= len(text); i++ {
		if text[i:i+len(old)] == old {
			return text[:i] + replacement + text[i+len(old):]
		}
	}
	return text
}

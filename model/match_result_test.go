// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNonexistentMatchResult(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	match, err := db.GetMatchResultForMatch(1114)
	assert.Nil(t, err)
	assert.Nil(t, match)
}

func TestMatchResultCrud(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	matchResult := BuildTestMatchResult(254, 5)
	assert.Nil(t, db.CreateMatchResult(matchResult))
	matchResult2, err := db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Equal(t, matchResult, matchResult2)

	matchResult.BlueScore.LegacyEndgamePoints = 1234
	assert.Nil(t, db.UpdateMatchResult(matchResult))
	matchResult2, err = db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Equal(t, matchResult, matchResult2)

	assert.Nil(t, db.DeleteMatchResult(matchResult.Id))
	matchResult2, err = db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Nil(t, matchResult2)
}

func TestTruncateMatchResults(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	matchResult := BuildTestMatchResult(254, 1)
	assert.Nil(t, db.CreateMatchResult(matchResult))
	assert.Nil(t, db.TruncateMatchResults())
	matchResult2, err := db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Nil(t, matchResult2)
}

func TestGetMatchResultForMatch(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	matchResult := BuildTestMatchResult(254, 2)
	assert.Nil(t, db.CreateMatchResult(matchResult))
	matchResult2 := BuildTestMatchResult(254, 5)
	assert.Nil(t, db.CreateMatchResult(matchResult2))
	matchResult3 := BuildTestMatchResult(254, 4)
	assert.Nil(t, db.CreateMatchResult(matchResult3))

	// Should return the match result with the highest play number (i.e. the most recent).
	matchResult4, err := db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Equal(t, matchResult2, matchResult4)
}

// The bytes an older build wrote have to decode to the same score they always meant. This is the
// whole compatibility promise of the seasons: the four numbers kept their names on the wire and in
// the database, and only the Go fields were renamed.
func TestAMatchResultFromBeforeTheSeasonsReadsTheSame(t *testing.T) {
	database := setupTestDb(t)
	defer database.Close()

	stored := `{"Id":1,"MatchId":7,"PlayNumber":1,"MatchType":"qualification",` +
		`"RedScore":{"AutoPoints":45,"TeleopPoints":80,"EndgamePoints":10,"FoulPoints":3},` +
		`"BlueScore":{"AutoPoints":15,"TeleopPoints":60,"EndgamePoints":50,"FoulPoints":0},` +
		`"Official":true}`
	matchResult := new(MatchResult)
	assert.Nil(t, json.Unmarshal([]byte(stored), matchResult))

	assert.False(t, matchResult.RedScore.HasTally())
	red := matchResult.RedScoreSummary()
	assert.Equal(t, 45, red.AutoPoints)
	assert.Equal(t, 80, red.TeleopPoints)
	assert.Equal(t, 10, red.EndgamePoints)
	assert.Equal(t, 3, red.FoulPoints)
	assert.Equal(t, 138, red.Score)
	blue := matchResult.BlueScoreSummary()
	assert.Equal(t, 125, blue.Score)

	// And it survives a round trip through the store, which is where the json tags earn their keep.
	matchResult.Id = 0
	assert.Nil(t, database.CreateMatchResult(matchResult))
	again, err := database.GetMatchResultForMatch(7)
	assert.Nil(t, err)
	if assert.NotNil(t, again) {
		assert.Equal(t, 138, again.RedScoreSummary().Score)
		assert.Equal(t, 45, again.RedScore.LegacyAutoPoints)
	}

	body, err := json.Marshal(again)
	assert.Nil(t, err)
	assert.Contains(t, string(body), `"AutoPoints":45`)
	assert.NotContains(t, string(body), "LegacyAutoPoints")
}

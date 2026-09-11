package web

import (
	"testing"

	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/stretchr/testify/assert"
)

func commitOneQualificationMatch(t *testing.T, web *Web) {
	match := model.Match{Type: "qualification", DisplayName: "1", Red1: 101, Red2: 102,
		Blue1: 103, Blue2: 104}
	assert.Nil(t, web.arena.Database.CreateMatch(&match))
	result := model.NewMatchResult()
	result.MatchId = match.Id
	result.MatchType = match.Type
	assert.Nil(t, web.commitMatchScore(&match, result, false))
}

func TestCommitUpdatesRankingsOutsideFll(t *testing.T) {
	web := setupTestWeb(t)
	defer web.arena.Database.Close()
	web.arena.EventSettings.IsFll = false

	web.suppressRankingOnCommit = false
	commitOneQualificationMatch(t, web)

	rankings, err := web.arena.Database.GetAllRankings()
	assert.Nil(t, err)
	assert.Equal(t, 4, len(rankings))
}

func TestCommitKeepsUnofficialFllRoundsOutOfRankings(t *testing.T) {
	web := setupTestWeb(t)
	defer web.arena.Database.Close()
	web.arena.EventSettings.IsFll = true

	web.suppressRankingOnCommit = true
	commitOneQualificationMatch(t, web)
	web.suppressRankingOnCommit = false

	rankings, err := web.arena.Database.GetAllRankings()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(rankings))
}

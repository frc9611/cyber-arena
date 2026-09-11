// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package game

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

// The random tiebreaker is not drawn here: it belongs to the team for the whole event, and
// CalculateRankings keeps the one it already had. Drawing it per result is what made a tied pair
// trade places on every recalculation.
func TestAddScoreSummary(t *testing.T) {
	redScore := TestScore1()
	blueScore := TestScore2()
	redSummary := redScore.Summarize()
	blueSummary := blueScore.Summarize()
	rankingFields := RankingFields{}

	// Add a loss.
	rankingFields.AddScoreSummary(redSummary, blueSummary)
	assert.Equal(t, RankingFields{2, 45, 30, 80, 0, 1, 0, 0, 1}, rankingFields)

	// Add a win.
	rankingFields.AddScoreSummary(blueSummary, redSummary)
	assert.Equal(t, RankingFields{2, 60, 55, 120, 0, 1, 1, 0, 2}, rankingFields)

	// Add a tie.
	rankingFields.AddScoreSummary(redSummary, redSummary)
	assert.Equal(t, RankingFields{3, 105, 85, 200, 0, 1, 1, 1, 3}, rankingFields)
}

func TestAddScoreSummaryKeepsTheDrawnTiebreaker(t *testing.T) {
	summary := TestScore1().Summarize()
	rankingFields := RankingFields{Random: 0.4242}
	rankingFields.AddScoreSummary(summary, summary)
	assert.Equal(t, 0.4242, rankingFields.Random)
}

func TestTeamWithNoMatchRanksLast(t *testing.T) {
	rankings := Rankings{
		{TeamId: 7, RankingFields: RankingFields{}},
		{TeamId: 3, RankingFields: RankingFields{RankingPoints: 1, Played: 4}},
		{TeamId: 9, RankingFields: RankingFields{}},
		{TeamId: 5, RankingFields: RankingFields{RankingPoints: 12, Played: 4}},
	}
	sort.Stable(rankings)
	assert.Equal(t, []int{5, 3, 7, 9}, []int{
		rankings[0].TeamId, rankings[1].TeamId, rankings[2].TeamId, rankings[3].TeamId})
}

func TestSortRankings(t *testing.T) {
	// Check tiebreakers.
	rankings := make(Rankings, 10)
	rankings[0] = Ranking{1, 0, 0, RankingFields{50, 50, 50, 50, 0.49, 3, 2, 1, 10}}
	rankings[1] = Ranking{2, 0, 0, RankingFields{50, 50, 50, 50, 0.51, 3, 2, 1, 10}}
	rankings[2] = Ranking{3, 0, 0, RankingFields{50, 50, 50, 49, 0.50, 3, 2, 1, 10}}
	rankings[3] = Ranking{4, 0, 0, RankingFields{50, 50, 50, 51, 0.50, 3, 2, 1, 10}}
	rankings[4] = Ranking{5, 0, 0, RankingFields{50, 50, 49, 50, 0.50, 3, 2, 1, 10}}
	rankings[5] = Ranking{6, 0, 0, RankingFields{50, 50, 51, 50, 0.50, 3, 2, 1, 10}}
	rankings[6] = Ranking{7, 0, 0, RankingFields{50, 49, 50, 50, 0.50, 3, 2, 1, 10}}
	rankings[7] = Ranking{8, 0, 0, RankingFields{50, 51, 50, 50, 0.50, 3, 2, 1, 10}}
	rankings[8] = Ranking{9, 0, 0, RankingFields{49, 50, 50, 50, 0.50, 3, 2, 1, 10}}
	rankings[9] = Ranking{10, 0, 0, RankingFields{51, 50, 50, 50, 0.50, 3, 2, 1, 10}}
	sort.Sort(rankings)
	assert.Equal(t, 10, rankings[0].TeamId)
	assert.Equal(t, 8, rankings[1].TeamId)
	assert.Equal(t, 6, rankings[2].TeamId)
	assert.Equal(t, 4, rankings[3].TeamId)
	assert.Equal(t, 2, rankings[4].TeamId)
	assert.Equal(t, 1, rankings[5].TeamId)
	assert.Equal(t, 3, rankings[6].TeamId)
	assert.Equal(t, 5, rankings[7].TeamId)
	assert.Equal(t, 7, rankings[8].TeamId)
	assert.Equal(t, 9, rankings[9].TeamId)

	// Check with unequal number of matches played.
	rankings = make(Rankings, 3)
	rankings[0] = Ranking{1, 0, 0, RankingFields{10, 25, 25, 25, 0.49, 3, 2, 1, 5}}
	rankings[1] = Ranking{2, 0, 0, RankingFields{19, 50, 50, 50, 0.51, 3, 2, 1, 9}}
	rankings[2] = Ranking{3, 0, 0, RankingFields{20, 50, 50, 50, 0.51, 3, 2, 1, 10}}
	sort.Sort(rankings)
	assert.Equal(t, 2, rankings[0].TeamId)
	assert.Equal(t, 3, rankings[1].TeamId)
	assert.Equal(t, 1, rankings[2].TeamId)
}

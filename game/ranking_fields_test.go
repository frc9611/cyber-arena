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
// Without a season, a win is worth what it was worth before the seasons existed.
func TestAddScoreSummary(t *testing.T) {
	redScore := TestScore1()
	blueScore := TestScore2()
	redSummary := redScore.Summarize()
	blueSummary := blueScore.Summarize()
	rankingFields := RankingFields{}

	// Add a win.
	rankingFields.AddScoreSummary(nil, redSummary, blueSummary, EntryNormal)
	assert.Equal(t, 2, rankingFields.RankingPoints)
	assert.Equal(t, 1, rankingFields.Wins)
	assert.Equal(t, 1, rankingFields.Played)
	assert.Equal(t, 45, rankingFields.AutoPoints)

	// Add a loss.
	rankingFields.AddScoreSummary(nil, blueSummary, redSummary, EntryNormal)
	assert.Equal(t, 2, rankingFields.RankingPoints)
	assert.Equal(t, 1, rankingFields.Losses)
	assert.Equal(t, 2, rankingFields.Played)

	// Add a tie.
	rankingFields.AddScoreSummary(nil, redSummary, redSummary, EntryNormal)
	assert.Equal(t, 3, rankingFields.RankingPoints)
	assert.Equal(t, 1, rankingFields.Ties)
	assert.Equal(t, 3, rankingFields.Played)
	assert.Equal(t, 105, rankingFields.AutoPoints)
}

// REEFSCAPE pays 3 for a win, and this fork paid 2 to every season that ever existed.
func TestTheSeasonSaysWhatAWinIsWorth(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	own := &ScoreSummary{Score: 54, Outcome: &MatchOutcome{RankingPoints: 1, Sort: []int{1, 54, 30, 12}}}
	other := &ScoreSummary{Score: 20, Outcome: &MatchOutcome{Sort: []int{0, 20, 0, 0}}}

	fields := RankingFields{}
	fields.AddScoreSummary(season, own, other, EntryNormal)
	assert.Equal(t, 4, fields.RankingPoints, "3 pela vitória mais o AUTO RP")
	assert.Equal(t, []int{1, 54, 30, 12}, fields.Sort)
	assert.Equal(t, 2, fields.RsDecimals)
}

// A surrogate appearance counts as played, pays nothing and feeds no tiebreaker. Skipping it was a
// fourth behaviour no manual describes.
func TestTheSeasonSaysWhatASurrogateIsWorth(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	own := &ScoreSummary{Score: 54, Outcome: &MatchOutcome{RankingPoints: 1, Sort: []int{1, 54, 30, 12}}}
	other := &ScoreSummary{Score: 20, Outcome: &MatchOutcome{Sort: []int{0, 20, 0, 0}}}

	fields := RankingFields{}
	fields.AddScoreSummary(season, own, other, EntrySurrogate)
	assert.Equal(t, 1, fields.Played)
	assert.Equal(t, 0, fields.RankingPoints)
	assert.Equal(t, 0, fields.Wins)
	assert.Equal(t, []int{0, 0, 0, 0}, fields.Sort)

	disqualified := RankingFields{}
	disqualified.AddScoreSummary(season, own, other, EntryDisqualified)
	assert.Equal(t, 1, disqualified.Played)
	assert.Equal(t, 1, disqualified.Disqualifications)
	assert.Equal(t, 0, disqualified.RankingPoints)
	assert.Equal(t, []int{0, 0, 0, 0}, disqualified.Sort)
}

// The official first criterion is the average rounded to two places. 19 in 11 and 26 in 15 are both
// 1.73 and tie; cross-multiplying separates them by one point out of 286.
func TestTheRankingScoreIsTheRoundedAverage(t *testing.T) {
	a := Ranking{TeamId: 1, RankingFields: RankingFields{
		RankingPoints: 19, Played: 11, RsDecimals: 2, Sort: []int{5}}}
	b := Ranking{TeamId: 2, RankingFields: RankingFields{
		RankingPoints: 26, Played: 15, RsDecimals: 2, Sort: []int{9}}}
	assert.Equal(t, 173, a.RankingScore())
	assert.Equal(t, 173, b.RankingScore())

	rankings := Rankings{a, b}
	SortRankings(rankings, nil)
	assert.Equal(t, 2, rankings[0].TeamId, "empatam no RS e o segundo critério decide")
}

func TestADirectionOfAscPutsTheSmallestFirst(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	ascending := *season
	ascending.Ranking.Tiebreakers = []SeasonSort{{ID: "fouls", Label: "Faltas", Direction: "asc"}}

	rankings := Rankings{
		{TeamId: 1, RankingFields: RankingFields{RankingPoints: 6, Played: 3, Sort: []int{9}}},
		{TeamId: 2, RankingFields: RankingFields{RankingPoints: 6, Played: 3, Sort: []int{2}}},
	}
	SortRankings(rankings, &ascending)
	assert.Equal(t, 2, rankings[0].TeamId)
}

func TestAddScoreSummaryKeepsTheDrawnTiebreaker(t *testing.T) {
	summary := TestScore1().Summarize()
	rankingFields := RankingFields{Random: 0.4242}
	rankingFields.AddScoreSummary(nil, summary, summary, EntryNormal)
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
	// Check tiebreakers: ranking points first, then the vector the season declared, in order.
	entry := func(team, points, first, second, third int, random float64, played int) Ranking {
		return Ranking{TeamId: team, RankingFields: RankingFields{
			RankingPoints: points, Sort: []int{first, second, third}, Random: random,
			Wins: 3, Losses: 2, Ties: 1, Played: played, RsDecimals: 2}}
	}
	rankings := Rankings{
		entry(1, 50, 50, 50, 50, 0.49, 10),
		entry(2, 50, 50, 50, 50, 0.51, 10),
		entry(3, 50, 50, 50, 49, 0.50, 10),
		entry(4, 50, 50, 50, 51, 0.50, 10),
		entry(5, 50, 50, 49, 50, 0.50, 10),
		entry(6, 50, 50, 51, 50, 0.50, 10),
		entry(7, 50, 49, 50, 50, 0.50, 10),
		entry(8, 50, 51, 50, 50, 0.50, 10),
		entry(9, 49, 50, 50, 50, 0.50, 10),
		entry(10, 51, 50, 50, 50, 0.50, 10),
	}
	SortRankings(rankings, nil)
	assert.Equal(t, []int{10, 8, 6, 4, 2, 1, 3, 5, 7, 9}, []int{
		rankings[0].TeamId, rankings[1].TeamId, rankings[2].TeamId, rankings[3].TeamId,
		rankings[4].TeamId, rankings[5].TeamId, rankings[6].TeamId, rankings[7].TeamId,
		rankings[8].TeamId, rankings[9].TeamId})

	// Check with unequal number of matches played: it is the average that ranks, not the total.
	rankings = Rankings{
		entry(1, 10, 25, 25, 25, 0.49, 5),
		entry(2, 19, 50, 50, 50, 0.51, 9),
		entry(3, 20, 50, 50, 50, 0.51, 10),
	}
	SortRankings(rankings, nil)
	assert.Equal(t, []int{2, 3, 1}, []int{
		rankings[0].TeamId, rankings[1].TeamId, rankings[2].TeamId})
}

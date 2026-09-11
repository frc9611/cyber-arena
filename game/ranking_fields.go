// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Game-specific fields by which teams are ranked and the logic for sorting rankings.

package game

import "sort"

// How one appearance of a team in a match counts. The three are declared by the season, because the
// manuals disagree with each other and with what this fork used to do, which was to skip the
// surrogate entirely — a fourth behaviour nobody asked for.
type RankingEntry int

const (
	EntryNormal RankingEntry = iota
	EntrySurrogate
	EntryDisqualified
)

type RankingFields struct {
	RankingPoints     int
	Sort              []int
	Wins              int
	Losses            int
	Ties              int
	Played            int
	Disqualifications int
	Random            float64
	RsDecimals        int

	// Written by every build before the seasons, and still decoded so an old event opens.
	AutoPoints    int
	EndgamePoints int
	TeleopPoints  int
}

type Ranking struct {
	TeamId       int `db:"id,manual"`
	Rank         int
	PreviousRank int
	RankingFields
}

type Rankings []Ranking

// Adds one appearance to a team's record. What a win is worth, what the tiebreaker vector holds and
// what a surrogate or a disqualified team takes away from the match are all read from the season.
func (fields *RankingFields) AddScoreSummary(season *Season, ownScore, opponentScore *ScoreSummary,
	entry RankingEntry) {
	standing := standingFor(season, entry)
	if standing.CountsPlayed {
		fields.Played += 1
	}
	if entry == EntryDisqualified {
		fields.Disqualifications += 1
	}
	if season != nil {
		fields.RsDecimals = season.Ranking.RsDecimals
	}

	if entry == EntryNormal {
		fields.RankingPoints += resultPoints(season, ownScore, opponentScore)
		if ownScore.Outcome != nil {
			fields.RankingPoints += ownScore.Outcome.RankingPoints
		}
		if ownScore.Score > opponentScore.Score {
			fields.Wins += 1
		} else if ownScore.Score == opponentScore.Score {
			fields.Ties += 1
		} else {
			fields.Losses += 1
		}
	} else {
		fields.RankingPoints += standing.Rp
	}

	var contribution []int
	if !standing.SortZero {
		if ownScore.Outcome != nil {
			contribution = ownScore.Outcome.Sort
		} else {
			// A result from before the seasons has no vector of its own. Its three period totals
			// are what the tiebreakers meant back then, and the legacy season declares them in this
			// order, so an old event keeps ranking the way it always did.
			contribution = []int{ownScore.AutoPoints, ownScore.EndgamePoints, ownScore.TeleopPoints}
		}
	}
	fields.addSort(season, contribution)

	// The three period columns are what a build from before the seasons wrote, and the legacy
	// season keeps them meaning the same thing through its own tiebreakers.
	fields.AutoPoints += ownScore.AutoPoints
	fields.EndgamePoints += ownScore.EndgamePoints
	fields.TeleopPoints += ownScore.TeleopPoints
}

func (fields *RankingFields) addSort(season *Season, contribution []int) {
	size := len(contribution)
	if season != nil && len(season.Ranking.Tiebreakers) > size {
		size = len(season.Ranking.Tiebreakers)
	}
	for len(fields.Sort) < size {
		fields.Sort = append(fields.Sort, 0)
	}
	for index, value := range contribution {
		if index >= len(fields.Sort) {
			break
		}
		switch aggregateAt(season, index) {
		case "max":
			if value > fields.Sort[index] {
				fields.Sort[index] = value
			}
		case "min":
			if fields.Played <= 1 || value < fields.Sort[index] {
				fields.Sort[index] = value
			}
		default:
			fields.Sort[index] += value
		}
	}
}

func standingFor(season *Season, entry RankingEntry) SeasonStanding {
	if entry == EntryNormal {
		return SeasonStanding{CountsPlayed: true}
	}
	fallback := SeasonStanding{Rp: 0, SortZero: true, CountsPlayed: true}
	if season == nil {
		return fallback
	}
	if entry == EntrySurrogate && season.Ranking.Surrogate != nil {
		return *season.Ranking.Surrogate
	}
	if entry == EntryDisqualified && season.Ranking.Disqualified != nil {
		return *season.Ranking.Disqualified
	}
	return fallback
}

func resultPoints(season *Season, ownScore, opponentScore *ScoreSummary) int {
	result := SeasonResult{Win: 2, Tie: 1, Loss: 0}
	if season != nil && season.Ranking.Result != nil {
		result = *season.Ranking.Result
	}
	if ownScore.Score > opponentScore.Score {
		return result.Win
	}
	if ownScore.Score == opponentScore.Score {
		return result.Tie
	}
	return result.Loss
}

// The Ranking Score is the average, rounded, and the first criterion compares the rounded value:
// 19 in 11 and 26 in 15 are both 1.73 and officially tie, while cross-multiplying separates them.
func (fields *RankingFields) RankingScore() int {
	if fields.Played == 0 {
		return 0
	}
	scale := 1
	for i := 0; i < fields.RsDecimals; i++ {
		scale *= 10
	}
	return (2*scale*fields.RankingPoints + fields.Played) / (2 * fields.Played)
}

// Helper function to implement the required interface for Sort.
func (rankings Rankings) Len() int {
	return len(rankings)
}

// Helper function to implement the required interface for Sort.
func (rankings Rankings) Less(i, j int) bool {
	return rankingBefore(rankings[i], rankings[j], nil)
}

// Helper function to implement the required interface for Sort.
func (rankings Rankings) Swap(i, j int) {
	rankings[i], rankings[j] = rankings[j], rankings[i]
}

// SortRankings orders the table by the season's own criteria, in the order the season declared them
// and in the direction it declared: "fewest fouls committed" is a real criterion, and negating the
// expression to fake it would put a negative number on the screen and in the CSV.
func SortRankings(rankings Rankings, season *Season) {
	sort.SliceStable(rankings, func(i, j int) bool {
		return rankingBefore(rankings[i], rankings[j], season)
	})
}

func rankingBefore(a, b Ranking, season *Season) bool {
	// A team with no match ranks last: every criterion of a team that never played is zero, which
	// otherwise ties it with everybody and lands it wherever the random tiebreaker says.
	if a.Played == 0 || b.Played == 0 {
		if a.Played != b.Played {
			return b.Played == 0
		}
		return a.TeamId < b.TeamId
	}

	if scoreA, scoreB := a.RankingScore(), b.RankingScore(); scoreA != scoreB {
		return scoreA > scoreB
	}

	size := len(a.Sort)
	if len(b.Sort) > size {
		size = len(b.Sort)
	}
	for index := 0; index < size; index++ {
		valueA, valueB := sortAt(a, index), sortAt(b, index)
		// A criterion that adds up is compared as an average, like the ranking score and like every
		// sort order the manual prints. Cross-multiplying keeps it in integer maths. One that takes
		// the best or the worst single match is already one number and is compared as it is.
		if aggregateAt(season, index) == "sum" {
			valueA, valueB = valueA*b.Played, valueB*a.Played
		}
		if valueA == valueB {
			continue
		}
		if season != nil && index < len(season.Ranking.Tiebreakers) &&
			season.Ranking.Tiebreakers[index].Direction == "asc" {
			return valueA < valueB
		}
		return valueA > valueB
	}

	// Nothing left that the rules can separate, so the draw decides. It belongs to the team for the
	// whole event: drawing it again on every calculation made a tied pair trade places on reload.
	return a.Random > b.Random
}

func aggregateAt(season *Season, index int) string {
	if season == nil || index >= len(season.Ranking.Tiebreakers) {
		return "sum"
	}
	if declared := season.Ranking.Tiebreakers[index].Agg; declared != "" {
		return declared
	}
	return "sum"
}

func sortAt(ranking Ranking, index int) int {
	if index < len(ranking.Sort) {
		return ranking.Sort[index]
	}
	return 0
}

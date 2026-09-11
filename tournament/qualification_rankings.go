// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Functions for calculating the qualification rankings.

package tournament

import (
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"math/rand"
	"sort"
)

// Determines the rankings from the stored match results, and saves them to the database.
func CalculateRankings(database *model.Database, preservePreviousRank bool) (game.Rankings, error) {
	matches, err := database.GetMatchesByType("qualification")
	if err != nil {
		return nil, err
	}
	rankings := make(map[int]*game.Ranking)
	for _, match := range matches {
		if !match.IsComplete() {
			continue
		}
		matchResult, err := database.GetMatchResultForMatch(match.Id)
		if err != nil {
			return nil, err
		}
		if !match.Red1IsSurrogate {
			addMatchResultToRankings(rankings, match.Red1, matchResult, true)
		}
		if !match.Red2IsSurrogate {
			addMatchResultToRankings(rankings, match.Red2, matchResult, true)
		}
		if !match.Red3IsSurrogate {
			addMatchResultToRankings(rankings, match.Red3, matchResult, true)
		}
		if !match.Blue1IsSurrogate {
			addMatchResultToRankings(rankings, match.Blue1, matchResult, false)
		}
		if !match.Blue2IsSurrogate {
			addMatchResultToRankings(rankings, match.Blue2, matchResult, false)
		}
		if !match.Blue3IsSurrogate {
			addMatchResultToRankings(rankings, match.Blue3, matchResult, false)
		}
	}

	// Retrieve old rankings so that we can display changes in rank as a result of this calculation.
	oldRankings, err := database.GetAllRankings()
	if err != nil {
		return nil, err
	}
	oldRankingsMap := make(map[int]game.Ranking, len(oldRankings))
	for _, ranking := range oldRankings {
		oldRankingsMap[ranking.TeamId] = ranking
	}

	// The random tiebreaker is drawn once per team and kept, not redrawn on every calculation: a
	// value that changes every time makes a tied pair trade places on every page load. The draw
	// walks the teams in order, because map order is not an order.
	teamIds := make([]int, 0, len(rankings))
	for teamId := range rankings {
		teamIds = append(teamIds, teamId)
	}
	sort.Ints(teamIds)
	for _, teamId := range teamIds {
		if old, ok := oldRankingsMap[teamId]; ok && old.Random != 0 {
			rankings[teamId].Random = old.Random
		} else {
			rankings[teamId].Random = rand.Float64()
		}
	}

	sortedRankings := sortRankings(rankings)
	for rank, ranking := range sortedRankings {
		sortedRankings[rank].Rank = rank + 1
		if oldRank, ok := oldRankingsMap[ranking.TeamId]; ok {
			if preservePreviousRank {
				sortedRankings[rank].PreviousRank = oldRank.PreviousRank
			} else {
				sortedRankings[rank].PreviousRank = oldRank.Rank
			}
		}
	}
	err = database.ReplaceAllRankings(sortedRankings)
	if err != nil {
		return nil, err
	}

	return sortedRankings, nil
}

// Incrementally accounts for the given match result in the set of rankings that are being built.
func addMatchResultToRankings(
	rankings map[int]*game.Ranking, teamId int, matchResult *model.MatchResult, isRed bool,
) {
	if teamId <= 0 {
		return
	}
	ranking := rankings[teamId]
	if ranking == nil {
		ranking = &game.Ranking{TeamId: teamId}
		rankings[teamId] = ranking
	}

	if isRed {
		ranking.AddScoreSummary(matchResult.RedScoreSummary(), matchResult.BlueScoreSummary())
	} else {
		ranking.AddScoreSummary(matchResult.BlueScoreSummary(), matchResult.RedScoreSummary())
	}
}

// The map has no order, so the slice is put in team order before a stable sort. Without both, two
// teams that tie on every criterion swap places between one calculation and the next.
func sortRankings(rankings map[int]*game.Ranking) game.Rankings {
	var sortedRankings game.Rankings
	for _, ranking := range rankings {
		sortedRankings = append(sortedRankings, *ranking)
	}
	sort.Slice(sortedRankings, func(i, j int) bool {
		return sortedRankings[i].TeamId < sortedRankings[j].TeamId
	})
	sort.Stable(sortedRankings)
	return sortedRankings
}

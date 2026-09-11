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
//
// A surrogate is no longer skipped. Skipping it was a fourth behaviour that no manual describes:
// the season declares whether the appearance counts as played, what it pays and whether it feeds
// the tiebreakers, and the same three answers cover a disqualified alliance.
func CalculateRankings(database *model.Database, preservePreviousRank bool) (game.Rankings, error) {
	matches, err := database.GetMatchesByType("qualification")
	if err != nil {
		return nil, err
	}
	settings, err := database.GetEventSettings()
	if err != nil {
		return nil, err
	}
	season := game.SeasonByKey(settings.SeasonKey)
	rankings := make(map[int]*game.Ranking)
	for _, match := range matches {
		if !match.IsComplete() {
			continue
		}
		matchResult, err := database.GetMatchResultForMatch(match.Id)
		if err != nil {
			return nil, err
		}
		red := []struct {
			team      int
			surrogate bool
		}{
			{match.Red1, match.Red1IsSurrogate},
			{match.Red2, match.Red2IsSurrogate},
			{match.Red3, match.Red3IsSurrogate},
		}
		blue := []struct {
			team      int
			surrogate bool
		}{
			{match.Blue1, match.Blue1IsSurrogate},
			{match.Blue2, match.Blue2IsSurrogate},
			{match.Blue3, match.Blue3IsSurrogate},
		}
		for _, station := range red {
			addMatchResultToRankings(season, rankings, station.team, matchResult, true,
				entryOf(station.surrogate, match.RedDisqualified))
		}
		for _, station := range blue {
			addMatchResultToRankings(season, rankings, station.team, matchResult, false,
				entryOf(station.surrogate, match.BlueDisqualified))
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

	sortedRankings := sortRankings(rankings, season)
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

func entryOf(surrogate, disqualified bool) game.RankingEntry {
	if disqualified {
		return game.EntryDisqualified
	}
	if surrogate {
		return game.EntrySurrogate
	}
	return game.EntryNormal
}

// Incrementally accounts for the given match result in the set of rankings that are being built.
func addMatchResultToRankings(season *game.Season, rankings map[int]*game.Ranking, teamId int,
	matchResult *model.MatchResult, isRed bool, entry game.RankingEntry) {
	if teamId <= 0 {
		return
	}
	ranking := rankings[teamId]
	if ranking == nil {
		ranking = &game.Ranking{TeamId: teamId}
		rankings[teamId] = ranking
	}

	if isRed {
		ranking.AddScoreSummary(season, matchResult.RedScoreSummary(), matchResult.BlueScoreSummary(), entry)
	} else {
		ranking.AddScoreSummary(season, matchResult.BlueScoreSummary(), matchResult.RedScoreSummary(), entry)
	}
}

// The map has no order, so the slice is put in team order before a stable sort. Without both, two
// teams that tie on every criterion swap places between one calculation and the next.
func sortRankings(rankings map[int]*game.Ranking, season *game.Season) game.Rankings {
	var sortedRankings game.Rankings
	for _, ranking := range rankings {
		sortedRankings = append(sortedRankings, *ranking)
	}
	sort.Slice(sortedRankings, func(i, j int) bool {
		return sortedRankings[i].TeamId < sortedRankings[j].TeamId
	})
	game.SortRankings(sortedRankings, season)
	return sortedRankings
}

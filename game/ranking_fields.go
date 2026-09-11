// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Game-specific fields by which teams are ranked and the logic for sorting rankings.

package game

type RankingFields struct {
	RankingPoints int
	AutoPoints    int
	EndgamePoints int
	TeleopPoints  int
	Random        float64
	Wins          int
	Losses        int
	Ties          int
	Played        int
}

type Ranking struct {
	TeamId       int `db:"id,manual"`
	Rank         int
	PreviousRank int
	RankingFields
}

type Rankings []Ranking

func (fields *RankingFields) AddScoreSummary(ownScore *ScoreSummary, opponentScore *ScoreSummary) {
	fields.Played += 1

	// Assign ranking points and wins/losses/ties.
	if ownScore.Score > opponentScore.Score {
		fields.RankingPoints += 2
		fields.Wins += 1
	} else if ownScore.Score == opponentScore.Score {
		fields.RankingPoints += 1
		fields.Ties += 1
	} else {
		fields.Losses += 1
	}

	// Assign tiebreaker points.
	fields.AutoPoints += ownScore.AutoPoints
	fields.EndgamePoints += ownScore.EndgamePoints
	fields.TeleopPoints += ownScore.TeleopPoints
}

// Helper function to implement the required interface for Sort.
func (rankings Rankings) Len() int {
	return len(rankings)
}

// Helper function to implement the required interface for Sort.
func (rankings Rankings) Less(i, j int) bool {
	a := rankings[i]
	b := rankings[j]

	// A team with no match ranks last: cross-multiplication by zero makes it tie with everybody.
	if a.Played == 0 || b.Played == 0 {
		if a.Played != b.Played {
			return b.Played == 0
		}
		return a.TeamId < b.TeamId
	}

	// Use cross-multiplication to keep it in integer math.
	if a.RankingPoints*b.Played == b.RankingPoints*a.Played {
		if a.AutoPoints*b.Played == b.AutoPoints*a.Played {
			if a.EndgamePoints*b.Played == b.EndgamePoints*a.Played {
				if a.TeleopPoints*b.Played == b.TeleopPoints*a.Played {
					return a.Random > b.Random
				}
				return a.TeleopPoints*b.Played > b.TeleopPoints*a.Played
			}
			return a.EndgamePoints*b.Played > b.EndgamePoints*a.Played
		}
		return a.AutoPoints*b.Played > b.AutoPoints*a.Played
	}
	return a.RankingPoints*b.Played > b.RankingPoints*a.Played
}

// Helper function to implement the required interface for Sort.
func (rankings Rankings) Swap(i, j int) {
	rankings[i], rankings[j] = rankings[j], rankings[i]
}

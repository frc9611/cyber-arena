// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Helper methods for use in tests in this package and others.

package game

func TestScore1() *Score {
	return &Score{
		LegacyAutoPoints:    45,
		LegacyTeleopPoints:  80,
		LegacyEndgamePoints: 30,
	}
}

func TestScore2() *Score {
	return &Score{
		LegacyAutoPoints:    15,
		LegacyTeleopPoints:  40,
		LegacyEndgamePoints: 25,
	}
}

func TestRanking1() *Ranking {
	return &Ranking{254, 1, 0, RankingFields{
		RankingPoints: 20, Sort: []int{625, 90, 554}, Wins: 3, Losses: 2, Ties: 1, Played: 10,
		Random: 0.254, RsDecimals: 2, AutoPoints: 625, EndgamePoints: 90, TeleopPoints: 554}}
}

func TestRanking2() *Ranking {
	return &Ranking{1114, 2, 1, RankingFields{
		RankingPoints: 18, Sort: []int{700, 625, 90}, Wins: 1, Losses: 3, Ties: 2, Played: 10,
		Random: 0.1114, RsDecimals: 2, AutoPoints: 700, EndgamePoints: 625, TeleopPoints: 90}}
}

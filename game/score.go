// Copyright 2020 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model representing the instantaneous score of a match.

package game

import "reflect"

type Score struct {
	SeasonKey string                 `json:"SeasonKey,omitempty"`
	Level     string                 `json:"Level,omitempty"`
	Robots    int                    `json:"Robots,omitempty"`
	Actions   map[string]int         `json:"Actions,omitempty"`
	Ever      map[string]int         `json:"Ever,omitempty"`
	Adjust    map[string]int         `json:"Adjust,omitempty"`
	Draw      map[string]string      `json:"Draw,omitempty"`
	Slots     map[string][]SlotState `json:"Slots,omitempty"`

	LegacyAutoPoints    int `json:"AutoPoints"`
	LegacyTeleopPoints  int `json:"TeleopPoints"`
	LegacyEndgamePoints int `json:"EndgamePoints"`
	LegacyFoulPoints    int `json:"FoulPoints"`
}

// Calculates and returns the summary fields used for ranking and display. A score with a tally is
// read through its season; one without is a result from before the seasons existed, and is the sum
// of the four numbers a person typed.
func (score *Score) Summarize() *ScoreSummary {
	return score.SummarizeAgainst(nil)
}

func (score *Score) SummarizeAgainst(opponent *Score) *ScoreSummary {
	summary := new(ScoreSummary)
	if !score.HasTally() {
		summary.AutoPoints = score.LegacyAutoPoints
		summary.TeleopPoints = score.LegacyTeleopPoints
		summary.EndgamePoints = score.LegacyEndgamePoints
		summary.FoulPoints = score.LegacyFoulPoints
		summary.Score = summary.AutoPoints + summary.TeleopPoints + summary.EndgamePoints +
			summary.FoulPoints
		return summary
	}

	season := SeasonByKey(score.SeasonKey)
	if season == nil {
		season = SeasonByKey(LegacySeasonKey)
	}
	outcome := Evaluate(season, score.EventLevel(season), score, opponent)
	summary.Score = outcome.Total
	summary.Outcome = outcome
	summary.AutoPoints = outcome.PeriodPoints[firstScoringPeriod(season)]
	summary.FoulPoints = outcome.Categories[CategoryFoulTaken]
	summary.TeleopPoints = summary.Score - summary.AutoPoints - summary.FoulPoints
	return summary
}

func (score *Score) EventLevel(season *Season) string {
	if score.Level != "" {
		return score.Level
	}
	if season == nil {
		return ""
	}
	return season.DefaultLevel()
}

func firstScoringPeriod(season *Season) string {
	if season == nil {
		return ""
	}
	for _, period := range season.ScoringPeriods() {
		return period
	}
	return ""
}

// Returns true if and only if all fields of the two scores are equal.
func (score *Score) Equals(other *Score) bool {
	if score.LegacyAutoPoints != other.LegacyAutoPoints ||
		score.LegacyTeleopPoints != other.LegacyTeleopPoints ||
		score.LegacyEndgamePoints != other.LegacyEndgamePoints ||
		score.LegacyFoulPoints != other.LegacyFoulPoints {
		return false
	}
	if score.SeasonKey != other.SeasonKey || score.Level != other.Level || score.Robots != other.Robots {
		return false
	}
	return reflect.DeepEqual(score.Actions, other.Actions) &&
		reflect.DeepEqual(score.Ever, other.Ever) &&
		reflect.DeepEqual(score.Adjust, other.Adjust) &&
		reflect.DeepEqual(score.Draw, other.Draw) &&
		reflect.DeepEqual(score.Slots, other.Slots)
}

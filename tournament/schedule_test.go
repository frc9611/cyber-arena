// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package tournament

import (
	"testing"
	"time"

	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/stretchr/testify/assert"
)

func block(start time.Time, numMatches, spacingSec int) model.ScheduleBlock {
	return model.ScheduleBlock{MatchType: "test", StartTime: start, NumMatches: numMatches,
		MatchSpacingSec: spacingSec}
}

func teamsNumbered(count int) []model.Team {
	teams := make([]model.Team, count)
	for i := 0; i < count; i++ {
		teams[i].Id = i + 101
	}
	return teams
}

func teamIdsOf(match model.Match, teamsPerAlliance int) []int {
	ids := []int{match.Red1, match.Red2, match.Blue1, match.Blue2}
	if teamsPerAlliance == 3 {
		ids = append(ids, match.Red3, match.Blue3)
	}
	return ids
}

func TestScheduleFillsEveryMatchWithFourTeams(t *testing.T) {
	teams := teamsNumbered(18)
	blocks := []model.ScheduleBlock{block(time.Unix(0, 0).UTC(), 9, 60)}

	matches, err := BuildRandomSchedule(teams, blocks, "test", 2)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(matches))

	for _, match := range matches {
		assert.Equal(t, "test", match.Type)
		assert.Equal(t, 0, match.Red3)
		assert.Equal(t, 0, match.Blue3)
		seen := map[int]bool{}
		for _, id := range teamIdsOf(match, 2) {
			assert.GreaterOrEqual(t, id, 101)
			assert.False(t, seen[id])
			seen[id] = true
		}
	}
}

func TestScheduleFillsEveryMatchWithSixTeams(t *testing.T) {
	teams := teamsNumbered(18)
	blocks := []model.ScheduleBlock{block(time.Unix(0, 0).UTC(), 9, 60)}

	matches, err := BuildRandomSchedule(teams, blocks, "test", 3)
	assert.Nil(t, err)

	for _, match := range matches {
		seen := map[int]bool{}
		for _, id := range teamIdsOf(match, 3) {
			assert.NotEqual(t, 0, id)
			assert.False(t, seen[id])
			seen[id] = true
		}
	}
}

func TestScheduleGivesEveryTeamTheSameNumberOfMatches(t *testing.T) {
	teams := teamsNumbered(12)
	blocks := []model.ScheduleBlock{block(time.Unix(0, 0).UTC(), 9, 60)}

	matches, err := BuildRandomSchedule(teams, blocks, "test", 2)
	assert.Nil(t, err)

	played := map[int]int{}
	for _, match := range matches {
		for _, id := range teamIdsOf(match, 2) {
			played[id]++
		}
	}
	assert.Equal(t, 12, len(played))
	for _, count := range played {
		assert.Equal(t, played[101], count)
	}
}

func TestScheduleTiming(t *testing.T) {
	teams := teamsNumbered(18)
	blocks := []model.ScheduleBlock{
		block(time.Unix(100, 0).UTC(), 10, 75),
		block(time.Unix(20000, 0).UTC(), 5, 1000),
		block(time.Unix(100000, 0).UTC(), 15, 29),
	}

	matches, err := BuildRandomSchedule(teams, blocks, "test", 2)
	assert.Nil(t, err)
	assert.Equal(t, time.Unix(100, 0).UTC(), matches[0].Time)
	assert.Equal(t, time.Unix(775, 0).UTC(), matches[9].Time)
	assert.Equal(t, time.Unix(20000, 0).UTC(), matches[10].Time)
	assert.Equal(t, time.Unix(24000, 0).UTC(), matches[14].Time)
	assert.Equal(t, time.Unix(100000, 0).UTC(), matches[15].Time)
}

func TestScheduleWithNoTeams(t *testing.T) {
	blocks := []model.ScheduleBlock{block(time.Unix(0, 0).UTC(), 2, 60)}

	matches, err := BuildRandomSchedule(nil, blocks, "test", 2)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(matches))
}

// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank & Gabriel Paulo Toledo)
//
// Functions for creating practice and qualification match schedules.

package tournament

import (
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/model"
)

const (
	schedulesDir = "schedules"
)

// Creates a random schedule for the given parameters and returns it as a list of matches.
func BuildRandomSchedule(arena *field.Arena, teams []model.Team, scheduleBlocks []model.ScheduleBlock,
	matchType string) ([]model.Match, error) {
	// Load the anonymized, pre-randomized match schedule for the given number of teams and matches per team.
	numTeams := len(teams)
	numMatches := countMatches(scheduleBlocks)
	TeamsPerMatch := arena.EventSettings.TeamsPerAlliance * 2
	matchesPerTeam := int(float32(numMatches*TeamsPerMatch) / float32(numTeams))

	// Adjust the number of matches to remove any excess from non-perfect block scheduling.
	numMatches = int(math.Ceil(float64(numTeams) * float64(matchesPerTeam) / float64(TeamsPerMatch)))

	// Randomize the schedule
	rand.Seed(time.Now().UnixNano())
	matches := make([]model.Match, 0, numMatches)
	teamIndices := rand.Perm(numTeams)

	for i := 0; i < numMatches; i++ {
		matchTeams := make([]model.Team, 0, TeamsPerMatch)
		for j := 0; j < TeamsPerMatch; j++ {
			teamIndex := teamIndices[(i*TeamsPerMatch+j)%numTeams]
			matchTeams = append(matchTeams, teams[teamIndex])
		}
		if arena.EventSettings.TeamsPerAlliance == 2 {
			matches = append(matches, model.Match{Blue1: matchTeams[0].Id, Blue2: matchTeams[1].Id, Red1: matchTeams[2].Id, Red2: matchTeams[3].Id, DisplayName: strconv.Itoa(i + 1), Type: matchType})
		} else if arena.EventSettings.TeamsPerAlliance == 3 {
			matches = append(matches, model.Match{Blue1: matchTeams[0].Id, Blue2: matchTeams[1].Id, Blue3: matchTeams[2].Id, Red1: matchTeams[3].Id, Red2: matchTeams[4].Id, Red3: matchTeams[5].Id, DisplayName: strconv.Itoa(i + 1), Type: matchType})
		}
	}

	matchIndex := 0
	for _, block := range scheduleBlocks {
		for i := 0; i < block.NumMatches && matchIndex < numMatches; i++ {
			matches[matchIndex].Time = block.StartTime.Add(time.Duration(i*block.MatchSpacingSec) * time.Second)
			matchIndex++
		}
	}

	return matches, nil
}

// Returns the total number of matches that can be run within the given schedule blocks.
func countMatches(scheduleBlocks []model.ScheduleBlock) int {
	numMatches := 0
	for _, block := range scheduleBlocks {
		numMatches += block.NumMatches
	}
	return numMatches
}

// matches[matchIndex].Time = block.StartTime.Add(time.Duration(i*block.MatchSpacingSec) * time.Second)

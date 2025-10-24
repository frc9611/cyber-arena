// Copyright 2025 CyberArena
// Minimal remote-syncable FLL score model.

package model

import (
	"sort"
	"time"
)

type FllScore struct {
	TeamId    int `db:"id,manual"`
	Rounds    []int
	Best      int
	UpdatedAt time.Time
}

func (database *Database) CreateFllScore(score *FllScore) error {
	return database.fllScoreTable.create(score)
}

func (database *Database) GetFllScoreByTeamId(teamId int) (*FllScore, error) {
	return database.fllScoreTable.getById(teamId)
}

func (database *Database) UpdateFllScore(score *FllScore) error {
	return database.fllScoreTable.update(score)
}

func (database *Database) DeleteFllScore(teamId int) error {
	return database.fllScoreTable.delete(teamId)
}

func (database *Database) TruncateFllScores() error {
	return database.fllScoreTable.truncate()
}

func (database *Database) GetAllFllScores() ([]FllScore, error) {
	scores, err := database.fllScoreTable.getAll()
	if err != nil {
		return nil, err
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].TeamId < scores[j].TeamId })
	return scores, nil
}

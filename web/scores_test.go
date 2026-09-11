// Copyright 2020 Team 254. All Rights Reserved.
// Author: kenschenke@gmail.com (Ken Schenke)

package web

import (
	"encoding/json"
	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

func TestGetScores(t *testing.T) {
	web := setupTestWeb(t)

	score1 := game.TestScore1()
	score2 := game.TestScore2()
	web.arena.RedScore.LegacyAutoPoints = score1.LegacyAutoPoints
	web.arena.RedScore.LegacyTeleopPoints = score1.LegacyTeleopPoints
	web.arena.RedScore.LegacyEndgamePoints = score1.LegacyEndgamePoints
	web.arena.BlueScore.LegacyAutoPoints = score2.LegacyAutoPoints
	web.arena.BlueScore.LegacyTeleopPoints = score2.LegacyTeleopPoints
	web.arena.BlueScore.LegacyEndgamePoints = score2.LegacyEndgamePoints

	recorder := web.getHttpResponse("/api/scores")
	assert.Equal(t, 200, recorder.Code)

	var reqScores jsonScore
	json.Unmarshal(recorder.Body.Bytes(), &reqScores)
	assert.Equal(t, score1.LegacyAutoPoints, reqScores.Red.Auto)
	assert.Equal(t, score1.LegacyTeleopPoints, reqScores.Red.Teleop)
	assert.Equal(t, score1.LegacyEndgamePoints, reqScores.Red.Endgame)
	assert.Equal(t, score2.LegacyAutoPoints, reqScores.Blue.Auto)
	assert.Equal(t, score2.LegacyTeleopPoints, reqScores.Blue.Teleop)
	assert.Equal(t, score2.LegacyEndgamePoints, reqScores.Blue.Endgame)
}

func TestPatchScores(t *testing.T) {
	web := setupTestWeb(t)
	var recorder *httptest.ResponseRecorder

	web.arena.MatchState = field.PreMatch
	recorder = web.patchHttpResponse("/api/scores", "{}")
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "Score cannot be updated in this match state\n", recorder.Body.String())

	score1 := game.TestScore1()
	score2 := game.TestScore2()
	web.arena.RedScore.LegacyAutoPoints = score1.LegacyAutoPoints
	web.arena.RedScore.LegacyTeleopPoints = score1.LegacyTeleopPoints
	web.arena.RedScore.LegacyEndgamePoints = score1.LegacyEndgamePoints
	web.arena.BlueScore.LegacyAutoPoints = score2.LegacyAutoPoints
	web.arena.BlueScore.LegacyTeleopPoints = score2.LegacyTeleopPoints
	web.arena.BlueScore.LegacyEndgamePoints = score2.LegacyEndgamePoints

	web.arena.MatchState = field.PostMatch
	recorder = web.patchHttpResponse("/api/scores",
		"{\"red\":{\"auto\":5,\"teleop\":10,\"endgame\":15}}")
	assert.Equal(t, 200, recorder.Code)

	assert.Equal(t, score1.LegacyAutoPoints+5, web.arena.RedScore.LegacyAutoPoints)
	assert.Equal(t, score1.LegacyTeleopPoints+10, web.arena.RedScore.LegacyTeleopPoints)
	assert.Equal(t, score1.LegacyEndgamePoints+15, web.arena.RedScore.LegacyEndgamePoints)
	assert.Equal(t, score2.LegacyAutoPoints, web.arena.BlueScore.LegacyAutoPoints)
	assert.Equal(t, score2.LegacyTeleopPoints, web.arena.BlueScore.LegacyTeleopPoints)
	assert.Equal(t, score2.LegacyEndgamePoints, web.arena.BlueScore.LegacyEndgamePoints)

	recorder = web.patchHttpResponse("/api/scores",
		"{\"blue\":{\"auto\":-5,\"teleop\":-10,\"endgame\":-15}}")
	assert.Equal(t, 200, recorder.Code)

	assert.Equal(t, score1.LegacyAutoPoints+5, web.arena.RedScore.LegacyAutoPoints)
	assert.Equal(t, score1.LegacyTeleopPoints+10, web.arena.RedScore.LegacyTeleopPoints)
	assert.Equal(t, score1.LegacyEndgamePoints+15, web.arena.RedScore.LegacyEndgamePoints)
	assert.Equal(t, score2.LegacyAutoPoints-5, web.arena.BlueScore.LegacyAutoPoints)
	assert.Equal(t, score2.LegacyTeleopPoints-10, web.arena.BlueScore.LegacyTeleopPoints)
	assert.Equal(t, score2.LegacyEndgamePoints-15, web.arena.BlueScore.LegacyEndgamePoints)
}

func TestPutScores(t *testing.T) {
	web := setupTestWeb(t)
	var recorder *httptest.ResponseRecorder

	web.arena.MatchState = field.PreMatch
	recorder = web.putHttpResponse("/api/scores", "{}")
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "Score cannot be updated in this match state\n", recorder.Body.String())

	score1 := game.TestScore1()
	score2 := game.TestScore2()
	web.arena.RedScore.LegacyAutoPoints = score1.LegacyAutoPoints
	web.arena.RedScore.LegacyTeleopPoints = score1.LegacyTeleopPoints
	web.arena.RedScore.LegacyEndgamePoints = score1.LegacyEndgamePoints
	web.arena.BlueScore.LegacyAutoPoints = score2.LegacyAutoPoints
	web.arena.BlueScore.LegacyTeleopPoints = score2.LegacyTeleopPoints
	web.arena.BlueScore.LegacyEndgamePoints = score2.LegacyEndgamePoints

	web.arena.MatchState = field.PostMatch
	recorder = web.putHttpResponse("/api/scores",
		"{\"red\":{\"auto\":5,\"teleop\":10,\"endgame\":15}}")
	assert.Equal(t, 200, recorder.Code)

	assert.Equal(t, 5, web.arena.RedScore.LegacyAutoPoints)
	assert.Equal(t, 10, web.arena.RedScore.LegacyTeleopPoints)
	assert.Equal(t, 15, web.arena.RedScore.LegacyEndgamePoints)
	assert.Equal(t, 0, web.arena.BlueScore.LegacyAutoPoints)
	assert.Equal(t, 0, web.arena.BlueScore.LegacyTeleopPoints)
	assert.Equal(t, 0, web.arena.BlueScore.LegacyEndgamePoints)

	recorder = web.putHttpResponse("/api/scores",
		"{\"blue\":{\"auto\":5,\"teleop\":10,\"endgame\":15}}")
	assert.Equal(t, 200, recorder.Code)

	assert.Equal(t, 0, web.arena.RedScore.LegacyAutoPoints)
	assert.Equal(t, 0, web.arena.RedScore.LegacyTeleopPoints)
	assert.Equal(t, 0, web.arena.RedScore.LegacyEndgamePoints)
	assert.Equal(t, 5, web.arena.BlueScore.LegacyAutoPoints)
	assert.Equal(t, 10, web.arena.BlueScore.LegacyTeleopPoints)
	assert.Equal(t, 15, web.arena.BlueScore.LegacyEndgamePoints)
}

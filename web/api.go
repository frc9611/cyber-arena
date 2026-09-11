// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web API for providing JSON-formatted event data.

package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/partner"
	"github.com/Team254/cheesy-arena-lite/websocket"
	"github.com/gorilla/mux"
)

type MatchResultWithSummary struct {
	model.MatchResult
	RedSummary  *game.ScoreSummary
	BlueSummary *game.ScoreSummary
}

type MatchWithResult struct {
	model.Match
	Result *MatchResultWithSummary
}

type RankingWithNickname struct {
	game.Ranking
	Nickname string
}

type allianceMatchup struct {
	Round              int
	Group              int
	DisplayName        string
	RedAllianceSource  string
	BlueAllianceSource string
	RedAlliance        *model.Alliance
	BlueAlliance       *model.Alliance
	IsActive           bool
	SeriesLeader       string
	SeriesStatus       string
	IsComplete         bool
}

// Generates a JSON dump of the matches and results.
func (web *Web) matchesApiHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matches, err := web.arena.Database.GetMatchesByType(vars["type"])
	if err != nil {
		handleWebErr(w, err)
		return
	}

	matchesWithResults := make([]MatchWithResult, len(matches))
	for i, match := range matches {
		matchesWithResults[i].Match = match
		matchResult, err := web.arena.Database.GetMatchResultForMatch(match.Id)
		if err != nil {
			handleWebErr(w, err)
			return
		}
		var matchResultWithSummary *MatchResultWithSummary
		if matchResult != nil {
			matchResultWithSummary = &MatchResultWithSummary{MatchResult: *matchResult}
			matchResultWithSummary.RedSummary = matchResult.RedScoreSummary()
			matchResultWithSummary.BlueSummary = matchResult.BlueScoreSummary()
		}
		matchesWithResults[i].Result = matchResultWithSummary
	}

	jsonData, err := json.MarshalIndent(matchesWithResults, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Generates a JSON dump of the sponsor slides for use by the audience display.
func (web *Web) sponsorSlidesApiHandler(w http.ResponseWriter, r *http.Request) {
	sponsors, err := web.arena.Database.GetAllSponsorSlides()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	if sponsors == nil {
		// Go marshals an empty slice to null, so explicitly create it so that it appears as an empty JSON array.
		sponsors = make([]model.SponsorSlide, 0)
	}
	jsonData, err := json.MarshalIndent(sponsors, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Generates a JSON dump of the qualification rankings, primarily for use by the rankings display.
func (web *Web) rankingsApiHandler(w http.ResponseWriter, r *http.Request) {
	rankings, err := web.arena.Database.GetAllRankings()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	var rankingsWithNicknames []RankingWithNickname
	if rankings == nil {
		// Go marshals an empty slice to null, so explicitly create it so that it appears as an empty JSON array.
		rankingsWithNicknames = make([]RankingWithNickname, 0)
	} else {
		rankingsWithNicknames = make([]RankingWithNickname, len(rankings))
	}

	// Get team info so that nicknames can be displayed.
	teams, err := web.arena.Database.GetAllTeams()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	teamNicknames := make(map[int]string)
	for _, team := range teams {
		teamNicknames[team.Id] = team.Nickname
	}
	for i, ranking := range rankings {
		rankingsWithNicknames[i] = RankingWithNickname{ranking, teamNicknames[ranking.TeamId]}
	}

	// Get the last match scored so we can report that on the display.
	matches, err := web.arena.Database.GetMatchesByType("qualification")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	highestPlayedMatch := ""
	for _, match := range matches {
		if match.IsComplete() {
			highestPlayedMatch = match.DisplayName
		}
	}

	data := struct {
		Rankings           []RankingWithNickname
		HighestPlayedMatch string
	}{rankingsWithNicknames, highestPlayedMatch}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Generates a JSON dump of the alliances.
func (web *Web) alliancesApiHandler(w http.ResponseWriter, r *http.Request) {
	alliances, err := web.arena.Database.GetAllAlliances()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	jsonData, err := json.MarshalIndent(alliances, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Websocket API for receiving arena status updates.
func (web *Web) arenaWebsocketApiHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer ws.Close()

	// Subscribe the websocket to the notifiers whose messages will be passed on to the client.
	ws.HandleNotifiers(web.arena.MatchTimingNotifier, web.arena.MatchLoadNotifier, web.arena.MatchTimeNotifier)
}

// Serves the avatar for a given team, or a default if none exists.
func (web *Web) teamAvatarsApiHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamId, err := strconv.Atoi(vars["teamId"])
	if err != nil {
		handleWebErr(w, err)
		return
	}

	avatarPath := fmt.Sprintf("%s/%d.png", partner.AvatarsDir, teamId)
	if _, err := os.Stat(avatarPath); os.IsNotExist(err) {
		avatarPath = fmt.Sprintf("%s/0.png", partner.AvatarsDir)
	}

	http.ServeFile(w, r, avatarPath)
}

func (web *Web) estopHandler(w http.ResponseWriter, r *http.Request) {
	web.arena.ResetFieldEstop()
	web.arena.SetAudienceDisplayMode("logo")
	w.WriteHeader(http.StatusOK)
	fmt.Println("Field estop reset via web API")
}

func (web *Web) bracketSvgApiHandler(w http.ResponseWriter, r *http.Request) {
	var activeMatch *model.Match
	showTemporaryConnectors := false
	if activeMatchValue, ok := r.URL.Query()["activeMatch"]; ok {
		if activeMatchValue[0] == "current" {
			activeMatch = web.arena.CurrentMatch
		} else if activeMatchValue[0] == "saved" {
			activeMatch = web.arena.SavedMatch
			showTemporaryConnectors = true
		}
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	if err := web.generateBracketSvg(w, activeMatch, showTemporaryConnectors); err != nil {
		handleWebErr(w, err)
		return
	}
}

func (web *Web) generateBracketSvg(w io.Writer, activeMatch *model.Match, showTemporaryConnectors bool) error {
	alliances, err := web.arena.Database.GetAllAlliances()
	if err != nil {
		return err
	}

	matchups := make(map[string]*allianceMatchup)
	if web.arena.PlayoffBracket != nil {
		for _, matchup := range web.arena.PlayoffBracket.GetAllMatchups() {
			allianceMatchup := allianceMatchup{
				Round:              matchup.Round,
				Group:              matchup.Group,
				DisplayName:        matchup.LongDisplayName(),
				RedAllianceSource:  matchup.RedAllianceSourceDisplayName(),
				BlueAllianceSource: matchup.BlueAllianceSourceDisplayName(),
				IsComplete:         matchup.IsComplete(),
			}
			if matchup.RedAllianceId > 0 {
				if len(alliances) > 0 {
					allianceMatchup.RedAlliance = &alliances[matchup.RedAllianceId-1]
				} else {
					allianceMatchup.RedAlliance = &model.Alliance{Id: matchup.RedAllianceId}
				}
			}
			if matchup.BlueAllianceId > 0 {
				if len(alliances) > 0 {
					allianceMatchup.BlueAlliance = &alliances[matchup.BlueAllianceId-1]
				} else {
					allianceMatchup.BlueAlliance = &model.Alliance{Id: matchup.BlueAllianceId}
				}
			}
			if activeMatch != nil {
				allianceMatchup.IsActive = activeMatch.ElimRound == matchup.Round &&
					activeMatch.ElimGroup == matchup.Group
			}
			allianceMatchup.SeriesLeader, allianceMatchup.SeriesStatus = matchup.StatusText()
			matchups[fmt.Sprintf("%d_%d", matchup.Round, matchup.Group)] = &allianceMatchup
		}
	}

	bracketType := "double"
	numAlliances := web.arena.EventSettings.NumElimAlliances
	if web.arena.EventSettings.ElimType == "single" {
		if numAlliances > 8 {
			bracketType = "16"
		} else if numAlliances > 4 {
			bracketType = "8"
		} else if numAlliances > 2 {
			bracketType = "4"
		} else {
			bracketType = "2"
		}
	}

	template, err := web.parseFiles("templates/bracket.svg")
	if err != nil {
		return err
	}
	data := struct {
		BracketType             string
		Matchups                map[string]*allianceMatchup
		ShowTemporaryConnectors bool
	}{bracketType, matchups, showTemporaryConnectors}
	return template.ExecuteTemplate(w, "bracket", data)
}

// Minimal structs for remote sync
type fllSyncUpsert struct {
	TeamId int   `json:"teamId"`
	Rounds []int `json:"rounds"`
	// Optional official round selection (1..3), 0 to clear/use Best. Absent leaves it alone.
	OfficialRound *int `json:"officialRound,omitempty"`
}

// GET /api/fll/scores — returns all per-team scores (Rounds, Best)
func (web *Web) fllScoresApiGetHandler(w http.ResponseWriter, r *http.Request) {
	// Only active in FLL mode
	if !web.arena.EventSettings.IsFll {
		http.NotFound(w, r)
		return
	}
	// If configured, pull latest from remote hub into local before serving.
	remoteUrl := strings.TrimRight(web.arena.EventSettings.RemoteSyncUrl, "/")
	if remoteUrl != "" && r.Header.Get("X-From-Remote") != "1" {
		req, _ := http.NewRequest("GET", remoteUrl+"/scores", nil)
		req.Header.Set("X-From-Remote", "1")
		// Optional key for reads; not required by our handler.
		client := &http.Client{Timeout: fllMeshTimeout}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error pulling scores from %s: %v", remoteUrl, err)
		}
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var remoteScores []model.FllScore
			if err := json.NewDecoder(resp.Body).Decode(&remoteScores); err == nil {
				// Merge into local Bolt (upsert)
				for _, s := range remoteScores {
					existing, _ := web.arena.Database.GetFllScoreByTeamId(s.TeamId)
					if existing == nil {
						_ = web.arena.Database.CreateFllScore(&model.FllScore{TeamId: s.TeamId, Rounds: s.Rounds, Best: s.Best, UpdatedAt: s.UpdatedAt, OfficialRound: s.OfficialRound})
					} else {
						existing.Rounds = s.Rounds
						existing.Best = s.Best
						existing.UpdatedAt = s.UpdatedAt
						existing.OfficialRound = s.OfficialRound
						_ = web.arena.Database.UpdateFllScore(existing)
					}
				}
			}
		}
	}

	scores, err := web.arena.Database.GetAllFllScores()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	// Build Nickname and Name maps
	teams, err := web.arena.Database.GetAllTeams()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	teamNick := make(map[int]string, len(teams))
	teamName := make(map[int]string, len(teams))
	for _, t := range teams {
		teamNick[t.Id] = t.Nickname
		teamName[t.Id] = t.Name
	}
	// Create response with Nickname and Name
	type fllScoreWithName struct {
		TeamId        int       `json:"TeamId"`
		Rounds        []int     `json:"Rounds"`
		Best          int       `json:"Best"`
		UpdatedAt     time.Time `json:"UpdatedAt"`
		Nickname      string    `json:"Nickname"`
		Name          string    `json:"Name"`
		OfficialRound int       `json:"OfficialRound,omitempty"`
	}
	resp := make([]fllScoreWithName, 0, len(scores))
	for _, s := range scores {
		resp = append(resp, fllScoreWithName{
			TeamId:        s.TeamId,
			Rounds:        s.Rounds,
			Best:          s.Best,
			UpdatedAt:     s.UpdatedAt,
			Nickname:      teamNick[s.TeamId],
			Name:          teamName[s.TeamId],
			OfficialRound: s.OfficialRound,
		})
	}
	jsonData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// POST /api/fll/scores — upsert a single team’s rounds; auth via EventSettings.RemoteSyncApiKey
func (web *Web) fllScoresApiPostHandler(w http.ResponseWriter, r *http.Request) {
	// Only active in FLL mode
	if !web.arena.EventSettings.IsFll {
		http.NotFound(w, r)
		return
	}
	if web.arena.EventSettings.RemoteSyncApiKey != "" {
		if r.Header.Get("X-API-Key") != web.arena.EventSettings.RemoteSyncApiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	var body fllSyncUpsert
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if body.TeamId <= 0 {
		http.Error(w, "teamId required", http.StatusBadRequest)
		return
	}
	// Normalize to 3 rounds for now (pad / trim), or preserve if omitted.
	preserveRounds := len(body.Rounds) == 0
	rounds := make([]int, 0, 3)
	for i := 0; i < len(body.Rounds) && i < 3; i++ {
		rounds = append(rounds, body.Rounds[i])
	}
	for len(rounds) < 3 {
		rounds = append(rounds, 0)
	}
	best := 0
	for _, v := range rounds {
		if v > best {
			best = v
		}
	}

	existing, err := web.arena.Database.GetFllScoreByTeamId(body.TeamId)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if preserveRounds && existing != nil {
		// Keep prior rounds / best if not provided
		rounds = existing.Rounds
		best = existing.Best
	}
	if existing == nil {
		s := &model.FllScore{TeamId: body.TeamId, Rounds: rounds, Best: best, UpdatedAt: time.Now().UTC()}
		// Apply OfficialRound if provided
		if body.OfficialRound != nil && *body.OfficialRound >= 0 && *body.OfficialRound <= 3 {
			s.OfficialRound = *body.OfficialRound
		}
		if err := web.arena.Database.CreateFllScore(s); err != nil {
			handleWebErr(w, err)
			return
		}
	} else {
		existing.Rounds = rounds
		existing.Best = best
		existing.UpdatedAt = time.Now().UTC()
		if body.OfficialRound != nil && *body.OfficialRound >= 0 && *body.OfficialRound <= 3 {
			existing.OfficialRound = *body.OfficialRound
		}
		if err := web.arena.Database.UpdateFllScore(existing); err != nil {
			handleWebErr(w, err)
			return
		}
	}

	// Forward to remote hub if configured and not already forwarded.
	if r.Header.Get("X-From-Remote") != "1" {
		payload, _ := json.Marshal(body)
		web.postToFllMaster("/scores", payload)
	}

	w.WriteHeader(http.StatusNoContent)
}

// No-op stub to avoid duplicate definitions; authoritative implementation is in match_play.go.
func (web *Web) syncFllFromMatchResultApi(_ *model.Match, _ *model.MatchResult) error { return nil }

// POST /api/fll/start-match — remote command to start match timer; auth via EventSettings.RemoteSyncApiKey
func (web *Web) fllStartMatchApiHandler(w http.ResponseWriter, r *http.Request) {
	// Only active in FLL mode
	if !web.arena.EventSettings.IsFll {
		http.NotFound(w, r)
		return
	}
	// Authenticate via API key
	if web.arena.EventSettings.RemoteSyncApiKey != "" {
		if r.Header.Get("X-API-Key") != web.arena.EventSettings.RemoteSyncApiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	// Check if we're coming from remote to prevent loops
	if r.Header.Get("X-From-Remote") == "1" {
		// Start the match locally
		err := web.arena.StartMatch()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	} else {
		http.Error(w, "Must be called from remote master", http.StatusBadRequest)
	}
}

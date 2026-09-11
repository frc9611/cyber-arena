// Handlers for FLL Official Round Review page and API.
package web

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Team254/cheesy-arena-lite/model"
)

// Shows the FLL review page.
func (web *Web) fllReviewPageHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	template, err := web.parseFiles("templates/fll_review.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
	}{web.arena.EventSettings}
	if err := template.ExecuteTemplate(w, "base", data); err != nil {
		handleWebErr(w, err)
		return
	}
}

// Payload from the review UI.
type fllReviewPayload struct {
	TeamId int   `json:"teamId"`
	Rounds []int `json:"rounds"`
}

// Saves edited rounds; Best is recomputed.
func (web *Web) fllReviewApiPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	if !web.arena.EventSettings.IsFll {
		http.NotFound(w, r)
		return
	}
	var body fllReviewPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if body.TeamId <= 0 {
		http.Error(w, "teamId required", http.StatusBadRequest)
		return
	}
	// Normalize rounds to 3 values.
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
	if existing == nil {
		fs := &model.FllScore{TeamId: body.TeamId, Rounds: rounds, Best: best, UpdatedAt: time.Now().UTC()}
		if err := web.arena.Database.CreateFllScore(fs); err != nil {
			handleWebErr(w, err)
			return
		}
	} else {
		existing.Rounds = rounds
		existing.Best = best
		existing.UpdatedAt = time.Now().UTC()
		if err := web.arena.Database.UpdateFllScore(existing); err != nil {
			handleWebErr(w, err)
			return
		}
	}

	// Forward to remote hub if configured and not looped.
	if r.Header.Get("X-From-Remote") != "1" {
		payload, _ := json.Marshal(body)
		web.postToFllMaster("/review", payload)
	}
	w.WriteHeader(http.StatusNoContent)
}

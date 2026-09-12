// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web handlers for the referee interface.

package web

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/websocket"
	"github.com/mitchellh/mapstructure"
)

// Renders the referee interface for assigning fouls.
func (web *Web) refereePanelHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsRefereeOrHigher(w, r) {
		return
	}

	template, err := web.parseFiles("templates/referee_panel.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	data := struct {
		*model.EventSettings
		Match *model.Match
	}{web.arena.EventSettings, web.arena.CurrentMatch}
	err = template.ExecuteTemplate(w, "base_no_navbar", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// The websocket endpoint for the refereee interface client to send control commands and receive status updates.
func (web *Web) refereePanelWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer ws.Close()

	// Subscribe the websocket to the notifiers whose messages will be passed on to the client, in a separate goroutine.
	go ws.HandleNotifiers(
		web.arena.MatchLoadNotifier,
		web.arena.MatchTimingNotifier,
		web.arena.MatchTimeNotifier,
		web.arena.RealtimeScoreNotifier,
		web.arena.ReloadDisplaysNotifier,
	)

	// Loop, waiting for commands and responding to them, until the client closes the connection.
	for {
		messageType, data, err := ws.Read()
		if err != nil {
			if err == io.EOF {
				// Client has closed the connection; nothing to do here.
				return
			}
			log.Println(err)
			return
		}

		switch messageType {
		case "updateRealtimeScore":
			// The FLL panel still writes its total this way; the sheet layout replaces it later.
			args := data.(map[string]interface{})
			web.arena.BlueScore.LegacyAutoPoints = int(args["blueAuto"].(float64))
			web.arena.RedScore.LegacyAutoPoints = int(args["redAuto"].(float64))
			web.arena.BlueScore.LegacyTeleopPoints = int(args["blueTeleop"].(float64))
			web.arena.RedScore.LegacyTeleopPoints = int(args["redTeleop"].(float64))
			web.arena.BlueScore.LegacyEndgamePoints = int(args["blueEndgame"].(float64))
			web.arena.RedScore.LegacyEndgamePoints = int(args["redEndgame"].(float64))
			web.arena.RealtimeScoreNotifier.Notify()

		case "signalReset":
			if web.arena.MatchState != field.PostMatch {
				// Don't allow clearing the field until the match is over.
				continue
			}
			web.arena.FieldReset = true
			web.arena.AllianceStationDisplayMode = "fieldReset"
			web.arena.AllianceStationDisplayModeNotifier.Notify()
		case "commitMatch":
			if web.arena.MatchState != field.PostMatch {
				// Don't allow committing the fouls until the match is over.
				continue
			}
			//web.arena.RedRealtimeScore.FoulsCommitted = true
			//web.arena.BlueRealtimeScore.FoulsCommitted = true
			web.arena.FieldReset = true
			web.arena.AllianceStationDisplayMode = "fieldReset"
			web.arena.AllianceStationDisplayModeNotifier.Notify()
			//web.arena.ScoringStatusNotifier.Notify()

		case "scoreTally":
			args := struct {
				Alliance string
				Action   string
				Group    string
				Slot     int
				Option   string
				Period   string
				Delta    int
				Absolute bool
			}{}
			if err := mapstructure.Decode(data, &args); err != nil {
				ws.WriteError(err.Error())
				continue
			}
			if err := web.applyTally(args.Alliance, args.Action, args.Group, args.Slot, args.Option,
				args.Period, args.Delta, args.Absolute); err != nil {
				ws.WriteError(err.Error())
				continue
			}
			web.arena.RealtimeScoreNotifier.Notify()

		default:
			ws.WriteError(fmt.Sprintf("Invalid message type '%s'.", messageType))
		}
	}
}

// One click, one tally. Every path that used to write points went through a different guard — the
// red one refused before the match and the blue one did not — so the guard lives here once, and
// nothing else in the panel knows what anything is worth.
func (web *Web) applyTally(alliance, actionId, groupId string, slot int, optionId, period string,
	delta int, absolute bool) error {
	if web.arena.MatchState == field.PreMatch {
		return fmt.Errorf("A partida ainda não começou.")
	}
	score := web.arena.RedScore
	if alliance == "blue" {
		score = web.arena.BlueScore
	} else if alliance != "red" {
		return fmt.Errorf("Aliança inválida: %q.", alliance)
	}
	season := game.SeasonByKey(web.arena.EventSettings.SeasonKey)
	if season == nil {
		return fmt.Errorf("Este evento não tem temporada configurada.")
	}
	if period == "" {
		period = web.arena.CurrentPeriod()
	}

	if groupId != "" {
		group := season.Group(groupId)
		if group == nil {
			return fmt.Errorf("Grupo desconhecido: %q.", groupId)
		}
		limit := group.SlotCount(score.Robots)
		if limit > 0 && (slot < 1 || slot > limit) {
			return fmt.Errorf("O lugar %d não existe em %s.", slot, group.Label)
		}
		if optionId != "" && group.Option(optionId) == nil {
			return fmt.Errorf("Opção desconhecida: %q.", optionId)
		}
		score.Occupy(group, slot, optionId, period)
		return nil
	}

	action := season.Action(actionId)
	if action == nil {
		return fmt.Errorf("Ação desconhecida: %q.", actionId)
	}
	if action.Unit == game.UnitFreeValue {
		score.SetAdjustment(action.ID, delta)
		return nil
	}
	if len(action.Periods) == 0 {
		period = ""
	} else if !actionAllows(action, period) {
		return fmt.Errorf("%s não acontece em %s.", action.Label, period)
	}
	if absolute {
		score.SetAction(action.ID, period, delta)
		return nil
	}
	if action.Unit == game.UnitToggle {
		if score.Count(action.ID, period) > 0 {
			score.SetAction(action.ID, period, 0)
		} else {
			score.SetAction(action.ID, period, 1)
		}
		return nil
	}
	next := score.Count(action.ID, period) + delta
	if next < 0 {
		next = 0
	}
	if action.Max > 0 && next > action.Max {
		next = action.Max
	}
	score.SetAction(action.ID, period, next)
	return nil
}

func actionAllows(action *game.SeasonAction, period string) bool {
	for _, candidate := range action.Periods {
		if candidate == period {
			return true
		}
	}
	return false
}

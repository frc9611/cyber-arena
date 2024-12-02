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
		web.arena.MatchTimeNotifier,
		web.arena.RealtimeScoreNotifier,
		web.arena.MatchTimingNotifier,
		web.arena.ReloadDisplaysNotifier,
		web.arena.PointsContextNotifier,
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
		case "addFoul":
			args := struct {
				Alliance string
				Points   int
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}

			// Add the foul to the correct alliance's list.
			if args.Alliance == "red" {
				web.arena.BlueScore.FoulPoints += args.Points
			} else {
				web.arena.RedScore.FoulPoints += args.Points

			}
			web.arena.RealtimeScoreNotifier.Notify()
		case "card":
			args := struct {
				Alliance string
				TeamId   int
				Card     string
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}

			// Set the card in the correct alliance's score.
			//var cards map[string]string
			if args.Alliance == "red" {
				//cards = web.arena.RedRealtimeScore.Cards
			} else {
				//cards = web.arena.BlueRealtimeScore.Cards
			}
			//if web.arena.CurrentMatch.Type == model.Playoff {
			// Cards apply to the whole alliance in playoffs.
			if args.Alliance == "red" {
				//cards[strconv.Itoa(web.arena.CurrentMatch.Red1)] = args.Card
				//cards[strconv.Itoa(web.arena.CurrentMatch.Red2)] = args.Card
				//cards[strconv.Itoa(web.arena.CurrentMatch.Red3)] = args.Card
			} else {
				//cards[strconv.Itoa(web.arena.CurrentMatch.Blue1)] = args.Card
				//cards[strconv.Itoa(web.arena.CurrentMatch.Blue2)] = args.Card
				//cards[strconv.Itoa(web.arena.CurrentMatch.Blue3)] = args.Card
			}
			//} else {
			//cards[strconv.Itoa(args.TeamId)] = args.Card
			//}
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

		case "CR_SCORE_CONTEXT":
			// parse the json
			web.arena.CurrentMatch.ScoreContext = data.(string)
			web.arena.PointsContextNotifier.Notify()

		case "updateRealtimeScore":
			args := data.(map[string]interface{})
			web.arena.BlueScore.AutoPoints = int(args["blueAuto"].(float64))
			web.arena.RedScore.AutoPoints = int(args["redAuto"].(float64))
			web.arena.BlueScore.TeleopPoints = int(args["blueTeleop"].(float64))
			web.arena.RedScore.TeleopPoints = int(args["redTeleop"].(float64))
			web.arena.BlueScore.EndgamePoints = int(args["blueEndgame"].(float64))
			web.arena.RedScore.EndgamePoints = int(args["redEndgame"].(float64))
			web.arena.RealtimeScoreNotifier.Notify()
			continue

		case "addBluePoints":
			// 1 = AUTO ; 2 = TELEOP ; 3 = END_GAME

			args := data.(map[string]interface{})
			modeId := int(args["modeId"].(float64))
			points := int(args["points"].(float64))

			switch modeId {
			case 1:
				web.arena.BlueScore.AutoPoints += points
			case 2:
				web.arena.BlueScore.TeleopPoints += points
			case 3:
				web.arena.BlueScore.EndgamePoints += points
			default:
				ws.WriteError(fmt.Sprintf("Tipo de modo de jogo invalido '%s'.", modeId))
			}

			web.arena.RealtimeScoreNotifier.Notify()

		case "addRedPoints":
			// 1 = AUTO ; 2 = TELEOP ; 3 = END_GAME

			args := data.(map[string]interface{})
			modeId := int(args["modeId"].(float64))
			points := int(args["points"].(float64))

			if web.arena.MatchState == 0 {
				break
			}

			switch modeId {
			case 1:
				web.arena.RedScore.AutoPoints += points
			case 2:
				web.arena.RedScore.TeleopPoints += points
			case 3:
				web.arena.RedScore.EndgamePoints += points
			default:
				ws.WriteError(fmt.Sprintf("Tipo de modo de jogo invalido '%s'.", modeId))
			}

			web.arena.RealtimeScoreNotifier.Notify()
		default:
			ws.WriteError(fmt.Sprintf("Invalid message type '%s'.", messageType))
		}
	}
}

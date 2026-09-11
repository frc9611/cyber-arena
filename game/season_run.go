package game

import (
	"encoding/json"
	"fmt"
)

type TestFailure struct {
	Case   string
	Reason string
}

func (failure TestFailure) Error() string {
	return fmt.Sprintf("case %q: %s", failure.Case, failure.Reason)
}

func (season *Season) runTest(test SeasonTest) (*MatchOutcome, error) {
	own, err := season.buildScore(test.Level, test.Robots, test.Draw, test.State, test.Steps)
	if err != nil {
		return nil, TestFailure{Case: test.Name, Reason: err.Error()}
	}
	var opponent *Score
	if test.Opponent != nil {
		opponent, err = season.buildScore(test.Level, test.Robots, test.Draw, test.Opponent, nil)
		if err != nil {
			return nil, TestFailure{Case: test.Name, Reason: "opponent: " + err.Error()}
		}
	}
	return Evaluate(season, test.Level, own, opponent), nil
}

// RunTests plays every case of the season and answers what did not match. It runs in go test over
// every embedded season, in the editor before publishing, and again when a document is loaded.
func (season *Season) RunTests() []TestFailure {
	failures := []TestFailure{}
	for _, test := range season.Tests {
		outcome, err := season.runTest(test)
		if err != nil {
			failure, ok := err.(TestFailure)
			if !ok {
				failure = TestFailure{Case: test.Name, Reason: err.Error()}
			}
			failures = append(failures, failure)
			continue
		}
		for _, reason := range compareExpect(season, test.Expect, outcome) {
			failures = append(failures, TestFailure{Case: test.Name, Reason: reason})
		}
	}
	return failures
}

func (season *Season) buildScore(level string, robots int, draw map[string]string,
	state *TestState, steps []TestStep) (*Score, error) {
	score := &Score{SeasonKey: season.Key, Level: level, Robots: robots}
	if robots == 0 {
		score.Robots = season.RobotsPerAlliance
	}
	for id, value := range draw {
		score.SetDraw(id, value)
	}

	if state != nil {
		for key, count := range state.Actions {
			id, period := SplitTallyKey(key)
			action := season.Action(id)
			if action == nil {
				return nil, fmt.Errorf("unknown action %q", id)
			}
			if period == "" && len(action.Periods) == 1 {
				period = action.Periods[0]
			}
			score.SetAction(id, period, count)
		}
		for id, value := range state.Adjust {
			if season.Action(id) == nil {
				return nil, fmt.Errorf("unknown adjustment %q", id)
			}
			score.SetAdjustment(id, value)
		}
		for groupId, slots := range state.Slots {
			group := season.Group(groupId)
			if group == nil {
				return nil, fmt.Errorf("unknown slot group %q", groupId)
			}
			for slotText, raw := range slots {
				slot := atoiOr(slotText, -1)
				if slot < 1 {
					return nil, fmt.Errorf("slot %q of %q is not a number", slotText, groupId)
				}
				optionId, period, err := readSlotLiteral(raw)
				if err != nil {
					return nil, fmt.Errorf("slot %d of %q: %v", slot, groupId, err)
				}
				score.Occupy(group, slot, optionId, period)
			}
		}
	}

	for index, step := range steps {
		if step.Group != "" {
			group := season.Group(step.Group)
			if group == nil {
				return nil, fmt.Errorf("step %d: unknown slot group %q", index+1, step.Group)
			}
			option := ""
			if step.Option != nil {
				option = *step.Option
			}
			score.Occupy(group, step.Slot, option, step.Period)
			continue
		}
		if step.Action == "" {
			return nil, fmt.Errorf("step %d names neither a group nor an action", index+1)
		}
		action := season.Action(step.Action)
		if action == nil {
			return nil, fmt.Errorf("step %d: unknown action %q", index+1, step.Action)
		}
		period := step.Period
		if period == "" && len(action.Periods) == 1 {
			period = action.Periods[0]
		}
		delta := step.Delta
		if delta == 0 {
			delta = 1
		}
		if action.Unit == UnitFreeValue {
			score.SetAdjustment(action.ID, score.Adjustment(action.ID)+delta)
			continue
		}
		score.AddAction(action.ID, period, delta)
	}
	return score, nil
}

func readSlotLiteral(raw json.RawMessage) (string, string, error) {
	var option string
	if json.Unmarshal(raw, &option) == nil {
		return option, "", nil
	}
	var detailed struct {
		Option string `json:"option"`
		In     string `json:"in"`
	}
	if json.Unmarshal(raw, &detailed) == nil {
		return detailed.Option, detailed.In, nil
	}
	return "", "", fmt.Errorf("neither an option id nor {option, in}")
}

func compareExpect(season *Season, expect TestExpect, outcome *MatchOutcome) []string {
	reasons := []string{}
	if expect.Total != nil && *expect.Total != outcome.Total {
		reasons = append(reasons, fmt.Sprintf("total %d, expected %d", outcome.Total, *expect.Total))
	}
	for id, value := range expect.Categories {
		if outcome.Categories[id] != value {
			reasons = append(reasons, fmt.Sprintf("category %s %d, expected %d",
				id, outcome.Categories[id], value))
		}
	}
	if expect.Flags != nil {
		if diff := sameSet(outcome.Flags, expect.Flags); diff != "" {
			reasons = append(reasons, "flags "+diff)
		}
	}
	for id, value := range expect.FlagPts {
		if outcome.FlagPoints[id] != value {
			reasons = append(reasons, fmt.Sprintf("flag points %s %d, expected %d",
				id, outcome.FlagPoints[id], value))
		}
	}
	if expect.RpEarned != nil {
		if diff := sameSet(outcome.RpEarned, expect.RpEarned); diff != "" {
			reasons = append(reasons, "rpEarned "+diff)
		}
	}
	if expect.RpEligible != nil {
		if diff := sameSet(outcome.RpEligible, expect.RpEligible); diff != "" {
			reasons = append(reasons, "rpEligible "+diff)
		}
	}
	if expect.Sort != nil {
		if len(expect.Sort) != len(season.Ranking.Tiebreakers) {
			reasons = append(reasons, fmt.Sprintf("the case expects %d tiebreakers and the season has %d",
				len(expect.Sort), len(season.Ranking.Tiebreakers)))
		} else if !sameInts(outcome.Sort, expect.Sort) {
			reasons = append(reasons, fmt.Sprintf("sort %v, expected %v", outcome.Sort, expect.Sort))
		}
	}
	return reasons
}

func sameSet(got, wanted []string) string {
	gotSet := map[string]bool{}
	for _, id := range got {
		gotSet[id] = true
	}
	wantedSet := map[string]bool{}
	for _, id := range wanted {
		wantedSet[id] = true
	}
	missing := []string{}
	extra := []string{}
	for id := range wantedSet {
		if !gotSet[id] {
			missing = append(missing, id)
		}
	}
	for id := range gotSet {
		if !wantedSet[id] {
			extra = append(extra, id)
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return ""
	}
	return fmt.Sprintf("%v: missing %v, unexpected %v", got, missing, extra)
}

func sameInts(got, wanted []int) bool {
	if len(got) != len(wanted) {
		return false
	}
	for i := range got {
		if got[i] != wanted[i] {
			return false
		}
	}
	return true
}

package game

import "strings"

type SlotState struct {
	Occupant   string            `json:"occupant,omitempty"`
	OccupiedIn string            `json:"occupiedIn,omitempty"`
	EverIn     []string          `json:"everIn,omitempty"`
	PerPeriod  map[string]string `json:"perPeriod,omitempty"`
}

func TallyKey(id, period string) string {
	if period == "" {
		return id
	}
	return id + "@" + period
}

func SplitTallyKey(key string) (string, string) {
	if at := strings.IndexByte(key, '@'); at >= 0 {
		return key[:at], key[at+1:]
	}
	return key, ""
}

func (score *Score) HasTally() bool {
	return len(score.Actions) > 0 || len(score.Slots) > 0 || len(score.Adjust) > 0 ||
		len(score.Ever) > 0 || score.SeasonKey != ""
}

func (score *Score) Count(id, period string) int {
	if score.Actions == nil {
		return 0
	}
	return score.Actions[TallyKey(id, period)]
}

func (score *Score) CountAll(action *SeasonAction) int {
	total := 0
	for _, period := range action.PeriodsOrAny() {
		total += score.Count(action.ID, period)
	}
	return total
}

func (score *Score) EverCount(id, period string) int {
	if score.Ever == nil {
		return 0
	}
	return score.Ever[TallyKey(id, period)]
}

func (score *Score) Adjustment(id string) int {
	if score.Adjust == nil {
		return 0
	}
	return score.Adjust[id]
}

func (score *Score) States(groupId string) []SlotState {
	if score.Slots == nil {
		return nil
	}
	return score.Slots[groupId]
}

func (score *Score) SetAction(id, period string, value int) {
	if score.Actions == nil {
		score.Actions = map[string]int{}
	}
	key := TallyKey(id, period)
	if value <= 0 {
		delete(score.Actions, key)
	} else {
		score.Actions[key] = value
	}
	if value > score.EverCount(id, period) {
		if score.Ever == nil {
			score.Ever = map[string]int{}
		}
		score.Ever[key] = value
	}
}

func (score *Score) AddAction(id, period string, delta int) {
	score.SetAction(id, period, score.Count(id, period)+delta)
}

func (score *Score) SetAdjustment(id string, value int) {
	if score.Adjust == nil {
		score.Adjust = map[string]int{}
	}
	if value == 0 {
		delete(score.Adjust, id)
		return
	}
	score.Adjust[id] = value
}

func (score *Score) SetDraw(id, value string) {
	if score.Draw == nil {
		score.Draw = map[string]string{}
	}
	score.Draw[id] = value
}

func (score *Score) ensureSlots(groupId string, count int) []SlotState {
	if score.Slots == nil {
		score.Slots = map[string][]SlotState{}
	}
	states := score.Slots[groupId]
	for len(states) < count {
		states = append(states, SlotState{})
	}
	score.Slots[groupId] = states
	return states
}

// Occupy writes one slot of a group. An empty option clears the occupant and keeps everIn, which is
// what tells "the points are gone" apart from "it was there during the autonomous".
func (score *Score) Occupy(group *SlotGroup, slot int, optionId, period string) {
	if slot < 1 {
		return
	}
	states := score.ensureSlots(group.ID, slot)
	state := &states[slot-1]
	if optionId == "" {
		if group.ExclusiveBy == ExclusiveBySlotTime && period != "" && state.PerPeriod != nil {
			delete(state.PerPeriod, period)
			if len(state.PerPeriod) == 0 {
				state.PerPeriod = nil
			}
		} else {
			state.PerPeriod = nil
		}
		state.Occupant = ""
		state.OccupiedIn = ""
		score.Slots[group.ID] = states
		return
	}
	option := group.Option(optionId)
	if option == nil {
		return
	}
	landed := period
	if landed == "" {
		landed = option.Period
	}
	state.Occupant = optionId
	state.OccupiedIn = landed
	if group.ExclusiveBy == ExclusiveBySlotTime && landed != "" {
		if state.PerPeriod == nil {
			state.PerPeriod = map[string]string{}
		}
		state.PerPeriod[landed] = optionId
	}
	if group.TracksEver && landed != "" && !contains(state.EverIn, landed) {
		state.EverIn = append(state.EverIn, landed)
	}
	score.Slots[group.ID] = states
}

func (score *Score) OccupiedCount(group *SlotGroup, period string) int {
	count := 0
	for _, state := range score.States(group.ID) {
		if group.ExclusiveBy == ExclusiveBySlotTime {
			for occupiedIn := range state.PerPeriod {
				if period == "" || occupiedIn == period {
					count++
				}
			}
			continue
		}
		if state.Occupant == "" {
			continue
		}
		if period == "" || state.OccupiedIn == period {
			count++
		}
	}
	return count
}

func (score *Score) EverOccupiedCount(group *SlotGroup, period string) int {
	count := 0
	for _, state := range score.States(group.ID) {
		if period == "" {
			if len(state.EverIn) > 0 {
				count++
			}
			continue
		}
		if contains(state.EverIn, period) {
			count++
		}
	}
	return count
}

func (score *Score) GroupPoints(group *SlotGroup, period string) int {
	total := 0
	for _, state := range score.States(group.ID) {
		if group.ExclusiveBy == ExclusiveBySlotTime {
			for occupiedIn, optionId := range state.PerPeriod {
				if period != "" && occupiedIn != period {
					continue
				}
				if option := group.Option(optionId); option != nil {
					total += option.PointsIn(occupiedIn)
				}
			}
			continue
		}
		if state.Occupant == "" {
			continue
		}
		if period != "" && state.OccupiedIn != period {
			continue
		}
		if option := group.Option(state.Occupant); option != nil {
			total += option.PointsIn(state.OccupiedIn)
		}
	}
	return total
}

func (score *Score) ActionPoints(action *SeasonAction, period string) int {
	if action.Unit == UnitFreeValue {
		// A free value is one number, not a count per period. The declared period says only where
		// its points land, so it is read once and never once per period.
		if period != "" && period != action.firstPeriod() {
			return 0
		}
		return score.Adjustment(action.ID)
	}
	total := 0
	for _, candidate := range action.PeriodsOrAny() {
		if period != "" && candidate != period {
			continue
		}
		count := score.Count(action.ID, candidate)
		if count == 0 {
			continue
		}
		if action.Unit == UnitTiered {
			total += action.tierValue(count)
			continue
		}
		total += count * action.PointsIn(candidate)
	}
	return total
}

func (action *SeasonAction) firstPeriod() string {
	if len(action.Periods) == 0 {
		return ""
	}
	return action.Periods[0]
}

func (action *SeasonAction) tierValue(count int) int {
	if len(action.Tiers) == 0 {
		return count * action.PointsIn("")
	}
	best := 0
	bestStep := -1
	for step, value := range action.Tiers {
		threshold := atoiOr(step, -1)
		if threshold < 0 || threshold > count {
			continue
		}
		if threshold > bestStep {
			bestStep = threshold
			best = value
		}
	}
	return best
}

func contains(list []string, value string) bool {
	for _, candidate := range list {
		if candidate == value {
			return true
		}
	}
	return false
}

func atoiOr(text string, fallback int) int {
	value := 0
	if text == "" {
		return fallback
	}
	for _, digit := range text {
		if digit < '0' || digit > '9' {
			return fallback
		}
		value = value*10 + int(digit-'0')
	}
	return value
}

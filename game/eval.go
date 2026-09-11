package game

import (
	"encoding/json"
	"fmt"
)

type Expr json.RawMessage

func (expr Expr) MarshalJSON() ([]byte, error) {
	if len(expr) == 0 {
		return []byte("null"), nil
	}
	return expr, nil
}

func (expr *Expr) UnmarshalJSON(body []byte) error {
	*expr = Expr(append([]byte(nil), body...))
	return nil
}

func (expr Expr) Empty() bool { return len(expr) == 0 || string(expr) == "null" }

type MatchOutcome struct {
	Total         int
	Categories    map[string]int
	PeriodPoints  map[string]int
	Flags         []string
	FlagPoints    map[string]int
	RpEarned      []string
	RpEligible    []string
	RankingPoints int
	Sort          []int
	Progress      map[string]string
}

type tallySide struct {
	season     *Season
	level      string
	score      *Score
	categories map[string]int
	periods    map[string]int
	total      int
}

type evaluator struct {
	own      *tallySide
	opponent *tallySide
	flags    map[string]bool
	flagPts  map[string]int
	eligible map[string]bool
	stage    int
}

const (
	stageTally = 4
	stageFlags = 6
	stageRp    = 7
	stageSort  = 10
)

// Evaluate turns a tally into everything a screen, a ranking or a public page needs. The stages run
// in a fixed order and never call back into an earlier one, which is what keeps it total: the
// opponent is read only through its tally side, so two alliances can never wait on each other.
func Evaluate(season *Season, level string, own, opponent *Score) *MatchOutcome {
	outcome := &MatchOutcome{
		Categories:   map[string]int{},
		PeriodPoints: map[string]int{},
		FlagPoints:   map[string]int{},
		Progress:     map[string]string{},
	}
	if season == nil || own == nil {
		return outcome
	}
	if level == "" {
		level = season.DefaultLevel()
	}

	ownSide := newTallySide(season, level, own)
	oppSide := newTallySide(season, level, opponent)
	ownSide.creditFouls(oppSide)
	oppSide.creditFouls(ownSide)
	ownSide.settleTotal()
	oppSide.settleTotal()

	eval := &evaluator{
		own:      ownSide,
		opponent: oppSide,
		flags:    map[string]bool{},
		flagPts:  map[string]int{},
		eligible: map[string]bool{},
		stage:    stageFlags,
	}

	for _, flag := range season.Flags {
		on := eval.truth(flag.When)
		eval.flags[flag.ID] = on
		if !on {
			continue
		}
		outcome.Flags = append(outcome.Flags, flag.ID)
		eval.flagPts[flag.ID] = flag.Points
		outcome.FlagPoints[flag.ID] = flag.Points
		if flag.Into == IntoMatchScore {
			ownSide.total += flag.Points
		}
	}

	eval.stage = stageRp
	earned := map[string]bool{}
	for _, rp := range season.Ranking.RankingPoints {
		if !eval.truth(rp.When) {
			continue
		}
		earned[rp.ID] = true
		outcome.RpEarned = append(outcome.RpEarned, rp.ID)
	}
	for id := range earned {
		eval.eligible[id] = true
	}
	for _, void := range season.Ranking.Voids {
		if eval.eligible[void.Rp] && eval.truth(void.When) {
			eval.eligible[void.Rp] = false
		}
	}
	for _, grant := range season.Ranking.Grants {
		if eval.truth(grant.When) {
			eval.eligible[grant.Rp] = true
		}
	}
	for _, rp := range season.Ranking.RankingPoints {
		if eval.eligible[rp.ID] {
			outcome.RpEligible = append(outcome.RpEligible, rp.ID)
			outcome.RankingPoints += rp.Points
		}
		outcome.Progress[rp.ID] = rp.Label
	}

	eval.stage = stageSort
	for _, tiebreaker := range season.Ranking.Tiebreakers {
		outcome.Sort = append(outcome.Sort, eval.number(tiebreaker.Expr))
	}

	outcome.Total = ownSide.total
	for id, value := range ownSide.categories {
		outcome.Categories[id] = value
	}
	for id, value := range ownSide.periods {
		outcome.PeriodPoints[id] = value
	}
	return outcome
}

func newTallySide(season *Season, level string, score *Score) *tallySide {
	side := &tallySide{
		season:     season,
		level:      level,
		score:      score,
		categories: map[string]int{},
		periods:    map[string]int{},
	}
	for _, category := range season.Categories {
		side.categories[category.ID] = 0
	}
	if score == nil {
		return side
	}

	for i := range season.SlotGroups {
		group := &season.SlotGroups[i]
		for _, period := range append(season.ScoringPeriods(), "") {
			if period == "" {
				continue
			}
			points := score.GroupPoints(group, period)
			side.categories[group.Category] += points
			side.periods[period] += points
		}
	}
	for i := range season.Actions {
		action := &season.Actions[i]
		if action.Kind == KindToggle {
			continue
		}
		for _, period := range action.PeriodsOrAny() {
			points := score.ActionPoints(action, period)
			if points == 0 {
				continue
			}
			if action.Kind == KindFoul || action.Kind == KindAdjustment {
				side.categories[CategoryFoulDone] += points
				continue
			}
			side.categories[action.Category] += points
			if period != "" {
				side.periods[period] += points
			}
		}
	}
	return side
}

func (side *tallySide) creditFouls(from *tallySide) {
	if from == nil || from.score == nil {
		return
	}
	credited := 0
	for i := range side.season.Actions {
		action := &side.season.Actions[i]
		if !action.CreditsOpponent {
			continue
		}
		for _, period := range action.PeriodsOrAny() {
			credited += from.score.ActionPoints(action, period)
		}
	}
	side.categories[CategoryFoulTaken] = credited
}

func (side *tallySide) settleTotal() {
	total := 0
	for _, category := range side.season.Categories {
		if category.Kind == CategoryFoulDone {
			continue
		}
		total += side.categories[category.ID]
	}
	side.total = total
}

func (eval *evaluator) truth(expr Expr) bool {
	if expr.Empty() {
		return false
	}
	return eval.value(expr, eval.own) != 0
}

func (eval *evaluator) number(expr Expr) int {
	if expr.Empty() {
		return 0
	}
	return eval.value(expr, eval.own)
}

func (eval *evaluator) value(expr Expr, side *tallySide) int {
	var scalar int
	if json.Unmarshal(expr, &scalar) == nil {
		return scalar
	}
	var flag bool
	if json.Unmarshal(expr, &flag) == nil {
		return boolToInt(flag)
	}
	var parts []json.RawMessage
	if json.Unmarshal(expr, &parts) != nil || len(parts) == 0 {
		return 0
	}
	var operator string
	if json.Unmarshal(parts[0], &operator) != nil {
		return 0
	}
	args := parts[1:]

	switch operator {
	case "n":
		return side.count(literal(args, 0), literal(args, 1))
	case "ever":
		return side.ever(literal(args, 0), literal(args, 1))
	case "pts":
		return side.points(literal(args, 0), literal(args, 1))
	case "cat":
		return side.categories[literal(args, 0)]
	case "total":
		return side.total
	case "thr":
		return side.season.Threshold(literal(args, 0), side.level)
	case "robots":
		return side.robots()
	case "draw":
		return side.drawIndex(literal(args, 0))
	case "flag":
		return boolToInt(eval.flags[literal(args, 0)])
	case "flagPts":
		return eval.flagPts[literal(args, 0)]
	case "rp":
		return boolToInt(eval.eligible[literal(args, 0)])
	case "opp":
		if len(args) == 0 {
			return 0
		}
		return eval.value(Expr(args[0]), eval.opponentOf(side))
	case "+":
		total := 0
		for _, arg := range args {
			total += eval.value(Expr(arg), side)
		}
		return total
	case "-":
		if len(args) == 0 {
			return 0
		}
		total := eval.value(Expr(args[0]), side)
		if len(args) == 1 {
			return -total
		}
		for _, arg := range args[1:] {
			total -= eval.value(Expr(arg), side)
		}
		return total
	case "*":
		total := 1
		for _, arg := range args {
			total *= eval.value(Expr(arg), side)
		}
		return total
	case "div":
		if len(args) < 2 {
			return 0
		}
		divisor := eval.value(Expr(args[1]), side)
		if divisor == 0 {
			return 0
		}
		return floorDiv(eval.value(Expr(args[0]), side), divisor)
	case "min", "max":
		if len(args) == 0 {
			return 0
		}
		best := eval.value(Expr(args[0]), side)
		for _, arg := range args[1:] {
			value := eval.value(Expr(arg), side)
			if (operator == "min") == (value < best) {
				best = value
			}
		}
		return best
	case ">=", ">", "<=", "<", "==":
		if len(args) < 2 {
			return 0
		}
		left := eval.value(Expr(args[0]), side)
		right := eval.value(Expr(args[1]), side)
		return boolToInt(compare(operator, left, right))
	case "and":
		for _, arg := range args {
			if eval.value(Expr(arg), side) == 0 {
				return 0
			}
		}
		return 1
	case "or":
		for _, arg := range args {
			if eval.value(Expr(arg), side) != 0 {
				return 1
			}
		}
		return 0
	case "not":
		if len(args) == 0 {
			return 1
		}
		return boolToInt(eval.value(Expr(args[0]), side) == 0)
	case "if":
		if len(args) < 3 {
			return 0
		}
		if eval.value(Expr(args[0]), side) != 0 {
			return eval.value(Expr(args[1]), side)
		}
		return eval.value(Expr(args[2]), side)
	case "countTrue":
		if len(args) == 0 {
			return 0
		}
		var list []json.RawMessage
		if json.Unmarshal(args[0], &list) != nil {
			return 0
		}
		count := 0
		for _, arg := range list {
			if eval.value(Expr(arg), side) != 0 {
				count++
			}
		}
		return count
	}
	return 0
}

func (eval *evaluator) opponentOf(side *tallySide) *tallySide {
	if side == eval.own {
		return eval.opponent
	}
	return eval.own
}

func (side *tallySide) count(id, period string) int {
	if side.score == nil {
		return 0
	}
	if group := side.season.Group(id); group != nil {
		return side.score.OccupiedCount(group, period)
	}
	action := side.season.Action(id)
	if action == nil {
		return 0
	}
	if action.Unit == UnitFreeValue {
		return side.score.Adjustment(id)
	}
	if period != "" {
		return side.score.Count(id, period)
	}
	return side.score.CountAll(action)
}

func (side *tallySide) ever(id, period string) int {
	if side.score == nil {
		return 0
	}
	if group := side.season.Group(id); group != nil {
		return side.score.EverOccupiedCount(group, period)
	}
	if period != "" {
		return side.score.EverCount(id, period)
	}
	total := 0
	if action := side.season.Action(id); action != nil {
		for _, candidate := range action.PeriodsOrAny() {
			total += side.score.EverCount(id, candidate)
		}
	}
	return total
}

func (side *tallySide) points(id, period string) int {
	if side.score == nil {
		return 0
	}
	if group := side.season.Group(id); group != nil {
		return side.score.GroupPoints(group, period)
	}
	if action := side.season.Action(id); action != nil {
		return side.score.ActionPoints(action, period)
	}
	return 0
}

func (side *tallySide) robots() int {
	if side.score != nil && side.score.Robots > 0 {
		return side.score.Robots
	}
	return side.season.RobotsPerAlliance
}

func (side *tallySide) drawIndex(id string) int {
	if side.score == nil {
		return 0
	}
	drawn := side.score.Draw[id]
	for _, candidate := range side.season.MatchRandom {
		if candidate.ID != id {
			continue
		}
		for index, value := range candidate.Values {
			if value.ID == drawn {
				return index + 1
			}
		}
	}
	return 0
}

func literal(args []json.RawMessage, index int) string {
	if index >= len(args) {
		return ""
	}
	var text string
	if json.Unmarshal(args[index], &text) != nil {
		return ""
	}
	return text
}

func compare(operator string, left, right int) bool {
	switch operator {
	case ">=":
		return left >= right
	case ">":
		return left > right
	case "<=":
		return left <= right
	case "<":
		return left < right
	case "==":
		return left == right
	}
	return false
}

func floorDiv(numerator, denominator int) int {
	quotient := numerator / denominator
	if (numerator%denominator != 0) && ((numerator < 0) != (denominator < 0)) {
		quotient--
	}
	return quotient
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func errExpr(context string, expr Expr, reason string) error {
	return fmt.Errorf("%s: %s (%s)", context, reason, string(expr))
}

package game

import (
	"encoding/json"
	"fmt"
	"sort"
)

var derivedReaders = map[string]bool{
	"flag": true, "flagPts": true, "rp": true, "opp": true,
}

var arity = map[string]int{
	"n": 1, "ever": 1, "pts": 1, "cat": 1, "total": 0, "thr": 1, "flag": 1, "flagPts": 1,
	"rp": 1, "robots": 0, "draw": 1, "opp": 1,
	"+": -1, "-": -1, "*": -1, "div": 2, "min": -1, "max": -1,
	">=": 2, ">": 2, "<=": 2, "<": 2, "==": 2,
	"and": -1, "or": -1, "not": 1, "if": 3, "countTrue": 1,
}

func init() {
	if len(arity) > MaxOperators {
		panic(fmt.Sprintf("the season language has %d operators, over the declared ceiling of %d",
			len(arity), MaxOperators))
	}
}

func (season *Season) Validate() error {
	if season.Schema != SeasonSchema {
		return fmt.Errorf("schema %d: this build reads schema %d", season.Schema, SeasonSchema)
	}
	if season.Key == "" || season.Program == "" || season.Name == "" {
		return fmt.Errorf("key, program and name are required")
	}
	if season.Revision < 1 {
		return fmt.Errorf("revision starts at 1")
	}
	if season.Format != FormatAlliance && season.Format != FormatSheet {
		return fmt.Errorf("format %q is neither %q nor %q", season.Format, FormatAlliance, FormatSheet)
	}
	if len(season.Periods) == 0 {
		return fmt.Errorf("at least one period is required")
	}
	if len(season.EventLevels) == 0 {
		return fmt.Errorf("at least one event level is required")
	}
	if len(season.Categories) == 0 {
		return fmt.Errorf("at least one category is required")
	}
	if season.Format == FormatAlliance {
		if season.RobotsPerAlliance < 1 {
			return fmt.Errorf("format %q needs robotsPerAlliance", FormatAlliance)
		}
		if len(season.Actions) == 0 && len(season.SlotGroups) == 0 {
			return fmt.Errorf("format %q needs actions or slotGroups", FormatAlliance)
		}
	}
	if season.Format == FormatSheet && season.Sheet == nil {
		return fmt.Errorf("format %q needs the sheet block", FormatSheet)
	}

	if err := season.validateNames(); err != nil {
		return err
	}
	if err := season.validateThresholds(); err != nil {
		return err
	}
	if err := season.validateExpressions(); err != nil {
		return err
	}
	return season.validateTests()
}

func (season *Season) validateNames() error {
	// Uniqueness is per reader, not global: "n"/"ever"/"pts" read actions and slot groups, so those
	// two share a namespace; "cat" reads categories and "flag" reads flags plus the toggle actions.
	// REEFSCAPE names a category and a slot group "leave" on purpose, and nothing can confuse them.
	tally := map[string]string{}
	flagged := map[string]string{}
	claim := func(space map[string]string, kind, id string) error {
		if id == "" {
			return fmt.Errorf("%s with no id", kind)
		}
		if previous, taken := space[id]; taken {
			return fmt.Errorf("id %q is used by both %s and %s", id, previous, kind)
		}
		space[id] = kind
		return nil
	}
	levels := map[string]bool{AnyLevel: true}
	for _, level := range season.EventLevels {
		levels[level.ID] = true
	}
	periods := map[string]bool{}
	for _, period := range season.Periods {
		if period.ID == "" {
			return fmt.Errorf("period with no id")
		}
		periods[period.ID] = true
	}
	categories := map[string]bool{}
	for _, category := range season.Categories {
		if categories[category.ID] {
			return fmt.Errorf("category %q appears twice", category.ID)
		}
		if category.ID == "" {
			return fmt.Errorf("category with no id")
		}
		if category.Kind != "" && category.Kind != CategoryFoulDone && category.Kind != CategoryFoulTaken {
			return fmt.Errorf("category %q has kind %q, which is not reserved", category.ID, category.Kind)
		}
		categories[category.ID] = true
	}

	for _, group := range season.SlotGroups {
		if err := claim(tally, "slot group", group.ID); err != nil {
			return err
		}
		if !categories[group.Category] {
			return fmt.Errorf("slot group %q points at unknown category %q", group.ID, group.Category)
		}
		if group.ExclusiveBy != ExclusiveBySlot && group.ExclusiveBy != ExclusiveBySlotTime {
			return fmt.Errorf("slot group %q has exclusiveBy %q", group.ID, group.ExclusiveBy)
		}
		if len(group.Options) == 0 {
			return fmt.Errorf("slot group %q has no options", group.ID)
		}
		optionIds := map[string]bool{}
		for _, option := range group.Options {
			if option.ID == "" || optionIds[option.ID] {
				return fmt.Errorf("slot group %q repeats or misses an option id", group.ID)
			}
			optionIds[option.ID] = true
			if option.Period != "" && !periods[option.Period] {
				return fmt.Errorf("option %q of %q lands in unknown period %q",
					option.ID, group.ID, option.Period)
			}
			if err := checkPointsPeriods(option.Points, periods,
				fmt.Sprintf("option %q of %q", option.ID, group.ID)); err != nil {
				return err
			}
		}
		if group.ExclusiveBy == ExclusiveBySlotTime && !group.Provenance {
			return fmt.Errorf("slot group %q is exclusive per period and has to carry provenance", group.ID)
		}
	}

	for _, action := range season.Actions {
		if err := claim(tally, "action", action.ID); err != nil {
			return err
		}
		switch action.Unit {
		case UnitCount, UnitTiered, UnitToggle, UnitChoice, UnitFreeValue:
		default:
			return fmt.Errorf("action %q has unit %q", action.ID, action.Unit)
		}
		switch action.Kind {
		case "", KindFoul, KindToggle, KindAdjustment:
		default:
			return fmt.Errorf("action %q has kind %q", action.ID, action.Kind)
		}
		if action.Kind == "" && action.Category == "" {
			return fmt.Errorf("action %q scores and names no category", action.ID)
		}
		if action.Category != "" && !categories[action.Category] {
			return fmt.Errorf("action %q points at unknown category %q", action.ID, action.Category)
		}
		for _, period := range action.Periods {
			if !periods[period] {
				return fmt.Errorf("action %q declares unknown period %q", action.ID, period)
			}
		}
		for _, period := range action.RemovalOrder {
			if !periods[period] {
				return fmt.Errorf("action %q removes from unknown period %q", action.ID, period)
			}
		}
		if err := checkPointsPeriods(action.Points, periods, "action "+action.ID); err != nil {
			return err
		}
	}

	for _, action := range season.Actions {
		if action.Kind == KindToggle {
			if err := claim(flagged, "toggle action", action.ID); err != nil {
				return err
			}
		}
	}
	for _, flag := range season.Flags {
		if err := claim(flagged, "flag", flag.ID); err != nil {
			return err
		}
		if flag.Into != "" && flag.Into != IntoMatchScore && flag.Into != IntoCoopertition {
			return fmt.Errorf("flag %q goes into %q, which is not a destination", flag.ID, flag.Into)
		}
	}
	draws := map[string]string{}
	for _, draw := range season.MatchRandom {
		if err := claim(draws, "draw", draw.ID); err != nil {
			return err
		}
		if draw.Scope != "match" && draw.Scope != "perAlliance" {
			return fmt.Errorf("draw %q has scope %q", draw.ID, draw.Scope)
		}
		if len(draw.Values) == 0 {
			return fmt.Errorf("draw %q has no values", draw.ID)
		}
	}

	rpIds := map[string]bool{}
	for _, rp := range season.Ranking.RankingPoints {
		if rp.ID == "" || rpIds[rp.ID] {
			return fmt.Errorf("ranking point %q repeats or misses an id", rp.ID)
		}
		rpIds[rp.ID] = true
	}
	for _, change := range append(season.Ranking.Voids, season.Ranking.Grants...) {
		if !rpIds[change.Rp] {
			return fmt.Errorf("voids/grants name unknown ranking point %q", change.Rp)
		}
	}
	sortIds := map[string]bool{}
	for _, tiebreaker := range season.Ranking.Tiebreakers {
		if tiebreaker.ID == "" || sortIds[tiebreaker.ID] {
			return fmt.Errorf("tiebreaker %q repeats or misses an id", tiebreaker.ID)
		}
		sortIds[tiebreaker.ID] = true
		if tiebreaker.Direction != "" && tiebreaker.Direction != "desc" && tiebreaker.Direction != "asc" {
			return fmt.Errorf("tiebreaker %q has direction %q", tiebreaker.ID, tiebreaker.Direction)
		}
		if tiebreaker.Agg != "" && tiebreaker.Agg != "sum" && tiebreaker.Agg != "max" && tiebreaker.Agg != "min" {
			return fmt.Errorf("tiebreaker %q aggregates by %q", tiebreaker.ID, tiebreaker.Agg)
		}
	}
	switch season.Ranking.Mode {
	case "alliance", "bestOf":
	default:
		return fmt.Errorf("ranking mode %q", season.Ranking.Mode)
	}
	if !levels[season.DefaultLevel()] {
		return fmt.Errorf("the first event level is not a level")
	}
	return nil
}

func checkPointsPeriods(raw json.RawMessage, periods map[string]bool, context string) error {
	if len(raw) == 0 {
		return nil
	}
	var scalar int
	if json.Unmarshal(raw, &scalar) == nil {
		return nil
	}
	var byPeriod map[string]int
	if json.Unmarshal(raw, &byPeriod) != nil {
		return fmt.Errorf("%s has points that are neither a number nor a map by period", context)
	}
	for period := range byPeriod {
		if period != AnyLevel && !periods[period] {
			return fmt.Errorf("%s pays in unknown period %q", context, period)
		}
	}
	return nil
}

func (season *Season) validateThresholds() error {
	levels := map[string]bool{AnyLevel: true}
	for _, level := range season.EventLevels {
		levels[level.ID] = true
	}
	for id, bands := range season.Thresholds {
		if len(bands) == 0 {
			return fmt.Errorf("threshold %q names no level", id)
		}
		for level := range bands {
			if !levels[level] {
				return fmt.Errorf("threshold %q names unknown level %q", id, level)
			}
		}
		if _, wildcard := bands[AnyLevel]; !wildcard {
			for _, level := range season.EventLevels {
				if _, given := bands[level.ID]; !given {
					return fmt.Errorf("threshold %q has no value for level %q and no %q",
						id, level.ID, AnyLevel)
				}
			}
		}
	}
	return nil
}

type exprScope struct {
	label      string
	insideOpp  bool
	allowRp    bool
	allowFlags bool
}

func (season *Season) validateExpressions() error {
	check := func(expr Expr, scope exprScope) error {
		return season.walkExpr(expr, scope)
	}
	for _, group := range season.SlotGroups {
		for period, expr := range group.MaxOccupied {
			if err := check(expr, exprScope{
				label: fmt.Sprintf("maxOccupied[%s] of %s", period, group.ID)}); err != nil {
				return err
			}
		}
	}
	for _, action := range season.Actions {
		if !action.Derived.Empty() {
			if err := check(action.Derived, exprScope{label: "derived of " + action.ID}); err != nil {
				return err
			}
		}
	}
	for _, flag := range season.Flags {
		if err := check(flag.When, exprScope{label: "flag " + flag.ID, allowFlags: true}); err != nil {
			return err
		}
	}
	for _, rp := range season.Ranking.RankingPoints {
		if err := check(rp.When, exprScope{label: "rp " + rp.ID, allowFlags: true}); err != nil {
			return err
		}
	}
	for _, change := range season.Ranking.Voids {
		if err := check(change.When, exprScope{label: "void of " + change.Rp, allowFlags: true}); err != nil {
			return err
		}
	}
	for _, change := range season.Ranking.Grants {
		if err := check(change.When, exprScope{label: "grant of " + change.Rp, allowFlags: true}); err != nil {
			return err
		}
	}
	for _, tiebreaker := range append(season.Ranking.Tiebreakers, season.MatchTiebreakers...) {
		if err := check(tiebreaker.Expr, exprScope{
			label: "tiebreaker " + tiebreaker.ID, allowFlags: true, allowRp: true}); err != nil {
			return err
		}
	}
	return nil
}

func (season *Season) walkExpr(expr Expr, scope exprScope) error {
	if expr.Empty() {
		return fmt.Errorf("%s has no expression", scope.label)
	}
	var scalar int
	if json.Unmarshal(expr, &scalar) == nil {
		return nil
	}
	var flag bool
	if json.Unmarshal(expr, &flag) == nil {
		return nil
	}
	var parts []json.RawMessage
	if json.Unmarshal(expr, &parts) != nil || len(parts) == 0 {
		return errExpr(scope.label, expr, "not an operator array")
	}
	var operator string
	if json.Unmarshal(parts[0], &operator) != nil {
		return errExpr(scope.label, expr, "the first item is not an operator name")
	}
	wanted, known := arity[operator]
	if !known {
		return errExpr(scope.label, expr, "unknown operator "+operator)
	}
	args := parts[1:]
	if wanted >= 0 && len(args) < wanted {
		return errExpr(scope.label, expr, fmt.Sprintf("%s wants %d arguments", operator, wanted))
	}

	if scope.insideOpp && derivedReaders[operator] {
		return errExpr(scope.label, expr, "opp only reads the tally, never "+operator)
	}
	if operator == "rp" && !scope.allowRp {
		return errExpr(scope.label, expr, "rp is only legal in a tiebreaker")
	}
	if (operator == "flag" || operator == "flagPts") && !scope.allowFlags {
		return errExpr(scope.label, expr, operator+" is not legal here")
	}

	switch operator {
	case "n", "ever", "pts":
		id := literal(args, 0)
		if season.Group(id) == nil && season.Action(id) == nil {
			return errExpr(scope.label, expr, "unknown action or slot group "+id)
		}
		if period := literal(args, 1); period != "" && season.Period(period) == nil {
			return errExpr(scope.label, expr, "unknown period "+period)
		}
		return nil
	case "cat":
		if season.byCategory[literal(args, 0)] == nil {
			return errExpr(scope.label, expr, "unknown category "+literal(args, 0))
		}
		return nil
	case "thr":
		if _, known := season.Thresholds[literal(args, 0)]; !known {
			return errExpr(scope.label, expr, "unknown threshold "+literal(args, 0))
		}
		return nil
	case "flag", "flagPts":
		id := literal(args, 0)
		if season.Flag(id) == nil {
			if action := season.Action(id); action == nil || action.Kind != KindToggle {
				return errExpr(scope.label, expr, "unknown flag "+id)
			}
		}
		return nil
	case "rp":
		for _, rp := range season.Ranking.RankingPoints {
			if rp.ID == literal(args, 0) {
				return nil
			}
		}
		return errExpr(scope.label, expr, "unknown ranking point "+literal(args, 0))
	case "draw":
		for _, draw := range season.MatchRandom {
			if draw.ID == literal(args, 0) {
				return nil
			}
		}
		return errExpr(scope.label, expr, "unknown draw "+literal(args, 0))
	case "total", "robots":
		return nil
	case "opp":
		inner := scope
		inner.insideOpp = true
		inner.allowFlags = false
		inner.allowRp = false
		return season.walkExpr(Expr(args[0]), inner)
	case "countTrue":
		var list []json.RawMessage
		if json.Unmarshal(args[0], &list) != nil || len(list) == 0 {
			return errExpr(scope.label, expr, "countTrue wants a list of conditions")
		}
		for _, item := range list {
			if err := season.walkExpr(Expr(item), scope); err != nil {
				return err
			}
		}
		return nil
	}

	for _, arg := range args {
		if err := season.walkExpr(Expr(arg), scope); err != nil {
			return err
		}
	}
	return nil
}

// Every ranking point has to be true in one case and false in another. A season whose RP never
// fires reads exactly like a season whose RP works, until a Sunday afternoon.
func (season *Season) validateTests() error {
	if len(season.Tests) == 0 {
		return fmt.Errorf("a season publishes with its own cases, and this one has none")
	}
	fired := map[string]bool{}
	quiet := map[string]bool{}
	for index, test := range season.Tests {
		if test.Name == "" {
			return fmt.Errorf("case %d has no name", index+1)
		}
		if test.Level != "" && !season.KnowsLevel(test.Level) {
			return fmt.Errorf("case %q runs at unknown level %q", test.Name, test.Level)
		}
		if test.State == nil && len(test.Steps) == 0 {
			return fmt.Errorf("case %q has neither state nor steps", test.Name)
		}
		if test.State != nil && len(test.Steps) > 0 {
			return fmt.Errorf("case %q has both state and steps", test.Name)
		}
	}
	for _, test := range season.Tests {
		outcome, err := season.runTest(test)
		if err != nil {
			return err
		}
		earned := map[string]bool{}
		for _, id := range outcome.RpEarned {
			earned[id] = true
			fired[id] = true
		}
		for _, rp := range season.Ranking.RankingPoints {
			if !earned[rp.ID] {
				quiet[rp.ID] = true
			}
		}
	}
	missing := []string{}
	for _, rp := range season.Ranking.RankingPoints {
		if !fired[rp.ID] {
			missing = append(missing, rp.ID+" (never true)")
		}
		if !quiet[rp.ID] {
			missing = append(missing, rp.ID+" (never false)")
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("the cases do not cover: %v", missing)
	}
	return nil
}

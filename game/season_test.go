package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Every embedded season plays its own cases. A season whose ranking point never fires reads exactly
// like a season whose ranking point works, and the difference only shows up on a Sunday afternoon.
func TestEverySeasonPassesItsOwnCases(t *testing.T) {
	packs := AllSeasons()
	assert.NotEmpty(t, packs)
	for _, season := range packs {
		for _, failure := range season.RunTests() {
			t.Errorf("%s r%d — %s", season.Key, season.Revision, failure.Error())
		}
	}
}

func TestEverySeasonValidates(t *testing.T) {
	for _, season := range AllSeasons() {
		assert.Nil(t, season.Validate(), season.Key)
	}
}

func TestReefscapeIsEmbedded(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	if !assert.NotNil(t, season) {
		return
	}
	assert.Equal(t, "FRC", season.Program)
	assert.Equal(t, 3, season.RobotsPerAlliance)
	assert.True(t, season.HasResult())
	assert.Equal(t, 3, season.Ranking.Result.Win)
	assert.Equal(t, 5, season.Threshold("coralPerLevel", "REGIONAL"))
	assert.Equal(t, 7, season.Threshold("coralPerLevel", "CHAMPIONSHIP"))
	assert.Equal(t, 2, season.Threshold("coopAlgae", "CHAMPIONSHIP"))
	assert.Equal(t, []string{"auto", "teleop"}, season.ScoringPeriods())
	assert.Equal(t, 15, season.Period("auto").Duration("REGIONAL"))
}

func TestAnEmptyKeyFallsBackToTheLegacySeason(t *testing.T) {
	assert.Equal(t, LegacySeasonKey, SeasonByKey("").Key)
	assert.Nil(t, SeasonByKey("frc-1999-nada"))
}

// A result from before the seasons existed has no tally, and has to keep summarizing to the same
// numbers it always did. This is the whole compatibility promise of the slice.
func TestAScoreWithNoTallyStillSumsTheFourNumbers(t *testing.T) {
	score := TestScore1()
	assert.False(t, score.HasTally())
	summary := score.Summarize()
	assert.Equal(t, 45, summary.AutoPoints)
	assert.Equal(t, 80, summary.TeleopPoints)
	assert.Equal(t, 30, summary.EndgamePoints)
	assert.Equal(t, 155, summary.Score)
	assert.Nil(t, summary.Outcome)
}

func TestTheTallyOfTheDossierScenario(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	own := &Score{SeasonKey: season.Key, Level: "REGIONAL", Robots: 3}
	for slot := 1; slot <= 3; slot++ {
		own.Occupy(season.Group("leave"), slot, "left", "")
		own.Occupy(season.Group("coralL4"), slot, "coral", "auto")
	}
	own.Occupy(season.Group("endgame"), 1, "deep", "")
	own.SetAction("algaeProcessor", "teleop", 2)

	opponent := &Score{SeasonKey: season.Key, Level: "REGIONAL", Robots: 3}
	opponent.SetAction("algaeProcessor", "teleop", 2)

	summary := own.SummarizeAgainst(opponent)
	assert.Equal(t, 54, summary.Score)
	if assert.NotNil(t, summary.Outcome) {
		assert.Equal(t, []int{1, 54, 30, 12}, summary.Outcome.Sort)
		assert.Equal(t, []string{"autoRp"}, summary.Outcome.RpEligible)
		assert.Equal(t, 1, summary.Outcome.RankingPoints)
	}
	// The autonomous half of the score is what the old AutoPoints field used to hold.
	assert.Equal(t, 30, summary.AutoPoints)
}

// The half of Team Update 20 that the official system got wrong: the coral taken out in the teleop
// stops paying and still counts for the AUTO RP.
func TestCoralTakenOutKeepsTheAutoRankingPoint(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	score := &Score{SeasonKey: season.Key, Level: "REGIONAL", Robots: 3}
	for slot := 1; slot <= 3; slot++ {
		score.Occupy(season.Group("leave"), slot, "left", "")
	}
	score.Occupy(season.Group("coralL2"), 3, "coral", "auto")
	outcome := Evaluate(season, "REGIONAL", score, nil)
	assert.Equal(t, 13, outcome.Total)

	score.Occupy(season.Group("coralL2"), 3, "", "")
	outcome = Evaluate(season, "REGIONAL", score, nil)
	assert.Equal(t, 9, outcome.Total)
	assert.Equal(t, 0, outcome.Categories["coral"])
	assert.Equal(t, []string{"autoRp"}, outcome.RpEarned)

	score.Occupy(season.Group("coralL2"), 3, "coral", "auto")
	outcome = Evaluate(season, "REGIONAL", score, nil)
	assert.Equal(t, 13, outcome.Total)
}

// A slot group with one slot per robot makes "one credit per robot" impossible to violate, instead
// of hoping a numeric ceiling holds.
func TestOneRobotGetsOneEndgameCredit(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	score := &Score{SeasonKey: season.Key, Level: "REGIONAL", Robots: 3}
	score.Occupy(season.Group("endgame"), 1, "deep", "")
	score.Occupy(season.Group("endgame"), 1, "park", "")
	outcome := Evaluate(season, "REGIONAL", score, nil)
	assert.Equal(t, 2, outcome.Categories["barge"])
	assert.Empty(t, outcome.RpEarned)
}

func TestTheFoulGoesToWhoCommittedItAndPaysWhoTookIt(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	own := &Score{SeasonKey: season.Key, Level: "REGIONAL", Robots: 3}
	own.SetAction("minorFoul", "", 2)
	opponent := &Score{SeasonKey: season.Key, Level: "REGIONAL", Robots: 3}
	opponent.SetAction("majorFoul", "", 1)

	outcome := Evaluate(season, "REGIONAL", own, opponent)
	assert.Equal(t, 6, outcome.Total)
	assert.Equal(t, 4, outcome.Categories[CategoryFoulDone])
	assert.Equal(t, 6, outcome.Categories[CategoryFoulTaken])
	assert.Equal(t, []int{0, 0, 0, 0}, outcome.Sort)

	other := Evaluate(season, "REGIONAL", opponent, own)
	assert.Equal(t, 4, other.Total)
	assert.Equal(t, 6, other.Categories[CategoryFoulDone])
}

func TestTheLevelDecidesTheThreshold(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	fill := func(level string) *MatchOutcome {
		score := &Score{SeasonKey: season.Key, Level: level, Robots: 3}
		score.SetAction("coralL1", "teleop", 5)
		for _, group := range []string{"coralL2", "coralL3", "coralL4"} {
			for slot := 1; slot <= 5; slot++ {
				score.Occupy(season.Group(group), slot, "coral", "teleop")
			}
		}
		return Evaluate(season, level, score, nil)
	}
	assert.Contains(t, fill("REGIONAL").RpEarned, "coralRp")
	assert.NotContains(t, fill("CHAMPIONSHIP").RpEarned, "coralRp")
}

// opp is the only operator that crosses to the other alliance, and it may only read the tally. The
// validator refuses anything else, which is what keeps the evaluator total.
func TestOppOnlyReadsTheTally(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	err := season.walkExpr(Expr(`["opp",["flag","coop"]]`), exprScope{label: "teste", allowFlags: true})
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "opp only reads the tally")
	}
	assert.Nil(t, season.walkExpr(Expr(`["opp",["n","algaeProcessor"]]`), exprScope{label: "teste"}))
}

func TestTheValidatorNamesWhatIsWrong(t *testing.T) {
	season := SeasonByKey("frc-2025-reefscape")
	cases := map[string]string{
		`["cat","nao-existe"]`:         "unknown category",
		`["thr","nao-existe"]`:         "unknown threshold",
		`["n","nao-existe"]`:           "unknown action or slot group",
		`["n","coralL2","nao-existe"]`: "unknown period",
		`["quicksort",1,2]`:            "unknown operator",
		`["rp","autoRp"]`:              "rp is only legal in a tiebreaker",
	}
	for expr, reason := range cases {
		err := season.walkExpr(Expr(expr), exprScope{label: "teste", allowFlags: true})
		if assert.NotNil(t, err, expr) {
			assert.Contains(t, err.Error(), reason, expr)
		}
	}
}

func TestDivRoundsTowardsMinusInfinity(t *testing.T) {
	assert.Equal(t, 3, floorDiv(7, 2))
	assert.Equal(t, -4, floorDiv(-7, 2))
	assert.Equal(t, -4, floorDiv(7, -2))
	assert.Equal(t, 3, floorDiv(-7, -2))
}

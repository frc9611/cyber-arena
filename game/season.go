package game

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed seasons/*.json
var seasonFiles embed.FS

const (
	SeasonSchema        = 2
	FormatAlliance      = "alliance"
	FormatSheet         = "sheet"
	LegacySeasonKey     = "legacy-generic"
	UnitCount           = "count"
	UnitTiered          = "tiered"
	UnitToggle          = "toggle"
	UnitChoice          = "choice"
	UnitFreeValue       = "freeValue"
	KindFoul            = "foul"
	KindToggle          = "toggle"
	KindAdjustment      = "adjustment"
	CategoryFoulDone    = "foulCommitted"
	CategoryFoulTaken   = "foulReceived"
	IntoMatchScore      = "matchScore"
	IntoCoopertition    = "coopertition"
	ExclusiveBySlot     = "slot"
	ExclusiveBySlotTime = "slotPeriod"
	SlotsRobots         = "robots"
	SlotsUnbounded      = "unbounded"
	AnyLevel            = "*"
	MaxOperators        = 32
)

type Season struct {
	Schema                int              `json:"schema"`
	Key                   string           `json:"key"`
	Revision              int              `json:"revision"`
	RevisionNote          string           `json:"revisionNote,omitempty"`
	Program               string           `json:"program"`
	Name                  string           `json:"name"`
	Year                  int              `json:"year"`
	Format                string           `json:"format"`
	RobotsPerAlliance     int              `json:"robotsPerAlliance,omitempty"`
	Periods               []SeasonPeriod   `json:"periods"`
	EventLevels           []SeasonLevel    `json:"eventLevels"`
	Thresholds            map[string]Bands `json:"thresholds,omitempty"`
	ThresholdsOverridable bool             `json:"thresholdsOverridable,omitempty"`
	Categories            []SeasonCategory `json:"categories"`
	SlotGroups            []SlotGroup      `json:"slotGroups,omitempty"`
	Actions               []SeasonAction   `json:"actions,omitempty"`
	Flags                 []SeasonFlag     `json:"flags,omitempty"`
	MatchRandom           []SeasonDraw     `json:"matchRandom,omitempty"`
	Field                 *SeasonField     `json:"field,omitempty"`
	Ranking               SeasonRanking    `json:"ranking"`
	MatchTiebreakers      []SeasonSort     `json:"matchTiebreakers,omitempty"`
	Sheet                 *SeasonSheet     `json:"sheet,omitempty"`
	Tests                 []SeasonTest     `json:"tests"`

	byAction   map[string]*SeasonAction
	byGroup    map[string]*SlotGroup
	byCategory map[string]*SeasonCategory
	byFlag     map[string]*SeasonFlag
	byPeriod   map[string]*SeasonPeriod
}

type Bands map[string]int

type SeasonPeriod struct {
	ID          string          `json:"id"`
	Label       string          `json:"label"`
	DurationSec json.RawMessage `json:"durationSec"`
	WarningSec  int             `json:"warningSec,omitempty"`
	Scoring     bool            `json:"scoring"`
}

type SeasonLevel struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type SeasonCategory struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Kind        string `json:"kind,omitempty"`
	RevealAtEnd bool   `json:"revealAtEnd,omitempty"`
}

type SlotGroup struct {
	ID          string          `json:"id"`
	Label       string          `json:"label"`
	Category    string          `json:"category"`
	Help        string          `json:"help,omitempty"`
	Layout      string          `json:"layout,omitempty"`
	Slots       json.RawMessage `json:"slots"`
	ExclusiveBy string          `json:"exclusiveBy"`
	Provenance  bool            `json:"provenance,omitempty"`
	TracksEver  bool            `json:"tracksEver,omitempty"`
	MaxOccupied map[string]Expr `json:"maxOccupied,omitempty"`
	Options     []SlotOption    `json:"options"`
}

type SlotOption struct {
	ID     string          `json:"id"`
	Label  string          `json:"label"`
	Short  string          `json:"short,omitempty"`
	Points json.RawMessage `json:"points"`
	Period string          `json:"period,omitempty"`
}

type SeasonAction struct {
	ID              string          `json:"id"`
	Label           string          `json:"label"`
	Category        string          `json:"category,omitempty"`
	Periods         []string        `json:"periods,omitempty"`
	Points          json.RawMessage `json:"points,omitempty"`
	Unit            string          `json:"unit"`
	Kind            string          `json:"kind,omitempty"`
	Max             int             `json:"max,omitempty"`
	CreditsOpponent bool            `json:"creditsOpponent,omitempty"`
	TracksEver      bool            `json:"tracksEver,omitempty"`
	RemovalOrder    []string        `json:"removalOrder,omitempty"`
	Group           string          `json:"group,omitempty"`
	Order           int             `json:"order,omitempty"`
	Color           string          `json:"color,omitempty"`
	Help            string          `json:"help,omitempty"`
	Hidden          bool            `json:"hidden,omitempty"`
	Confirm         bool            `json:"confirm,omitempty"`
	Derived         Expr            `json:"derived,omitempty"`
	Tiers           map[string]int  `json:"tiers,omitempty"`
	Choices         []SlotOption    `json:"choices,omitempty"`
}

type SeasonFlag struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Points   int    `json:"points"`
	Into     string `json:"into,omitempty"`
	QualOnly bool   `json:"qualOnly,omitempty"`
	When     Expr   `json:"when"`
}

type SeasonDraw struct {
	ID     string        `json:"id"`
	Label  string        `json:"label"`
	Scope  string        `json:"scope"`
	Values []SeasonLevel `json:"values"`
}

type SeasonField struct {
	Family string          `json:"family"`
	Params json.RawMessage `json:"params,omitempty"`
}

type SeasonRanking struct {
	Mode          string           `json:"mode"`
	RsDecimals    int              `json:"rsDecimals,omitempty"`
	Result        *SeasonResult    `json:"result,omitempty"`
	Surrogate     *SeasonStanding  `json:"surrogate,omitempty"`
	Disqualified  *SeasonStanding  `json:"disqualified,omitempty"`
	RankingPoints []SeasonRp       `json:"rankingPoints,omitempty"`
	Voids         []SeasonRpChange `json:"voids,omitempty"`
	Grants        []SeasonRpChange `json:"grants,omitempty"`
	Tiebreakers   []SeasonSort     `json:"tiebreakers,omitempty"`
	Rounds        int              `json:"rounds,omitempty"`
	Tiebreak      string           `json:"tiebreak,omitempty"`
}

type SeasonResult struct {
	Win  int `json:"win"`
	Tie  int `json:"tie"`
	Loss int `json:"loss"`
}

type SeasonStanding struct {
	Rp           int  `json:"rp"`
	SortZero     bool `json:"sortZero"`
	CountsPlayed bool `json:"countsPlayed"`
}

type SeasonRp struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Points int    `json:"points"`
	When   Expr   `json:"when"`
}

type SeasonRpChange struct {
	Rp   string `json:"rp"`
	When Expr   `json:"when"`
}

type SeasonSort struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Expr      Expr   `json:"expr"`
	Agg       string `json:"agg,omitempty"`
	Direction string `json:"direction,omitempty"`
}

type SeasonSheet struct {
	TeamsPerTable int          `json:"teamsPerTable"`
	Groups        []SheetGroup `json:"groups"`
	Rounds        int          `json:"rounds,omitempty"`
}

type SheetGroup struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Entries    []string `json:"entries"`
	OneOf      []string `json:"oneOf,omitempty"`
	VoidsGroup string   `json:"voidsGroup,omitempty"`
	VoidsMatch bool     `json:"voidsMatch,omitempty"`
}

type SeasonTest struct {
	Name     string            `json:"name"`
	Level    string            `json:"level"`
	Robots   int               `json:"robots,omitempty"`
	Draw     map[string]string `json:"draw,omitempty"`
	State    *TestState        `json:"state,omitempty"`
	Steps    []TestStep        `json:"steps,omitempty"`
	Opponent *TestState        `json:"opponent,omitempty"`
	Expect   TestExpect        `json:"expect"`
}

type TestState struct {
	Actions map[string]int                        `json:"actions,omitempty"`
	Adjust  map[string]int                        `json:"adjust,omitempty"`
	Slots   map[string]map[string]json.RawMessage `json:"slots,omitempty"`
}

type TestStep struct {
	Group  string  `json:"group,omitempty"`
	Slot   int     `json:"slot,omitempty"`
	Option *string `json:"option,omitempty"`
	Period string  `json:"period,omitempty"`
	Action string  `json:"action,omitempty"`
	Delta  int     `json:"delta,omitempty"`
}

type TestExpect struct {
	Total      *int           `json:"total,omitempty"`
	Categories map[string]int `json:"categories,omitempty"`
	Flags      []string       `json:"flags,omitempty"`
	FlagPts    map[string]int `json:"flagPts,omitempty"`
	RpEarned   []string       `json:"rpEarned,omitempty"`
	RpEligible []string       `json:"rpEligible,omitempty"`
	Sort       []int          `json:"sort,omitempty"`
}

var seasons = map[string]*Season{}
var seasonOrder []string

func init() {
	entries, err := seasonFiles.ReadDir("seasons")
	if err != nil {
		panic(fmt.Sprintf("season pack unreadable: %v", err))
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		body, err := seasonFiles.ReadFile("seasons/" + entry.Name())
		if err != nil {
			panic(fmt.Sprintf("season %s unreadable: %v", entry.Name(), err))
		}
		season, err := ParseSeason(body)
		if err != nil {
			panic(fmt.Sprintf("season %s invalid: %v", entry.Name(), err))
		}
		seasons[season.Key] = season
		seasonOrder = append(seasonOrder, season.Key)
	}
	sort.Strings(seasonOrder)
}

func ParseSeason(body []byte) (*Season, error) {
	season := new(Season)
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(season); err != nil {
		return nil, err
	}
	season.index()
	if err := season.Validate(); err != nil {
		return nil, err
	}
	return season, nil
}

func (season *Season) index() {
	season.byAction = map[string]*SeasonAction{}
	season.byGroup = map[string]*SlotGroup{}
	season.byCategory = map[string]*SeasonCategory{}
	season.byFlag = map[string]*SeasonFlag{}
	season.byPeriod = map[string]*SeasonPeriod{}
	for i := range season.Actions {
		season.byAction[season.Actions[i].ID] = &season.Actions[i]
	}
	for i := range season.SlotGroups {
		season.byGroup[season.SlotGroups[i].ID] = &season.SlotGroups[i]
	}
	for i := range season.Categories {
		season.byCategory[season.Categories[i].ID] = &season.Categories[i]
	}
	for i := range season.Flags {
		season.byFlag[season.Flags[i].ID] = &season.Flags[i]
	}
	for i := range season.Periods {
		season.byPeriod[season.Periods[i].ID] = &season.Periods[i]
	}
}

func SeasonByKey(key string) *Season {
	if key == "" {
		return seasons[LegacySeasonKey]
	}
	return seasons[key]
}

func SeasonKeys() []string {
	answer := make([]string, len(seasonOrder))
	copy(answer, seasonOrder)
	return answer
}

func AllSeasons() []*Season {
	answer := make([]*Season, 0, len(seasonOrder))
	for _, key := range seasonOrder {
		answer = append(answer, seasons[key])
	}
	return answer
}

func RegisterSeason(season *Season) {
	if _, known := seasons[season.Key]; !known {
		seasonOrder = append(seasonOrder, season.Key)
		sort.Strings(seasonOrder)
	}
	seasons[season.Key] = season
}

func (season *Season) Action(id string) *SeasonAction { return season.byAction[id] }
func (season *Season) Group(id string) *SlotGroup     { return season.byGroup[id] }
func (season *Season) Period(id string) *SeasonPeriod { return season.byPeriod[id] }
func (season *Season) Flag(id string) *SeasonFlag     { return season.byFlag[id] }

func (season *Season) HasResult() bool {
	return season.Ranking.Result != nil
}

func (season *Season) KnowsLevel(level string) bool {
	for _, candidate := range season.EventLevels {
		if candidate.ID == level {
			return true
		}
	}
	return false
}

func (season *Season) DefaultLevel() string {
	if len(season.EventLevels) == 0 {
		return ""
	}
	return season.EventLevels[0].ID
}

func (season *Season) Threshold(id, level string) int {
	return season.Thresholds[id].at(level)
}

func (bands Bands) at(level string) int {
	if bands == nil {
		return 0
	}
	if value, ok := bands[level]; ok {
		return value
	}
	return bands[AnyLevel]
}

func (season *Season) ScoringPeriods() []string {
	answer := make([]string, 0, len(season.Periods))
	for _, period := range season.Periods {
		if period.Scoring {
			answer = append(answer, period.ID)
		}
	}
	return answer
}

func (period *SeasonPeriod) Duration(level string) int {
	if len(period.DurationSec) == 0 {
		return 0
	}
	var scalar int
	if err := json.Unmarshal(period.DurationSec, &scalar); err == nil {
		return scalar
	}
	var bands Bands
	if err := json.Unmarshal(period.DurationSec, &bands); err == nil {
		return bands.at(level)
	}
	return 0
}

func (group *SlotGroup) SlotCount(robots int) int {
	var scalar int
	if err := json.Unmarshal(group.Slots, &scalar); err == nil {
		return scalar
	}
	var word string
	if err := json.Unmarshal(group.Slots, &word); err == nil {
		switch word {
		case SlotsRobots:
			return robots
		case SlotsUnbounded:
			return 0
		}
	}
	return 0
}

func (group *SlotGroup) Unbounded() bool {
	var word string
	return json.Unmarshal(group.Slots, &word) == nil && word == SlotsUnbounded
}

func (group *SlotGroup) Option(id string) *SlotOption {
	for i := range group.Options {
		if group.Options[i].ID == id {
			return &group.Options[i]
		}
	}
	return nil
}

func pointsIn(raw json.RawMessage, period string) int {
	if len(raw) == 0 {
		return 0
	}
	var scalar int
	if err := json.Unmarshal(raw, &scalar); err == nil {
		return scalar
	}
	var byPeriod map[string]int
	if err := json.Unmarshal(raw, &byPeriod); err == nil {
		if value, ok := byPeriod[period]; ok {
			return value
		}
		if value, ok := byPeriod[AnyLevel]; ok {
			return value
		}
	}
	return 0
}

func (option *SlotOption) PointsIn(period string) int {
	return pointsIn(option.Points, period)
}

func (action *SeasonAction) PointsIn(period string) int {
	return pointsIn(action.Points, period)
}

func (action *SeasonAction) PeriodsOrAny() []string {
	if len(action.Periods) > 0 {
		return action.Periods
	}
	return []string{""}
}

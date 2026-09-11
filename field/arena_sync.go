package field

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Team254/cheesy-arena-lite/config"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/partner"
	"github.com/Team254/cheesy-arena-lite/version"
)

const (
	arenaSyncMinSeconds     = 5
	arenaSyncBackoffSeconds = 30
	arenaSyncBackoffMax     = 300
)

type arenaSyncConfig struct {
	MasterUrl  string
	Token      string
	ClientUid  string
	ClientName string
	PublicUrl  string
	EventSlug  string
	EventName  string
	InstanceId int64
	Generation int
	VenueSlot  int
	VenueLabel string
	Mode       string
	Enabled    bool
	Seconds    int
}

var arenaSettingsMutex sync.Mutex

func (arena *Arena) arenaSyncSettings() *arenaSyncConfig {
	return arena.arenaSync.Load()
}

func (arena *Arena) ReloadArenaMasterClient() {
	settings := arena.EventSettings
	seconds := settings.ArenaSyncSeconds
	if seconds == 0 && arena.Config != nil {
		seconds = arena.Config.SyncSeconds
	}
	if seconds > 0 && seconds < arenaSyncMinSeconds {
		seconds = arenaSyncMinSeconds
	}
	masterUrl := settings.ArenaMasterUrl
	token := settings.ArenaToken
	if arena.Config != nil {
		if arena.Config.MasterUrl != "" {
			masterUrl = arena.Config.MasterUrl
		}
		if arena.Config.TokenFromEnv {
			token = arena.Config.Token
		}
	}
	next := &arenaSyncConfig{
		MasterUrl:  masterUrl,
		Token:      token,
		ClientUid:  settings.ArenaClientUid,
		ClientName: settings.ArenaClientName,
		PublicUrl:  settings.ArenaPublicUrl,
		EventSlug:  settings.ArenaEventSlug,
		EventName:  settings.Name,
		InstanceId: settings.ArenaInstanceId,
		Generation: settings.ArenaSyncGeneration,
		VenueSlot:  settings.ArenaVenueSlot,
		VenueLabel: settings.ArenaVenueLabel,
		Mode:       arena.Mode(),
		Enabled:    settings.ArenaSyncEnabled && settings.ArenaConfirmedAt != "" && settings.ArenaCheckedAt != "",
		Seconds:    seconds,
	}
	arena.arenaSync.Store(next)
	arena.ArenaMasterClient = partner.NewArenaMasterClient(masterUrl, token, version.Version)
}

func (arena *Arena) WakeArenaSync() {
	if arena.arenaSyncWake == nil {
		return
	}
	select {
	case arena.arenaSyncWake <- struct{}{}:
	default:
	}
}

func (arena *Arena) RunArenaSync() {
	backoff := 0
	for {
		settings := arena.arenaSyncSettings()
		wait := arenaSyncBackoffSeconds
		if settings != nil && settings.Seconds > 0 {
			wait = settings.Seconds
		}
		if backoff > wait {
			wait = backoff
		}
		select {
		case <-arena.arenaSyncWake:
		case <-time.After(time.Duration(wait) * time.Second):
		}

		settings = arena.arenaSyncSettings()
		if settings == nil || !settings.Enabled || settings.Token == "" || settings.MasterUrl == "" {
			backoff = 0
			continue
		}
		if settings.Mode == config.ModeStandalone {
			backoff = 0
			continue
		}
		if !arena.arenaSyncing.CompareAndSwap(false, true) {
			continue
		}
		retry := arena.syncOnce(settings)
		arena.arenaSyncing.Store(false)
		if retry {
			if backoff == 0 {
				backoff = arenaSyncBackoffSeconds
			} else if backoff < arenaSyncBackoffMax {
				backoff *= 2
			}
		} else {
			backoff = 0
		}
	}
}

func (arena *Arena) SyncNow() (*partner.ArenaSyncResult, error) {
	settings := arena.arenaSyncSettings()
	if settings == nil || settings.Token == "" || settings.MasterUrl == "" {
		return nil, fmt.Errorf("Esta arena ainda não foi conectada ao Arena Master.")
	}
	snapshot, err := arena.BuildArenaSnapshot(settings)
	if err != nil {
		return nil, err
	}
	result, err := arena.ArenaMasterClient.PushSnapshot(snapshot)
	arena.recordSyncOutcome(result, err)
	return result, err
}

func (arena *Arena) syncOnce(settings *arenaSyncConfig) bool {
	snapshot, err := arena.BuildArenaSnapshot(settings)
	if err != nil {
		log.Printf("Arena Master: não consegui ler o banco para sincronizar: %v", err)
		return true
	}
	result, err := arena.ArenaMasterClient.PushSnapshot(snapshot)
	arena.recordSyncOutcome(result, err)
	if err == nil {
		return false
	}
	code, message := partner.DescribeArenaError(err)
	log.Printf("Arena Master (%s): %s", code, message)
	if arenaErr, ok := err.(*partner.ArenaError); ok {
		return arenaErr.Retry
	}
	return true
}

func (arena *Arena) recordSyncOutcome(result *partner.ArenaSyncResult, err error) {
	now := time.Now().Format(time.RFC3339)
	arena.saveArenaProgress(func(settings *model.EventSettings) {
		if err != nil {
			code, message := partner.DescribeArenaError(err)
			settings.ArenaLastErrorCode = code
			settings.ArenaLastError = message
			settings.ArenaLastErrorAt = now
			return
		}
		settings.ArenaLastError = ""
		settings.ArenaLastErrorCode = ""
		settings.ArenaLastErrorAt = ""
		settings.ArenaLastSyncAt = now
		if result != nil && !result.Unchanged {
			settings.ArenaRevision = result.InstanceRevision
			settings.ArenaEventRevision = result.EventRevision
		}
	})
}

func (arena *Arena) saveArenaProgress(mutate func(*model.EventSettings)) {
	arenaSettingsMutex.Lock()
	defer arenaSettingsMutex.Unlock()
	fresh, err := arena.Database.GetEventSettings()
	if err != nil {
		log.Printf("Arena Master: não consegui reler as configurações: %v", err)
		return
	}
	mutate(fresh)
	if err := arena.Database.UpdateEventSettings(fresh); err != nil {
		log.Printf("Arena Master: não consegui gravar as configurações: %v", err)
		return
	}
	*arena.EventSettings = *fresh
	arena.ReloadArenaMasterClient()
}

func (arena *Arena) BuildArenaSnapshot(settings *arenaSyncConfig) (*partner.ArenaSnapshot, error) {
	teams, err := arena.Database.GetAllTeams()
	if err != nil {
		return nil, err
	}
	snapshotTeams := make([]partner.ArenaBootstrapTeam, 0, len(teams))
	for _, team := range teams {
		if team.Id <= 0 {
			continue
		}
		snapshotTeams = append(snapshotTeams, partner.ArenaBootstrapTeam{
			Number:     team.Id,
			Name:       team.Name,
			Nickname:   team.Nickname,
			City:       team.City,
			StateProv:  team.StateProv,
			Country:    team.Country,
			RookieYear: team.RookieYear,
			RobotName:  team.RobotName,
		})
	}

	snapshotMatches := make([]partner.ArenaSnapshotMatch, 0)
	for _, matchType := range []string{"practice", "qualification", "elimination"} {
		matches, err := arena.Database.GetMatchesByType(matchType)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			snapshotMatches = append(snapshotMatches, arena.describeMatch(settings, match))
		}
	}

	rankings, err := arena.Database.GetAllRankings()
	if err != nil {
		return nil, err
	}
	snapshotRankings := make([]partner.ArenaSnapshotRanking, 0, len(rankings))
	for _, ranking := range rankings {
		snapshotRankings = append(snapshotRankings, partner.ArenaSnapshotRanking{
			TeamNumber:    ranking.TeamId,
			Rank:          ranking.Rank,
			RankingPoints: ranking.RankingPoints,
			Wins:          ranking.Wins,
			Losses:        ranking.Losses,
			Ties:          ranking.Ties,
			Played:        ranking.Played,
		})
	}

	alliances, err := arena.Database.GetAllAlliances()
	if err != nil {
		return nil, err
	}
	snapshotAlliances := make([]partner.ArenaSnapshotAlliance, 0, len(alliances))
	for _, alliance := range alliances {
		snapshotAlliances = append(snapshotAlliances, partner.ArenaSnapshotAlliance{
			Number:      alliance.Id,
			TeamNumbers: joinTeamIds(alliance.TeamIds...),
		})
	}

	awards, err := arena.Database.GetAllAwards()
	if err != nil {
		return nil, err
	}
	snapshotAwards := make([]partner.ArenaSnapshotAward, 0, len(awards))
	for _, award := range awards {
		entry := partner.ArenaSnapshotAward{
			Type:       awardTypeName(award.Type),
			AwardName:  award.AwardName,
			PersonName: award.PersonName,
		}
		if award.TeamId > 0 {
			team := award.TeamId
			entry.TeamNumber = &team
		}
		snapshotAwards = append(snapshotAwards, entry)
	}

	var snapshotFll *[]partner.ArenaSnapshotFllScore
	if arena.EventSettings.IsFll {
		scores, err := arena.Database.GetAllFllScores()
		if err != nil {
			return nil, err
		}
		rows := make([]partner.ArenaSnapshotFllScore, 0, len(scores)*3)
		for _, score := range scores {
			for index, value := range score.Rounds {
				round := index + 1
				points := value
				rows = append(rows, partner.ArenaSnapshotFllScore{
					TeamNumber: score.TeamId,
					RoundIndex: round,
					Score:      &points,
					Official:   score.OfficialRound == round,
					UpdatedAt:  arenaTime(score.UpdatedAt),
				})
			}
		}
		snapshotFll = &rows
	}

	snapshot := &partner.ArenaSnapshot{
		ClientUid:        settings.ClientUid,
		InstanceId:       settings.InstanceId,
		Generation:       settings.Generation,
		CoversAll:        true,
		LocalEventName:   settings.EventName,
		LocalFingerprint: arena.EventSettings.ArenaLocalFingerprint,
		Teams:            &snapshotTeams,
		Matches:          &snapshotMatches,
		Rankings:         &snapshotRankings,
		Alliances:        &snapshotAlliances,
		Awards:           &snapshotAwards,
		FllScores:        snapshotFll,
	}
	snapshot.ContentHash = arenaContentHash(snapshot)
	return snapshot, nil
}

func (arena *Arena) describeMatch(settings *arenaSyncConfig, match model.Match) partner.ArenaSnapshotMatch {
	entry := partner.ArenaSnapshotMatch{
		MatchKey:    fmt.Sprintf("%d-%s-%d", settings.InstanceId, match.Type, match.Id),
		Type:        match.Type,
		DisplayName: match.DisplayName,
		ScheduledAt: arenaTime(match.Time),
		RedTeams:    joinTeamIds(match.Red1, match.Red2, match.Red3),
		BlueTeams:   joinTeamIds(match.Blue1, match.Blue2, match.Blue3),
		Status:      string(match.Status),
	}
	if settings.VenueSlot > 0 {
		slot := settings.VenueSlot
		entry.VenueSlot = &slot
	}
	if match.Status == game.MatchNotPlayed {
		return entry
	}
	entry.CommittedAt = arenaTime(match.ScoreCommittedAt)
	result, err := arena.Database.GetMatchResultForMatch(match.Id)
	if err != nil || result == nil {
		return entry
	}
	red := result.RedScoreSummary().Score
	blue := result.BlueScoreSummary().Score
	entry.RedScore = &red
	entry.BlueScore = &blue
	return entry
}

func joinTeamIds(ids ...int) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			parts = append(parts, strconv.Itoa(id))
		}
	}
	return strings.Join(parts, ",")
}

func awardTypeName(awardType model.AwardType) string {
	switch awardType {
	case model.FinalistAward:
		return "finalist"
	case model.WinnerAward:
		return "winner"
	default:
		return "judged"
	}
}

func arenaTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02T15:04:05")
}

func arenaContentHash(snapshot *partner.ArenaSnapshot) string {
	payload := struct {
		Generation int
		CoversAll  bool
		Teams      *[]partner.ArenaBootstrapTeam
		Matches    *[]partner.ArenaSnapshotMatch
		Rankings   *[]partner.ArenaSnapshotRanking
		Alliances  *[]partner.ArenaSnapshotAlliance
		Awards     *[]partner.ArenaSnapshotAward
		FllScores  *[]partner.ArenaSnapshotFllScore
	}{
		snapshot.Generation, snapshot.CoversAll, snapshot.Teams, snapshot.Matches,
		snapshot.Rankings, snapshot.Alliances, snapshot.Awards, snapshot.FllScores,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func (arena *Arena) BumpArenaGeneration() {
	arena.saveArenaProgress(func(settings *model.EventSettings) {
		settings.ArenaSyncGeneration++
	})
}

func osHostname() (string, error) {
	return os.Hostname()
}

func (arena *Arena) ForgetArenaConnectionAfterRestore() {
	host := ""
	if name, err := osHostname(); err == nil {
		host = name
	}
	arena.saveArenaProgress(func(settings *model.EventSettings) {
		settings.ArenaSyncEnabled = false
		settings.ArenaConfirmedAt = ""
		settings.ArenaCheckedAt = ""
		settings.ArenaSyncGeneration++
		if settings.ArenaClientHost != "" && host != "" && settings.ArenaClientHost != host {
			settings.ArenaClientUid = ""
			settings.ArenaClientHost = host
		}
	})
}

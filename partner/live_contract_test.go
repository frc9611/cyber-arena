package partner

import (
	"os"
	"testing"
	"time"
)

func TestLiveContract(t *testing.T) {
	base := os.Getenv("ARENA_LIVE_URL")
	token := os.Getenv("ARENA_LIVE_TOKEN")
	uid := os.Getenv("ARENA_LIVE_UID")
	if base == "" || token == "" {
		t.Skip("sem servidor vivo")
	}
	client := NewArenaMasterClient(base, token, "1.4.0-vernum")

	boot, err := client.Bootstrap(uid)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Logf("evento %q elimType=%s isFll=%v locais=%d %s equipes=%d",
		boot.Event.Name, boot.Event.ElimType, boot.Event.IsFll,
		boot.Event.Venue.Count, boot.Event.Venue.KindLabelPlural, len(boot.Teams))

	if _, err := client.Heartbeat(&ArenaHeartbeat{
		ClientUid: uid, ClientName: "teste-contrato", ClientVersion: "1.4.0-vernum",
		Mode: "LOCAL", EventSlug: boot.Event.Slug,
	}); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	teams := []ArenaBootstrapTeam{{Number: 9611, Nickname: "CyberRain"}, {Number: 2530, Nickname: "Tuiuiu"}}
	score := 58
	other := 61
	committed := time.Date(2026, 9, 11, 14, 29, 0, 0, time.UTC).Format("2006-01-02T15:04:05")
	matches := []ArenaSnapshotMatch{{
		MatchKey: "9-qualification-1", Type: "qualification", DisplayName: "1",
		ScheduledAt: committed, RedTeams: "9611", BlueTeams: "2530",
		RedScore: &score, BlueScore: &other, Status: "B", CommittedAt: committed,
	}}
	rankings := []ArenaSnapshotRanking{{TeamNumber: 9611, Rank: 1, RankingPoints: 3, Wins: 1, Played: 1}}
	alliances := []ArenaSnapshotAlliance{{Number: 1, TeamNumbers: "9611,2530"}}
	team := 9611
	awards := []ArenaSnapshotAward{{Type: "winner", AwardName: "Vencedores", TeamNumber: &team}}

	result, err := client.PushSnapshot(&ArenaSnapshot{
		ClientUid: uid, CoversAll: true, Generation: 1, ContentHash: "contrato-1",
		LocalEventName: "Teste de contrato",
		Teams:          &teams, Matches: &matches, Rankings: &rankings,
		Alliances: &alliances, Awards: &awards,
	})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	t.Logf("aceito: equipes=%d partidas=%d rankings=%d aliancas=%d premios=%d rev=%d ignorados=%d avisos=%v",
		result.Teams, result.Matches, result.Rankings, result.Alliances, result.Awards,
		result.InstanceRevision, len(result.Ignored), result.Notices)

	again, err := client.PushSnapshot(&ArenaSnapshot{
		ClientUid: uid, CoversAll: true, Generation: 1, ContentHash: "contrato-1",
		Teams: &teams, Matches: &matches, Rankings: &rankings, Alliances: &alliances, Awards: &awards,
	})
	if err != nil {
		t.Fatalf("sync repetido: %v", err)
	}
	if !again.Unchanged {
		t.Fatalf("esperava 204 no mesmo hash, veio revisao %d", again.InstanceRevision)
	}
	t.Logf("mesmo hash: 204, sem reescrita")
}

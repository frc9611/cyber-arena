package partner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ArenaTokenHeader   = "X-Arena-Token"
	ArenaTokenScheme   = "ak_"
	arenaRequestExpiry = 20 * time.Second
)

type ArenaVenueSlot struct {
	Slot   int    `json:"slot"`
	Label  string `json:"label"`
	Hint   string `json:"hint"`
	Active bool   `json:"active"`
}

type ArenaVenue struct {
	Kind              string           `json:"kind"`
	KindLabel         string           `json:"kindLabel"`
	KindLabelPlural   string           `json:"kindLabelPlural"`
	ScheduleMode      string           `json:"scheduleMode"`
	ScheduleModeLabel string           `json:"scheduleModeLabel"`
	Count             int              `json:"count"`
	Slots             []ArenaVenueSlot `json:"slots"`
}

type ArenaCounts struct {
	Teams     int64 `json:"teams"`
	Matches   int64 `json:"matches"`
	Rankings  int64 `json:"rankings"`
	Alliances int64 `json:"alliances"`
	Awards    int64 `json:"awards"`
	FllScores int64 `json:"fllScores"`
}

type ArenaBootstrapTeam struct {
	Number     int    `json:"number"`
	Name       string `json:"name"`
	Nickname   string `json:"nickname"`
	City       string `json:"city"`
	StateProv  string `json:"stateProv"`
	Country    string `json:"country"`
	RookieYear int    `json:"rookieYear"`
	RobotName  string `json:"robotName"`
}

type ArenaBootstrap struct {
	ApiVersion int `json:"apiVersion"`
	Server     struct {
		Instance      string `json:"instance"`
		ServerTime    string `json:"serverTime"`
		VernumVersion string `json:"vernumVersion"`
	} `json:"server"`
	Instance struct {
		InstanceId           int64  `json:"instanceId"`
		ClientName           string `json:"clientName"`
		Kind                 string `json:"kind"`
		KindLabel            string `json:"kindLabel"`
		VenueSlot            *int   `json:"venueSlot"`
		VenueLabel           string `json:"venueLabel"`
		Claimed              bool   `json:"claimed"`
		ClaimedByThisMachine bool   `json:"claimedByThisMachine"`
		ClaimedAt            string `json:"claimedAt"`
		LastSeenAt           string `json:"lastSeenAt"`
		DeleteSuspended      bool   `json:"deleteSuspended"`
	} `json:"instance"`
	Event struct {
		EventId             int64       `json:"eventId"`
		Slug                string      `json:"slug"`
		Name                string      `json:"name"`
		Location            string      `json:"location"`
		TournamentType      string      `json:"tournamentType"`
		TournamentTypeLabel string      `json:"tournamentTypeLabel"`
		ElimType            string      `json:"elimType"`
		IsFll               bool        `json:"isFll"`
		TeamsPerAlliance    int         `json:"teamsPerAlliance"`
		NumElimAlliances    int         `json:"numElimAlliances"`
		StartsAt            string      `json:"startsAt"`
		EndsAt              string      `json:"endsAt"`
		Published           bool        `json:"published"`
		Venue               ArenaVenue  `json:"venue"`
		Counts              ArenaCounts `json:"counts"`
		CountsFromHere      ArenaCounts `json:"countsFromHere"`
	} `json:"event"`
	Teams []ArenaBootstrapTeam `json:"teams"`
}

type ArenaHeartbeat struct {
	ClientUid     string `json:"clientUid"`
	ClientName    string `json:"clientName"`
	ClientVersion string `json:"clientVersion"`
	Mode          string `json:"mode"`
	PublicUrl     string `json:"publicUrl"`
	EventSlug     string `json:"eventSlug"`
	VenueSlot     *int   `json:"venueSlot,omitempty"`
	VenueLabel    string `json:"venueLabel,omitempty"`
	Revision      int64  `json:"revision"`
	ContentHash   string `json:"contentHash"`
	Takeover      bool   `json:"takeover"`
}

type ArenaHeartbeatResult struct {
	InstanceId      int64    `json:"instanceId"`
	ClientName      string   `json:"clientName"`
	VenueSlot       *int     `json:"venueSlot"`
	VenueLabel      string   `json:"venueLabel"`
	VenueKindLabel  string   `json:"venueKindLabel"`
	ClaimedNow      bool     `json:"claimedNow"`
	ServerTime      string   `json:"serverTime"`
	ServerInstance  string   `json:"serverInstance"`
	DeleteSuspended bool     `json:"deleteSuspended"`
	Anchor          bool     `json:"anchor"`
	Notices         []string `json:"notices"`
}

type ArenaSnapshotMatch struct {
	MatchKey    string `json:"matchKey"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	ScheduledAt string `json:"scheduledAt,omitempty"`
	VenueSlot   *int   `json:"venueSlot,omitempty"`
	RedTeams    string `json:"redTeams"`
	BlueTeams   string `json:"blueTeams"`
	RedScore    *int   `json:"redScore,omitempty"`
	BlueScore   *int   `json:"blueScore,omitempty"`
	Status      string `json:"status,omitempty"`
	CommittedAt string `json:"committedAt,omitempty"`
}

type ArenaSnapshotRanking struct {
	TeamNumber    int    `json:"teamNumber"`
	Rank          int    `json:"rank"`
	RankingPoints int    `json:"rankingPoints"`
	Wins          int    `json:"wins"`
	Losses        int    `json:"losses"`
	Ties          int    `json:"ties"`
	Played        int    `json:"played"`
	SortFields    string `json:"sortFields,omitempty"`
}

type ArenaSnapshotAlliance struct {
	Number      int    `json:"number"`
	TeamNumbers string `json:"teamNumbers"`
}

type ArenaSnapshotAward struct {
	Type       string `json:"type"`
	AwardName  string `json:"awardName"`
	TeamNumber *int   `json:"teamNumber,omitempty"`
	PersonName string `json:"personName,omitempty"`
}

type ArenaSnapshotFllScore struct {
	TeamNumber int    `json:"teamNumber"`
	RoundIndex int    `json:"roundIndex"`
	Score      *int   `json:"score,omitempty"`
	MatchKey   string `json:"matchKey,omitempty"`
	Official   bool   `json:"official"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

type ArenaSnapshot struct {
	ClientUid        string                   `json:"clientUid"`
	InstanceId       int64                    `json:"instanceId"`
	Generation       int                      `json:"generation"`
	CoversAll        bool                     `json:"coversAll"`
	LocalEventName   string                   `json:"localEventName"`
	LocalFingerprint string                   `json:"localFingerprint"`
	ContentHash      string                   `json:"contentHash"`
	Teams            *[]ArenaBootstrapTeam    `json:"teams"`
	Matches          *[]ArenaSnapshotMatch    `json:"matches"`
	Rankings         *[]ArenaSnapshotRanking  `json:"rankings"`
	Alliances        *[]ArenaSnapshotAlliance `json:"alliances"`
	Awards           *[]ArenaSnapshotAward    `json:"awards"`
	FllScores        *[]ArenaSnapshotFllScore `json:"fllScores"`
}

type ArenaIgnored struct {
	What   string `json:"what"`
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

type ArenaSyncResult struct {
	InstanceRevision int64          `json:"instanceRevision"`
	EventRevision    int64          `json:"eventRevision"`
	Teams            int            `json:"teams"`
	Matches          int            `json:"matches"`
	Rankings         int            `json:"rankings"`
	Alliances        int            `json:"alliances"`
	Awards           int            `json:"awards"`
	FllScores        int            `json:"fllScores"`
	Ignored          []ArenaIgnored `json:"ignored"`
	Notices          []string       `json:"notices"`
	Unchanged        bool           `json:"-"`
}

type ArenaMasterClient struct {
	BaseUrl string
	Token   string
	Version string
	client  *http.Client
}

func NewArenaMasterClient(baseUrl, token, version string) *ArenaMasterClient {
	return &ArenaMasterClient{
		BaseUrl: strings.TrimRight(baseUrl, "/"),
		Token:   token,
		Version: version,
		client:  &http.Client{Timeout: arenaRequestExpiry, CheckRedirect: refuseRedirect},
	}
}

func refuseRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func (client *ArenaMasterClient) Ready() bool {
	return client != nil && client.BaseUrl != "" && client.Token != ""
}

func (client *ArenaMasterClient) Bootstrap(clientUid string) (*ArenaBootstrap, error) {
	path := "/public/arena/instance/bootstrap"
	if clientUid != "" {
		path += "?clientUid=" + url.QueryEscape(clientUid)
	}
	body, err := client.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	bootstrap := new(ArenaBootstrap)
	if err := json.Unmarshal(body, bootstrap); err != nil {
		return nil, &ArenaError{Code: "resposta_ilegivel",
			Message: "O Arena Master respondeu algo que não consegui ler."}
	}
	return bootstrap, nil
}

func (client *ArenaMasterClient) Heartbeat(beat *ArenaHeartbeat) (*ArenaHeartbeatResult, error) {
	body, err := client.do("POST", "/public/arena/instance/heartbeat", beat)
	if err != nil {
		return nil, err
	}
	result := new(ArenaHeartbeatResult)
	if err := json.Unmarshal(body, result); err != nil {
		return nil, &ArenaError{Code: "resposta_ilegivel",
			Message: "O Arena Master respondeu algo que não consegui ler."}
	}
	return result, nil
}

func (client *ArenaMasterClient) PushSnapshot(snapshot *ArenaSnapshot) (*ArenaSyncResult, error) {
	body, err := client.do("POST", "/public/arena/instance/sync", snapshot)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return &ArenaSyncResult{Unchanged: true}, nil
	}
	result := new(ArenaSyncResult)
	if err := json.Unmarshal(body, result); err != nil {
		return nil, &ArenaError{Code: "resposta_ilegivel",
			Message: "O Arena Master respondeu algo que não consegui ler."}
	}
	return result, nil
}

func (client *ArenaMasterClient) RevokeSelf() error {
	_, err := client.do("POST", "/public/arena/instance/revoke-self", nil)
	return err
}

func (client *ArenaMasterClient) do(method, path string, payload interface{}) ([]byte, error) {
	if !client.Ready() {
		return nil, &ArenaError{Code: "nao_configurado",
			Message: "Esta arena ainda não foi conectada ao Arena Master."}
	}
	var reader io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, &ArenaError{Code: "corpo_invalido", Message: "Não consegui montar o envio."}
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, client.BaseUrl+path, reader)
	if err != nil {
		return nil, &ArenaError{Code: "endereco_invalido",
			Message: "O endereço do Arena Master não é um endereço válido."}
	}
	request.Header.Set(ArenaTokenHeader, client.Token)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "cyber-arena/"+client.Version)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.client.Do(request)
	if err != nil {
		return nil, describeTransportError(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode >= 300 {
		return nil, describeHttpError(response.StatusCode, body)
	}
	return body, nil
}

type ArenaError struct {
	Code    string
	Message string
	Status  int
	Retry   bool
}

func (e *ArenaError) Error() string {
	return e.Message
}

func DescribeTransportError(err error) *ArenaError {
	return describeTransportError(err)
}

func describeTransportError(err error) *ArenaError {
	text := err.Error()
	switch {
	case strings.Contains(text, "no such host"):
		return &ArenaError{Code: "dns", Retry: true,
			Message: "Não achei esse endereço na rede. Confira se está escrito certo e se esta máquina tem internet."}
	case strings.Contains(text, "certificate") || strings.Contains(text, "tls"):
		return &ArenaError{Code: "tls",
			Message: "O certificado do servidor não confere. Não vou mandar o token por uma conexão que não posso verificar."}
	case isTimeout(err):
		return &ArenaError{Code: "timeout", Retry: true,
			Message: "O Arena Master demorou demais para responder. Vou tentar de novo sozinho."}
	case strings.Contains(text, "connection refused"):
		return &ArenaError{Code: "recusado", Retry: true,
			Message: "Nada atendeu nesse endereço e nessa porta."}
	default:
		return &ArenaError{Code: "rede", Retry: true,
			Message: "Não consegui falar com o Arena Master: " + text}
	}
}

func isTimeout(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

func describeHttpError(status int, body []byte) *ArenaError {
	message, hasMessage := vernumMessage(body)
	switch status {
	case http.StatusUnauthorized:
		return &ArenaError{Status: status, Code: "nao_autorizado",
			Message: "O servidor pediu login numa rota que deveria ser da instância. " +
				"Confira se o endereço do Arena Master está sem caminho a mais no fim."}
	case http.StatusForbidden:
		if hasMessage {
			return &ArenaError{Status: status, Code: "token_invalido",
				Message: "O Arena Master recusou o token: “" + message + "”"}
		}
		return &ArenaError{Status: status, Code: "token_invalido",
			Message: "O Arena Master recusou o token desta instância."}
	case http.StatusNotFound:
		if hasMessage {
			return &ArenaError{Status: status, Code: "nao_encontrado",
				Message: "O Arena Master respondeu: “" + message + "”"}
		}
		return &ArenaError{Status: status, Code: "sem_modulo_arena",
			Message: "Esse servidor respondeu, mas não tem o módulo Arena instalado."}
	case http.StatusConflict:
		if hasMessage {
			return &ArenaError{Status: status, Code: "conflito_instancia", Message: message}
		}
		return &ArenaError{Status: status, Code: "conflito_instancia",
			Message: "Este token já está em uso por outra máquina."}
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		if hasMessage {
			return &ArenaError{Status: status, Code: "recusado", Message: message}
		}
		return &ArenaError{Status: status, Code: "recusado",
			Message: fmt.Sprintf("O Arena Master recusou o envio (HTTP %d).", status)}
	}
	if status >= 500 {
		return &ArenaError{Status: status, Code: "servidor", Retry: true,
			Message: fmt.Sprintf("O Arena Master está com problema (HTTP %d). Vou tentar de novo sozinho.", status)}
	}
	return &ArenaError{Status: status, Code: "inesperado",
		Message: fmt.Sprintf("O Arena Master respondeu HTTP %d.", status)}
}

func vernumMessage(body []byte) (string, bool) {
	var payload struct {
		Message string `json:"message"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	if payload.Message == "" || payload.Path != "" {
		return "", false
	}
	return payload.Message, true
}

func DescribeArenaError(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	if arenaErr, ok := err.(*ArenaError); ok {
		return arenaErr.Code, arenaErr.Message
	}
	return "rede", err.Error()
}

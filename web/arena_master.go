package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Team254/cheesy-arena-lite/config"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/partner"
	"github.com/Team254/cheesy-arena-lite/version"
	"github.com/google/uuid"
)

const (
	arenaStepBlocked   = "modo_incompativel"
	arenaStepPassword  = "senha"
	arenaStepConnect   = "conectar"
	arenaStepCheck     = "conferir"
	arenaStepRegister  = "registrar"
	arenaStepImport    = "importar"
	arenaStepReady     = "pronto"
	arenaProbeTimeout  = 10 * time.Second
	arenaMaxTokenChars = 200
)

func (web *Web) arenaStep() string {
	settings := web.arena.EventSettings
	if web.arena.Config.ModeFromEnv && web.arena.Config.Mode == config.ModeStandalone {
		return arenaStepBlocked
	}
	/*
	 * A arena em nuvem foi provisionada pelo Arena Master, que injetou endereco e token e ja sabe de
	 * que evento ela e. Nao ha o que um operador configure aqui, e mostrar o assistente sugeriria que
	 * ha — entao a tela e so o painel, e o registro acontece sozinho no boot.
	 */
	if web.arena.Mode() == config.ModeCloud {
		return arenaStepReady
	}
	if settings.AdminPassword == "" {
		return arenaStepPassword
	}
	if settings.ArenaToken == "" && !web.arena.Config.TokenFromEnv {
		return arenaStepConnect
	}
	if settings.ArenaCheckedAt == "" {
		return arenaStepCheck
	}
	if settings.ArenaConfirmedAt == "" {
		return arenaStepRegister
	}
	if settings.ArenaBootstrappedAt == "" {
		return arenaStepImport
	}
	return arenaStepReady
}

func (web *Web) arenaMasterPageHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	template, err := web.parseFiles("templates/arena_master.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		Step     string
		Mode     string
		EnvToken bool
		EnvMode  bool
		Version  string
	}{
		web.arena.EventSettings,
		web.arenaStep(),
		web.arena.Mode(),
		web.arena.Config.TokenFromEnv,
		web.arena.Config.ModeFromEnv,
		version.Version,
	}
	if err := template.ExecuteTemplate(w, "base", data); err != nil {
		handleWebErr(w, err)
	}
}

func (web *Web) arenaMasterStatusHandler(w http.ResponseWriter, r *http.Request) {
	if !web.apiRequireAdmin(w, r) {
		return
	}
	settings := web.arena.EventSettings
	venue := 0
	if settings.ArenaVenueSlot > 0 {
		venue = settings.ArenaVenueSlot
	}
	writeJson(w, http.StatusOK, map[string]interface{}{
		"step":           web.arenaStep(),
		"mode":           web.arena.Mode(),
		"modeFromEnv":    web.arena.Config.ModeFromEnv,
		"tokenFromEnv":   web.arena.Config.TokenFromEnv,
		"masterUrl":      settings.ArenaMasterUrl,
		"tokenPrefix":    settings.ArenaTokenPrefix,
		"hasToken":       settings.ArenaToken != "",
		"eventSlug":      settings.ArenaEventSlug,
		"eventName":      settings.ArenaEventName,
		"clientName":     settings.ArenaClientName,
		"publicUrl":      settings.ArenaPublicUrl,
		"venueSlot":      venue,
		"venueLabel":     settings.ArenaVenueLabel,
		"venueKind":      settings.ArenaVenueKind,
		"checkedAt":      settings.ArenaCheckedAt,
		"confirmedAt":    settings.ArenaConfirmedAt,
		"bootstrappedAt": settings.ArenaBootstrappedAt,
		"syncEnabled":    settings.ArenaSyncEnabled,
		"syncSeconds":    settings.ArenaSyncSeconds,
		"revision":       settings.ArenaRevision,
		"eventRevision":  settings.ArenaEventRevision,
		"lastSyncAt":     settings.ArenaLastSyncAt,
		"lastError":      settings.ArenaLastError,
		"lastErrorCode":  settings.ArenaLastErrorCode,
		"lastErrorAt":    settings.ArenaLastErrorAt,
		"version":        version.Version,
	})
}

func (web *Web) arenaMasterPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if !web.apiRequireJsonPost(w, r) {
		return
	}
	if web.arena.EventSettings.AdminPassword != "" && !web.userIsAdminApi(w, r) {
		return
	}
	var body struct {
		Password string `json:"password"`
		Confirm  string `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJsonError(w, http.StatusBadRequest, "corpo_invalido", "Não consegui ler o pedido.")
		return
	}
	if len(body.Password) < 6 {
		writeJsonError(w, http.StatusBadRequest, "senha_curta", "A senha precisa de pelo menos 6 caracteres.")
		return
	}
	if strings.ContainsAny(body.Password, `"'<>&`) {
		writeJsonError(w, http.StatusBadRequest, "senha_invalida",
			`A senha não pode conter aspas nem os sinais < > &.`)
		return
	}
	if body.Password != body.Confirm {
		writeJsonError(w, http.StatusBadRequest, "senha_diferente", "As duas senhas não são iguais.")
		return
	}
	if web.arena.MatchState != 0 {
		writeJsonError(w, http.StatusConflict, "partida_em_andamento",
			"Tem uma partida em andamento. Termine a partida e volte.")
		return
	}
	host, _ := os.Hostname()
	err := web.arena.SaveArenaSettings(func(settings *model.EventSettings) {
		settings.AdminPassword = body.Password
		if settings.ArenaClientUid == "" {
			settings.ArenaClientUid = strings.ReplaceAll(uuid.New().String(), "-", "")
			settings.ArenaClientHost = host
		}
		if settings.ArenaClientName == "" {
			settings.ArenaClientName = host
		}
		if settings.ArenaMode == "" {
			settings.ArenaMode = config.ModeLocal
		}
	})
	if err != nil {
		writeJsonError(w, http.StatusInternalServerError, "gravacao", err.Error())
		return
	}
	session := model.UserSession{
		Token: uuid.New().String(), Username: adminUser, CreatedAt: time.Now(), Role: "ADMIN",
	}
	if err := web.arena.Database.CreateUserSession(&session); err != nil {
		handleWebErr(w, err)
		return
	}
	web.setSessionCookie(w, r, session.Token)
	writeJson(w, http.StatusOK, map[string]interface{}{"ok": true, "step": web.arenaStep()})
}

func (web *Web) userIsAdminApi(w http.ResponseWriter, r *http.Request) bool {
	session := web.getUserSessionFromCookie(r)
	if session == nil || session.Role != "ADMIN" {
		writeJsonError(w, http.StatusUnauthorized, "sessao_expirada", "Sua sessão terminou. Entre de novo.")
		return false
	}
	return true
}

func (web *Web) arenaMasterConnectHandler(w http.ResponseWriter, r *http.Request) {
	if !web.apiRequireJsonPost(w, r) || !web.apiRequireAdmin(w, r) {
		return
	}
	var body struct {
		Url           string `json:"url"`
		Token         string `json:"token"`
		ExpectedSlug  string `json:"expectedSlug"`
		SaveAnyway    bool   `json:"saveAnyway"`
		AllowInsecure string `json:"allowInsecure"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJsonError(w, http.StatusBadRequest, "corpo_invalido", "Não consegui ler o pedido.")
		return
	}
	token := strings.TrimSpace(body.Token)
	if web.arena.Config.TokenFromEnv {
		token = web.arena.Config.Token
	}
	if token == "" {
		writeJsonError(w, http.StatusBadRequest, "token_vazio",
			"Cole o token que o administrador do evento te mandou.")
		return
	}
	if len(token) > arenaMaxTokenChars || !strings.HasPrefix(token, partner.ArenaTokenScheme) {
		writeJsonError(w, http.StatusBadRequest, "token_estranho",
			"Isso não parece um token de instância. Ele começa com ak_ e não tem espaços.")
		return
	}
	masterUrl, rewrote, cleaned, err := normalizeMasterUrl(body.Url)
	if err != nil {
		writeJsonError(w, http.StatusBadRequest, "endereco_invalido", err.Error())
		return
	}
	if strings.HasPrefix(masterUrl, "http://") && !isPrivateHost(masterUrl) &&
		body.AllowInsecure != "CONFIRMO" {
		writeJsonError(w, http.StatusBadRequest, "http_publico",
			"Esse endereço é público e http:// manda o token desta arena em texto puro pela rede — "+
				"e esse token autoriza substituir o evento inteiro no servidor. Use https://, ou digite "+
				"CONFIRMO se você sabe o que está fazendo.")
		return
	}

	expected := strings.TrimSpace(strings.ToLower(body.ExpectedSlug))
	if body.SaveAnyway {
		web.storeConnection(masterUrl, token, expected, "")
		writeJson(w, http.StatusOK, map[string]interface{}{
			"ok": true, "stage": "saved-offline", "url": masterUrl, "rewrote": rewrote, "cleaned": cleaned,
			"message": "Guardei o endereço e o token sem testar. Quando esta máquina tiver rede, volte " +
				"aqui e clique em Conferir — nada é enviado antes disso.",
			"step": web.arenaStep(),
		})
		return
	}

	serverVersion, err := probeVernumServer(masterUrl)
	if err != nil {
		code, message := partner.DescribeArenaError(err)
		writeJson(w, http.StatusOK, map[string]interface{}{
			"ok": false, "stage": "url", "code": code, "message": message, "url": masterUrl,
			"rewrote": rewrote, "cleaned": cleaned,
		})
		return
	}

	client := partner.NewArenaMasterClient(masterUrl, token, version.Version)
	boot, err := client.Bootstrap(web.arena.EventSettings.ArenaClientUid)
	if err != nil {
		code, message := partner.DescribeArenaError(err)
		writeJson(w, http.StatusOK, map[string]interface{}{
			"ok": false, "stage": "token", "code": code, "message": message, "url": masterUrl,
			"vernumVersion": serverVersion,
		})
		return
	}
	if expected != "" && !strings.Contains(strings.ToLower(boot.Event.Slug), expected) {
		writeJson(w, http.StatusOK, map[string]interface{}{
			"ok": false, "stage": "slug", "code": "evento_diferente", "url": masterUrl,
			"message": fmt.Sprintf("Este token é do evento %q, e você escreveu %q. "+
				"Confira com quem te mandou o token antes de seguir.", boot.Event.Slug, expected),
		})
		return
	}

	web.storeConnection(masterUrl, token, expected, boot.Event.Slug)
	writeJson(w, http.StatusOK, map[string]interface{}{
		"ok": true, "stage": "ok", "url": masterUrl, "rewrote": rewrote, "cleaned": cleaned,
		"vernumVersion": serverVersion, "eventName": boot.Event.Name, "eventSlug": boot.Event.Slug,
		"venueCount": boot.Event.Venue.Count, "venueKindPlural": boot.Event.Venue.KindLabelPlural,
		"step": web.arenaStep(),
	})
}

func (web *Web) storeConnection(masterUrl, token, expectedSlug, eventSlug string) {
	prefix := tokenPrefixOf(token)
	web.arena.SaveArenaSettings(func(settings *model.EventSettings) {
		changed := settings.ArenaMasterUrl != masterUrl || settings.ArenaTokenPrefix != prefix
		settings.ArenaMasterUrl = masterUrl
		settings.ArenaToken = token
		settings.ArenaTokenPrefix = prefix
		settings.ArenaExpectedSlug = expectedSlug
		if eventSlug != "" {
			settings.ArenaEventSlug = eventSlug
			settings.ArenaServerVerifiedAt = time.Now().Format(time.RFC3339)
		}
		if changed {
			settings.ArenaCheckedAt = ""
			settings.ArenaConfirmedAt = ""
			settings.ArenaBootstrapJson = ""
			settings.ArenaSyncEnabled = false
		}
	})
}

func (web *Web) arenaMasterCheckHandler(w http.ResponseWriter, r *http.Request) {
	if !web.apiRequireJsonPost(w, r) || !web.apiRequireAdmin(w, r) {
		return
	}
	settings := web.arena.EventSettings
	client := partner.NewArenaMasterClient(settings.ArenaMasterUrl, settings.ArenaToken, version.Version)
	boot, err := client.Bootstrap(settings.ArenaClientUid)
	if err != nil {
		code, message := partner.DescribeArenaError(err)
		writeJson(w, http.StatusOK, map[string]interface{}{"ok": false, "code": code, "message": message})
		return
	}
	raw, _ := json.Marshal(boot)
	fingerprint := web.localFingerprint()
	now := time.Now().Format(time.RFC3339)
	web.arena.SaveArenaSettings(func(s *model.EventSettings) {
		s.ArenaCheckedAt = now
		s.ArenaBootstrapJson = string(raw)
		s.ArenaLocalFingerprint = fingerprint
		s.ArenaEventSlug = boot.Event.Slug
		s.ArenaEventName = boot.Event.Name
		s.ArenaVenueKind = boot.Event.Venue.KindLabel
		s.ArenaVenueKindPlural = boot.Event.Venue.KindLabelPlural
		s.ArenaInstanceId = boot.Instance.InstanceId
	})

	local, _ := web.arena.Database.GetAllTeams()
	writeJson(w, http.StatusOK, map[string]interface{}{
		"ok": true, "checkedAt": now, "fingerprint": fingerprint,
		"bootstrap": boot, "teamDiff": diffTeams(local, boot.Teams),
		"settingsDiff": map[string]interface{}{
			"elimType":         []string{web.arena.EventSettings.ElimType, boot.Event.ElimType},
			"isFll":            []bool{web.arena.EventSettings.IsFll, boot.Event.IsFll},
			"teamsPerAlliance": []int{web.arena.EventSettings.TeamsPerAlliance, boot.Event.TeamsPerAlliance},
			"numElimAlliances": []int{web.arena.EventSettings.NumElimAlliances, boot.Event.NumElimAlliances},
			"eventName":        []string{web.arena.EventSettings.Name, boot.Event.Name},
		},
		"step": web.arenaStep(),
	})
}

func (web *Web) arenaMasterRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if !web.apiRequireJsonPost(w, r) || !web.apiRequireAdmin(w, r) {
		return
	}
	var body struct {
		ClientName  string `json:"clientName"`
		PublicUrl   string `json:"publicUrl"`
		VenueSlot   int    `json:"venueSlot"`
		VenueLabel  string `json:"venueLabel"`
		Takeover    bool   `json:"takeover"`
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJsonError(w, http.StatusBadRequest, "corpo_invalido", "Não consegui ler o pedido.")
		return
	}
	if web.arena.MatchState != 0 {
		writeJsonError(w, http.StatusConflict, "partida_em_andamento",
			"Tem uma partida em andamento. Termine a partida e volte.")
		return
	}
	settings := web.arena.EventSettings
	if settings.ArenaCheckedAt == "" {
		writeJsonError(w, http.StatusConflict, "sem_conferencia",
			"Confira os dados com o Arena Master antes de concluir.")
		return
	}
	if body.Fingerprint != "" && body.Fingerprint != web.localFingerprint() {
		writeJsonError(w, http.StatusConflict, "mudou_desde_a_conferencia",
			"A lista de equipes ou a configuração desta arena mudou depois que você conferiu. "+
				"Não vou concluir sobre uma tela que você não viu. Confira de novo.")
		return
	}
	name := strings.TrimSpace(body.ClientName)
	if name == "" {
		name = settings.ArenaClientName
	}
	if name == "" {
		writeJsonError(w, http.StatusBadRequest, "sem_nome",
			"Dê um nome a esta máquina. O administrador vai ver essa lista com várias linhas parecidas.")
		return
	}
	publicUrl := strings.TrimSpace(body.PublicUrl)
	if publicUrl == "" {
		publicUrl = guessPublicUrl(settings.ArenaMasterUrl, web.arena.Config.Port)
	}

	client := partner.NewArenaMasterClient(settings.ArenaMasterUrl, settings.ArenaToken, version.Version)
	beat := &partner.ArenaHeartbeat{
		ClientUid: settings.ArenaClientUid, ClientName: name, ClientVersion: version.Version,
		Mode: strings.ToUpper(web.arena.Mode()), PublicUrl: publicUrl,
		EventSlug: settings.ArenaEventSlug, Takeover: body.Takeover,
		Revision: settings.ArenaRevision, ContentHash: "",
	}
	if body.VenueSlot > 0 {
		slot := body.VenueSlot
		beat.VenueSlot = &slot
		beat.VenueLabel = body.VenueLabel
	}
	result, err := client.Heartbeat(beat)
	if err != nil {
		code, message := partner.DescribeArenaError(err)
		writeJson(w, http.StatusOK, map[string]interface{}{
			"ok": false, "code": code, "message": message,
			"canTakeover": code == "conflito_instancia",
		})
		return
	}
	now := time.Now().Format(time.RFC3339)
	web.arena.SaveArenaSettings(func(s *model.EventSettings) {
		s.ArenaInstanceId = result.InstanceId
		s.ArenaIsAnchor = result.Anchor
		s.ArenaClientName = name
		s.ArenaPublicUrl = publicUrl
		s.ArenaConfirmedAt = now
		s.ArenaSyncEnabled = true
		if result.VenueSlot != nil {
			s.ArenaVenueSlot = *result.VenueSlot
		}
		s.ArenaVenueLabel = result.VenueLabel
		if result.VenueKindLabel != "" {
			s.ArenaVenueKind = result.VenueKindLabel
		}
	})
	web.arena.WakeArenaSync()
	writeJson(w, http.StatusOK, map[string]interface{}{
		"ok": true, "instanceId": result.InstanceId, "claimedNow": result.ClaimedNow,
		"venueSlot": result.VenueSlot, "venueLabel": result.VenueLabel,
		"notices": result.Notices, "step": web.arenaStep(),
	})
}

func (web *Web) arenaMasterApplyHandler(w http.ResponseWriter, r *http.Request) {
	if !web.apiRequireJsonPost(w, r) || !web.apiRequireAdmin(w, r) {
		return
	}
	var body struct {
		ImportTeams   bool `json:"importTeams"`
		ApplySettings bool `json:"applySettings"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	cached := web.arena.EventSettings.ArenaBootstrapJson
	if cached == "" {
		writeJsonError(w, http.StatusConflict, "sem_cache",
			"Não tenho a conferência guardada. Clique em Conferir antes de importar.")
		return
	}
	boot := new(partner.ArenaBootstrap)
	if err := json.Unmarshal([]byte(cached), boot); err != nil {
		writeJsonError(w, http.StatusConflict, "cache_ilegivel",
			"A conferência guardada está ilegível. Confira de novo.")
		return
	}

	messages := []string{}
	added, kept := 0, 0
	if body.ImportTeams {
		if !web.canModifyTeamList() {
			writeJsonError(w, http.StatusConflict, "agenda_existente",
				"Já existe agenda de classificação nesta arena. Acrescentar equipe agora invalidaria a "+
					"agenda e a classificação, que foram calculadas sobre a lista antiga. Limpe a agenda "+
					"antes, ou siga sem importar.")
			return
		}
		for _, team := range boot.Teams {
			if team.Number <= 0 {
				continue
			}
			if existing, _ := web.arena.Database.GetTeamById(team.Number); existing != nil {
				kept++
				continue
			}
			record := model.Team{
				Id: team.Number, Name: team.Name, Nickname: team.Nickname, City: team.City,
				StateProv: team.StateProv, Country: team.Country, RookieYear: team.RookieYear,
				RobotName: team.RobotName,
			}
			if err := web.arena.Database.CreateTeam(&record); err != nil {
				handleWebErr(w, err)
				return
			}
			added++
		}
		messages = append(messages, fmt.Sprintf("%d equipes acrescentadas, %d já existiam aqui e não foram tocadas.", added, kept))
	}

	if body.ApplySettings {
		elim := boot.Event.ElimType
		alliances := boot.Event.NumElimAlliances
		if elim == "double" {
			alliances = 8
		} else if alliances < 2 || alliances > 16 {
			alliances = 8
		}
		web.arena.SaveArenaSettings(func(s *model.EventSettings) {
			s.Name = boot.Event.Name
			s.ElimType = elim
			s.NumElimAlliances = alliances
			s.IsFll = boot.Event.IsFll
			if boot.Event.TeamsPerAlliance > 0 {
				s.TeamsPerAlliance = boot.Event.TeamsPerAlliance
			}
		})
		messages = append(messages, fmt.Sprintf("Configuração aplicada: %s, %d alianças, %d equipes por aliança.",
			elim, alliances, boot.Event.TeamsPerAlliance))
	}

	now := time.Now().Format(time.RFC3339)
	web.arena.SaveArenaSettings(func(s *model.EventSettings) { s.ArenaBootstrappedAt = now })
	web.arena.WakeArenaSync()
	writeJson(w, http.StatusOK, map[string]interface{}{
		"ok": true, "teamsAdded": added, "teamsKept": kept, "messages": messages, "step": web.arenaStep(),
	})
}

func (web *Web) arenaMasterResetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.apiRequireJsonPost(w, r) || !web.apiRequireAdmin(w, r) {
		return
	}
	var body struct {
		Revoke bool `json:"revoke"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	revoked := false
	if body.Revoke && web.arena.ArenaMasterClient.Ready() {
		if err := web.arena.ArenaMasterClient.RevokeSelf(); err == nil {
			revoked = true
		}
	}
	web.arena.SaveArenaSettings(func(s *model.EventSettings) {
		s.ArenaMasterUrl = ""
		s.ArenaToken = ""
		s.ArenaTokenPrefix = ""
		s.ArenaExpectedSlug = ""
		s.ArenaEventSlug = ""
		s.ArenaEventName = ""
		s.ArenaInstanceId = 0
		s.ArenaVenueSlot = 0
		s.ArenaVenueLabel = ""
		s.ArenaCheckedAt = ""
		s.ArenaConfirmedAt = ""
		s.ArenaBootstrappedAt = ""
		s.ArenaBootstrapJson = ""
		s.ArenaServerVerifiedAt = ""
		s.ArenaSyncEnabled = false
		s.ArenaLastError = ""
		s.ArenaLastErrorCode = ""
	})
	writeJson(w, http.StatusOK, map[string]interface{}{
		"ok": true, "revoked": revoked, "step": web.arenaStep(),
	})
}

func (web *Web) arenaMasterSyncNowHandler(w http.ResponseWriter, r *http.Request) {
	if !web.apiRequireJsonPost(w, r) || !web.apiRequireAdmin(w, r) {
		return
	}
	result, err := web.arena.SyncNow()
	if err != nil {
		code, message := partner.DescribeArenaError(err)
		writeJson(w, http.StatusOK, map[string]interface{}{"ok": false, "code": code, "message": message})
		return
	}
	if result.Unchanged {
		writeJson(w, http.StatusOK, map[string]interface{}{
			"ok": true, "unchanged": true, "message": "Nada mudou desde o último envio.",
		})
		return
	}
	writeJson(w, http.StatusOK, map[string]interface{}{
		"ok": true, "revision": result.InstanceRevision, "matches": result.Matches,
		"teams": result.Teams, "ignored": result.Ignored, "notices": result.Notices,
		"message": fmt.Sprintf("Enviado: %d equipes, %d partidas. Revisão %d.",
			result.Teams, result.Matches, result.InstanceRevision),
	})
}

func (web *Web) localFingerprint() string {
	teams, _ := web.arena.Database.GetAllTeams()
	ids := make([]int, 0, len(teams))
	for _, team := range teams {
		ids = append(ids, team.Id)
	}
	sort.Ints(ids)
	settings := web.arena.EventSettings
	parts := []string{settings.Name, settings.ElimType, strconv.FormatBool(settings.IsFll),
		strconv.Itoa(settings.TeamsPerAlliance), strconv.Itoa(settings.NumElimAlliances),
		settings.ArenaTokenPrefix}
	for _, id := range ids {
		parts = append(parts, strconv.Itoa(id))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func normalizeMasterUrl(raw string) (string, bool, []string, error) {
	value := strings.TrimSpace(raw)
	cleaned := []string{}
	rewrote := false
	if value == "" {
		return "", false, nil, fmt.Errorf("Digite o endereço do Arena Master. Exemplo: https://server.frc9611.com")
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
		rewrote = true
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", false, nil, fmt.Errorf("Esse endereço não é válido. Exemplo: https://server.frc9611.com")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false, nil, fmt.Errorf("O endereço tem que começar com http:// ou https://.")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		cleaned = append(cleaned, parsed.Path)
	}
	return parsed.Scheme + "://" + parsed.Host, rewrote, cleaned, nil
}

func isPrivateHost(rawUrl string) bool {
	parsed, err := url.Parse(rawUrl)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if host == "localhost" || strings.HasSuffix(host, ".local") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return true
	}
	shared := net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}
	return shared.Contains(ip)
}

func probeVernumServer(masterUrl string) (string, error) {
	client := &http.Client{Timeout: arenaProbeTimeout}
	response, err := client.Get(masterUrl + "/serverInfo")
	if err != nil {
		return "", partner.DescribeTransportError(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", &partner.ArenaError{Code: "nao_e_vernum", Status: response.StatusCode,
			Message: fmt.Sprintf("Esse endereço respondeu HTTP %d em /serverInfo. "+
				"Ele não parece ser um servidor Vernum.", response.StatusCode)}
	}
	var info struct {
		VernumVersion string `json:"vernumVersion"`
	}
	if err := json.NewDecoder(response.Body).Decode(&info); err != nil || info.VernumVersion == "" {
		return "", &partner.ArenaError{Code: "nao_e_vernum",
			Message: "Esse endereço respondeu, mas não como um servidor Vernum responderia."}
	}
	return info.VernumVersion, nil
}

func guessPublicUrl(masterUrl string, port int) string {
	host := ""
	parsed, err := url.Parse(masterUrl)
	if err == nil && parsed.Host != "" {
		target := parsed.Hostname() + ":80"
		if conn, err := net.DialTimeout("udp", target, time.Second); err == nil {
			if local, ok := conn.LocalAddr().(*net.UDPAddr); ok {
				host = local.IP.String()
			}
			conn.Close()
		}
	}
	if host == "" {
		if name, err := os.Hostname(); err == nil {
			host = name
		}
	}
	if host == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

func tokenPrefixOf(token string) string {
	parts := strings.SplitN(strings.TrimPrefix(token, partner.ArenaTokenScheme), "_", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func diffTeams(local []model.Team, remote []partner.ArenaBootstrapTeam) map[string][]int {
	here := map[int]string{}
	for _, team := range local {
		if team.Id > 0 {
			here[team.Id] = team.Nickname
		}
	}
	there := map[int]string{}
	for _, team := range remote {
		if team.Number > 0 {
			there[team.Number] = team.Nickname
		}
	}
	same, different, onlyHere, onlyThere := []int{}, []int{}, []int{}, []int{}
	for number, nickname := range here {
		other, ok := there[number]
		switch {
		case !ok:
			onlyHere = append(onlyHere, number)
		case other == nickname:
			same = append(same, number)
		default:
			different = append(different, number)
		}
	}
	for number := range there {
		if _, ok := here[number]; !ok {
			onlyThere = append(onlyThere, number)
		}
	}
	sort.Ints(same)
	sort.Ints(different)
	sort.Ints(onlyHere)
	sort.Ints(onlyThere)
	return map[string][]int{"iguais": same, "diferentes": different, "soAqui": onlyHere, "soLa": onlyThere}
}

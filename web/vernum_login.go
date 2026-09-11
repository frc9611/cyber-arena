package web

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/google/uuid"
)

type vernumTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
	Me          struct {
		User struct {
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"user"`
	} `json:"me"`
}

type vernumArenaMe struct {
	CanManage bool `json:"canManage"`
	Events    []struct {
		EventId   int64  `json:"eventId"`
		Slug      string `json:"slug"`
		CanManage bool   `json:"canManage"`
	} `json:"events"`
}

func (web *Web) vernumRedirectUri(r *http.Request) string {
	scheme := "http"
	if requestIsSecure(r) {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, web.arena.Config.Path("/sso/callback"))
}

func (web *Web) startVernumLogin(w http.ResponseWriter, r *http.Request) {
	cfg := web.arena.Config
	if cfg.VernumSsoUrl == "" || cfg.VernumClientId == "" {
		web.renderLogin(w, r, "Esta arena deveria entrar pelo Vernum, mas subiu sem VERNUM_SSO_URL ou "+
			"VERNUM_CLIENT_ID. Quem sobe o processo define isso.")
		return
	}
	state := uuid.New().String()
	verifier, err := newCodeVerifier()
	if err != nil {
		web.renderLogin(w, r, "Não consegui preparar a entrada pelo Vernum. Tente de novo.")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     vernumStateCookie,
		Value:    state + "|" + verifier + "|" + r.URL.Query().Get("redirect"),
		Path:     web.cookiePath(),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   600,
	})
	target := fmt.Sprintf(
		"%s/entrar-com-vernum?client_id=%s&redirect_uri=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		cfg.VernumSsoUrl,
		url.QueryEscape(cfg.VernumClientId),
		url.QueryEscape(web.vernumRedirectUri(r)),
		url.QueryEscape(state),
		url.QueryEscape(codeChallengeOf(verifier)))
	http.Redirect(w, r, target, 303)
}

func (web *Web) vernumCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		web.renderLogin(w, r, "O Vernum não devolveu um código de acesso. Tente entrar de novo.")
		return
	}
	stateCookie, err := r.Cookie(vernumStateCookie)
	if err != nil {
		web.renderLogin(w, r, "A volta do Vernum chegou sem a marca desta tentativa. Tente entrar de novo.")
		return
	}
	parts := splitState(stateCookie.Value)
	if parts[0] == "" || parts[0] != r.URL.Query().Get("state") {
		web.renderLogin(w, r, "A volta do Vernum não bate com a tentativa que saiu daqui.")
		return
	}

	token, err := web.exchangeVernumCode(code, web.vernumRedirectUri(r), parts[1])
	if err != nil {
		web.renderLogin(w, r, err.Error())
		return
	}
	role, err := web.vernumRoleForThisEvent(token.AccessToken)
	if err != nil {
		web.renderLogin(w, r, err.Error())
		return
	}

	name := token.Me.User.Username
	if name == "" {
		name = "vernum"
	}
	session := model.UserSession{
		Token:     uuid.New().String(),
		Username:  name,
		CreatedAt: time.Now(),
		Role:      role,
	}
	if err := web.arena.Database.CreateUserSession(&session); err != nil {
		handleWebErr(w, err)
		return
	}
	web.setSessionCookie(w, r, session.Token)
	http.SetCookie(w, &http.Cookie{Name: vernumStateCookie, Value: "", Path: web.cookiePath(), MaxAge: -1})

	redirectUrl := parts[2]
	if redirectUrl == "" {
		redirectUrl = web.arena.Config.Path("/")
	}
	http.Redirect(w, r, redirectUrl, 303)
}

func (web *Web) exchangeVernumCode(code, redirectUri, verifier string) (*vernumTokenResponse, error) {
	cfg := web.arena.Config
	body, _ := json.Marshal(map[string]string{
		"clientId":     cfg.VernumClientId,
		"clientSecret": cfg.VernumClientSecret,
		"code":         code,
		"redirectUri":  redirectUri,
		"codeVerifier": verifier,
	})
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(cfg.VernumApiUrl+"/public/sso/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Não consegui falar com o Vernum: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("O Vernum recusou a entrada (HTTP %d).", resp.StatusCode)
	}
	token := new(vernumTokenResponse)
	if err := json.NewDecoder(resp.Body).Decode(token); err != nil || token.AccessToken == "" {
		return nil, fmt.Errorf("O Vernum respondeu algo que não consegui ler.")
	}
	return token, nil
}

func (web *Web) vernumRoleForThisEvent(accessToken string) (string, error) {
	cfg := web.arena.Config
	req, err := http.NewRequest("GET", cfg.VernumApiUrl+"/arena/me", nil)
	if err != nil {
		return "", fmt.Errorf("Não consegui perguntar ao Vernum quem é você.")
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Não consegui falar com o Arena Master: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("Você entrou no Vernum, mas não tem acesso ao Arena Master. " +
			"Peça a permissão a quem administra a plataforma.")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("O Arena Master respondeu HTTP %d.", resp.StatusCode)
	}
	me := new(vernumArenaMe)
	if err := json.NewDecoder(resp.Body).Decode(me); err != nil {
		return "", fmt.Errorf("O Arena Master respondeu algo que não consegui ler.")
	}
	slug := web.arena.EventSettings.ArenaEventSlug
	for _, event := range me.Events {
		if slug != "" && event.Slug != slug {
			continue
		}
		if event.CanManage || me.CanManage {
			return "ADMIN", nil
		}
		return "REGULAR", nil
	}
	if me.CanManage {
		return "ADMIN", nil
	}
	return "", fmt.Errorf("Você não foi convidado para este evento no Arena Master.")
}

func newCodeVerifier() (string, error) {
	drawn := make([]byte, 48)
	if _, err := rand.Read(drawn); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(drawn), nil
}

func codeChallengeOf(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func splitState(value string) [3]string {
	parts := strings.SplitN(value, "|", 3)
	answer := [3]string{}
	for i := 0; i < len(parts) && i < 3; i++ {
		answer[i] = parts[i]
	}
	return answer
}

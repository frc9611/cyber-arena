// Copyright 2018 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web routes for authenticating with the server.

package web

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Team254/cheesy-arena-lite/config"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/google/uuid"
)

// Shows the login form.
func (web *Web) loginHandler(w http.ResponseWriter, r *http.Request) {
	if web.arena.Mode() == config.ModeCloud {
		web.startVernumLogin(w, r)
		return
	}
	web.renderLogin(w, r, "")
}

// Processes the login request.
func (web *Web) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	username := r.PostFormValue("username")
	scope, err := web.checkAuthPassword(username, r.PostFormValue("password"), r)
	if err != nil {
		web.renderLogin(w, r, err.Error())
		return
	}

	if scope == "INVALID" {
		web.renderLogin(w, r, fmt.Errorf("[VERNUMSERVER] Você não tem permissão para acessar o sistema").Error())
		return
	}

	session := model.UserSession{
		Token:     uuid.New().String(),
		Username:  username,
		CreatedAt: time.Now(),
		Role:      scope,
	}

	if err := web.arena.Database.CreateUserSession(&session); err != nil {
		handleWebErr(w, err)
		return
	}

	web.setSessionCookie(w, r, session.Token)
	redirectUrl := r.URL.Query().Get("redirect")
	if redirectUrl == "" {
		redirectUrl = "/"
	}
	http.Redirect(w, r, redirectUrl, 303)
}

func (web *Web) renderLogin(w http.ResponseWriter, r *http.Request, errorMessage string) {
	template, err := web.parseFiles("templates/login.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		ErrorMessage string
	}{web.arena.EventSettings, errorMessage}
	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Returns true if the given user is authorized for admin operations. Used for HTTP cookie authentication.
func (web *Web) userIsAdmin(w http.ResponseWriter, r *http.Request) bool {
	if web.arena.EventSettings.AdminPassword == "" && web.authCanBeSkipped() {
		// Disable auth if there is no password configured.
		return true
	}
	session := web.getUserSessionFromCookie(r)
	if session != nil && session.Role == "ADMIN" {
		return true
	} else {
		redirect := r.URL.Path
		if r.URL.RawQuery != "" {
			redirect += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, web.arena.Config.Path("/login?redirect=")+url.QueryEscape(redirect), 307)
		return false
	}
}

// Returns true if the given user is authorized for admin operations. Used for HTTP cookie authentication.
func (web *Web) userIsRefereeOrHigher(w http.ResponseWriter, r *http.Request) bool {
	if web.arena.EventSettings.AdminPassword == "" && web.authCanBeSkipped() {
		// Disable auth if there is no password configured.
		return true
	}
	session := web.getUserSessionFromCookie(r)
	if session != nil && (session.Role == "REGULAR" || session.Role == "ADMIN") {
		return true
	} else {
		redirect := r.URL.Path
		if r.URL.RawQuery != "" {
			redirect += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, web.arena.Config.Path("/login")+"?redirect="+url.QueryEscape(redirect), 307)
		return false
	}
}

// Secure follows how the request actually arrived, not the mode. A cloud arena behind an ingress
// with no certificate is served over plain HTTP, and a Secure cookie there is dropped by the
// browser without a word: the session never sticks, the next page sends the person back to the SSO,
// and the login spins forever.
//
// The path is the base path of this arena, so two arenas on the same host cannot read each other's
// session or answer each other's login.
func (web *Web) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionTokenCookie,
		Value:    token,
		Path:     web.cookiePath(),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
	})
}

func (web *Web) cookiePath() string {
	if web.arena.Config == nil || web.arena.Config.BasePath == "" {
		return "/"
	}
	return web.arena.Config.BasePath
}

func requestIsSecure(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (web *Web) getUserSessionFromCookie(r *http.Request) *model.UserSession {
	token, err := r.Cookie(sessionTokenCookie)
	if err != nil {
		return nil
	}
	session, _ := web.arena.Database.GetUserSessionByToken(token.Value)
	return session
}

/*
 * A arena de campo sem senha e uma decisao de quem esta no ginasio: a rede e a sala, e pedir senha
 * atrapalha mais do que protege. A arena em nuvem esta na internet, e a mesma ausencia de senha
 * deixaria o evento inteiro aberto para qualquer pessoa que descobrisse o endereco. Ali a porta e o
 * login do Vernum, e nao ha porta alternativa.
 */
func (web *Web) authCanBeSkipped() bool {
	return web.arena.Mode() != config.ModeCloud
}

func requestIsFromThisMachine(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsLoopback()
}

func (web *Web) checkAuthPassword(user, password string, r *http.Request) (string, error) {
	mode := web.arena.Mode()
	if mode == config.ModeCloud {
		return "INVALID", fmt.Errorf("Esta arena entra pelo Vernum. Use o botão de entrar com o Vernum.")
	}
	if mode == config.ModeLocal && user == localUser && password == localPassword {
		if !requestIsFromThisMachine(r) {
			return "INVALID", fmt.Errorf("O login %s só vale no próprio computador da arena. "+
				"De outra máquina, entre como %s com a senha das configurações.", localUser, adminUser)
		}
		return "ADMIN", nil
	}
	if user == adminUser && web.arena.EventSettings.AdminPassword != "" &&
		password == web.arena.EventSettings.AdminPassword {
		return "ADMIN", nil
	}
	return "INVALID", fmt.Errorf("Usuário ou senha incorretos.")
}

func (web *Web) apiRequireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if web.arena.EventSettings.AdminPassword == "" {
		writeJsonError(w, http.StatusForbidden, "mesa_sem_senha",
			"Defina a senha de administrador antes de conectar esta arena a qualquer lugar.")
		return false
	}
	session := web.getUserSessionFromCookie(r)
	if session == nil || session.Role != "ADMIN" {
		writeJsonError(w, http.StatusUnauthorized, "sessao_expirada",
			"Sua sessão terminou. Entre de novo.")
		return false
	}
	return true
}

func (web *Web) apiRequireJsonPost(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJsonError(w, http.StatusUnsupportedMediaType, "corpo_invalido",
			"Esta rota só aceita application/json.")
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		writeJsonError(w, http.StatusForbidden, "origem_ausente",
			"Pedido de escrita sem origem. Use a tela do assistente.")
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host != r.Host {
		writeJsonError(w, http.StatusForbidden, "origem_estranha",
			"Este pedido veio de outra página.")
		return false
	}
	return true
}

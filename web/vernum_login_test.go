package web

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodeChallengeMatchesRfc7636(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	assert.Equal(t, "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", codeChallengeOf(verifier))
}

func TestCodeVerifierIsLongAndUrlSafe(t *testing.T) {
	verifier, err := newCodeVerifier()
	assert.Nil(t, err)
	assert.True(t, len(verifier) >= 43 && len(verifier) <= 128, "comprimento %d", len(verifier))
	for _, char := range verifier {
		assert.True(t, char == '-' || char == '_' ||
			(char >= '0' && char <= '9') || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z'),
			"caractere inválido %q", char)
	}
}

func TestSplitStateKeepsTheRedirectWhole(t *testing.T) {
	parts := splitState("st|vf|/e/copa/match_play?x=1|2")
	assert.Equal(t, "st", parts[0])
	assert.Equal(t, "vf", parts[1])
	assert.Equal(t, "/e/copa/match_play?x=1|2", parts[2])

	old := splitState("st")
	assert.Equal(t, "st", old[0])
	assert.Equal(t, "", old[1])
	assert.Equal(t, "", old[2])
}

// The loop this fixes: a cloud arena behind an ingress with no certificate answers over plain HTTP,
// a Secure cookie there is dropped by the browser without a word, and the next page sends the
// person straight back to the SSO. Forever.
func TestTheSessionCookieIsOnlySecureOverHttps(t *testing.T) {
	web := setupTestWeb(t)
	web.arena.Config.Mode = "cloud"
	web.arena.Config.ModeFromEnv = true

	plain := httptest.NewRecorder()
	web.setSessionCookie(plain, httptest.NewRequest("GET", "http://arena.exemplo/e/copa/", nil), "abc")
	assert.Contains(t, plain.Header().Get("Set-Cookie"), "abc")
	assert.NotContains(t, plain.Header().Get("Set-Cookie"), "Secure")

	forwarded := httptest.NewRequest("GET", "http://arena.exemplo/e/copa/", nil)
	forwarded.Header.Set("X-Forwarded-Proto", "https")
	behind := httptest.NewRecorder()
	web.setSessionCookie(behind, forwarded, "abc")
	assert.Contains(t, behind.Header().Get("Set-Cookie"), "Secure")
}

func TestTheCookieIsScopedToThisArena(t *testing.T) {
	web := setupTestWeb(t)
	web.arena.Config.BasePath = "/e/copa"
	scoped := httptest.NewRecorder()
	web.setSessionCookie(scoped, httptest.NewRequest("GET", "http://arena.exemplo/e/copa/", nil), "abc")
	assert.Contains(t, scoped.Header().Get("Set-Cookie"), "Path=/e/copa")

	web.arena.Config.BasePath = ""
	root := httptest.NewRecorder()
	web.setSessionCookie(root, httptest.NewRequest("GET", "http://arena.exemplo/", nil), "abc")
	assert.Contains(t, root.Header().Get("Set-Cookie"), "Path=/")
}

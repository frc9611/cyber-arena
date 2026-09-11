package web

import (
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

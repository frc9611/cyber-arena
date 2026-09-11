package partner

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArenaSnapshotSeparatesNullFromEmpty(t *testing.T) {
	empty := make([]ArenaBootstrapTeam, 0)
	snapshot := ArenaSnapshot{Teams: &empty}
	encoded, err := json.Marshal(snapshot)
	assert.Nil(t, err)
	assert.Contains(t, string(encoded), `"teams":[]`)
	assert.Contains(t, string(encoded), `"matches":null`)
	assert.Contains(t, string(encoded), `"fllScores":null`)
}

func TestArenaClientSendsTokenAndBody(t *testing.T) {
	var gotToken, gotAgent, gotBody, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get(ArenaTokenHeader)
		gotAgent = r.Header.Get("User-Agent")
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"instanceRevision":7,"eventRevision":12,"matches":3,"ignored":[],"notices":[]}`))
	}))
	defer server.Close()

	client := NewArenaMasterClient(server.URL, "ak_abc_segredo", "1.4.0-vernum")
	matches := []ArenaSnapshotMatch{{MatchKey: "1-qualification-3", Type: "qualification"}}
	result, err := client.PushSnapshot(&ArenaSnapshot{ClientUid: "maquina-1", CoversAll: true, Matches: &matches})

	assert.Nil(t, err)
	assert.Equal(t, "ak_abc_segredo", gotToken)
	assert.Equal(t, "cyber-arena/1.4.0-vernum", gotAgent)
	assert.Equal(t, "/public/arena/instance/sync", gotPath)
	assert.Contains(t, gotBody, `"clientUid":"maquina-1"`)
	assert.Contains(t, gotBody, `"coversAll":true`)
	assert.Equal(t, int64(7), result.InstanceRevision)
	assert.False(t, result.Unchanged)
}

func TestArenaClientReadsUnchanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewArenaMasterClient(server.URL, "ak_abc_segredo", "test")
	result, err := client.PushSnapshot(&ArenaSnapshot{})
	assert.Nil(t, err)
	assert.True(t, result.Unchanged)
}

func TestArenaClientTranslatesRefusedToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"Forbidden","status":403,"message":"Token de instância inválido ou revogado."}`))
	}))
	defer server.Close()

	client := NewArenaMasterClient(server.URL, "ak_errado_x", "test")
	_, err := client.Bootstrap("maquina-1")
	code, message := DescribeArenaError(err)
	assert.Equal(t, "token_invalido", code)
	assert.Contains(t, message, "Token de instância inválido")
	assert.False(t, err.(*ArenaError).Retry)
}

func TestArenaClientSeparatesMissingModuleFromMissingEvent(t *testing.T) {
	withMessage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Not Found","status":404,"message":"Evento não encontrado."}`))
	}))
	defer withMessage.Close()
	code, message := DescribeArenaError(errorOf(NewArenaMasterClient(withMessage.URL, "ak_a_b", "test")))
	assert.Equal(t, "nao_encontrado", code)
	assert.Contains(t, message, "Evento não encontrado")

	withoutMessage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"timestamp":"now","status":404,"error":"Not Found","path":"/public/arena/instance/bootstrap"}`))
	}))
	defer withoutMessage.Close()
	code, message = DescribeArenaError(errorOf(NewArenaMasterClient(withoutMessage.URL, "ak_a_b", "test")))
	assert.Equal(t, "sem_modulo_arena", code)
	assert.Contains(t, message, "não tem o módulo Arena")
}

func TestArenaClientRetriesOnlyWhatIsWorthRetrying(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	_, err := NewArenaMasterClient(server.URL, "ak_a_b", "test").Bootstrap("")
	assert.True(t, err.(*ArenaError).Retry)

	conflict := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"status":409,"message":"Este token já está em uso pela máquina \"mesa-1\"."}`))
	}))
	defer conflict.Close()
	_, err = NewArenaMasterClient(conflict.URL, "ak_a_b", "test").Bootstrap("")
	assert.False(t, err.(*ArenaError).Retry)
	code, message := DescribeArenaError(err)
	assert.Equal(t, "conflito_instancia", code)
	assert.Contains(t, message, "mesa-1")
}

func TestArenaClientRefusesRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("o token seguiu um redirecionamento para outro host")
	}))
	defer final.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/public/arena/instance/bootstrap", http.StatusFound)
	}))
	defer redirector.Close()

	_, err := NewArenaMasterClient(redirector.URL, "ak_a_b", "test").Bootstrap("")
	assert.NotNil(t, err)
	assert.False(t, strings.Contains(err.Error(), "panic"))
}

func TestArenaClientRefusesWhenNotConfigured(t *testing.T) {
	_, err := NewArenaMasterClient("", "", "test").Bootstrap("")
	code, _ := DescribeArenaError(err)
	assert.Equal(t, "nao_configurado", code)
}

func errorOf(client *ArenaMasterClient) error {
	_, err := client.Bootstrap("maquina-1")
	return err
}

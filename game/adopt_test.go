package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Uma arena adota o documento que o Arena Master entregou mesmo sem o binário dela embutir aquela
// revisão — e recusa o que não passa nos próprios casos, que é o que impede uma temporada quebrada
// de entrar em campo numa manhã de domingo.
func TestAdoptASeasonThisBuildDoesNotEmbed(t *testing.T) {
	body := []byte(`{
      "schema": 2, "key": "xyz-2030-teste", "revision": 1, "program": "TESTE",
      "name": "Temporada de teste", "year": 2030, "format": "alliance", "robotsPerAlliance": 2,
      "periods": [{ "id": "teleop", "label": "Teleoperado", "durationSec": 100, "scoring": true }],
      "eventLevels": [{ "id": "EVENTO", "label": "Evento" }],
      "thresholds": { "meta": { "*": 10 } },
      "categories": [
        { "id": "pontos", "label": "Pontos" },
        { "id": "foulCommitted", "label": "Cometidas", "kind": "foulCommitted" },
        { "id": "foulReceived", "label": "Recebidas", "kind": "foulReceived" }],
      "actions": [
        { "id": "cesta", "label": "Cesta", "category": "pontos", "unit": "count",
          "periods": ["teleop"], "points": { "teleop": 5 } }],
      "ranking": {
        "mode": "alliance", "rsDecimals": 2, "result": { "win": 3, "tie": 1, "loss": 0 },
        "rankingPoints": [
          { "id": "metaRp", "label": "META RP", "points": 1,
            "when": [">=", ["cat", "pontos"], ["thr", "meta"]] }],
        "tiebreakers": [{ "id": "total", "label": "Total", "expr": ["total"] }] },
      "tests": [
        { "name": "duas cestas dão 10 e o RP", "level": "EVENTO",
          "state": { "actions": { "cesta@teleop": 2 } },
          "expect": { "total": 10, "rpEarned": ["metaRp"], "sort": [10] } },
        { "name": "uma cesta não dá o RP", "level": "EVENTO",
          "state": { "actions": { "cesta@teleop": 1 } },
          "expect": { "total": 5, "rpEarned": [] } }]
    }`)

	assert.Nil(t, SeasonByKey("xyz-2030-teste"), "este binário não embute essa temporada")
	season, err := AdoptSeason(body)
	assert.Nil(t, err)
	if assert.NotNil(t, season) {
		assert.Equal(t, "xyz-2030-teste", season.Key)
		assert.Same(t, season, SeasonByKey("xyz-2030-teste"), "fica registrada e pontua daqui em diante")
	}

	score := &Score{SeasonKey: "xyz-2030-teste", Level: "EVENTO", Robots: 2}
	score.SetAction("cesta", "teleop", 3)
	assert.Equal(t, 15, score.Summarize().Score)

	// O mesmo documento com um valor trocado deixa de bater com o próprio caso e é recusado.
	broken := []byte(replaceOnce(string(body), `"points": { "teleop": 5 }`, `"points": { "teleop": 6 }`))
	_, err = AdoptSeason(broken)
	if assert.NotNil(t, err, "um pacote que falha nos próprios casos não pode ser adotado") {
		assert.Contains(t, err.Error(), "casos")
	}
}

func replaceOnce(text, old, replacement string) string {
	for i := 0; i+len(old) <= len(text); i++ {
		if text[i:i+len(old)] == old {
			return text[:i] + replacement + text[i+len(old):]
		}
	}
	return text
}

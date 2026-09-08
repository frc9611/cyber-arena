// Copyright 2023 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side logic for the referee interface.

var websocket;
let redFoulsHashCode = 0;
let blueFoulsHashCode = 0;
let _cr_scoreContext_DEFAULT = {
  blue: {
    auto: {
      "ZONA_BAIXA": 0,
      "PLATAFORMA": 0,
      "SAIU_AREA": 0,
    },
    teleop: {
      "ZONA_BAIXA": 0,
      "ZONA_ALTA": 0,
    },
    endgame: {
      "PLATAFORMA": 0,
      "ESTACIONAR": 0,
    },
  },
  red: {
    auto: {
      "ZONA_BAIXA": 0,
      "PLATAFORMA": 0,
      "SAIU_AREA": 0,
    },
    teleop: {
      "ZONA_BAIXA": 0,
      "ZONA_ALTA": 0,
    },
    endgame: {
      "PLATAFORMA": 0,
      "ESTACIONAR": 0,
    },
  },
};

let _cr_scoreContext = _cr_scoreContext_DEFAULT;

const _cr_scoreContextBackup = JSON.parse(JSON.stringify(_cr_scoreContext));
console.log(JSON.stringify(_cr_scoreContext));

const MODE_ID_MAP = {
  1: "auto",
  2: "teleop",
  3: "endgame",
};

// Sends the foul to the server to add it to the list.
const addFoul = function(alliance, points) {
  websocket.send("addFoul", {Alliance: alliance, Points: points});
}

// Sends a websocket message to signal to the teams that they may enter the field.
var signalReset = function() {
  websocket.send("signalReset");
};

// Signals the scorekeeper that foul entry is complete for this match.
var commitMatch = function() {
  websocket.send("commitMatch");
};

// Sends a websocket message to update the realtime score
var updateRealtimeScore = function() {
  console.log("SCORE_UPDATE", {
    blueAuto: parseInt($("#blueAutoScore").val()),
    redAuto: parseInt($("#redAutoScore").val()),
    blueTeleop: parseInt($("#blueTeleopScore").val()),
    redTeleop: parseInt($("#redTeleopScore").val()),
    blueEndgame: parseInt($("#blueEndgameScore").val()),
    redEndgame: parseInt($("#redEndgameScore").val()),
  });
  
  websocket.send("updateRealtimeScore", {
    blueAuto: parseInt($("#blueAutoScore").val()),
    redAuto: parseInt($("#redAutoScore").val()),
    blueTeleop: parseInt($("#blueTeleopScore").val()),
    redTeleop: parseInt($("#redTeleopScore").val()),
    blueEndgame: parseInt($("#blueEndgameScore").val()),
    redEndgame: parseInt($("#redEndgameScore").val())
  });
};

var addBluePoints = function(actionKey, points, modeId, ref) {
  // console.log("blue", points, modeId);

  if (points < 0 && _cr_scoreContext.blue[MODE_ID_MAP[modeId]][actionKey] === 0) return;
  _cr_scoreContext.blue[MODE_ID_MAP[modeId]][actionKey] += points / Math.abs(points);
  ref.value = _cr_scoreContext.blue[MODE_ID_MAP[modeId]][actionKey];
  websocket.send("addBluePoints", {
    points,
    modeId,
  });
  websocket.send("CR_SCORE_CONTEXT", JSON.stringify(_cr_scoreContext));
};

var addRedPoints = function(actionKey, points, modeId, ref) {
  // console.log("red", points, modeId);

  if (points < 0 && _cr_scoreContext.red[MODE_ID_MAP[modeId]][actionKey] === 0) return;
  _cr_scoreContext.red[MODE_ID_MAP[modeId]][actionKey] += points / Math.abs(points);
  ref.value = _cr_scoreContext.red[MODE_ID_MAP[modeId]][actionKey];
  websocket.send("addRedPoints", {
    points,
    modeId,
  });
  websocket.send("CR_SCORE_CONTEXT", JSON.stringify(_cr_scoreContext));
};

const loadContext = function() {
  // console.log($("cr-curr-value-score-auto-ZONA_BAIXA"), _cr_scoreContext.blue.auto.ZONA_BAIXA);
  
  $("#cr-curr-value-score-auto-ZONA_BAIXA").prop("value", _cr_scoreContext.blue.auto.ZONA_BAIXA);
  $("#cr-curr-value-score-auto-PLATAFORMA").prop("value", _cr_scoreContext.blue.auto.PLATAFORMA);
  $("#cr-curr-value-score-auto-SAIU_AREA").prop("value", _cr_scoreContext.blue.auto.SAIU_AREA);
  $("#cr-curr-value-score-teleop-ZONA_BAIXA").prop("value", _cr_scoreContext.blue.teleop.ZONA_BAIXA);
  $("#cr-curr-value-score-teleop-ZONA_ALTA").prop("value", _cr_scoreContext.blue.teleop.ZONA_ALTA);
  $("#cr-curr-value-score-endgame-PLATAFORMA").prop("value", _cr_scoreContext.blue.endgame.PLATAFORMA);
  $("#cr-curr-value-score-endgame-ESTACIONAR").prop("value", _cr_scoreContext.blue.endgame.ESTACIONAR);
  $("#cr-curr-value-red-score-auto-ZONA_BAIXA").prop("value", _cr_scoreContext.red.auto.ZONA_BAIXA);
  $("#cr-curr-value-red-score-auto-PLATAFORMA").prop("value", _cr_scoreContext.red.auto.PLATAFORMA);
  $("#cr-curr-value-red-score-auto-SAIU_AREA").prop("value", _cr_scoreContext.red.auto.SAIU_AREA);
  $("#cr-curr-value-red-score-teleop-ZONA_BAIXA").prop("value", _cr_scoreContext.red.teleop.ZONA_BAIXA);
  $("#cr-curr-value-red-score-teleop-ZONA_ALTA").prop("value", _cr_scoreContext.red.teleop.ZONA_ALTA);
  $("#cr-curr-value-red-score-endgame-PLATAFORMA").prop("value", _cr_scoreContext.red.endgame.PLATAFORMA);
  $("#cr-curr-value-red-score-endgame-ESTACIONAR").prop("value", _cr_scoreContext.red.endgame.ESTACIONAR);
}

const resetContextDisplay = function() {
  // console.log($("#cr-curr-value-score-auto-ZONA_BAIXA"));

  _cr_scoreContext = _cr_scoreContext_DEFAULT;

  websocket.send("CR_SCORE_CONTEXT", JSON.stringify(_cr_scoreContext));
  
  $("#cr-curr-value-score-auto-ZONA_BAIXA").prop("value", "0");
  $("#cr-curr-value-score-auto-PLATAFORMA").prop("value", "0");
  $("#cr-curr-value-score-auto-SAIU_AREA").prop("value", "0");
  $("#cr-curr-value-score-teleop-ZONA_BAIXA").prop("value", "0");
  $("#cr-curr-value-score-teleop-ZONA_ALTA").prop("value", "0");
  $("#cr-curr-value-score-endgame-PLATAFORMA").prop("value", "0");
  $("#cr-curr-value-score-endgame-ESTACIONAR").prop("value", "0");
  $("#cr-curr-value-red-score-auto-ZONA_BAIXA").prop("value", "0");
  $("#cr-curr-value-red-score-auto-PLATAFORMA").prop("value", "0");
  $("#cr-curr-value-red-score-auto-SAIU_AREA").prop("value", "0");
  $("#cr-curr-value-red-score-teleop-ZONA_BAIXA").prop("value", "0");
  $("#cr-curr-value-red-score-teleop-ZONA_ALTA").prop("value", "0");
  $("#cr-curr-value-red-score-endgame-PLATAFORMA").prop("value", "0");
  $("#cr-curr-value-red-score-endgame-ESTACIONAR").prop("value", "0");
}

const handleContext = function(data) {
  console.log(data)
  
  if (data == null) {
    return _cr_scoreContext = _cr_scoreContext_DEFAULT;
  }
  
  try {
    _cr_scoreContext = JSON.parse(data);
    
    loadContext();

    console.log("Contexto carregado", _cr_scoreContext);
    
  } catch (error) {
    console.error("Erro ao carregar o contexto", error)
  }
};

// Handles a websocket message to update the teams for the current match.
var handleMatchLoad = function(data) {
  // Toda que vez que uma partida nova é carregada as pontuações vão a '0'
  resetContextDisplay();

  console.log("PARTIDA CARREGADA");

  $("#matchName").text(data.Match.LongName);
};

// Handles a websocket message to update the match status.
const handleMatchTime = function(data) {
  console.log(data);
  if(data.MatchState == 3 || data.MatchState == 4) {
    $(".cr-btn-score-auto").attr("disabled", false);
  } else if(data.MatchState == 5) { 
    $(".cr-btn-score-teleop").attr("disabled", false);
  } else if(data.MatchState == 6) {
    $(".cr-btn-score-endgame").attr("disabled", false);
  } else{
    $(".cr-btn-score").attr("disabled", true);
  }
  translateMatchTime(data, function(matchState, matchStateText, countdownSec) {
    $("#matchState").text(matchStateText);
    // minutes = Math.floor(countdownSec / 60);
    // seconds = countdownSec % 60;
    formattedTime = new Date(countdownSec * 1000).toLocaleTimeString().slice(3).replace("PM", "").trim();
    $("#matchTime").text(formattedTime);
  });
  $(".control-button").attr("data-enabled", matchStates[data.MatchState] === "POST_MATCH");
};

// Handles a websocket message to update the match score.
var handleRealtimeScore = function(data) {
  $("#redScore").text(data.Red.ScoreSummary.Score);
  $("#blueScore").text(data.Blue.ScoreSummary.Score);
  if (parseInt($("#redAutoScore").val()) != data.Red.Score.AutoPoints) {
    $("#redAutoScore").val(data.Red.Score.AutoPoints);
  }
  if (parseInt($("#redTeleopScore").val()) != data.Red.Score.TeleopPoints) {
    $("#redTeleopScore").val(data.Red.Score.TeleopPoints);
  }
  if (parseInt($("#redEndgameScore").val()) != data.Red.Score.EndgamePoints) {
    $("#redEndgameScore").val(data.Red.Score.EndgamePoints);
  }
  if (parseInt($("#blueAutoScore").val()) != data.Blue.Score.AutoPoints) {
    $("#blueAutoScore").val(data.Blue.Score.AutoPoints);
  }
  if (parseInt($("#blueTeleopScore").val()) != data.Blue.Score.TeleopPoints) {
    $("#blueTeleopScore").val(data.Blue.Score.TeleopPoints);
  }
  if (parseInt($("#blueEndgameScore").val()) != data.Blue.Score.EndgamePoints) {
    $("#blueEndgameScore").val(data.Blue.Score.EndgamePoints);
  }
}

// Handles a websocket message to update the scoring commit status.
const handleScoringStatus = function(data) {
  if (data.RefereeScoreReady) {
    $("#commitButton").attr("data-enabled", false);
  }
  $("#redScoreStatus").text("Red Scoring " + data.NumRedScoringPanelsReady + "/" + data.NumRedScoringPanels);
  $("#redScoreStatus").attr("data-ready", data.RedScoreReady);
  $("#blueScoreStatus").text("Blue Scoring " + data.NumBlueScoringPanelsReady + "/" + data.NumBlueScoringPanels);
  $("#blueScoreStatus").attr("data-ready", data.BlueScoreReady);

  // loadContext();
}

// Produces a hash code of the given object for use in equality comparisons.
const hashObject = function(object) {
  const s = JSON.stringify(object);
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = Math.imul(31, h) + s.charCodeAt(i) | 0;
  }
  return h;
}

$(function() {
  // Read the configuration for this display from the URL query string.
  var urlParams = new URLSearchParams(window.location.search);
  $(".headRef-dependent").attr("data-hr", urlParams.get("hr"));

  // Set up the websocket back to the server.
  websocket = new CheesyWebsocket("/referee/websocket", {
    matchLoad: function(event) { handleMatchLoad(event.data); },
    matchTiming: function(event) { handleMatchTiming(event.data); },
    matchTime: function(event) { handleMatchTime(event.data); },
    realtimeScore: function(event) { handleRealtimeScore(event.data); },
    scoringStatus: function(event) { handleScoringStatus(event.data); },
    pointsContext: function(event) { handleContext(event.data); }
  });
});

// Dynamic FLL scoring (built from window.FLLC_CRITERIA)
(function(){
  const buildFllUi = function() {
    if (!window.isFLL || !Array.isArray(window.FLLC_CRITERIA)) return;
    const container = document.getElementById('fll-scoring');
    if (!container) return;

    // Local state
    const state = {};

    const applyConstraints = () => {
      // Handle requiresTrue dependencies
      window.FLLC_CRITERIA.forEach(c => {
        if (c.requiresTrue && c.requiresTrue.length && c.type === 'yesno') {
          const enabled = c.requiresTrue.every(dep => !!state[dep]);
          const el = document.getElementById('crit-' + c.id);
          if (el) {
            el.disabled = !enabled;
            if (!enabled) { state[c.id] = false; el.checked = false; }
          }
        }
      });
    };

    const computeScore = () => {
      let total = 0;
      window.FLLC_CRITERIA.forEach(c => {
        const val = state[c.id];
        if (c.type === 'yesno') {
          if (val) total += (c.pointsYes || 0);
        } else if (c.type === 'count') {
          const n = Math.max(c.min||0, Math.min(c.max||0, parseInt(val||0)));
          total += n * (c.pointsPer || 0);
        } else if (c.type === 'countMap') {
          const n = Math.max(c.min||0, Math.min(c.max||0, parseInt(val||0)));
          if (c.map && (n in c.map)) total += c.map[n];
        } else if (c.type === 'enum') {
          // no points
        }
      });
      // Push to arena as Blue score only (FLL single side)
      try {
        $("#blueAutoScore").val(0);
        $("#blueTeleopScore").val(total);
        $("#blueEndgameScore").val(0);
        websocket.send("updateRealtimeScore", {
          blueAuto: 0,
          redAuto: 0,
          blueTeleop: total,
          redTeleop: 0,
          blueEndgame: 0,
          redEndgame: 0
        });
      } catch(e) {}
    };

    const renderYesNo = (c) => {
      const wrap = document.createElement('div');
      wrap.className = 'panel panel-default';
      wrap.innerHTML = `<div class="panel-heading"><b>${c.label}</b></div>
        <div class="panel-body">
          <label style="font-size: 24px; color:#444;display:flex;gap:8px;align-items:center;">
            <input type="checkbox" class="cr-checkbox" id="crit-${c.id}" /> Sim
          </label>
        </div>`;
      const input = wrap.querySelector('input');
      input.addEventListener('change', () => { state[c.id] = !!input.checked; applyConstraints(); computeScore(); });
      state[c.id] = false;
      return wrap;
    };

    const renderCount = (c) => {
      const wrap = document.createElement('div');
      wrap.className = 'panel panel-default';
      wrap.innerHTML = `<div class="panel-heading"><b>${c.label}</b></div>
        <div class="panel-body" style="display:flex;gap:8px;align-items:center;">
          <button class="cr-btn-score" data-score-type="down">-</button>
          <input class="cr-text-input-score-points" type="number" min="${c.min||0}" max="${c.max||0}" value="0" id="crit-${c.id}"/>
          <button class="cr-btn-score" data-score-type="up">+</button>
          <span style="color:#666;font-size:1em;font-weight: 600;">/ ${c.max||0}</span>
        </div>`;
      const [btnDec, input, btnInc] = wrap.querySelectorAll('button, input');
      const sync = () => { let v = parseInt(input.value||0); if (isNaN(v)) v=0; v = Math.max(c.min||0, Math.min(c.max||0, v)); input.value = v; state[c.id]=v; computeScore(); };
      btnDec.addEventListener('click', () => { input.value = Math.max((c.min||0), (parseInt(input.value||0)-1)); sync(); });
      btnInc.addEventListener('click', () => { input.value = Math.min((c.max||0), (parseInt(input.value||0)+1)); sync(); });
      input.addEventListener('input', sync);
      state[c.id] = 0;
      return wrap;
    };

    const renderCountMap = (c) => {
      const wrap = document.createElement('div');
      wrap.className = 'panel panel-default';
      wrap.innerHTML = `<div class="panel-heading"><b>${c.label}</b></div>
        <div class="panel-body" style="display:flex;gap:8px;align-items:center;">
          <input type="range" min="${c.min||0}" max="${c.max||0}" value="0" id="crit-${c.id}" />
          <span id="crit-${c.id}-val">0</span>
        </div>`;
      const input = wrap.querySelector('input');
      const span = wrap.querySelector('span');
      input.addEventListener('input', () => { state[c.id] = parseInt(input.value||0); span.textContent = input.value; computeScore(); });
      state[c.id] = 0; span.textContent = '0';
      return wrap;
    };

    const renderEnum = (c) => {
      const wrap = document.createElement('div');
      wrap.className = 'panel panel-default';
      const options = (c.options||[]).map(o => `<option value="${o.key}">${o.label}</option>`).join('');
      wrap.innerHTML = `<div class="panel-heading"><b>${c.label}</b></div>
        <div class="panel-body">
          <select id="crit-${c.id}" class="form-control">${options}</select>
        </div>`;
      const sel = wrap.querySelector('select');
      sel.addEventListener('change', () => { state[c.id] = sel.value; computeScore(); });
      state[c.id] = (c.options && c.options.length) ? c.options[0].key : null;
      return wrap;
    };

    window.fllResetScoring = function(){ Object.keys(state).forEach(k => state[k] = 0); document.querySelectorAll('#fll-scoring input, #fll-scoring select').forEach(el => {
      if (el.type === 'checkbox') el.checked = false;
      else if (el.type === 'number' || el.type === 'range') el.value = 0;
      else if (el.tagName === 'SELECT') el.selectedIndex = 0;
    }); computeScore(); };

    // Render all
    window.FLLC_CRITERIA.forEach(c => {
      let el;
      if (c.type === 'yesno') el = renderYesNo(c);
      else if (c.type === 'count') el = renderCount(c);
      else if (c.type === 'countMap') el = renderCountMap(c);
      else if (c.type === 'enum') el = renderEnum(c);
      if (el) container.appendChild(el);
    });

    applyConstraints();
    computeScore();
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', buildFllUi);
  } else {
    buildFllUi();
  }
})();

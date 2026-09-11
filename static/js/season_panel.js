// Draws the scoring panel from the season document. Nothing here knows what anything is worth: a
// click sends one message and the server answers with the score it computed.

var seasonPanel = (function() {
  var season = null;
  var level = "";
  var robots = { red: 3, blue: 3 };
  var state = { red: emptyTally(), blue: emptyTally() };
  var send = function() {};
  var period = "";

  function emptyTally() {
    return { Actions: {}, Adjust: {}, Slots: {} };
  }

  function tallyKey(id, inPeriod) {
    return inPeriod ? id + "@" + inPeriod : id;
  }

  function scoringPeriods() {
    return (season.periods || []).filter(function(p) { return p.scoring; });
  }

  function periodOf(action) {
    if (!action.periods || action.periods.length === 0) {
      return "";
    }
    if (action.periods.indexOf(period) >= 0) {
      return period;
    }
    return action.periods[action.periods.length - 1];
  }

  function slotCount(group, alliance) {
    if (group.slots === "robots" || group.slots === '"robots"') {
      return robots[alliance] || 0;
    }
    var parsed = parseInt(group.slots, 10);
    return isNaN(parsed) ? 0 : parsed;
  }

  function optionOf(group, id) {
    return (group.options || []).find(function(option) { return option.id === id; });
  }

  function slotState(alliance, groupId, slot) {
    var slots = (state[alliance].Slots || {})[groupId] || [];
    return slots[slot - 1] || {};
  }

  function countOf(alliance, id, inPeriod) {
    return (state[alliance].Actions || {})[tallyKey(id, inPeriod)] || 0;
  }

  function adjustmentOf(alliance, id) {
    return (state[alliance].Adjust || {})[id] || 0;
  }

  function element(tag, className, text) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined && text !== null) node.textContent = text;
    return node;
  }

  function drawCounter(alliance, action) {
    var box = element("div", "sp-entry");
    box.appendChild(element("span", "sp-entry-label", action.label));
    var row = element("div", "sp-row");
    var inPeriod = periodOf(action);

    var minus = element("button", "sp-btn sp-btn-minus", "−");
    minus.type = "button";
    minus.onclick = function() {
      send({ alliance: alliance, action: action.id, period: inPeriod, delta: -1 });
    };
    var value = element("span", "sp-value", String(countOf(alliance, action.id, inPeriod)));
    var plus = element("button", "sp-btn sp-btn-plus", "+");
    plus.type = "button";
    plus.onclick = function() {
      send({ alliance: alliance, action: action.id, period: inPeriod, delta: 1 });
    };
    row.appendChild(minus);
    row.appendChild(value);
    row.appendChild(plus);
    box.appendChild(row);
    if (action.help) box.appendChild(element("span", "sp-help", action.help));
    return box;
  }

  function drawToggle(alliance, action) {
    var box = element("div", "sp-entry");
    var inPeriod = periodOf(action);
    var on = countOf(alliance, action.id, inPeriod) > 0;
    var button = element("button", "sp-toggle" + (on ? " sp-on" : ""), action.label);
    button.type = "button";
    button.onclick = function() {
      send({ alliance: alliance, action: action.id, period: inPeriod, delta: 1 });
    };
    box.appendChild(button);
    return box;
  }

  function drawFreeValue(alliance, action) {
    var box = element("div", "sp-entry");
    box.appendChild(element("span", "sp-entry-label", action.label));
    var row = element("div", "sp-row");
    var input = document.createElement("input");
    input.type = "number";
    input.className = "sp-number";
    input.value = String(adjustmentOf(alliance, action.id));
    var apply = element("button", "sp-btn", "Aplicar");
    apply.type = "button";
    apply.onclick = function() {
      var value = parseInt(input.value, 10) || 0;
      if (action.confirm && !window.confirm(action.label + ": aplicar " + value + "?")) return;
      send({ alliance: alliance, action: action.id, delta: value });
    };
    row.appendChild(input);
    row.appendChild(apply);
    box.appendChild(row);
    return box;
  }

  function drawFoul(alliance, action) {
    var inPeriod = periodOf(action);
    var button = element("button", "sp-foul sp-foul-" + alliance,
      action.label + " (" + countOf(alliance, action.id, inPeriod) + ")");
    button.type = "button";
    button.onclick = function() {
      send({ alliance: alliance, action: action.id, period: inPeriod, delta: 1 });
    };
    button.oncontextmenu = function(event) {
      event.preventDefault();
      send({ alliance: alliance, action: action.id, period: inPeriod, delta: -1 });
    };
    return button;
  }

  // A grid of branches: click to place in the period the clock is in, click again to take out. The
  // colour says which period put it there, which is what the referee has to be able to see.
  function drawSlotGrid(alliance, group) {
    var box = element("div", "sp-entry sp-group");
    box.appendChild(element("span", "sp-entry-label", group.label));
    if (group.help) box.appendChild(element("span", "sp-help", group.help));
    var grid = element("div", "sp-grid");
    var total = slotCount(group, alliance);
    for (var slot = 1; slot <= total; slot++) {
      grid.appendChild(drawSlotCell(alliance, group, slot));
    }
    box.appendChild(grid);
    return box;
  }

  function drawSlotCell(alliance, group, slot) {
    var current = slotState(alliance, group.id, slot);
    var taken = !!current.occupant;
    var cell = element("button", "sp-slot" + (taken ? " sp-slot-on sp-slot-" + (current.occupiedIn || "") : ""),
      String(slot));
    cell.type = "button";
    if (taken) {
      var option = optionOf(group, current.occupant);
      cell.title = (option ? option.label : current.occupant) + " · " + (current.occupiedIn || "");
    }
    cell.onclick = function() {
      var first = (group.options || [])[0];
      send({
        alliance: alliance, group: group.id, slot: slot,
        option: taken ? "" : (first ? first.id : ""),
      });
    };
    return cell;
  }

  // One column per robot, one button per option: marking another option replaces the first, because
  // the slot holds one occupant. That is the rule, not a ceiling somebody has to respect.
  function drawSlotRobots(alliance, group) {
    var box = element("div", "sp-entry sp-group");
    box.appendChild(element("span", "sp-entry-label", group.label));
    if (group.help) box.appendChild(element("span", "sp-help", group.help));
    var total = slotCount(group, alliance);
    for (var slot = 1; slot <= total; slot++) {
      box.appendChild(drawRobotRow(alliance, group, slot));
    }
    return box;
  }

  function drawRobotRow(alliance, group, slot) {
    var current = slotState(alliance, group.id, slot);
    var row = element("div", "sp-robot-row");
    row.appendChild(element("span", "sp-robot-name", "Robô " + slot));
    (group.options || []).forEach(function(option) {
      var on = current.occupant === option.id;
      var button = element("button", "sp-choice" + (on ? " sp-on" : ""), option.short || option.label);
      button.type = "button";
      button.title = option.label;
      button.onclick = function() {
        send({ alliance: alliance, group: group.id, slot: slot, option: on ? "" : option.id });
      };
      row.appendChild(button);
    });
    return row;
  }

  function drawAlliance(alliance) {
    var column = element("div", "sp-alliance sp-alliance-" + alliance);
    column.appendChild(element("h4", "sp-alliance-title",
      alliance === "red" ? "Aliança Vermelha" : "Aliança Azul"));

    (season.slotGroups || []).forEach(function(group) {
      var layout = group.layout === "robots" || group.slots === "robots" ? "robots" : "grid";
      column.appendChild(layout === "robots"
        ? drawSlotRobots(alliance, group)
        : drawSlotGrid(alliance, group));
    });

    (season.actions || []).forEach(function(action) {
      if (action.hidden || action.kind === "foul" || action.kind === "adjustment") return;
      if (action.unit === "toggle") {
        column.appendChild(drawToggle(alliance, action));
      } else if (action.unit === "freeValue") {
        column.appendChild(drawFreeValue(alliance, action));
      } else {
        column.appendChild(drawCounter(alliance, action));
      }
    });
    return column;
  }

  function drawFouls(alliance) {
    var column = element("div", "sp-alliance sp-alliance-" + alliance);
    column.appendChild(element("h4", "sp-alliance-title",
      alliance === "red" ? "Faltas da Vermelha" : "Faltas da Azul"));
    (season.actions || []).forEach(function(action) {
      if (action.kind === "foul") {
        column.appendChild(drawFoul(alliance, action));
      } else if (action.kind === "adjustment") {
        column.appendChild(drawFreeValue(alliance, action));
      }
    });
    return column;
  }

  function render() {
    var host = document.getElementById("seasonPanel");
    var foulHost = document.getElementById("seasonFouls");
    if (!host) return;
    host.innerHTML = "";
    if (foulHost) foulHost.innerHTML = "";
    if (!season) {
      host.appendChild(element("p", "sp-empty",
        "Este evento não tem temporada configurada. Escolha uma em Configurações."));
      return;
    }
    host.appendChild(drawAlliance("red"));
    host.appendChild(drawAlliance("blue"));
    if (foulHost) {
      foulHost.appendChild(drawFouls("red"));
      foulHost.appendChild(drawFouls("blue"));
    }
  }

  function renderOutcome(alliance, outcome) {
    var host = document.getElementById(alliance + "Outcome");
    if (!host || !outcome) return;
    host.innerHTML = "";
    (season ? season.categories : []).forEach(function(category) {
      var value = (outcome.Categories || {})[category.id] || 0;
      var line = element("div", "sp-cat");
      line.appendChild(element("span", "sp-cat-label", category.label));
      line.appendChild(element("span", "sp-cat-value", String(value)));
      host.appendChild(line);
    });
    var earned = outcome.RpEligible || [];
    ((season && season.ranking && season.ranking.rankingPoints) || []).forEach(function(rp) {
      var got = earned.indexOf(rp.id) >= 0;
      var line = element("div", "sp-rp" + (got ? " sp-on" : ""));
      line.appendChild(element("span", "sp-cat-label", rp.label));
      line.appendChild(element("span", "sp-cat-value", got ? "ganho" : "—"));
      host.appendChild(line);
    });
  }

  return {
    setSender: function(sender) { send = sender; },
    setPeriod: function(value) {
      if (value === period) return;
      period = value;
    },
    onMatchLoad: function(data) {
      season = data.Season || null;
      level = data.EventLevel || "";
      robots.red = data.RedRobots || 0;
      robots.blue = data.BlueRobots || 0;
      state.red = emptyTally();
      state.blue = emptyTally();
      render();
    },
    onRealtimeScore: function(data) {
      if (!data) return;
      state.red = data.Red && data.Red.Score ? data.Red.Score : emptyTally();
      state.blue = data.Blue && data.Blue.Score ? data.Blue.Score : emptyTally();
      render();
      renderOutcome("red", data.Red && data.Red.ScoreSummary && data.Red.ScoreSummary.Outcome);
      renderOutcome("blue", data.Blue && data.Blue.ScoreSummary && data.Blue.ScoreSummary.Outcome);
    },
    hasSeason: function() { return !!season; },
    level: function() { return level; },
  };
})();

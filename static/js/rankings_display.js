// Copyright 2014 Team 254. All Rights Reserved.
// Author: nick@team254.com (Nick Eyre)
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side methods for the rankings display.

var websocket;
var initialDwellMs = 3000;  // How long the display waits upon initial load before scrolling.
var scrollMsPerRow;  // How long in milliseconds it takes to scroll a height of one row.
var staticUpdateIntervalMs = 10000;  // How long between updates if not scrolling.
var standingsTemplate; // assigned at runtime based on mode
var standingsHeaderTemplate;
var fllStandingsTemplate; // only in FLL mode
var rankingsData;
var prevHighestPlayedMatch;

function getIsFLL() {
  var body = document.body;
  if (!body) return false;
  var attr = body.getAttribute('data-is-fll');
  // Go boolean truthy check against "true" string.
  return String(attr).toLowerCase() === 'true';
}

// Loads the JSON rankings data from the event server.
var getRankingsData = function(callback) {
  if (getIsFLL()) {
    $.getJSON(arenaUrl("/api/fll/scores")).done(function(scoreData) {
      var list = (scoreData || []).map(function(s) {
        var rounds = (s.Rounds && s.Rounds.length ? s.Rounds : [0,0,0]);
        var nick = s.Nickname || "";
        var name = s.Name || "";
        var best = (typeof s.Best === 'number' ? s.Best : 0);
        if (s.OfficialRound && s.OfficialRound >= 1 && s.OfficialRound <= 3) {
          best = rounds[s.OfficialRound - 1] || 0;
        }
        return {
          TeamId: s.TeamId,
          Nickname: nick || name,
          FLLRounds: rounds,
          FLLBest: best,
          R1: rounds[0] || 0,
          R2: rounds[1] || 0,
          R3: rounds[2] || 0
        };
      });
      list.sort(function(a, b) {
        if (b.FLLBest !== a.FLLBest) return b.FLLBest - a.FLLBest;
        return (a.TeamId || 0) - (b.TeamId || 0);
      });
      list.forEach(function(entry, idx) { entry.Rank = idx + 1; });
      rankingsData = { Rankings: list, Iteration: "", HighestPlayedMatch: "" };
      if (callback) callback(rankingsData);
    }).fail(function() {
      // On failure, return empty list
      rankingsData = { Rankings: [], Iteration: "", HighestPlayedMatch: "" };
      if (callback) callback(rankingsData);
    });
    return; // prevent non-FLL path
  }

  $.getJSON(arenaUrl("/api/rankings"), function(data) {
    rankingsData = data;
    if (callback) {
      callback(rankingsData);
    }
  });
};

function finishFll(list) {
  list.sort(function(a, b) {
    if (b.FLLBest !== a.FLLBest) return b.FLLBest - a.FLLBest;
    return (a.TeamId || 0) - (b.TeamId || 0);
  });
  list.forEach(function(entry, idx) { entry.Rank = idx + 1; });
  rankingsData = { Rankings: list, Iteration: "", HighestPlayedMatch: "" };
  if (typeof arguments.callee.caller === 'function') {
    // noop
  }
}

// Updates the rankings in place and initiates scrolling if they are long enough to require it.
// The header comes from the same answer as the rows: the columns are the tiebreakers the season
// declared, and a season with five of them draws five.
var updateStandingsHeader = function() {
  if (getIsFLL() || !standingsHeaderTemplate) {
    return;
  }
  $("#standingsHeader").html(standingsHeaderTemplate(rankingsData));
};

var updateStaticRankings = function() {
  getRankingsData(function() {
    updateStandingsHeader();
    var template = getIsFLL() ? fllStandingsTemplate : standingsTemplate;
    var rankingsHtml = template(rankingsData);
    // Populate both tables so there is always visible content even if we don't scroll.
    $("#rankings1").html(rankingsHtml);
    $("#scroller").css("transform", "translate(0px, -2px);");
    prevHighestPlayedMatch = rankingsData.HighestPlayedMatch;
    setHighestPlayedMatch(rankingsData.HighestPlayedMatch);
    if ($("#rankings2").height() > $("#container").height()) {
      // Initiate scrolling.
      setTimeout(cycleRankings, initialDwellMs);
    } else {
      // Rankings are too short; just update in place.
      setTimeout(updateStaticRankings, staticUpdateIntervalMs);
    }
  });
};

// Seamlessly copies the newer table contents to the older one, resets the scrolling, and loads new data.
var cycleRankings = function() {
  // Overwrite the top data with the bottom data and reset the scrolling back up to the top of the top table.
  $("#rankings1").html($("#rankings2").html());
  $("#scroller").css({ transform: "translate(0px, -1px);" });

  // Load new data into the now out-of-sight bottom table.
  var template = getIsFLL() ? fllStandingsTemplate : standingsTemplate;
  var rankingsHtml = template(rankingsData);
  $("#rankings2").html(rankingsHtml);

  // Delay updating the "Standings as of" message by one cycle because the tables are always one cycle behind
  // the data loading.
  setHighestPlayedMatch(prevHighestPlayedMatch);
  prevHighestPlayedMatch = rankingsData.HighestPlayedMatch;

  if ($("#rankings1").height() > $("#container").height()) {
    // Kick off another scrolling animation.
    var scrollDistance = $("#rankings1").height() + parseInt($("#rankings1").css("border-bottom-width"));
    var scrollTime = scrollMsPerRow * $("#rankings1 tr").length;
    $("#scroller").transition({y: -scrollDistance}, scrollTime, "linear", cycleRankings);

    // Set the data to be reloaded two seconds before the scrolling terminates.
    var reloadDataTime = Math.max(0, scrollTime - 2000);
    setTimeout(getRankingsData, reloadDataTime);
  } else {
    // The rankings got shorter for whatever reason, so revert to static updating.
    setTimeout(updateStaticRankings, staticUpdateIntervalMs);
  }
};

// Updates the "Standings as of" message with the given value, or blanks it out if there is no data yet.
var setHighestPlayedMatch = function(highestPlayedMatch) {
  if (getIsFLL()) {
    // Hide for FLL mode.
    $("#highestPlayedMatch").text("");
    return;
  }
  if (highestPlayedMatch === "") {
    $("#highestPlayedMatch").text("");
  } else {
    $("#highestPlayedMatch").text("Standings as of Qualification Match " + highestPlayedMatch);
  }
};

// Handles a websocket message to update the event status message.
var handleEventStatus = function(data) {
  $("#earlyLateMessage").text(data.EarlyLateMessage);
};

$(function() {
  // Read the configuration for this display from the URL query string.
  var urlParams = new URLSearchParams(window.location.search);
  scrollMsPerRow = urlParams.get("scrollMsPerRow");

  // Pick templates based on mode.
  if (getIsFLL()) {
    fllStandingsTemplate = Handlebars.compile($("#fllStandingsTemplate").html());
  }
  // Always compile default in case non-FLL.
  var defaultTemplateEl = $("#standingsTemplate");
  if (defaultTemplateEl.length) {
    standingsTemplate = Handlebars.compile(defaultTemplateEl.html());
  }
  var headerTemplateEl = $("#standingsHeaderTemplate");
  if (headerTemplateEl.length) {
    standingsHeaderTemplate = Handlebars.compile(headerTemplateEl.html());
  }

  // Set up the websocket back to the server. Used only for remote forcing of reloads.
  websocket = new CheesyWebsocket("/displays/rankings/websocket", {
    eventStatus: function(event) { handleEventStatus(event.data); },
  });

  updateStaticRankings();
});

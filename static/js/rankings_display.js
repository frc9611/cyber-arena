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
  $.getJSON("/api/rankings", function(data) {
    // In FLL mode, transform the rankings to show per-match results and Best.
    if (getIsFLL()) {
      var teams = (data && data.Rankings) ? data.Rankings : [];
      // Attempt to fetch per-team FLL rounds from remote sync endpoint (local proxy)
      $.getJSON("/api/fll/scores").done(function(scoreData) {
        var byId = {};
        (scoreData || []).forEach(function(s) { byId[s.TeamId] = s; });
        var fllRankings = teams.map(function(t) {
          var s = byId[t.TeamId] || { Rounds: [0,0,0], Best: 0 };
          return {
            TeamId: t.TeamId,
            Nickname: t.Nickname,
            FLLRounds: (s.Rounds && s.Rounds.length ? s.Rounds : [0,0,0]),
            FLLBest: (typeof s.Best === 'number' ? s.Best : 0)
          };
        });
        fllRankings.sort(function(a, b) {
          if (b.FLLBest !== a.FLLBest) return b.FLLBest - a.FLLBest;
          return (a.TeamId || 0) - (b.TeamId || 0);
        });
        fllRankings.forEach(function(entry, idx) { entry.Rank = idx + 1; });
        rankingsData = {
          Rankings: fllRankings,
          Iteration: data.Iteration || "",
          HighestPlayedMatch: ""
        };
        if (callback) callback(rankingsData);
      }).fail(function() {
        // Fallback to zeros if remote not available
        var fllRankings = teams.map(function(t) {
          return { TeamId: t.TeamId, Nickname: t.Nickname, FLLRounds: [0,0,0], FLLBest: 0 };
        });
        fllRankings.sort(function(a, b) {
          if (b.FLLBest !== a.FLLBest) return b.FLLBest - a.FLLBest;
          return (a.TeamId || 0) - (b.TeamId || 0);
        });
        fllRankings.forEach(function(entry, idx) { entry.Rank = idx + 1; });
        rankingsData = { Rankings: fllRankings, Iteration: data.Iteration || "", HighestPlayedMatch: "" };
        if (callback) callback(rankingsData);
      });
      return; // prevent calling callback twice
    } else {
      rankingsData = data;
    }
    if (callback) {
      callback(rankingsData);
    }
  });
};

// Updates the rankings in place and initiates scrolling if they are long enough to require it.
var updateStaticRankings = function() {
  getRankingsData(function() {
    var template = getIsFLL() ? fllStandingsTemplate : standingsTemplate;
    var rankingsHtml = template(rankingsData);
    $("#rankings2").html(rankingsHtml);
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

  // Set up the websocket back to the server. Used only for remote forcing of reloads.
  websocket = new CheesyWebsocket("/displays/rankings/websocket", {
    eventStatus: function(event) { handleEventStatus(event.data); },
  });

  updateStaticRankings();
});

var arenaStatus = {};
var arenaBootstrap = null;

function arenaPost(path, body, done) {
  $.ajax({
    url: arenaUrl(path),
    type: "POST",
    contentType: "application/json",
    data: JSON.stringify(body || {}),
    dataType: "json",
    success: function(data) { done(data); },
    error: function(xhr) {
      var payload = {};
      try { payload = JSON.parse(xhr.responseText); } catch (e) { payload = {}; }
      done({ ok: false, code: payload.error || "http_" + xhr.status,
             message: payload.message || "Não consegui falar com esta arena." });
    }
  });
}

function arenaShow(step) {
  $("#arenaBlocked, #arenaPanel, #arenaStep1, #arenaStep2, #arenaStep3, #arenaStep4, #arenaStep5").hide();
  if (step === "modo_incompativel") { $("#arenaBlocked").show(); return; }
  if (step === "senha") { $("#arenaStep1").show(); return; }
  if (step === "conectar") { $("#arenaStep2").show(); return; }
  if (step === "conferir") { $("#arenaStep3").show(); return; }
  if (step === "registrar") { $("#arenaStep4").show(); return; }
  if (step === "importar") { $("#arenaStep5").show(); return; }
  $("#arenaPanel").show();
}

function arenaRefresh(done) {
  $.getJSON(arenaUrl("/api/arena-master/status"))
    .done(function(data) {
      arenaStatus = data;
      arenaShow(data.step);
      arenaPaintPanel(data);
      if (done) { done(data); }
    })
    .fail(function(xhr) {
      if (xhr.status === 401) { window.location = arenaUrl("/login?redirect=") + encodeURIComponent(arenaUrl("/setup/arena")); return; }
      var said = xhr.responseJSON && xhr.responseJSON.message;
      $("#arenaError").text(said || "Perdi contato com esta arena. O cyber-arena parou?").show();
    });
}

function arenaPaintPanel(data) {
  $("#arenaEventName").text(data.eventName ? data.eventName + " (" + data.eventSlug + ")" : "—");
  $("#arenaMasterUrl").text(data.masterUrl || "—");
  $("#arenaClientName").text(data.clientName || "—");
  $("#arenaVenue").text(data.venueSlot ? data.venueLabel || ("local " + data.venueSlot) : "local único");
  $("#arenaTokenPrefix").text(data.tokenPrefix ? "ak_" + data.tokenPrefix + "…" : "—");
  $("#arenaLastSync").text(data.lastSyncAt || "nunca");
  $("#arenaRevision").text(data.revision ? String(data.revision) : "—");

  var state = $("#arenaState");
  if (data.lastError) {
    state.text("Com problema").removeClass().addClass("text-danger");
    $("#arenaError").text(data.lastError).show();
  } else if (data.syncEnabled) {
    state.text("Conectado e sincronizando").removeClass().addClass("text-success");
    $("#arenaError").hide();
  } else {
    state.text("Conectado, mas o envio está desligado").removeClass().addClass("text-warning");
    $("#arenaError").hide();
  }
}

function arenaSay(target, ok, message) {
  $(target).html("").append($("<div>")
    .addClass("alert " + (ok ? "alert-success" : "alert-danger"))
    .text(message));
}

$(function() {
  arenaShow($("#arenaWizard").data("step"));
  arenaRefresh();
  window.setInterval(function() { if ($("#arenaPanel").is(":visible")) { arenaRefresh(); } }, 5000);

  $("#arenaSavePassword").click(function() {
    $("#arenaPasswordError").text("");
    arenaPost("/api/arena-master/password",
      { password: $("#arenaPassword").val(), confirm: $("#arenaPasswordConfirm").val() },
      function(data) {
        if (!data.ok) { $("#arenaPasswordError").text(data.message); return; }
        arenaRefresh();
      });
  });

  $("#arenaConnect").click(function() { arenaConnect(false); });
  $("#arenaSaveOffline").click(function() { arenaConnect(true); });

  function arenaConnect(offline) {
    arenaPost("/api/arena-master/connect", {
      url: $("#arenaUrl").val(),
      token: $("#arenaToken").val(),
      expectedSlug: $("#arenaExpectedSlug").val(),
      saveAnyway: offline,
      allowInsecure: $("#arenaConfirmInsecure").val()
    }, function(data) {
      if (data.code === "http_publico") { $("#arenaInsecure").show(); }
      if (!data.ok) { arenaSay("#arenaConnectResult", false, data.message); return; }
      if (data.stage === "saved-offline") { arenaSay("#arenaConnectResult", true, data.message); return; }
      var text = "Evento " + data.eventName + " (" + data.eventSlug + ") — bate com o que você escreveu.";
      if (data.venueCount > 1) {
        text += " Este evento tem " + data.venueCount + " " + data.venueKindPlural +
          ". No próximo passo você escolhe qual é esta máquina.";
      }
      arenaSay("#arenaConnectResult", true, text);
      window.setTimeout(arenaRefresh, 900);
    });
  }

  $("#arenaCheck, #arenaCheckAgain").click(function() {
    arenaPost("/api/arena-master/check", {}, function(data) {
      if (!data.ok) { arenaSay("#arenaCheckResult", false, data.message); return; }
      arenaBootstrap = data.bootstrap;
      arenaPaintCheck(data);
      arenaPaintVenues(data.bootstrap);
    });
  });

  function arenaPaintCheck(data) {
    var event = data.bootstrap.event;
    var box = $("#arenaCheckResult").html("");
    var list = $("<dl>").addClass("dl-horizontal");
    function row(term, value) {
      list.append($("<dt>").text(term)).append($("<dd>").text(value));
    }
    row("Evento", event.name + " (" + event.slug + ")");
    row("Formato", event.tournamentTypeLabel);
    row("Locais", event.venue.count + " " + event.venue.kindLabelPlural + " · " + event.venue.scheduleModeLabel);
    row("Esta instância", data.bootstrap.instance.clientName || "sem nome");
    row("Equipes lá", String(event.counts.teams));
    row("Já enviado por ESTA máquina", event.countsFromHere.matches + " partidas, " +
        event.countsFromHere.teams + " equipes");
    box.append(list);

    var diff = data.teamDiff;
    box.append($("<p>").text("Equipes: " + diff.iguais.length + " iguais, " + diff.diferentes.length +
      " diferentes, " + diff.soAqui.length + " só nesta arena, " + diff.soLa.length + " só no Arena Master."));

    box.append($("<div>").addClass("alert alert-warning").text(
      "A partir do primeiro envio, o que estiver NESTA máquina substitui o que ESTA MÁQUINA mandou " +
      "antes — hoje, " + event.countsFromHere.matches + " partidas. O que as outras máquinas mandaram " +
      "não é tocado. Se este não é o computador que vai conduzir este local, pare aqui."));
    box.append($("<button>").addClass("btn btn-primary").text("Conferi: é este evento").click(function() {
      arenaRefresh();
    }));
  }

  function arenaPaintVenues(boot) {
    var venue = boot.event.venue;
    if (!venue.slots || venue.slots.length < 2) { $("#arenaVenuePick").hide(); return; }
    var options = $("#arenaVenueOptions").html("");
    $.each(venue.slots, function(index, slot) {
      if (!slot.active) { return; }
      var id = "arenaVenue" + slot.slot;
      var label = $("<label>").addClass("radio");
      label.append($("<input>").attr({ type: "radio", name: "arenaVenueSlot", id: id, value: slot.slot }));
      label.append(document.createTextNode(" " + slot.label + (slot.hint ? " — " + slot.hint : "")));
      options.append(label);
    });
    $("#arenaVenuePick").show();
  }

  $("#arenaRegister").click(function() {
    var slot = parseInt($("input[name=arenaVenueSlot]:checked").val(), 10) || 0;
    var label = $("input[name=arenaVenueSlot]:checked").parent().text().trim();
    arenaPost("/api/arena-master/register", {
      clientName: $("#arenaClientNameInput").val(),
      publicUrl: $("#arenaPublicUrlInput").val(),
      venueSlot: slot,
      venueLabel: label,
      takeover: $("#arenaTakeover").is(":checked"),
      fingerprint: ""
    }, function(data) {
      if (!data.ok) {
        arenaSay("#arenaRegisterResult", false, data.message);
        if (data.canTakeover) { $("#arenaTakeoverBox").show(); }
        return;
      }
      arenaSay("#arenaRegisterResult", true, "Registrado. O envio automático está ligado.");
      window.setTimeout(arenaRefresh, 900);
    });
  });

  $("#arenaApply").click(function() {
    arenaPost("/api/arena-master/apply", {
      importTeams: $("#arenaImportTeams").is(":checked"),
      applySettings: $("#arenaApplySettings").is(":checked")
    }, function(data) {
      if (!data.ok) { arenaSay("#arenaApplyResult", false, data.message); return; }
      arenaSay("#arenaApplyResult", true, data.messages.join(" "));
      window.setTimeout(arenaRefresh, 900);
    });
  });

  $("#arenaSkipApply").click(function() {
    arenaPost("/api/arena-master/apply", { importTeams: false, applySettings: false }, function() {
      arenaRefresh();
    });
  });

  $("#arenaSyncNow").click(function() {
    arenaPost("/api/arena-master/sync-now", {}, function(data) {
      arenaSay("#arenaError", data.ok, data.message);
      $("#arenaError").show();
      arenaRefresh();
    });
  });

  $("#arenaDisconnect").click(function() {
    if (!window.confirm("Desconectar esta arena do Arena Master? O token deixa de valer aqui.")) { return; }
    arenaPost("/api/arena-master/reset", { revoke: true }, function() { arenaRefresh(); });
  });
});

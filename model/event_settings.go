// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model and datastore read/write methods for event-level configuration.

package model

import "github.com/Team254/cheesy-arena-lite/game"

type EventSettings struct {
	Id                          int `db:"id"`
	Name                        string
	ElimType                    string
	NumElimAlliances            int
	SelectionRound1Order        string
	SelectionRound2Order        string
	TeamDownloadOrigin          string // TBA, FTCScout, 'none'
	TbaPublishingEnabled        bool
	TbaEventCode                string
	TbaSecretId                 string
	TbaSecret                   string
	NetworkSecurityEnabled      bool
	ApAddress                   string
	ApUsername                  string
	ApPassword                  string
	ApTeamChannel               int
	ApAdminChannel              int
	ApAdminWpaKey               string
	Ap2Address                  string
	Ap2Username                 string
	Ap2Password                 string
	Ap2TeamChannel              int
	SwitchAddress               string
	SwitchPassword              string
	PlcAddress                  string
	AdminPassword               string
	WarmupDurationSec           int
	AutoDurationSec             int
	PauseDurationSec            int
	TeleopDurationSec           int
	WarningRemainingDurationSec int
	TeamsPerAlliance            int
	IsFll                       bool
	// Remote sync for multi-table tournaments (optional)
	RemoteSyncUrl     string
	RemoteSyncApiKey  string
	RemoteSyncClients string // Comma-separated list of client URLs (for master node)
	// Vernum Arena Master
	ArenaMode             string
	ArenaMasterUrl        string
	ArenaServerVerifiedAt string
	ArenaToken            string
	ArenaTokenPrefix      string
	ArenaExpectedSlug     string
	ArenaClientUid        string
	ArenaClientHost       string
	ArenaClientName       string
	ArenaPublicUrl        string
	ArenaInstanceId       int64
	ArenaEventSlug        string
	ArenaEventName        string
	ArenaVenueSlot        int
	ArenaVenueLabel       string
	ArenaVenueKind        string
	ArenaVenueKindPlural  string
	ArenaCheckedAt        string
	ArenaConfirmedAt      string
	ArenaBootstrappedAt   string
	ArenaBootstrapJson    string
	ArenaLocalFingerprint string
	ArenaIsAnchor         bool
	SeasonKey             string
	SeasonRevision        int
	SeasonHash            string
	SeasonBody            string
	EventLevel            string
	ArenaSyncEnabled      bool
	ArenaSyncSeconds      int
	ArenaSyncGeneration   int
	ArenaRevision         int64
	ArenaEventRevision    int64
	ArenaLastSyncAt       string
	ArenaLastError        string
	ArenaLastErrorCode    string
	ArenaLastErrorAt      string
}

func (database *Database) GetEventSettings() (*EventSettings, error) {
	allEventSettings, err := database.eventSettingsTable.getAll()
	if err != nil {
		return nil, err
	}
	if len(allEventSettings) == 1 {
		return &allEventSettings[0], nil
	}

	// Database record doesn't exist yet; create it now.
	eventSettings := EventSettings{
		Name:                        "Untitled Event",
		ElimType:                    "single",
		NumElimAlliances:            8,
		SelectionRound1Order:        "L",
		SelectionRound2Order:        "",
		TeamDownloadOrigin:          "none",
		ApTeamChannel:               157,
		ApAdminChannel:              0,
		ApAdminWpaKey:               "1234Five",
		Ap2TeamChannel:              0,
		WarmupDurationSec:           game.MatchTiming.WarmupDurationSec,
		AutoDurationSec:             game.MatchTiming.AutoDurationSec,
		PauseDurationSec:            game.MatchTiming.PauseDurationSec,
		TeleopDurationSec:           game.MatchTiming.TeleopDurationSec,
		WarningRemainingDurationSec: game.MatchTiming.WarningRemainingDurationSec,
		TeamsPerAlliance:            2,
		IsFll:                       false,
		RemoteSyncUrl:               "",
		RemoteSyncApiKey:            "",
		RemoteSyncClients:           "",
		SeasonKey:                   game.LegacySeasonKey,
		EventLevel:                  "",
	}

	if err := database.eventSettingsTable.create(&eventSettings); err != nil {
		return nil, err
	}
	return &eventSettings, nil
}

func (database *Database) UpdateEventSettings(eventSettings *EventSettings) error {
	return database.eventSettingsTable.update(eventSettings)
}

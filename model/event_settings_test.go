// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventSettingsReadWrite(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	eventSettings, err := db.GetEventSettings()
	assert.Nil(t, err)
	assert.Equal(
		t,
		EventSettings{
			Id:                          1,
			Name:                        "Untitled Event",
			ElimType:                    "single",
			NumElimAlliances:            8,
			SelectionRound1Order:        "L",
			SelectionRound2Order:        "",
			TeamDownloadOrigin:          "TBA",
			ApTeamChannel:               157,
			ApAdminChannel:              0,
			ApAdminWpaKey:               "1234Five",
			WarmupDurationSec:           0,
			AutoDurationSec:             15,
			PauseDurationSec:            2,
			TeleopDurationSec:           135,
			WarningRemainingDurationSec: 30,
		},
		*eventSettings,
	)

	eventSettings.Name = "Chezy Champs"
	eventSettings.NumElimAlliances = 6
	eventSettings.SelectionRound1Order = "F"
	eventSettings.SelectionRound2Order = "L"
	err = db.UpdateEventSettings(eventSettings)
	assert.Nil(t, err)
	eventSettings2, err := db.GetEventSettings()
	assert.Nil(t, err)
	assert.Equal(t, eventSettings, eventSettings2)
}

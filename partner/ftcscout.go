// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Methods for publishing data to and retrieving data from The Blue Alliance.

package partner

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	ftcScoutBaseUrl = "https://api.ftcscout.org/rest/v1"
)

type FTCScoutClient struct {
	BaseUrl string
}

type FTCScoutTeam struct {
	TeamNumber int    `json:"number"`
	Name       string `json:"schoolName"`
	Nickname   string `json:"name"`
	City       string `json:"city"`
	StateProv  string `json:"state"`
	Country    string `json:"country"`
	RookieYear int    `json:"rookieYear"`
}

func NewFtcScoutClient() *FTCScoutClient {
	return &FTCScoutClient{BaseUrl: ftcScoutBaseUrl}
}

func (client *FTCScoutClient) GetTeam(teamNumber int) (*FTCScoutTeam, error) {
	path := fmt.Sprintf("/teams/%d", teamNumber)
	resp, err := client.getRequest(path)
	if err != nil {
		return nil, err
	}

	// Get the response and handle errors
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var teamData FTCScoutTeam
	err = json.Unmarshal(body, &teamData)

	return &teamData, err
}

func (client *FTCScoutClient) DownloadTeamAvatar(teamNumber, year int) error {
	path := fmt.Sprintf("/api/v3/team/%s/media/%d", (teamNumber), year)
	resp, err := client.getRequest(path)
	if err != nil {
		return err
	}

	// Get the response and handle errors
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var mediaItems []*TbaMediaItem
	err = json.Unmarshal(body, &mediaItems)
	if err != nil {
		return err
	}

	for _, item := range mediaItems {
		if item.Type == "avatar" {
			base64String, ok := item.Details["base64Image"].(string)
			if !ok {
				return fmt.Errorf("Could not interpret avatar response from TBA: %v", item)
			}
			avatarBytes, err := base64.StdEncoding.DecodeString(base64String)
			if err != nil {
				return err
			}

			// Store the avatar to disk as a PNG file.
			avatarPath := fmt.Sprintf("%s/%d.png", AvatarsDir, teamNumber)
			ioutil.WriteFile(avatarPath, avatarBytes, 0644)
			return nil
		}
	}

	return fmt.Errorf("No avatar found for team %d in year %d.", teamNumber, year)
}

// Sends a GET request to the TBA API.
func (client *FTCScoutClient) getRequest(path string) (*http.Response, error) {
	url := client.BaseUrl + path

	// Make an HTTP GET request with the TBA auth headers.
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return httpClient.Do(req)
}

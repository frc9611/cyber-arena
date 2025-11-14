// Handlers for FLL Remote Sync Management page.
package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Team254/cheesy-arena-lite/model"
)

// Shows the remote sync management page (master node only).
func (web *Web) remoteSyncPageHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	template, err := web.parseFiles("templates/remote_sync.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
	}{web.arena.EventSettings}
	if err := template.ExecuteTemplate(w, "base", data); err != nil {
		handleWebErr(w, err)
		return
	}
}

// API handler to trigger global start match from the remote sync page.
func (web *Web) remoteSyncStartMatchHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}
	if !web.arena.EventSettings.IsFll {
		http.Error(w, "Only available in FLL mode", http.StatusBadRequest)
		return
	}
	if web.arena.EventSettings.RemoteSyncApiKey == "" {
		http.Error(w, "Remote sync API key not configured", http.StatusBadRequest)
		return
	}

	// Parse request body to get excludeMaster flag
	var requestData struct {
		ExcludeMaster bool `json:"excludeMaster"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		// Default to false if parsing fails (backward compatibility)
		requestData.ExcludeMaster = false
	}

	// Start match on master node only if not excluded
	if !requestData.ExcludeMaster {
		err := web.arena.StartMatch()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Broadcast to all configured client systems
	web.broadcastStartMatchToClients()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// API handler to check status of remote sync connection.
func (web *Web) remoteSyncStatusHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	status := map[string]interface{}{
		"isFll":         web.arena.EventSettings.IsFll,
		"remoteSyncUrl": web.arena.EventSettings.RemoteSyncUrl,
		"hasApiKey":     web.arena.EventSettings.RemoteSyncApiKey != "",
		"canStartMatch": web.arena.MatchState == 0, // PreMatch
		"currentMatch":  web.arena.CurrentMatch.DisplayName,
		"matchType":     web.arena.CurrentMatch.Type,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// API handler to get list of connected instances (for master node)
func (web *Web) remoteSyncInstancesHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	// Get list of configured client URLs
	instances := []map[string]interface{}{}

	// Add the master itself
	masterInfo := map[string]interface{}{
		"eventName":    web.arena.EventSettings.Name,
		"currentMatch": web.arena.CurrentMatch.DisplayName,
		"matchType":    web.arena.CurrentMatch.Type,
		"isMaster":     true,
		"status":       "online",
	}
	instances = append(instances, masterInfo)

	// Query each configured client for their info
	if web.arena.EventSettings.RemoteSyncClients != "" {
		clientUrls := strings.Split(web.arena.EventSettings.RemoteSyncClients, ",")
		for _, clientUrl := range clientUrls {
			clientUrl = strings.TrimSpace(clientUrl)
			if clientUrl == "" {
				continue
			}
			clientInfo := web.getClientInfo(clientUrl)
			if clientInfo != nil {
				instances = append(instances, clientInfo)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"instances": instances,
	})
}

// API endpoint to return this instance's info (for remote sync discovery)
func (web *Web) remoteSyncInfoHandler(w http.ResponseWriter, r *http.Request) {
	// Allow unauthenticated access for discovery
	if !web.arena.EventSettings.IsFll {
		http.NotFound(w, r)
		return
	}

	info := map[string]interface{}{
		"eventName":    web.arena.EventSettings.Name,
		"currentMatch": web.arena.CurrentMatch.DisplayName,
		"matchType":    web.arena.CurrentMatch.Type,
		"matchState":   web.arena.MatchState,
		"isFll":        web.arena.EventSettings.IsFll,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Helper function to broadcast start match command to all configured clients
func (web *Web) broadcastStartMatchToClients() {
	if web.arena.EventSettings.RemoteSyncClients == "" {
		return
	}

	clientUrls := strings.Split(web.arena.EventSettings.RemoteSyncClients, ",")
	for _, clientUrl := range clientUrls {
		clientUrl = strings.TrimSpace(clientUrl)
		if clientUrl == "" {
			continue
		}
		go web.sendStartMatchToClient(clientUrl)
	}
}

// Helper function to send start match command to a single client
func (web *Web) sendStartMatchToClient(clientUrl string) {
	req, err := http.NewRequest("POST", clientUrl+"/api/fll/start-match", nil)
	if err != nil {
		log.Printf("Error creating start-match request for %s: %v", clientUrl, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if web.arena.EventSettings.RemoteSyncApiKey != "" {
		req.Header.Set("X-API-Key", web.arena.EventSettings.RemoteSyncApiKey)
	}
	req.Header.Set("X-From-Remote", "1")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error broadcasting start-match to %s: %v", clientUrl, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		log.Printf("Client %s returned status %d for start-match", clientUrl, resp.StatusCode)
	}
}

// Helper function to get info from a client node
func (web *Web) getClientInfo(clientUrl string) map[string]interface{} {
	req, err := http.NewRequest("GET", clientUrl+"/api/remote-sync/info", nil)
	if err != nil {
		log.Printf("Error creating info request for %s: %v", clientUrl, err)
		return nil
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching info from %s: %v", clientUrl, err)
		return map[string]interface{}{
			"eventName":    "Unknown",
			"currentMatch": "Unknown",
			"matchType":    "unknown",
			"isMaster":     false,
			"status":       "offline",
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Client %s returned status %d for info", clientUrl, resp.StatusCode)
		return map[string]interface{}{
			"eventName":    "Unknown",
			"currentMatch": "Unknown",
			"matchType":    "unknown",
			"isMaster":     false,
			"status":       "error",
		}
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		log.Printf("Error decoding info from %s: %v", clientUrl, err)
		return map[string]interface{}{
			"eventName":    "Unknown",
			"currentMatch": "Unknown",
			"matchType":    "unknown",
			"isMaster":     false,
			"status":       "error",
		}
	}

	info["isMaster"] = false
	info["status"] = "online"
	return info
}

package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	ModeStandalone = "standalone"
	ModeLocal      = "local"
	ModeCloud      = "cloud"

	DefaultArenaMasterUrl = "https://server.frc9611.com"
	DefaultPort           = 9080
	DefaultDbPath         = "./event.db"
	DefaultSyncSeconds    = 30
)

type Config struct {
	Mode               string
	ModeFromEnv        bool
	Port               int
	DbPath             string
	BasePath           string
	MasterUrl          string
	Token              string
	TokenFromEnv       bool
	SyncSeconds        int
	VernumSsoUrl       string
	VernumApiUrl       string
	VernumClientId     string
	VernumClientSecret string
	VenueSlot          int
	VenueSlotFromEnv   bool
	VenueLabel         string
	FllRole            string
	FllKey             string
	FllMasterUrl       string
	FllClients         string
}

func Load() *Config {
	mode := normalizeMode(os.Getenv("ARENA_MODE"))
	config := &Config{
		Mode:               mode,
		ModeFromEnv:        mode != "",
		Port:               intOr("ARENA_PORT", DefaultPort),
		DbPath:             stringOr("ARENA_DB_PATH", DefaultDbPath),
		BasePath:           normalizeBasePath(os.Getenv("ARENA_BASE_PATH")),
		MasterUrl:          strings.TrimRight(strings.TrimSpace(os.Getenv("ARENA_MASTER_URL")), "/"),
		Token:              strings.TrimSpace(os.Getenv("ARENA_TOKEN")),
		SyncSeconds:        intOr("ARENA_SYNC_SECONDS", DefaultSyncSeconds),
		VernumSsoUrl:       strings.TrimRight(strings.TrimSpace(os.Getenv("VERNUM_SSO_URL")), "/"),
		VernumApiUrl:       strings.TrimRight(strings.TrimSpace(os.Getenv("VERNUM_API_URL")), "/"),
		VernumClientId:     strings.TrimSpace(os.Getenv("VERNUM_CLIENT_ID")),
		VernumClientSecret: strings.TrimSpace(os.Getenv("VERNUM_CLIENT_SECRET")),
		VenueSlot:          intOr("ARENA_VENUE_SLOT", 0),
		VenueLabel:         strings.TrimSpace(os.Getenv("ARENA_VENUE_LABEL")),
		FllRole:            strings.ToLower(strings.TrimSpace(os.Getenv("ARENA_FLL_ROLE"))),
		FllKey:             strings.TrimSpace(os.Getenv("ARENA_FLL_KEY")),
		FllMasterUrl:       strings.TrimSpace(os.Getenv("ARENA_FLL_MASTER_URL")),
		FllClients:         strings.TrimSpace(os.Getenv("ARENA_FLL_CLIENTS")),
	}
	config.TokenFromEnv = config.Token != ""
	config.VenueSlotFromEnv = strings.TrimSpace(os.Getenv("ARENA_VENUE_SLOT")) != ""
	if config.Mode == "" {
		config.Mode = ModeStandalone
	}
	return config
}

func (config *Config) IsFllMaster() bool {
	return config.FllRole == "master"
}

func (config *Config) FllClientList() []string {
	list := make([]string, 0)
	for _, raw := range strings.Split(config.FllClients, ",") {
		if url := strings.TrimRight(strings.TrimSpace(raw), "/"); url != "" {
			list = append(list, url)
		}
	}
	return list
}

func (config *Config) EffectiveMode(settingsMode string) string {
	if config.ModeFromEnv {
		return config.Mode
	}
	if normalizeMode(settingsMode) != "" {
		return normalizeMode(settingsMode)
	}
	return ModeStandalone
}

func (config *Config) IsStandalone(settingsMode string) bool {
	return config.EffectiveMode(settingsMode) == ModeStandalone
}

func (config *Config) IsLocal(settingsMode string) bool {
	return config.EffectiveMode(settingsMode) == ModeLocal
}

func (config *Config) IsCloud(settingsMode string) bool {
	return config.EffectiveMode(settingsMode) == ModeCloud
}

func ModeClass(mode string) string {
	switch mode {
	case ModeLocal:
		return "bg-cyber-local"
	case ModeCloud:
		return "bg-cyber-cloud"
	default:
		return "bg-cyber-purple"
	}
}

func ModeLabel(mode string) string {
	switch mode {
	case ModeLocal:
		return "Local"
	case ModeCloud:
		return "Nuvem"
	default:
		return ""
	}
}

func (config *Config) Path(path string) string {
	if config.BasePath == "" {
		return path
	}
	if path == "" || path == "/" {
		return config.BasePath + "/"
	}
	if !strings.HasPrefix(path, "/") {
		return config.BasePath + "/" + path
	}
	return config.BasePath + path
}

func normalizeMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ModeStandalone:
		return ModeStandalone
	case ModeLocal:
		return ModeLocal
	case ModeCloud:
		return ModeCloud
	default:
		return ""
	}
}

func normalizeBasePath(raw string) string {
	path := strings.TrimSpace(raw)
	if path == "" || path == "/" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}

func stringOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func intOr(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil {
		return fallback
	}
	return value
}

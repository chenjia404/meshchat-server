package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains all runtime configuration required by the service.
type Config struct {
	AppName            string
	LogLevel           string
	HTTPAddr           string
	AdminHTTPAddr      string
	ShutdownTimeout    time.Duration
	ServerMode         string
	ServerAdminPeerIDs []string
	AdminUsername      string
	AdminPassword      string
	DatabaseURL        string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	JWTSecret          string
	JWTIssuer          string
	JWTExpiration      time.Duration
	ChallengeTTL       time.Duration
	IPFSAPIURL             string
	IPFSGatewayUpstreamURL string
	IPFSGatewayBaseURL     string
	AutoMigrate        bool
	LegacyAPIRoot      bool
	WSWriteWait        time.Duration
	WSPongWait         time.Duration
	WSPingInterval     time.Duration
	WSSendBuffer       int
	OnlineTTL          time.Duration
}

// Load builds config from environment variables with sensible local defaults.
func Load() Config {
	return Config{
		AppName:            getEnv("APP_NAME", "meshchat-server"),
		LogLevel:           strings.ToLower(getEnv("LOG_LEVEL", "info")),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		AdminHTTPAddr:      getEnv("ADMIN_HTTP_ADDR", ":8081"),
		ShutdownTimeout:    getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		ServerMode:         strings.ToLower(getEnv("SERVER_MODE", "restricted")),
		ServerAdminPeerIDs: getEnvCSV("SERVER_ADMIN_PEER_IDS"),
		AdminUsername:      getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:      getEnv("ADMIN_PASSWORD", "admin123456"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://meshchat:meshchat@localhost:5432/meshchat?sslmode=disable"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("REDIS_DB", 0),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		JWTIssuer:          getEnv("JWT_ISSUER", "meshchat-server"),
		JWTExpiration:      getEnvDuration("JWT_EXPIRATION", 72*time.Hour),
		ChallengeTTL:       getEnvDuration("AUTH_CHALLENGE_TTL", 5*time.Minute),
		IPFSAPIURL:             getEnv("IPFS_API_URL", "http://localhost:5001"),
		IPFSGatewayUpstreamURL: getEnv("IPFS_GATEWAY_UPSTREAM", ""),
		IPFSGatewayBaseURL:     getEnv("IPFS_GATEWAY_BASE_URL", ""),
		AutoMigrate:        getEnvBool("AUTO_MIGRATE", true),
		LegacyAPIRoot:      getEnvBool("LEGACY_API_ROOT", true),
		WSWriteWait:        getEnvDuration("WS_WRITE_WAIT", 10*time.Second),
		WSPongWait:         getEnvDuration("WS_PONG_WAIT", 60*time.Second),
		WSPingInterval:     getEnvDuration("WS_PING_INTERVAL", 54*time.Second),
		WSSendBuffer:       getEnvInt("WS_SEND_BUFFER", 128),
		OnlineTTL:          getEnvDuration("ONLINE_TTL", 90*time.Second),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvCSV(key string) []string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, item := range parts {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

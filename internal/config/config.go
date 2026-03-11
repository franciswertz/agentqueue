package config

import (
    "os"
    "strconv"
    "strings"
    "time"
)

type Config struct {
    MQTTBrokerURL     string
    MQTTUsername      string
    MQTTPassword      string
    MQTTClientID      string
    MQTTEnqueueTopic  string
    MQTTCompleteTopic string
    MQTTQoS           byte

    DBDSN string

    WorkerPollInterval time.Duration
    MaxAttempts        int
    JobLockTimeout     time.Duration

	OpenCodeCmd     string
	OpenCodeArgs    []string
	OpenCodeTimeout time.Duration
	OpenCodeDir     string
	OpenCodeMaxOutputBytes int
	OpenCodeDebugMem bool
	OpenCodeDebugArgs bool

    HTTPAddr string
}

func Load() Config {
	clientID := getenv("MQTT_CLIENT_ID", "agentq-worker")
	if strings.HasSuffix(clientID, "-") {
		if hostname, err := os.Hostname(); err == nil && hostname != "" {
			clientID = clientID + hostname
		}
	}

	return Config{
		MQTTBrokerURL:     getenv("MQTT_BROKER_URL", "tcp://localhost:1883"),
		MQTTUsername:      getenv("MQTT_USERNAME", ""),
		MQTTPassword:      getenv("MQTT_PASSWORD", ""),
		MQTTClientID:      clientID,
		MQTTEnqueueTopic:  getenv("MQTT_ENQUEUE_TOPIC", "jobs/enqueue/+"),
		MQTTCompleteTopic: getenv("MQTT_COMPLETE_TOPIC", "jobs/complete/{app_id}"),
		MQTTQoS:           byte(getenvInt("MQTT_QOS", 1)),

        DBDSN: getenv("DB_DSN", "postgres://postgres:postgres@localhost:5432/agentq?sslmode=disable"),

        WorkerPollInterval: getenvDuration("WORKER_POLL_INTERVAL", 2*time.Second),
        MaxAttempts:        getenvInt("MAX_ATTEMPTS", 3),
        JobLockTimeout:     getenvDuration("JOB_LOCK_TIMEOUT", 30*time.Second),

		OpenCodeCmd:     getenv("OPENCODE_CMD", "opencode"),
		OpenCodeArgs:    strings.Fields(getenv("OPENCODE_ARGS", "run --format json")),
		OpenCodeTimeout: getenvDuration("OPENCODE_TIMEOUT", 2*time.Minute),
		OpenCodeDir:     getenv("OPENCODE_DIR", ""),
		OpenCodeMaxOutputBytes: getenvInt("OPENCODE_MAX_OUTPUT_BYTES", 5*1024*1024),
		OpenCodeDebugMem: strings.EqualFold(getenv("OPENCODE_DEBUG_MEM", ""), "true"),
		OpenCodeDebugArgs: strings.EqualFold(getenv("OPENCODE_DEBUG_ARGS", ""), "true"),

        HTTPAddr: getenv("HTTP_ADDR", ":8080"),
    }
}

func getenv(key, fallback string) string {
    value := os.Getenv(key)
    if value == "" {
        return fallback
    }
    return value
}

func getenvInt(key string, fallback int) int {
    value := os.Getenv(key)
    if value == "" {
        return fallback
    }
    parsed, err := strconv.Atoi(value)
    if err != nil {
        return fallback
    }
    return parsed
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
    value := os.Getenv(key)
    if value == "" {
        return fallback
    }
    parsed, err := time.ParseDuration(value)
    if err != nil {
        return fallback
    }
    return parsed
}

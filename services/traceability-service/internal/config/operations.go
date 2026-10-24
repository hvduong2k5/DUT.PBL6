package config

import (
	"errors"
	"strconv"
	"strings"
)

type Operations struct {
	RelayEnabled, ExpiryEnabled bool
	Brokers                     []string
}

func LoadOperations(getenv func(string) string) (Operations, error) {
	result := Operations{ExpiryEnabled: true}
	for _, flag := range []struct {
		name   string
		target *bool
	}{{"OUTBOX_RELAY_ENABLED", &result.RelayEnabled}, {"EXPIRY_WORKER_ENABLED", &result.ExpiryEnabled}} {
		if value := getenv(flag.name); value != "" {
			b, err := strconv.ParseBool(value)
			if err != nil {
				return Operations{}, errors.New("worker enable flags must be booleans")
			}
			*flag.target = b
		}
	}
	if result.RelayEnabled {
		for _, broker := range strings.Split(getenv("KAFKA_BROKERS"), ",") {
			broker = strings.TrimSpace(broker)
			if broker == "" {
				return Operations{}, errors.New("enabled outbox relay requires explicit KAFKA_BROKERS")
			}
			result.Brokers = append(result.Brokers, broker)
		}
	}
	return result, nil
}

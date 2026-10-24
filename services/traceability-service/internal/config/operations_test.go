package config

import "testing"

func TestWorkerCapabilitiesCannotSilentlyFallback(t *testing.T) {
	cases := []struct {
		name  string
		env   map[string]string
		valid bool
	}{
		{"sandbox defaults", map[string]string{}, true},
		{"enabled without broker", map[string]string{"OUTBOX_RELAY_ENABLED": "true"}, false},
		{"bad boolean", map[string]string{"EXPIRY_WORKER_ENABLED": "perhaps"}, false},
		{"explicit off", map[string]string{"OUTBOX_RELAY_ENABLED": "false", "EXPIRY_WORKER_ENABLED": "false"}, true},
		{"enabled", map[string]string{"OUTBOX_RELAY_ENABLED": "true", "KAFKA_BROKERS": "127.0.0.1:19093"}, true},
		{"empty broker segment", map[string]string{"OUTBOX_RELAY_ENABLED": "true", "KAFKA_BROKERS": "kafka:9092,"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadOperations(func(key string) string { return tc.env[key] })
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}

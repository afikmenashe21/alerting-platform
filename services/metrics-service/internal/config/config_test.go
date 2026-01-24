// Package config provides tests for configuration validation.
package config

import (
	"testing"
)

// TestConfig_Validate tests the Validate method with various scenarios.
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				HTTPPort:    "8083",
				PostgresDSN: "postgres://user:pass@localhost:5432/db",
				RedisAddr:   "localhost:6379",
			},
			wantErr: false,
		},
		{
			name: "empty http-port",
			config: Config{
				HTTPPort:    "",
				PostgresDSN: "postgres://user:pass@localhost:5432/db",
				RedisAddr:   "localhost:6379",
			},
			wantErr: true,
			errMsg:  "http-port cannot be empty",
		},
		{
			name: "empty postgres-dsn",
			config: Config{
				HTTPPort:    "8083",
				PostgresDSN: "",
				RedisAddr:   "localhost:6379",
			},
			wantErr: true,
			errMsg:  "postgres-dsn cannot be empty",
		},
		{
			name: "empty redis-addr",
			config: Config{
				HTTPPort:    "8083",
				PostgresDSN: "postgres://user:pass@localhost:5432/db",
				RedisAddr:   "",
			},
			wantErr: true,
			errMsg:  "redis-addr cannot be empty",
		},
		{
			name: "all fields empty",
			config: Config{
				HTTPPort:    "",
				PostgresDSN: "",
				RedisAddr:   "",
			},
			wantErr: true,
			errMsg:  "http-port cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("Config.Validate() error = %v, want error message %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

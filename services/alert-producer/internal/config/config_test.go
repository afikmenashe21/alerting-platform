package config

import (
	"testing"
)

func TestParseDistribution(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]int
		wantErr bool
	}{
		{
			name:  "valid distribution",
			input: "HIGH:30,MEDIUM:40,LOW:20,CRITICAL:10",
			want: map[string]int{
				"HIGH":    30,
				"MEDIUM":  40,
				"LOW":     20,
				"CRITICAL": 10,
			},
			wantErr: false,
		},
		{
			name:    "invalid sum",
			input:   "HIGH:30,MEDIUM:40",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "HIGH:30:EXTRA",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid percentage",
			input:   "HIGH:150",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDistribution(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDistribution() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Check that all expected values are present
				for k, v := range tt.want {
					if got[k] != v {
						t.Errorf("ParseDistribution() got[%s] = %v, want %v", k, got[k], v)
					}
				}
				// Check that we don't have extra values
				for k, v := range got {
					if tt.want[k] != v {
						t.Errorf("ParseDistribution() unexpected got[%s] = %v", k, v)
					}
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with RPS",
			config: Config{
				KafkaBrokers: "localhost:9092",
				Topic:        "alerts.new",
				RPS:          10.0,
				Duration:     60,
				SeverityDist: "HIGH:50,LOW:50",
				SourceDist:   "api:100",
				NameDist:     "error:100",
			},
			wantErr: false,
		},
		{
			name: "valid config with burst",
			config: Config{
				KafkaBrokers: "localhost:9092",
				Topic:        "alerts.new",
				BurstSize:    100,
				SeverityDist: "HIGH:50,LOW:50",
				SourceDist:   "api:100",
				NameDist:     "error:100",
			},
			wantErr: false,
		},
		{
			name: "missing brokers",
			config: Config{
				Topic:   "alerts.new",
				RPS:     10.0,
				Duration: 60,
			},
			wantErr: true,
		},
		{
			name: "missing topic",
			config: Config{
				KafkaBrokers: "localhost:9092",
				RPS:          10.0,
				Duration:     60,
			},
			wantErr: true,
		},
		{
			name: "no RPS or burst",
			config: Config{
				KafkaBrokers: "localhost:9092",
				Topic:        "alerts.new",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

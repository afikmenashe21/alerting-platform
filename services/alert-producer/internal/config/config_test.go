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
		{
			name: "invalid severity distribution",
			config: Config{
				KafkaBrokers: "localhost:9092",
				Topic:        "alerts.new",
				RPS:          10.0,
				Duration:     60,
				SeverityDist: "HIGH:50,LOW:30", // Doesn't sum to 100
				SourceDist:   "api:100",
				NameDist:     "error:100",
			},
			wantErr: true,
		},
		{
			name: "invalid source distribution",
			config: Config{
				KafkaBrokers: "localhost:9092",
				Topic:        "alerts.new",
				RPS:          10.0,
				Duration:     60,
				SeverityDist: "HIGH:100",
				SourceDist:   "api:50,db:30", // Doesn't sum to 100
				NameDist:     "error:100",
			},
			wantErr: true,
		},
		{
			name: "invalid name distribution",
			config: Config{
				KafkaBrokers: "localhost:9092",
				Topic:        "alerts.new",
				RPS:          10.0,
				Duration:     60,
				SeverityDist: "HIGH:100",
				SourceDist:   "api:100",
				NameDist:     "error:50,timeout:30", // Doesn't sum to 100
			},
			wantErr: true,
		},
		{
			name: "zero duration with continuous mode",
			config: Config{
				KafkaBrokers: "localhost:9092",
				Topic:        "alerts.new",
				RPS:          10.0,
				Duration:     0,
				SeverityDist: "HIGH:100",
				SourceDist:   "api:100",
				NameDist:     "error:100",
			},
			wantErr: true,
		},
		{
			name: "negative RPS",
			config: Config{
				KafkaBrokers: "localhost:9092",
				Topic:        "alerts.new",
				RPS:          -1.0,
				Duration:     60,
				SeverityDist: "HIGH:100",
				SourceDist:   "api:100",
				NameDist:     "error:100",
			},
			wantErr: true,
		},
		{
			name: "zero RPS with burst",
			config: Config{
				KafkaBrokers: "localhost:9092",
				Topic:        "alerts.new",
				RPS:          0,
				BurstSize:    100,
				SeverityDist: "HIGH:100",
				SourceDist:   "api:100",
				NameDist:     "error:100",
			},
			wantErr: false,
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

func TestParseDistribution_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]int
		wantErr bool
	}{
		{
			name:  "single value",
			input: "HIGH:100",
			want: map[string]int{
				"HIGH": 100,
			},
			wantErr: false,
		},
		{
			name:  "with whitespace",
			input: " HIGH : 50 , LOW : 50 ",
			want: map[string]int{
				"HIGH": 50,
				"LOW":  50,
			},
			wantErr: false,
		},
		{
			name:    "negative percentage",
			input:   "HIGH:-10",
			wantErr: true,
		},
		{
			name:    "percentage over 100",
			input:   "HIGH:150",
			wantErr: true,
		},
		{
			name:    "non-numeric percentage",
			input:   "HIGH:abc",
			wantErr: true,
		},
		{
			name:    "missing colon",
			input:   "HIGH50",
			wantErr: true,
		},
		{
			name:    "multiple colons",
			input:   "HIGH:50:EXTRA",
			wantErr: true,
		},
		{
			name:  "zero percentage",
			input: "HIGH:0,LOW:100",
			want: map[string]int{
				"HIGH": 0,
				"LOW":  100,
			},
			wantErr: false,
		},
		{
			name:    "empty key",
			input:   ":50,OTHER:50",
			wantErr: false, // Empty key is allowed (trimmed), but we'll check the result
			want: map[string]int{
				"":     50,
				"OTHER": 50,
			},
		},
		{
			name:  "many values",
			input: "A:10,B:20,C:30,D:40",
			want: map[string]int{
				"A": 10,
				"B": 20,
				"C": 30,
				"D": 40,
			},
			wantErr: false,
		},
		{
			name:    "empty parts in comma-separated list",
			input:   "A:50,,B:50",
			want: map[string]int{
				"A": 50,
				"B": 50,
			},
			wantErr: false,
		},
		{
			name:    "whitespace only part",
			input:   "A:50,  ,B:50",
			want: map[string]int{
				"A": 50,
				"B": 50,
			},
			wantErr: false,
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
				// Check that we don't have unexpected values (except empty key which is trimmed)
				for k, v := range got {
					if k != "" && tt.want[k] != v {
						t.Errorf("ParseDistribution() unexpected got[%s] = %v", k, v)
					}
				}
			}
		})
	}
}

// Package generator provides alert generation with configurable weighted distributions.
// It supports deterministic generation via seed-based RNG for reproducible test data.
package generator

import (
	"fmt"
	"math/rand"
	"time"

	"alert-producer/internal/config"

	"github.com/google/uuid"
)

// Alert represents a single alert event that will be published to Kafka.
// It follows the schema defined in the project brief with schema_version for evolution.
type Alert struct {
	AlertID       string            `json:"alert_id"`
	SchemaVersion int               `json:"schema_version"`
	EventTS       int64             `json:"event_ts"`
	Severity      string            `json:"severity"`
	Source        string            `json:"source"`
	Name          string            `json:"name"`
	Context       map[string]string `json:"context,omitempty"`
}

// Generator creates alerts according to configured distributions.
// It maintains separate weighted distributions for severity, source, and name fields.
type Generator struct {
	rng           *rand.Rand
	severityDist  []weightedValue
	sourceDist    []weightedValue
	nameDist      []weightedValue
	schemaVersion int
}

// weightedValue represents a single value in a weighted distribution.
type weightedValue struct {
	value  string // The value to select
	weight int    // Weight (percentage) for this value
}

const (
	// contextEnvironmentProbability is the probability of adding an environment context field
	contextEnvironmentProbability = 0.3
	// contextRegionProbability is the probability of adding a region context field
	contextRegionProbability = 0.2
)

// New creates a new alert generator with the given configuration.
// It initializes the RNG (with seed if provided) and parses all distribution strings.
// Panics if distribution parsing fails (should be caught during config validation).
func New(cfg config.Config) *Generator {
	gen := &Generator{
		schemaVersion: 1,
	}

	// Initialize RNG
	if cfg.Seed != 0 {
		gen.rng = rand.New(rand.NewSource(cfg.Seed))
	} else {
		gen.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	// Parse distributions (should already be validated in config.Validate)
	// We still handle errors here as a safety measure, but they should not occur
	var err error
	gen.severityDist, err = parseWeightedDistribution(cfg.SeverityDist)
	if err != nil {
		panic(fmt.Sprintf("invalid severity distribution (should be caught in config validation): %v", err))
	}

	gen.sourceDist, err = parseWeightedDistribution(cfg.SourceDist)
	if err != nil {
		panic(fmt.Sprintf("invalid source distribution (should be caught in config validation): %v", err))
	}

	gen.nameDist, err = parseWeightedDistribution(cfg.NameDist)
	if err != nil {
		panic(fmt.Sprintf("invalid name distribution (should be caught in config validation): %v", err))
	}

	return gen
}

// parseWeightedDistribution converts a distribution string into a slice of weighted values.
// This internal function is used during generator initialization.
func parseWeightedDistribution(distStr string) ([]weightedValue, error) {
	distMap, err := config.ParseDistribution(distStr)
	if err != nil {
		return nil, err
	}

	result := make([]weightedValue, 0, len(distMap))
	for value, weight := range distMap {
		result = append(result, weightedValue{value: value, weight: weight})
	}

	return result, nil
}

// Generate creates a new alert with random values according to the configured distributions.
// Each alert gets a unique UUID, current timestamp, and values selected from weighted distributions.
// Optional context fields are added probabilistically.
func (g *Generator) Generate() *Alert {
	alert := &Alert{
		AlertID:       uuid.New().String(),
		SchemaVersion: g.schemaVersion,
		EventTS:       time.Now().Unix(),
		Severity:      g.selectWeighted(g.severityDist),
		Source:        g.selectWeighted(g.sourceDist),
		Name:          g.selectWeighted(g.nameDist),
		Context:       make(map[string]string),
	}

	// Add optional context fields probabilistically for more realistic test data
	if g.rng.Float64() < contextEnvironmentProbability {
		alert.Context["environment"] = g.selectFrom([]string{"prod", "staging", "dev"})
	}
	if g.rng.Float64() < contextRegionProbability {
		alert.Context["region"] = g.selectFrom([]string{"us-east-1", "us-west-2", "eu-west-1"})
	}

	return alert
}

// selectWeighted selects a value from a weighted distribution using cumulative probability.
// Uses the generator's RNG to ensure deterministic behavior when seeded.
func (g *Generator) selectWeighted(choices []weightedValue) string {
	if len(choices) == 0 {
		return "unknown"
	}

	// Calculate total weight
	total := 0
	for _, c := range choices {
		total += c.weight
	}

	// Select random number in [0, total)
	r := g.rng.Intn(total)

	// Find which value corresponds to this random number using cumulative distribution
	cumulative := 0
	for _, c := range choices {
		cumulative += c.weight
		if r < cumulative {
			return c.value
		}
	}

	// Fallback (shouldn't happen, but ensures we always return something)
	return choices[len(choices)-1].value
}

// selectFrom randomly selects a value from a slice of strings with uniform probability.
func (g *Generator) selectFrom(choices []string) string {
	if len(choices) == 0 {
		return ""
	}
	return choices[g.rng.Intn(len(choices))]
}

// GenerateBoilerplate creates a single alert with fixed values that match common test rules.
// Uses HIGH severity, api source, timeout name - a common rule combination.
func GenerateBoilerplate() *Alert {
	return &Alert{
		AlertID:       uuid.New().String(),
		SchemaVersion: 1,
		EventTS:       time.Now().Unix(),
		Severity:      "HIGH",
		Source:        "api",
		Name:          "timeout",
		Context:       make(map[string]string),
	}
}

// GenerateTestAlert creates a test alert with specific values: LOW severity, test-source, test-name.
// This matches the test rule for client afik-test.
func GenerateTestAlert() *Alert {
	return &Alert{
		AlertID:       uuid.New().String(),
		SchemaVersion: 1,
		EventTS:       time.Now().Unix(),
		Severity:      "LOW",
		Source:        "test-source",
		Name:          "test-name",
		Context:       make(map[string]string),
	}
}

// GenerateCustomAlert creates an alert with user-specified severity, source, and name.
func GenerateCustomAlert(severity, source, name string) *Alert {
	return &Alert{
		AlertID:       uuid.New().String(),
		SchemaVersion: 1,
		EventTS:       time.Now().Unix(),
		Severity:      severity,
		Source:        source,
		Name:          name,
		Context:       make(map[string]string),
	}
}

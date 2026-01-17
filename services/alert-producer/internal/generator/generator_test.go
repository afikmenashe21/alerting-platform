package generator

import (
	"testing"

	"alert-producer/internal/config"
)

func TestGenerator_Generate(t *testing.T) {
	cfg := config.Config{
		SeverityDist: "HIGH:50,LOW:50",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42, // Deterministic
	}

	gen := New(cfg)
	alert := gen.Generate()

	// Check required fields
	if alert.AlertID == "" {
		t.Error("AlertID should not be empty")
	}
	if alert.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", alert.SchemaVersion)
	}
	if alert.EventTS == 0 {
		t.Error("EventTS should not be zero")
	}
	if alert.Severity != "HIGH" && alert.Severity != "LOW" {
		t.Errorf("Severity = %s, want HIGH or LOW", alert.Severity)
	}
	if alert.Source != "api" {
		t.Errorf("Source = %s, want api", alert.Source)
	}
	if alert.Name != "error" {
		t.Errorf("Name = %s, want error", alert.Name)
	}
	if alert.Context == nil {
		t.Error("Context should not be nil")
	}
}

func TestGenerator_SelectWeighted(t *testing.T) {
	cfg := config.Config{
		SeverityDist: "HIGH:50,LOW:50",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}

	gen := New(cfg)

	// Test weighted selection
	choices := []weightedValue{
		{value: "A", weight: 50},
		{value: "B", weight: 50},
	}

	// Run multiple times to check distribution
	results := make(map[string]int)
	for i := 0; i < 1000; i++ {
		result := gen.selectWeighted(choices)
		results[result]++
	}

	// Both should appear roughly equally (allowing for randomness)
	if results["A"] == 0 || results["B"] == 0 {
		t.Errorf("Both values should appear, got A=%d, B=%d", results["A"], results["B"])
	}
}

func TestGenerator_SelectFrom(t *testing.T) {
	cfg := config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}

	gen := New(cfg)

	choices := []string{"a", "b", "c"}
	result := gen.selectFrom(choices)

	if result == "" {
		t.Error("selectFrom should return a non-empty string")
	}

	found := false
	for _, c := range choices {
		if c == result {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("selectFrom returned %s, not in choices", result)
	}
}

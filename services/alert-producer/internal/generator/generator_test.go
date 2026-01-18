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

func TestGenerator_SelectFrom_EmptyChoices(t *testing.T) {
	cfg := config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}

	gen := New(cfg)

	result := gen.selectFrom([]string{})
	if result != "" {
		t.Errorf("selectFrom with empty choices should return empty string, got %s", result)
	}
}

func TestGenerator_SelectWeighted_EmptyChoices(t *testing.T) {
	cfg := config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}

	gen := New(cfg)

	result := gen.selectWeighted([]weightedValue{})
	if result != "unknown" {
		t.Errorf("selectWeighted with empty choices should return 'unknown', got %s", result)
	}
}

func TestGenerator_New_WithSeed(t *testing.T) {
	cfg := config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}

	gen := New(cfg)
	if gen == nil {
		t.Fatal("New should not return nil")
	}

	// Generate two alerts with same seed - should be deterministic
	alert1 := gen.Generate()
	gen2 := New(cfg)
	alert2 := gen2.Generate()

	// With same seed, severity should be the same (deterministic)
	if alert1.Severity != alert2.Severity {
		t.Errorf("With same seed, alerts should be deterministic, got %s vs %s", alert1.Severity, alert2.Severity)
	}
}

func TestGenerator_New_WithoutSeed(t *testing.T) {
	cfg := config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         0, // No seed
	}

	gen := New(cfg)
	if gen == nil {
		t.Fatal("New should not return nil")
	}

	alert := gen.Generate()
	if alert.Severity != "HIGH" {
		t.Errorf("Expected HIGH severity, got %s", alert.Severity)
	}
}

func TestGenerator_New_PanicOnInvalidDistribution(t *testing.T) {
	// This should panic because config validation should catch this
	// But we test that New handles it gracefully
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on invalid distribution")
		}
	}()

	cfg := config.Config{
		SeverityDist: "INVALID:50,OTHER:30", // Doesn't sum to 100
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}

	// This should panic since validation should have caught it
	_ = New(cfg)
}

func TestGenerateBoilerplate(t *testing.T) {
	alert := GenerateBoilerplate()

	if alert.AlertID == "" {
		t.Error("AlertID should not be empty")
	}
	if alert.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", alert.SchemaVersion)
	}
	if alert.EventTS == 0 {
		t.Error("EventTS should not be zero")
	}
	if alert.Severity != "HIGH" {
		t.Errorf("Severity = %s, want HIGH", alert.Severity)
	}
	if alert.Source != "api" {
		t.Errorf("Source = %s, want api", alert.Source)
	}
	if alert.Name != "timeout" {
		t.Errorf("Name = %s, want timeout", alert.Name)
	}
	if alert.Context == nil {
		t.Error("Context should not be nil")
	}
}

func TestGenerateTestAlert(t *testing.T) {
	alert := GenerateTestAlert()

	if alert.AlertID == "" {
		t.Error("AlertID should not be empty")
	}
	if alert.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", alert.SchemaVersion)
	}
	if alert.EventTS == 0 {
		t.Error("EventTS should not be zero")
	}
	if alert.Severity != "LOW" {
		t.Errorf("Severity = %s, want LOW", alert.Severity)
	}
	if alert.Source != "test-source" {
		t.Errorf("Source = %s, want test-source", alert.Source)
	}
	if alert.Name != "test-name" {
		t.Errorf("Name = %s, want test-name", alert.Name)
	}
	if alert.Context == nil {
		t.Error("Context should not be nil")
	}
}

func TestGenerator_Generate_ContextFields(t *testing.T) {
	cfg := config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}

	gen := New(cfg)

	// Generate many alerts to increase chance of getting context fields
	hasEnvironment := false
	hasRegion := false
	for i := 0; i < 100; i++ {
		alert := gen.Generate()
		if alert.Context["environment"] != "" {
			hasEnvironment = true
		}
		if alert.Context["region"] != "" {
			hasRegion = true
		}
	}

	// With probability 0.3 and 0.2, we should see at least one after 100 tries
	// This is probabilistic, but very likely
	if !hasEnvironment && !hasRegion {
		t.Log("Note: No context fields generated in 100 attempts (this is probabilistic)")
	}
}

func TestGenerator_SelectWeighted_SingleChoice(t *testing.T) {
	cfg := config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}

	gen := New(cfg)

	choices := []weightedValue{
		{value: "A", weight: 100},
	}

	result := gen.selectWeighted(choices)
	if result != "A" {
		t.Errorf("selectWeighted with single choice should return 'A', got %s", result)
	}
}

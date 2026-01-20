package processor

import (
	"context"
	"errors"
	"testing"
	"time"

	"alert-producer/internal/config"
	"alert-producer/internal/generator"
)

// mockPublisher is a mock implementation of AlertPublisher for testing
type mockPublisher struct {
	published []*generator.Alert
	shouldErr bool
	errMsg    string
	closed    bool
}

func newMockPublisher(shouldErr bool, errMsg string) *mockPublisher {
	return &mockPublisher{
		published: make([]*generator.Alert, 0),
		shouldErr: shouldErr,
		errMsg:    errMsg,
		closed:    false,
	}
}

func (m *mockPublisher) Publish(ctx context.Context, alert *generator.Alert) error {
	if m.shouldErr {
		return errors.New(m.errMsg)
	}
	m.published = append(m.published, alert)
	return nil
}

func (m *mockPublisher) Close() error {
	m.closed = true
	return nil
}

func TestNewProcessor(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")

	proc := NewProcessor(gen, pub, cfg)
	if proc == nil {
		t.Fatal("NewProcessor should not return nil")
	}
	if proc.generator != gen {
		t.Error("Processor should store the generator")
	}
	if proc.publisher != pub {
		t.Error("Processor should store the publisher")
	}
	if proc.cfg != cfg {
		t.Error("Processor should store the config")
	}
}

func TestProcessor_Process_BurstMode(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
		BurstSize:    5,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.Process(ctx)
	if err != nil {
		t.Fatalf("Process should not error, got: %v", err)
	}

	// Should publish boilerplate + 5 burst alerts = 6 total
	if len(pub.published) != 6 {
		t.Errorf("Expected 6 published alerts (1 boilerplate + 5 burst), got %d", len(pub.published))
	}

	// First should be boilerplate
	if pub.published[0].Severity != "HIGH" || pub.published[0].Source != "api" || pub.published[0].Name != "timeout" {
		t.Error("First alert should be boilerplate")
	}
}

func TestProcessor_Process_ContinuousMode(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
		RPS:          100.0, // Higher RPS to ensure at least one alert is published
		Duration:     200 * time.Millisecond, // Longer duration to ensure alerts are published
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.Process(ctx)
	if err != nil {
		t.Fatalf("Process should not error, got: %v", err)
	}

	// Should publish at least boilerplate + some continuous alerts
	if len(pub.published) < 2 {
		t.Errorf("Expected at least 2 published alerts (boilerplate + continuous), got %d", len(pub.published))
	}

	// First should be boilerplate
	if pub.published[0].Severity != "HIGH" || pub.published[0].Source != "api" || pub.published[0].Name != "timeout" {
		t.Error("First alert should be boilerplate")
	}
}

func TestProcessor_Process_ErrorOnBoilerplate(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
		BurstSize:    5,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(true, "publish error")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.Process(ctx)
	if err == nil {
		t.Fatal("Process should error when boilerplate publish fails")
	}
}

func TestProcessor_ProcessBurst(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.ProcessBurst(ctx, 3)
	if err != nil {
		t.Fatalf("ProcessBurst should not error, got: %v", err)
	}

	if len(pub.published) != 3 {
		t.Errorf("Expected 3 published alerts, got %d", len(pub.published))
	}
}

func TestProcessor_ProcessBurst_Error(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(true, "publish error")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.ProcessBurst(ctx, 3)
	if err == nil {
		t.Fatal("ProcessBurst should error when publish fails")
	}
}

func TestProcessor_ProcessBurst_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := proc.ProcessBurst(ctx, 100)
	if err == nil {
		t.Fatal("ProcessBurst should error when context is cancelled")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestProcessor_ProcessContinuous(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.ProcessContinuous(ctx, 100.0, 200*time.Millisecond) // Higher RPS and longer duration
	if err != nil {
		t.Fatalf("ProcessContinuous should not error, got: %v", err)
	}

	if len(pub.published) < 1 {
		t.Errorf("Expected at least 1 published alert, got %d", len(pub.published))
	}
}

func TestProcessor_ProcessContinuous_Error(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(true, "publish error")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	// Use higher RPS and longer duration to ensure an alert is published before duration expires
	err := proc.ProcessContinuous(ctx, 100.0, 1*time.Second)
	if err == nil {
		t.Fatal("ProcessContinuous should error when publish fails")
	}
}

func TestProcessor_ProcessContinuous_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := proc.ProcessContinuous(ctx, 10.0, 5*time.Second)
	if err == nil {
		t.Fatal("ProcessContinuous should error when context is cancelled")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestProcessor_ProcessTest_BurstMode(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.ProcessTest(ctx, 10.0, 1*time.Second, 5)
	if err != nil {
		t.Fatalf("ProcessTest should not error, got: %v", err)
	}

	if len(pub.published) != 5 {
		t.Errorf("Expected 5 published alerts, got %d", len(pub.published))
	}

	// First should be test alert
	first := pub.published[0]
	if first.Severity != "LOW" || first.Source != "test-source" || first.Name != "test-name" {
		t.Errorf("First alert should be test alert, got: %+v", first)
	}
}

func TestProcessor_ProcessTest_ContinuousMode(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	// Use higher RPS and longer duration to ensure alerts are published
	err := proc.ProcessTest(ctx, 100.0, 200*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("ProcessTest should not error, got: %v", err)
	}

	if len(pub.published) < 1 {
		t.Fatalf("Expected at least 1 published alert, got %d", len(pub.published))
	}

	// First should be test alert
	first := pub.published[0]
	if first.Severity != "LOW" || first.Source != "test-source" || first.Name != "test-name" {
		t.Errorf("First alert should be test alert, got: %+v", first)
	}
}

func TestProcessor_ProcessTest_Error(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(true, "publish error")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.ProcessTest(ctx, 10.0, 1*time.Second, 5)
	if err == nil {
		t.Fatal("ProcessTest should error when publish fails")
	}
}

func TestProcessor_ProcessTest_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := proc.ProcessTest(ctx, 10.0, 1*time.Second, 100)
	if err == nil {
		t.Fatal("ProcessTest should error when context is cancelled")
	}
}

func TestProcessor_runBurstModeWithSize_ProgressLogging(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	// Use burst size that triggers progress logging (100, 200, etc.)
	err := proc.runBurstModeWithSize(ctx, 250)
	if err != nil {
		t.Fatalf("runBurstModeWithSize should not error, got: %v", err)
	}

	if len(pub.published) != 250 {
		t.Errorf("Expected 250 published alerts, got %d", len(pub.published))
	}
}

func TestProcessor_runContinuousModeWithParams_DurationReached(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	// Use higher RPS and longer duration to ensure alerts are published
	err := proc.runContinuousModeWithParams(ctx, 100.0, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("runContinuousModeWithParams should not error, got: %v", err)
	}

	// Should have published some alerts before duration expired
	if len(pub.published) == 0 {
		t.Error("Expected at least some published alerts before duration expired")
	}
}

func TestProcessor_runTestBurstMode_ProgressLogging(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.runTestBurstMode(ctx, 150, nil)
	if err != nil {
		t.Fatalf("runTestBurstMode should not error, got: %v", err)
	}

	if len(pub.published) != 150 {
		t.Errorf("Expected 150 published alerts, got %d", len(pub.published))
	}

	// First should be test alert
	if pub.published[0].Severity != "LOW" || pub.published[0].Source != "test-source" {
		t.Error("First alert should be test alert")
	}
}

func TestProcessor_runTestContinuousMode_TestAlertSent(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	// Use higher RPS to ensure alerts are published
	err := proc.runTestContinuousMode(ctx, 100.0, 200*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("runTestContinuousMode should not error, got: %v", err)
	}

	if len(pub.published) < 1 {
		t.Fatal("Expected at least 1 published alert")
	}

	// First should be test alert
	first := pub.published[0]
	if first.Severity != "LOW" || first.Source != "test-source" || first.Name != "test-name" {
		t.Errorf("First alert should be test alert, got: %+v", first)
	}

	// Subsequent alerts should be varied (not test alerts)
	if len(pub.published) > 1 {
		second := pub.published[1]
		if second.Severity == "LOW" && second.Source == "test-source" && second.Name == "test-name" {
			t.Error("Second alert should not be test alert")
		}
	}
}

func TestProcessor_runBurstMode(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
		BurstSize:    10,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.runBurstMode(ctx)
	if err != nil {
		t.Fatalf("runBurstMode should not error, got: %v", err)
	}

	if len(pub.published) != 10 {
		t.Errorf("Expected 10 published alerts, got %d", len(pub.published))
	}
}

func TestProcessor_runContinuousMode(t *testing.T) {
	cfg := &config.Config{
		SeverityDist: "HIGH:100",
		SourceDist:   "api:100",
		NameDist:     "error:100",
		Seed:         42,
		RPS:          100.0,
		Duration:     200 * time.Millisecond,
	}
	gen := generator.New(*cfg)
	pub := newMockPublisher(false, "")
	proc := NewProcessor(gen, pub, cfg)

	ctx := context.Background()
	err := proc.runContinuousMode(ctx)
	if err != nil {
		t.Fatalf("runContinuousMode should not error, got: %v", err)
	}

	if len(pub.published) < 1 {
		t.Error("Expected at least 1 published alert")
	}
}

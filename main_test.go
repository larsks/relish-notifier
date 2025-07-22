package main

import (
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOrderStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status OrderStatus
		want   string
	}{
		{
			name:   "Order Placed",
			status: OrderStatusPlaced,
			want:   "Order Placed",
		},
		{
			name:   "Preparing Your Order",
			status: OrderStatusPreparing,
			want:   "Preparing Your Order",
		},
		{
			name:   "Order Arrived",
			status: OrderStatusArrived,
			want:   "Order Arrived",
		},
		{
			name:   "Unknown",
			status: OrderStatusUnknown,
			want:   "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("OrderStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTextToStatus(t *testing.T) {
	tests := []struct {
		name string
		text string
		want OrderStatus
	}{
		{
			name: "Valid Order Placed",
			text: "Order Placed",
			want: OrderStatusPlaced,
		},
		{
			name: "Valid Preparing",
			text: "Preparing Your Order",
			want: OrderStatusPreparing,
		},
		{
			name: "Valid Arrived",
			text: "Order Arrived",
			want: OrderStatusArrived,
		},
		{
			name: "Invalid status returns Unknown",
			text: "Invalid Status",
			want: OrderStatusUnknown,
		},
		{
			name: "Empty string returns Unknown",
			text: "",
			want: OrderStatusUnknown,
		},
		{
			name: "Case sensitive - different case returns Unknown",
			text: "order placed",
			want: OrderStatusUnknown,
		},
		{
			name: "Whitespace around valid status returns Unknown",
			text: " Order Placed ",
			want: OrderStatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := textToStatus(tt.text); got != tt.want {
				t.Errorf("textToStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		want     slog.Level
	}{
		{
			name:     "Debug level",
			logLevel: "debug",
			want:     slog.LevelDebug,
		},
		{
			name:     "Info level",
			logLevel: "info",
			want:     slog.LevelInfo,
		},
		{
			name:     "Warning level",
			logLevel: "warning",
			want:     slog.LevelWarn,
		},
		{
			name:     "Warn level (alias)",
			logLevel: "warn",
			want:     slog.LevelWarn,
		},
		{
			name:     "Error level",
			logLevel: "error",
			want:     slog.LevelError,
		},
		{
			name:     "Invalid level defaults to Warn",
			logLevel: "invalid",
			want:     slog.LevelWarn,
		},
		{
			name:     "Empty level defaults to Warn",
			logLevel: "",
			want:     slog.LevelWarn,
		},
		{
			name:     "Case insensitive - uppercase",
			logLevel: "DEBUG",
			want:     slog.LevelDebug,
		},
		{
			name:     "Case insensitive - mixed case",
			logLevel: "InFo",
			want:     slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := setupLogger(tt.logLevel)
			if logger == nil {
				t.Fatal("setupLogger() returned nil")
			}

			// Test that the logger was created successfully
			if !logger.Enabled(nil, tt.want) {
				t.Errorf("Logger level not set correctly, expected level %v to be enabled", tt.want)
			}

			// For levels below the set level, they should be disabled
			// (except for the edge case where we're testing the exact level)
			if tt.want > slog.LevelDebug {
				lowerLevel := tt.want - 1
				if logger.Enabled(nil, lowerLevel) {
					t.Errorf("Logger should not have level %v enabled when set to %v", lowerLevel, tt.want)
				}
			}
		})
	}
}

func TestNewNotifier(t *testing.T) {
	config := &Config{
		Headless:    true,
		Extensions:  false,
		Interval:    60,
		Once:        true,
		PageTimeout: 30 * time.Second,
		Command:     "echo test",
		LogLevel:    "debug",
	}

	credentials := &Credentials{
		Username: "test@example.com",
		Password: "testpassword",
	}

	logger := setupLogger("debug")

	notifier := NewNotifier(config, credentials, logger)

	if notifier == nil {
		t.Fatal("NewNotifier() returned nil")
	}

	if !reflect.DeepEqual(notifier.config, config) {
		t.Errorf("NewNotifier() config = %v, want %v", notifier.config, config)
	}

	if !reflect.DeepEqual(notifier.credentials, credentials) {
		t.Errorf("NewNotifier() credentials = %v, want %v", notifier.credentials, credentials)
	}

	if notifier.logger != logger {
		t.Errorf("NewNotifier() logger = %v, want %v", notifier.logger, logger)
	}

	if notifier.loginUrl != defaultLoginURL {
		t.Errorf("NewNotifier() loginUrl = %v, want %v", notifier.loginUrl, defaultLoginURL)
	}

	// Browser and page should be nil until initialized
	if notifier.browser != nil {
		t.Errorf("NewNotifier() browser should be nil initially, got %v", notifier.browser)
	}

	if notifier.page != nil {
		t.Errorf("NewNotifier() page should be nil initially, got %v", notifier.page)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test that our expected defaults make sense
	var config Config

	// Check zero values are sensible
	if config.Headless != false {
		t.Errorf("Config.Headless default = %v, want false", config.Headless)
	}

	if config.Extensions != false {
		t.Errorf("Config.Extensions default = %v, want false", config.Extensions)
	}

	if config.Interval != 0 {
		t.Errorf("Config.Interval default = %v, want 0", config.Interval)
	}

	if config.Once != false {
		t.Errorf("Config.Once default = %v, want false", config.Once)
	}

	if config.PageTimeout != 0 {
		t.Errorf("Config.PageTimeout default = %v, want 0", config.PageTimeout)
	}

	if config.Command != "" {
		t.Errorf("Config.Command default = %v, want empty string", config.Command)
	}

	if config.LogLevel != "" {
		t.Errorf("Config.LogLevel default = %v, want empty string", config.LogLevel)
	}
}

// Test that our logger actually writes output at the correct level
func TestSetupLoggerOutput(t *testing.T) {
	// Capture stderr for testing
	r, w, _ := os.Pipe()
	originalStderr := os.Stderr
	os.Stderr = w

	logger := setupLogger("info")

	// Log at different levels
	logger.Debug("debug message")  // Should not appear
	logger.Info("info message")    // Should appear
	logger.Warn("warning message") // Should appear
	logger.Error("error message")  // Should appear

	// Restore stderr and read output
	w.Close()
	os.Stderr = originalStderr

	output := make([]byte, 1024)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	// Debug should not appear
	if strings.Contains(outputStr, "debug message") {
		t.Error("Debug message should not appear at INFO level")
	}

	// Info, warn, error should appear
	if !strings.Contains(outputStr, "info message") {
		t.Error("Info message should appear at INFO level")
	}

	if !strings.Contains(outputStr, "warning message") {
		t.Error("Warning message should appear at INFO level")
	}

	if !strings.Contains(outputStr, "error message") {
		t.Error("Error message should appear at INFO level")
	}
}

// Benchmark the most frequently called function
func BenchmarkTextToStatus(b *testing.B) {
	testCases := []string{
		"Order Placed",
		"Preparing Your Order", 
		"Order Arrived",
		"Invalid Status",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testCase := testCases[i%len(testCases)]
		textToStatus(testCase)
	}
}

func BenchmarkOrderStatusString(b *testing.B) {
	statuses := []OrderStatus{
		OrderStatusPlaced,
		OrderStatusPreparing,
		OrderStatusArrived,
		OrderStatusUnknown,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := statuses[i%len(statuses)]
		_ = status.String()
	}
}
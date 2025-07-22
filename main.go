/*
 *   relish-notifier -- get notified when your food arrives
 *   Copyright (C) 2025 Lars Kellogg-Stedman
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

// Version is set via ldflags during build
var version = "dev"

type OrderStatus string

const (
	OrderStatusPlaced    OrderStatus = "Order Placed"
	OrderStatusPreparing OrderStatus = "Preparing Your Order"
	OrderStatusArrived   OrderStatus = "Order Arrived"
	OrderStatusUnknown   OrderStatus = "Unknown"
)

const defaultLoginURL string = "https://relish.ezcater.com/schedule"

// String converts an OrderStatus value to its string representation
func (os OrderStatus) String() string {
	return string(os)
}

// textToStatus converts a string to the corresponding OrderStatus enum value
func textToStatus(text string) OrderStatus {
	switch text {
	case string(OrderStatusPlaced):
		return OrderStatusPlaced
	case string(OrderStatusPreparing):
		return OrderStatusPreparing
	case string(OrderStatusArrived):
		return OrderStatusArrived
	default:
		return OrderStatusUnknown
	}
}

type Config struct {
	Headless    bool
	Extensions  bool
	Interval    int
	Once        bool
	PageTimeout time.Duration
	Command     string
	Verbose     int
}

type Credentials struct {
	Username string
	Password string
}

type Notifier struct {
	browser     *rod.Browser
	page        *rod.Page
	config      *Config
	credentials *Credentials
	logger      *slog.Logger
	loginUrl    string
}

// NewNotifier creates a new Notifier instance with the provided configuration, credentials, and logger
func NewNotifier(config *Config, credentials *Credentials, logger *slog.Logger) *Notifier {
	return &Notifier{
		config:      config,
		credentials: credentials,
		logger:      logger,
		loginUrl:    defaultLoginURL,
	}
}

// initializeBrowser sets up the browser instance with stealth options and configures the page
func (n *Notifier) initializeBrowser() error {
	n.logger.Debug("initializing browser")

	launcher := launcher.New()

	// Set headless mode explicitly (Rod defaults to headless=true)
	launcher = launcher.Headless(n.config.Headless)

	if !n.config.Extensions {
		launcher = launcher.Set("disable-extensions")
	}

	// Set stealth options similar to selenium-stealth
	launcher = launcher.
		Set("exclude-switches", "enable-automation").
		Set("disable-blink-features", "AutomationControlled").
		Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	url := launcher.MustLaunch()
	browser := rod.New().ControlURL(url)

	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	n.browser = browser
	n.page = browser.MustPage()

	// Set page timeout
	n.page.Timeout(n.config.PageTimeout)

	return nil
}

// Close shuts down the browser instance if it exists
func (n *Notifier) Close() {
	if n.browser != nil {
		n.browser.MustClose()
	}
}

// Login navigates to the Relish login page and authenticates using stored credentials
func (n *Notifier) Login() error {
	n.logger.Info("logging in")

	if err := n.page.Navigate(n.loginUrl); err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}

	// Wait for and fill email field
	if err := n.waitAndSubmit("#identity_email", "[name='commit']", n.credentials.Username); err != nil {
		return fmt.Errorf("failed to submit email: %w", err)
	}

	// Wait for and fill password field
	if err := n.waitAndSubmit("#password", "[name='action']", n.credentials.Password); err != nil {
		return fmt.Errorf("failed to submit password: %w", err)
	}

	return nil
}

// waitAndSubmit waits for a form field, fills it with data, then clicks the specified button
func (n *Notifier) waitAndSubmit(fieldSelector, buttonSelector, data string) error {
	n.logger.Debug("waiting for element before clicking", "field", fieldSelector, "button", buttonSelector)

	// Wait for field to be present and fill it
	field := n.page.MustElement(fieldSelector)
	if err := field.Input(data); err != nil {
		return fmt.Errorf("failed to input data: %w", err)
	}

	// Find and click button
	button := n.page.MustElement(buttonSelector)
	if err := button.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click button: %w", err)
	}

	// Wait for navigation to complete
	n.page.MustWaitNavigation()()

	return nil
}

// CheckOrderStatus scrapes the order status from the Relish website and returns the parsed status
func (n *Notifier) CheckOrderStatus() (OrderStatus, error) {
	n.logger.Debug("checking order status")

	// Look for the schedule-card-label element
	element, err := n.page.Element(".schedule-card-label")
	if err != nil {
		n.logger.Warn("timeout waiting for order status")
		return OrderStatusUnknown, fmt.Errorf("failed to find order status element: %w", err)
	}

	text, err := element.Text()
	if err != nil {
		return OrderStatusUnknown, fmt.Errorf("failed to get element text: %w", err)
	}

	status := textToStatus(strings.TrimSpace(text))
	if status == OrderStatusUnknown {
		n.logger.Warn("unknown order status", "status", text)
	}

	return status, nil
}

// Refresh reloads the current page in the browser
func (n *Notifier) Refresh() error {
	n.logger.Debug("reloading page")
	return n.page.Reload()
}

// getCredentials retrieves login credentials from the system keychain or environment variables
func getCredentials() (*Credentials, error) {
	var username, password string

	// Try keyring first
	username, err := keyring.Get("relish-notifier", "EMAIL")
	if err != nil {
		// Keyring failed, try environment variables
		username = os.Getenv("RELISH_USERNAME")
		if username == "" {
			return nil, fmt.Errorf("failed to get username from keyring (%w) and RELISH_USERNAME environment variable is not set", err)
		}
	}

	password, err = keyring.Get("relish-notifier", "PASSWORD")
	if err != nil {
		// Keyring failed, try environment variables
		password = os.Getenv("RELISH_PASSWORD")
		if password == "" {
			return nil, fmt.Errorf("failed to get password from keyring (%w) and RELISH_PASSWORD environment variable is not set", err)
		}
	}

	if username == "" || password == "" {
		return nil, fmt.Errorf("missing credentials: both keyring and environment variables are empty")
	}

	return &Credentials{
		Username: username,
		Password: password,
	}, nil
}

// setupLogger creates a structured logger with the appropriate log level based on verbosity
func setupLogger(verbose int) *slog.Logger {
	var level slog.Level

	switch {
	case verbose <= 0:
		level = slog.LevelWarn // Default: warning level
	case verbose == 1:
		level = slog.LevelInfo // -v: info level
	case verbose >= 2:
		level = slog.LevelDebug // -vv or more: debug level
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
}

// main sets up the CLI interface and executes the root command
func main() {
	var config Config

	rootCmd := &cobra.Command{
		Use:     "relish-notifier",
		Short:   "Monitor Relish orders and send notifications",
		Long:    "Monitor Relish orders and send notifications.\n\nCredentials are retrieved from the system keychain (service: relish-notifier, accounts: EMAIL/PASSWORD).\nIf keychain is unavailable, environment variables RELISH_USERNAME and RELISH_PASSWORD will be used as fallback.",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNotifier(&config)
		},
	}

	rootCmd.Flags().BoolVar(&config.Headless, "headless", true, "Run Chrome in headless mode")
	rootCmd.Flags().BoolVar(&config.Extensions, "extensions", true, "Enable browser extensions")
	rootCmd.Flags().IntVarP(&config.Interval, "check-interval", "i", 30, "How often to check for delivery (seconds)")
	rootCmd.Flags().BoolVar(&config.Once, "once", false, "Check once and exit")
	rootCmd.Flags().DurationVarP(&config.PageTimeout, "page-timeout", "t", 10*time.Second, "Set page timeout")
	rootCmd.Flags().StringVarP(&config.Command, "command", "c", "", "Run this command when your order has arrived")
	rootCmd.Flags().CountVarP(&config.Verbose, "verbose", "v", "Increase verbosity (-v: info, -vv: debug)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runNotifier initializes the notifier, logs in, and runs the main monitoring loop
func runNotifier(config *Config) error {
	logger := setupLogger(config.Verbose)

	// Get credentials
	credentials, err := getCredentials()
	if err != nil {
		return err
	}

	// Create notifier
	notifier := NewNotifier(config, credentials, logger)
	defer notifier.Close()

	// Initialize browser
	if err := notifier.initializeBrowser(); err != nil {
		return err
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("received interrupt signal")
		cancel()
	}()

	// Login
	if err := notifier.Login(); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	// Main monitoring loop
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		status, err := notifier.CheckOrderStatus()
		if err != nil {
			logger.Error("failed to check order status", "error", err)
		} else {
			logger.Info("notifier reports status", "status", status)

			if status == OrderStatusArrived {
				fmt.Println("order has arrived")
				if config.Command != "" {
					cmd := exec.Command("sh", "-c", config.Command)
					if err := cmd.Run(); err != nil {
						logger.Error("failed to run command", "error", err)
					}
				}
				return nil
			}
		}

		if config.Once {
			fmt.Println("order has not arrived")
			os.Exit(1)
		}

		logger.Info("Checking again", "interval_seconds", config.Interval)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(config.Interval) * time.Second):
		}

		if err := notifier.Refresh(); err != nil {
			logger.Error("failed to refresh page", "error", err)
		}
	}
}

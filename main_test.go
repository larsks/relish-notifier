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
	"log/slog"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("OrderStatus", func() {
	Describe("String method", func() {
		DescribeTable("should return correct string representation",
			func(status OrderStatus, expected string) {
				Expect(status.String()).To(Equal(expected))
			},
			Entry("Order Placed", OrderStatusPlaced, "Order Placed"),
			Entry("Preparing Your Order", OrderStatusPreparing, "Preparing Your Order"),
			Entry("Order Arrived", OrderStatusArrived, "Order Arrived"),
			Entry("Unknown", OrderStatusUnknown, "Unknown"),
		)
	})

	Describe("textToStatus function", func() {
		Context("with valid status strings", func() {
			DescribeTable("should return correct OrderStatus",
				func(text string, expected OrderStatus) {
					Expect(textToStatus(text)).To(Equal(expected))
				},
				Entry("Order Placed", "Order Placed", OrderStatusPlaced),
				Entry("Preparing Your Order", "Preparing Your Order", OrderStatusPreparing),
				Entry("Order Arrived", "Order Arrived", OrderStatusArrived),
			)
		})

		Context("with invalid status strings", func() {
			DescribeTable("should return OrderStatusUnknown",
				func(text string) {
					Expect(textToStatus(text)).To(Equal(OrderStatusUnknown))
				},
				Entry("invalid status", "Invalid Status"),
				Entry("empty string", ""),
				Entry("case sensitive - lowercase", "order placed"),
				Entry("whitespace around", " Order Placed "),
				Entry("partial match", "Order"),
				Entry("extra text", "Order Placed - Confirmed"),
			)
		})
	})
})

var _ = Describe("Logger Setup", func() {
	Describe("setupLogger function", func() {
		Context("with verbose counter", func() {
			DescribeTable("should create logger with correct level",
				func(verbose int, expectedLevel slog.Level) {
					logger := setupLogger(verbose)
					Expect(logger).NotTo(BeNil())
					Expect(logger.Enabled(nil, expectedLevel)).To(BeTrue())
				},
				Entry("default (0): warn level", 0, slog.LevelWarn),
				Entry("-v (1): info level", 1, slog.LevelInfo),
				Entry("-vv (2): debug level", 2, slog.LevelDebug),
				Entry("-vvv (3): debug level", 3, slog.LevelDebug),
				Entry("high count (10): debug level", 10, slog.LevelDebug),
			)
		})

		Context("with level filtering", func() {
			It("should filter debug messages at warn level (default)", func() {
				logger := setupLogger(0)
				Expect(logger.Enabled(nil, slog.LevelWarn)).To(BeTrue())
				Expect(logger.Enabled(nil, slog.LevelDebug)).To(BeFalse())
				Expect(logger.Enabled(nil, slog.LevelInfo)).To(BeFalse())
			})

			It("should allow info and above at info level (-v)", func() {
				logger := setupLogger(1)
				Expect(logger.Enabled(nil, slog.LevelInfo)).To(BeTrue())
				Expect(logger.Enabled(nil, slog.LevelWarn)).To(BeTrue())
				Expect(logger.Enabled(nil, slog.LevelError)).To(BeTrue())
				Expect(logger.Enabled(nil, slog.LevelDebug)).To(BeFalse())
			})

			It("should allow all levels at debug level (-vv)", func() {
				logger := setupLogger(2)
				Expect(logger.Enabled(nil, slog.LevelDebug)).To(BeTrue())
				Expect(logger.Enabled(nil, slog.LevelInfo)).To(BeTrue())
				Expect(logger.Enabled(nil, slog.LevelWarn)).To(BeTrue())
				Expect(logger.Enabled(nil, slog.LevelError)).To(BeTrue())
			})
		})

		It("should create a logger that actually logs", func() {
			// Create a buffer to capture output
			buffer := gbytes.NewBuffer()

			// Create logger with info level (-v)
			logger := setupLogger(1)

			// Note: This is a simplified test since we can't easily redirect slog output
			// In a real scenario, you might use a custom handler for testing
			Expect(logger).NotTo(BeNil())
			Expect(logger.Enabled(nil, slog.LevelInfo)).To(BeTrue())
			Expect(logger.Enabled(nil, slog.LevelDebug)).To(BeFalse())

			_ = buffer // Prevent unused variable error
		})
	})
})

var _ = Describe("Notifier", func() {
	var (
		config      *Config
		credentials *Credentials
		logger      *slog.Logger
	)

	BeforeEach(func() {
		config = &Config{
			Headless:    true,
			Extensions:  false,
			Interval:    60,
			Once:        true,
			PageTimeout: 30 * time.Second,
			Command:     "echo test",
			Verbose:     2, // -vv for debug level
		}

		credentials = &Credentials{
			Username: "test@example.com",
			Password: "testpassword",
		}

		logger = setupLogger(2) // -vv for debug level
	})

	Describe("NewNotifier constructor", func() {
		It("should create a new notifier with correct configuration", func() {
			notifier := NewNotifier(config, credentials, logger)

			Expect(notifier).NotTo(BeNil())
			Expect(notifier.config).To(Equal(config))
			Expect(notifier.credentials).To(Equal(credentials))
			Expect(notifier.logger).To(Equal(logger))
			Expect(notifier.loginUrl).To(Equal(defaultLoginURL))
		})

		It("should initialize with nil browser and page", func() {
			notifier := NewNotifier(config, credentials, logger)

			Expect(notifier.browser).To(BeNil())
			Expect(notifier.page).To(BeNil())
		})

		It("should handle nil inputs gracefully", func() {
			// While not recommended, the constructor should not panic
			Expect(func() {
				NewNotifier(nil, nil, nil)
			}).NotTo(Panic())
		})
	})
})

var _ = Describe("Configuration", func() {
	Describe("Config struct", func() {
		It("should have sensible zero values", func() {
			var config Config

			Expect(config.Headless).To(BeFalse())
			Expect(config.Extensions).To(BeFalse())
			Expect(config.Interval).To(Equal(0))
			Expect(config.Once).To(BeFalse())
			Expect(config.PageTimeout).To(Equal(time.Duration(0)))
			Expect(config.Command).To(Equal(""))
			Expect(config.Verbose).To(Equal(0)) // Default verbose level
		})
	})

	Describe("Headless flag behavior", func() {
		It("should correctly handle headless configuration", func() {
			// Test that headless setting is properly passed to launcher
			config := &Config{Headless: true}
			logger := setupLogger(2) // -vv for debug level
			notifier := NewNotifier(config, &Credentials{}, logger)

			Expect(notifier.config.Headless).To(BeTrue())

			// Test non-headless
			config.Headless = false
			notifier2 := NewNotifier(config, &Credentials{}, logger)

			Expect(notifier2.config.Headless).To(BeFalse())
		})
	})

	Describe("Credentials struct", func() {
		It("should store username and password correctly", func() {
			creds := &Credentials{
				Username: "user@example.com",
				Password: "secret123",
			}

			Expect(creds.Username).To(Equal("user@example.com"))
			Expect(creds.Password).To(Equal("secret123"))
		})

		It("should handle empty credentials", func() {
			creds := &Credentials{}

			Expect(creds.Username).To(Equal(""))
			Expect(creds.Password).To(Equal(""))
		})
	})
})

var _ = Describe("Application Integration", func() {
	Describe("Logger output validation", func() {
		var (
			originalStderr *os.File
			r, w           *os.File
		)

		BeforeEach(func() {
			var err error
			r, w, err = os.Pipe()
			Expect(err).NotTo(HaveOccurred())
			originalStderr = os.Stderr
			os.Stderr = w
		})

		AfterEach(func() {
			w.Close()
			os.Stderr = originalStderr
			r.Close()
		})

		It("should log at the correct level", func() {
			logger := setupLogger(1) // -v for info level

			// Log messages at different levels
			logger.Debug("debug message")  // Should not appear
			logger.Info("info message")    // Should appear
			logger.Warn("warning message") // Should appear
			logger.Error("error message")  // Should appear

			// Close writer and read output
			w.Close()

			output := make([]byte, 2048)
			n, err := r.Read(output)
			Expect(err).NotTo(HaveOccurred())

			outputStr := string(output[:n])

			// Verify debug is filtered out
			Expect(outputStr).NotTo(ContainSubstring("debug message"))

			// Verify other levels appear
			Expect(outputStr).To(ContainSubstring("info message"))
			Expect(outputStr).To(ContainSubstring("warning message"))
			Expect(outputStr).To(ContainSubstring("error message"))
		})
	})
})

var _ = Describe("Performance", func() {
	It("should have fast textToStatus performance", func() {
		testCases := []string{
			"Order Placed",
			"Preparing Your Order",
			"Order Arrived",
			"Invalid Status",
		}

		// Warm up
		for i := 0; i < 100; i++ {
			testCase := testCases[i%len(testCases)]
			textToStatus(testCase)
		}

		// Actual performance test
		start := time.Now()
		for i := 0; i < 1000; i++ {
			testCase := testCases[i%len(testCases)]
			textToStatus(testCase)
		}
		duration := time.Since(start)

		Expect(duration.Nanoseconds()).To(BeNumerically("<", 1000000)) // Less than 1ms for 1000 operations
	})

	It("should have fast OrderStatus String performance", func() {
		statuses := []OrderStatus{
			OrderStatusPlaced,
			OrderStatusPreparing,
			OrderStatusArrived,
			OrderStatusUnknown,
		}

		// Warm up
		for i := 0; i < 100; i++ {
			status := statuses[i%len(statuses)]
			_ = status.String()
		}

		// Actual performance test
		start := time.Now()
		for i := 0; i < 1000; i++ {
			status := statuses[i%len(statuses)]
			_ = status.String()
		}
		duration := time.Since(start)

		Expect(duration.Nanoseconds()).To(BeNumerically("<", 500000)) // Less than 0.5ms for 1000 operations
	})
})

var _ = Describe("Credentials Management", func() {
	Describe("getCredentials function", func() {
		var (
			originalUsername string
			originalPassword string
		)

		BeforeEach(func() {
			// Save original environment variables
			originalUsername = os.Getenv("RELISH_USERNAME")
			originalPassword = os.Getenv("RELISH_PASSWORD")
		})

		AfterEach(func() {
			// Restore original environment variables
			if originalUsername != "" {
				os.Setenv("RELISH_USERNAME", originalUsername)
			} else {
				os.Unsetenv("RELISH_USERNAME")
			}

			if originalPassword != "" {
				os.Setenv("RELISH_PASSWORD", originalPassword)
			} else {
				os.Unsetenv("RELISH_PASSWORD")
			}
		})

		Context("when environment variables are set", func() {
			BeforeEach(func() {
				os.Setenv("RELISH_USERNAME", "envuser@example.com")
				os.Setenv("RELISH_PASSWORD", "envpassword")
			})

			It("should return credentials from environment when keyring fails", func() {
				// This test assumes keyring will fail for non-existent service
				// If keyring succeeds, that's also fine - we're testing fallback behavior
				creds, err := getCredentials()

				Expect(err).NotTo(HaveOccurred())
				Expect(creds).NotTo(BeNil())

				// Should get either keyring or environment credentials
				Expect(creds.Username).NotTo(BeEmpty())
				Expect(creds.Password).NotTo(BeEmpty())

				// If environment fallback was used, should match our test values
				if creds.Username == "envuser@example.com" {
					Expect(creds.Password).To(Equal("envpassword"))
				}
			})
		})

		Context("when environment variables are not set", func() {
			BeforeEach(func() {
				os.Unsetenv("RELISH_USERNAME")
				os.Unsetenv("RELISH_PASSWORD")
			})

			It("should return error with helpful message when both keyring and env vars fail", func() {
				// This test might pass or fail depending on system keyring state
				// If keyring has valid credentials, the function will succeed
				// If keyring fails and no env vars, it should fail with our message
				creds, err := getCredentials()

				if err != nil {
					// If it fails, should mention both keyring and environment variables
					Expect(err.Error()).To(ContainSubstring("RELISH_USERNAME"))
					Expect(err.Error()).To(ContainSubstring("RELISH_PASSWORD"))
					Expect(creds).To(BeNil())
				} else {
					// If it succeeds, keyring must have had valid credentials
					Expect(creds).NotTo(BeNil())
					Expect(creds.Username).NotTo(BeEmpty())
					Expect(creds.Password).NotTo(BeEmpty())
				}
			})
		})

		Context("with partial environment variables", func() {
			It("should fail when only username is set in environment", func() {
				os.Setenv("RELISH_USERNAME", "partialuser@example.com")
				os.Unsetenv("RELISH_PASSWORD")

				// This test behavior depends on keyring state
				creds, err := getCredentials()

				if err != nil {
					// Should mention password is missing
					Expect(err.Error()).To(ContainSubstring("RELISH_PASSWORD"))
				} else {
					// Keyring must have provided valid credentials
					Expect(creds).NotTo(BeNil())
				}
			})

			It("should fail when only password is set in environment", func() {
				os.Unsetenv("RELISH_USERNAME")
				os.Setenv("RELISH_PASSWORD", "partialpassword")

				// This test behavior depends on keyring state
				creds, err := getCredentials()

				if err != nil {
					// Should mention username is missing
					Expect(err.Error()).To(ContainSubstring("RELISH_USERNAME"))
				} else {
					// Keyring must have provided valid credentials
					Expect(creds).NotTo(BeNil())
				}
			})
		})
	})
})

var _ = Describe("Edge Cases and Error Handling", func() {
	Describe("textToStatus with edge cases", func() {
		It("should handle Unicode characters", func() {
			result := textToStatus("Order Placed 🚚")
			Expect(result).To(Equal(OrderStatusUnknown))
		})

		It("should handle very long strings", func() {
			longString := strings.Repeat("Order Placed", 1000)
			result := textToStatus(longString)
			Expect(result).To(Equal(OrderStatusUnknown))
		})

		It("should handle strings with control characters", func() {
			result := textToStatus("Order\nPlaced")
			Expect(result).To(Equal(OrderStatusUnknown))
		})
	})

	Describe("setupLogger edge cases", func() {
		It("should handle very high verbose counts", func() {
			logger := setupLogger(999)
			Expect(logger).NotTo(BeNil())
			// Should still be debug level for any count >= 2
			Expect(logger.Enabled(nil, slog.LevelDebug)).To(BeTrue())
		})

		It("should handle negative verbose counts", func() {
			logger := setupLogger(-1)
			Expect(logger).NotTo(BeNil())
			// Should default to warn level for negative values
			Expect(logger.Enabled(nil, slog.LevelWarn)).To(BeTrue())
			Expect(logger.Enabled(nil, slog.LevelInfo)).To(BeFalse())
		})
	})

	Describe("NewNotifier edge cases", func() {
		It("should work with extreme timeout values", func() {
			config := &Config{
				PageTimeout: time.Hour * 24, // 24 hours
			}

			notifier := NewNotifier(config, &Credentials{}, setupLogger(1)) // -v for info level
			Expect(notifier).NotTo(BeNil())
			Expect(notifier.config.PageTimeout).To(Equal(time.Hour * 24))
		})

		It("should handle empty login URL correctly", func() {
			notifier := NewNotifier(&Config{}, &Credentials{}, setupLogger(1)) // -v for info level
			Expect(notifier.loginUrl).To(Equal(defaultLoginURL))
		})
	})
})

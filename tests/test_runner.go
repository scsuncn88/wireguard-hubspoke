package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	testType     = flag.String("type", "all", "Test type: unit, integration, functional, or all")
	verbose      = flag.Bool("v", false, "Verbose output")
	coverage     = flag.Bool("coverage", false, "Enable coverage reporting")
	benchmarks   = flag.Bool("bench", false, "Run benchmarks")
	parallel     = flag.Int("parallel", 4, "Number of parallel test processes")
	timeout      = flag.Duration("timeout", 10*time.Minute, "Test timeout")
	outputFormat = flag.String("output", "standard", "Output format: standard, json, or junit")
)

func main() {
	flag.Parse()
	
	fmt.Println("🧪 WireGuard SD-WAN Test Runner")
	fmt.Println("================================")
	
	if err := runTests(); err != nil {
		fmt.Printf("❌ Tests failed: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("✅ All tests completed successfully!")
}

func runTests() error {
	// Set up test environment
	if err := setupTestEnvironment(); err != nil {
		return fmt.Errorf("failed to setup test environment: %w", err)
	}
	
	// Run tests based on type
	switch *testType {
	case "unit":
		return runUnitTests()
	case "integration":
		return runIntegrationTests()
	case "functional":
		return runFunctionalTests()
	case "all":
		return runAllTests()
	default:
		return fmt.Errorf("invalid test type: %s", *testType)
	}
}

func setupTestEnvironment() error {
	fmt.Println("🔧 Setting up test environment...")
	
	// Create necessary directories
	dirs := []string{
		"../tmp/test-data",
		"../tmp/test-backups",
		"../tmp/test-configs",
		"../tmp/test-logs",
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	// Set test environment variables
	testEnvVars := map[string]string{
		"GO_ENV":           "test",
		"DB_HOST":          "localhost",
		"DB_PORT":          "5432",
		"DB_NAME":          "wg_sdwan_test",
		"DB_USER":          "test_user",
		"DB_PASSWORD":      "test_password",
		"JWT_SECRET":       "test-jwt-secret-key",
		"LOG_LEVEL":        "debug",
		"CONTROLLER_HOST":  "localhost",
		"CONTROLLER_PORT":  "8080",
		"WG_INTERFACE":     "wg-test",
		"WG_SUBNET":        "10.200.0.0/16",
		"HA_ENABLED":       "false",
		"BACKUP_PATH":      "../tmp/test-backups",
		"CONFIG_PATH":      "../tmp/test-configs",
	}
	
	for key, value := range testEnvVars {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}
	
	fmt.Println("✅ Test environment setup complete")
	return nil
}

func runUnitTests() error {
	fmt.Println("🧪 Running unit tests...")
	
	args := []string{"test", "./unit/..."}
	args = append(args, getCommonTestArgs()...)
	
	cmd := exec.Command("go", args...)
	cmd.Dir = filepath.Dir(os.Args[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func runIntegrationTests() error {
	fmt.Println("🔗 Running integration tests...")
	
	args := []string{"test", "./integration/..."}
	args = append(args, getCommonTestArgs()...)
	
	cmd := exec.Command("go", args...)
	cmd.Dir = filepath.Dir(os.Args[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func runFunctionalTests() error {
	fmt.Println("🎯 Running functional tests...")
	
	args := []string{"test", "./functional/..."}
	args = append(args, getCommonTestArgs()...)
	
	cmd := exec.Command("go", args...)
	cmd.Dir = filepath.Dir(os.Args[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func runAllTests() error {
	fmt.Println("🚀 Running all tests...")
	
	testTypes := []struct {
		name string
		fn   func() error
	}{
		{"Unit Tests", runUnitTests},
		{"Integration Tests", runIntegrationTests},
		{"Functional Tests", runFunctionalTests},
	}
	
	for _, test := range testTypes {
		fmt.Printf("\n--- %s ---\n", test.name)
		if err := test.fn(); err != nil {
			return fmt.Errorf("%s failed: %w", test.name, err)
		}
	}
	
	// Generate coverage report if requested
	if *coverage {
		return generateCoverageReport()
	}
	
	return nil
}

func getCommonTestArgs() []string {
	var args []string
	
	if *verbose {
		args = append(args, "-v")
	}
	
	if *coverage {
		args = append(args, "-coverprofile=coverage.out", "-covermode=atomic")
	}
	
	if *benchmarks {
		args = append(args, "-bench=.")
	}
	
	args = append(args, fmt.Sprintf("-parallel=%d", *parallel))
	args = append(args, fmt.Sprintf("-timeout=%s", timeout.String()))
	
	switch *outputFormat {
	case "json":
		args = append(args, "-json")
	case "junit":
		// Would need additional tooling for JUnit XML output
		// For now, use standard format
	}
	
	return args
}

func generateCoverageReport() error {
	fmt.Println("📊 Generating coverage report...")
	
	// Generate HTML coverage report
	cmd := exec.Command("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate HTML coverage report: %w", err)
	}
	
	// Generate text coverage summary
	cmd = exec.Command("go", "tool", "cover", "-func=coverage.out")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to generate coverage summary: %w", err)
	}
	
	fmt.Println("📈 Coverage Summary:")
	fmt.Println(string(output))
	
	// Extract total coverage percentage
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			fmt.Printf("🎯 Total Coverage: %s\n", strings.Fields(line)[2])
			break
		}
	}
	
	return nil
}

func cleanup() {
	fmt.Println("🧹 Cleaning up test environment...")
	
	// Remove test files
	testDirs := []string{
		"../tmp/test-data",
		"../tmp/test-backups",
		"../tmp/test-configs",
		"../tmp/test-logs",
	}
	
	for _, dir := range testDirs {
		os.RemoveAll(dir)
	}
	
	// Remove coverage files
	os.Remove("coverage.out")
	os.Remove("coverage.html")
	
	fmt.Println("✅ Cleanup complete")
}
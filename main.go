package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mcp/filesystem-ultra/cache"
	"github.com/mcp/filesystem-ultra/core"
	"github.com/mcp/filesystem-ultra/mcp"
	"github.com/mcp/filesystem-ultra/protocol"
)

// Configuration holds all server configuration
type Configuration struct {
	CacheSize        int64  // Cache size in bytes
	ParallelOps      int    // Max concurrent operations
	BinaryThreshold  int64  // File size threshold for binary protocol
	VSCodeAPIEnabled bool   // Enable VSCode API integration when available
	DebugMode        bool   // Enable debug logging
	LogLevel         string // Log level (info, debug, error)
}

// DefaultConfiguration returns optimized defaults based on system
func DefaultConfiguration() *Configuration {
	// Auto-detect optimal settings based on system resources
	cpuCount := runtime.NumCPU()
	parallelOps := cpuCount * 2 // 2x CPU cores for I/O bound operations
	if parallelOps > 16 {
		parallelOps = 16 // Cap at 16 to avoid overhead
	}

	return &Configuration{
		CacheSize:        100 * 1024 * 1024, // 100MB default
		ParallelOps:      parallelOps,
		BinaryThreshold:  1024 * 1024, // 1MB threshold
		VSCodeAPIEnabled: true,
		DebugMode:        false,
		LogLevel:         "info",
	}
}

func main() {
	config := DefaultConfiguration()

	// Parse command line arguments
	var (
		cacheSize       = flag.String("cache-size", "100MB", "Memory cache limit (e.g., 50MB, 1GB)")
		parallelOps     = flag.Int("parallel-ops", config.ParallelOps, "Max concurrent operations")
		binaryThreshold = flag.String("binary-threshold", "1MB", "File size threshold for binary protocol")
		vsCodeAPI       = flag.Bool("vscode-api", true, "Enable VSCode API integration when available")
		debugMode       = flag.Bool("debug", false, "Enable debug mode")
		logLevel        = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		version         = flag.Bool("version", false, "Show version information")
		benchmark       = flag.Bool("bench", false, "Run performance benchmark")
	)
	flag.Parse()

	if *version {
		fmt.Printf("MCP Filesystem Server Ultra-Fast v1.0.0\n")
		fmt.Printf("Build: %s\n", time.Now().Format("2006-01-02"))
		fmt.Printf("Go: %s\n", runtime.Version())
		fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	// Parse cache size
	if size, err := parseSize(*cacheSize); err != nil {
		log.Fatalf("Invalid cache size: %v", err)
	} else {
		config.CacheSize = size
	}

	// Parse binary threshold
	if threshold, err := parseSize(*binaryThreshold); err != nil {
		log.Fatalf("Invalid binary threshold: %v", err)
	} else {
		config.BinaryThreshold = threshold
	}

	config.ParallelOps = *parallelOps
	config.VSCodeAPIEnabled = *vsCodeAPI
	config.DebugMode = *debugMode
	config.LogLevel = *logLevel

	// Setup logging
	setupLogging(config)

	log.Printf("🚀 Starting MCP Filesystem Server Ultra-Fast")
	log.Printf("📊 Config: Cache=%s, Parallel=%d, Binary=%s, VSCode=%v",
		formatSize(config.CacheSize), config.ParallelOps,
		formatSize(config.BinaryThreshold), config.VSCodeAPIEnabled)

	if *benchmark {
		runBenchmark(config)
		return
	}

	// Initialize components
	ctx := context.Background()

	// Initialize cache system
	cacheSystem, err := cache.NewIntelligentCache(config.CacheSize)
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}
	defer cacheSystem.Close()

	// Initialize protocol handler
	protocolHandler := protocol.NewOptimizedHandler(config.BinaryThreshold)

	// Initialize core engine
	engine, err := core.NewUltraFastEngine(&core.Config{
		Cache:            cacheSystem,
		ProtocolHandler:  protocolHandler,
		ParallelOps:      config.ParallelOps,
		VSCodeAPIEnabled: config.VSCodeAPIEnabled,
		DebugMode:        config.DebugMode,
	})
	if err != nil {
		log.Fatalf("Failed to initialize engine: %v", err)
	}
	defer engine.Close()

	// Create MCP server implementation
	impl := &mcp.Implementation{
		Name:    "filesystem-ultra",
		Version: "1.0.0",
	}

	// Create server options
	opts := &mcp.ServerOptions{}

	// Create MCP server
	server := mcp.NewServer(impl, opts)

	// Register all optimized tools
	if err := registerTools(server, engine); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start performance monitoring
	go engine.StartMonitoring(ctx)

	log.Printf("✅ Server ready - Waiting for connections...")

	// Create stdio transport and connect
	transport := mcp.NewStdioTransport()
	_, err = server.Connect(ctx, transport)
	if err != nil {
		log.Fatalf("Server connection error: %v", err)
	}
}

// parseSize parses size strings like "50MB", "1GB", etc.
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))
	
	var multiplier int64 = 1
	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	return size * multiplier, nil
}

// formatSize formats bytes to human readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes
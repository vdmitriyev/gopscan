package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var version string = "0.1.0"
var build string = "0.0.0" // do not remove or modify

const logFileName = "gopscan.log"

func newLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout", logFileName}
	config.EncoderConfig = zapcore.EncoderConfig{
		MessageKey:    "msg",
		LevelKey:      "level",
		TimeKey:       "ts",
		NameKey:       "logger",
		CallerKey:     "caller",
		FunctionKey:   zapcore.OmitKey,
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		//EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		//	enc.AppendString(t.UTC().Format("2006-01-02 15:04:05"))
		//},
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	config.Encoding = "console" // Change to "console" for line format
	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}
	return logger, nil
}

func scanPort(ctx context.Context, wg *sync.WaitGroup, hostname string, port int, logger *zap.Logger) {
	defer wg.Done()
	address := fmt.Sprintf("%s:%d", hostname, port)
	dialer := net.Dialer{Timeout: 1 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err == nil {
		logger.Info("OPEN PORT", zap.String("host", hostname), zap.Int("port", port))
		conn.Close()
	}
}

func processServer(ctx context.Context, wg *sync.WaitGroup, hostname string, ports []int, logger *zap.Logger) {
	defer wg.Done()
	var innerWg sync.WaitGroup
	for _, port := range ports {
		innerWg.Add(1)
		go scanPort(ctx, &innerWg, hostname, port, logger)
	}
	innerWg.Wait()
}

func readServersFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open servers file: %w", err)
	}
	defer file.Close()

	var servers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		server := strings.TrimSpace(scanner.Text())
		if server != "" {
			servers = append(servers, server)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read servers file: %w", err)
	}

	return servers, nil
}

func readPortsFromFiles(dirPath string) ([]int, error) {
	var allPorts []int
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".txt" {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read port file %s: %w", path, err)
			}
			portsStr := strings.Split(strings.TrimSpace(string(content)), ",")
			for _, pStr := range portsStr {
				port, err := strconv.Atoi(strings.TrimSpace(pStr))
				if err != nil {
					zap.L().Warn("Invalid port format in file", zap.String("file", path), zap.String("value", pStr))
					continue
				}
				allPorts = append(allPorts, port)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to process port files: %w", err)
	}
	return uniqueInts(allPorts), nil
}

func uniqueInts(intSlice []int) []int {
	keys := make(map[int]bool)
	list := []int{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func uniqueStrs(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func main() {
	serversFile := flag.String("servers", "servers.txt", "Path to the file containing list of servers (one per line)")
	portsDir := flag.String("portsdir", "ports", "Path to the directory containing files with comma-separated ports")
	versionFull := flag.Bool("version", false, "Prints full version of CLI")
	versionShort := flag.Bool("version-short", false, "Prints version of CLI")
	flag.Parse()

	if *versionShort {
		fmt.Println(version)
		return
	}

	if *versionFull {
		fmt.Println("Version: ", version)
		fmt.Println("Build: ", build)
		return
	}

	logger, err := newLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	servers, err := readServersFromFile(*serversFile)
	if err != nil {
		zap.L().Error("Error reading servers file", zap.Error(err))
		os.Exit(1)
	}
	zap.L().Info("Read servers from file", zap.Int("count", len(servers)), zap.String("file", *serversFile))

	ports, err := readPortsFromFiles(*portsDir)
	if err != nil {
		zap.L().Error("Error reading ports from directory", zap.Error(err))
		os.Exit(1)
	}
	zap.L().Info("Read ports from directory", zap.Int("count", len(ports)), zap.String("directory", *portsDir))

	var allHosts []string
	allHosts = append(allHosts, servers...)
	uniqueHosts := uniqueStrs(allHosts)
	zap.L().Info("Total unique hosts to scan", zap.Int("count", len(uniqueHosts)))

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, host := range uniqueHosts {
		wg.Add(1)
		go processServer(ctx, &wg, host, ports, zap.L())
	}

	wg.Wait()

	zap.L().Info("Port scan completed")
	fmt.Println("Port scan completed. Check", logFileName, "for open ports.")
}

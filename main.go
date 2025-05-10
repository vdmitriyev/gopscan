package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var version string = "0.1.1"
var build string = "0.0.0" // do not remove or modify

const appName = "gopscan"
const logFileName = "gopscan.log"

var globalPortMap = sync.Map{}

type ScanResult struct {
	Content string
}

type ServerConfig struct {
	Name         string `yaml:"name"`
	AllowedPorts []int  `yaml:"allowedPorts"`
}

type ServersConfig struct {
	Servers []ServerConfig `yaml:"servers"`
}

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
		// 	enc.AppendString(t.UTC().Format("2006-01-02 15:04:05"))
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
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err == nil {
		logger.Info("OPEN PORT", zap.String("host", hostname), zap.Int("port", port))
		// Update global map
		if hostData, ok := globalPortMap.Load(hostname); ok {
			if data, ok := hostData.(map[string]interface{}); ok {
				if ports, ok := data["ports"].([]int); ok {
					data["ports"] = append(ports, port)
					globalPortMap.Store(hostname, data)
				}
			}
		} else {
			globalPortMap.Store(hostname, map[string]interface{}{
				"server": hostname,
				"ports":  []int{port},
			})
		}
		conn.Close()
	}
}

func processServer(ctx context.Context, wg *sync.WaitGroup, serverName string, allowedPorts []int, portsToScan []int, logger *zap.Logger) {
	defer wg.Done()

	// Create a map for faster lookups of allowed ports
	allowedPortMap := make(map[int]bool)
	for _, port := range allowedPorts {
		allowedPortMap[port] = true
	}

	var filteredPortsToScan []int
	for _, port := range portsToScan {
		if !allowedPortMap[port] {
			filteredPortsToScan = append(filteredPortsToScan, port)
		}
	}

	var innerWg sync.WaitGroup
	for _, port := range filteredPortsToScan {
		innerWg.Add(1)
		go scanPort(ctx, &innerWg, serverName, port, logger)
	}
	innerWg.Wait()
}

func readServersFromFile(filePath string) (map[string][]int, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read servers file: %w", err)
	}

	var config ServersConfig
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	serversWithPorts := make(map[string][]int)
	for _, server := range config.Servers {
		serversWithPorts[server.Name] = server.AllowedPorts
	}

	return serversWithPorts, nil
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

func formatOpenPorts() string {
	var buffer bytes.Buffer
	serverPortMap := make(map[string][]int)

	globalPortMap.Range(func(key, value interface{}) bool {
		if data, ok := value.(map[string]interface{}); ok {
			if server, ok := data["server"].(string); ok {
				if ports, ok := data["ports"].([]int); ok {
					serverPortMap[server] = append(serverPortMap[server], ports...)
				}
			}
		}
		return true
	})

	var sortedServers []string
	for server := range serverPortMap {
		sortedServers = append(sortedServers, server)
	}
	sort.Strings(sortedServers)

	for _, server := range sortedServers {
		ports := serverPortMap[server]
		sort.Ints(ports)
		buffer.WriteString(fmt.Sprintf("Server: %s\nOpen ports:\n", server))
		for _, port := range ports {
			buffer.WriteString(fmt.Sprintf(" - %d\n", port))
		}
		buffer.WriteString("\n")
	}

	return buffer.String()
}

func generateReport(data ScanResult) (string, error) {
	tmpl := `Hi,

please find below list of open ports.

{{.Content}}

Best regards,
Auto-Admins
`
	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return tpl.String(), nil
}

func sendReportByEmail(report string) {
	err := SendEmail(fmt.Sprintf("%s: Open Ports Report", appName), report)
	if err != nil {
		zap.L().Error("Error while sending report by email", zap.Error(err))
	}
	fmt.Println("Report:\n", report) // Placeholder for email sending
}

func handleReport() {
	reportContent := formatOpenPorts()
	if len(reportContent) != 0 {
		reportData := ScanResult{Content: reportContent}
		report, err := generateReport(reportData)
		if err != nil {
			zap.L().Error("Error generating report", zap.Error(err))
		} else {
			sendReportByEmail(report)
		}
	} else {
		zap.L().Info("Report will not be send. No open ports that are not allowed. Nothing to report here.")
	}
}

func main() {

	serversFile := flag.String("servers", "servers.yaml", "Path to the YAML file containing list of servers and allowed ports")
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

	serversWithPorts, err := readServersFromFile(*serversFile)
	if err != nil {
		zap.L().Error("Error reading servers file", zap.Error(err))
		os.Exit(1)
	}
	zap.L().Info("Read servers from file", zap.Int("count", len(serversWithPorts)), zap.String("file", *serversFile))

	portsToScan, err := readPortsFromFiles(*portsDir)
	if err != nil {
		zap.L().Error("Error reading ports from directory", zap.Error(err))
		os.Exit(1)
	}
	zap.L().Info("Read ports from directory", zap.Int("count", len(portsToScan)), zap.String("directory", *portsDir))

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for serverName, allowedPorts := range serversWithPorts {
		wg.Add(1)
		zap.L().Info(fmt.Sprintf("Start scanning the server: %s", serverName))
		// Skip scanning of allowed ports
		go processServer(ctx, &wg, serverName, allowedPorts, portsToScan, zap.L())
	}

	wg.Wait()

	zap.L().Info("Port scan completed")
	handleReport()

	fmt.Println("Port scan completed. Check for detailed logs: ", logFileName)
}

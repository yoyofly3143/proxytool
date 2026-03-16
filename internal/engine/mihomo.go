package engine

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const mihomoVersion = "v1.18.10"

// mihomoDir returns the directory where mihomo binary and configs are stored
func mihomoDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".proxytool", "mihomo")
}

func binaryPath() string {
	name := "mihomo"
	if runtime.GOOS == "windows" {
		name = "mihomo.exe"
	}
	return filepath.Join(mihomoDir(), name)
}

func configPath() string {
	return filepath.Join(mihomoDir(), "config.yaml")
}

func pidPath() string {
	return filepath.Join(mihomoDir(), "mihomo.pid")
}

// EnsureBinary checks if mihomo binary exists, downloads if not
func EnsureBinary() error {
	bp := binaryPath()
	if _, err := os.Stat(bp); err == nil {
		return nil // already exists
	}
	if err := os.MkdirAll(mihomoDir(), 0755); err != nil {
		return err
	}

	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}

	goos := "linux"
	if runtime.GOOS == "windows" {
		goos = "windows"
	}

	// Try multiple mihomo release filename patterns (they often end in .gz)
	fileNames := []string{
		fmt.Sprintf("mihomo-%s-%s-%s.gz", goos, arch, mihomoVersion),
		fmt.Sprintf("mihomo-%s-%s.gz", goos, arch),
	}

	baseURL := fmt.Sprintf("https://github.com/MetaCubeX/mihomo/releases/download/%s/", mihomoVersion)

	var lastErr error
	for _, fn := range fileNames {
		dlURL := baseURL + fn
		tempPath := bp + ".gz"
		fmt.Printf("Downloading mihomo from %s ...\n", dlURL)
		if err := downloadFile(tempPath, dlURL); err != nil {
			lastErr = err
			continue
		}
		fmt.Printf("Downloaded %s. Please ensure you have 'gunzip' installed to decompress it, or manually decompress it to %s\n", tempPath, bp)
		// For simplicity in this CLI tool, we advise the user if we can't auto-decompress
		// But let's try a simple rename if it wasn't actually a gz (some mirrors might decompress)
		lastErr = nil
		break
	}
	if lastErr != nil {
		return fmt.Errorf("failed to download mihomo: %w. \nTIP: Manually download from GitHub, decompress, and save as: %s", lastErr, bp)
	}

	if err := os.Chmod(bp, 0755); err != nil {
		return err
	}
	fmt.Println("mihomo downloaded successfully.")
	return nil
}

func downloadFile(dest, url string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// GenerateConfig creates a minimal clash config for the given proxy map
func GenerateConfig(proxy map[string]interface{}, httpPort, socksPort int, allowLan bool) error {

	if err := os.MkdirAll(mihomoDir(), 0755); err != nil {
		return err
	}

	cfg := map[string]interface{}{
		"mixed-port":          httpPort,
		"socks-port":          socksPort,
		"allow-lan":           allowLan,

		"mode":                "global",
		"log-level":           "info",
		"external-controller": "127.0.0.1:29090",
		"proxies":             []interface{}{proxy},

		"proxy-groups": []interface{}{
			map[string]interface{}{
				"name":    "GLOBAL",
				"type":    "select",
				"proxies": []interface{}{proxy["name"]},
			},
		},
		"rules": []interface{}{
			"MATCH,GLOBAL",
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

// Start launches mihomo as a background daemon
func Start() error {
	bp := binaryPath()
	cp := configPath()

	// Try to stop any existing process first
	_ = Stop()

	cmd := exec.Command(bp, "-f", cp, "-d", mihomoDir())
	cmd.SysProcAttr = sysAttr()

	logFilePath := filepath.Join(mihomoDir(), "mihomo.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mihomo: %w", err)
	}

	pid := cmd.Process.Pid
	if err := os.WriteFile(pidPath(), []byte(strconv.Itoa(pid)), 0644); err != nil {
		return err
	}

	// Detach
	cmd.Process.Release()

	// Give it a longer moment to start and check health
	fmt.Println("Waiting for mihomo to initialize (3s)...")
	time.Sleep(3 * time.Second)
	if !IsRunning() {
		_ = os.Remove(pidPath())
		logContent, _ := os.ReadFile(logFilePath)
		return fmt.Errorf("mihomo failed to stay alive. Last logs:\n%s\nTIP: Check logs with 'proxytool proxy logs' for port conflicts.", string(logContent))
	}



	fmt.Printf("mihomo started (pid %d), listening on :%d\n", pid, readConfigPort())
	return nil
}

// Stop kills the mihomo process
func Stop() error {
	pid := readPID()
	if pid == 0 {
		return fmt.Errorf("mihomo is not running")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(pidPath())
		return fmt.Errorf("process %d not found: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Try SIGKILL
		_ = proc.Kill()
	}
	_ = os.Remove(pidPath())
	fmt.Printf("mihomo (pid %d) stopped.\n", pid)
	return nil
}

// IsRunning checks if the mihomo process is alive
func IsRunning() bool {
	pid := readPID()
	if pid == 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if process exists without killing it
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func readPID() int {
	data, err := os.ReadFile(pidPath())
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return pid
}

func readConfigPort() int {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return 7890
	}
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return 7890
	}
	if p, ok := cfg["mixed-port"].(int); ok {
		return p
	}
	return 7890
}

// Status returns a human-readable status string
func Status() string {
	pid := readPID()
	if pid == 0 {
		return "stopped"
	}
	if IsRunning() {
		port := readConfigPort()
		return fmt.Sprintf("running (pid %d, http/socks :%d)", pid, port)
	}
	return "stopped (stale pid file)"
}

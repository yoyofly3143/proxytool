package dockerproxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type daemonConfig map[string]interface{}

func daemonJSONPath() string {
	// Prefer /etc/docker/daemon.json for system Docker (typical on Linux servers)
	if _, err := os.Stat("/etc/docker"); err == nil {
		return "/etc/docker/daemon.json"
	}
	// Fallback: Docker Desktop style
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".docker", "daemon.json")
}

// Enable sets proxy config in docker daemon.json
func Enable(httpPort int) error {
	proxyVal := fmt.Sprintf("http://127.0.0.1:%d", httpPort)
	path := daemonJSONPath()

	cfg, err := loadDaemon(path)
	if err != nil {
		return err
	}

	cfg["proxies"] = map[string]interface{}{
		"http-proxy":  proxyVal,
		"https-proxy": proxyVal,
		"no-proxy":    "localhost,127.0.0.1,::1",
	}

	if err := saveDaemon(path, cfg); err != nil {
		return err
	}

	fmt.Printf("Docker proxy enabled: %s\n", proxyVal)
	fmt.Printf("Config written to: %s\n", path)
	fmt.Println("Run 'systemctl restart docker' to apply.")
	return nil
}

// Disable removes proxy config from docker daemon.json
func Disable() error {
	path := daemonJSONPath()
	cfg, err := loadDaemon(path)
	if err != nil {
		return err
	}

	delete(cfg, "proxies")

	if err := saveDaemon(path, cfg); err != nil {
		return err
	}

	fmt.Printf("Docker proxy disabled in %s\n", path)
	fmt.Println("Run 'systemctl restart docker' to apply.")
	return nil
}

// Status reads current Docker proxy status
func Status() string {
	path := daemonJSONPath()
	cfg, err := loadDaemon(path)
	if err != nil {
		return fmt.Sprintf("unknown (cannot read %s)", path)
	}
	if proxies, ok := cfg["proxies"].(map[string]interface{}); ok {
		if hp, ok := proxies["http-proxy"].(string); ok && hp != "" {
			return fmt.Sprintf("enabled (%s) [%s]", hp, path)
		}
	}
	return fmt.Sprintf("disabled [%s]", path)
}

func loadDaemon(path string) (daemonConfig, error) {
	cfg := make(daemonConfig)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create parent dir if needed
			_ = os.MkdirAll(filepath.Dir(path), 0755)
			return cfg, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid daemon.json: %w", err)
	}
	return cfg, nil
}

func saveDaemon(path string, cfg daemonConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

package sysproxy

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const envFile = "/etc/environment"

// Enable sets http/https proxy in /etc/environment
func Enable(host string, port int) error {
	proxyVal := fmt.Sprintf("http://%s:%d", host, port)
	lines, err := readLines(envFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	vars := map[string]string{
		"http_proxy":  proxyVal,
		"https_proxy": proxyVal,
		"HTTP_PROXY":  proxyVal,
		"HTTPS_PROXY": proxyVal,
		"no_proxy":    "localhost,127.0.0.1,::1",
		"NO_PROXY":    "localhost,127.0.0.1,::1",
	}

	// Remove old proxy lines
	var filtered []string
	for _, line := range lines {
		key := strings.SplitN(line, "=", 2)[0]
		if _, skip := vars[strings.TrimSpace(key)]; !skip {
			filtered = append(filtered, line)
		}
	}

	// Append new values
	for k, v := range vars {
		filtered = append(filtered, fmt.Sprintf("%s=%s", k, v))
	}

	if err := writeLines(envFile, filtered); err != nil {
		return err
	}

	// Also set for current process
	for k, v := range vars {
		os.Setenv(k, v)
	}

	fmt.Printf("System proxy enabled: %s\n", proxyVal)
	fmt.Println("Note: run 'source /etc/environment' or re-login for shell-level effect.")
	return nil
}

// Disable removes proxy settings from /etc/environment
func Disable() error {
	lines, err := readLines(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	proxyKeys := map[string]bool{
		"http_proxy": true, "https_proxy": true,
		"HTTP_PROXY": true, "HTTPS_PROXY": true,
		"no_proxy": true, "NO_PROXY": true,
	}

	var filtered []string
	for _, line := range lines {
		key := strings.TrimSpace(strings.SplitN(line, "=", 2)[0])
		if !proxyKeys[key] {
			filtered = append(filtered, line)
		}
	}

	if err := writeLines(envFile, filtered); err != nil {
		return err
	}

	for k := range proxyKeys {
		os.Unsetenv(k)
	}

	fmt.Println("System proxy disabled.")
	fmt.Println("Note: run 'source /etc/environment' or re-login for shell-level effect.")
	return nil
}

// Status reads current proxy status from /etc/environment
func Status() string {
	lines, err := readLines(envFile)
	if err != nil {
		return "unknown (cannot read /etc/environment)"
	}
	for _, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "http_proxy" && parts[1] != "" {
			return fmt.Sprintf("enabled (%s)", parts[1])
		}
	}
	return "disabled"
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeLines(path string, lines []string) error {
	content := strings.Join(lines, "\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0644)
}

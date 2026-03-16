package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Node represents a parsed proxy node
type Node struct {
	Name   string
	Type   string // ss, vmess, trojan, vless, etc.
	Server string
	Port   int
	// Raw clash proxy map for config generation
	Raw map[string]interface{}
}

// ClashConfig is a minimal clash YAML structure
type ClashConfig struct {
	Proxies []map[string]interface{} `yaml:"proxies"`
}

// Download fetches a subscription URL
func Download(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "clash.meta")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// Parse auto-detects and parses subscription content (Clash YAML or V2Ray base64)
func Parse(data []byte) ([]Node, error) {
	content := strings.TrimSpace(string(data))

	if looksLikeClash(content) {
		return parseClash(data)
	}

	decoded, err := base64Decode(content)
	if err == nil && len(decoded) > 0 {
		lines := strings.Split(strings.TrimSpace(string(decoded)), "\n")
		var nodes []Node
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			n, err := parseURI(line)
			if err == nil {
				nodes = append(nodes, n)
			}
		}
		if len(nodes) > 0 {
			return nodes, nil
		}
	}

	return nil, fmt.Errorf("unrecognized subscription format")
}

func looksLikeClash(content string) bool {
	return strings.Contains(content, "proxies:") ||
		strings.HasPrefix(content, "port:") ||
		strings.HasPrefix(content, "mixed-port:")
}

func parseClash(data []byte) ([]Node, error) {
	var cfg ClashConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid clash yaml: %w", err)
	}
	var nodes []Node
	for _, p := range cfg.Proxies {
		name, _ := p["name"].(string)
		typ, _ := p["type"].(string)
		server, _ := p["server"].(string)
		port := 0
		switch v := p["port"].(type) {
		case int:
			port = v
		case float64:
			port = int(v)
		}
		if name == "" || server == "" {
			continue
		}
		nodes = append(nodes, Node{
			Name:   name,
			Type:   typ,
			Server: server,
			Port:   port,
			Raw:    p,
		})
	}
	return nodes, nil
}

func parseURI(line string) (Node, error) {
	u, err := url.Parse(line)
	if err != nil {
		return Node{}, err
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "vmess":
		return parseVmess(line)
	case "ss":
		return parseSS(u)
	case "trojan":
		return parseTrojan(u)
	case "vless":
		return parseVless(u)
	default:
		return Node{}, fmt.Errorf("unsupported scheme: %s", scheme)
	}
}

func parseVmess(line string) (Node, error) {
	b64 := strings.TrimPrefix(line, "vmess://")
	decoded, err := base64Decode(b64)
	if err != nil {
		return Node{}, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(decoded, &m); err != nil {
		return Node{}, fmt.Errorf("invalid vmess json: %w", err)
	}
	name, _ := m["ps"].(string)
	if name == "" {
		if add, ok := m["add"].(string); ok {
			name = add
		}
	}
	server, _ := m["add"].(string)
	port := parseIntField(m["port"])
	return Node{
		Name:   name,
		Type:   "vmess",
		Server: server,
		Port:   port,
		Raw:    buildVmessRaw(m, name),
	}, nil
}

func parseSS(u *url.URL) (Node, error) {
	host := u.Hostname()
	port := parseIntStr(u.Port())
	name := u.Fragment
	if name == "" {
		name = host
	}
	method, password := "", ""
	if u.User != nil {
		userinfo := u.User.Username()
		decoded, err := base64Decode(userinfo)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				method = parts[0]
				password = parts[1]
			}
		} else {
			method = userinfo
			password, _ = u.User.Password()
		}
	}
	raw := map[string]interface{}{
		"name": name, "type": "ss",
		"server": host, "port": port,
		"cipher": method, "password": password,
	}
	return Node{Name: name, Type: "ss", Server: host, Port: port, Raw: raw}, nil
}

func parseTrojan(u *url.URL) (Node, error) {
	host := u.Hostname()
	port := parseIntStr(u.Port())
	name := u.Fragment
	if name == "" {
		name = host
	}
	password := ""
	if u.User != nil {
		password = u.User.Username()
	}
	raw := map[string]interface{}{
		"name": name, "type": "trojan",
		"server": host, "port": port,
		"password": password, "tls": true, "skip-cert-verify": true,
	}
	return Node{Name: name, Type: "trojan", Server: host, Port: port, Raw: raw}, nil
}

func parseVless(u *url.URL) (Node, error) {
	host := u.Hostname()
	port := parseIntStr(u.Port())
	name := u.Fragment
	if name == "" {
		name = host
	}
	uuid := ""
	if u.User != nil {
		uuid = u.User.Username()
	}
	q := u.Query()
	raw := map[string]interface{}{
		"name":             name,
		"type":             "vless",
		"server":           host,
		"port":             port,
		"uuid":             uuid,
		"network":          q.Get("type"),
		"tls":              q.Get("security") == "tls" || q.Get("security") == "reality",
		"skip-cert-verify": true,
		"udp":              true,
	}

	if q.Get("security") == "reality" {
		raw["reality-opts"] = map[string]interface{}{
			"public-key": q.Get("pbk"),
			"short-id":   q.Get("sid"),
		}
		raw["servername"] = q.Get("sni")
		if fp := q.Get("fp"); fp != "" {
			raw["client-fingerprint"] = fp
		}
	} else if q.Get("security") == "tls" {
		raw["servername"] = q.Get("sni")
		if fp := q.Get("fp"); fp != "" {
			raw["client-fingerprint"] = fp
		}
	}


	if alpn := q.Get("alpn"); alpn != "" {
		raw["alpn"] = strings.Split(alpn, ",")
	}


	if flow := q.Get("flow"); flow != "" {
		raw["flow"] = flow
	}

	if q.Get("type") == "ws" {
		wsOpts := map[string]interface{}{}
		if path := q.Get("path"); path != "" {
			wsOpts["path"] = path
		}
		if host := q.Get("host"); host != "" {
			wsOpts["headers"] = map[string]interface{}{"Host": host}
		}
		if len(wsOpts) > 0 {
			raw["ws-opts"] = wsOpts
		}
	} else if q.Get("type") == "grpc" {
		raw["grpc-opts"] = map[string]interface{}{
			"grpc-service-name": q.Get("serviceName"),
		}
	}

	return Node{Name: name, Type: "vless", Server: host, Port: port, Raw: raw}, nil
}

func buildVmessRaw(m map[string]interface{}, name string) map[string]interface{} {
	port := parseIntField(m["port"])
	alterId := parseIntField(m["aid"])
	network, _ := m["net"].(string)
	if network == "" {
		network = "tcp"
	}
	raw := map[string]interface{}{
		"name": name, "type": "vmess",
		"server": m["add"], "port": port,
		"uuid": m["id"], "alterId": alterId,
		"cipher": "auto", "network": network,
	}
	if tls, _ := m["tls"].(string); tls == "tls" {
		raw["tls"] = true
		raw["skip-cert-verify"] = true
	}
	wsOpts := map[string]interface{}{}
	if host, ok := m["host"].(string); ok && host != "" {
		wsOpts["headers"] = map[string]interface{}{"Host": host}
	}
	if path, ok := m["path"].(string); ok && path != "" {
		wsOpts["path"] = path
	}
	if len(wsOpts) > 0 && network == "ws" {
		raw["ws-opts"] = wsOpts
	}
	return raw
}

func parseIntField(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		n := 0
		fmt.Sscanf(val, "%d", &n)
		return n
	}
	return 0
}

func parseIntStr(s string) int {
	n := 0
	fmt.Sscanf(s, "%d", &n)
	return n
}

func base64Decode(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	encodings := []base64.Encoding{
		*base64.StdEncoding,
		*base64.URLEncoding,
		*base64.RawStdEncoding,
		*base64.RawURLEncoding,
	}
	// Pad
	pad := s
	if mod := len(pad) % 4; mod != 0 {
		pad += strings.Repeat("=", 4-mod)
	}
	for _, enc := range encodings {
		if decoded, err := enc.DecodeString(pad); err == nil {
			return decoded, nil
		}
		if decoded, err := enc.DecodeString(s); err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("not valid base64")
}

// CachePath returns the local cache file path for a subscription name
func CachePath(name string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".proxytool", "subs", name+".cache")
}

// SaveCache saves raw subscription data to disk
func SaveCache(name string, data []byte) error {
	p := CachePath(name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// LoadCache loads cached subscription data
func LoadCache(name string) ([]byte, error) {
	return os.ReadFile(CachePath(name))
}

// Package deploy prepares and executes reproducible deployment operations.
package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

var serviceName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func Unit(name, bin, dir string) (string, error) {
	if !serviceName.MatchString(name) {
		return "", fmt.Errorf("invalid service name")
	}
	for _, p := range []string{bin, dir} {
		if !filepath.IsAbs(p) || strings.ContainsAny(p, "\r\n\x00") {
			return "", fmt.Errorf("binary and working directory require absolute paths")
		}
	}
	quote := func(s string) string {
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, `%`, `%%`).Replace(s) + `"`
	}
	return fmt.Sprintf("[Unit]\nDescription=LIAF service (%s)\nAfter=network.target\n\n[Service]\nType=simple\nWorkingDirectory=%s\nExecStart=%s\nEnvironmentFile=-/etc/liaf/%s.env\nRestart=on-failure\nRestartSec=3\nNoNewPrivileges=true\n\n[Install]\nWantedBy=multi-user.target\n", name, quote(dir), quote(bin), name), nil
}

// BindCaddy adds/replaces one host route using the Caddy configuration API.
// The ETag prevents overwriting another administrator's concurrent changes.
func BindCaddy(ctx context.Context, client *http.Client, admin, server, domain, upstream string) error {
	if !serviceName.MatchString(server) {
		return fmt.Errorf("invalid Caddy server name")
	}
	if domain == "" || strings.ContainsAny(domain, " /\\\r\n:*?") {
		return fmt.Errorf("invalid domain")
	}
	if _, _, err := net.SplitHostPort(upstream); err != nil {
		return fmt.Errorf("invalid upstream: %w", err)
	}
	u, err := url.Parse(admin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("invalid admin URL")
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/config/apps/http/servers/" + server + "/routes"
	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	resp.Body.Close()
	if readErr != nil {
		return readErr
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Caddy GET: %d: %s", resp.StatusCode, body)
	}
	var routes []map[string]any
	if err = json.Unmarshal(body, &routes); err != nil {
		return err
	}
	id := "liaf-" + domain
	route := map[string]any{"@id": id, "match": []any{map[string]any{"host": []string{domain}}}, "handle": []any{map[string]any{"handler": "reverse_proxy", "upstreams": []any{map[string]any{"dial": upstream}}}}, "terminal": true}
	updated := []map[string]any{route}
	for _, r := range routes {
		if r["@id"] != id {
			updated = append(updated, r)
		}
	}
	payload, err := json.Marshal(updated)
	if err != nil {
		return err
	}
	req, _ = http.NewRequestWithContext(ctx, "PATCH", u.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if etag := resp.Header.Get("ETag"); etag != "" {
		req.Header.Set("If-Match", etag)
	}
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ = io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Caddy PATCH: %d: %s", resp.StatusCode, body)
	}
	return nil
}

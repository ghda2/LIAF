package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

func adminJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func adminError(w http.ResponseWriter, status int, message string) {
	adminJSON(w, status, map[string]any{"status": "error", "message": message})
}

func (s *Server) serveAdmin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/_liaf/publish" && r.URL.Path != "/_liaf/reload" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		adminError(w, 405, "Use POST")
		return
	}
	if s.options.DeployToken == "" {
		adminError(w, 503, "Configure LIAF_DEPLOY_TOKEN to enable administration")
		return
	}
	provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if provided == "" {
		provided = r.Header.Get("X-LIAF-Token")
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(s.options.DeployToken)) != 1 {
		adminError(w, 401, "Invalid deployment token")
		return
	}
	start := time.Now()
	if r.URL.Path == "/_liaf/reload" {
		if err := s.Reload(); err != nil {
			adminError(w, 500, err.Error())
			return
		}
		adminJSON(w, 200, map[string]any{"status": "ok", "files": s.CurrentCache().Count(), "reload_ms": float64(time.Since(start).Microseconds()) / 1000})
		return
	}
	if GetEmbeddedFS() != nil || s.options.Dir == "" {
		adminError(w, 409, "Embedded assets are read-only; use a disk site for publishing")
		return
	}
	target := r.Header.Get("X-LIAF-Path")
	if target == "" {
		target = r.URL.Query().Get("path")
	}
	target = strings.TrimPrefix(target, "/")
	if target == "" || target == "." || strings.ContainsAny(target, "\\:\x00") || strings.HasPrefix(target, "/_") || path.Clean(target) != target {
		adminError(w, 400, "Invalid publication path")
		return
	}
	for _, part := range strings.Split(target, "/") {
		if part == ".." || strings.HasPrefix(part, ".") {
			adminError(w, 400, "Invalid publication path")
			return
		}
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.options.MaxPublishSize))
	if err != nil {
		adminError(w, 413, "Publication exceeds body limit or is unreadable")
		return
	}
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	root, err := os.OpenRoot(s.options.Dir)
	if err != nil {
		adminError(w, 500, err.Error())
		return
	}
	defer root.Close()
	if err = root.MkdirAll(path.Dir(target), 0755); err != nil {
		adminError(w, 400, err.Error())
		return
	}
	// A hidden sibling is written completely before replacing the destination.
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		adminError(w, 500, err.Error())
		return
	}
	tmp := path.Join(path.Dir(target), ".liaf-publish-"+hex.EncodeToString(nonce[:]))
	f, err := root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		adminError(w, 500, err.Error())
		return
	}
	defer root.Remove(tmp)
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		adminError(w, 500, err.Error())
		return
	}
	old, readErr := root.ReadFile(target)
	if readErr != nil && !os.IsNotExist(readErr) {
		adminError(w, 400, readErr.Error())
		return
	}
	if err = root.Rename(tmp, target); err != nil {
		adminError(w, 500, err.Error())
		return
	}
	if err = s.reloadLocked(); err != nil {
		var rollback error
		if readErr == nil {
			rollback = root.WriteFile(tmp, old, 0644)
			if rollback == nil {
				rollback = root.Rename(tmp, target)
			}
		} else {
			rollback = root.Remove(target)
		}
		if rollback != nil {
			err = fmt.Errorf("reload: %v; rollback: %w", err, rollback)
		}
		adminError(w, 500, err.Error())
		return
	}
	adminJSON(w, 200, map[string]any{"status": "published", "path": "/" + target, "files": s.CurrentCache().Count(), "reload_ms": float64(time.Since(start).Microseconds()) / 1000})
}

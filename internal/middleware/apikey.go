package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"
)

type KeyManager struct {
	mu   sync.RWMutex
	keys map[string]string
	path string
	done chan struct{}
	once sync.Once
}

func NewKeyManager(path string) *KeyManager {
	km := &KeyManager{
		keys: make(map[string]string),
		path: path,
		done: make(chan struct{}),
	}
	if err := km.reload(); err != nil {
		slog.Warn("Gagal memuat api_keys.json saat startup", "path", path, "error", err)
	}
	go km.runReload(60 * time.Second)
	return km
}

func (km *KeyManager) reload() error {
	data, err := os.ReadFile(km.path)
	if err != nil {
		return err
	}
	var newKeys map[string]string
	if err := json.Unmarshal(data, &newKeys); err != nil {
		return err
	}
	km.mu.Lock()
	km.keys = newKeys
	km.mu.Unlock()
	return nil
}

func (km *KeyManager) Lookup(key string) (string, bool) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	developer, ok := km.keys[key]
	return developer, ok
}

func (km *KeyManager) runReload(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := km.reload(); err != nil {
				slog.Warn("Gagal reload api_keys.json, mempertahankan daftar lama", "path", km.path, "error", err)
			}
		case <-km.done:
			return
		}
	}
}

func (km *KeyManager) Stop() {
	km.once.Do(func() {
		close(km.done)
	})
}

func (km *KeyManager) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tCtx := logger.GetTraceContext(r.Context())
			key := r.Header.Get("X-API-KEY")

			if key == "" {
				response.Error(w, r, http.StatusUnauthorized, "Akses ditolak: Header X-API-KEY kosong")
				return
			}

			developerName, exists := km.Lookup(key)
			if !exists {
				response.Error(w, r, http.StatusUnauthorized, "Akses ditolak: API Key tidak terdaftar")
				return
			}

			tCtx.Developer = developerName
			next.ServeHTTP(w, r)
		})
	}
}

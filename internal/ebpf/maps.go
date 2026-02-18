package ebpf

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

type CacheKey struct {
	Method      string
	Path        string
	Host        string
	ContentType string
	Accept      string
}

func NewCacheKey(r *http.Request) CacheKey {
	return CacheKey{
		Method:      r.Method,
		Path:        r.URL.Path,
		Host:        r.Host,
		ContentType: r.Header.Get("Content-Type"),
		Accept:      r.Header.Get("Accept"),
	}
}

func (k CacheKey) Hash() string {
	parts := []string{
		k.Method,
		k.Path,
		k.Host,
	}

	if k.ContentType != "" {
		parts = append(parts, k.ContentType)
	}
	if k.Accept != "" && !strings.Contains(k.Accept, "*/*") {
		parts = append(parts, k.Accept)
	}

	combined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:16])
}

func (k CacheKey) String() string {
	return k.Method + " " + k.Path
}

type CacheValue struct {
	Status      int
	Headers     map[string]string
	Body        []byte
	ContentType string
}

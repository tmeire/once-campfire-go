package web

import (
	"bytes"
	"encoding/hex"
	"hash"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/basecamp/once-campfire-go/internal/database"
)

func notModified(w http.ResponseWriter, r *http.Request, etag string, modified time.Time) bool {
	if r.Method != "GET" && r.Method != "HEAD" {
		return false
	}
	fresh := false
	if value, ok := r.Header["If-None-Match"]; ok {
		for _, line := range value {
			for _, tag := range strings.Split(line, ",") {
				if strings.TrimSpace(tag) == etag || strings.TrimSpace(tag) == "*" {
					fresh = true
				}
			}
		}
	} else if since, err := http.ParseTime(r.Header.Get("If-Modified-Since")); err == nil && !modified.IsZero() {
		fresh = !since.Before(modified.Truncate(time.Second))
	}
	if fresh {
		w.Header().Del("Content-Type")
		w.Header().Del("Content-Length")
		w.WriteHeader(http.StatusNotModified)
	}
	return fresh
}
func messageFreshness(w http.ResponseWriter, r *http.Request, messages []database.Message) bool {
	h := sha256Pool.Get().(hash.Hash)
	h.Reset()
	defer sha256Pool.Put(h)

	var buf [64]byte
	var modified time.Time
	for _, m := range messages {
		b := append(buf[:0], "messages/"...)
		b = strconv.AppendInt(b, m.ID, 10)
		b = append(b, '-')
		start := len(b)
		b = m.UpdatedAt.UTC().AppendFormat(b, "20060102150405.000000")
		dot := bytes.IndexByte(b[start:], '.')
		if dot != -1 {
			copy(b[start+dot:], b[start+dot+1:])
			b = b[:len(b)-1]
		}
		b = append(b, '/')
		h.Write(b)
		if m.UpdatedAt.After(modified) {
			modified = m.UpdatedAt
		}
	}
	if r.Header.Get("Turbo-Frame") != "" {
		h.Write([]byte("frame/"))
	}
	h.Write([]byte("messages/index"))
	var digest [32]byte
	h.Sum(digest[:0])
	etag := "W/\"" + hex.EncodeToString(digest[:16]) + "\""
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", modified.UTC().Format(http.TimeFormat))
	w.Header().Set("Cache-Control", "max-age=0, private, must-revalidate")
	return notModified(w, r, etag, modified)
}

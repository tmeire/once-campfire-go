package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/basecamp/once-campfire-go/internal/database"
	"github.com/basecamp/once-campfire-go/internal/rails"
)

type discardRecorder struct {
	header http.Header
	status int
}

func (d *discardRecorder) Header() http.Header {
	if d.header == nil {
		d.header = make(http.Header)
	}
	return d.header
}

func (d *discardRecorder) Write(b []byte) (int, error) {
	return len(b), nil
}

func (d *discardRecorder) WriteHeader(status int) {
	d.status = status
}

func setupBenchmarkRoom(b *testing.B) (*Server, *http.Cookie, int64) {
	b.Helper()
	root := b.TempDir()
	db, err := database.Open(filepath.Join(root, "bench.sqlite3"), 4)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { db.Close() })

	secrets, err := rails.NewSecrets("bench-secrets-very-secure-key-1234567890")
	if err != nil {
		b.Fatal(err)
	}

	app, err := New(db, secrets, false, root)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(app.Close)

	ctx := context.Background()
	user, err := db.Setup(ctx, "Bench User", "bench@test", "digest")
	if err != nil {
		b.Fatal(err)
	}

	token, err := db.StartSession(ctx, user.ID, "bench-agent", "127.0.0.1")
	if err != nil {
		b.Fatal(err)
	}

	signed, err := secrets.SignCookie("session_token", token, time.Now().Add(time.Hour))
	if err != nil {
		b.Fatal(err)
	}
	cookie := &http.Cookie{Name: "session_token", Value: rails.EscapeCookie(signed)}

	rooms, err := db.Rooms(ctx, user.ID)
	if err != nil || len(rooms) == 0 {
		b.Fatal("no rooms created during setup")
	}
	room := rooms[0]

	// Seed 40 messages to represent a realistic full room page
	for i := 1; i <= 40; i++ {
		body := fmt.Sprintf("<p>Message number %d in the benchmark room with some <strong>bold</strong> text.</p>", i)
		plain := fmt.Sprintf("Message number %d in the benchmark room with some bold text.", i)
		_, err := db.CreateMessage(ctx, user.ID, room.ID, fmt.Sprintf("bench-%d", i), body, plain)
		if err != nil {
			b.Fatal(err)
		}
	}

	return app, cookie, room.ID
}

func BenchmarkRoomPage(b *testing.B) {
	app, cookie, roomID := setupBenchmarkRoom(b)
	path := fmt.Sprintf("/rooms/%d", roomID)

	// Warmup run to populate template/fragment caches
	req := httptest.NewRequest("GET", path, nil)
	req.AddCookie(cookie)
	rec := &discardRecorder{}
	app.ServeHTTP(rec, req)
	if rec.status != 200 {
		b.Fatalf("warmup failed with status %d", rec.status)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", path, nil)
		req.AddCookie(cookie)
		rec := &discardRecorder{}
		app.ServeHTTP(rec, req)
		if rec.status != 200 {
			b.Fatalf("request failed with status %d", rec.status)
		}
	}
}

func BenchmarkRoomPageParallel(b *testing.B) {
	app, cookie, roomID := setupBenchmarkRoom(b)
	path := fmt.Sprintf("/rooms/%d", roomID)

	// Warmup run
	req := httptest.NewRequest("GET", path, nil)
	req.AddCookie(cookie)
	rec := &discardRecorder{}
	app.ServeHTTP(rec, req)
	if rec.status != 200 {
		b.Fatalf("warmup failed with status %d", rec.status)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", path, nil)
			req.AddCookie(cookie)
			rec := &discardRecorder{}
			app.ServeHTTP(rec, req)
			if rec.status != 200 {
				b.Fatalf("request failed with status %d", rec.status)
			}
		}
	})
}

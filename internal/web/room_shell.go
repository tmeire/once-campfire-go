package web

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"html/template"
	"strings"
	"sync"
)

var sha256Pool = sync.Pool{
	New: func() any { return sha256.New() },
}

func roomShellKey(p *page) string {
	h := sha256Pool.Get().(hash.Hash)
	h.Reset()
	defer sha256Pool.Put(h)

	var b [8]byte
	putInt := func(v int64) {
		binary.LittleEndian.PutUint64(b[:], uint64(v))
		h.Write(b[:])
	}
	putBool := func(v bool) {
		if v {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
	}
	putString := func(s string) {
		putInt(int64(len(s)))
		h.Write([]byte(s))
	}

	putInt(p.User.ID)
	putInt(int64(p.User.Role))
	putString(p.User.Name)
	putInt(p.User.UpdatedAt.UnixNano())

	putInt(p.Room.ID)
	putString(p.Room.Name)
	putString(p.Room.Type)
	putInt(p.Room.UpdatedAt.UnixNano())

	putBool(p.Invitation)
	putString(p.JoinCode)
	putString(p.Stream)
	putString(p.Title)
	putString(p.Origin)
	putString(p.Platform.Browser)
	putString(p.Platform.OperatingSystem)
	putBool(p.Platform.IOS)
	putBool(p.Platform.Android)
	putBool(p.Platform.Mac)
	putBool(p.Platform.Windows)
	putBool(p.Platform.Mobile)
	putString(p.Notice)
	putString(p.Error)
	putString(string(p.CustomStyles))

	putInt(p.Account.ID)
	putInt(p.Account.UpdatedAt.UnixNano())
	putBool(p.Account.HasLogo)
	putString(p.VAPIDPublicKey)
	putString(p.Account.JoinCode)

	putBool(p.Frame)
	putBool(p.CanCreateRooms)
	putBool(p.Reload)
	putBool(p.Chat)
	putString(p.Screen)
	putString(p.BodyClass)

	var digest [32]byte
	h.Sum(digest[:0])
	return "room-shell/" + hex.EncodeToString(digest[:])
}

func (s *Server) roomShellEntry(p page) (fragmentEntry, error) {
	p.Messages, p.RawMessages, p.MessagesHTML, p.LoadedAt = nil, nil, "", ""
	key := roomShellKey(&p)
	entry, ok := s.fragments.entry(key)
	if !ok {
		messageMarker := "\x00campfire-" + rand.Text() + "\x00"
		loadedMarker := "campfire-loaded-" + rand.Text()
		p.MessagesHTML, p.LoadedAt = template.HTML(messageMarker), loadedMarker
		b := borrowBuffer()
		defer releaseBuffer(b)
		if err := s.templates.ExecuteTemplate(b, "room", p); err != nil {
			return fragmentEntry{}, err
		}
		raw := b.Bytes()
		idxLoaded := bytes.Index(raw, []byte(loadedMarker))
		idxMsg := bytes.Index(raw, []byte(messageMarker))
		var prefix, middle, suffix []byte
		var suffixDigest [32]byte
		if idxLoaded != -1 && idxMsg != -1 && idxLoaded < idxMsg {
			prefix = bytes.Clone(raw[:idxLoaded])
			middle = bytes.Clone(raw[idxLoaded+len(loadedMarker) : idxMsg])
			suffix = bytes.Clone(raw[idxMsg+len(messageMarker):])
			suffixDigest = sha256.Sum256(suffix)
		}
		entry = s.fragments.putEntry(fragmentEntry{
			key:           key,
			html:          template.HTML(b.String()),
			messageMarker: messageMarker,
			loadedMarker:  loadedMarker,
			prefix:        prefix,
			middle:        middle,
			suffix:        suffix,
			suffixDigest:  suffixDigest,
		})
	}
	return entry, nil
}

// Cache only the surrounding HTML. Authorization and page data are read afresh;
// messages and the refresh timestamp are inserted separately for each request.
func (s *Server) roomShell(p page) (string, string, error) {
	loadedAt := p.LoadedAt
	entry, err := s.roomShellEntry(p)
	if err != nil {
		return "", "", err
	}
	if entry.prefix != nil {
		var b strings.Builder
		b.Grow(len(entry.prefix) + len(loadedAt) + len(entry.middle) + len(entry.messageMarker) + len(entry.suffix))
		b.Write(entry.prefix)
		b.WriteString(loadedAt)
		b.Write(entry.middle)
		b.WriteString(entry.messageMarker)
		b.Write(entry.suffix)
		return b.String(), entry.messageMarker, nil
	}
	return strings.ReplaceAll(string(entry.html), entry.loadedMarker, loadedAt), entry.messageMarker, nil
}

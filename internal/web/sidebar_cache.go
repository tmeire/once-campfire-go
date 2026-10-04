package web

import (
	"encoding/hex"
	"hash"
	"strconv"

	"github.com/basecamp/once-campfire-go/internal/database"
)

// Key every value the sidebar template reads. Authorization and membership data
// are still read afresh before looking up the rendered fragment.
func sidebarCacheKey(p page) string {
	h := sha256Pool.Get().(hash.Hash)
	h.Reset()
	defer sha256Pool.Put(h)

	var buf [64]byte
	user := func(u database.User) {
		b := append(buf[:0], 'u')
		b = strconv.AppendInt(b, u.ID, 10)
		b = append(b, '/')
		b = strconv.AppendInt(b, u.UpdatedAt.UnixMicro(), 10)
		b = append(b, '/')
		b = strconv.AppendInt(b, int64(len(u.Name)), 10)
		b = append(b, ':')
		h.Write(b)
		h.Write([]byte(u.Name))
		h.Write([]byte{'/'})
	}
	user(p.User)
	b := strconv.AppendBool(buf[:0], p.CanCreateRooms)
	b = append(b, '/')
	h.Write(b)
	h.Write([]byte(p.RoomsStream))
	h.Write([]byte{'/'})
	h.Write([]byte(p.UserRoomsStream))
	h.Write([]byte{'/'})
	for _, room := range p.SidebarRooms {
		b := append(buf[:0], 'r')
		b = strconv.AppendInt(b, room.ID, 10)
		b = append(b, '/')
		b = strconv.AppendInt(b, room.UpdatedAt.UnixMicro(), 10)
		b = append(b, '/')
		b = strconv.AppendBool(b, room.Unread)
		b = append(b, '/')
		b = strconv.AppendInt(b, int64(len(room.Type)), 10)
		b = append(b, ':')
		h.Write(b)
		h.Write([]byte(room.Type))
		b = append(buf[:0], '/')
		b = strconv.AppendInt(b, int64(len(room.Name)), 10)
		b = append(b, ':')
		h.Write(b)
		h.Write([]byte(room.Name))
		h.Write([]byte{'/'})
		for _, member := range room.Members {
			user(member)
		}
		h.Write([]byte{';'})
	}
	h.Write([]byte{'|'})
	for _, member := range p.Placeholders {
		user(member)
	}
	var digest [32]byte
	h.Sum(digest[:0])
	return "sidebar/" + hex.EncodeToString(digest[:])
}

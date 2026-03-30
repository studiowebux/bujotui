package storage

import (
	"crypto/rand"
	"encoding/hex"
)

// newEntryID generates a random 16-char hex string (8 bytes of entropy).
// Used as a stable identifier for entries to enable merge deduplication.
func newEntryID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

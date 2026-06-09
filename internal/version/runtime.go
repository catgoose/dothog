package version

import (
	"crypto/rand"
	"encoding/hex"
)

// RuntimeID is a unique identifier set once at application startup. It lets
// log queries scope to a single process lifetime (e.g. jq '.runtime_id == "..."').
var RuntimeID string

func init() {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err == nil {
		RuntimeID = hex.EncodeToString(bytes)
	}
}

package internal

import (
	"fmt"
)

// BuildQueryName builds a DNS query name from protocol fields
func BuildQueryName(nonce, seq, sessionID string, dataLabel string) string {
	return fmt.Sprintf("%s-%s-%s.%s.aura.net.", nonce, seq, sessionID, dataLabel)
}

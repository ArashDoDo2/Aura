package internal
package internal

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// e.g. 1a2b-0001-abcd.data.aura.net
	queryRegexp = regexp.MustCompile(`^([0-9a-fA-F]{4})-([0-9a-fA-F]{4})-([0-9a-fA-F]{4})\\.([a-z2-7]{1,63})\\.aura\\.net\\.?$`)
)

type QueryFields struct {
	Nonce     string
	Seq       string
	SessionID string
	DataLabel string
}

// ParseQueryName parses the DNS query name into protocol fields
func ParseQueryName(qname string) (*QueryFields, error) {
	qname = strings.ToLower(strings.TrimSuffix(qname, "."))
	matches := queryRegexp.FindStringSubmatch(qname)
	if matches == nil {
		return nil, fmt.Errorf("invalid query format: %s", qname)
	}
	return &QueryFields{
		Nonce:     matches[1],
		Seq:       matches[2],
		SessionID: matches[3],
		DataLabel: matches[4],
	}, nil
}

// BuildQueryName builds a DNS query name from protocol fields
func BuildQueryName(nonce, seq, sessionID string, dataLabel string) string {
	return fmt.Sprintf("%s-%s-%s.%s.aura.net.", nonce, seq, sessionID, dataLabel)
}

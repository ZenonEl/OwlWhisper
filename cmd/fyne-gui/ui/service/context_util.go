package services

import (
	"fmt"
	"sort"
	"strings"
)

// CreateContextIDForPeers создает детерминированный ID чата 1-на-1.
func CreateContextIDForPeers(myPeerID, peerID string) string {
	ids := []string{myPeerID, peerID}
	sort.Strings(ids)
	return fmt.Sprintf("chat-1-on-1-%s", strings.Join(ids, "-"))
}

package internal

import (
"encoding/hex"
"fmt"
"log"
"net"
"strings"
"sync"
"time"

"github.com/miekg/dns"
)

// Packet Structure: [Nonce(4hex)]-[Seq(4hex)]-[SessionID(4hex)].[Base32Data].aura.net.
// Example: a1b2-0001-cafe.mfzwizj.aura.net.
// - Nonce: 4-char hex for cache busting (prevents DNS caching)
// - Seq: 4-char hex sequence number (0000-FFFF)
// - SessionID: 4-char hex session identifier
// - Base32Data: Base32-encoded payload (no padding, lowercase)

type QueryFields struct {
Nonce     string
Seq       string
SessionID string
DataLabel string
}

func ParseQueryName(name string) (*QueryFields, error) {
parts := strings.Split(name, ".")
if len(parts) < 3 {
return nil, fmt.Errorf("invalid qname format")
}
headerParts := strings.Split(parts[0], "-")
if len(headerParts) < 3 {
return nil, fmt.Errorf("invalid header format")
}
return &QueryFields{
Nonce:     headerParts[0],
Seq:       headerParts[1],
SessionID: headerParts[2],
DataLabel: parts[1],
}, nil
}

const (
ZoneName       = "aura.net."
WhatsAppHost   = "e1.whatsapp.net"
WhatsAppPort   = 5222 // ONLY port 5222 - filters media/CDN traffic automatically
SessionTimeout = 60 * time.Second
MaxIPv6Payload = 16
)

type session struct {
id        string
tcpConn   net.Conn
tcpMutex  sync.Mutex
lastSeen  time.Time
inBuffer  []byte
outBuffer map[uint16][]byte
seqSeen   map[uint16]bool
}

type Server struct {
sessions map[string]*session
mu       sync.RWMutex
zone     string
}

func NewServer(zone string) *Server {
return &Server{
sessions: make(map[string]*session),
zone:     zone,
}
}

func (s *Server) ListenAndServe(addr string) error {
dns.HandleFunc(s.zone, s.handleDNS)
srv := &dns.Server{Addr: addr, Net: "udp"}
log.Printf("Aura-Server listening on %s for zone %s (WhatsApp port %d only)", addr, s.zone, WhatsAppPort)
return srv.ListenAndServe()
}

func (s *Server) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
if len(r.Question) == 0 {
return
}
q := r.Question[0]
if q.Qtype != dns.TypeAAAA || !strings.HasSuffix(q.Name, s.zone) {
return
}
fields, err := ParseQueryName(q.Name)
if err != nil {
return
}
seq, _ := hex.DecodeString(fields.Seq)
if len(seq) != 2 {
return
}
seqNum := uint16(seq[0])<<8 | uint16(seq[1])

sess := s.getSession(fields.SessionID)
sess.lastSeen = time.Now()

// Handle upstream data (client -> WhatsApp)
if fields.DataLabel != "" {
data, err := DecodeLabelToData(fields.DataLabel)
if err == nil && !sess.seqSeen[seqNum] {
sess.outBuffer[seqNum] = data
sess.seqSeen[seqNum] = true
go s.forwardToWhatsApp(sess, seqNum)
}
}

// Prepare AAAA response with downstream data (WhatsApp -> client)
resp := new(dns.Msg)
resp.SetReply(r)
var payload []byte
sess.tcpMutex.Lock()
if len(sess.inBuffer) > 0 {
if len(sess.inBuffer) > MaxIPv6Payload {
payload = sess.inBuffer[:MaxIPv6Payload]
sess.inBuffer = sess.inBuffer[MaxIPv6Payload:]
} else {
payload = sess.inBuffer
sess.inBuffer = nil
}
}
sess.tcpMutex.Unlock()
if len(payload) > 0 {
ip, _ := PackDataToIPv6(payload)
resp.Answer = append(resp.Answer, &dns.AAAA{
Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
AAAA: ip,
})
}
w.WriteMsg(resp)
}

func (s *Server) getSession(sessionID string) *session {
s.mu.Lock()
sess, ok := s.sessions[sessionID]
if !ok {
// TEXT-ONLY ENFORCEMENT: Only connect to WhatsApp port 5222
// This automatically filters out media/CDN traffic
whatsappAddr := fmt.Sprintf("%s:%d", WhatsAppHost, WhatsAppPort)
tcpConn, err := net.DialTimeout("tcp", whatsappAddr, 5*time.Second)
if err != nil {
log.Printf("Failed to connect to %s: %v", whatsappAddr, err)
s.mu.Unlock()
return &session{id: sessionID, outBuffer: make(map[uint16][]byte), seqSeen: make(map[uint16]bool)}
}
sess = &session{
id:        sessionID,
tcpConn:   tcpConn,
lastSeen:  time.Now(),
outBuffer: make(map[uint16][]byte),
seqSeen:   make(map[uint16]bool),
}
s.sessions[sessionID] = sess
go s.readFromWhatsApp(sess)
go s.sessionTimeoutWatcher(sessionID)
}
s.mu.Unlock()
return sess
}

func (s *Server) forwardToWhatsApp(sess *session, seqNum uint16) {
sess.tcpMutex.Lock()
data, ok := sess.outBuffer[seqNum]
if ok && sess.tcpConn != nil {
sess.tcpConn.Write(data)
delete(sess.outBuffer, seqNum)
}
sess.tcpMutex.Unlock()
}

func (s *Server) readFromWhatsApp(sess *session) {
buf := make([]byte, 1024)
for {
n, err := sess.tcpConn.Read(buf)
if err != nil {
log.Printf("WhatsApp read error: %v", err)
return
}
sess.tcpMutex.Lock()
sess.inBuffer = append(sess.inBuffer, buf[:n]...)
sess.tcpMutex.Unlock()
}
}

func (s *Server) sessionTimeoutWatcher(sessionID string) {
ticker := time.NewTicker(10 * time.Second)
defer ticker.Stop()
for range ticker.C {
s.mu.RLock()
sess, ok := s.sessions[sessionID]
s.mu.RUnlock()
if !ok {
return
}
if time.Since(sess.lastSeen) > SessionTimeout {
s.mu.Lock()
if sess.tcpConn != nil {
sess.tcpConn.Close()
}
delete(s.sessions, sessionID)
s.mu.Unlock()
log.Printf("Session %s timed out and cleaned up", sessionID)
return
}
}
}

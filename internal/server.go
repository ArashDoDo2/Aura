package internal

import (
"fmt"
"log"
"net"
"strings"
"sync"
"time"

"github.com/miekg/dns"
)

// Packet Structure: [Nonce(4hex)]-[Seq(4hex)]-[SessionID(4hex)].[Base32Data].<domain>
// Example: a1b2-0001-cafe.mfzwizj.example.com.
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
WhatsAppHost   = "e1.whatsapp.net"
WhatsAppPort   = 5222 // ONLY port 5222 - filters media/CDN traffic automatically
SessionTimeout = 60 * time.Second
MaxIPv6Payload = 16
)

type session struct {
conn     net.Conn
buffer   []byte
lastSeen time.Time
mu       sync.Mutex
}

type Server struct {
mu       sync.Mutex
sessions map[string]*session
Domain   string // Configurable domain (e.g., "example.com.")
}

func NewServer(domain string) *Server {
s := &Server{
sessions: make(map[string]*session),
Domain:   domain,
}
go s.sessionTimeoutWatcher()
return s
}

func (s *Server) ListenAndServe(addr string) error {
dns.HandleFunc(s.Domain, s.handleDNS)
srv := &dns.Server{Addr: addr, Net: "udp"}
log.Printf("Aura Server listening on %s for domain %s (WhatsApp port %d only)", addr, s.Domain, WhatsAppPort)
return srv.ListenAndServe()
}

func (s *Server) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
m := new(dns.Msg)
m.SetReply(r)
m.Authoritative = true

if len(r.Question) == 0 {
w.WriteMsg(m)
return
}

q := r.Question[0]
if q.Qtype != dns.TypeAAAA {
m.SetRcode(r, dns.RcodeNotImplemented)
w.WriteMsg(m)
return
}

// Verify query is for our domain
if !strings.HasSuffix(q.Name, s.Domain) {
m.SetRcode(r, dns.RcodeNameError)
w.WriteMsg(m)
return
}

qf, err := ParseQueryName(q.Name)
if err != nil {
m.SetRcode(r, dns.RcodeFormatError)
w.WriteMsg(m)
return
}

sess, err := s.getSession(qf.SessionID, q.Name)
if err != nil {
log.Printf("Session error: %v", err)
m.SetRcode(r, dns.RcodeServerFailure)
w.WriteMsg(m)
return
}

sess.mu.Lock()
sess.lastSeen = time.Now()

// Handle upstream data if present
if len(qf.DataLabel) > 0 {
data, err := DecodeLabelToData(qf.DataLabel)
if err == nil && len(data) > 0 {
sess.conn.Write(data)
}
}

// Read downstream data (non-blocking)
buf := make([]byte, MaxIPv6Payload*10)
sess.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
n, _ := sess.conn.Read(buf)
if n > 0 {
sess.buffer = append(sess.buffer, buf[:n]...)
}
sess.conn.SetReadDeadline(time.Time{})

// Pack data into IPv6 responses
for len(sess.buffer) >= MaxIPv6Payload {
chunk := sess.buffer[:MaxIPv6Payload]
sess.buffer = sess.buffer[MaxIPv6Payload:]
ip, _ := PackDataToIPv6(chunk)
rr := &dns.AAAA{
Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
AAAA: ip,
}
m.Answer = append(m.Answer, rr)
}
if len(sess.buffer) > 0 && len(m.Answer) == 0 {
chunk := sess.buffer
sess.buffer = nil
for len(chunk) < MaxIPv6Payload {
chunk = append(chunk, 0)
}
ip, _ := PackDataToIPv6(chunk[:MaxIPv6Payload])
rr := &dns.AAAA{
Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
AAAA: ip,
}
m.Answer = append(m.Answer, rr)
}

sess.mu.Unlock()
w.WriteMsg(m)
}

func (s *Server) getSession(sid string, qname string) (*session, error) {
s.mu.Lock()
defer s.mu.Unlock()

if sess, ok := s.sessions[sid]; ok {
return sess, nil
}

// TEXT-ONLY ENFORCEMENT: Block all non-5222 connections
// This filters out media, voice, CDN traffic
if strings.Contains(qname, "media") || strings.Contains(qname, "cdn") {
return nil, fmt.Errorf("media/CDN traffic blocked")
}

// ONLY connect to WhatsApp port 5222 (text messages)
conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", WhatsAppHost, WhatsAppPort), 5*time.Second)
if err != nil {
return nil, err
}

sess := &session{
conn:     conn,
lastSeen: time.Now(),
}
s.sessions[sid] = sess
log.Printf("New session: %s", sid)
return sess, nil
}

func (s *Server) sessionTimeoutWatcher() {
for {
time.Sleep(10 * time.Second)
s.mu.Lock()
for sid, sess := range s.sessions {
if time.Since(sess.lastSeen) > SessionTimeout {
sess.conn.Close()
delete(s.sessions, sid)
log.Printf("Session timeout: %s", sid)
}
}
s.mu.Unlock()
}
}

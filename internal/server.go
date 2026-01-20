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

func ParseQueryName(name, domain string) (*QueryFields, error) {
	if !strings.HasSuffix(name, domain) {
		return nil, fmt.Errorf("qname %s does not end with domain %s", name, domain)
	}

	trimmed := strings.TrimSuffix(name, domain)
	trimmed = strings.TrimSuffix(trimmed, ".")

	headerAndLabel := strings.SplitN(trimmed, ".", 2)
	header := headerAndLabel[0]
	dataLabel := ""
	if len(headerAndLabel) == 2 {
		dataLabel = headerAndLabel[1]
	}

	headerParts := strings.Split(header, "-")
	if len(headerParts) < 3 {
		return nil, fmt.Errorf("invalid header format")
	}

	return &QueryFields{
		Nonce:     headerParts[0],
		Seq:       headerParts[1],
		SessionID: headerParts[2],
		DataLabel: dataLabel,
	}, nil
}

const (
	WhatsAppHost   = "e1.whatsapp.net"
	WhatsAppPort   = 5222 // Default text-only port to filter media/CDN traffic
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
	mu         sync.Mutex
	sessions   map[string]*session
	Domain     string // Configurable domain (e.g., "example.com.")
	TargetHost string
	TargetPort int
}

func NewServer(domain, targetHost string, targetPort int) *Server {
	s := &Server{
		sessions:   make(map[string]*session),
		Domain:     domain,
		TargetHost: targetHost,
		TargetPort: targetPort,
	}
	go s.sessionTimeoutWatcher()
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	pattern := s.Domain
	if !strings.HasSuffix(pattern, ".") {
		pattern = pattern + "."
	}
	dns.HandleFunc(pattern, s.handleDNS)
	srv := &dns.Server{Addr: addr, Net: "udp"}
	log.Printf("Aura Server listening on %s for domain %s (target %s:%d)", addr, s.Domain, s.TargetHost, s.TargetPort)
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

	qf, err := ParseQueryName(q.Name, s.Domain)
	if err != nil {
		m.SetRcode(r, dns.RcodeFormatError)
		w.WriteMsg(m)
		return
	}

	log.Printf("DNS query %s  seq=%s session=%s labelLen=%d",
		q.Name, qf.Seq, qf.SessionID, len(qf.DataLabel))

	sess, err := s.getSession(qf.SessionID, q.Name)
	if err != nil {
		log.Printf("Session error: %v", err)
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		return
	}

	sess.mu.Lock()
	sess.lastSeen = time.Now()

	if len(qf.DataLabel) > 0 {
		traceLabel := qf.DataLabel
		if len(traceLabel) > 60 {
			traceLabel = traceLabel[:60] + "..."
		}
		log.Printf("Raw DNS label session=%s seq=%s label=%s", qf.SessionID, qf.Seq, traceLabel)
		normalized := strings.ToUpper(qf.DataLabel)
		data, err := DecodeLabelToData(normalized)
		if err != nil {
			log.Printf("Decode error session=%s seq=%s: %v", qf.SessionID, qf.Seq, err)
		} else if len(data) > 0 {
			n, err := sess.conn.Write(data)
			if err != nil {
				log.Printf("WhatsApp write error session=%s: %v", qf.SessionID, err)
			} else {
				log.Printf("Forwarded %d bytes to WhatsApp session=%s seq=%s", n, qf.SessionID, qf.Seq)
			}
		}
	}

	// Read downstream data (non-blocking)
	buf := make([]byte, MaxIPv6Payload*10)
	sess.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	n, _ := sess.conn.Read(buf)
	if n > 0 {
		sess.buffer = append(sess.buffer, buf[:n]...)
		log.Printf("Received %d bytes from Target for session=%s", n, qf.SessionID)
		log.Printf("Buffered %d bytes from WhatsApp session=%s", n, qf.SessionID)
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

	written := len(m.Answer)
	if written > 0 {
		log.Printf("Responding with %d AAAA records session=%s", written, qf.SessionID)
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

	// Connect to configured target (defaults to WhatsApp text port)
	host := s.TargetHost
	port := s.TargetPort
	if host == "" {
		host = WhatsAppHost
	}
	if port == 0 {
		port = WhatsAppPort
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
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

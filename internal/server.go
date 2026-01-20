// --- Encoding/Decoding helpers ---

import "encoding/base32"

type QueryFields struct {
	Nonce     string
	Seq       string
	SessionID string
	DataLabel string
}

func ParseQueryName(name string) (*QueryFields, error) {
	// نام دامنه ورودی به صورت: nonce-seq-sessionid.data.aura.net. است
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

var b32Encoder = base32.StdEncoding.WithPadding(base32.NoPadding)

func DecodeLabelToData(label string) ([]byte, error) {
	if label == "" {
		return nil, nil
	}
	return b32Encoder.DecodeString(strings.ToUpper(label))
}

func PackDataToIPv6(data []byte) (net.IP, error) {
	ip := make([]byte, 16)
	copy(ip, data)
	return net.IP(ip), nil
}
package internal

import (
	"encoding/hex"
	"encoding/base32"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

const (
	ZoneName       = "aura.net."
	WhatsAppHost   = "e1.whatsapp.net:5222" // قابل تغییر برای سرورهای مختلف
	SessionTimeout = 60 * time.Second
	MaxIPv6Payload = 16
)

type session struct {
	id        string
	tcpConn   net.Conn
	tcpMutex  sync.Mutex
	lastSeen  time.Time
	inBuffer  []byte // داده‌های دریافتی از واتس‌اپ
	outBuffer map[uint16][]byte // داده‌های ارسالی از کلاینت، بر اساس Seq
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
	log.Printf("Aura-Server listening on %s for zone %s", addr, s.zone)
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

	// دریافت داده از کلاینت و ارسال به واتس‌اپ
	if fields.DataLabel != "" {
		data, err := DecodeLabelToData(fields.DataLabel)
		if err == nil && !sess.seqSeen[seqNum] {
			sess.outBuffer[seqNum] = data
			sess.seqSeen[seqNum] = true
			go s.forwardToWhatsApp(sess, seqNum)
		}
	}

	// آماده‌سازی پاسخ AAAA با حداکثر ۱۶ بایت از واتس‌اپ
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
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
			AAAA: ip,
		})
	}
	w.WriteMsg(resp)
		type QueryFields struct {
}

func (s *Server) getSession(sessionID string) *session {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		tcpConn, err := net.DialTimeout("tcp", WhatsAppHost, 5*time.Second)
		if err != nil {
			log.Printf("Failed to connect to WhatsApp: %v", err)
			return &session{id: sessionID, outBuffer: make(map[uint16][]byte), seqSeen: make(map[uint16]bool)}
		}
		sess = &session{
			id:       sessionID,
			tcpConn:  tcpConn,
			lastSeen: time.Now(),
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
















































































































































































}	}		}			return			log.Printf("Session %s timed out and cleaned up", sessionID)			s.mu.Unlock()			delete(s.sessions, sessionID)			}				sess.tcpConn.Close()			if sess.tcpConn != nil {			s.mu.Lock()		if time.Since(sess.lastSeen) > SessionTimeout {		}			return		if !ok {		s.mu.RUnlock()		sess, ok := s.sessions[sessionID]		s.mu.RLock()	for range ticker.C {	defer ticker.Stop()	ticker := time.NewTicker(10 * time.Second)func (s *Server) sessionTimeoutWatcher(sessionID string) {}	}		sess.tcpMutex.Unlock()		sess.inBuffer = append(sess.inBuffer, buf[:n]...)		sess.tcpMutex.Lock()		}			return			log.Printf("WhatsApp read error: %v", err)		if err != nil {		n, err := sess.tcpConn.Read(buf)	for {	buf := make([]byte, 1024)func (s *Server) readFromWhatsApp(sess *session) {}	sess.tcpMutex.Unlock()	}		delete(sess.outBuffer, seqNum)		sess.tcpConn.Write(data)	if ok && sess.tcpConn != nil {	data, ok := sess.outBuffer[seqNum]	sess.tcpMutex.Lock()func (s *Server) forwardToWhatsApp(sess *session, seqNum uint16) {}	return sess	s.mu.Unlock()	}		go s.sessionTimeoutWatcher(sessionID)		go s.readFromWhatsApp(sess)		s.sessions[sessionID] = sess		}			seqSeen:   make(map[uint16]bool),			outBuffer: make(map[uint16][]byte),			lastSeen: time.Now(),			tcpConn:  tcpConn,			id:       sessionID,		sess = &session{		}			return &session{id: sessionID, outBuffer: make(map[uint16][]byte), seqSeen: make(map[uint16]bool)}			log.Printf("Failed to connect to WhatsApp: %v", err)		if err != nil {		tcpConn, err := net.DialTimeout("tcp", WhatsAppHost, 5*time.Second)	if !ok {	sess, ok := s.sessions[sessionID]	s.mu.Lock()func (s *Server) getSession(sessionID string) *session {}	w.WriteMsg(resp)	}		})			AAAA: ip,			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},		resp.Answer = append(resp.Answer, &dns.AAAA{		ip, _ := PackDataToIPv6(payload)	if len(payload) > 0 {	sess.tcpMutex.Unlock()	}		}			sess.inBuffer = nil			payload = sess.inBuffer		} else {			sess.inBuffer = sess.inBuffer[MaxIPv6Payload:]			payload = sess.inBuffer[:MaxIPv6Payload]		if len(sess.inBuffer) > MaxIPv6Payload {	if len(sess.inBuffer) > 0 {	sess.tcpMutex.Lock()	var payload []byte	resp.SetReply(r)	resp := new(dns.Msg)	// Prepare AAAA response with up to 16 bytes from WhatsApp	}		}			go s.forwardToWhatsApp(sess, seqNum)			sess.seqSeen[seqNum] = true			sess.outBuffer[seqNum] = data		if err == nil && !sess.seqSeen[seqNum] {		data, err := DecodeLabelToData(fields.DataLabel)	if fields.DataLabel != "" {	// Handle incoming data from client	sess.lastSeen = time.Now()	sess := s.getSession(fields.SessionID)	// Get or create session	seqNum := uint16(seq[0])<<8 | uint16(seq[1])	}		return	if len(seq) != 2 {	seq, _ := hex.DecodeString(fields.Seq)	}		return	if err != nil {	fields, err := ParseQueryName(q.Name)	}		return	if q.Qtype != dns.TypeAAAA || !strings.HasSuffix(q.Name, s.zone) {	q := r.Question[0]	}		return	if len(r.Question) == 0 {func (s *Server) handleDNS(w dns.ResponseWriter, r *dns.Msg) {}	return srv.ListenAndServe()	log.Printf("Aura-Server listening on %s for zone %s", addr, s.zone)	srv := &dns.Server{Addr: addr, Net: "udp"}	dns.HandleFunc(s.zone, s.handleDNS)func (s *Server) ListenAndServe(addr string) error {}	}		zone:     zone,		sessions: make(map[string]*session),	return &Server{func NewServer(zone string) *Server {}	zone     string	mu       sync.RWMutex	sessions map[string]*sessiontype Server struct {}	seqSeen   map[uint16]bool	outBuffer map[uint16][]byte // Data from client to WhatsApp, keyed by Seq	inBuffer  []byte // Data from WhatsApp to client	lastSeen  time.Time	tcpMutex  sync.Mutex	tcpConn   net.Conn	id        stringtype session struct {)	MaxIPv6Payload   = 16	SessionTimeout   = 60 * time.Second	WhatsAppHost     = "e1.whatsapp.net:5222" // can be made dynamic	ZoneName         = "aura.net."const ()	"github.com/miekg/dns"	"time"	"sync"	"strings"	"net"	"log"	"fmt"	"encoding/hex"	"context"import (package internal
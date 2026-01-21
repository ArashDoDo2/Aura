package internal

import (
	"bufio"
	"context"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

const (
	DefaultSocks5Port = 1080
	MaxChunkSize      = 30                     // Fragment TCP data into 30-byte chunks
	MaxDataLabelLen   = 63                     // DNS label max length
	PollInterval      = 100 * time.Millisecond // Sequential polling interval (faster downstream polling)
)

// b32Encoder: Base32 with no padding, lowercase for DNS compatibility
var b32Encoder = base32.StdEncoding.WithPadding(base32.NoPadding)

type AuraClient struct {
	DNSServer  string // Public DNS like "1.1.1.1:53" (empty = use system resolver)
	Domain     string // Target domain (e.g., "example.com.")
	Socks5Port int    // SOCKS5 proxy listen port
	SessionID  string
	Mutex      sync.Mutex
	Seq        uint16
}

type bufferedConn struct {
	net.Conn
	reader         *bufio.Reader
	tlsFastPath    bool
	firstTLSRecord []byte
	tlsRecordSent  bool
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.reader.Read(p)
}

func NewAuraClient(dnsServer, domain string, port int) *AuraClient {
	rand.Seed(time.Now().UnixNano())
	if port == 0 {
		port = DefaultSocks5Port
	}
	// If DNS server is empty, system resolver will be used
	return &AuraClient{
		DNSServer:  dnsServer,
		Domain:     domain,
		Socks5Port: port,
		SessionID:  randomHex(4), // 4-char hex session ID
	}
}

// getEffectiveDNSServer returns the DNS server to use
// If c.DNSServer is empty, uses system default resolver
func (c *AuraClient) getEffectiveDNSServer() (string, error) {
	if c.DNSServer != "" {
		return c.DNSServer, nil
	}

	// Use system DNS resolver
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		// Fallback to common public DNS
		log.Printf("Could not read system DNS config, using 8.8.8.8:53")
		return "8.8.8.8:53", nil
	}

	if len(config.Servers) > 0 {
		// Use first system DNS server
		server := config.Servers[0]
		if !strings.Contains(server, ":") {
			server = server + ":53"
		}
		return server, nil
	}

	// Fallback
	return "8.8.8.8:53", nil
}

func randomHex(n int) string {
	b := make([]byte, n/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// StartSocks5 starts the SOCKS5 proxy and returns when context is cancelled
func (c *AuraClient) StartSocks5(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", c.Socks5Port))
	if err != nil {
		return err
	}
	defer ln.Close()

	effectiveDNS := c.DNSServer
	if effectiveDNS == "" {
		effectiveDNS = "<system resolver>"
	}
	log.Printf("Aura Client SOCKS5 listening on 127.0.0.1:%d", c.Socks5Port)
	log.Printf("DNS: %s, Domain: %s, Session: %s", effectiveDNS, c.Domain, c.SessionID)

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		}
		go c.handleSocks5Conn(ctx, conn)
	}
}

func (c *AuraClient) handleSocks5Conn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Strict Peek & Branch: Check first byte to determine protocol
	first, err := reader.Peek(1)
	if err != nil {
		log.Printf("Failed to peek first byte: %v", err)
		return
	}

	bc := &bufferedConn{Conn: conn, reader: reader}

	// Immediate Diversion for Non-SOCKS5 Traffic
	if first[0] != 0x05 {
		// Log transparent mode activation
		log.Printf("Non-SOCKS5 traffic detected (0x%02x), switching to Transparent Mode", first[0])

		// If TLS ClientHello detected, buffer the complete record
		if first[0] == 0x16 {
			if err := c.bufferTLSRecord(reader, bc); err != nil {
				log.Printf("Failed to buffer TLS record: %v", err)
				return
			}
			log.Printf("TLS handshake buffered, ready for fast-path transmission")
		}

		// Handle tunnel with default target (WhatsApp proxy mode)
		c.handleTunnel(ctx, bc)
		// CRITICAL: Return immediately to prevent fall-through to SOCKS5 logic
		return
	}

	// SOCKS5 Handshake (only executed for SOCKS5 protocol 0x05)
	buf := make([]byte, 262)
	if _, err := io.ReadFull(reader, buf[:2]); err != nil {
		return
	}

	nMethods := int(buf[1])
	if _, err := io.ReadFull(reader, buf[:nMethods]); err != nil {
		return
	}
	conn.Write([]byte{0x05, 0x00}) // No Auth

	if _, err := io.ReadFull(reader, buf[:4]); err != nil {
		return
	}
	if buf[1] != 0x01 {
		return
	} // Only CONNECT supported

	// Read destination address (read but ignore)
	switch buf[3] {
	case 0x01: // IPv4
		if _, err := io.ReadFull(reader, buf[:4+2]); err != nil {
			return
		}
	case 0x03: // Domain name
		if _, err := io.ReadFull(reader, buf[:1]); err != nil {
			return
		}
		dlen := int(buf[0])
		if _, err := io.ReadFull(reader, buf[:dlen+2]); err != nil {
			return
		}
	}

	// Send success response
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	c.handleTunnel(ctx, bc)
}

func (c *AuraClient) bufferTLSRecord(reader *bufio.Reader, bc *bufferedConn) error {
	header := make([]byte, 5)
	if _, err := io.ReadFull(reader, header); err != nil {
		return err
	}

	length := int(binary.BigEndian.Uint16(header[3:5]))
	record := make([]byte, 5+length)
	copy(record, header)
	if length > 0 {
		if _, err := io.ReadFull(reader, record[5:]); err != nil {
			return err
		}
	}

	bc.firstTLSRecord = append([]byte(nil), record...)
	bc.tlsFastPath = true
	bc.tlsRecordSent = false
	log.Printf("Buffered TLS record (%d bytes) before tunneling", len(record))
	return nil
}

func (c *AuraClient) handleTunnel(ctx context.Context, conn net.Conn) {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(2)
	// Upstream: TCP -> DNS
	go func() {
		defer wg.Done()
		c.tcpToDNS(ctx, conn)
		cancel()
	}()
	// Downstream: DNS -> TCP (Sequential polling)
	go func() {
		defer wg.Done()
		c.dnsToTCP(ctx, conn)
		cancel()
	}()
	wg.Wait()
}

// tcpToDNS: Fragment TCP data into 30-byte chunks, Base32 encode, send as DNS AAAA queries
func (c *AuraClient) tcpToDNS(ctx context.Context, conn net.Conn) {
	if bc, ok := conn.(*bufferedConn); ok && bc.tlsFastPath && !bc.tlsRecordSent && len(bc.firstTLSRecord) > 0 {
		c.flushTLSRecord(bc.firstTLSRecord)
		bc.tlsRecordSent = true
		bc.firstTLSRecord = nil
	}

	buf := make([]byte, MaxChunkSize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		if n == 0 {
			continue
		}
		log.Printf("TCP read %d bytes from SOCKS5", n)
		c.sendDNSPacket(buf[:n])
	}
}

func (c *AuraClient) sendDNSPacket(data []byte) {
	c.Mutex.Lock()
	seq := c.Seq
	c.Seq++
	c.Mutex.Unlock()

	// Cache Busting: Random nonce prevents DNS caching
	nonce := randomHex(4)
	label := strings.ToLower(b32Encoder.EncodeToString(data))
	snippet := label
	if len(snippet) > 10 {
		snippet = snippet[:10]
	}
	log.Printf("Encoded chunk seq=0x%04x first10=%s len=%d", seq, snippet, len(label))

	// Packet Structure: [Nonce]-[Seq]-[SessionID].[Base32Data].<domain>
	qname := fmt.Sprintf("%s-%04x-%s.%s.%s", nonce, seq, c.SessionID, label, c.Domain)

	m := new(dns.Msg)
	m.SetQuestion(qname, dns.TypeAAAA)
	log.Printf("DNS uplink chunk seq=0x%04x len=%d domain=%s", seq, len(data), qname)
	c.sendQuery(m)
}

func (c *AuraClient) sendDNSPacketFast(data []byte) {
	c.Mutex.Lock()
	seq := c.Seq
	c.Seq++
	c.Mutex.Unlock()

	nonce := randomHex(4)
	label := strings.ToLower(b32Encoder.EncodeToString(data))
	snippet := label
	if len(snippet) > 10 {
		snippet = snippet[:10]
	}
	log.Printf("Fast-path chunk seq=0x%04x first10=%s len=%d", seq, snippet, len(label))

	qname := fmt.Sprintf("%s-%04x-%s.%s.%s", nonce, seq, c.SessionID, label, c.Domain)

	m := new(dns.Msg)
	m.SetQuestion(qname, dns.TypeAAAA)
	log.Printf("Fast-path DNS uplink seq=0x%04x len=%d domain=%s", seq, len(data), qname)
	go c.sendQuery(m)
}

func (c *AuraClient) flushTLSRecord(record []byte) {
	log.Printf("Flushing TLS record fast path (%d bytes)", len(record))
	for offset := 0; offset < len(record); {
		end := offset + MaxChunkSize
		if end > len(record) {
			end = len(record)
		}
		c.sendDNSPacketFast(record[offset:end])
		offset = end
	}
}

func (c *AuraClient) sendQuery(m *dns.Msg) {
	dnsServer, err := c.getEffectiveDNSServer()
	if err != nil {
		log.Printf("DNS server error: %v", err)
		return
	}

	dnsClient := new(dns.Client)
	dnsClient.Net = "udp"
	dnsClient.Timeout = 2 * time.Second
	_, _, err = dnsClient.Exchange(m, dnsServer)
	if err != nil {
		log.Printf("DNS query error: %v", err)
	}
}

// dnsToTCP: Sequential polling - Extract 16 bytes from each IPv6 address in DNS response
func (c *AuraClient) dnsToTCP(ctx context.Context, conn net.Conn) {
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp := c.pollDNS()
			if len(resp) > 0 {
				conn.Write(resp)
			}
		}
	}
}

func (c *AuraClient) pollDNS() []byte {
	// Use special sequence ffff for polling
	nonce := randomHex(4)
	qname := fmt.Sprintf("%s-ffff-%s.%s", nonce, c.SessionID, c.Domain)

	m := new(dns.Msg)
	m.SetQuestion(qname, dns.TypeAAAA)

	dnsServer, err := c.getEffectiveDNSServer()
	if err != nil {
		return nil
	}

	log.Printf("Polling DNS for session=%s qname=%s server=%s", c.SessionID, qname, dnsServer)

	dnsClient := new(dns.Client)
	dnsClient.Timeout = 2 * time.Second
	resp, _, err := dnsClient.Exchange(m, dnsServer)

	if err != nil {
		log.Printf("Poll query error session=%s err=%v", c.SessionID, err)
		return nil
	}

	if resp == nil {
		return nil
	}

	// Extract data from IPv6 addresses (16 bytes each)
	var out []byte
	for _, ans := range resp.Answer {
		if aaaa, ok := ans.(*dns.AAAA); ok {
			out = append(out, aaaa.AAAA[:]...)
		}
	}
	if len(out) > 0 {
		log.Printf("DNS response poll len=%d seq=ffff domain=%s", len(out), qname)
	}
	return out
}

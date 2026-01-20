package internal

import (
	"context"
	"encoding/base32"
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
	Socks5Port      = 1080
	WhatsAppPort    = 5222
	MaxDataLabelLen = 63
	DomainSuffix    = "aura.net."
)

// encoding استاندارد برای ساب‌دومین (بدون حروف بزرگ و بدون Padding)
var b32Encoder = base32.StdEncoding.WithPadding(base32.NoPadding)

type AuraClient struct {
	DNSServer string // مثال: "8.8.8.8:53"
	SessionID string
	Mutex     sync.Mutex
	Seq       uint16
}

func NewAuraClient(dnsServer string) *AuraClient {
	rand.Seed(time.Now().UnixNano())
	return &AuraClient{
		DNSServer: dnsServer,
		SessionID: randomHex(4),
	}
}

func randomHex(n int) string {
	b := make([]byte, n/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// EncodeDataToLabel تبدیل دیتای باینری به Base32 برای استفاده در DNS
func EncodeDataToLabel(data []byte) string {
	return strings.ToLower(b32Encoder.EncodeToString(data))
}

func (c *AuraClient) StartSocks5(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", Socks5Port))
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Printf("Aura SOCKS5 listening on :%d", Socks5Port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}
		go c.handleSocks5Conn(ctx, conn)
	}
}

func (c *AuraClient) handleSocks5Conn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	// مرحله Handshake ساده SOCKS5
	buf := make([]byte, 262)
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}

	nMethods := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:nMethods]); err != nil {
		return
	}
	conn.Write([]byte{0x05, 0x00}) // No Auth

	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return
	}
	if buf[1] != 0x01 {
		return
	} // فقط دستور CONNECT پشتیبانی می‌شود

	var host string
	switch buf[3] {
	case 0x01: // IPv4
		if _, err := io.ReadFull(conn, buf[:4+2]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 0x03: // Domain name
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return
		}
		dlen := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:dlen+2]); err != nil {
			return
		}
		host = string(buf[:dlen])
	}

	// ما فقط ترافیک واتس‌اپ را اجازه می‌دهیم
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	c.handleWhatsApp(ctx, conn)
}

func (c *AuraClient) handleWhatsApp(ctx context.Context, conn net.Conn) {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(2)
	// ارسال از موبایل به استرالیا
	go func() {
		defer wg.Done()
		c.tcpToDNS(ctx, conn)
		cancel()
	}()
	// دریافت از استرالیا به موبایل
	go func() {
		defer wg.Done()
		c.dnsToTCP(ctx, conn)
		cancel()
	}()
	wg.Wait()
}

func (c *AuraClient) tcpToDNS(ctx context.Context, conn net.Conn) {
	buf := make([]byte, 30) // قطعات کوچک برای جا شدن در DNS Label
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		c.sendDNSPacket(buf[:n])
	}
}

func (c *AuraClient) sendDNSPacket(data []byte) {
	c.Mutex.Lock()
	seq := c.Seq
	c.Seq++
	c.Mutex.Unlock()

	label := EncodeDataToLabel(data)
	nonce := randomHex(4)

	// ساختار: [Nonce]-[Sequence]-[SessionID].[Data].aura.net.
	qname := fmt.Sprintf("%s-%04x-%s.%s.%s", nonce, seq, c.SessionID, label, DomainSuffix)

	m := new(dns.Msg)
	m.SetQuestion(qname, dns.TypeAAAA)
	c.sendQuery(m)
}

func (c *AuraClient) sendQuery(m *dns.Msg) {
	dnsClient := new(dns.Client)
	dnsClient.Net = "udp"
	dnsClient.Timeout = 2 * time.Second
	_, _, err := dnsClient.Exchange(m, c.DNSServer)
	if err != nil {
		log.Printf("Upstream DNS Error: %v", err)
	}
}

func (c *AuraClient) dnsToTCP(ctx context.Context, conn net.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			resp := c.pollDNS()
			if len(resp) > 0 {
				conn.Write(resp)
			} else {
				time.Sleep(1 * time.Second) // فاصله بین Polling در زمان بیکاری
			}
		}
	}
}

func (c *AuraClient) pollDNS() []byte {
	nonce := randomHex(4)
	// ارسال کوئری خالی با Seq خاص (ffff) برای دریافت دیتا
	qname := fmt.Sprintf("%s-ffff-%s..%s", nonce, c.SessionID, DomainSuffix)

	m := new(dns.Msg)
	m.SetQuestion(qname, dns.TypeAAAA)

	dnsClient := new(dns.Client)
	dnsClient.Timeout = 2 * time.Second
	resp, _, err := dnsClient.Exchange(m, c.DNSServer)

	if err != nil || resp == nil {
		return nil
	}

	var out []byte
	for _, ans := range resp.Answer {
		if aaaa, ok := ans.(*dns.AAAA); ok {
			// استخراج دیتا از ۱۶ بایت آی‌پی IPv6
			out = append(out, aaaa.AAAA[:]...)
		}
	}
	return out
}

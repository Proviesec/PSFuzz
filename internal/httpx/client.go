// Package httpx provides an HTTP client for PSFuzz with proxy, TLS, throttling, and safe-mode (no loopback) support.
package httpx

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/Proviesec/PSFuzz/internal/config"
	"golang.org/x/net/http2"
)

type Client struct {
	cfg    *config.Config
	http   *http.Client
	rand   *rand.Rand
	tick   <-chan time.Time
	ticker *time.Ticker // so Close() can Stop() it; only set when ThrottleRPS > 0
	scope  map[string]struct{}
	scale  float64
}

type RequestSpec struct {
	URL     string
	Method  string
	Body    string
	Headers map[string]string
	Delay   time.Duration
}

// Doer performs HTTP requests. *Client implements Doer; use this interface in callers that need to mock or swap the client.
type Doer interface {
	Do(ctx context.Context, spec RequestSpec) (*http.Response, error)
}

// New builds an HTTP client from cfg. Returns an error if cfg is nil (callers can handle instead of panic).
func New(cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("config must not be nil")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Proxy != "" {
		proxyStr := cfg.Proxy
		if cfg.ProxyUser != "" && !strings.Contains(proxyStr, "@") {
			if u, err := url.Parse(proxyStr); err == nil && u.Host != "" {
				u.User = url.UserPassword(cfg.ProxyUser, cfg.ProxyPass)
				proxyStr = u.String()
			}
		}
		if proxyURL, err := url.Parse(proxyStr); err == nil {
			if proxyURL.Scheme == "socks5" || proxyURL.Scheme == "socks5h" {
				transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialSOCKS5(ctx, proxyURL, network, addr, cfg.ProxyUser, cfg.ProxyPass)
				}
			} else {
				transport.Proxy = http.ProxyURL(proxyURL)
			}
		}
	}
	if cfg.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if cfg.UseHTTP2 {
		_ = http2.ConfigureTransport(transport)
	}
	h := &http.Client{Timeout: cfg.Timeout, Transport: transport}
	if !cfg.FollowRedirects {
		h.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	c := &Client{cfg: cfg, http: h, rand: rand.New(rand.NewSource(time.Now().UnixNano())), scale: 1.0}
	if cfg.ThrottleRPS > 0 {
		c.ticker = time.NewTicker(time.Second / time.Duration(cfg.ThrottleRPS))
		c.tick = c.ticker.C
	}
	if len(cfg.AllowedHosts) > 0 {
		c.scope = map[string]struct{}{}
		for _, h := range cfg.AllowedHosts {
			c.scope[strings.ToLower(strings.TrimSpace(h))] = struct{}{}
		}
	}
	return c, nil
}

// Close releases resources (e.g. rate-limit ticker). Call when the client is no longer used.
func (c *Client) Close() {
	if c.ticker != nil {
		c.ticker.Stop()
		c.ticker = nil
		c.tick = nil
	}
}

func (c *Client) Do(ctx context.Context, spec RequestSpec) (*http.Response, error) {
	if c.cfg.DelayMax > 0 {
		delay := c.cfg.DelayMin
		if c.cfg.DelayMax > c.cfg.DelayMin {
			jitter := c.rand.Float64()
			delay = c.cfg.DelayMin + time.Duration(jitter*float64(c.cfg.DelayMax-c.cfg.DelayMin))
		}
		if c.scale > 1.0 {
			delay = time.Duration(float64(delay) * c.scale)
		}
		if spec.Delay > 0 {
			delay += spec.Delay
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	} else if spec.Delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(spec.Delay):
		}
	}
	if c.tick != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.tick:
		}
	}
	if err := c.validateTarget(ctx, spec.URL); err != nil {
		return nil, fmt.Errorf("validate target: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.RetryCount; attempt++ {
		resp, err := c.doOnce(ctx, spec)
		if err == nil && !isRetryStatus(resp.StatusCode, c.cfg.BypassTooManyRequests) {
			return resp, nil
		}
		if err == nil && isRetryStatus(resp.StatusCode, c.cfg.BypassTooManyRequests) {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("retryable status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		if attempt == c.cfg.RetryCount {
			break
		}
		backoff := c.backoff(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, fmt.Errorf("request after %d retries: %w", c.cfg.RetryCount+1, lastErr)
}

func (c *Client) SetScale(scale float64) {
	if scale < 1.0 {
		scale = 1.0
	}
	c.scale = scale
}

func (c *Client) Replay(ctx context.Context, spec RequestSpec, proxyURL string) {
	if proxyURL == "" {
		return
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if p, err := url.Parse(proxyURL); err == nil {
		transport.Proxy = http.ProxyURL(p)
	} else {
		return
	}
	if c.cfg.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Timeout: c.cfg.Timeout, Transport: transport}
	req, err := http.NewRequestWithContext(ctx, spec.Method, spec.URL, strings.NewReader(spec.Body))
	if err != nil {
		return
	}
	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

func dialSOCKS5(ctx context.Context, proxyURL *url.URL, network, addr string, user, pass string) (net.Conn, error) {
	proxyAddr := proxyURL.Host
	if !strings.Contains(proxyAddr, ":") {
		proxyAddr += ":1080"
	}
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, err
	}
	// greeting
	var methods []byte
	if user != "" {
		methods = []byte{0x02}
	} else {
		methods = []byte{0x00}
	}
	if _, err := conn.Write([]byte{0x05, byte(len(methods))}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := conn.Write(methods); err != nil {
		_ = conn.Close()
		return nil, err
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp[0] != 0x05 || resp[1] == 0xFF {
		_ = conn.Close()
		return nil, fmt.Errorf("socks5 auth not accepted")
	}
	if resp[1] == 0x02 {
		if err := socks5UserPass(conn, user, pass); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	portNum, err := net.LookupPort(network, port)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	addrType := byte(0x03)
	addrBytes := []byte(host)
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			addrType = 0x01
			addrBytes = ip.To4()
		} else {
			addrType = 0x04
			addrBytes = ip.To16()
		}
	} else if proxyURL.Scheme == "socks5" {
		ips, err := net.DefaultResolver.LookupIP(ctx, network, host)
		if err == nil && len(ips) > 0 {
			ip := ips[0]
			if ip.To4() != nil {
				addrType = 0x01
				addrBytes = ip.To4()
			} else {
				addrType = 0x04
				addrBytes = ip.To16()
			}
		}
	}

	req := []byte{0x05, 0x01, 0x00, addrType}
	if addrType == 0x03 {
		req = append(req, byte(len(addrBytes)))
	}
	req = append(req, addrBytes...)
	req = append(req, byte(portNum>>8), byte(portNum))
	if _, err := conn.Write(req); err != nil {
		_ = conn.Close()
		return nil, err
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if reply[1] != 0x00 {
		_ = conn.Close()
		return nil, fmt.Errorf("socks5 connect failed")
	}
	// consume bind addr
	var skip int
	switch reply[3] {
	case 0x01:
		skip = 4
	case 0x04:
		skip = 16
	case 0x03:
		lenb := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenb); err != nil {
			_ = conn.Close()
			return nil, err
		}
		skip = int(lenb[0])
	}
	if skip > 0 {
		if _, err := io.ReadFull(conn, make([]byte, skip)); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	if _, err := io.ReadFull(conn, make([]byte, 2)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func socks5UserPass(conn net.Conn, user, pass string) error {
	if user == "" {
		user = " "
	}
	if pass == "" {
		pass = " "
	}
	ub := []byte(user)
	pb := []byte(pass)
	req := []byte{0x01, byte(len(ub))}
	req = append(req, ub...)
	req = append(req, byte(len(pb)))
	req = append(req, pb...)
	if _, err := conn.Write(req); err != nil {
		return err
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}
	if resp[1] != 0x00 {
		return fmt.Errorf("socks5 auth failed")
	}
	return nil
}
func (c *Client) doOnce(ctx context.Context, spec RequestSpec) (*http.Response, error) {
	method := spec.Method
	if method == "" {
		method = c.cfg.RequestMethod
	}
	body := spec.Body
	if body == "" {
		body = c.cfg.RequestData
	}
	if method == "" {
		if body != "" {
			method = http.MethodPost
		} else {
			method = http.MethodGet
		}
	}
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, spec.URL, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range c.cfg.RequestHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}
	if c.cfg.RandomUserAgent {
		req.Header.Set("User-Agent", randomUserAgent(c.rand))
	} else if c.cfg.RequestUserAgent != "" {
		req.Header.Set("User-Agent", c.cfg.RequestUserAgent)
	}
	for k, v := range c.cfg.RequestCookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	if c.cfg.BasicAuthUser != "" {
		req.SetBasicAuth(c.cfg.BasicAuthUser, c.cfg.BasicAuthPass)
	}
	return c.http.Do(req)
}

func (c *Client) validateTarget(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return errors.New("missing host")
	}
	if c.scope != nil {
		if _, ok := c.scope[host]; !ok {
			return fmt.Errorf("host %s outside allowed scope", host)
		}
	}
	if !c.cfg.SafeMode {
		return nil
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return errors.New("safe mode: localhost blocked")
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		if blockedIP(ip) {
			return fmt.Errorf("safe mode: private IP %s blocked", ip.String())
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return fmt.Errorf("safe mode: host %s resolved to blocked IP", host)
		}
	}
	return nil
}

func blockedIP(ip netip.Addr) bool {
	if !ip.IsValid() {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

func isRetryStatus(code int, retry429 bool) bool {
	return (retry429 && code == http.StatusTooManyRequests) || (code >= 500 && code <= 599)
}

func (c *Client) backoff(attempt int) time.Duration {
	base := float64(c.cfg.RetryBackoff)
	pow := math.Pow(2, float64(attempt))
	jitter := 0.8 + c.rand.Float64()*0.4
	return time.Duration(base * pow * jitter)
}

func randomUserAgent(r *rand.Rand) string {
	if r == nil {
		return "PSFuzz/1.0.0"
	}
	uas := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13.4; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Mobile Safari/537.36",
		"curl/8.5.0",
		"Wget/1.21.4",
	}
	return uas[r.Intn(len(uas))]
}

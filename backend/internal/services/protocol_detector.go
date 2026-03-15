package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ProtocolProbeResult holds the result of probing a single protocol.
type ProtocolProbeResult struct {
	Protocol  string `json:"protocol"`
	Available bool   `json:"available"`
	Port      int    `json:"port"`
	Error     string `json:"error,omitempty"`
}

// ProtocolDetectionResult holds all probe results for a BMC.
type ProtocolDetectionResult struct {
	BMCIPAddress        string                `json:"bmc_ip_address"`
	RecommendedProtocol string                `json:"recommended_protocol"`
	Probes              []ProtocolProbeResult `json:"probes"`
}

// ProtocolDetector probes a BMC to determine which protocols are available.
type ProtocolDetector struct{}

func NewProtocolDetector() *ProtocolDetector { return &ProtocolDetector{} }

// Detect probes the BMC for Redfish, IPMI, SNMP, and SSH availability.
// Returns results in fallback priority order: Redfish → IPMI → SNMP → SSH.
func (d *ProtocolDetector) Detect(ctx context.Context, bmcIP string) ProtocolDetectionResult {
	// Remove brackets if an IPv6 address is already bracketed to prevent double-bracketing in JoinHostPort
	if len(bmcIP) > 2 && bmcIP[0] == '[' && bmcIP[len(bmcIP)-1] == ']' {
		bmcIP = bmcIP[1 : len(bmcIP)-1]
	}

	result := ProtocolDetectionResult{BMCIPAddress: bmcIP}

	// Probe in parallel
	type probe struct {
		protocol string
		port     int
		fn       func() (bool, error)
	}

	probes := []probe{
		{"redfish", 443, func() (bool, error) { return d.probeRedfish(ctx, bmcIP, 443) }},
		{"redfish_http", 80, func() (bool, error) { return d.probeRedfish(ctx, bmcIP, 80) }},
		{"ipmi", 623, func() (bool, error) { return d.probeUDP(ctx, bmcIP, 623) }},
		{"snmp", 161, func() (bool, error) { return d.probeUDP(ctx, bmcIP, 161) }},
		{"ssh", 22, func() (bool, error) { return d.probeTCP(ctx, bmcIP, 22) }},
	}

	type probeResult struct {
		idx    int
		result ProtocolProbeResult
	}
	ch := make(chan probeResult, len(probes))

	for i, p := range probes {
		go func(idx int, pr probe) {
			ok, err := pr.fn()
			res := ProtocolProbeResult{Protocol: pr.protocol, Port: pr.port, Available: ok}
			if err != nil {
				res.Error = err.Error()
			}
			ch <- probeResult{idx: idx, result: res}
		}(i, p)
	}

	probeResults := make([]ProtocolProbeResult, len(probes))
	for range probes {
		r := <-ch
		probeResults[r.idx] = r.result
	}

	// Filter out redfish_http duplicate — only include if redfish (443) failed
	for _, pr := range probeResults {
		if pr.Protocol == "redfish_http" {
			continue
		}
		result.Probes = append(result.Probes, pr)
	}

	// Determine recommended protocol in fallback order
	priority := []string{"redfish", "ipmi", "snmp", "ssh"}
	for _, proto := range priority {
		for _, pr := range result.Probes {
			if pr.Protocol == proto && pr.Available {
				result.RecommendedProtocol = proto
				break
			}
		}
		if result.RecommendedProtocol != "" {
			break
		}
	}

	if result.RecommendedProtocol == "" {
		result.RecommendedProtocol = "unknown"
	}

	return result
}

func (d *ProtocolDetector) probeRedfish(ctx context.Context, host string, port int) (bool, error) {
	scheme := "https"
	if port == 80 {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s/redfish/v1", scheme, net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	// Any response (even 401) means Redfish is available
	return resp.StatusCode < 500, nil
}

func (d *ProtocolDetector) probeTCP(ctx context.Context, host string, port int) (bool, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

func (d *ProtocolDetector) probeUDP(ctx context.Context, host string, port int) (bool, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("udp", addr, 3*time.Second)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

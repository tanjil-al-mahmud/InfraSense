package services

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/infrasense/backend/internal/models"
)

// IPMIService handles connections to Generic IPMI/Legacy BMC devices
type IPMIService struct{}

func NewIPMIService() *IPMIService {
	return &IPMIService{}
}

// TestConnection tests IPMI connectivity using lanplus
func (s *IPMIService) TestConnection(ctx context.Context, host string, port int, cred *models.DeviceCredential, password string) models.ConnectionTestResult {
	if cred == nil || cred.Username == nil || *cred.Username == "" {
		return models.ConnectionTestResult{
			Success: false,
			Message: "IPMI requires username and password",
		}
	}

	if port == 0 {
		port = 623
	}

	// Just run an ipmitool chassis status command to verify login
	cmdArgs := []string{
		"-I", "lanplus",
		"-H", host,
		"-p", fmt.Sprintf("%d", port),
		"-U", *cred.Username,
		"-P", password,
		"-R", "1", // retry once
		"-N", "5", // 5 second timeout
		"chassis", "status",
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(out))
		if errMsg == "" {
			errMsg = err.Error()
		}
		slog.Error("IPMI test connection failed",
			"event", "ipmi_test_failed",
			"host", host,
			"error", err,
			"output", errMsg,
		)

		var friendlyMsg string
		switch {
		case strings.Contains(errMsg, "RAKP 2 HMAC is invalid"):
			friendlyMsg = "Authentication failed: Wrong username or password"
		case strings.Contains(errMsg, "Unable to establish IPMI"):
			friendlyMsg = "Cannot connect: IPMI may be disabled or port blocked. Check BMC settings."
		case strings.Contains(errMsg, "Error in open session"):
			friendlyMsg = "Session error: Check IPMI over LAN is enabled in BMC settings"
		case strings.Contains(errMsg, "timeout"), strings.Contains(errMsg, "Timed out"):
			friendlyMsg = "Connection timeout: Device unreachable. Check IP address and network."
		case strings.Contains(errMsg, "connection refused"):
			friendlyMsg = fmt.Sprintf("Connection refused on port %d. Check IPMI port in BMC settings.", port)
		default:
			friendlyMsg = fmt.Sprintf("IPMI error: %s", strings.TrimSpace(errMsg))
		}

		return models.ConnectionTestResult{
			Success: false,
			Message: friendlyMsg,
		}
	}

	return models.ConnectionTestResult{
		Success: true,
		Message: "Connected successfully via IPMI 2.0 (lanplus)",
	}
}

// SyncDevice pulls FRU data to populate the device inventory automatically
func (s *IPMIService) SyncDevice(ctx context.Context, host string, port int, cred *models.DeviceCredential, password string) (models.DeviceSyncResult, error) {
	result := models.DeviceSyncResult{
		Success: true,
		Message: "Sync completed via IPMI 2.0",
	}

	if port == 0 {
		port = 623
	}

	cmdArgs := []string{
		"-I", "lanplus",
		"-H", host,
		"-p", fmt.Sprintf("%d", port),
		"-U", *cred.Username,
		"-P", password,
		"-R", "1",
		"-N", "5",
		"fru",
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Log error but don't fail entirely, some older BMCs might not return FRU
		slog.Error("IPMI sync fru failed", "event", "ipmi_sync_fru_error", "host", host, "error", err, "output", string(out))
		result.Message = fmt.Sprintf("Sync connected, but FRU read failed: %v", err)
		return result, nil
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Product Manufacturer", "Board Mfg":
			if result.Manufacturer == nil || *result.Manufacturer == "" {
				m := val
				result.Manufacturer = &m
			}
		case "Product Name":
			m := val
			result.Model = &m
		case "Product Serial":
			s := val
			result.SerialNumber = &s
		}
	}

	// Additional attempt to get Firmware Version via BMC info if needed:
	mcCmd := exec.CommandContext(ctx, "ipmitool", "-I", "lanplus", "-H", host, "-p", fmt.Sprintf("%d", port), "-U", *cred.Username, "-P", password, "-R", "1", "-N", "5", "mc", "info")
	if mcOut, mcErr := mcCmd.CombinedOutput(); mcErr == nil {
		for _, line := range strings.Split(string(mcOut), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "Firmware Revision") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					// Add to system info or log it (FirmwareVersion is not a raw field on DeviceSyncResult)
					slog.Info("IPMI found firmware", "host", host, "version", strings.TrimSpace(parts[1]))
				}
				break
			}
		}
	}

	return result, nil
}

// PowerControl executes power actions via ipmitool
func (s *IPMIService) PowerControl(ctx context.Context, host string, port int, cred *models.DeviceCredential, password string, resetType string) models.PowerControlResult {
	if cred == nil || cred.Username == nil || *cred.Username == "" {
		return models.PowerControlResult{
			Success: false,
			Message: "IPMI requires username and password",
		}
	}

	if port == 0 {
		port = 623
	}

	var ipmiCmd string
	switch resetType {
	case "On":
		ipmiCmd = "on"
	case "ForceOff":
		ipmiCmd = "off"
	case "GracefulShutdown":
		ipmiCmd = "soft"
	case "ForceRestart":
		ipmiCmd = "reset"
	case "PowerCycle":
		ipmiCmd = "cycle"
	case "GracefulRestart":
		// IPMI 2.0 doesn't have a direct "GracefulRestart", "soft" is a graceful shutdown
		// which might be followed by a restart if configured in BIOS, but "reset" or "cycle"
		// are common for restarts. We'll use "cycle" for PowerCycle and "reset" for ForceRestart.
		// For GracefulRestart, we'll try "soft".
		ipmiCmd = "soft"
	default:
		return models.PowerControlResult{
			Success: false,
			Message: fmt.Sprintf("Unsupported power action for IPMI: %s", resetType),
		}
	}

	cmdArgs := []string{
		"-I", "lanplus",
		"-H", host,
		"-p", fmt.Sprintf("%d", port),
		"-U", *cred.Username,
		"-P", password,
		"-R", "1",
		"-N", "5",
		"chassis", "power", ipmiCmd,
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("IPMI power control failed", "host", host, "action", resetType, "error", err, "output", string(out))
		return models.PowerControlResult{
			Success:   false,
			Message:   fmt.Sprintf("IPMI power command failed: %v", err),
			ResetType: resetType,
		}
	}

	return models.PowerControlResult{
		Success:   true,
		Message:   fmt.Sprintf("IPMI power command '%s' sent successfully", ipmiCmd),
		ResetType: resetType,
	}
}

// BootControl executes boot override actions via ipmitool
func (s *IPMIService) BootControl(ctx context.Context, host string, port int, cred *models.DeviceCredential, password string, target string, once bool) models.BootControlResult {
	if cred == nil || cred.Username == nil || *cred.Username == "" {
		return models.BootControlResult{
			Success: false,
			Message: "IPMI requires username and password",
		}
	}

	if port == 0 {
		port = 623
	}

	var ipmiTarget string
	switch target {
	case "Pxe":
		ipmiTarget = "pxe"
	case "Cd":
		ipmiTarget = "cdrom"
	case "Hdd":
		ipmiTarget = "disk"
	case "BiosSetup":
		ipmiTarget = "bios"
	case "None":
		ipmiTarget = "none"
	default:
		return models.BootControlResult{
			Success: false,
			Message: fmt.Sprintf("Unsupported boot target for IPMI: %s", target),
		}
	}

	cmdArgs := []string{
		"-I", "lanplus",
		"-H", host,
		"-p", fmt.Sprintf("%d", port),
		"-U", *cred.Username,
		"-P", password,
		"-R", "1",
		"-N", "5",
		"chassis", "bootdev", ipmiTarget,
	}
	
	// Add options for 'once' or persistent
	if !once {
		cmdArgs = append(cmdArgs, "options=persistent")
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("IPMI boot control failed", "host", host, "target", target, "error", err, "output", string(out))
		return models.BootControlResult{
			Success: false,
			Message: fmt.Sprintf("IPMI boot command failed: %v", err),
			Target:  target,
		}
	}

	return models.BootControlResult{
		Success: true,
		Message: fmt.Sprintf("IPMI boot target set to '%s' (once: %v)", ipmiTarget, once),
		Target:  target,
	}
}

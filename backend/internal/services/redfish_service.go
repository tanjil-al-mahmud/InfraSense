package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/infrasense/backend/internal/models"
)

// RedfishService handles on-demand Redfish API operations (test connection, sync, power control).
type RedfishService struct{}

func NewRedfishService() *RedfishService { return &RedfishService{} }

func buildHTTPClient(cred *models.DeviceCredential, timeout time.Duration) *http.Client {
	tlsCfg := &tls.Config{InsecureSkipVerify: !cred.SSLVerify} //nolint:gosec
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{TLSClientConfig: tlsCfg, MaxIdleConnsPerHost: 4},
	}
}

func baseURL(bmcIP string, cred *models.DeviceCredential) string {
	scheme := cred.HTTPScheme
	if scheme == "" {
		scheme = "https"
	}
	port := 443
	if cred.Port != nil {
		port = *cred.Port
	} else if scheme == "http" {
		port = 80
	}
	return fmt.Sprintf("%s://%s:%d", scheme, bmcIP, port)
}

func redfishGet(ctx context.Context, client *http.Client, base, path, user, pass string) (map[string]any, int, error) {
	return redfishGetWithRetry(ctx, client, base, path, user, pass, 2)
}

func redfishGetWithRetry(ctx context.Context, client *http.Client, base, path, user, pass string, maxRetries int) (map[string]any, int, error) {
	url := base + path
	var lastErr error
	var lastStatus int
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("create request: %w", err)
		}
		if user != "" || pass != "" {
			req.SetBasicAuth(user, pass)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("OData-Version", "4.0")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if ctx.Err() != nil {
				return nil, 0, ctx.Err()
			}
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastStatus = resp.StatusCode
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, resp.StatusCode, fmt.Errorf("HTTP %d: authentication failed", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
			if resp.StatusCode < 500 {
				return nil, resp.StatusCode, lastErr
			}
			continue
		}
		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("decode JSON: %w", err)
		}
		return result, resp.StatusCode, nil
	}
	return nil, lastStatus, lastErr
}

func redfishPost(ctx context.Context, client *http.Client, base, path, user, pass string, body map[string]any) (map[string]any, int, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, strings.NewReader(string(b)))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("OData-Version", "4.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: authentication failed", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}
	if len(respBody) == 0 {
		return map[string]any{}, resp.StatusCode, nil
	}
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return map[string]any{}, resp.StatusCode, nil
	}
	return result, resp.StatusCode, nil
}

func redfishPatch(ctx context.Context, client *http.Client, base, path, user, pass string, body map[string]any) (map[string]any, int, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, base+path, strings.NewReader(string(b)))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("OData-Version", "4.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: authentication failed", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}
	var result map[string]any
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &result)
	}
	return result, resp.StatusCode, nil
}

func redfishGetCollection(ctx context.Context, client *http.Client, base, path, user, pass string) ([]map[string]any, error) {
	data, _, err := redfishGet(ctx, client, base, path, user, pass)
	if err != nil {
		return nil, err
	}
	members, _ := data["Members"].([]any)
	var result []map[string]any
	for _, m := range members {
		if link, ok := m.(map[string]any); ok {
			if href, ok := link["@odata.id"].(string); ok {
				item, _, err := redfishGet(ctx, client, base, href, user, pass)
				if err == nil {
					result = append(result, item)
				}
			}
		}
	}
	return result, nil
}

// ── Helper extractors ─────────────────────────────────────────────────────────

func strVal(m map[string]any, key string) *string {
	if v, ok := m[key].(string); ok && v != "" {
		return &v
	}
	return nil
}

func floatVal(m map[string]any, key string) *float64 {
	switch v := m[key].(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	}
	return nil
}

func intVal(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func int64Val(m map[string]any, key string) *int64 {
	switch v := m[key].(type) {
	case float64:
		i := int64(v)
		return &i
	case int:
		i := int64(v)
		return &i
	case int64:
		return &v
	}
	return nil
}

func boolVal(m map[string]any, key string) *bool {
	if v, ok := m[key].(bool); ok {
		return &v
	}
	return nil
}

func healthVal(m map[string]any) string {
	if status, ok := m["Status"].(map[string]any); ok {
		if h, ok := status["Health"].(string); ok {
			return h
		}
	}
	return ""
}

func stateVal(m map[string]any) string {
	if status, ok := m["Status"].(map[string]any); ok {
		if s, ok := status["State"].(string); ok {
			return s
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func oDataLink(m map[string]any, key string) string {
	if sub, ok := m[key].(map[string]any); ok {
		if href, ok := sub["@odata.id"].(string); ok {
			return href
		}
	}
	return ""
}

func strDeref(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
}

func portFromCred(cred *models.DeviceCredential) int {
	if cred.Port != nil {
		return *cred.Port
	}
	return 443
}

// ── Path discovery ────────────────────────────────────────────────────────────

func discoverSystemPath(ctx context.Context, client *http.Client, base, user, pass string) (string, error) {
	collData, _, err := redfishGet(ctx, client, base, "/redfish/v1/Systems", user, pass)
	if err == nil {
		if members, ok := collData["Members"].([]any); ok && len(members) > 0 {
			if first, ok := members[0].(map[string]any); ok {
				if href, ok := first["@odata.id"].(string); ok && href != "" {
					return href, nil
				}
			}
		}
	}
	for _, path := range []string{
		"/redfish/v1/Systems/System.Embedded.1",
		"/redfish/v1/Systems/1",
		"/redfish/v1/Systems/Self",
	} {
		_, status, err := redfishGet(ctx, client, base, path, user, pass)
		if err == nil {
			return path, nil
		}
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			return "", fmt.Errorf("HTTP %d: authentication failed", status)
		}
	}
	return "", fmt.Errorf("could not discover system path")
}

func discoverChassisPath(ctx context.Context, client *http.Client, base, user, pass string) string {
	data, _, err := redfishGet(ctx, client, base, "/redfish/v1/Chassis", user, pass)
	if err == nil {
		if members, ok := data["Members"].([]any); ok && len(members) > 0 {
			if first, ok := members[0].(map[string]any); ok {
				if href, ok := first["@odata.id"].(string); ok && href != "" {
					return href
				}
			}
		}
	}
	for _, p := range []string{"/redfish/v1/Chassis/System.Embedded.1", "/redfish/v1/Chassis/1"} {
		if _, _, err := redfishGet(ctx, client, base, p, user, pass); err == nil {
			return p
		}
	}
	return ""
}

func discoverManagerPath(ctx context.Context, client *http.Client, base, user, pass string) string {
	data, _, err := redfishGet(ctx, client, base, "/redfish/v1/Managers", user, pass)
	if err == nil {
		if members, ok := data["Members"].([]any); ok && len(members) > 0 {
			if first, ok := members[0].(map[string]any); ok {
				if href, ok := first["@odata.id"].(string); ok && href != "" {
					return href
				}
			}
		}
	}
	for _, p := range []string{"/redfish/v1/Managers/iDRAC.Embedded.1", "/redfish/v1/Managers/1"} {
		if _, _, err := redfishGet(ctx, client, base, p, user, pass); err == nil {
			return p
		}
	}
	return ""
}

func classifyError(err error, statusCode int) string {
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return "AUTH_FAILED"
	}
	if err == nil {
		return "UNKNOWN"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
		return "TIMEOUT"
	case strings.Contains(msg, "certificate") || strings.Contains(msg, "tls") || strings.Contains(msg, "x509"):
		return "SSL_ERROR"
	case strings.Contains(msg, "connection refused") || strings.Contains(msg, "no route"):
		return "BMC_OFFLINE"
	case statusCode == http.StatusNotFound:
		return "ENDPOINT_NOT_FOUND"
	case statusCode >= 500:
		return "BMC_ERROR"
	default:
		return "NETWORK_ERROR"
	}
}

// ── TestConnection ────────────────────────────────────────────────────────────

// TestConnection verifies BMC reachability, Redfish endpoint, and authentication.
func (s *RedfishService) TestConnection(ctx context.Context, bmcIP string, cred *models.DeviceCredential, password string) models.ConnectionTestResult {
	timeout := time.Duration(cred.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := buildHTTPClient(cred, timeout)
	base := baseURL(bmcIP, cred)

	data, statusCode, err := redfishGet(ctx, client, base, "/redfish/v1", "", "")
	if err != nil {
		code := classifyError(err, statusCode)
		return models.ConnectionTestResult{
			Success:   false,
			Message:   fmt.Sprintf("Cannot reach Redfish endpoint (%s:%d): %v", bmcIP, portFromCred(cred), err),
			ErrorCode: code,
		}
	}
	redfishVersion, _ := data["RedfishVersion"].(string)

	username := ""
	if cred.Username != nil {
		username = *cred.Username
	}
	systemPath, err := discoverSystemPath(ctx, client, base, username, password)
	if err != nil {
		code := classifyError(err, 0)
		if strings.Contains(err.Error(), "authentication failed") {
			code = "AUTH_FAILED"
		}
		return models.ConnectionTestResult{
			Success:   false,
			Message:   fmt.Sprintf("Authentication failed: %v", err),
			ErrorCode: code,
		}
	}
	_ = systemPath
	return models.ConnectionTestResult{
		Success:        true,
		Message:        fmt.Sprintf("Connection successful — Redfish v%s", redfishVersion),
		RedfishVersion: redfishVersion,
	}
}

// ── SyncDevice ────────────────────────────────────────────────────────────────

// SyncDevice connects to the BMC and retrieves full hardware telemetry.
func (s *RedfishService) SyncDevice(ctx context.Context, bmcIP string, cred *models.DeviceCredential, password string) (models.DeviceSyncResult, error) {
	timeout := time.Duration(cred.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	client := buildHTTPClient(cred, timeout)
	base := baseURL(bmcIP, cred)
	username := ""
	if cred.Username != nil {
		username = *cred.Username
	}

	result := models.DeviceSyncResult{}
	addStep := func(name, status, msg string) {
		result.Steps = append(result.Steps, models.SyncStep{Name: name, Status: status, Message: msg})
	}

	// ── Step 1: System discovery ──────────────────────────────────────────────
	systemPath, err := discoverSystemPath(ctx, client, base, username, password)
	if err != nil {
		addStep("System Discovery", "error", err.Error())
		return result, fmt.Errorf("system discovery failed: %w", err)
	}
	addStep("System Discovery", "ok", systemPath)

	// ── Step 2: System information ────────────────────────────────────────────
	sysData, _, err := redfishGet(ctx, client, base, systemPath, username, password)
	if err != nil {
		addStep("System Information", "error", err.Error())
		return result, fmt.Errorf("failed to retrieve system information: %w", err)
	}
	addStep("System Information", "ok", "")

	result.Success = true
	result.Message = "Sync completed"
	result.Manufacturer = strVal(sysData, "Manufacturer")
	result.Model = strVal(sysData, "Model")
	result.SerialNumber = strVal(sysData, "SerialNumber")
	result.ServiceTag = strVal(sysData, "SKU")
	result.AssetTag = strVal(sysData, "AssetTag")
	result.PowerState = strVal(sysData, "PowerState")
	result.BIOSVersion = strVal(sysData, "BiosVersion")
	result.SystemUUID = strVal(sysData, "UUID")
	result.SystemRevision = strVal(sysData, "SystemType")
	result.SystemUptimeSeconds = int64Val(sysData, "PowerOnMinutes")
	if result.SystemUptimeSeconds != nil {
		v := *result.SystemUptimeSeconds * 60
		result.SystemUptimeSeconds = &v
	}
	// Boot mode from BootProgress or Boot
	if boot, ok := sysData["Boot"].(map[string]any); ok {
		if mode := strVal(boot, "BootSourceOverrideMode"); mode != nil {
			result.BootMode = mode
		}
	}
	if status, ok := sysData["Status"].(map[string]any); ok {
		if h, ok := status["Health"].(string); ok {
			result.HealthStatus = &h
		}
	}
	// Lifecycle controller version from OEM
	if oem, ok := sysData["Oem"].(map[string]any); ok {
		if dell, ok := oem["Dell"].(map[string]any); ok {
			if dellSys, ok := dell["DellSystem"].(map[string]any); ok {
				result.LifecycleControllerVersion = strVal(dellSys, "LifecycleControllerVersion")
			}
		}
	}

	// ── Step 3: OS Information ────────────────────────────────────────────────
	if osLink := oDataLink(sysData, "OperatingSystem"); osLink != "" {
		osData, _, err := redfishGet(ctx, client, base, osLink, username, password)
		if err == nil {
			result.OS = &models.OSInfo{
				Name:    strDeref(strVal(osData, "Name"), ""),
				Version: strDeref(strVal(osData, "Version"), ""),
				Kernel:  strDeref(strVal(osData, "KernelVersion"), ""),
			}
			addStep("OS Information", "ok", result.OS.Name)
		} else {
			addStep("OS Information", "skipped", "not available")
		}
	} else {
		// Try OEM OS info (HPE iLO, Dell iDRAC)
		if oem, ok := sysData["Oem"].(map[string]any); ok {
			osName := ""
			osVer := ""
			if hpe, ok := oem["Hpe"].(map[string]any); ok {
				osName = strDeref(strVal(hpe, "OperatingSystemName"), "")
				osVer = strDeref(strVal(hpe, "OperatingSystemVersion"), "")
			}
			if dell, ok := oem["Dell"].(map[string]any); ok {
				if dellSys, ok := dell["DellSystem"].(map[string]any); ok {
					osName = strDeref(strVal(dellSys, "OSName"), osName)
					osVer = strDeref(strVal(dellSys, "OSVersion"), osVer)
				}
			}
			if osName != "" {
				result.OS = &models.OSInfo{Name: osName, Version: osVer}
			}
		}
		addStep("OS Information", "skipped", "no dedicated OS link")
	}

	// ── Step 4: Processors ────────────────────────────────────────────────────
	if procLink := oDataLink(sysData, "Processors"); procLink != "" {
		procs, err := redfishGetCollection(ctx, client, base, procLink, username, password)
		if err == nil {
			for _, p := range procs {
				proc := models.ProcessorInfo{
					Name:         strDeref(strVal(p, "Name"), strDeref(strVal(p, "Id"), "CPU")),
					Model:        strDeref(strVal(p, "Model"), ""),
					Manufacturer: strDeref(strVal(p, "Manufacturer"), ""),
					Socket:       strDeref(strVal(p, "Socket"), ""),
					Cores:        intVal(p, "TotalCores"),
					Threads:      intVal(p, "TotalThreads"),
					SpeedMHz:     intVal(p, "OperatingSpeedMHz"),
					MaxSpeedMHz:  intVal(p, "MaxSpeedMHz"),
					Health:       healthVal(p),
				}
				// Cache size
				if cache, ok := p["Cache"].([]any); ok && len(cache) > 0 {
					totalMiB := 0
					for _, c := range cache {
						if cm, ok := c.(map[string]any); ok {
							totalMiB += intVal(cm, "MaxSizeMiB")
						}
					}
					proc.CacheSizeMiB = totalMiB
				}
				result.Processors = append(result.Processors, proc)
			}
			addStep("Processors", "ok", fmt.Sprintf("%d found", len(result.Processors)))
		} else {
			addStep("Processors", "error", err.Error())
		}
	} else {
		addStep("Processors", "skipped", "no link in system resource")
	}

	// ── Step 5: Memory ────────────────────────────────────────────────────────
	if memLink := oDataLink(sysData, "Memory"); memLink != "" {
		mems, err := redfishGetCollection(ctx, client, base, memLink, username, password)
		if err == nil {
			var totalMiB float64
			for _, m := range mems {
				capMiB := floatVal(m, "CapacityMiB")
				if capMiB != nil {
					totalMiB += *capMiB
				}
				capGB := 0.0
				if capMiB != nil {
					capGB = *capMiB / 1024.0
				}
				mem := models.MemoryInfo{
					Name:         strDeref(strVal(m, "Name"), strDeref(strVal(m, "Id"), "DIMM")),
					CapacityGB:   capGB,
					Manufacturer: strDeref(strVal(m, "Manufacturer"), ""),
					PartNumber:   strDeref(strVal(m, "PartNumber"), ""),
					SerialNumber: strDeref(strVal(m, "SerialNumber"), ""),
					Location:     strDeref(strVal(m, "DeviceLocator"), strDeref(strVal(m, "MemoryLocation"), "")),
					SpeedMHz:     intVal(m, "OperatingSpeedMhz"),
					MemoryType:   strDeref(strVal(m, "MemoryDeviceType"), strDeref(strVal(m, "MemoryType"), "")),
					Health:       healthVal(m),
				}
				// ECC
				if ec := strVal(m, "ErrorCorrection"); ec != nil {
					ecc := strings.Contains(strings.ToLower(*ec), "ecc") || strings.Contains(strings.ToLower(*ec), "multi")
					mem.ECCEnabled = &ecc
				}
				result.MemoryModules = append(result.MemoryModules, mem)
			}
			if totalMiB > 0 {
				gb := totalMiB / 1024.0
				result.MemoryTotalGB = &gb
			}
			addStep("Memory", "ok", fmt.Sprintf("%d modules", len(result.MemoryModules)))
		} else {
			addStep("Memory", "error", err.Error())
		}
	} else {
		addStep("Memory", "skipped", "no link in system resource")
	}

	// ── Step 6: Storage (controllers + physical + virtual disks + enclosures) ─
	if storLink := oDataLink(sysData, "Storage"); storLink != "" {
		controllers, err := redfishGetCollection(ctx, client, base, storLink, username, password)
		if err == nil {
			for _, ctrl := range controllers {
				ctrlName := strDeref(strVal(ctrl, "Name"), strDeref(strVal(ctrl, "Id"), "Controller"))
				fwVer := ""
				batteryHealth := ""
				if ctrls, ok := ctrl["StorageControllers"].([]any); ok && len(ctrls) > 0 {
					if c0, ok := ctrls[0].(map[string]any); ok {
						fwVer = strDeref(strVal(c0, "FirmwareVersion"), "")
					}
				}
				// Battery health from OEM (Dell)
				if oem, ok := ctrl["Oem"].(map[string]any); ok {
					if dell, ok := oem["Dell"].(map[string]any); ok {
						if bat, ok := dell["DellControllerBattery"].(map[string]any); ok {
							batteryHealth = healthVal(bat)
						}
					}
				}
				result.StorageControllers = append(result.StorageControllers, models.StorageControllerInfo{
					Name:          ctrlName,
					Model:         strDeref(strVal(ctrl, "Model"), ""),
					Manufacturer:  strDeref(strVal(ctrl, "Manufacturer"), ""),
					Health:        healthVal(ctrl),
					FirmwareVer:   fwVer,
					BatteryHealth: batteryHealth,
				})

				// Physical drives
				collectDrives := func(driveLinks []any) {
					for _, m := range driveLinks {
						if link, ok := m.(map[string]any); ok {
							if href, ok := link["@odata.id"].(string); ok {
								d, _, err := redfishGet(ctx, client, base, href, username, password)
								if err == nil {
									result.Drives = append(result.Drives, parseDrive(d))
								}
							}
						}
					}
				}
				if drivesLink := oDataLink(ctrl, "Drives"); drivesLink != "" {
					drives, _ := redfishGetCollection(ctx, client, base, drivesLink, username, password)
					for _, d := range drives {
						result.Drives = append(result.Drives, parseDrive(d))
					}
				} else if members, ok := ctrl["Drives"].([]any); ok {
					collectDrives(members)
				}

				// Virtual disks (Volumes)
				if volLink := oDataLink(ctrl, "Volumes"); volLink != "" {
					vols, _ := redfishGetCollection(ctx, client, base, volLink, username, password)
					for _, v := range vols {
						capBytes := floatVal(v, "CapacityBytes")
						capGB := 0.0
						if capBytes != nil {
							capGB = *capBytes / (1024 * 1024 * 1024)
						}
						result.VirtualDisks = append(result.VirtualDisks, models.VirtualDiskInfo{
							Name:        strDeref(strVal(v, "Name"), strDeref(strVal(v, "Id"), "VD")),
							RAIDLevel:   strDeref(strVal(v, "RAIDType"), strDeref(strVal(v, "VolumeType"), "")),
							CapacityGB:  capGB,
							WritePolicy: strDeref(strVal(v, "WriteCachePolicy"), ""),
							ReadPolicy:  strDeref(strVal(v, "ReadCachePolicy"), ""),
							CachePolicy: strDeref(strVal(v, "CacheSetting"), ""),
							Health:      healthVal(v),
							Status:      stateVal(v),
						})
					}
				}

				// Storage enclosures
				if encLink := oDataLink(ctrl, "Enclosures"); encLink != "" {
					encs, _ := redfishGetCollection(ctx, client, base, encLink, username, password)
					for _, e := range encs {
						result.StorageEnclosures = append(result.StorageEnclosures, models.StorageEnclosureInfo{
							Name:        strDeref(strVal(e, "Name"), strDeref(strVal(e, "Id"), "Enclosure")),
							BackplaneID: strDeref(strVal(e, "Id"), ""),
							Controller:  ctrlName,
							Health:      healthVal(e),
						})
					}
				}
			}
			addStep("Storage", "ok", fmt.Sprintf("%d controllers, %d drives, %d virtual, %d enclosures",
				len(result.StorageControllers), len(result.Drives), len(result.VirtualDisks), len(result.StorageEnclosures)))
		} else {
			addStep("Storage", "error", err.Error())
		}
	} else {
		addStep("Storage", "skipped", "no link in system resource")
	}

	// ── Step 7: Chassis — Thermal, Power, Voltages, Intrusion ────────────────
	chassisPath := discoverChassisPath(ctx, client, base, username, password)
	if chassisPath != "" {
		// Thermal
		thermalData, _, err := redfishGet(ctx, client, base, chassisPath+"/Thermal", username, password)
		if err == nil {
			if temps, ok := thermalData["Temperatures"].([]any); ok {
				for _, t := range temps {
					if tm, ok := t.(map[string]any); ok {
						result.Temperatures = append(result.Temperatures, models.TemperatureReading{
							Name:            strDeref(strVal(tm, "Name"), "Sensor"),
							ReadingCelsius:  floatVal(tm, "ReadingCelsius"),
							UpperThreshWarn: floatVal(tm, "UpperThresholdNonCritical"),
							UpperThreshCrit: floatVal(tm, "UpperThresholdCritical"),
							Health:          healthVal(tm),
						})
					}
				}
			}
			if fans, ok := thermalData["Fans"].([]any); ok {
				for _, f := range fans {
					if fm, ok := f.(map[string]any); ok {
						reading := floatVal(fm, "Reading")
						if reading == nil {
							reading = floatVal(fm, "ReadingRPM")
						}
						fanID := strDeref(strVal(fm, "MemberId"), strDeref(strVal(fm, "Id"), ""))
						result.Fans = append(result.Fans, models.FanReading{
							Name:        strDeref(strVal(fm, "Name"), strDeref(strVal(fm, "FanName"), "Fan")),
							FanID:       fanID,
							ReadingRPM:  reading,
							LowerThresh: floatVal(fm, "LowerThresholdNonCritical"),
							Health:      healthVal(fm),
						})
					}
				}
			}
			addStep("Thermal Sensors", "ok", fmt.Sprintf("%d temps, %d fans", len(result.Temperatures), len(result.Fans)))
		} else {
			addStep("Thermal Sensors", "error", err.Error())
		}

		// Power & Voltages
		powerData, _, err := redfishGet(ctx, client, base, chassisPath+"/Power", username, password)
		if err == nil {
			if controls, ok := powerData["PowerControl"].([]any); ok && len(controls) > 0 {
				if ctrl, ok := controls[0].(map[string]any); ok {
					result.TotalPowerWatts = floatVal(ctrl, "PowerConsumedWatts")
				}
			}
			if psus, ok := powerData["PowerSupplies"].([]any); ok {
				for _, p := range psus {
					if pm, ok := p.(map[string]any); ok {
						psu := models.PowerSupplyInfo{
							Name:             strDeref(strVal(pm, "Name"), strDeref(strVal(pm, "MemberId"), "PSU")),
							PowerInputWatts:  floatVal(pm, "PowerInputWatts"),
							PowerOutputWatts: floatVal(pm, "PowerOutputWatts"),
							LastPowerOutputW: floatVal(pm, "LastPowerOutputWatts"),
							PowerCapWatts:    floatVal(pm, "PowerCapacityWatts"),
							Health:           healthVal(pm),
							Status:           stateVal(pm),
						}
						// Redundancy
						if redArr, ok := pm["Redundancy"].([]any); ok && len(redArr) > 0 {
							if r0, ok := redArr[0].(map[string]any); ok {
								psu.Redundancy = strDeref(strVal(r0, "Mode"), "")
							}
						}
						result.PowerSupplies = append(result.PowerSupplies, psu)
					}
				}
			}
			if volts, ok := powerData["Voltages"].([]any); ok {
				for _, v := range volts {
					if vm, ok := v.(map[string]any); ok {
						result.Voltages = append(result.Voltages, models.VoltageReading{
							Name:            strDeref(strVal(vm, "Name"), "Voltage"),
							ReadingVolts:    floatVal(vm, "ReadingVolts"),
							UpperThreshWarn: floatVal(vm, "UpperThresholdNonCritical"),
							LowerThreshWarn: floatVal(vm, "LowerThresholdNonCritical"),
							Health:          healthVal(vm),
						})
					}
				}
			}
			addStep("Power & Voltages", "ok", fmt.Sprintf("%d PSUs, %d voltages", len(result.PowerSupplies), len(result.Voltages)))
		} else {
			addStep("Power & Voltages", "error", err.Error())
		}

		// Intrusion detection
		chassisData, _, err := redfishGet(ctx, client, base, chassisPath, username, password)
		if err == nil {
			if phySec, ok := chassisData["PhysicalSecurity"].(map[string]any); ok {
				if state, ok := phySec["IntrusionSensor"].(string); ok {
					result.IntrusionStatus = &state
				}
			}
		}
	} else {
		addStep("Thermal Sensors", "skipped", "chassis path not found")
		addStep("Power & Voltages", "skipped", "chassis path not found")
	}

	// ── Step 8: Network Interfaces ────────────────────────────────────────────
	if etherLink := oDataLink(sysData, "EthernetInterfaces"); etherLink != "" {
		nics, err := redfishGetCollection(ctx, client, base, etherLink, username, password)
		if err == nil {
			for _, n := range nics {
				mac := strDeref(strVal(n, "MACAddress"), strDeref(strVal(n, "PermanentMACAddress"), ""))
				if mac == "" {
					continue
				}
				nic := models.FullNICInfo{
					Name:       strDeref(strVal(n, "Name"), strDeref(strVal(n, "Id"), "NIC")),
					Model:      strDeref(strVal(n, "Description"), ""),
					MACAddress: mac,
					LinkStatus: strDeref(strVal(n, "LinkStatus"), ""),
					SpeedMbps:  intVal(n, "SpeedMbps"),
					Health:     healthVal(n),
				}
				// Firmware version from OEM or direct field
				if fw := strVal(n, "FirmwareVersion"); fw != nil {
					nic.FirmwareVersion = *fw
				} else if oem, ok := n["Oem"].(map[string]any); ok {
					if dell, ok := oem["Dell"].(map[string]any); ok {
						if dellNIC, ok := dell["DellNIC"].(map[string]any); ok {
							nic.FirmwareVersion = strDeref(strVal(dellNIC, "FamilyVersion"), "")
						}
					}
				}
				// IPv4
				if addrs, ok := n["IPv4Addresses"].([]any); ok && len(addrs) > 0 {
					if a, ok := addrs[0].(map[string]any); ok {
						nic.IPv4Address = strDeref(strVal(a, "Address"), "")
					}
				}
				// IPv6
				if addrs, ok := n["IPv6Addresses"].([]any); ok && len(addrs) > 0 {
					if a, ok := addrs[0].(map[string]any); ok {
						nic.IPv6Address = strDeref(strVal(a, "Address"), "")
					}
				}
				result.NICs = append(result.NICs, nic)
			}
			addStep("Network Interfaces", "ok", fmt.Sprintf("%d NICs", len(result.NICs)))
		} else {
			addStep("Network Interfaces", "error", err.Error())
		}
	} else {
		addStep("Network Interfaces", "skipped", "no link in system resource")
	}

	// ── Step 9: PCIe Slots ────────────────────────────────────────────────────
	if pcieLink := oDataLink(sysData, "PCIeSlots"); pcieLink != "" {
		pcieData, _, err := redfishGet(ctx, client, base, pcieLink, username, password)
		if err == nil {
			if slots, ok := pcieData["Slots"].([]any); ok {
				for i, s := range slots {
					if sm, ok := s.(map[string]any); ok {
						devName := ""
						if devs, ok := sm["PCIeDevice"].([]any); ok && len(devs) > 0 {
							if d, ok := devs[0].(map[string]any); ok {
								devName = strDeref(strVal(d, "Name"), "")
							}
						}
						result.PCIeSlots = append(result.PCIeSlots, models.PCIeSlotInfo{
							Name:       fmt.Sprintf("Slot %d", i+1),
							SlotType:   strDeref(strVal(sm, "SlotType"), ""),
							PCIeType:   strDeref(strVal(sm, "PCIeType"), ""),
							Status:     stateVal(sm),
							DeviceName: devName,
						})
					}
				}
			}
			addStep("PCIe Slots", "ok", fmt.Sprintf("%d slots", len(result.PCIeSlots)))
		} else {
			addStep("PCIe Slots", "error", err.Error())
		}
	} else {
		addStep("PCIe Slots", "skipped", "no link in system resource")
	}

	// ── Step 10: BMC / Manager info ───────────────────────────────────────────
	managerPath := discoverManagerPath(ctx, client, base, username, password)
	if managerPath != "" {
		mgrData, _, err := redfishGet(ctx, client, base, managerPath, username, password)
		if err == nil {
			result.BMCFirmware = strVal(mgrData, "FirmwareVersion")
			result.BMCName = strVal(mgrData, "Name")
			result.BMCHardwareVersion = strVal(mgrData, "HardwareVersion")
			// BMC MAC / DNS from EthernetInterfaces under manager
			if ethLink := oDataLink(mgrData, "EthernetInterfaces"); ethLink != "" {
				ethItems, _ := redfishGetCollection(ctx, client, base, ethLink, username, password)
				for _, e := range ethItems {
					if mac := strVal(e, "MACAddress"); mac != nil && result.BMCMACAddress == nil {
						result.BMCMACAddress = mac
					}
					if dns := strVal(e, "FQDN"); dns != nil && result.BMCDNSName == nil {
						result.BMCDNSName = dns
					}
					if dns := strVal(e, "HostName"); dns != nil && result.BMCDNSName == nil {
						result.BMCDNSName = dns
					}
				}
			}
			// License info (Dell iDRAC specific)
			if lic, ok := mgrData["Oem"].(map[string]any); ok {
				if dell, ok := lic["Dell"].(map[string]any); ok {
					if idrac, ok := dell["iDRAC"].(map[string]any); ok {
						if licInfo, ok := idrac["License"].(map[string]any); ok {
							result.BMCLicense = strVal(licInfo, "LicenseDescription")
						}
					}
				}
			}
			addStep("BMC Info", "ok", strDeref(result.BMCFirmware, "unknown"))
		} else {
			addStep("BMC Info", "error", err.Error())
		}

		// ── Step 11: System Event Log ─────────────────────────────────────────
		selEntries := collectSEL(ctx, client, base, managerPath, username, password)
		if selEntries != nil {
			result.SELEntries = selEntries
			addStep("System Event Log", "ok", fmt.Sprintf("%d entries", len(selEntries)))
		} else {
			addStep("System Event Log", "skipped", "SEL not available")
		}

		// ── Step 12: Lifecycle / Audit Logs ───────────────────────────────────
		lcLogs := collectLifecycleLogs(ctx, client, base, managerPath, username, password)
		if lcLogs != nil {
			result.LifecycleLogs = lcLogs
			addStep("Lifecycle Logs", "ok", fmt.Sprintf("%d entries", len(lcLogs)))
		} else {
			addStep("Lifecycle Logs", "skipped", "not available")
		}
	} else {
		addStep("BMC Info", "skipped", "manager path not found")
		addStep("System Event Log", "skipped", "manager path not found")
		addStep("Lifecycle Logs", "skipped", "manager path not found")
	}

	return result, nil
}

// parseDrive extracts DriveInfo from a Redfish Drive resource map.
func parseDrive(d map[string]any) models.DriveInfo {
	capBytes := floatVal(d, "CapacityBytes")
	capGB := 0.0
	if capBytes != nil {
		capGB = *capBytes / (1024 * 1024 * 1024)
	}
	return models.DriveInfo{
		Name:               strDeref(strVal(d, "Name"), strDeref(strVal(d, "Id"), "Drive")),
		Model:              strDeref(strVal(d, "Model"), ""),
		Manufacturer:       strDeref(strVal(d, "Manufacturer"), ""),
		SerialNumber:       strDeref(strVal(d, "SerialNumber"), ""),
		CapacityGB:         capGB,
		MediaType:          strDeref(strVal(d, "MediaType"), ""),
		BusProtocol:        strDeref(strVal(d, "Protocol"), ""),
		TemperatureCelsius: floatVal(d, "TemperatureCelsius"),
		WritePolicy:        strDeref(strVal(d, "WriteCachePolicy"), ""),
		ReadPolicy:         strDeref(strVal(d, "ReadCachePolicy"), ""),
		Health:             healthVal(d),
		Status:             stateVal(d),
	}
}

// ── Log collectors ────────────────────────────────────────────────────────────

func collectSEL(ctx context.Context, client *http.Client, base, managerPath, user, pass string) []models.SELEntry {
	if managerPath == "" {
		return nil
	}
	for _, path := range []string{
		managerPath + "/LogServices/Sel/Entries",
		managerPath + "/LogServices/SEL/Entries",
		managerPath + "/LogServices/Log1/Entries",
	} {
		data, _, err := redfishGet(ctx, client, base, path, user, pass)
		if err != nil {
			continue
		}
		members, _ := data["Members"].([]any)
		var entries []models.SELEntry
		for _, m := range members {
			if entry, ok := m.(map[string]any); ok {
				entries = append(entries, models.SELEntry{
					ID:       strDeref(strVal(entry, "Id"), ""),
					Severity: strDeref(strVal(entry, "Severity"), "Info"),
					Message:  strDeref(strVal(entry, "Message"), strDeref(strVal(entry, "Name"), "")),
					Created:  strDeref(strVal(entry, "Created"), ""),
				})
			}
		}
		return entries
	}
	return nil
}

func collectLifecycleLogs(ctx context.Context, client *http.Client, base, managerPath, user, pass string) []models.LifecycleLogEntry {
	if managerPath == "" {
		return nil
	}
	for _, path := range []string{
		managerPath + "/LogServices/Lclog/Entries",
		managerPath + "/LogServices/LCLog/Entries",
		managerPath + "/LogServices/Audit/Entries",
		managerPath + "/LogServices/AuditLog/Entries",
	} {
		data, _, err := redfishGet(ctx, client, base, path, user, pass)
		if err != nil {
			continue
		}
		members, _ := data["Members"].([]any)
		var entries []models.LifecycleLogEntry
		for _, m := range members {
			if entry, ok := m.(map[string]any); ok {
				category := ""
				if oem, ok := entry["Oem"].(map[string]any); ok {
					if dell, ok := oem["Dell"].(map[string]any); ok {
						category = strDeref(strVal(dell, "Category"), "")
					}
				}
				entries = append(entries, models.LifecycleLogEntry{
					ID:       strDeref(strVal(entry, "Id"), ""),
					Severity: strDeref(strVal(entry, "Severity"), "Info"),
					Message:  strDeref(strVal(entry, "Message"), strDeref(strVal(entry, "Name"), "")),
					Category: category,
					Created:  strDeref(strVal(entry, "Created"), ""),
				})
			}
		}
		return entries
	}
	return nil
}

// ── Hardware Control Operations ───────────────────────────────────────────────

// PowerControl sends a Redfish ResetType action to the system.
// Valid ResetType values: On, ForceOff, GracefulShutdown, ForceRestart, PowerCycle, GracefulRestart
func (s *RedfishService) PowerControl(ctx context.Context, bmcIP string, cred *models.DeviceCredential, password, resetType string) models.PowerControlResult {
	timeout := time.Duration(cred.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := buildHTTPClient(cred, timeout)
	base := baseURL(bmcIP, cred)
	username := ""
	if cred.Username != nil {
		username = *cred.Username
	}

	systemPath, err := discoverSystemPath(ctx, client, base, username, password)
	if err != nil {
		return models.PowerControlResult{Success: false, Message: fmt.Sprintf("System discovery failed: %v", err), ResetType: resetType}
	}

	// Discover the Reset action target URL
	sysData, _, err := redfishGet(ctx, client, base, systemPath, username, password)
	if err != nil {
		return models.PowerControlResult{Success: false, Message: fmt.Sprintf("Failed to read system resource: %v", err), ResetType: resetType}
	}

	actionTarget := systemPath + "/Actions/ComputerSystem.Reset"
	if actions, ok := sysData["Actions"].(map[string]any); ok {
		if reset, ok := actions["#ComputerSystem.Reset"].(map[string]any); ok {
			if target, ok := reset["target"].(string); ok && target != "" {
				actionTarget = target
			}
		}
	}

	_, statusCode, err := redfishPost(ctx, client, base, actionTarget, username, password, map[string]any{
		"ResetType": resetType,
	})
	if err != nil {
		code := classifyError(err, statusCode)
		return models.PowerControlResult{
			Success:   false,
			Message:   fmt.Sprintf("Power control failed (HTTP %d, %s): %v", statusCode, code, err),
			ResetType: resetType,
		}
	}
	return models.PowerControlResult{
		Success:   true,
		Message:   fmt.Sprintf("Power action '%s' sent successfully", resetType),
		ResetType: resetType,
	}
}

// BootControl sets the one-time or persistent boot override on the system.
// Valid targets: Pxe, Cd, Hdd, BiosSetup, None
func (s *RedfishService) BootControl(ctx context.Context, bmcIP string, cred *models.DeviceCredential, password, target string, once bool) models.BootControlResult {
	timeout := time.Duration(cred.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := buildHTTPClient(cred, timeout)
	base := baseURL(bmcIP, cred)
	username := ""
	if cred.Username != nil {
		username = *cred.Username
	}

	systemPath, err := discoverSystemPath(ctx, client, base, username, password)
	if err != nil {
		return models.BootControlResult{Success: false, Message: fmt.Sprintf("System discovery failed: %v", err), Target: target}
	}

	enabled := "Once"
	if !once {
		enabled = "Continuous"
	}

	_, statusCode, err := redfishPatch(ctx, client, base, systemPath, username, password, map[string]any{
		"Boot": map[string]any{
			"BootSourceOverrideTarget":  target,
			"BootSourceOverrideEnabled": enabled,
		},
	})
	if err != nil {
		return models.BootControlResult{
			Success: false,
			Message: fmt.Sprintf("Boot control failed (HTTP %d): %v", statusCode, err),
			Target:  target,
		}
	}
	return models.BootControlResult{
		Success: true,
		Message: fmt.Sprintf("Boot override '%s' (%s) set successfully", target, enabled),
		Target:  target,
	}
}

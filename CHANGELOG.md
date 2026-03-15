# Changelog

## [1.1.0] - 2026-03-14

### Added

- BMC connection testing via `POST /devices/:id/test-connection` (Redfish / IPMI)
- On-demand device sync via `POST /devices/:id/sync` — pulls system info, sensors, power, CPU, memory, and event logs from BMC
- Connection settings per device credential: port, HTTP scheme, SSL verify, polling interval, timeout, retry attempts
## [1.0.0] - 2026-03-15
### Added
- Complete hybrid infrastructure monitoring platform
- Device management for all major server vendors:
  Dell iDRAC, HPE iLO, Supermicro, Lenovo XCC, Cisco CIMC,
  Huawei iBMC, Fujitsu iRMC, ASUS ASMB, Gigabyte BMC,
  Ericsson BMC, IEIT BMC, APC/Eaton UPS, Proxmox, Linux, Windows
- IPMI collector for out-of-band monitoring
- Redfish collector for modern BMC APIs (Dell iDRAC9, HPE iLO5)
- SNMP collector for UPS devices
- Proxmox collector for virtualization monitoring
- Alert engine with Prometheus and Alertmanager
- Multi-channel notifications: Email, Telegram, Slack
- React TypeScript dashboard with real-time metrics
- Grafana integration for advanced analytics
- JWT authentication with RBAC (Admin, Operator, Viewer)
- Maintenance windows with alert suppression
- Audit logging for all operations
- Docker Compose deployment (Windows and Ubuntu)
- Ubuntu 24.04 native installation script
- GitHub Actions CI/CD pipeline

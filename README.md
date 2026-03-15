# InfraSense

InfraSense is a complete hybrid infrastructure monitoring platform designed for modern data centers and hybrid cloud environments. It provides deep visibility into server hardware, virtualization platforms, and power infrastructure.

## Features

- **Multi-Vendor Support**: Dell, HPE, Supermicro, Lenovo, Cisco, Huawei, Fujitsu, ASUS, Gigabyte, and more.
- **Protocol Collectors**: IPMI, Redfish, SNMP, and Proxmox API.
- **Real-time Dashboard**: Built with React and TypeScript for high-performance metrics visualization.
- **Alert Engine**: Powered by Prometheus and Alertmanager with multi-channel notifications (Email, Telegram, Slack).
- **Security**: JWT-based authentication with Role-Based Access Control (RBAC).
- **Maintenance Planning**: Scheduled maintenance windows with intelligent alert suppression.
- **Audit Logging**: Comprehensive tracking of all system operations.

## Supported Devices

| Vendor | Platform | Protocols |
|--------|----------|-----------|
| Dell | iDRAC 7/8/9 | IPMI, Redfish |
| HPE | iLO 4/5 | IPMI, Redfish |
| Proxmox | VE 7.x/8.x | REST API |
| Generic | IPMI 2.0 | IPMI |
| UPS | APC, Eaton | SNMP |
| OS | Linux, Windows | Node Exporter, WMI |

## Quick Install (Ubuntu 24.04)

Get up and running on a fresh Ubuntu server with these 5 commands:

```bash
git clone https://github.com/tanjil-al-mahmud/infrasense.git
cd infrasense
chmod +x scripts/install-ubuntu.sh
sudo ./scripts/install-ubuntu.sh
# Follow the on-screen instructions
```

## Quick Install (Windows Docker Desktop)

Deploy locally on Windows in 4 simple steps:

```powershell
git clone https://github.com/tanjil-al-mahmud/infrasense.git
cd infrasense
cp .env.example .env
docker-compose up -d
```

## Documentation

- [Quick Start Guide](docs/QUICK-START.md)
- [Architecture Overview](docs/ARCHITECTURE.md)
- [API Reference](docs/API_REFERENCE.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

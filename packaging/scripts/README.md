# InfraSense Installation Scripts

This directory contains scripts for building and installing InfraSense packages.

## Scripts

### install.sh

Automated installation script that detects the operating system and installs the appropriate InfraSense package.

**Usage:**
```bash
# Download and run
curl -fsSL https://releases.infrasense.io/install.sh | sudo bash

# Or download first, then run
wget https://releases.infrasense.io/install.sh
chmod +x install.sh
sudo ./install.sh
```

**What it does:**
1. Checks if running as root
2. Detects operating system and version
3. Validates required ports are available (80, 443, 5432, 8428)
4. Installs dependencies (postgresql-client, systemd, wget)
5. Downloads appropriate package (.deb or .rpm)
6. Installs the package
7. Displays success message with access URL and credentials

**Supported Operating Systems:**
- Ubuntu 22.04 and later
- Debian 12 and later
- RHEL 8 and later
- Rocky Linux 8 and later
- AlmaLinux 8 and later
- CentOS Stream 8 and later

For unsupported operating systems, the script displays Docker installation instructions.

**Requirements:**
- Root privileges (sudo)
- Internet connection
- Ports 80, 443, 5432, 8428 available

### build_deb.sh

Builds the .deb package for Debian/Ubuntu systems.

**Usage:**
```bash
cd infrasense/packaging/scripts
./build_deb.sh
```

### build_rpm.sh

Builds the .rpm package natively on RHEL/Rocky/AlmaLinux systems.

**Usage:**
```bash
cd infrasense/packaging/scripts
./build_rpm.sh
```

### build_rpm_docker.sh

Builds the .rpm package using Docker (works on any system with Docker).

**Usage:**
```bash
cd infrasense/packaging/scripts
./build_rpm_docker.sh
```

## Testing

Validation tests are available in the `../tests/` directory:

```bash
# Test install.sh script structure (Linux)
cd ../tests
./test_install_script.sh

# Test install.sh script structure (Windows)
cd ../tests
powershell -ExecutionPolicy Bypass -File test_install_script.ps1
```

## Error Handling

The install.sh script handles various error scenarios:

- **Not running as root**: Exits with message to use sudo
- **Cannot detect OS**: Exits if /etc/os-release not found
- **Port already in use**: Exits with message about which port is occupied
- **Package download failure**: Exits with network error message
- **Package installation failure**: Exits with installation error
- **Unsupported OS**: Shows Docker installation instructions

## Post-Installation

After successful installation:

1. Access the dashboard at `http://<server-ip>`
2. Log in with default admin credentials (check `/var/log/infrasense/install.log`)
3. Change the default admin password immediately
4. Register your first device
5. Configure alert rules
6. Set up notification channels

## Documentation

- Full documentation: https://docs.infrasense.io
- Installation guide: https://docs.infrasense.io/installation
- Support: https://github.com/infrasense/infrasense/issues

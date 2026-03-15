# InfraSense Packaging

This directory contains the packaging infrastructure for the InfraSense Platform.

## Directory Structure

```
packaging/
├── deb/                        # Debian package tree
│   ├── DEBIAN/
│   │   ├── control             # Package metadata
│   │   ├── postinst            # Post-install script (creates user, initializes DB, starts services)
│   │   └── prerm               # Pre-removal script (stops and disables services)
│   ├── etc/
│   │   ├── infrasense/         # Configuration files (/etc/infrasense/)
│   │   │   ├── config.yml      # Main configuration file
│   │   │   └── *.env           # Per-service environment variable files
│   │   └── systemd/system/     # Systemd service units
│   └── usr/
│       ├── local/bin/          # Service binaries (/usr/local/bin/)
│       └── local/share/infrasense/migrations/  # SQL migration files
├── scripts/
│   └── build_deb.sh            # Build script for the .deb package
└── tests/
    ├── suite.sh                # Test suite (runs inside Docker containers)
    ├── run_tests.sh            # Test runner (spins up Docker containers)
    └── test_deb_install.sh     # Standalone Ubuntu 22.04 install test
```

## Prerequisites

- **dpkg-deb** — required to build the package (available on Debian/Ubuntu: `apt-get install dpkg`)
- **Docker** — required to run the installation tests
- **Go 1.21+** — required to compile the actual service binaries

## Building the .deb Package

### 1. Compile the Go binaries (production builds)

Before building for production, compile the Go binaries and place them in `packaging/deb/usr/local/bin/`:

```bash
# From the repo root
GOOS=linux GOARCH=amd64 go build -o packaging/deb/usr/local/bin/infrasense-api \
    ./backend/cmd/server

GOOS=linux GOARCH=amd64 go build -o packaging/deb/usr/local/bin/infrasense-ipmi-collector \
    ./collectors/ipmi-collector/cmd

GOOS=linux GOARCH=amd64 go build -o packaging/deb/usr/local/bin/infrasense-redfish-collector \
    ./collectors/redfish-collector/cmd

GOOS=linux GOARCH=amd64 go build -o packaging/deb/usr/local/bin/infrasense-snmp-collector \
    ./collectors/snmp-collector/cmd

GOOS=linux GOARCH=amd64 go build -o packaging/deb/usr/local/bin/infrasense-notification-service \
    ./notification-service/cmd

GOOS=linux GOARCH=amd64 go build -o packaging/deb/usr/local/bin/infrasense-proxmox-collector \
    ./collectors/proxmox-collector/cmd
```

### 2. Run the build script

```bash
cd infrasense/packaging/scripts
chmod +x build_deb.sh
./build_deb.sh
```

The package will be written to `infrasense/dist/infrasense_1.0.0_amd64.deb`.

To build a specific version:

```bash
./build_deb.sh --version 1.2.3
```

## Running the Installation Tests

The test suite installs the package inside Docker containers and verifies the expected outcomes.

### Run all tests (Ubuntu 22.04 + Rocky Linux 8)

```bash
cd infrasense/packaging/tests
chmod +x run_tests.sh
./run_tests.sh
```

### Run only the .deb tests (Ubuntu 22.04)

```bash
./run_tests.sh --deb-only
```

### Standalone Ubuntu 22.04 test

```bash
./test_deb_install.sh
# or pass a specific package path:
./test_deb_install.sh /path/to/infrasense_1.0.0_amd64.deb
```

The test runner looks for the built package in `infrasense/dist/infrasense_*.deb`.

## Building the .rpm Package

### 1. Compile the Go binaries (production builds)

Before building for production, compile the Go binaries (same as for .deb package above).

### 2. Run the build script

#### Option A: Build on a RHEL/Rocky/AlmaLinux system

```bash
cd infrasense/packaging/scripts
chmod +x build_rpm.sh
./build_rpm.sh
```

#### Option B: Build using Docker (recommended for cross-platform builds)

```bash
cd infrasense/packaging/scripts
chmod +x build_rpm_docker.sh
./build_rpm_docker.sh
```

The package will be written to `infrasense/dist/infrasense-1.0.0-1.x86_64.rpm`.

To build a specific version:

```bash
./build_rpm.sh --version 1.2.3
# or
./build_rpm_docker.sh --version 1.2.3
```

## Running the RPM Installation Tests

### Run only the .rpm tests (Rocky Linux 8)

```bash
cd infrasense/packaging/tests
chmod +x test_rpm_install.sh
./test_rpm_install.sh
```

### Standalone Rocky Linux 8 test

```bash
./test_rpm_install.sh /path/to/infrasense-1.0.0-1.x86_64.rpm
```

The test runner looks for the built package in `infrasense/dist/infrasense-*.rpm`.

## Supported Platforms

| Platform       | Package format | Tested via              |
|----------------|---------------|-------------------------|
| Ubuntu 22.04+  | .deb          | `ubuntu:22.04` Docker   |
| Debian 12+     | .deb          | `debian:12` Docker      |
| RHEL 8+        | .rpm          | `rockylinux:8` Docker   |
| Rocky Linux 8+ | .rpm          | `rockylinux:8` Docker   |
| AlmaLinux 8+   | .rpm          | `almalinux:8` Docker    |

## What the Post-Install Script Does

1. Creates the `infrasense` system user and group
2. Creates `/var/lib/infrasense`, `/var/log/infrasense`, `/etc/infrasense` with correct ownership
3. Creates the PostgreSQL `infrasense` role and database (if PostgreSQL is available)
4. Runs database migrations from `/usr/local/share/infrasense/migrations/`
5. Creates a default `admin` user with a randomly generated password
6. Enables and starts all InfraSense systemd services
7. Prints the admin credentials to the console

## What the Pre-Removal Script Does

1. Stops all InfraSense systemd services
2. Disables all InfraSense systemd services

> **Note:** The pre-removal script does NOT drop the PostgreSQL database or delete
> `/etc/infrasense/` to preserve configuration and data across reinstalls.

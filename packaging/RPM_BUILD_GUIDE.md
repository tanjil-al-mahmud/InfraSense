# RPM Package Build and Test Guide

This guide provides step-by-step instructions for building and testing the InfraSense Platform RPM package.

## Prerequisites

### For Building

**Option A: Native Build (on RHEL/Rocky/AlmaLinux)**
- `rpmbuild` tool: `dnf install rpm-build rpmdevtools`
- Go 1.21+ (for compiling binaries)

**Option B: Docker Build (recommended for cross-platform)**
- Docker installed and running
- No other dependencies required

### For Testing

- Docker installed and running
- Built RPM package in `infrasense/dist/` directory

## Building the RPM Package

### Step 1: Compile Go Binaries (Optional for Production)

For production builds, compile the actual Go binaries:

```bash
# From the repository root
cd infrasense

# Backend API server
GOOS=linux GOARCH=amd64 go build -o dist/bin/infrasense-api \
    ./backend/cmd/server

# IPMI Collector
GOOS=linux GOARCH=amd64 go build -o dist/bin/infrasense-ipmi-collector \
    ./collectors/ipmi-collector/cmd

# Redfish Collector
GOOS=linux GOARCH=amd64 go build -o dist/bin/infrasense-redfish-collector \
    ./collectors/redfish-collector/cmd

# SNMP Collector
GOOS=linux GOARCH=amd64 go build -o dist/bin/infrasense-snmp-collector \
    ./collectors/snmp-collector/cmd

# Proxmox Collector
GOOS=linux GOARCH=amd64 go build -o dist/bin/infrasense-proxmox-collector \
    ./collectors/proxmox-collector/cmd

# Notification Service
GOOS=linux GOARCH=amd64 go build -o dist/bin/infrasense-notification-service \
    ./notification-service/cmd
```

**Note:** If you skip this step, the build scripts will create stub binaries for structural testing.

### Step 2: Build the RPM Package

#### Option A: Native Build (on RHEL/Rocky/AlmaLinux)

```bash
cd infrasense/packaging/scripts
chmod +x build_rpm.sh
./build_rpm.sh
```

#### Option B: Docker Build (recommended)

```bash
cd infrasense/packaging/scripts
chmod +x build_rpm_docker.sh
./build_rpm_docker.sh
```

The package will be created at: `infrasense/dist/infrasense-1.0.0-1.x86_64.rpm`

To build a specific version:

```bash
./build_rpm.sh --version 1.2.3
# or
./build_rpm_docker.sh --version 1.2.3
```

### Step 3: Verify the Package

```bash
# View package information
rpm -qip infrasense/dist/infrasense-1.0.0-1.x86_64.rpm

# View package contents
rpm -qlp infrasense/dist/infrasense-1.0.0-1.x86_64.rpm
```

## Testing the RPM Package

### Test on Rocky Linux 8

```bash
cd infrasense/packaging/tests
chmod +x test_rpm_install.sh
./test_rpm_install.sh
```

Or specify the package path:

```bash
./test_rpm_install.sh /path/to/infrasense-1.0.0-1.x86_64.rpm
```

### Test on AlmaLinux 8

```bash
cd infrasense/packaging/tests
chmod +x test_rpm_almalinux.sh
./test_rpm_almalinux.sh
```

### Test on Both Platforms

```bash
cd infrasense/packaging/tests
chmod +x test_all_rpm.sh
./test_all_rpm.sh
```

This will run tests on both Rocky Linux 8 and AlmaLinux 8.

## What the Tests Verify

The test suite (`suite.sh`) verifies the following requirements:

1. **Binary Installation** (Requirement 26.3)
   - All binaries installed to `/usr/local/bin/`
   - Binaries are executable

2. **Configuration Files** (Requirement 26.4)
   - Configuration files installed to `/etc/infrasense/`
   - Correct file permissions

3. **Systemd Service Units** (Requirement 26.5)
   - Service units installed to `/usr/lib/systemd/system/`
   - Services can be enabled and started

4. **System User and Group** (Requirement 26.6)
   - `infrasense` system user created
   - `infrasense` system group created

5. **Database Initialization** (Requirement 26.7)
   - PostgreSQL database created (if PostgreSQL available)
   - Database migrations applied

6. **Service Startup** (Requirement 26.8)
   - All services enabled
   - All services started successfully

7. **Package Removal** (Requirement 26.9)
   - Services stopped on package removal
   - Services disabled on package removal

## Troubleshooting

### Build Failures

**Error: rpmbuild not found**
```bash
# Install rpmbuild
dnf install rpm-build rpmdevtools
```

**Error: Docker not found**
```bash
# Install Docker
# See: https://docs.docker.com/engine/install/
```

### Test Failures

**Error: Package not found**
```bash
# Build the package first
cd infrasense/packaging/scripts
./build_rpm_docker.sh
```

**Error: Docker container fails to start**
```bash
# Check Docker is running
docker ps

# Check Docker has sufficient resources
docker info
```

**Error: PostgreSQL initialization fails**
- This is expected in test containers without PostgreSQL installed
- The package will log a warning and continue
- For production, ensure PostgreSQL is installed before package installation

## Manual Installation Testing

To manually test the package installation:

```bash
# Start a Rocky Linux 8 container
docker run -it --privileged rockylinux:8 bash

# Inside the container:
# 1. Copy the package (mount it as a volume when starting the container)
# 2. Install dependencies
dnf install -y postgresql

# 3. Install the package
dnf install -y /path/to/infrasense-1.0.0-1.x86_64.rpm

# 4. Check service status
systemctl status infrasense-api
systemctl status infrasense-ipmi-collector
# ... etc

# 5. Check logs
journalctl -u infrasense-api
```

## Package Structure

The RPM package includes:

```
/usr/local/bin/
├── infrasense-api
├── infrasense-ipmi-collector
├── infrasense-redfish-collector
├── infrasense-snmp-collector
├── infrasense-proxmox-collector
└── infrasense-notification-service

/etc/infrasense/
└── config.yml

/usr/lib/systemd/system/
├── infrasense-api.service
├── infrasense-ipmi-collector.service
├── infrasense-redfish-collector.service
├── infrasense-snmp-collector.service
├── infrasense-proxmox-collector.service
├── infrasense-notification-service.service
└── infrasense-frontend.service

/usr/local/share/infrasense/migrations/
└── *.sql (database migration files)

/var/lib/infrasense/
└── (runtime data directory)

/var/log/infrasense/
└── (log directory)
```

## Post-Installation

After successful installation, the package will:

1. Create the `infrasense` system user and group
2. Initialize the PostgreSQL database (if available)
3. Run database migrations
4. Create a default admin user with a random password
5. Start all InfraSense services
6. Display the admin credentials

**Important:** Save the admin credentials displayed during installation. They are also stored in `/etc/infrasense/.admin_credentials`.

## Next Steps

After successful package installation and testing:

1. Configure the platform via `/etc/infrasense/config.yml`
2. Access the web interface at `http://localhost`
3. Login with the admin credentials
4. Register devices and configure monitoring

## References

- Requirements: See `.kiro/specs/infrasense-platform/requirements.md` (Requirement 26)
- Tasks: See `.kiro/specs/infrasense-platform/tasks.md` (Task 6.10)
- RPM Spec File: `infrasense/packaging/rpm/infrasense.spec`

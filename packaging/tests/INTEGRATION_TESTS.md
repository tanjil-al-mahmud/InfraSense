# InfraSense Package Installation Integration Tests

## Overview

This document describes the integration tests for InfraSense package installation, covering both `.deb` (Debian/Ubuntu) and `.rpm` (RHEL/Rocky/AlmaLinux) packages.

**Task**: 6.12 Write integration tests for package installation  
**Requirements**: 25.8, 26.8

## Test Coverage

The integration test suite (`suite.sh`) verifies the following aspects of package installation:

### Test 1: Pre-installation Sanity
- Verifies that InfraSense binaries are not present before installation
- Ensures clean test environment

### Test 2: Package Installation
- Installs the package using the appropriate package manager (apt/dnf)
- Verifies installation completes without errors
- **Validates**: Requirements 25.8, 26.8

### Test 3: Binary Installation
- Verifies all service binaries are installed to `/usr/local/bin/`:
  - `infrasense-api`
  - `infrasense-ipmi-collector`
  - `infrasense-redfish-collector`
  - `infrasense-snmp-collector`
  - `infrasense-notification-service`
- **Validates**: Requirements 25.3, 26.3

### Test 4: Configuration Files
- Verifies configuration directory `/etc/infrasense/` exists
- Verifies main configuration file `/etc/infrasense/config.yml` exists
- **Validates**: Requirements 25.4, 26.4

### Test 5: Systemd Service Units
- Verifies all systemd service units are installed to `/etc/systemd/system/`:
  - `infrasense-api.service`
  - `infrasense-ipmi-collector.service`
  - `infrasense-redfish-collector.service`
  - `infrasense-snmp-collector.service`
  - `infrasense-notification-service.service`
- **Validates**: Requirements 25.5, 26.5

### Test 6: System User and Group
- Verifies `infrasense` system user is created
- Verifies `infrasense` system group is created
- **Validates**: Requirements 25.6, 26.6

### Test 7: Services Start Successfully
- Verifies all InfraSense services start successfully after installation
- Checks service status using `systemctl is-active`
- **Validates**: Requirements 25.8, 26.8 (critical requirement)

### Test 8: Database Schema Initialized
- Verifies PostgreSQL database schema is initialized
- Checks for presence of core tables:
  - `devices`
  - `users`
  - `alert_rules`
  - `maintenance_windows`
  - `audit_logs`
- **Validates**: Requirements 25.7, 26.7

### Test 9: Default Admin User Created
- Verifies default `admin` user exists in the database
- **Validates**: Requirements 25.7, 26.7

### Test 10: API Server Accessible
- Verifies API server is listening on port 80
- Verifies `/health` endpoint returns HTTP 200
- Uses 30-second timeout with retry logic
- **Validates**: Requirements 25.8, 26.8 (critical requirement)

### Test 11: API Authentication
- Tests admin login with default credentials
- Verifies JWT token is returned in response
- **Validates**: Requirements 25.8, 26.8

## Test Infrastructure

### Files

- **`suite.sh`**: Main test suite executed inside Docker containers
- **`run_tests.sh`**: Test orchestrator that spins up Docker containers and runs the suite
- **`test_deb_install.sh`**: Standalone test for Ubuntu 22.04 .deb installation
- **`test_rpm_install.sh`**: Standalone test for Rocky Linux 8 .rpm installation
- **`README.md`**: Quick reference guide for running tests

### Test Execution Flow

```
run_tests.sh
    ├─> Locates .deb and .rpm packages in infrasense/dist/
    ├─> Starts Docker container (ubuntu:22.04 or rockylinux:8)
    ├─> Mounts package to /tmp/infrasense_pkg
    ├─> Mounts suite.sh to /tmp/suite.sh
    ├─> Injects INSTALL_CMD environment variable
    ├─> Executes suite.sh inside container
    ├─> Collects test results (PASS/FAIL counts)
    └─> Removes container and reports summary
```

## Prerequisites

### System Requirements
- **Docker**: Required for running tests in isolated containers
- **Bash**: Required for test scripts (Linux/macOS/WSL)
- **Built Packages**: 
  - `.deb` package in `infrasense/dist/infrasense_*.deb`
  - `.rpm` package in `infrasense/dist/infrasense-*.rpm`

### Building Packages

Before running tests, build the packages:

```bash
# Build .deb package
cd infrasense/packaging/scripts
./build_deb.sh

# Build .rpm package (using Docker)
./build_rpm_docker.sh
```

## Running the Tests

### Run All Tests (Both .deb and .rpm)

```bash
cd infrasense/packaging/tests
./run_tests.sh
```

Expected output:
```
[INFO] ========================================================
[INFO] Suite: deb-ubuntu-22.04  (image: ubuntu:22.04)
[INFO] ========================================================
[PASS] Package installed without errors
[PASS] File exists: /usr/local/bin/infrasense-api
[PASS] Service active: infrasense-api
[PASS] DB table exists: devices
[PASS] API server accessible on port 80
...
========================================================
  Results: 25 passed, 0 failed
========================================================
```

### Run Only .deb Tests (Ubuntu 22.04)

```bash
./run_tests.sh --deb-only
```

Or use the standalone script:
```bash
./test_deb_install.sh
```

### Run Only .rpm Tests (Rocky Linux 8)

```bash
./run_tests.sh --rpm-only
```

Or use the standalone script:
```bash
./test_rpm_install.sh
```

### Custom Package Location

```bash
# Specify custom package directory
./run_tests.sh --package-dir /path/to/dist

# Or pass package directly to standalone scripts
./test_deb_install.sh /path/to/infrasense_1.0.0_amd64.deb
./test_rpm_install.sh /path/to/infrasense-1.0.0.x86_64.rpm
```

## Test Output Format

Each test prints `[PASS]` or `[FAIL]` followed by a description:

```
[PASS] Package installed without errors
[PASS] File exists: /usr/local/bin/infrasense-api
[PASS] Service active: infrasense-api
[FAIL] DB table missing: devices
```

The final summary shows total pass/fail counts:
```
========================================================
  Results: 23 passed, 2 failed
========================================================
```

Exit codes:
- `0`: All tests passed
- `1`: One or more tests failed

## Test Assertions

The test suite provides several assertion helpers:

| Function | Description |
|----------|-------------|
| `assert_file_exists <path>` | Checks a file or directory exists |
| `assert_service_active <name>` | Checks a systemd service is active |
| `assert_port_open <host> <port> [desc]` | Waits up to 30s for a TCP port |
| `assert_http_ok <url> [desc]` | Waits up to 30s for HTTP 200 |
| `assert_db_table_exists <table>` | Checks a PostgreSQL table exists |
| `assert_db_user_exists <username>` | Checks a user row exists in `users` table |
| `assert_pass <desc> <cmd...>` | Passes if command exits 0 |
| `assert_fail <desc> <cmd...>` | Passes if command exits non-0 |

## Troubleshooting

### Docker Not Available

If Docker is not installed or not running:
```
ERROR: docker is not installed or not in PATH
ERROR: Docker daemon is not running
```

**Solution**: Install Docker and ensure the daemon is running:
```bash
# Ubuntu/Debian
sudo apt-get install docker.io
sudo systemctl start docker

# RHEL/Rocky/AlmaLinux
sudo dnf install docker
sudo systemctl start docker
```

### Package Not Found

If the package file is missing:
```
[FAIL] deb-ubuntu-22.04: package file not found: infrasense/dist/infrasense_MISSING.deb
```

**Solution**: Build the package first:
```bash
cd infrasense/packaging/scripts
./build_deb.sh
```

### Service Fails to Start

If a service fails to start during tests:
```
[FAIL] Service not active: infrasense-api
```

**Debugging**:
1. Check the container logs:
   ```bash
   docker ps -a  # Find container ID
   docker logs <container-id>
   ```

2. Inspect the service status inside the container:
   ```bash
   docker exec <container-id> systemctl status infrasense-api
   docker exec <container-id> journalctl -u infrasense-api
   ```

3. Check the post-install script logs:
   ```bash
   docker exec <container-id> cat /var/log/infrasense/postinst.log
   ```

### Database Connection Issues

If database tests fail:
```
[FAIL] DB table missing: devices
```

**Common causes**:
- PostgreSQL not installed in the container
- Database initialization failed during post-install
- Incorrect database credentials

**Debugging**:
```bash
docker exec <container-id> su - infrasense -c "psql -U infrasense -d infrasense -c '\dt'"
docker exec <container-id> cat /var/log/infrasense/postinst.log
```

### Port Already in Use

If port 80 is already in use on the host:
```
[FAIL] Port not open after 30s: API server (127.0.0.1:80)
```

**Solution**: Stop any services using port 80, or modify the test to use a different port.

## Continuous Integration

These tests are designed to run in CI/CD pipelines:

### GitHub Actions Example

```yaml
name: Package Integration Tests

on: [push, pull_request]

jobs:
  test-packages:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build .deb package
        run: |
          cd infrasense/packaging/scripts
          ./build_deb.sh
      
      - name: Build .rpm package
        run: |
          cd infrasense/packaging/scripts
          ./build_rpm_docker.sh
      
      - name: Run integration tests
        run: |
          cd infrasense/packaging/tests
          ./run_tests.sh
```

### GitLab CI Example

```yaml
test:packages:
  image: docker:latest
  services:
    - docker:dind
  script:
    - cd infrasense/packaging/scripts
    - ./build_deb.sh
    - ./build_rpm_docker.sh
    - cd ../tests
    - ./run_tests.sh
```

## Adding New Tests

To add new test cases to the suite:

1. Open `suite.sh`
2. Add a new test section:
   ```bash
   # ── Test 12: Your new test ────────────────────────────────────────────────
   info "--- Test 12: Your new test description ---"
   assert_file_exists "/path/to/file"
   assert_service_active "your-service"
   ```

3. Use the provided assertion helpers
4. Update this documentation with the new test description

## Test Maintenance

### When to Update Tests

Update the test suite when:
- New services are added to the platform
- New configuration files are introduced
- Database schema changes require new tables
- Installation process changes (new post-install steps)
- New requirements are added to the specification

### Test Review Checklist

- [ ] All binaries listed in requirements are tested
- [ ] All systemd services are verified to start
- [ ] All configuration files are checked for existence
- [ ] Database schema initialization is verified
- [ ] API endpoints are tested for accessibility
- [ ] Default credentials are tested
- [ ] Both .deb and .rpm packages are tested
- [ ] Tests run successfully in CI/CD pipeline

## Related Documentation

- **`README.md`**: Quick reference for running tests
- **`../README.md`**: Packaging overview and build instructions
- **`../scripts/README.md`**: Build script documentation
- **`.kiro/specs/infrasense-platform/requirements.md`**: Requirements 25.8, 26.8
- **`.kiro/specs/infrasense-platform/tasks.md`**: Task 6.12 details

## Summary

The integration test suite provides comprehensive verification of InfraSense package installation for both Debian-based and RHEL-based Linux distributions. The tests ensure that:

1. ✅ Packages install without errors
2. ✅ All binaries are correctly placed
3. ✅ Configuration files are installed
4. ✅ Systemd services are installed and start successfully
5. ✅ Database schema is initialized
6. ✅ Default admin user is created
7. ✅ API server is accessible and functional

These tests validate **Requirements 25.8 and 26.8**, ensuring that the InfraSense platform is production-ready after package installation.

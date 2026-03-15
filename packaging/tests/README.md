# InfraSense Package Installation Integration Tests

These tests verify that the `.deb` and `.rpm` packages install correctly and that
all post-installation steps complete successfully.

## What is tested

| # | Check | Requirement |
|---|-------|-------------|
| 1 | Binary absent before install | – |
| 2 | Package installs without errors | 25.8 / 26.8 |
| 3 | Binaries installed to `/usr/local/bin/` | 25.3 / 26.3 |
| 4 | Config files installed to `/etc/infrasense/` | 25.4 / 26.4 |
| 5 | Systemd service units installed | 25.5 / 26.5 |
| 6 | `infrasense` system user and group created | 25.6 / 26.6 |
| 7 | All services start successfully | 25.8 / 26.8 |
| 8 | Database schema initialized (core tables present) | 25.7 / 26.7 |
| 9 | Default `admin` user created | 25.7 / 26.7 |
| 10 | API server accessible on port 80 | 25.8 / 26.8 |
| 11 | Admin login returns a JWT token | 25.8 / 26.8 |

## Prerequisites

- Docker installed and running
- Built packages in `infrasense/dist/`:
  - `infrasense_<version>_amd64.deb`
  - `infrasense-<version>.x86_64.rpm`

## Running the tests

### Run both suites

```bash
cd infrasense/packaging/tests
./run_tests.sh
```

### Run only the .deb suite

```bash
./run_tests.sh --deb-only
# or
./test_deb_install.sh
```

### Run only the .rpm suite

```bash
./run_tests.sh --rpm-only
# or
./test_rpm_install.sh
```

### Custom package location

```bash
./run_tests.sh --package-dir /path/to/dist
# or pass the package directly
./test_deb_install.sh /path/to/infrasense_1.0.0_amd64.deb
./test_rpm_install.sh /path/to/infrasense-1.0.0.x86_64.rpm
```

## Output format

Each check prints `[PASS]` or `[FAIL]` followed by a description.
The final summary shows total pass/fail counts and the script exits with code `0`
(all passed) or `1` (one or more failures).

```
[PASS] Package installed without errors
[PASS] File exists: /usr/local/bin/infrasense-api
[FAIL] Service not active: infrasense-api
...
========================================================
  Results: 9 passed, 1 failed
========================================================
```

## How it works

1. `run_tests.sh` locates the package files and starts a Docker container for each
   target OS (`ubuntu:22.04` for deb, `rockylinux:8` for rpm).
2. The package is bind-mounted into the container at `/tmp/infrasense_pkg`.
3. `suite.sh` is executed inside the container. It installs the package using the
   appropriate package manager command and then runs all checks.
4. The container is removed after the suite completes.

## Adding new checks

Add new `assert_*` calls to `suite.sh`. The helper functions available are:

| Function | Description |
|----------|-------------|
| `assert_file_exists <path>` | Checks a file or directory exists |
| `assert_service_active <name>` | Checks a systemd service is active |
| `assert_port_open <host> <port>` | Waits up to 30 s for a TCP port |
| `assert_http_ok <url>` | Waits up to 30 s for HTTP 200 |
| `assert_db_table_exists <table>` | Checks a PostgreSQL table exists |
| `assert_db_user_exists <username>` | Checks a user row exists in the `users` table |
| `assert_pass <desc> <cmd...>` | Passes if command exits 0 |
| `assert_fail <desc> <cmd...>` | Passes if command exits non-0 |

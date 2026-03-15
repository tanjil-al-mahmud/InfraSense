Name:           infrasense
Version:        1.0.0
Release:        1%{?dist}
Summary:        InfraSense Platform - Infrastructure Monitoring Solution
License:        Proprietary
URL:            https://github.com/infrasense/infrasense-platform
BuildArch:      x86_64

Requires:       postgresql, systemd
Requires(pre):  shadow-utils
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd

%description
InfraSense is a comprehensive infrastructure monitoring platform that
collects hardware metrics via IPMI, Redfish, SNMP, and Proxmox protocols.
It provides real-time alerting, audit logging, and a web-based dashboard
for managing and monitoring physical and virtual infrastructure.

# -----------------------------------------------------------------------
# prep / build / check are intentionally empty for a binary-only package
# -----------------------------------------------------------------------
%prep
%build
%check

# -----------------------------------------------------------------------
# %install – copy binaries, configuration files, and systemd units
# -----------------------------------------------------------------------
%install
rm -rf %{buildroot}

# Binaries → /usr/local/bin/
install -d %{buildroot}/usr/local/bin
for bin in \
    infrasense-api \
    infrasense-ipmi-collector \
    infrasense-redfish-collector \
    infrasense-snmp-collector \
    infrasense-proxmox-collector \
    infrasense-notification-service; do
    install -m 0755 %{_builddir}/%{name}-%{version}/bin/${bin} \
        %{buildroot}/usr/local/bin/${bin}
done

# Configuration files → /etc/infrasense/
install -d %{buildroot}/etc/infrasense
install -m 0640 %{_builddir}/%{name}-%{version}/config/config.yml \
    %{buildroot}/etc/infrasense/config.yml

# Systemd service units → /usr/lib/systemd/system/
install -d %{buildroot}/usr/lib/systemd/system
for svc in \
    infrasense-api \
    infrasense-ipmi-collector \
    infrasense-redfish-collector \
    infrasense-snmp-collector \
    infrasense-proxmox-collector \
    infrasense-notification-service \
    infrasense-frontend; do
    install -m 0644 \
        %{_builddir}/%{name}-%{version}/systemd/${svc}.service \
        %{buildroot}/usr/lib/systemd/system/${svc}.service
done

# Database migrations → /usr/local/share/infrasense/migrations/
install -d %{buildroot}/usr/local/share/infrasense/migrations
cp -r %{_builddir}/%{name}-%{version}/migrations/. \
    %{buildroot}/usr/local/share/infrasense/migrations/

# Runtime directories (owned by the infrasense user)
install -d %{buildroot}/var/lib/infrasense
install -d %{buildroot}/var/log/infrasense

# -----------------------------------------------------------------------
# %files – declare all installed paths
# -----------------------------------------------------------------------
%files
%defattr(-,root,root,-)

# Binaries
/usr/local/bin/infrasense-api
/usr/local/bin/infrasense-ipmi-collector
/usr/local/bin/infrasense-redfish-collector
/usr/local/bin/infrasense-snmp-collector
/usr/local/bin/infrasense-proxmox-collector
/usr/local/bin/infrasense-notification-service

# Configuration (noreplace preserves local edits on upgrade)
%dir %attr(0750,infrasense,infrasense) /etc/infrasense
%config(noreplace) %attr(0640,infrasense,infrasense) /etc/infrasense/config.yml

# Systemd units
/usr/lib/systemd/system/infrasense-api.service
/usr/lib/systemd/system/infrasense-ipmi-collector.service
/usr/lib/systemd/system/infrasense-redfish-collector.service
/usr/lib/systemd/system/infrasense-snmp-collector.service
/usr/lib/systemd/system/infrasense-proxmox-collector.service
/usr/lib/systemd/system/infrasense-notification-service.service
/usr/lib/systemd/system/infrasense-frontend.service

# Migrations
%dir /usr/local/share/infrasense
%dir /usr/local/share/infrasense/migrations
/usr/local/share/infrasense/migrations/*

# Runtime directories
%dir %attr(0750,infrasense,infrasense) /var/lib/infrasense
%dir %attr(0750,infrasense,infrasense) /var/log/infrasense

# -----------------------------------------------------------------------
# %pre – create system user and group before files are installed
# -----------------------------------------------------------------------
%pre
if ! getent group infrasense > /dev/null 2>&1; then
    groupadd --system infrasense
fi

if ! getent passwd infrasense > /dev/null 2>&1; then
    useradd \
        --system \
        --no-create-home \
        --shell /sbin/nologin \
        --gid infrasense \
        infrasense
fi

exit 0

# -----------------------------------------------------------------------
# %post – post-installation: initialise DB schema and start services
# -----------------------------------------------------------------------
%post
%systemd_post \
    infrasense-api.service \
    infrasense-ipmi-collector.service \
    infrasense-redfish-collector.service \
    infrasense-snmp-collector.service \
    infrasense-proxmox-collector.service \
    infrasense-notification-service.service \
    infrasense-frontend.service

# ----------------------------------------------------------------
# 1. Ensure runtime directories exist with correct ownership
# ----------------------------------------------------------------
for dir in /var/lib/infrasense /var/log/infrasense /etc/infrasense; do
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
    fi
    chown infrasense:infrasense "$dir"
    chmod 750 "$dir"
done

# ----------------------------------------------------------------
# 2. Initialize PostgreSQL database (Requirement 26.7)
# ----------------------------------------------------------------
if command -v psql > /dev/null 2>&1; then
    # Create the infrasense PostgreSQL role if it does not exist
    if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='infrasense'" 2>/dev/null | grep -q 1; then
        DB_PASSWORD=$(openssl rand -base64 24)
        sudo -u postgres psql -c "CREATE USER infrasense WITH PASSWORD '${DB_PASSWORD}';" 2>/dev/null || true
        echo "DB_PASSWORD=${DB_PASSWORD}" > /etc/infrasense/.db_credentials
        chmod 600 /etc/infrasense/.db_credentials
        chown infrasense:infrasense /etc/infrasense/.db_credentials
    fi

    # Create the infrasense database if it does not exist
    if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='infrasense'" 2>/dev/null | grep -q 1; then
        sudo -u postgres psql -c "CREATE DATABASE infrasense OWNER infrasense;" 2>/dev/null || true
    fi

    sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE infrasense TO infrasense;" 2>/dev/null || true
else
    echo "WARNING: psql not found. Skipping PostgreSQL initialisation." >&2
    echo "         Please initialise the database manually before starting InfraSense." >&2
fi

# ----------------------------------------------------------------
# 3. Run database migrations (Requirement 26.7)
# ----------------------------------------------------------------
MIGRATIONS_DIR=/usr/local/share/infrasense/migrations

if [ -x /usr/local/bin/infrasense-api ]; then
    if [ -f /etc/infrasense/.db_credentials ]; then
        . /etc/infrasense/.db_credentials
    fi

    DB_URL="postgres://infrasense:${DB_PASSWORD:-}@localhost:5432/infrasense?sslmode=disable"

    if [ -d "$MIGRATIONS_DIR" ]; then
        /usr/local/bin/infrasense-api migrate \
            --db-url "$DB_URL" \
            --migrations-dir "$MIGRATIONS_DIR" \
            || echo "WARNING: Database migration failed. Please run migrations manually." >&2
    elif command -v migrate > /dev/null 2>&1; then
        migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" up \
            || echo "WARNING: Database migration failed. Please run migrations manually." >&2
    else
        echo "WARNING: No migration tool found. Please run database migrations manually." >&2
    fi
else
    echo "WARNING: infrasense-api binary not found. Skipping migrations." >&2
fi

# ----------------------------------------------------------------
# 4. Create default admin user
# ----------------------------------------------------------------
ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -dc 'A-Za-z0-9' | head -c 20)

if [ -x /usr/local/bin/infrasense-api ]; then
    /usr/local/bin/infrasense-api create-admin \
        --username admin \
        --password "${ADMIN_PASSWORD}" \
        --email "admin@infrasense.local" \
        2>/dev/null || true
fi

echo "ADMIN_PASSWORD=${ADMIN_PASSWORD}" > /etc/infrasense/.admin_credentials
chmod 600 /etc/infrasense/.admin_credentials
chown root:root /etc/infrasense/.admin_credentials

# ----------------------------------------------------------------
# 5. Enable and start all InfraSense services (Requirement 26.8)
# ----------------------------------------------------------------
if command -v systemctl > /dev/null 2>&1 && systemctl is-system-running --quiet 2>/dev/null; then
    systemctl daemon-reload

    for svc in \
        infrasense-api \
        infrasense-ipmi-collector \
        infrasense-redfish-collector \
        infrasense-snmp-collector \
        infrasense-proxmox-collector \
        infrasense-notification-service \
        infrasense-frontend; do
        systemctl enable "${svc}.service" || true
        systemctl start  "${svc}.service" || true
    done
fi

# ----------------------------------------------------------------
# 6. Print installation success message
# ----------------------------------------------------------------
echo ""
echo "============================================================"
echo "  InfraSense Platform installed successfully!"
echo "============================================================"
echo ""
echo "  Access URL : http://localhost"
echo ""
echo "  Admin credentials"
echo "    Username : admin"
echo "    Password : ${ADMIN_PASSWORD}"
echo ""
echo "  IMPORTANT: Save these credentials now."
echo "  They are also stored in /etc/infrasense/.admin_credentials"
echo ""
echo "  To check service status:"
echo "    systemctl status infrasense-api"
echo ""
echo "============================================================"
echo ""

exit 0

# -----------------------------------------------------------------------
# %preun – stop and disable services before package removal (Req 26.9)
# -----------------------------------------------------------------------
%preun
%systemd_preun \
    infrasense-api.service \
    infrasense-ipmi-collector.service \
    infrasense-redfish-collector.service \
    infrasense-snmp-collector.service \
    infrasense-proxmox-collector.service \
    infrasense-notification-service.service \
    infrasense-frontend.service

# On full removal (not upgrade), explicitly stop and disable services
if [ "$1" -eq 0 ]; then
    if command -v systemctl > /dev/null 2>&1; then
        for svc in \
            infrasense-api \
            infrasense-ipmi-collector \
            infrasense-redfish-collector \
            infrasense-snmp-collector \
            infrasense-proxmox-collector \
            infrasense-notification-service \
            infrasense-frontend; do
            if systemctl is-active --quiet "${svc}.service" 2>/dev/null; then
                systemctl stop "${svc}.service" || true
            fi
            if systemctl is-enabled --quiet "${svc}.service" 2>/dev/null; then
                systemctl disable "${svc}.service" || true
            fi
        done
    fi
fi

exit 0

# -----------------------------------------------------------------------
# %postun – post-removal cleanup
# -----------------------------------------------------------------------
%postun
%systemd_postun_with_restart \
    infrasense-api.service \
    infrasense-ipmi-collector.service \
    infrasense-redfish-collector.service \
    infrasense-snmp-collector.service \
    infrasense-proxmox-collector.service \
    infrasense-notification-service.service \
    infrasense-frontend.service

exit 0

# -----------------------------------------------------------------------
# %changelog
# -----------------------------------------------------------------------
%changelog
* Mon Jan 01 2024 InfraSense Team <support@infrasense.local> - 1.0.0-1
- Initial release of InfraSense Platform RPM package
- Supports RHEL 8+, CentOS Stream, Rocky Linux, and AlmaLinux (Requirements 26.1, 26.2)
- Installs binaries to /usr/local/bin/ (Requirement 26.3)
- Installs configuration files to /etc/infrasense/ (Requirement 26.4)
- Installs systemd service units to /usr/lib/systemd/system/ (Requirement 26.5)
- Creates infrasense system user and group (Requirement 26.6)
- Post-install script initialises Asset_Database schema (Requirement 26.7)
- Post-install script starts all InfraSense services (Requirement 26.8)
- Pre-removal script stops all InfraSense services (Requirement 26.9)

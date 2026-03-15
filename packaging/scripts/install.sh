#!/bin/bash
#
# InfraSense Platform Auto-Installer Script
# Detects OS and installs the appropriate package (.deb or .rpm)
#
# Requirements: 34.1, 34.2, 34.3, 34.4, 34.5, 34.8, 34.9, 34.10
#

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print functions
print_error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}$1${NC}"
}

print_info() {
    echo -e "${YELLOW}$1${NC}"
}

# Check if running as root (Requirement 34.1)
if [ "$EUID" -ne 0 ]; then
    print_error "This script must be run as root"
    echo "Please run: sudo $0"
    exit 1
fi

# Detect operating system distribution and version (Requirement 34.1)
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    VERSION=$VERSION_ID
    VERSION_CODENAME=${VERSION_CODENAME:-}
else
    print_error "Cannot detect operating system. /etc/os-release not found."
    exit 1
fi

print_info "Detected OS: $OS $VERSION"

# Validate required ports are available (Requirement 34.9)
print_info "Checking if required ports are available..."

check_port() {
    local port=$1
    if command -v netstat >/dev/null 2>&1; then
        if netstat -tuln 2>/dev/null | grep -q ":$port "; then
            print_error "Port $port is already in use"
            echo "InfraSense requires ports 80, 443, 5432, and 8428 to be available."
            echo "Please free up the port or stop the conflicting service."
            exit 1
        fi
    elif command -v ss >/dev/null 2>&1; then
        if ss -tuln 2>/dev/null | grep -q ":$port "; then
            print_error "Port $port is already in use"
            echo "InfraSense requires ports 80, 443, 5432, and 8428 to be available."
            echo "Please free up the port or stop the conflicting service."
            exit 1
        fi
    else
        print_info "Warning: Cannot check port availability (netstat/ss not found)"
    fi
}

check_port 80
check_port 443
check_port 5432
check_port 8428

print_success "All required ports are available"

# Function to install .deb package (Requirement 34.2)
install_deb() {
    print_info "Installing InfraSense on $OS $VERSION..."
    
    # Install required dependencies (Requirement 34.5)
    print_info "Installing dependencies..."
    apt-get update || {
        print_error "Failed to update package lists"
        exit 1
    }
    
    apt-get install -y wget postgresql-client systemd || {
        print_error "Failed to install dependencies"
        exit 1
    }
    
    # Download .deb package
    print_info "Downloading InfraSense package..."
    PACKAGE_URL="https://releases.infrasense.io/latest/infrasense_amd64.deb"
    PACKAGE_FILE="/tmp/infrasense_amd64.deb"
    
    wget -O "$PACKAGE_FILE" "$PACKAGE_URL" || {
        print_error "Failed to download package from $PACKAGE_URL"
        echo "Please check your internet connection or download the package manually."
        exit 1
    }
    
    # Install .deb package
    print_info "Installing InfraSense package..."
    dpkg -i "$PACKAGE_FILE" || {
        print_info "Fixing dependencies..."
        apt-get install -f -y || {
            print_error "Failed to install InfraSense package"
            exit 1
        }
    }
    
    # Clean up
    rm -f "$PACKAGE_FILE"
    
    print_success "InfraSense package installed successfully"
}

# Function to install .rpm package (Requirement 34.3)
install_rpm() {
    print_info "Installing InfraSense on $OS $VERSION..."
    
    # Install required dependencies (Requirement 34.5)
    print_info "Installing dependencies..."
    
    if command -v dnf >/dev/null 2>&1; then
        dnf install -y wget postgresql systemd || {
            print_error "Failed to install dependencies"
            exit 1
        }
    elif command -v yum >/dev/null 2>&1; then
        yum install -y wget postgresql systemd || {
            print_error "Failed to install dependencies"
            exit 1
        }
    else
        print_error "Neither dnf nor yum package manager found"
        exit 1
    fi
    
    # Download .rpm package
    print_info "Downloading InfraSense package..."
    PACKAGE_URL="https://releases.infrasense.io/latest/infrasense_x86_64.rpm"
    PACKAGE_FILE="/tmp/infrasense_x86_64.rpm"
    
    wget -O "$PACKAGE_FILE" "$PACKAGE_URL" || {
        print_error "Failed to download package from $PACKAGE_URL"
        echo "Please check your internet connection or download the package manually."
        exit 1
    }
    
    # Install .rpm package
    print_info "Installing InfraSense package..."
    rpm -ivh "$PACKAGE_FILE" || {
        print_error "Failed to install InfraSense package"
        exit 1
    }
    
    # Clean up
    rm -f "$PACKAGE_FILE"
    
    print_success "InfraSense package installed successfully"
}

# Function to display Docker installation instructions (Requirement 34.4)
show_docker_instructions() {
    print_error "Unsupported operating system: $OS $VERSION"
    echo ""
    echo "InfraSense native packages are only available for:"
    echo "  - Ubuntu 22.04 and later"
    echo "  - Debian 12 and later"
    echo "  - RHEL 8 and later"
    echo "  - Rocky Linux 8 and later"
    echo "  - AlmaLinux 8 and later"
    echo ""
    echo "Please use Docker installation instead:"
    echo ""
    echo "  # Install Docker and Docker Compose"
    echo "  curl -fsSL https://get.docker.com | sh"
    echo ""
    echo "  # Clone InfraSense repository"
    echo "  git clone https://github.com/infrasense/infrasense.git"
    echo "  cd infrasense"
    echo ""
    echo "  # Start InfraSense services"
    echo "  docker-compose up -d"
    echo ""
    echo "For more information, visit: https://docs.infrasense.io/installation/docker"
    echo ""
    exit 1
}

# Install based on detected OS
case "$OS" in
    ubuntu)
        # Ubuntu 22.04 and later supported
        if [ "${VERSION%%.*}" -ge 22 ]; then
            install_deb
        else
            show_docker_instructions
        fi
        ;;
    debian)
        # Debian 12 and later supported
        if [ "${VERSION%%.*}" -ge 12 ]; then
            install_deb
        else
            show_docker_instructions
        fi
        ;;
    rhel|rocky|almalinux)
        # RHEL/Rocky/AlmaLinux 8 and later supported
        if [ "${VERSION%%.*}" -ge 8 ]; then
            install_rpm
        else
            show_docker_instructions
        fi
        ;;
    centos)
        # CentOS Stream 8 and later supported
        if [ "${VERSION%%.*}" -ge 8 ]; then
            install_rpm
        else
            show_docker_instructions
        fi
        ;;
    *)
        show_docker_instructions
        ;;
esac

# Get the primary IP address for access URL
PRIMARY_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
if [ -z "$PRIMARY_IP" ]; then
    PRIMARY_IP="<server-ip>"
fi

# Print installation success message (Requirement 34.8)
echo ""
echo "========================================="
print_success "InfraSense Installation Complete!"
echo "========================================="
echo ""
echo "Access the dashboard at: http://$PRIMARY_IP"
echo ""
echo "Default admin credentials:"
echo "  Username: admin"
echo "  Password: (check /var/log/infrasense/install.log)"
echo ""
echo "IMPORTANT: Change the default admin password immediately!"
echo ""
echo "Next steps:"
echo "  1. Log in to the dashboard"
echo "  2. Change the default admin password"
echo "  3. Register your first device"
echo "  4. Configure alert rules"
echo "  5. Set up notification channels"
echo ""
echo "Documentation: https://docs.infrasense.io"
echo "Support: https://github.com/infrasense/infrasense/issues"
echo ""

exit 0

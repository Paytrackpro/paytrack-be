#!/bin/bash

# PayTrack Installation Script for Ubuntu
# This script installs PayTrack and its dependencies on Ubuntu

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PAYTRACK_USER="paytrack"
PAYTRACK_GROUP="paytrack"
PAYTRACK_HOME="/opt/paytrack"
PAYTRACK_CONFIG="/etc/paytrack"
PAYTRACK_LOGS="/var/log/paytrack"
GO_VERSION="1.21.6"

# Functions
print_header() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root"
        exit 1
    fi
}

# Check Ubuntu version
check_ubuntu_version() {
    if ! command -v lsb_release &> /dev/null; then
        print_error "This script is designed for Ubuntu. lsb_release not found."
        exit 1
    fi
    
    local ubuntu_version=$(lsb_release -rs)
    local major_version=$(echo "$ubuntu_version" | cut -d. -f1)
    
    if [ "$major_version" -lt 18 ]; then
        print_error "Ubuntu 18.04 or higher is required. Current version: $ubuntu_version"
        exit 1
    fi
    
    print_success "Ubuntu version check passed ($ubuntu_version)"
}

# Update system packages
update_system() {
    print_info "Updating system packages..."
    apt-get update -y
    apt-get upgrade -y
    print_success "System packages updated"
}

# Install required packages
install_dependencies() {
    print_info "Installing required packages..."
    
    # Install basic dependencies
    apt-get install -y \
        curl \
        wget \
        git \
        build-essential \
        ca-certificates \
        gnupg \
        lsb-release \
        software-properties-common \
        apt-transport-https \
        unzip \
        supervisor \
        nginx
    
    print_success "Basic dependencies installed"
}

# Install PostgreSQL
install_postgresql() {
    print_info "Installing PostgreSQL..."
    
    # Install PostgreSQL
    apt-get install -y postgresql postgresql-contrib
    
    # Start and enable PostgreSQL
    systemctl start postgresql
    systemctl enable postgresql
    
    print_success "PostgreSQL installed and started"
}

# Install Go
install_go() {
    print_info "Installing Go $GO_VERSION..."
    
    # Remove existing Go installation
    rm -rf /usr/local/go
    
    # Download and install Go
    wget -q https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz
    
    # Add Go to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    chmod +x /etc/profile.d/go.sh
    
    print_success "Go $GO_VERSION installed"
}

# Create PayTrack user and directories
create_user_and_directories() {
    print_info "Creating PayTrack user and directories..."
    
    # Create user and group
    if ! getent group "$PAYTRACK_GROUP" > /dev/null 2>&1; then
        groupadd "$PAYTRACK_GROUP"
    fi
    
    if ! getent passwd "$PAYTRACK_USER" > /dev/null 2>&1; then
        useradd -r -g "$PAYTRACK_GROUP" -d "$PAYTRACK_HOME" -s /bin/bash "$PAYTRACK_USER"
    fi
    
    # Create directories
    mkdir -p "$PAYTRACK_HOME"
    mkdir -p "$PAYTRACK_CONFIG"
    mkdir -p "$PAYTRACK_LOGS"
    mkdir -p "$PAYTRACK_HOME/logs"
    mkdir -p "$PAYTRACK_HOME/uploads"
    mkdir -p /var/lib/paytrack/backups
    
    # Set permissions
    chown -R "$PAYTRACK_USER:$PAYTRACK_GROUP" "$PAYTRACK_HOME"
    chown -R "$PAYTRACK_USER:$PAYTRACK_GROUP" "$PAYTRACK_LOGS"
    chown -R "$PAYTRACK_USER:$PAYTRACK_GROUP" /var/lib/paytrack
    
    print_success "User and directories created"
}

# Setup database
setup_database() {
    print_info "Setting up PostgreSQL database..."
    
    # Create database and user
    sudo -u postgres psql -c "CREATE USER paytrack WITH PASSWORD 'paytrack123';"
    sudo -u postgres psql -c "CREATE DATABASE mgmtng OWNER paytrack;"
    sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE mgmtng TO paytrack;"
    
    # Configure PostgreSQL for local connections
    local pg_version=$(sudo -u postgres psql -t -c "SELECT version();" | grep -oP '\d+\.\d+')
    local pg_config_dir="/etc/postgresql/${pg_version}/main"
    
    # Update pg_hba.conf for local connections
    if [ -f "$pg_config_dir/pg_hba.conf" ]; then
        cp "$pg_config_dir/pg_hba.conf" "$pg_config_dir/pg_hba.conf.backup"
        sed -i "s/local   all             paytrack                                peer/local   all             paytrack                                md5/" "$pg_config_dir/pg_hba.conf"
        systemctl restart postgresql
    fi
    
    print_success "Database setup completed"
}

# Build PayTrack
build_paytrack() {
    print_info "Building PayTrack..."
    
    # Source Go environment
    export PATH=$PATH:/usr/local/go/bin

    # Set GOMODCACHE to user's module cache directory
    export GOMODCACHE=$PAYTRACK_HOME/go/pkg/mod
    mkdir -p $GOMODCACHE
    
    # Set GOCACHE to user's cache directory
    export GOCACHE=$PAYTRACK_HOME/.cache
    
    # Build the application
    cd "$(dirname "$0")/.."
    env PATH="$PATH" GOCACHE="$GOCACHE" GOMODCACHE="$GOMODCACHE" go build -o mgmtngd ./cmd/mgmtngd
    
    # Copy binary to installation directory
    cp mgmtngd "$PAYTRACK_HOME/"
    chown "$PAYTRACK_USER:$PAYTRACK_GROUP" "$PAYTRACK_HOME/mgmtngd"
    chmod +x "$PAYTRACK_HOME/mgmtngd"
    
    print_success "PayTrack built successfully"
}

# Install configuration
install_configuration() {
    print_info "Installing configuration..."
    
    # Create configuration file
    cat > "$PAYTRACK_CONFIG/config.yaml" << EOF
db:
  dns: "host=localhost user=paytrack password=paytrack123 dbname=mgmtng port=5432 sslmode=disable TimeZone=UTC"

webServer:
  port: 6789
  hmacSecretKey: "$(openssl rand -hex 32)"
  aliveSessionHours: 24
  aesSecretKey: "$(openssl rand -hex 32)"
  authType: 0
  authHost: "http://localhost:8001"
  service:
    exchange: "bittrex"
    allowexchanges: "binance,kucoin,mexc"
    coimarketcapKey: ""
    authType: 0
    authHost: "localhost:50051"

logLevel: "info"
logDir: "$PAYTRACK_LOGS"

mail:
  addr: "smtp.gmail.com:587"
  userName: ""
  password: ""
  host: "smtp.gmail.com"
  from: "noreply@paytrack.local"
EOF
    
    chown "$PAYTRACK_USER:$PAYTRACK_GROUP" "$PAYTRACK_CONFIG/config.yaml"
    chmod 600 "$PAYTRACK_CONFIG/config.yaml"
    
    print_success "Configuration installed"
}

# Install systemd service
install_systemd_service() {
    print_info "Installing systemd service..."
    
    # Copy service file
    cp "$(dirname "$0")/paytrack.service" /etc/systemd/system/
    
    # Reload systemd and enable service
    systemctl daemon-reload
    systemctl enable paytrack.service
    
    print_success "Systemd service installed"
}

# Install backup scripts
install_backup_scripts() {
    print_info "Installing backup scripts..."
    
    # Copy backup scripts
    cp "$(dirname "$0")/backup.sh" "$PAYTRACK_HOME/"
    cp "$(dirname "$0")/restore.sh" "$PAYTRACK_HOME/"
    
    # Make scripts executable
    chmod +x "$PAYTRACK_HOME/backup.sh"
    chmod +x "$PAYTRACK_HOME/restore.sh"
    
    chown "$PAYTRACK_USER:$PAYTRACK_GROUP" "$PAYTRACK_HOME/backup.sh"
    chown "$PAYTRACK_USER:$PAYTRACK_GROUP" "$PAYTRACK_HOME/restore.sh"
    
    # Create backup cron job
    cat > /etc/cron.d/paytrack-backup << EOF
# PayTrack database backup - daily at 2 AM
0 2 * * * $PAYTRACK_USER cd $PAYTRACK_HOME && ./backup.sh
EOF
    
    print_success "Backup scripts installed"
}

# Configure firewall
configure_firewall() {
    print_info "Configuring firewall..."
    
    if command -v ufw &> /dev/null; then
        # Allow SSH, HTTP, HTTPS, and PayTrack port
        ufw --force enable
        ufw allow ssh
        ufw allow http
        ufw allow https
        ufw allow 6789/tcp
        
        print_success "Firewall configured"
    else
        print_info "UFW not installed, skipping firewall configuration"
    fi
}

# Start services
start_services() {
    print_info "Starting services..."
    
    # Start PayTrack
    systemctl start paytrack.service
    
    # Check if service is running
    if systemctl is-active --quiet paytrack.service; then
        print_success "PayTrack service started successfully"
    else
        print_error "Failed to start PayTrack service"
        systemctl status paytrack.service
        exit 1
    fi
}

# Main installation function
main() {
    print_header "PayTrack Installation for Ubuntu"
    
    check_root
    check_ubuntu_version
    update_system
    install_dependencies
    install_postgresql
    install_go
    create_user_and_directories
    setup_database
    build_paytrack
    install_configuration
    install_systemd_service
    install_backup_scripts
    configure_firewall
    start_services
    
    print_header "Installation Complete!"
    print_success "PayTrack has been installed successfully!"
    print_info "Service status: systemctl status paytrack.service"
    print_info "Logs: journalctl -u paytrack.service -f"
    print_info "Configuration: $PAYTRACK_CONFIG/config.yaml"
    print_info "Application directory: $PAYTRACK_HOME"
    print_info "Web interface: http://localhost:6789"
    print_info ""
    print_info "To create an admin user, run:"
    print_info "sudo -u postgres psql -d mgmtng -c \"UPDATE users SET role = 1 WHERE user_name = 'your_username';\""
    print_info ""
    print_info "To backup database: $PAYTRACK_HOME/backup.sh"
    print_info "To restore database: $PAYTRACK_HOME/restore.sh backup_file.sql"
}

# Run main function
main "$@" 
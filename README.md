# PayTrack - Payment Management System

PayTrack is a comprehensive payment management system built with Go, designed to handle payment requests, approvals, and cryptocurrency transactions.

## Features

- **Payment Management**: Create, track, and manage payment requests
- **User Management**: Role-based access control with approvers and administrators
- **Project Management**: Organize payments by projects
- **Cryptocurrency Support**: Multiple exchange integrations (Binance, Bittrex, CoinMarketCap)
- **Real-time Notifications**: WebSocket-based live updates
- **Email Integration**: Automated email notifications
- **Backup & Recovery**: Comprehensive database backup solutions
- **Docker Support**: Containerized deployment options

## Technology Stack

- **Backend**: Go 1.21+
- **Database**: PostgreSQL 15+
- **Web Framework**: Chi Router
- **ORM**: GORM
- **Authentication**: JWT with optional external auth service
- **Real-time**: WebSocket (Socket.IO)
- **Email**: SMTP integration
- **Containerization**: Docker & Docker Compose

## Installation & Deployment

PayTrack can be deployed using two methods:

### Method 1: Docker Compose (Recommended)

Docker Compose provides the easiest way to deploy PayTrack with all dependencies included.

#### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 2GB+ available RAM
- 10GB+ available disk space

#### Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd paytrack-be
   ```

2. **Create environment file**
   ```bash
   cp environment.template .env
   ```

3. **Configure environment variables**
   Edit `.env` file and update the following critical settings:
   ```env
   # Change these security keys in production
   HMAC_SECRET_KEY=your-super-secret-hmac-key-change-this-in-production
   AES_SECRET_KEY=your-super-secret-aes-key-change-this-in-production
   
   # Database password
   POSTGRES_PASSWORD=your-secure-database-password
   
   # Email configuration (optional)
   MAIL_USERNAME=your-email@gmail.com
   MAIL_PASSWORD=your-app-password
   ```

4. **Create configuration file**
   ```bash
   mkdir -p private
   envsubst < config.production.yaml > private/config.yaml
   ```

5. **Start the services**
   ```bash
   docker-compose up -d
   ```

6. **Verify deployment**
   ```bash
   docker-compose ps
   curl http://localhost:6789/health
   ```

7. **Access the application**
   - PayTrack: http://localhost:6789
   - PostgreSQL: localhost:5432

#### Docker Compose Management

- **Start services**: `docker-compose up -d`
- **Stop services**: `docker-compose down`
- **View logs**: `docker-compose logs -f paytrack`
- **Update application**: `docker-compose pull && docker-compose up -d`
- **Scale services**: `docker-compose up -d --scale paytrack=2`

### Method 2: Direct Installation on Ubuntu

This method installs PayTrack directly on Ubuntu servers for production environments.

#### Prerequisites

- Ubuntu 18.04+ (20.04+ recommended)
- Root access or sudo privileges
- 4GB+ available RAM
- 20GB+ available disk space

#### Automated Installation

1. **Download and run the installation script**
   ```bash
   # Clone repository
   git clone <repository-url>
   cd paytrack-be
   
   # Run installation script
   sudo ./scripts/install.sh
   ```

The installation script will:
- Install Go 1.21+
- Install PostgreSQL 15+
- Create paytrack user and directories
- Build the application
- Configure systemd service
- Setup database
- Install backup scripts
- Configure firewall

#### Manual Installation

If you prefer manual installation:

1. **Update system packages**
   ```bash
   sudo apt update && sudo apt upgrade -y
   ```

2. **Install dependencies**
   ```bash
   sudo apt install -y curl wget git build-essential ca-certificates \
       postgresql postgresql-contrib nginx supervisor
   ```

3. **Install Go**
   ```bash
   wget https://golang.org/dl/go1.21.6.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
   echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
   source /etc/profile.d/go.sh
   ```

4. **Create user and directories**
   ```bash
   sudo useradd -r -s /bin/bash paytrack
   sudo mkdir -p /opt/paytrack /etc/paytrack /var/log/paytrack
   sudo chown -R paytrack:paytrack /opt/paytrack /var/log/paytrack
   ```

5. **Setup database**
   ```bash
   sudo -u postgres psql -c "CREATE USER paytrack WITH PASSWORD 'paytrack123';"
   sudo -u postgres psql -c "CREATE DATABASE mgmtng OWNER paytrack;"
   sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE mgmtng TO paytrack;"
   ```

6. **Build application**
   ```bash
   cd paytrack-be
   go build -o mgmtngd ./cmd/mgmtngd
   sudo cp mgmtngd /opt/paytrack/
   sudo chown paytrack:paytrack /opt/paytrack/mgmtngd
   ```

7. **Install configuration**
   ```bash
   sudo cp config.production.yaml /etc/paytrack/config.yaml
   sudo chown paytrack:paytrack /etc/paytrack/config.yaml
   sudo chmod 600 /etc/paytrack/config.yaml
   ```

8. **Install systemd service**
   ```bash
   sudo cp scripts/paytrack.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable paytrack.service
   sudo systemctl start paytrack.service
   ```

#### Service Management

- **Start service**: `sudo systemctl start paytrack.service`
- **Stop service**: `sudo systemctl stop paytrack.service`
- **Restart service**: `sudo systemctl restart paytrack.service`
- **Check status**: `sudo systemctl status paytrack.service`
- **View logs**: `sudo journalctl -u paytrack.service -f`

## Configuration

### Database Configuration

PayTrack uses PostgreSQL as its primary database. The database configuration is specified in the `config.yaml` file:

```yaml
db:
  dns: "host=localhost user=paytrack password=paytrack123 dbname=mgmtng port=5432 sslmode=disable TimeZone=UTC"
```

### Web Server Configuration

```yaml
webServer:
  port: 6789
  hmacSecretKey: "your-secret-key"
  aliveSessionHours: 24
  aesSecretKey: "your-aes-key"
  authType: 0  # 0=local auth, 1=external auth
```

### Email Configuration

```yaml
mail:
  addr: "smtp.gmail.com:587"
  userName: "your-email@gmail.com"
  password: "your-app-password"
  host: "smtp.gmail.com"
  from: "noreply@paytrack.local"
```

## Data Backup & Recovery

PayTrack includes comprehensive backup and recovery solutions to protect your data.

### Backup Methods

#### 1. Automated Daily Backups

The installation automatically sets up daily backups via cron:

```bash
# Check backup cron job
sudo cat /etc/cron.d/paytrack-backup

# Manual backup
sudo -u paytrack /opt/paytrack/backup.sh
```

#### 2. Docker Compose Backups

For Docker deployments:

```bash
# Backup using Docker
docker-compose exec postgres pg_dump -U paytrack mgmtng > backup.sql

# Or use the backup script
docker-compose exec paytrack ./backup.sh
```

#### 3. Manual Database Backup

```bash
# SQL format backup
pg_dump -h localhost -U paytrack -d mgmtng > paytrack_backup.sql

# Custom format backup (recommended)
pg_dump -h localhost -U paytrack -d mgmtng -Fc > paytrack_backup.custom

# Compressed backup
pg_dump -h localhost -U paytrack -d mgmtng | gzip > paytrack_backup.sql.gz
```

### Backup Script Features

The included backup script (`scripts/backup.sh`) provides:

- **Multiple formats**: SQL, custom, and compressed backups
- **Automatic cleanup**: Removes backups older than 30 days
- **Backup verification**: Checks backup integrity
- **Detailed logging**: Comprehensive backup information
- **Error handling**: Graceful failure handling

#### Backup Script Usage

```bash
# Basic backup
./scripts/backup.sh

# Custom backup directory
BACKUP_DIR=/custom/backup/path ./scripts/backup.sh

# Custom retention period (7 days)
BACKUP_RETENTION_DAYS=7 ./scripts/backup.sh

# Custom database connection
DB_HOST=remote-host DB_USER=custom-user ./scripts/backup.sh
```

### Data Recovery

#### 1. Using Restore Script

The restore script (`scripts/restore.sh`) provides easy recovery:

```bash
# List available backups
./scripts/restore.sh --list

# Restore from backup
./scripts/restore.sh paytrack_backup_20240101_120000.sql.gz

# Force restore (skip confirmation)
./scripts/restore.sh --force backup_file.sql

# Restore from custom format
./scripts/restore.sh backup_file.custom
```

#### 2. Manual Recovery

```bash
# Drop existing database
sudo -u postgres psql -c "DROP DATABASE IF EXISTS mgmtng;"

# Create new database
sudo -u postgres psql -c "CREATE DATABASE mgmtng OWNER paytrack;"

# Restore from SQL backup
psql -h localhost -U paytrack -d mgmtng < backup.sql

# Restore from custom format
pg_restore -h localhost -U paytrack -d mgmtng backup.custom

# Restore from compressed backup
gunzip -c backup.sql.gz | psql -h localhost -U paytrack -d mgmtng
```

#### 3. Docker Recovery

```bash
# Stop services
docker-compose down

# Restore database
docker-compose up -d postgres
docker-compose exec postgres psql -U paytrack -d mgmtng < backup.sql

# Start all services
docker-compose up -d
```

### Backup Best Practices

1. **Regular Backups**: Ensure daily automated backups are running
2. **Multiple Locations**: Store backups in multiple locations (local + cloud)
3. **Test Restores**: Regularly test backup restoration procedures
4. **Monitor Backup Size**: Monitor backup file sizes for anomalies
5. **Secure Storage**: Encrypt backups containing sensitive data
6. **Document Procedures**: Keep recovery procedures documented and accessible

### Backup Storage Recommendations

- **Local Storage**: `/var/lib/paytrack/backups` (default)
- **Network Storage**: NFS or SMB mounted drives
- **Cloud Storage**: AWS S3, Google Cloud Storage, Azure Blob
- **Offsite Backup**: Regular transfer to offsite locations

## User Management

### Creating Admin Users

After installation, create an admin user:

```bash
# Connect to database
sudo -u postgres psql -d mgmtng

# Update user role (1 = admin)
UPDATE users SET "role" = 1 WHERE "user_name" = 'your_username';
```

### User Roles

- **Role 0**: Regular user
- **Role 1**: Administrator
- **Role 2**: Approver

## Security Considerations

1. **Change Default Passwords**: Update all default passwords in production
2. **Use Strong Secrets**: Generate strong HMAC and AES keys
3. **Enable SSL**: Configure SSL certificates for HTTPS
4. **Firewall Configuration**: Restrict access to necessary ports only
5. **Regular Updates**: Keep system and dependencies updated
6. **Database Security**: Use strong database passwords and restrict access
7. **File Permissions**: Ensure proper file and directory permissions

## Monitoring & Maintenance

### Health Checks

- **Application Health**: `curl http://localhost:6789/health`
- **Database Health**: `pg_isready -h localhost -U paytrack`
- **Service Status**: `systemctl status paytrack.service`

### Log Management

- **Application Logs**: `/var/log/paytrack/mgmt.log`
- **System Logs**: `journalctl -u paytrack.service`
- **Database Logs**: `/var/log/postgresql/postgresql-*.log`

### Performance Monitoring

- **Resource Usage**: `top`, `htop`, `free -h`
- **Database Performance**: `pg_stat_statements`
- **Disk Usage**: `df -h`, `du -sh /opt/paytrack`

## Troubleshooting

### Common Issues

1. **Service won't start**
   ```bash
   sudo systemctl status paytrack.service
   sudo journalctl -u paytrack.service -f
   ```

2. **Database connection issues**
   ```bash
   pg_isready -h localhost -U paytrack
   sudo -u postgres psql -d mgmtng
   ```

3. **Port conflicts**
   ```bash
   sudo netstat -tulpn | grep :6789
   sudo lsof -i :6789
   ```

4. **Permission issues**
   ```bash
   sudo chown -R paytrack:paytrack /opt/paytrack
   sudo chmod 755 /opt/paytrack/mgmtngd
   ```

### Getting Help

- **Check logs**: Always check application and system logs first
- **Verify configuration**: Ensure configuration files are correct
- **Test connectivity**: Verify database and network connectivity
- **Resource availability**: Check disk space and memory usage

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue in the repository
- Check the troubleshooting section
- Review the logs for error messages

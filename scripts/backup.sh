#!/bin/bash

# PayTrack Database Backup Script
# This script creates a backup of the PayTrack PostgreSQL database

set -e

# Configuration
DB_NAME=${DB_NAME:-mgmtng}
DB_USER=${DB_USER:-paytrack}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
BACKUP_DIR=${BACKUP_DIR:-./backups}
BACKUP_RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-30}

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Generate timestamp for backup filename
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="$BACKUP_DIR/paytrack_backup_$TIMESTAMP.sql"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting PayTrack database backup...${NC}"

# Check if pg_dump is available
if ! command -v pg_dump &> /dev/null; then
    echo -e "${RED}Error: pg_dump is not installed. Please install PostgreSQL client tools.${NC}"
    exit 1
fi

# Perform database backup
echo -e "${YELLOW}Creating backup: $BACKUP_FILE${NC}"

if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    --no-password \
    --verbose \
    --format=custom \
    --file="$BACKUP_FILE.custom" 2>/dev/null; then
    
    echo -e "${GREEN}Custom format backup created successfully: $BACKUP_FILE.custom${NC}"
else
    echo -e "${RED}Custom format backup failed. Trying SQL format...${NC}"
fi

# Create SQL backup as well
if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    --no-password \
    --verbose \
    --file="$BACKUP_FILE" 2>/dev/null; then
    
    echo -e "${GREEN}SQL backup created successfully: $BACKUP_FILE${NC}"
    
    # Compress the SQL backup
    if command -v gzip &> /dev/null; then
        gzip "$BACKUP_FILE"
        echo -e "${GREEN}Backup compressed: $BACKUP_FILE.gz${NC}"
    fi
else
    echo -e "${RED}SQL backup failed${NC}"
    exit 1
fi

# Clean up old backups
echo -e "${YELLOW}Cleaning up old backups (older than $BACKUP_RETENTION_DAYS days)...${NC}"
find "$BACKUP_DIR" -name "paytrack_backup_*.sql*" -type f -mtime +$BACKUP_RETENTION_DAYS -delete 2>/dev/null || true
find "$BACKUP_DIR" -name "paytrack_backup_*.custom" -type f -mtime +$BACKUP_RETENTION_DAYS -delete 2>/dev/null || true

echo -e "${GREEN}Database backup completed successfully!${NC}"
echo -e "${GREEN}Backup location: $BACKUP_DIR${NC}"

# Show backup size
if [ -f "$BACKUP_FILE.gz" ]; then
    BACKUP_SIZE=$(du -h "$BACKUP_FILE.gz" | cut -f1)
    echo -e "${GREEN}Backup size: $BACKUP_SIZE${NC}"
elif [ -f "$BACKUP_FILE" ]; then
    BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    echo -e "${GREEN}Backup size: $BACKUP_SIZE${NC}"
fi

# Create a backup info file
cat > "$BACKUP_DIR/backup_info_$TIMESTAMP.txt" << EOF
PayTrack Database Backup Information
===================================
Backup Date: $(date)
Database: $DB_NAME
Host: $DB_HOST
Port: $DB_PORT
User: $DB_USER
Backup File: $(basename "$BACKUP_FILE")
Backup Size: $BACKUP_SIZE
EOF

echo -e "${GREEN}Backup information saved to: $BACKUP_DIR/backup_info_$TIMESTAMP.txt${NC}" 
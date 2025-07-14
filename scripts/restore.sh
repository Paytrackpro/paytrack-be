#!/bin/bash

# PayTrack Database Restore Script
# This script restores a PayTrack PostgreSQL database from backup

set -e

# Configuration
DB_NAME=${DB_NAME:-mgmtng}
DB_USER=${DB_USER:-paytrack}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
BACKUP_DIR=${BACKUP_DIR:-./backups}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to display usage
usage() {
    echo -e "${YELLOW}Usage: $0 [OPTIONS] <backup_file>${NC}"
    echo -e "${YELLOW}Options:${NC}"
    echo -e "  -h, --help              Show this help message"
    echo -e "  -f, --force             Skip confirmation prompt"
    echo -e "  -l, --list              List available backups"
    echo -e "  --db-name <name>        Database name (default: mgmtng)"
    echo -e "  --db-user <user>        Database user (default: paytrack)"
    echo -e "  --db-host <host>        Database host (default: localhost)"
    echo -e "  --db-port <port>        Database port (default: 5432)"
    echo -e "  --backup-dir <dir>      Backup directory (default: ./backups)"
    echo -e ""
    echo -e "${YELLOW}Examples:${NC}"
    echo -e "  $0 paytrack_backup_20240101_120000.sql.gz"
    echo -e "  $0 -f paytrack_backup_20240101_120000.custom"
    echo -e "  $0 -l"
}

# Function to list available backups
list_backups() {
    echo -e "${YELLOW}Available backups in $BACKUP_DIR:${NC}"
    if [ -d "$BACKUP_DIR" ]; then
        ls -la "$BACKUP_DIR"/paytrack_backup_*.sql* "$BACKUP_DIR"/paytrack_backup_*.custom 2>/dev/null || echo -e "${RED}No backups found${NC}"
    else
        echo -e "${RED}Backup directory not found: $BACKUP_DIR${NC}"
    fi
}

# Parse command line arguments
FORCE=false
LIST=false
BACKUP_FILE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -l|--list)
            LIST=true
            shift
            ;;
        --db-name)
            DB_NAME="$2"
            shift 2
            ;;
        --db-user)
            DB_USER="$2"
            shift 2
            ;;
        --db-host)
            DB_HOST="$2"
            shift 2
            ;;
        --db-port)
            DB_PORT="$2"
            shift 2
            ;;
        --backup-dir)
            BACKUP_DIR="$2"
            shift 2
            ;;
        -*)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            exit 1
            ;;
        *)
            BACKUP_FILE="$1"
            shift
            ;;
    esac
done

# If list option is specified, show backups and exit
if [ "$LIST" = true ]; then
    list_backups
    exit 0
fi

# Check if backup file is provided
if [ -z "$BACKUP_FILE" ]; then
    echo -e "${RED}Error: Backup file is required${NC}"
    usage
    exit 1
fi

# Check if backup file exists
if [ ! -f "$BACKUP_FILE" ]; then
    # Try to find the file in the backup directory
    if [ -f "$BACKUP_DIR/$BACKUP_FILE" ]; then
        BACKUP_FILE="$BACKUP_DIR/$BACKUP_FILE"
    else
        echo -e "${RED}Error: Backup file not found: $BACKUP_FILE${NC}"
        exit 1
    fi
fi

echo -e "${YELLOW}Starting PayTrack database restore...${NC}"

# Check if required tools are available
if ! command -v psql &> /dev/null; then
    echo -e "${RED}Error: psql is not installed. Please install PostgreSQL client tools.${NC}"
    exit 1
fi

# Show restore information
echo -e "${YELLOW}Restore Information:${NC}"
echo -e "Database: $DB_NAME"
echo -e "Host: $DB_HOST"
echo -e "Port: $DB_PORT"
echo -e "User: $DB_USER"
echo -e "Backup File: $BACKUP_FILE"
echo -e "File Size: $(du -h "$BACKUP_FILE" | cut -f1)"

# Confirmation prompt
if [ "$FORCE" = false ]; then
    echo -e "${RED}WARNING: This will drop and recreate the database $DB_NAME${NC}"
    echo -e "${RED}All existing data will be lost!${NC}"
    read -p "Are you sure you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Restore cancelled${NC}"
        exit 0
    fi
fi

# Determine file type and restore method
if [[ "$BACKUP_FILE" == *.custom ]]; then
    echo -e "${YELLOW}Restoring from custom format backup...${NC}"
    
    # Drop and recreate database
    echo -e "${YELLOW}Dropping existing database...${NC}"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "DROP DATABASE IF EXISTS $DB_NAME;" postgres
    
    echo -e "${YELLOW}Creating new database...${NC}"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "CREATE DATABASE $DB_NAME;" postgres
    
    # Restore from custom format
    if pg_restore -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        --no-password \
        --verbose \
        --clean \
        --if-exists \
        "$BACKUP_FILE"; then
        echo -e "${GREEN}Custom format restore completed successfully!${NC}"
    else
        echo -e "${RED}Custom format restore failed${NC}"
        exit 1
    fi
    
elif [[ "$BACKUP_FILE" == *.sql.gz ]]; then
    echo -e "${YELLOW}Restoring from compressed SQL backup...${NC}"
    
    # Drop and recreate database
    echo -e "${YELLOW}Dropping existing database...${NC}"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "DROP DATABASE IF EXISTS $DB_NAME;" postgres
    
    echo -e "${YELLOW}Creating new database...${NC}"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "CREATE DATABASE $DB_NAME;" postgres
    
    # Restore from compressed SQL
    if gunzip -c "$BACKUP_FILE" | psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME"; then
        echo -e "${GREEN}Compressed SQL restore completed successfully!${NC}"
    else
        echo -e "${RED}Compressed SQL restore failed${NC}"
        exit 1
    fi
    
elif [[ "$BACKUP_FILE" == *.sql ]]; then
    echo -e "${YELLOW}Restoring from SQL backup...${NC}"
    
    # Drop and recreate database
    echo -e "${YELLOW}Dropping existing database...${NC}"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "DROP DATABASE IF EXISTS $DB_NAME;" postgres
    
    echo -e "${YELLOW}Creating new database...${NC}"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "CREATE DATABASE $DB_NAME;" postgres
    
    # Restore from SQL
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" < "$BACKUP_FILE"; then
        echo -e "${GREEN}SQL restore completed successfully!${NC}"
    else
        echo -e "${RED}SQL restore failed${NC}"
        exit 1
    fi
    
else
    echo -e "${RED}Error: Unsupported backup file format${NC}"
    echo -e "${RED}Supported formats: .sql, .sql.gz, .custom${NC}"
    exit 1
fi

echo -e "${GREEN}Database restore completed successfully!${NC}"
echo -e "${GREEN}Database: $DB_NAME${NC}"
echo -e "${GREEN}Host: $DB_HOST:$DB_PORT${NC}"

# Verify the restore
echo -e "${YELLOW}Verifying restore...${NC}"
TABLE_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null || echo "0")

if [ "$TABLE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}Verification successful: $TABLE_COUNT tables found${NC}"
else
    echo -e "${RED}Verification failed: No tables found${NC}"
fi 
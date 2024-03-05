#!/usr/bin/with-contenv bashio
CONFIG_PATH=/data/options.json

# Create main config
bashio::log.info "Setting up configuration"

# Create main config
export LOG_LEVEL=$(bashio::config 'log_level')
export BACKUP_DIRECTORY=$(bashio::config 'backup_directory')
export PROTON_DRIVE_USER=$(bashio::config 'proton_drive_user')
export PROTON_DRIVE_PASSWORD=$(bashio::config 'proton_drive_password')

# Start application
bashio::log.info "Starting Home Assistant Proton Drive Backup"
./hassio_proton_drive_backup
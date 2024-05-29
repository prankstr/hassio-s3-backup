#!/usr/bin/with-contenv bashio
CONFIG_PATH=/data/options.json

# Create main config
bashio::log.info "Setting up configuration"

# Create main config
export STORAGE_BACKEND=$(bashio::config 'storage_backend')
export LOG_LEVEL=$(bashio::config 'log_level')
export BACKUP_DIRECTORY=$(bashio::config 'backup_directory')
export PROTON_DRIVE_USER=$(bashio::config 'proton.username')
export PROTON_DRIVE_PASSWORD=$(bashio::config 'proton.password')
export STORJ_ACCESS_GRANT=$(bashio::config 'storj.access_grant')
export S3_ENDPOINT=$(bashio::config 's3.endpoint')
export S3_BUCKET_NAME=$(bashio::config 's3.bucket')
export S3_ACCESS_KEY_ID=$(bashio::config 's3.access_key_id')
export S3_SECRET_ACCESS_KEY=$(bashio::config 's3.secret_access_key')

# Start application
bashio::log.info "Starting Home Assistant Proton Drive Backup"
env
./hassio_proton_drive_backup

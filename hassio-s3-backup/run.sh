#!/usr/bin/with-contenv bashio
CONFIG_PATH=/data/options.json

# Create main config
bashio::log.info "Setting up configuration"

# Create main config
export LOG_LEVEL=$(bashio::config 'log_level')
export S3_ENDPOINT=$(bashio::config 's3_endpoint')
export S3_BUCKET_NAME=$(bashio::config 's3_bucket')
export S3_ACCESS_KEY_ID=$(bashio::config 's3_access_key_id')
export S3_SECRET_ACCESS_KEY=$(bashio::config 's3_secret_access_key')

# Start application
bashio::log.info "Starting Home Assistant Proton Drive Backup"
./hassio_s3_backup

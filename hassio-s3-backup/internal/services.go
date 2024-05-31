package internal

import (
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/hassio"
	"hassio-proton-drive-backup/internal/storage"
)

type Services struct {
	ConfigService  *config.Service
	StorageService storage.Service
	HassioService  *hassio.Service
}

## Configuration

Configure the add-on through the Home Assistant UI with the following options:

- `log_level`: Set the logging level (options: "Info", "Debug", "Warn", "Error"; default: "Info").
- `endpoint`: The endpoint for the S3 compatible storage.
- `bucket`: The directory used for storing backups on Proton Drive(default: "Home Assistant Backups")
- `access_key_id`: The S3 Access key ID.
- `secret_access_key`: The S3 Secret Access Key.

## Features

- Creates backups on a configurable schedule.
- Sync backups to S3.
- Housekeeping of old backups.

## Limitations

It's still early days so a lot.. Some of the most noteworthy:

- **Only supports full backups and restores.**
  - All partial backups are ignored for now. Partial restore can be done from Home Assistants own interface if needed.
- **Handles all full backups, even the ones created outside of the addon.**
  - Any full backup in home assistant will be recognized by the addon and synced to Proton Drive and it's currently not possible to set a backup to only exist in Home Assistant or Proton Drive.
- **Doesn't handle orphaned backups in Proton Drive.**
  - If the addon looses track of a backup in Proton Drive it won't be added back. So it won't be cleaned up by the addon or possible to restore to this backup from the addon, it can however be downloaded manually and restored to of course.


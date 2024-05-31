## Configuration
Configure the add-on through the Home Assistant UI with the following options:
- `backup_directory`: The directory used for storing backups on Proton Drive(default: "Home Assistant Backups")
- `log_level`: Set the logging level (options: "Info", "Debug", "Warn", "Error"; default: "Info").
- `proton_drive_user`: Proton Drive username
- `proton_drive_password`: Proton Drive password

## Features
- Creates backups on a configurable schedule.
- Sync backups to Proton Drive.
- Housekeeping of old backups.

## Limitations
It's still early days so a lot.. Some of the most noteworthy:

- **You have to use your Proton Drive username and password, no 2FA support.**
  - Proton Drive does currently not support any type of application credentials, hence you have you use your owner username and password to login. 2FA is not supported as it wouldn't allow uploading the backups on a schedule without manual intervention. Might add a manual sync mode in the future.
- **Only supports full backups and restores.**
  - All partial backups are ignored for now. Partial restore can be done from Home Assistants own interface if needed.
- **Handles all full backups, even the ones created outside of the addon.**
  - Any full backup in home assistant will be recognized by the addon and synced to Proton Drive and it's currently not possible to set a backup to only exist in Home Assistant or Proton Drive.
- **Doesn't handle orphaned backups in Proton Drive.**
  - If the addon looses track of a backup in Proton Drive it won't be added back. So it won't be cleaned up by the addon or possible to restore to this backup from the addon, it can however be downloaded manually and restored to of course.
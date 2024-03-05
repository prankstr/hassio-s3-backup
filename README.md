# Home Assistant Proton Drive Backup

## Overview
Home Assistant Proton Drive Backup allows for the automated scheduling and synchronization of Home Assistant backups to Proton Drive. This addon emerged from my personal need for a Proton Drive-based backup solution and is heavily inspired by [Home Assistant Google Drive Backup](https://github.com/sabeechen/hassio-google-drive-backup).

If you're just looking for an easy backup solution for Home Assistant and don't mind using Google Drive you probably want to use [Home Assistant Google Drive Backup](https://github.com/sabeechenhassio-google-drive-backup) in favor of this. As of now, this addon doesn't match the featureset and reliability of the Google Drive counterpart.

But if Proton Drive is your platform of choice, this addon offers a solution for that purpose!

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

## Installation

### Prerequisites
- A Proton Drive account

### Steps
1. **Add Repository to Home Assistant:**
   [![Add repository to my Home Assistant](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fprankstr%2Fhassio-proton-drive-backup) 

    Click the big blue button.
    
    Or manually:
   - Navigate to the Add-on Store in your Home Assistant UI: `Settings` -> `Add-ons` -> `Add-on Store`.
   - Click the 3-dots in the upper right corner, select `Repositories`, and paste in this URL: [https://github.com/prankstr/hassio-proton-drive-backup](https://github.com/prankstr/hassio-proton-drive-backup)

2. **Install Home Assistant Proton Drive Backup**
   - Refresh the page
   - Find Home Assistant Proton Drive Backup in the list of available add-ons, open it and click 'Install'.

## Configuration
Configure the add-on through the Home Assistant UI with the following options:
- `backup_directory`: The directory used for storing backups on Proton Drive(default: "Home Assistant Backups")
- `log_level`: Set the logging level (options: "Info", "Debug", "Warn", "Error"; default: "Info").
- `proton_drive_user`: Proton Drive username
- `proton_drive_password`: Proton Drive password

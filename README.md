# Home Assistant S3 Backup

![Home Page Preview](images/home.png "Home Assistant S3 Backup")

## Overview

Home Assistant S3 Backup allows for the automated scheduling and synchronization of Home Assistant backups to S3. This addon emerged from my personal need for a S3-based backup solution and is heavily inspired by [Home Assistant Google Drive Backup](https://github.com/sabeechen/hassio-google-drive-backup).

If you're just looking for an easy backup solution for Home Assistant and don't mind using Google Drive you probably want to use [Home Assistant Google Drive Backup](https://github.com/sabeechen/hassio-google-drive-backup) in favor of this. As of now, this addon doesn't match the featureset and reliability of the Google Drive counterpart.

But if S3 is your platform of choice, this addon offers a solution for that purpose!

## Features

- Creates backups on a configurable schedule.
- Sync backups to S3.
- Housekeeping of old backups.

## Limitations

It's still early days so a lot.. Some of the most noteworthy:

- **No sensors or monitoring**
  - The addon doesn't create any sensors in Home Assistant or provide other means for monitoring the backup.
- **No generational backups**
  - If you need to save backups for longer you'd have to move them out of the backup directoy so they're no longer managed by the addon.
- **Only supports full backups and restores.**
  - All partial backups are ignored for now. Partial restore can be done from Home Assistants own interface if needed.
- **Handles all full backups, even the ones created outside of the addon.**
  - Any full backup in home assistant will be recognized by the addon and synced to S3 and it's currently not possible to set a backup to only exist in Home Assistant or S3.

## Installation

1. **Add Repository to Home Assistant:**
   [![Add repository to my Home Assistant](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fprankstr%2Fhassio-proton-drive-backup)

   Click the big blue button.

   Or manually:

   - Navigate to the Add-on Store in your Home Assistant UI: `Settings` -> `Add-ons` -> `Add-on Store`.
   - Click the 3-dots in the upper right corner, select `Repositories`, and paste in this URL: [https://github.com/prankstr/hassio-proton-drive-backup](https://github.com/prankstr/hassio-proton-drive-backup)

2. **Install Home Assistant S3 Backup**
   - Refresh the page
   - Find Home Assistant S3 Backup in the list of available add-ons, open it and click 'Install'.

## Configuration

Configure the add-on through the Home Assistant UI with the following options:

- `log_level`: Set the logging level (options: "Info", "Debug", "Warn", "Error"; default: "Info").
- `endpoint`: The endpoint for the S3 compatible storage.
- `bucket`: The directory used for storing backups on Proton Drive(default: "Home Assistant Backups")
- `access_key_id`: The S3 Access key ID.
- `secret_access_key`: The S3 Secret Access Key.

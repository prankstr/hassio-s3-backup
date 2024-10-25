# Home Assistant S3 Backup

Find all the available information over at [GitHub](https://github.com/prankstr/hassio-s3-backup)

## Installation

1. **Add Repository to Home Assistant:**

[![Add repository to my Home Assistant](https://my.home-assistant.io/badges/supervisor_add_add-on_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_add-on_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fprankstr%2Fhassio-s3-backup)
Click the big blue button.

Or manually:

- Navigate to the Add-on Store in your Home Assistant UI: `Settings` -> `Add-ons` -> `Add-on Store`.
- Click the 3-dots in the upper right corner, select `Repositories`, and paste in this URL: [https://github.com/prankstr/hassio-s3-backup](https://github.com/prankstr/hassio-s3-backup).

2. **Install Home Assistant S3 Backup**
   - Refresh the page
   - Find Home Assistant S3 Backup in the list of available add-ons, open it and click 'Install'.

## Configuration

Before starting, the add-on needs to be configured with the following settings:

- `log_level`: Set the logging level (options: "Info", "Debug", "Warn", "Error"; default: "Info").
- `s3_bucket`: Name of bucket in S3 where backups will be stored(default: "home-assistant-backups")
- `s3_endpoint`: The endpoint for the S3 compatible storage.
- `s3_access_key_id`: The S3 Access key ID.
- `s3_secret_access_key`: The S3 Secret Access Key.

When the add-on is running backup related setting can be configured from the UI. It should be pretty self-explanatory but here's a quick rundown of the settings:

> [!NOTE]
> Number of backups to keep is initially set to 0 for both S3 and Home Assistant. This means that no backups will be deleted and is a safe guard to prevent unwanted deletion of backups when enabling the add-on for the first time.
>
> It's recommended to set these to a reasonable value to avoid running out of storage. As soon as you set a number the add-on will remove any full backups exceeding this number.

- **Name format**: The format of the name of the backup. Supports placeholders for date and time(default: Full Backup {year}-{month}-{day} {hr24}:{min}:{sec})
- **Number of backups to keep in S3**: The number of backups to keep in S3 before deleting the oldest ones(default: 0)
- **Number of backups to keep in Home Assistant**: The number of backups to keep in Home Assistant before deleting the oldest ones(default: 0)
- **Days between backups:** The number of days between backups(default: 3)

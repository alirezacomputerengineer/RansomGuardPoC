# RansomGuard - Ransomware Detection Proof of Concept (PoC)

RansomGuard is a Proof of Concept (PoC) project aimed at detecting ransomware attacks on servers by creating and monitoring honeypot files. If any honeypot file is modified, RansomGuard will immediately trigger an alert, providing real-time detection of suspicious activity that could indicate ransomware attempting to encrypt or alter files.

## Table of Contents

- [Introduction](#introduction)
- [Features](#features)
- [Technical Overview](#technical-overview)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Alerting System](#alerting-system)
- [Contributing](#contributing)
- [License](#license)

## Introduction

Ransomware attacks are one of the most critical security threats, with potential to cause significant financial and operational damage. RansomGuard aims to mitigate this risk by creating honeypot files that act as decoys. If ransomware attempts to modify these files, RansomGuard detects this activity and issues an alert to system administrators, allowing them to take quick action.

## Features

- **Honeypot File Creation**: Automatically generates files to serve as ransomware bait.
- **Continuous Monitoring**: Monitors honeypot files for any changes, detecting potential ransomware activity in real-time.
- **Alert System**: Triggers alerts immediately upon detection of modifications, notifying admins.
- **Configurable Options**: Customize monitoring parameters and alert preferences.
- **Extensible Alerting**: Integrate with various alerting channels (e.g., email, Slack, webhook) for real-time notification.

## Technical Overview

RansomGuard is developed in Go and utilizes a file-monitoring service to detect any unauthorized modifications to honeypot files. Key components include:

1. **Honeypot File Management**: Creates and maintains decoy files across specified directories.
2. **File Monitoring**: Watches for modifications to honeypot files using `inotify` (Linux) or other file monitoring tools. 
3. **Alerting**: When changes are detected, an alert is sent via the configured alerting channel(s).

## Installation

To install RansomGuard, follow these steps:

1. Download the Source Code:
Download the latest version of RansomGuard from this GitHub page or by clicking the "Download ZIP" option on the repository’s main page. Once downloaded, extract the ZIP file.

2. Navigate to the Project Directory:
   ```bash
   cd /path/to/extracted/ransomguard
   ```

3. Build the project:
   ```bash
   go build -o ransomguard
   ```

## Usage

 Run RansomGuard with configuration file:
   ```bash
   ./ransomguard
   ```
## Configuration

RansomGuard’s behavior can be configured via a JSON configuration file. Below is an example configuration:

```json
{
  "IFTTT_URL": "https://maker.ifttt.com/trigger/alert/with/key/your-ifttt-key",
  "honeypot_files": [
    {
      "name": "decoy1",
      "extensions": ".txt",
      "volume": 1024,
      "route": "/path/to/decoy1"
    },
    {
      "name": "decoy2",
      "extensions": ".docx",
      "volume": 2048,
      "route": "/path/to/decoy2"
    },
    {
      "name": "decoy3",
      "extensions": ".pdf",
      "volume": 512,
      "route": "/path/to/decoy3"
    }
  ]
}
```

### Configuration Options

- **IFTTT_URL**: The webhook URL for IFTTT (If This Then That) to trigger alerts. Replace `your-ifttt-key` with your IFTTT key.
- **honeypot_files**: An array of honeypot file objects. Each object represents a decoy file with the following properties:
  - **name**: The base name of the honeypot file (e.g., `"decoy1"`).
  - **extensions**: The file extension to use for the honeypot file (e.g., `".txt"`, `".pdf"`).
  - **volume**: The file size in kilobytes (e.g., `1024` for 1 MB).
  - **route**: The directory path where the honeypot file will be stored.

## Alerting System

RansomGuard provides flexible alerting options utelizing IFTTT service:

1. **Email Alerts**: Notifies administrators through email. SMTP settings should be configured in the YAML file.
2. **Slack Notifications**: Uses a Slack webhook to post alert messages in a specified channel.
3. **Custom Webhook**: Configurable webhook integration for custom alerting solutions.

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature/YourFeature`).
3. Commit your changes (`git commit -m 'Add YourFeature'`).
4. Push the branch (`git push origin feature/YourFeature`).
5. Open a pull request.

## License

This project is licensed under the MIT License.

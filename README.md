# Console Tool

A GitCall-compatible console application that enables running commands and managing files through Corezoid platform integration.

## Overview

This tool serves as a bridge between the Corezoid GitCall system and various command-line utilities. 
## Features

- **File Management**: Downloads input files from provided URLs before command execution
- **Command Execution**: Runs specified commands with arguments in isolated temporary directories
- **Result Upload**: Automatically uploads generated files back to the platform
- **Security**: Executes commands in sandboxed temporary directories with proper cleanup

## API Format

### Request Structure
```json
{
  "upload_url": "https://mw-api.simulator.company/v/1.0/upload/...",
  "access_token": "atn_...",
  "files": [
    {
      "file_link": "https://...",
      "file_name": "input.mp3"
    }
  ],
  "command": "ls",
  "args": ["-lah"]
}
```

- **`upload_url`** (string, required): The API endpoint URL where generated files will be uploaded after command execution
- **`access_token`** (string, required): Bearer token for authentication with the upload API
- **`files`** (array, optional): List of input files to download before command execution
    - **`file_link`** (string): Direct download URL for the file
    - **`file_name`** (string): Name to save the file as in the working directory
- **`command`** (string, required): The command to execute (e.g., "ls", "ffmpeg", "python")
- **`args`** (array, required): Array of command-line arguments to pass to the command

### Response Structure
```json
{
  "result": "command output here...",
  "status": "success",
  "uploaded_files": [
    {
      "file_link": "https://...",
      "file_name": "output.mp3",
      "size": "2196160"
    }
  ]
}
```

- **`result`** (string): Complete output from the executed command (stdout + stderr)
- **`status`** (string): Execution status, typically "success" for successful operations
- **`uploaded_files`** (array): List of files that were generated 
  - **`file_link`** (string): Download URL for the uploaded file
  - **`file_name`** (string): Name of the uploaded file
  - **`size`** (string): File size in bytes

## GitCall Configuration

To configure Corezoid GitCall use the provided `Dockerfile` as your template. 




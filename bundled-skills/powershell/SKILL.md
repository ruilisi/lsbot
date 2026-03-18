---
name: powershell
description: Run PowerShell (pwsh) commands and scripts. Use when the user asks to automate Windows tasks, manage files/processes/services, query system info, or run .ps1 scripts. Also covers cross-platform pwsh on macOS/Linux.
metadata:
  {
    "openclaw":
      {
        "emoji": "🖥️",
        "requires": { "bins": ["pwsh"] },
        "install":
          [
            {
              "id": "brew",
              "kind": "brew",
              "formula": "powershell",
              "bins": ["pwsh"],
              "label": "Install PowerShell (brew)",
            },
          ],
      },
  }
---

# PowerShell

Use `pwsh` (PowerShell Core 7+) for all commands — not `powershell.exe` (Windows PowerShell 5).

Run a one-liner:

```bash
pwsh -Command "Get-Date"
```

Run a script file:

```bash
pwsh -File ./script.ps1
```

Run with no profile (faster, avoids profile side-effects):

```bash
pwsh -NoProfile -Command "..."
```

## Variables and Output

```powershell
$name = "world"
Write-Output "Hello, $name"

# Suppress output
$null = Some-Command

# Capture output
$result = Get-Process | Where-Object { $_.CPU -gt 10 }
```

## Files and Directories

```powershell
# List files
Get-ChildItem -Path C:\Users -Recurse -Filter "*.log"
Get-ChildItem | Sort-Object LastWriteTime -Descending | Select-Object -First 10

# Copy / Move / Delete
Copy-Item src.txt dst.txt
Move-Item old.txt new.txt
Remove-Item file.txt
Remove-Item folder -Recurse -Force

# Read / Write files
Get-Content file.txt
Set-Content file.txt "content"
Add-Content file.txt "appended line"
"line1", "line2" | Out-File output.txt -Encoding UTF8
```

## Processes

```powershell
# List processes
Get-Process
Get-Process -Name chrome

# Kill a process
Stop-Process -Name notepad
Stop-Process -Id 1234

# Start a process
Start-Process notepad.exe
Start-Process pwsh -ArgumentList "-File script.ps1" -Wait
```

## Services (Windows)

```powershell
Get-Service
Get-Service -Name wuauserv
Start-Service -Name Spooler
Stop-Service -Name Spooler
Restart-Service -Name Spooler
Set-Service -Name Spooler -StartupType Automatic
```

## System Information

```powershell
# OS / hardware
Get-ComputerInfo | Select-Object WindowsVersion, TotalPhysicalMemory
[System.Environment]::OSVersion
$env:COMPUTERNAME; $env:USERNAME

# Disk usage
Get-PSDrive -PSProvider FileSystem

# Network
Get-NetIPAddress
Test-NetConnection google.com -Port 443
Resolve-DnsName google.com
```

## Registry (Windows)

```powershell
# Read
Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion" -Name ProductName

# Write
Set-ItemProperty -Path "HKCU:\Software\MyApp" -Name "Setting" -Value "Value"

# Create key
New-Item -Path "HKCU:\Software\MyApp" -Force
```

## Networking / HTTP

```powershell
# GET request
$resp = Invoke-WebRequest -Uri "https://api.example.com/data" -UseBasicParsing
$resp.Content

# POST JSON
$body = @{ key = "value" } | ConvertTo-Json
Invoke-RestMethod -Uri "https://api.example.com" -Method POST -Body $body -ContentType "application/json"

# Download file
Invoke-WebRequest -Uri "https://example.com/file.zip" -OutFile "file.zip"
```

## JSON

```powershell
# Parse JSON
$data = '{"name":"Alice","age":30}' | ConvertFrom-Json
$data.name

# Build JSON
@{ name = "Alice"; scores = @(95, 87) } | ConvertTo-Json -Depth 5
```

## Pipelines and Filtering

```powershell
# Filter objects
Get-Process | Where-Object { $_.WorkingSet -gt 100MB }

# Select columns
Get-Process | Select-Object Name, Id, CPU | Sort-Object CPU -Descending

# Group and count
Get-ChildItem -Recurse | Group-Object Extension | Sort-Object Count -Descending

# First / Last
Get-EventLog -LogName System -Newest 20 | Select-Object -First 5
```

## Error Handling

```powershell
try {
    Get-Item "nonexistent" -ErrorAction Stop
} catch {
    Write-Error "Failed: $_"
}

# Suppress non-terminating errors
Get-Item "missing" -ErrorAction SilentlyContinue
```

## Scheduled Tasks (Windows)

```powershell
# List tasks
Get-ScheduledTask | Where-Object { $_.State -eq "Ready" }

# Create task
$action  = New-ScheduledTaskAction -Execute "pwsh.exe" -Argument "-File C:\scripts\backup.ps1"
$trigger = New-ScheduledTaskTrigger -Daily -At "02:00AM"
Register-ScheduledTask -TaskName "NightlyBackup" -Action $action -Trigger $trigger -RunLevel Highest
```

## Execution Policy

```powershell
# Check current policy
Get-ExecutionPolicy

# Allow local scripts (run once as admin)
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

# Bypass for a single run
pwsh -ExecutionPolicy Bypass -File script.ps1
```

## Useful One-liners

```powershell
# Find large files (>100 MB)
Get-ChildItem C:\ -Recurse -ErrorAction SilentlyContinue |
  Where-Object { $_.Length -gt 100MB } |
  Sort-Object Length -Descending |
  Select-Object FullName, @{n="MB";e={[math]::Round($_.Length/1MB,1)}}

# Kill all processes matching a name
Get-Process chrome -ErrorAction SilentlyContinue | Stop-Process -Force

# Get public IP
(Invoke-RestMethod https://api.ipify.org?format=json).ip

# Measure command time
Measure-Command { Get-ChildItem C:\ -Recurse }
```

## Tips

- Use `-ErrorAction SilentlyContinue` to suppress non-fatal errors
- Use `-WhatIf` on destructive commands to preview without executing
- `$_` is the current pipeline object; `$?` is last command success bool
- Prefer `Invoke-RestMethod` over `Invoke-WebRequest` for JSON APIs (auto-parses)
- On macOS/Linux, Windows-only cmdlets (Get-Service, registry) are unavailable

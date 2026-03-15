# Test script for install.sh validation
# Tests basic script structure and logic without actually installing

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$InstallScript = Join-Path $ScriptDir "..\scripts\install.sh"

# Test counter
$TestsPassed = 0
$TestsFailed = 0

function Test-Pattern {
    param(
        [string]$TestName,
        [string]$Pattern,
        [string]$FilePath
    )
    
    $content = Get-Content -Path $FilePath -Raw
    if ($content -match $Pattern) {
        Write-Host "checkmark $TestName" -ForegroundColor Green
        $script:TestsPassed++
    } else {
        Write-Host "X $TestName" -ForegroundColor Red
        $script:TestsFailed++
    }
}

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Testing install.sh script" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: Script exists
if (Test-Path $InstallScript) {
    Write-Host "[PASS] Script file exists" -ForegroundColor Green
    $TestsPassed++
} else {
    Write-Host "[FAIL] Script file exists" -ForegroundColor Red
    $TestsFailed++
    exit 1
}

# Test 2: Script has shebang
Test-Pattern "Script has bash shebang" "^#!/bin/bash" $InstallScript

# Test 3: Script has set -e
Test-Pattern "Script has 'set -e' for error handling" "set -e" $InstallScript

# Test 4: Script checks for root
Test-Pattern "Script checks for root privileges" 'EUID.*-ne 0' $InstallScript

# Test 5: Script detects OS from /etc/os-release
Test-Pattern "Script detects OS from /etc/os-release" "/etc/os-release" $InstallScript

# Test 6-9: Script checks required ports
Test-Pattern "Script checks port 80 availability" "check_port 80" $InstallScript
Test-Pattern "Script checks port 443 availability" "check_port 443" $InstallScript
Test-Pattern "Script checks port 5432 availability" "check_port 5432" $InstallScript
Test-Pattern "Script checks port 8428 availability" "check_port 8428" $InstallScript

# Test 10-11: Script has install functions
Test-Pattern "Script has install_deb function" "install_deb\(\)" $InstallScript
Test-Pattern "Script has install_rpm function" "install_rpm\(\)" $InstallScript

# Test 12-16: Script handles distributions
Test-Pattern "Script handles Ubuntu distribution" "ubuntu\)" $InstallScript
Test-Pattern "Script handles Debian distribution" "debian\)" $InstallScript
Test-Pattern "Script handles RHEL distribution" "rhel" $InstallScript
Test-Pattern "Script handles Rocky Linux distribution" "rocky" $InstallScript
Test-Pattern "Script handles AlmaLinux distribution" "almalinux" $InstallScript

# Test 17: Script shows Docker instructions
Test-Pattern "Script has Docker installation instructions" "docker-compose" $InstallScript

# Test 18-19: Script installs dependencies
Test-Pattern "Script installs postgresql dependency" "postgresql" $InstallScript
Test-Pattern "Script installs systemd dependency" "systemd" $InstallScript

# Test 20-21: Script downloads packages
Test-Pattern "Script downloads .deb package" "infrasense_amd64\.deb" $InstallScript
Test-Pattern "Script downloads .rpm package" "infrasense_x86_64\.rpm" $InstallScript

# Test 22-24: Script prints success information
Test-Pattern "Script prints installation success message" "Installation Complete" $InstallScript
Test-Pattern "Script prints access URL" "Access the dashboard" $InstallScript
Test-Pattern "Script prints admin credentials info" "admin" $InstallScript

# Test 25: Script has error handling
Test-Pattern "Script has error handling with exit codes" "exit 1" $InstallScript

Write-Host ""
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Test Results" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Passed: $TestsPassed" -ForegroundColor Green
Write-Host "Failed: $TestsFailed" -ForegroundColor Red
Write-Host ""

if ($TestsFailed -eq 0) {
    Write-Host "All tests passed!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "Some tests failed" -ForegroundColor Red
    exit 1
}

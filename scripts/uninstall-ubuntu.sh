#!/bin/bash

# InfraSense Uninstall Script

echo "Uninstalling InfraSense..."

# Stop and disable services
echo "Stopping services..."
sudo systemctl stop infrasense-*
sudo systemctl disable infrasense-*

# Remove systemd units
echo "Removing systemd units..."
sudo rm /etc/systemd/system/infrasense-*
sudo systemctl daemon-reload

# Remove installation directory (optional - prompt user)
read -p "Do you want to remove the installation directory and databases? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Removing files and databases..."
    # Warning: this is destructive
    # sudo rm -rf /opt/infrasense
    # sudo -u postgres dropdb infrasense
fi

echo "Uninstallation complete!"

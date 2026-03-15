#!/bin/bash

# InfraSense Update Script

echo "Updating InfraSense..."

# Pull latest changes
git pull origin main

# Rebuild backend
echo "Rebuilding backend..."
cd ../backend
go build -o ../bin/infrasense-backend ./cmd/server

# Rebuild frontend
echo "Rebuilding frontend..."
cd ../frontend
npm install
npm run build

# Restart services
echo "Restarting services..."
sudo systemctl restart infrasense-*

echo "Update complete!"

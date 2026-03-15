#!/bin/bash

# InfraSense Status Script

echo "Checking InfraSense services status..."

SERVICES=(
    "infrasense-backend"
    "infrasense-frontend"
    "infrasense-notification"
    "infrasense-ipmi-collector"
    "infrasense-snmp-collector"
)

for service in "${SERVICES[@]}"; do
    if systemctl is-active --quiet "$service"; then
        echo -e "[ \e[32mOK\e[0m ] $service is running"
    else
        echo -e "[ \e[31mFAIL\e[0m ] $service is not running"
    fi
done

echo "Checking Docker containers..."
docker ps --filter "name=infrasense"

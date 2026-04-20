#!/bin/bash

CONFIG_FILE="${1}"
BACKUP_FILE="${CONFIG_FILE}.bak"

if [ -f "$BACKUP_FILE" ]; then
  echo "Restoring original config from backup"
  mv "$BACKUP_FILE" "$CONFIG_FILE"
else
  echo "No backup found for $CONFIG_FILE"
  exit 1
fi

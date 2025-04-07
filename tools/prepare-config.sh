#!/bin/bash

CONFIG_FILE="${1}"
BACKUP_FILE="${CONFIG_FILE}.bak"

# Backup the original config if not already backed up
if [ ! -f "$BACKUP_FILE" ]; then
  echo "Creating backup at $BACKUP_FILE"
  cp "$CONFIG_FILE" "$BACKUP_FILE"
else
  echo "Backup already exists at $BACKUP_FILE"
fi

echo "Transforming config file: ${CONFIG_FILE}"

jq 'to_entries | map_values({ (.key): ("$" + .key) }) | reduce .[] as $item ({}; . + $item)' \
  "$CONFIG_FILE" > "${CONFIG_FILE}.tmp" && mv "${CONFIG_FILE}.tmp" "$CONFIG_FILE"

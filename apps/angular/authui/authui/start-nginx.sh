#!/usr/bin/env bash

# Log the start of the script
echo "$(date '+%Y/%m/%d %H:%M:%S') [info] Script execution started"

export EXISTING_VARS=$(printenv | awk -F= '{print $1}' | sed 's/^/\$/g' | grep -v '^\$_' | paste -sd,);

# Log the environment variables being exported (optional, depending on security/privacy)
echo "$(date '+%Y/%m/%d %H:%M:%S') [info] Using environment variables: ${EXISTING_VARS}"

# Process files in JSFOLDER
for file in $JSFOLDER;
do
  echo "$(date '+%Y/%m/%d %H:%M:%S') [info] Processing file: ${file}"
  envsubst "${EXISTING_VARS}" < "${file}" > "${file}.tmp"
  if [[ $? -ne 0 ]]; then
    echo "$(date '+%Y/%m/%d %H:%M:%S') [error] Failed to process file: ${file}"
    exit 1
  fi
  mv "${file}.tmp" "${file}"
done

# Log before nginx starts
echo "$(date '+%Y/%m/%d %H:%M:%S') [notice] Starting nginx"

# Start nginx
nginx -g 'daemon off;'

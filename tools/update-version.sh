#!/bin/bash

# Script: update-version.sh
# Description: A Bash script that updates a version property in a specified file to a new version number.
# Usage: ./update-version.sh <file_path> <new_version> <project_name>

# Function to update version property in a file
update_version() {
    local file="${1}"
    local new_version="${2}"

    if [ ! -f "${file}" ]; then
        echo "Error: ${file} not found!"
        exit 1
    fi

    # Update version property in the file
    sed -i "s/version\s*=\s*'\(.*\)'/version = '${new_version}'/g" "${file}"
}

# Check if correct number of arguments are provided
if [ "${#}" -lt 3 ]; then
    echo "Usage: ${0} <file_path> <new_version> <project_name>"
    exit 1
fi

file_path="${1}"
new_version="${2}"
project_name="${3}"

update_version "${file_path}" "${new_version}"

git add "${file_path}"
git commit -m "chore(${project_name}): update project version file to version ${new_version}"
git push

echo "[${project_name}]: Updated version to ${new_version} in ${file_path}"
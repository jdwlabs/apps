#!/bin/bash

# Script: update-app.sh
# Description:
# Usage:
git_repository="https://github.com/jdwillmsen/jdw-apps.git"
destination_dir="jdw-apps"

clone_repository() {
  git clone ${git_repository} ${destination_dir}
  cd ${destination_dir}
}

# Function to check if the repository is a Git repository
check_git_repository() {
  if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Error: Not inside a Git repository."
    exit 1
  fi
}

# Function to check if a file exists and is tracked by Git
check_file() {
  local file_path=${1}
  if [ ! -f "${file_path}" ]; then
    echo "Error: File ${file_path} does not exist."
    exit 1
  fi

  if ! git ls-files --error-unmatch "${file_path}" >/dev/null 2>&1; then
    echo "Error: File ${file_path} is not tracked by Git."
    exit 1
  fi
}

# Function to check if there are uncommitted changes in the repository
check_uncommitted_changes() {
  if ! git diff-index --quiet HEAD --; then
    echo "Error: There are uncommitted changes in the repository. Please commit or stash them first."
    exit 1
  fi
}


# Function to update a file in a Git repository
update_file() {
  local file_path=${1}
  local new_version=${2}
  local project_name=${3}

  clone_repository
  check_git_repository
  check_file "${file_path}"
  check_uncommitted_changes

  git checkout HEAD "${file_path}"

  # Replace the app version line in the file
  sed -i "s/^appVersion: .*/appVersion: \"${new_version}\"/" "${file_path}"

  git add ${file_path}
  git commit -m "chore(${project_name}): update app version to version ${new_version}"
  git push

  echo "[${project_name}]: appVersion in ${file_path} updated to ${new_version}."
}

# Check if correct number of arguments are provided
if [ "${#}" -lt 3 ]; then
  echo "Usage: ${0} <version_number> <file_path> <project_name>"
  exit 1
fi

update_file "${1}" "${2}" "${3}"
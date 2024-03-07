#! /bin/bash

# Script: create-version.sh
# Description: Creates a version file with the specified version number at the provided path.
# Usage: ./create-version.sh <version_number> <file_path> <project_name>

# Check if correct number of arguments are provided
if [ "$#" -lt 3 ]; then
  echo "hi"
    echo "Usage: $0 <version_number> <file_path> <project_name>"
    exit 1
fi

# Assign arguments to variables
version=$1
path=$2
project_name=$3

# Create directory if it doesn't exist
mkdir -p $path

# Create version file
file_path="$path/VERSION"
echo "$version" > "$file_path"

git add $file_path
git commit -m "chore($project_name): update project version file to version $version"
git push

echo "[$project_name]: Version file for version $version created at $file_path"
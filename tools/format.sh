#!/bin/bash

# Script: format.sh
# Description: Apply nx formatting for repository.
# Usage: ./format.sh

npx nx format:write

git add .
git commit -m "style: apply nx formatting"
git push

echo "NX formatting applied"
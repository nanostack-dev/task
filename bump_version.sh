#!/bin/bash

# Exit on error
set -e

# Function to increment the patch version
increment_version() {
  version=$1
  base_version=$(echo "$version" | awk -F. '{print $1"."$2}')
  patch_version=$(echo "$version" | awk -F. '{print $3}')
  new_patch_version=$((patch_version + 1))
  echo "$base_version.$new_patch_version"
}

# Ensure the script is run inside a git repository
if [ ! -d ".git" ]; then
  echo "Error: This script must be run inside a Git repository."
  exit 1
fi

# Get the current version tag
current_version=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "Current version: $current_version"

# Increment the version
new_version=$(increment_version "$current_version")
echo "New version: $new_version"

# Commit changes and tag the new version
git add .
git commit -m "Bump version to $new_version"
git tag "$new_version"

echo "Pushing changes to the remote repository..."

git push origin main # Change 'main' to your default branch if necessary
git push origin "$new_version"

echo "Version updated and pushed: $new_version"

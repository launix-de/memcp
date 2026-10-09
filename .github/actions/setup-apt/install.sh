#!/usr/bin/env bash
# Copyright (C) 2026 Carl-Philip Hänsch
# SPDX-License-Identifier: GPL-3.0-or-later
set -euo pipefail
APT_SOURCES_DIRECTORY=${APT_SOURCES_DIRECTORY:-/etc/apt}
INDEX_TIMEOUT_SECONDS=${INDEX_TIMEOUT_SECONDS:-120}
INSTALL_TIMEOUT_SECONDS=${INSTALL_TIMEOUT_SECONDS:-240}
read -r -a packages <<< "${PACKAGES//$'\n'/ }"
if [ "${#packages[@]}" -eq 0 ]; then
  echo '::error title=CI package setup::No packages requested'
  exit 1
fi
for package in "${packages[@]}"; do
  if [[ ! "$package" =~ ^[a-z0-9][a-z0-9+.-]*$ ]]; then
    echo '::error title=CI package setup::Expected package names, not apt options'
    exit 1
  fi
done
# These browser repositories are unrelated to the build dependencies.
sudo rm -f "$APT_SOURCES_DIRECTORY/sources.list.d/google-chrome.list" \
  "$APT_SOURCES_DIRECTORY/sources.list.d/google-chrome.sources"
apt_options=(-o Acquire::Retries=3 -o Acquire::http::Timeout=30 -o Acquire::https::Timeout=30)
update_indexes() {
  timeout --kill-after=10s "$INDEX_TIMEOUT_SECONDS" \
    sudo apt-get "${apt_options[@]}" update --error-on=any
}
archive_selected=false
select_archive_mirror() {
  # A second failure on the recovery mirror remains an infrastructure error.
  if "$archive_selected"; then
    return 1
  fi
  shopt -s nullglob
  source_files=("$APT_SOURCES_DIRECTORY/sources.list" \
    "$APT_SOURCES_DIRECTORY"/sources.list.d/*.list \
    "$APT_SOURCES_DIRECTORY"/sources.list.d/*.sources)
  # Local mirror lists are repository inputs too. Follow the mirror+file
  # references in both legacy and Deb822 sources, retaining their trust fields.
  for source_file in "${source_files[@]}"; do
    if [ -f "$source_file" ]; then
      while IFS= read -r mirror_uri; do
        source_files+=("${mirror_uri#mirror+file:}")
      done < <(grep -oE 'mirror\+file:/[^[:space:]]+' "$source_file" || true)
    fi
  done
  for source_file in "${source_files[@]}"; do
    if [ -f "$source_file" ]; then
      # Preserve suites, components, Signed-By and unrelated repositories.
      sudo sed -i \
        -e 's#http://azure\.archive\.ubuntu\.com/ubuntu#https://archive.ubuntu.com/ubuntu#g' \
        -e 's#https://azure\.archive\.ubuntu\.com/ubuntu#https://archive.ubuntu.com/ubuntu#g' \
        "$source_file" || return 1
    fi
  done
  archive_selected=true
}
install_packages() {
  timeout --kill-after=10s "$INSTALL_TIMEOUT_SECONDS" \
    sudo apt-get "${apt_options[@]}" install -y "${packages[@]}"
}
if ! update_indexes; then
  echo '::warning title=CI package setup::APT index download failed; retrying once with the Ubuntu archive mirror'
  select_archive_mirror
  if ! update_indexes; then
    echo '::error title=CI package setup::INFRASTRUCTURE_FAILURE: package indexes unavailable after bounded recovery; build/tests have not started'
    exit 1
  fi
fi
if install_packages; then
  exit 0
else
  install_status=$?
fi
# Dependency or configuration errors must not become a generic retry. GNU
# timeout's status alone permits this one bounded mirror recovery.
if [[ "$install_status" == 124 || "$install_status" == 137 ]] && select_archive_mirror; then
  echo '::warning title=CI package setup::APT package installation timed out; retrying once with the Ubuntu archive mirror'
  if update_indexes && install_packages; then
    exit 0
  fi
fi
echo '::error title=CI package setup::INFRASTRUCTURE_FAILURE: required package installation failed; build/tests have not started'
exit 1

#!/usr/bin/env bash

# Copyright © 2020 The Homeport Team
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

if ! hash curl 2>/dev/null; then
  echo "Required tool curl is not installed."
  exit 1
fi

if [[ "$(uname -m)" != "x86_64" ]]; then
  echo -e "Unsupported machine type \\033[1m$(uname -m)\\033[0m: Please check \\033[4;94mhttps://api.github.com/repos/homeport/build-load/releases\\033[0m manually"
  exit 1
fi

if [[ $# -eq 0 ]]; then
  if ! hash jq 2>/dev/null; then
    echo -e 'Required tool \033[1mjq\033[0m is not installed.'
    exit 1
  fi

  # Find the latest version using the GitHub API
  SELECTED_TAG="$(curl --silent --location https://api.github.com/repos/homeport/build-load/releases | jq --raw-output '.[0].tag_name')"
else
  # Use provided argument as tag to download
  SELECTED_TAG="$1"
fi

SYSTEM_UNAME="$(uname | tr '[:upper:]' '[:lower:]')"

# Find a suitable install location
if [[ -w /usr/local/bin ]]; then
  TARGET_DIR=/usr/local/bin

elif [[ -w "$HOME/bin" ]] && grep -q -e "$HOME/bin" -e '\~/bin' <<<"$PATH"; then
  TARGET_DIR=$HOME/bin

else
  echo -e "Unable to determine a writable install location. Make sure that you have write access to either \\033[1m/usr/local/bin\\033[0m or \\033[1m$HOME/bin\\033[0m and that is in your PATH."
  exit 1
fi

# Download and install
case "${SYSTEM_UNAME}" in
  darwin | linux)
    DOWNLOAD_URI="https://github.com/homeport/build-load/releases/download/${SELECTED_TAG}/build-load-${SYSTEM_UNAME}-amd64"

    echo -e "Downloading \\033[4;94m${DOWNLOAD_URI}\\033[0m to place it into \\033[1m${TARGET_DIR}\\033[0m"
    if curl --progress-bar --location "${DOWNLOAD_URI}" --output "${TARGET_DIR}/build-load"; then
      chmod a+rx "${TARGET_DIR}/build-load"
      echo -e "\\nSuccessfully installed \\033[1mbuild-load\\033[0m $(build-load version) into \\033[1m${TARGET_DIR}\\033[0m\\n"
    fi
    ;;

  *)
    echo "Unsupported operating system: ${SYSTEM_UNAME}"
    exit 1
    ;;
esac

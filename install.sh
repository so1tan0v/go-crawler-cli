#!/bin/sh

set -e

VERSION="v1.0.2"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=amd64

URL="https://github.com/so1tan0v/go-crawler-cli/releases/download/$VERSION/so1-crawler-${OS}-${ARCH}"
sudo curl -L $URL -o /usr/local/bin/so1-crawler
sudo chmod +x /usr/local/bin/so1-crawler
#!/usr/bin/env bash
# Use the root deploy.sh to deploy everything, or just the website:
#
#   ./deploy.sh           -- build binaries + deploy website
#   ./deploy.sh --infra   -- provision infra + build + deploy website
#
# This script exists for reference; the root deploy.sh is the entry point.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${SCRIPT_DIR}/../deploy.sh" "$@"

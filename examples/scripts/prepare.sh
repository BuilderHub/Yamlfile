#!/bin/sh
set -e
echo "=== prepare script (loaded via run.script, mounted read-only, never in final image layers) ==="
echo "prepared-at=$(date -u +%Y-%m-%dT%H:%M:%SZ)" > /prep.txt
echo "arch=$(uname -m)" >> /prep.txt
cat /prep.txt

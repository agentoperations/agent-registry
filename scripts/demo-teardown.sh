#!/usr/bin/env bash
# Cleanup after showtime recording
pkill -f "agentctl server start --port 8585" 2>/dev/null || true
rm -f /tmp/demo-showtime.db
rm -rf /tmp/demo-artifacts
echo "Cleaned up."

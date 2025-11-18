#!/bin/bash

# KCP Protocol Test Script

echo "=== Testing KCP Protocol ==="
echo ""

# Start KCP server in background
echo "Starting KCP server..."
./build/iperf-go -s -proto kcp > /tmp/kcp_server.log 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"

# Wait for server to start
sleep 2

# Run KCP client
echo ""
echo "Running KCP client test (3 seconds)..."
./build/iperf-go -c 127.0.0.1 -proto kcp -d 3 -i 1000

# Wait a bit for cleanup
sleep 1

# Kill server
echo ""
echo "Stopping server..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo ""
echo "=== KCP Test Complete ==="

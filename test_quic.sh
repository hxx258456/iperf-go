#!/bin/bash

# QUIC Protocol Test Script

echo "=== Testing QUIC Protocol ==="
echo ""

# Start QUIC server in background
echo "Starting QUIC server..."
./build/iperf-go -s -proto quic > /tmp/quic_server.log 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"

# Wait for server to start
sleep 2

# Run QUIC client
echo ""
echo "Running QUIC client test (3 seconds)..."
./build/iperf-go -c 127.0.0.1 -proto quic -d 3 -i 1000

# Wait a bit for cleanup
sleep 1

# Kill server
echo ""
echo "Stopping server..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo ""
echo "=== QUIC Test Complete ==="

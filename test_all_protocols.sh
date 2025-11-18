#!/bin/bash

echo "============================================"
echo "Testing all protocols for process exit issue"
echo "============================================"

test_protocol() {
    PROTO=$1
    echo ""
    echo "=== Testing $PROTO ==="
    killall -9 iperf-go 2>/dev/null
    sleep 1

    ./build/iperf-go -s -proto $PROTO &
    SERVER_PID=$!
    sleep 2

    ./build/iperf-go -c 127.0.0.1 -proto $PROTO -d 3 -i 1000
    CLIENT_EXIT=$?

    sleep 2

    if ps -p $SERVER_PID > /dev/null 2>&1; then
        echo "❌ $PROTO: Server process still running (PID: $SERVER_PID)"
        killall -9 iperf-go 2>/dev/null
        return 1
    else
        echo "✓ $PROTO: All processes exited successfully"
        return 0
    fi
}

# Test all protocols
test_protocol "tcp"
TCP_RESULT=$?

test_protocol "kcp"
KCP_RESULT=$?

test_protocol "quic"
QUIC_RESULT=$?

echo ""
echo "============================================"
echo "Test Summary:"
echo "============================================"
[ $TCP_RESULT -eq 0 ] && echo "✓ TCP: PASS" || echo "❌ TCP: FAIL"
[ $KCP_RESULT -eq 0 ] && echo "✓ KCP: PASS" || echo "❌ KCP: FAIL"
[ $QUIC_RESULT -eq 0 ] && echo "✓ QUIC: PASS" || echo "❌ QUIC: FAIL"

if [ $TCP_RESULT -eq 0 ] && [ $KCP_RESULT -eq 0 ] && [ $QUIC_RESULT -eq 0 ]; then
    echo ""
    echo "✅ ALL TESTS PASSED! Client processes exit correctly for all protocols."
    exit 0
else
    echo ""
    echo "❌ SOME TESTS FAILED"
    exit 1
fi

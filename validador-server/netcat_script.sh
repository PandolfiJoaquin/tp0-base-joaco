echo "Starting health-check"

if [ "$(echo "ping" | nc -w server 12345)" = "ping" ]; then
    exit 0
fi
exit 1

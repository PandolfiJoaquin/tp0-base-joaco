echo "Starting health-check"

if [ "$(echo "ping" | nc server 12345)" = "ping" ]; then
    exit 0
fi
exit 1

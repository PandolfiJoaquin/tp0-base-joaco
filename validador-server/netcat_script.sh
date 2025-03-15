echo "Starting health-check"

if [ "$(echo "ping" | nc server 12345)" = "ping" ]; then
    echo "Match"
    exit 0
fi
echo "No match"
exit 1

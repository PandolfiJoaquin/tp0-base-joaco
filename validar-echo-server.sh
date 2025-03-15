#build image for script
docker build -f ./validador-server/Dockerfile -t "validator:latest" ./validador-server
#run container, script, and check exit code
docker run --network tp0_testing_net validator:latest 2>&1 ./validador-server/docker-run-logs.txt
exit_code=$?
if [ $exit_code -eq 0 ]; then
   echo "action: test_echo_server | result: success"
else
    echo echo "action: test_echo_server | result: fail"
fi
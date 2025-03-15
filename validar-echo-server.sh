#build image for script
docker build -f ./validador-server/Dockerfile -t "validator:latest" ./validador-server
#run container and run script on it
docker run --network tp0_testing_net validator:latest

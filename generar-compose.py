import sys

HEADER='''
name: tp0
services:
'''

SERVER_SERVICE='''

  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL=DEBUG
    networks:
      - testing_net
'''

CLIENT_SERVICE='''
  client<client-number>:
    container_name: client<client-number>
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=<client-number>
      - CLI_LOG_LEVEL=DEBUG
    networks:
      - testing_net
    depends_on:
      - server
'''

NETWORKS='''
networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
'''


def main(args):
    if len(args) != 1:
        print("Not enough arguments")
        correctAmountOfArguments(args)
        return
    
    try:
        amtOfClients = int(args[0])
    except ValueError:
        print("amount of clients should be alphanumeric")
    
    dockerFile = HEADER + SERVER_SERVICE
    dockerFile += '\n'.join([CLIENT_SERVICE.replace('<client-number>', str(i+1)) for i in range(amtOfClients) ])
    dockerFile += NETWORKS

    print(dockerFile)

def correctAmountOfArguments(args):
    print("expected python args: <amount-of-clients>")
    print(f"given: {args}")



main(sys.argv[1:])
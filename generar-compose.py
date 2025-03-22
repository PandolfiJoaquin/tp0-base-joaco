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
    networks:
      - testing_net
    volumes:
      - type: bind
        source: ./server/config.ini
        target: /config.ini
'''

CLIENT_SERVICE='''
  client<client-number>:
    container_name: client<client-number>
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=<client-number>
      - CLI_BET_NAME=gyro
      - CLI_BET_SURNAME=zeppelli
      - CLI_BET_DNI=12345678
      - CLI_BET_BIRTHDATE=1886-08-24
      - CLI_BET_NUMBER=4815162342
    
    volumes:
      - type: bind
        source: ./client/config.yaml
        target: /config.yaml
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
import logging
from common import utils

def recv_int_2_bytes(client_sock):
    return int.from_bytes(recvall(client_sock, 2),"little")


def recv_string(client_sock, ctx=None):
    suffix = f"| ctx: {ctx}" if ctx is not None else ""
    l = recv_int_2_bytes(client_sock)
    d = recvall(client_sock, l).decode('utf-8')
    return d


def recv_int_one_byte(client_sock):
    return int.from_bytes(recvall(client_sock, 8), "little")


def send_results(client_sock, agency_id):

    #get the results
    winnersForAgency = get_results(agency_id)
    #send the results
    if len(winnersForAgency) == 0:
        winnersForAgency = ["no-winner-on-this-agency"]
    amt_of_winners = len(winnersForAgency)

    logging.info(f"sending the amount of winners")
    client_sock.sendall(amt_of_winners.to_bytes(1, "little"))
    if winnersForAgency[0] == "no-winner-on-this-agency":
        send_string(client_sock, winnersForAgency[0])
        logging.info(f"sending that there is no winner")
        return
    for bet in winnersForAgency:
        logging.info(f"sending dni of winner")
        send_string(client_sock, str(bet.document))


def respond_pending(client_sock, agency_id):
    logging.debug("results are not ready yet. waiting for agency")
    logging.debug("reading agency info")
    logging.debug(f"responding results not done to {agency_id}")
    ackear(client_sock)
    logging.debug("client notified about delay")


def recv_bet(client_sock):
    msg_code = client_sock.recv(1)[0]
    if msg_code != 1:
        logging.info(f"invalid bet recv: {msg_code}")
        return "Invalid msg"


    name = recv_string(client_sock, ctx="name")
    surname = recv_string(client_sock, ctx="surname")
    dni = recv_string(client_sock, ctx="dni")
    birthdate = recv_string(client_sock, ctx="birthdate")
    bet_number = recv_string(client_sock, ctx="bet_number")
    agency = recv_string(client_sock, ctx="agency")
    bet = utils.Bet(
        agency=agency,
        first_name=name,
        last_name=surname,
        document=dni,
        birthdate=birthdate,
        number=bet_number
    )

    return bet


def recv_batch(client_sock, t):
    if t != 2:
        logging.info(f"batches are not being used. {t}")
        return [recv_bet(client_sock)]
    batch = []
    amt_of_bets = recv_int_2_bytes(client_sock)
    try:
        for i in range(amt_of_bets):
            batch.append(recv_bet(client_sock))

        return batch
    except ConnectionError as e:
        logging.error(f"action: apuesta_recibida | result: fail | cantidad: {amt_of_bets}.")
        client_sock.settimeout(5)
        client_sock.sendall(b'\x00') #send error on batch
        return []

def get_results(agency):
    logging.debug("getting bets")
    return [bet for bet in utils.load_bets()
            if utils.has_won(bet) and bet.agency == agency]


def send_string(client_sock, string):
    l = len(string)
    client_sock.sendall(l.to_bytes(2, "little"))
    client_sock.sendall(string.encode('utf-8'))

def ackear(client_sock):
    client_sock.settimeout(5)
    client_sock.sendall(b'\x00')
    client_sock.settimeout(None)

def recvall(skt, n):
    data = bytearray()
    while len(data) < n:
        packet = skt.recv(n - len(data))
        if not packet:
            raise ConnectionError("Connection closed before receiving all data")
        data.extend(packet)
    return bytes(data)
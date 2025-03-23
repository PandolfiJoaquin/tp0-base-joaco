import socket
import logging
import signal
import common.utils as utils


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        signal.signal(signal.SIGTERM, self.__sigterm_handler)
        signal.signal(signal.SIGINT, self.__sigterm_handler)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.client_has_not_ended = False

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        self.running = True
        while self.running:
            
            client_sock = self.__accept_new_connection()
            self.__handle_client_connection(client_sock)

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        self.client_has_not_ended = True
        try:
            while self.client_has_not_ended:
                t_raw = recvall(client_sock, 1)
                logging.info(f"t_raw: {t_raw}")
                t = int.from_bytes(t_raw, "little")
                if t == 1:
                    logging.info(f"got a bet instead of a batch")
                if t == 2:
                    logging.info("action: receive_batch | result: on_progress")
                    batch = self.__recv_batch(client_sock)
                    logging.info(f"action: recieve_batch | result: success | amt of bets: {len(batch)}")
                    for bet in batch:
                        utils.store_bets([bet])
                    client_sock.settimeout(5)
                    client_sock.sendall(b'\x00')
                    client_sock.settimeout(0)
                    continue
                self.client_has_not_ended = False
                
                    
                
                
                addr = client_sock.getpeername()

                

        except OSError as e:
            logging.error("action: receive_message | result: fail | error: {e}")
        finally:
            client_sock.shutdown(socket.SHUT_WR)
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

    def __sigterm_handler(self, sig, frame):
        self._server_socket.shutdown(socket.SHUT_RDWR)
        self._server_socket.close()
        self.running = False
        exit(0)


    def __recv_batch(self, client_sock,t):
        if t != 2:
            logging.info(f"batches are not being used. {t}")
            return [self.__recv_bet(client_sock)]
        batch = []
        amt_of_bets = self.recv_int_2_bytes(client_sock)
        logging.debug(f"{amt_of_bets}")
        for i in range(amt_of_bets):
            batch.append(self.__recv_bet(client_sock))
            logging.info(f"received {i+1}/{4} bets")
        
        logging.info(f"all bets have been received")
        
        return batch

    def __recv_bet(self, client_sock):
        msg_code = client_sock.recv(1)[0]
        if msg_code != 1:
            logging.info(f"invalid bet recv: {msg_code}")
            return "Invalid msg"
    
        
        name = self.__recv_string(client_sock, ctx="name")
        surname = self.__recv_string(client_sock, ctx="surname")
        dni = self.__recv_string(client_sock, ctx="dni")
        birthdate = self.__recv_string(client_sock, ctx="birthdate")
        bet_number = self.__recv_string(client_sock, ctx="bet_number")
        agency = self.__recv_string(client_sock, ctx="agency")
        bet = utils.Bet(
            agency=agency,
            first_name=name,
            last_name=surname,
            document=dni,
            birthdate=birthdate,
            number=bet_number
        )
        logging.debug(f"Bet(agency={bet.agency}, first_name={bet.first_name}, last_name={bet.last_name}, document={bet.document}, birthdate={bet.birthdate}, number={bet.number})")

        
        return bet
        

    def __recv_int_one_byte(self, client_sock):
        return int.from_bytes(recvall(client_sock, 8), "little")
    
    def __recv_string(self, client_sock, ctx=None):
        suffix = f"| ctx: {ctx}" if ctx is not None else ""
        l = self.recv_int_2_bytes(client_sock)
        #logging.debug(f"now readingd {l} bytes" + suffix)
        d = recvall(client_sock, l).decode('utf-8')
        #logging.debug(f"content: {d}" + suffix)
        return d

    def recv_int_2_bytes(self, client_sock):
        return int.from_bytes(recvall(client_sock, 2),"little")

def recvall(skt, n):
    data = bytearray()
    while len(data) < n:
        packet = skt.recv(n - len(data))
        if not packet:
            raise ConnectionError("Connection closed before receiving all data")
        data.extend(packet)
    return bytes(data)


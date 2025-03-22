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
        try:
            bet = self.__recv_bet(client_sock)
            addr = client_sock.getpeername()
            #logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: {bet}')
            utils.store_bets([bet])
            logging.info(f"action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}")
            client_sock.settimeout(5)
            client_sock.sendall(b'\x00')

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


    def __recv_bet(self, client_sock):
        msg_code = client_sock.recv(1)[0]
        if msg_code != 1:
            return "Invalid msg"
        
        name = self.__recv_string(client_sock)
        surname = self.__recv_string(client_sock)
        dni = self.__recv_int(client_sock)
        birthdate = self.__recv_string(client_sock)
        bet_number = self.__recv_int(client_sock)
        agency = self.__recv_int(client_sock)
        
        return utils.Bet(
            agency=agency,
            first_name=name,
            last_name=surname,
            document=dni,
            birthdate=birthdate,
            number=bet_number
        )
        

    def __recv_int(self, client_sock):
        return int.from_bytes(recvall(client_sock, 8), "little")
    
    def __recv_string(self, client_sock):
        l = int.from_bytes(recvall(client_sock, 2),"little")
        logging.debug(f"now readingd {l} bytes")
        d = recvall(client_sock, l).decode('utf-8')
        logging.debug(f"content: {d}")
        return d

def recvall(skt, n):
    data = bytearray()
    while len(data) < n:
        packet = skt.recv(n - len(data))
        if not packet:
            raise ConnectionError("Connection closed before receiving all data")
        data.extend(packet)
    return bytes(data)


import socket
import logging
import signal



from multiprocessing import Process, Manager

from common import utils
from common.protocol import *


class Server:
    def __init__(self, port, listen_backlog, config_params):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        signal.signal(signal.SIGTERM, self.__sigterm_handler)
        signal.signal(signal.SIGINT, self.__sigterm_handler)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.clients = int(config_params["client_amount"])
        self.running = False
        self.skts = []



    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        self.running = True
        with Manager() as manager:
            processes = []
            file_lock = manager.Lock()
            agencies_done_lock = manager.Lock()
            agencies_done = manager.list()
            while self.running:
                try:
                    client_sock = self.__accept_new_connection()
                    # start this in a process
                    processes.append(Process(
                        target=self.__handle_client_connection,
                        args=(client_sock, file_lock, agencies_done_lock, agencies_done)
                    ))
                    processes[-1].start()
                except OSError as e:
                    if self.running:
                        logging.error(f"action: accept_connections | result: fail | error: {e}")
                #add the process to a list
                for p in processes:
                    if not p.is_alive():
                        p.join()

            for p in processes:
                if not p.is_alive():
                    p.join()



    def __handle_client_connection(self, client_sock, file_lock, agencies_done_lock, agencies_done):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        
        addr = client_sock.getpeername()
        try:
            t_raw = recvall(client_sock, 1)
            logging.info(f"t_raw: {t_raw}")
            t = int.from_bytes(t_raw, "little")
            if t == 1:
                logging.info(f"got a bet instead of a batch")

            if t == 2: #batch of bets
                batch = recv_batch(client_sock, t)
                if len(batch) == 0:
                    return
                with file_lock:
                    for bet in batch:
                        utils.store_bets([bet])

                logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(batch)}")
                ackear(client_sock)
            
            if t == 3: #agency done sending bets

                agency_id = int.from_bytes(recvall(client_sock, 1), "little")

                with agencies_done_lock:
                    if len(agencies_done) == self.clients:
                        raise RuntimeError("received too many agencies done messages")
                    agencies_done.append(agency_id)
                    logging.info(f"agency {agency_id} done. agencies left: {set([i + 1 for i in range(self.clients)]) - set(agencies_done)}")
                    ackear(client_sock)
                    if len(agencies_done) == self.clients:
                        logging.info(f"all agencies done")
                    logging.info("action: sorteo | result: success")

            if t == 4:
                agency_id = int.from_bytes(recvall(client_sock, 1), "little")
                with agencies_done_lock:
                    amt_of_agencies_done = len(agencies_done)
                if amt_of_agencies_done != self.clients:
                    respond_pending(client_sock, agency_id)
                else:
                    winnersForAgency = []
                    with file_lock:
                        winnersForAgency = get_results(agency_id)
                    if len(winnersForAgency) == 0:
                        winnersForAgency = ["no-winner-on-this-agency"]
                    send_results(client_sock, agency_id, winnersForAgency)



        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")

        try:
            client_sock.shutdown(socket.SHUT_WR)
            client_sock.close()
        except OSError:
            logging.debug("socket already closed")

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
        self.running = False
        self._server_socket.shutdown(socket.SHUT_RDWR)
        self._server_socket.close()
        for skt in self.skts:
            try:
                skt.shutdown(socket.SHUT_RDWR)
                skt.close()
            except OSError:
                continue

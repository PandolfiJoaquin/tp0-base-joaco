package common

import (
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID              string
	ServerAddress   string
	LoopAmount      int
	LoopPeriod      time.Duration
	MaxBetsPerBatch int
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

type Bet struct {
	Name      string
	Surname   string
	Dni       string
	BirthDate string
	Number    string
	Agency    string
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {

	betsCh := c.loadBets()

	msgDoneCh := make(chan bool)
	errCh := make(chan error)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go c.sendAllBets(msgDoneCh, errCh, betsCh)
	select {
	case <-msgDoneCh:
		log.Infof("action: send_batch | result: success | client_id: %v", c.config.ID)

	case err := <-errCh:
		log.Infof("error received")
		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
		}
		break
	case sig := <-sigChan:
		log.Debugf("action: receive_message | client_id: %v | signal received: %v", c.config.ID, sig)
		if sig == syscall.SIGTERM {
			log.Infof("action: receive_message | result: fail | client_id: %v | signal: %v",
				c.config.ID,
				sig,
			)
			return
		}
	}
}

func (c *Client) sendAllBets(msgDoneCh chan<- bool, errCh chan<- error, bets chan Bet) {
	dataChan := c.protocolSerializeBets(bets)
	for dataToSend := range dataChan {
		err := c.sendData(dataToSend, "batch", true)
		if err != nil {
			errCh <- err
			return
		}
	}

	allDoneMsgSerialized := c.protocolNotifyAgencyDoneMsg()

	if err := c.sendData(allDoneMsgSerialized, "agency done", true); err != nil {
		errCh <- err
		return
	}

	askResults := c.protocolAskResults()
	for {
		log.Infof("creating socket")
		if err := c.createClientSocket(); err != nil {
			errCh <- err
			return
		}
		//ask for data
		log.Infof("asking for data with msg: %v", askResults)
		sended := 0
		for sended < len(askResults) {
			n, err := c.conn.Write(askResults)
			if err != nil {
				errCh <- err
				return
			}
			sended += n
		}
		log.Infof("receiving data")
		//receive data
		buff := make([]byte, 1)
		j := 0
		for j < 1 {

			i, err := c.conn.Read(buff)
			if err != nil {
				errCh <- err
				return
			}
			j += i
		}
		log.Infof("data received: %v", buff)
		if buff[0] == 0 {
			log.Infof("results are not ready yet")
			if err := c.conn.Close(); err != nil {
				errCh <- err
				return
			}
			time.Sleep(c.config.LoopPeriod)
			continue
		}
		log.Infof("results are ready")
		log.Infof("winners to receive: %v", buff[0])
		results, err := c.getResults(buff[0])
		if err != nil {
			errCh <- err
			return
		}
		cantGanadores := len(results)
		if results[0] == "no-winner-on-this-agency" {
			cantGanadores = 0
		}
		log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v", cantGanadores)
		break
	}

	//implement the other side of the protocol in server.py

	msgDoneCh <- true

	return
}

func (c *Client) sendData(dataToSend []byte, ctx string, waitAckAndClose bool) error {
	//create socket
	if err := c.createClientSocket(); err != nil {
		return err
	}
	//send data. retry if necessary
	dataSended := 0
	for dataSended < len(dataToSend) {
		n, err := c.conn.Write(dataToSend)
		if err != nil {
			return err
		}
		dataSended += n
	}
	if !waitAckAndClose {
		return nil
	}

	//receive ack for the batch from the server
	buff := make([]byte, 1)
	if err := c.conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		return err
	}
	n := 0
	for n < 1 {
		i, err := c.conn.Read(buff)
		if err != nil {
			return err
		}
		n += i
	}
	if buff[0] != 0 {
		log.Errorf("Error sending %v. Server response: %v", ctx, buff)
	} /*else {
		log.Debugf("data for %v successfully", ctx)
	}*/

	if err := c.conn.Close(); err != nil {
		log.Errorf("action: close connection | result: failed | client_id: %v | msg: %v",
			c.config.ID,
			err,
		)
	}
	return nil
}

func (c *Client) loadBets() chan Bet {
	ch := make(chan Bet)
	go func(ch chan Bet) {
		file, err := os.Open(fmt.Sprintf("agency-%v.csv", c.config.ID))
		if err != nil {
			panic(err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		for {
			record, err := reader.Read()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				panic(err)
			}
			ch <- Bet{
				Name:      record[0],
				Surname:   record[1],
				Dni:       record[2],
				BirthDate: record[3],
				Number:    record[4],
				Agency:    c.config.ID,
			}
		}
		close(ch)
	}(ch)
	return ch
}

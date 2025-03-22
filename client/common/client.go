package common

import (
	"bytes"
	"encoding/binary"
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
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	bet    Bet
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
func NewClient(config ClientConfig, bet Bet) *Client {
	client := &Client{
		config: config,
		bet:    bet,
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
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {

		msgDoneCh := make(chan bool)
		errCh := make(chan error)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM)

		go c.sendAndReadResponse(msgDoneCh, errCh, c.bet)
		select {
		case <-msgDoneCh:
			break
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

		// Wait a time between sending one message and the next one

	}
}

func (c *Client) sendAndReadResponse(msgDoneCh chan<- bool, errCh chan<- error, bet Bet) {
	if err := c.createClientSocket(); err != nil {
		errCh <- err
		return
	}

	// TODO: Modify the send to avoid short-write
	log.Infof("unserialize bet: %v", bet)
	dataToSend := c.protocolSerializeBet(bet)
	dataSended := 0
	log.Debug(len(dataToSend))
	for dataSended < len(dataToSend) {
		n, err := c.conn.Write(dataToSend)
		if err != nil {
			errCh <- err
			return
		}
		dataSended += n
		log.Debugf("sended: %v bytes. data left: %v bytes",
			n,
			len(dataToSend)-dataSended)
	}
	buff := make([]byte, 1)
	if err := c.conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		errCh <- err
	}
	n := 0
	for n < 1 {
		i, err := c.conn.Write(buff)
		if err != nil {
			errCh <- err
			return
		}
		n += i
	}
	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", bet.Dni, bet.Number)
	time.Sleep(c.config.LoopPeriod)

	msgDoneCh <- true
	if err := c.conn.Close(); err != nil {
		log.Errorf("action: close connection | result: failed | client_id: %v | msg: %v",
			c.config.ID,
			err,
		)
	}

	return
}

func (c *Client) protocolSerializeBet(bet Bet) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint8(1))
	protocolEncodeString(buf, bet.Name)
	protocolEncodeString(buf, bet.Surname)
	//binary.Write(buf, binary.LittleEndian, bet.Dni)
	protocolEncodeString(buf, bet.Dni)
	protocolEncodeString(buf, bet.BirthDate)
	//binary.Write(buf, binary.LittleEndian, bet.Number)
	protocolEncodeString(buf, bet.Number)
	//binary.Write(buf, binary.LittleEndian, bet.Agency)
	protocolEncodeString(buf, bet.Agency)

	return buf.Bytes()

}

func protocolEncodeString(buf *bytes.Buffer, s string) {
	nameBytes := []byte(s)
	binary.Write(buf, binary.LittleEndian, uint16(len(nameBytes))) // Write length
	buf.Write(nameBytes)                                           // Write actual string bytes
}

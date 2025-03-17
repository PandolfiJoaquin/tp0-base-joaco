package common

import (
	"bufio"
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
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
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
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// Create the connection the server in every loop iteration. Send an

		msgDoneCh := make(chan bool)
		errCh := make(chan error)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM)

		go c.sendAndReadResponse(msgID, msgDoneCh, errCh)
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
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

func (c *Client) sendAndReadResponse(msgID int, msgDoneCh chan<- bool, errCh chan<- error) {
	if err := c.createClientSocket(); err != nil {
		errCh <- err
		return
	}

	// TODO: Modify the send to avoid short-write
	if _, err := fmt.Fprintf(
		c.conn,
		"[CLIENT %v] Message N°%v\n",
		c.config.ID,
		msgID,
	); err != nil {
		errCh <- err
		return
	}

	msg, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		errCh <- err
		return
	}
	log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
		c.config.ID,
		msg,
	)
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

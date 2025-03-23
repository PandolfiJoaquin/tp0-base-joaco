package common

import (
	"bytes"
	"encoding/binary"
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

	bets := c.loadBets()

	msgDoneCh := make(chan bool)
	errCh := make(chan error)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go c.sendAllBets(msgDoneCh, errCh, bets)
	select {
	case <-msgDoneCh:
		log.Infof("action: send_batch | result: success | client_id: %v", c.config.ID)
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

func (c *Client) sendAllBets(msgDoneCh chan<- bool, errCh chan<- error, bets []Bet) {
	if err := c.createClientSocket(); err != nil {
		errCh <- err
		return
	}
	dataChan := c.protocolSerializeBets(bets)
	for dataToSend := range dataChan {
		log.Debugf("dataSended: %v", dataToSend)
		dataSended := 0
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
			i, err := c.conn.Read(buff)
			if err != nil {
				errCh <- err
				return
			}
			n += i
		}
		log.Debugf("batch sended successfully", buff)
	}

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

func (c *Client) loadBets() []Bet {
	file, err := os.Open(fmt.Sprintf("agency-%v.csv", c.config.ID))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	bets := make([]Bet, 0)
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			panic(err)
		}
		bets = append(bets, Bet{
			Name:      record[0],
			Surname:   record[1],
			Dni:       record[2],
			BirthDate: record[3],
			Number:    record[4],
			Agency:    c.config.ID,
		})

		// Process the record (a []string)

	}
	return bets
}

// for 2 bytes (uint16) integers
func lengthToBytes(length int) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint16(length))
	return buf.Bytes()
}

func (c *Client) protocolSerializeBets(bets []Bet) chan []byte {
	ch := make(chan []byte)
	go func() {
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, uint8(2))
		t := buf.Bytes()
		betsInBatch := 0
		payload := make([]byte, 0)
		for _, bet := range bets {
			//type length (amount of bets being send) payload (bet1bet2bet3)

			if len(t)+2+len(payload) > 8000 || betsInBatch > 2 /*maxAmount*/ {
				log.Debugf("t: %v, l: %v, payload: %v", len(t), betsInBatch, len(payload))
				log.Debugf("Batch full. sending batch with %v bets", betsInBatch)
				ch <- append(append(t, lengthToBytes(betsInBatch)...), payload...)
				betsInBatch = 0
				payload = make([]byte, 0)

			}
			betBytes := c.protocolSerializeBet(bet)
			payload = append(payload, betBytes...)
			betsInBatch += 1
		}
		if len(payload) > 0 {
			log.Debugf("sending last batch with %v bets", betsInBatch)
			ch <- append(append(t, lengthToBytes(betsInBatch)...), payload...)
		}
		close(ch)
	}()
	return ch
}

func protocolEncodeString(buf *bytes.Buffer, s string) {
	nameBytes := []byte(s)
	binary.Write(buf, binary.LittleEndian, uint16(len(nameBytes))) // Write length
	buf.Write(nameBytes)                                           // Write actual string bytes
}

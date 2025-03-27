package common

import (
	"bytes"
	"encoding/binary"
	"strconv"
)

func protocolEncodeString(buf *bytes.Buffer, s string) {
	nameBytes := []byte(s)
	binary.Write(buf, binary.LittleEndian, uint16(len(nameBytes))) // Write length
	buf.Write(nameBytes)                                           // Write actual string bytes
}

func (c *Client) readString(l uint16) (string, error) { //read a string of length l
	buff := make([]byte, l)
	j := 0
	for j < int(l) {
		i, err := c.conn.Read(buff)
		if err != nil {
			log.Errorf("error reading string")
			return "", err
		}
		j += i
	}
	return string(buff), nil

}

func (c *Client) protocolAskResults() []byte {
	id, _ := strconv.Atoi(c.config.ID)
	log.Infof("agency done. notifying server")
	msg := append([]byte{uint8(4)}, uint8(id))
	return msg
}

func (c *Client) getResults(amtOfWinners uint8) ([]string, error) {
	result := make([]string, 0)
	for i := uint8(0); i < amtOfWinners; i++ {
		//read a 2 bytes number
		lRaw := make([]byte, 2)
		j := 0
		for j < 2 {
			i, err := c.conn.Read(lRaw)
			if err != nil {
				log.Errorf("error lenght")
				return nil, nil
			}
			j += i
		}
		l := binary.LittleEndian.Uint16(lRaw)
		//read the string
		dni, err := c.readString(l)
		if err != nil {
			return nil, err
		}
		result = append(result, dni)

	}
	return result, nil
}

func (c *Client) protocolSerializeBets(bets chan Bet) chan []byte {
	ch := make(chan []byte)
	go func() {
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, uint8(2))
		t := buf.Bytes()
		betsInBatch := 0
		payload := make([]byte, 0)
		for bet := range bets {
			//type length (amount of bets being send) payload (bet1bet2bet3)

			if len(t)+2+len(payload) > 8000 || betsInBatch >= c.config.MaxBetsPerBatch {
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

		//notify the server that the agency is done sending bets
		//string  to uint8

		close(ch)
	}()
	return ch
}

// for 2 bytes (uint16) integers
func lengthToBytes(length int) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint16(length))
	return buf.Bytes()
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

func (c *Client) protocolNotifyAgencyDoneMsg() []byte {
	id, _ := strconv.Atoi(c.config.ID)
	log.Infof("agency done. notifying server")
	allDoneMsgSerialized := append([]byte{uint8(3)}, uint8(id))
	return allDoneMsgSerialized
}

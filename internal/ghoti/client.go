package ghoti

import (
	"bytes"
	"fmt"
	"net"
	"strconv"

	"github.com/fran150/ghoti-sdk-go-v1/internal/config"
)

type GhotiClient struct {
	config config.Config
	conn   net.Conn
}

// TODO: Can I convert the error in a "client error"?
func NewFromConfig(config config.Config) (*GhotiClient, error) {
	conn, err := net.Dial(config.Protocol(), config.Server())
	if err != nil {
		return nil, err
	}

	return &GhotiClient{
		config: config,
		conn:   conn,
	}, nil
}

func (c *GhotiClient) Close() error {
	return c.conn.Close()
}

// Keeps reading from the TCP connection until there is no more data
func (c *GhotiClient) readAll() ([]byte, error) {
	buffer := make([]byte, c.config.ReadBufferSize())

	output := bytes.NewBuffer(nil)

	for {
		chunk, err := c.conn.Read(buffer)
		if err != nil {
			return nil, err
		}

		_, err = output.Write(buffer[:chunk])
		if err != nil {
			return nil, err
		}

		if chunk < c.config.ReadBufferSize() {
			break
		}
	}

	return output.Bytes(), nil
}

func (c *GhotiClient) Read(slot int) (string, error) {
	request := fmt.Sprintf("r%03d\n", slot)
	_, err := c.conn.Write([]byte(request))
	if err != nil {
		return "", err
	}

	response, err := c.readAll()
	if err != nil {
		return "", err
	}

	messages := bytes.Split(response, []byte("\n"))

	for _, message := range messages {
		if message[0] == 'e' {
			return "", fmt.Errorf("error from server: %s", response[1:])
		}

		if message[0] == 'v' {
			updatedSlot, err := strconv.Atoi(string(response[1:4]))
			if err != nil || updatedSlot != slot {
				return "", fmt.Errorf("invalid response from server: %s", response)
			}

			return string(response[4 : len(response)-1]), nil
		}
	}

	return "", fmt.Errorf("invalid response from server: %s", response)
}

func (c *GhotiClient) Write(slot int, data string) error {
	buff := make([]byte, 41)
	_, err := c.conn.Write([]byte(fmt.Sprintf("w%03d%s\n", slot, data)))
	if err != nil {
		return err
	}

	n, err := c.conn.Read(buff)
	if err != nil {
		return err
	}

	value := string(buff[:n])
	if value[0] == 'e' {
		return fmt.Errorf("error from server: %s", value[1:])
	}

	if value[0] == 'v' {
		updatedSlot, err := strconv.Atoi(value[1:4])
		if err != nil || updatedSlot == slot {
			return nil
		} else {
			return fmt.Errorf("invalid response from server: %s", value)
		}
	} else {
		return fmt.Errorf("invalid response from server: %s", value)
	}
}

package ghoti

import (
	"fmt"
	"net"
	"strconv"

	"github.com/fran150/ghoti-sdk-go-v1/internal/config"
)

type GhotiClient struct {
	conn net.Conn
}

// TODO: Can I convert the error in a "client error"?
func NewFromConfig(config *config.Config) (*GhotiClient, error) {
	conn, err := net.Dial(config.Protocol(), config.Server())
	if err != nil {
		return nil, err
	}

	return &GhotiClient{
		conn: conn,
	}, nil
}

func (c *GhotiClient) Close() error {
	return c.conn.Close()
}

func (c *GhotiClient) Read(slot int) (string, error) {
	buff := make([]byte, 40)
	request := fmt.Sprintf("r%03d\n", slot)
	_, err := c.conn.Write([]byte(request))
	if err != nil {
		return "", err
	}

	n, err := c.conn.Read(buff)
	if err != nil {
		return "", err
	}

	value := string(buff[:n])
	if value[0] == 'e' {
		return "", fmt.Errorf("error from server: %s", value[1:])
	}

	if value[0] == 'v' {
		updatedSlot, err := strconv.Atoi(value[1:4])
		if err != nil || updatedSlot == slot {
			return value[4:], nil
		} else {
			return "", fmt.Errorf("invalid response from server: %s", value)
		}
	} else {
		return "", fmt.Errorf("invalid response from server: %s", value)
	}
}

func (c *GhotiClient) Write(slot int, data string) error {
	buff := make([]byte, 40)
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

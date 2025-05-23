package ghoti

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/fran150/ghoti-sdk-go-v1/internal/config"
)

// BroadcastHandler is a function that handles broadcast messages
type BroadcastHandler func(slot int, data string)

// GhotiClient represents a client connection to a Ghoti server
type GhotiClient struct {
	config           config.Config
	conn             net.Conn
	reader           *bufio.Reader
	mutex            sync.Mutex
	pendingRequests  map[int]chan string
	broadcastHandler BroadcastHandler
	done             chan struct{}
	wg               sync.WaitGroup
}

// NewFromConfig creates a new GhotiClient from a configuration
func NewFromConfig(config config.Config) (*GhotiClient, error) {
	conn, err := net.Dial(config.Protocol(), config.Server())
	if err != nil {
		return nil, err
	}

	client := &GhotiClient{
		config:          config,
		conn:            conn,
		reader:          bufio.NewReader(conn),
		pendingRequests: make(map[int]chan string),
		done:            make(chan struct{}),
	}

	// Start the message listener
	client.wg.Add(1)
	go client.listen()

	return client, nil
}

// SetBroadcastHandler sets the handler for broadcast messages
func (c *GhotiClient) SetBroadcastHandler(handler BroadcastHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.broadcastHandler = handler
}

// Close closes the connection to the server
func (c *GhotiClient) Close() error {
	close(c.done)
	c.wg.Wait()
	return c.conn.Close()
}

// listen continuously reads messages from the server and processes them
func (c *GhotiClient) listen() {
	defer c.wg.Done()

	for {
		select {
		case <-c.done:
			return
		default:
			line, err := c.reader.ReadString('\n')
			if err != nil {
				// Connection closed or error
				c.handleError(fmt.Errorf("connection error: %w", err))
				return
			}

			line = strings.TrimSuffix(line, "\n")
			if len(line) == 0 {
				continue
			}

			// Process the message
			c.processMessage(line)
		}
	}
}

// processMessage processes a message received from the server
func (c *GhotiClient) processMessage(message string) {
	if len(message) == 0 {
		return
	}

	messageType := message[0]
	switch messageType {
	case 'v': // Value response
		c.handleValueResponse(message)
	case 'e': // Error response
		c.handleErrorResponse(message)
	case 'a': // Async/broadcast message
		c.handleBroadcastMessage(message)
	default:
		c.handleError(fmt.Errorf("unknown message type: %c", messageType))
	}
}

// handleValueResponse processes a value response from the server
func (c *GhotiClient) handleValueResponse(message string) {
	// Value responses for slot operations have format: v000data
	if len(message) < 4 {
		c.handleError(fmt.Errorf("invalid value response format: %s", message))
		return
	}

	// Extract slot number
	slotStr := message[1:4]
	slot, err := strconv.Atoi(slotStr)
	if err != nil {
		c.handleError(fmt.Errorf("invalid slot number in response: %s", slotStr))
		return
	}

	// Extract data
	data := message[4:]

	// Forward to waiting request if any
	c.mutex.Lock()
	ch, exists := c.pendingRequests[slot]
	c.mutex.Unlock()

	if exists {
		ch <- data
	} else {
		// This could be a response to an auth command
		if strings.HasPrefix(data, c.config.Auth().User()) {
			// This is likely an auth response, ignore it
			return
		}
		// Unexpected response
		c.handleError(fmt.Errorf("received response for slot %d with no pending request", slot))
	}
}

// handleErrorResponse processes an error response from the server
func (c *GhotiClient) handleErrorResponse(message string) {
	// Error responses have format: e000
	if len(message) < 4 {
		c.handleError(fmt.Errorf("invalid error response format: %s", message))
		return
	}

	// For now, we'll just log the error
	// In a real implementation, you might want to forward this to the appropriate request
	fmt.Printf("Server error: %s\n", message[1:])
}

// handleBroadcastMessage processes a broadcast message from the server
func (c *GhotiClient) handleBroadcastMessage(message string) {
	// Broadcast messages have format: a000data
	if len(message) < 4 {
		c.handleError(fmt.Errorf("invalid broadcast message format: %s", message))
		return
	}

	// Extract slot number
	slotStr := message[1:4]
	slot, err := strconv.Atoi(slotStr)
	if err != nil {
		c.handleError(fmt.Errorf("invalid slot number in broadcast: %s", slotStr))
		return
	}

	// Extract data
	data := message[4:]

	// Call the broadcast handler if set
	c.mutex.Lock()
	handler := c.broadcastHandler
	c.mutex.Unlock()

	if handler != nil {
		handler(slot, data)
	}
}

// handleError handles an error in the client
func (c *GhotiClient) handleError(err error) {
	// For critical errors, close the connection
	fmt.Printf("Client error: %v\n", err)
	// In a real implementation, you might want to reconnect or notify the user
}

// Auth authenticates with the server using the configured credentials
func (c *GhotiClient) Auth() error {
	// Send user command
	userCmd := fmt.Sprintf("u%s\n", c.config.Auth().User())
	_, err := c.conn.Write([]byte(userCmd))
	if err != nil {
		return fmt.Errorf("failed to send user command: %w", err)
	}

	// Wait for response (handled by listener)
	// In a real implementation, you might want to wait for a specific response

	// Send password command
	passCmd := fmt.Sprintf("p%s\n", c.config.Auth().Pass())
	_, err = c.conn.Write([]byte(passCmd))
	if err != nil {
		return fmt.Errorf("failed to send password command: %w", err)
	}

	// Wait for response (handled by listener)
	// In a real implementation, you might want to wait for a specific response

	return nil
}

// Read reads the value from a slot
func (c *GhotiClient) Read(slot int) (string, error) {
	if slot < 0 || slot > 999 {
		return "", fmt.Errorf("invalid slot number: %d", slot)
	}

	// Create a channel to receive the response
	responseCh := make(chan string, 1)

	// Register the pending request
	c.mutex.Lock()
	c.pendingRequests[slot] = responseCh
	c.mutex.Unlock()

	// Clean up when done
	defer func() {
		c.mutex.Lock()
		delete(c.pendingRequests, slot)
		c.mutex.Unlock()
	}()

	// Send the read command
	cmd := fmt.Sprintf("r%03d\n", slot)
	_, err := c.conn.Write([]byte(cmd))
	if err != nil {
		return "", fmt.Errorf("failed to send read command: %w", err)
	}

	// Wait for the response
	select {
	case response := <-responseCh:
		return response, nil
	case <-c.done:
		return "", fmt.Errorf("client closed")
	}
}

// Write writes a value to a slot
func (c *GhotiClient) Write(slot int, data string) error {
	if slot < 0 || slot > 999 {
		return fmt.Errorf("invalid slot number: %d", slot)
	}

	if len(data) > 36 {
		return fmt.Errorf("data too long: maximum length is 36 characters")
	}

	// Create a channel to receive the response
	responseCh := make(chan string, 1)

	// Register the pending request
	c.mutex.Lock()
	c.pendingRequests[slot] = responseCh
	c.mutex.Unlock()

	// Clean up when done
	defer func() {
		c.mutex.Lock()
		delete(c.pendingRequests, slot)
		c.mutex.Unlock()
	}()

	// Send the write command
	cmd := fmt.Sprintf("w%03d%s\n", slot, data)
	_, err := c.conn.Write([]byte(cmd))
	if err != nil {
		return fmt.Errorf("failed to send write command: %w", err)
	}

	// Wait for the response
	select {
	case <-responseCh:
		return nil
	case <-c.done:
		return fmt.Errorf("client closed")
	}
}

// Broadcast sends a message to all connected clients
func (c *GhotiClient) Broadcast(slot int, data string) (int, int, int, error) {
	if slot < 0 || slot > 999 {
		return 0, 0, 0, fmt.Errorf("invalid slot number: %d", slot)
	}

	if len(data) > 36 {
		return 0, 0, 0, fmt.Errorf("data too long: maximum length is 36 characters")
	}

	// Create a channel to receive the response
	responseCh := make(chan string, 1)

	// Register the pending request
	c.mutex.Lock()
	c.pendingRequests[slot] = responseCh
	c.mutex.Unlock()

	// Clean up when done
	defer func() {
		c.mutex.Lock()
		delete(c.pendingRequests, slot)
		c.mutex.Unlock()
	}()

	// Send the write command (broadcast uses the write command)
	cmd := fmt.Sprintf("w%03d%s\n", slot, data)
	_, err := c.conn.Write([]byte(cmd))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to send broadcast command: %w", err)
	}

	// Wait for the response
	select {
	case response := <-responseCh:
		// Parse the response format: a/b/c
		parts := strings.Split(response, "/")
		if len(parts) != 3 {
			return 0, 0, 0, fmt.Errorf("invalid broadcast response format: %s", response)
		}

		received, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid received count: %s", parts[0])
		}

		total, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid total count: %s", parts[1])
		}

		failed, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid failed count: %s", parts[2])
		}

		return received, total, failed, nil
	case <-c.done:
		return 0, 0, 0, fmt.Errorf("client closed")
	}
}

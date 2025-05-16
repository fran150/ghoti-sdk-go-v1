package ghoti

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fran150/ghoti-sdk-go-v1/internal/config"
	"github.com/fran150/ghoti-sdk-go-v1/pkg/model"
)

// Response represents a response from the Ghoti server
type Response struct {
	Data  string
	Error error
}

// BroadcastHandler is a function that handles broadcast messages
type BroadcastHandler func(slot int, data string)

// Client represents a client connection to a Ghoti server
type Client struct {
	config           config.Config
	conn             net.Conn
	reader           *bufio.Reader
	mutex            sync.Mutex
	pendingRequests  map[int]chan Response
	broadcastHandler BroadcastHandler
	done             chan struct{}
	wg               sync.WaitGroup
}

// NewClient creates a new Client from a configuration
func NewClient(config config.Config) (*Client, error) {
	conn, err := net.Dial(config.Protocol(), config.Server())
	if err != nil {
		return nil, err
	}

	client := &Client{
		config:          config,
		conn:            conn,
		reader:          bufio.NewReader(conn),
		pendingRequests: make(map[int]chan Response),
		done:            make(chan struct{}),
	}

	// Start the message listener
	client.wg.Add(1)
	go client.listen()

	return client, nil
}

// SetBroadcastHandler sets the handler for broadcast messages
func (c *Client) SetBroadcastHandler(handler BroadcastHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.broadcastHandler = handler
}

// Close closes the connection to the server
func (c *Client) Close() error {
	close(c.done)
	c.wg.Wait()
	return c.conn.Close()
}

// listen continuously reads messages from the server and processes them
func (c *Client) listen() {
	defer c.wg.Done()

	for {
		select {
		case <-c.done:
			return
		default:
			line, err := c.reader.ReadString('\n')
			if err != nil {
				// Connection closed or error
				c.handleFatalError(fmt.Errorf("connection error: %w", err))
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
func (c *Client) processMessage(message string) {
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
		c.handleFatalError(fmt.Errorf("unknown message type: %c", messageType))
	}
}

// handleValueResponse processes a value response from the server
func (c *Client) handleValueResponse(message string) {
	// Special case for auth responses which don't have a slot
	if len(message) > 1 && message[1:] == c.config.Auth().User() {
		// This is an auth response, ignore it
		return
	}

	// Value responses for slot operations have format: v000data
	if len(message) < 4 {
		c.handleFatalError(fmt.Errorf("invalid value response format: %s", message))
		return
	}

	// Extract slot number
	slotStr := message[1:4]
	slot, err := strconv.Atoi(slotStr)
	if err != nil {
		c.handleFatalError(fmt.Errorf("invalid slot number in response: %s", slotStr))
		return
	}

	// Extract data
	data := message[4:]

	// Forward to waiting request if any
	c.mutex.Lock()
	ch, exists := c.pendingRequests[slot]
	c.mutex.Unlock()

	if exists {
		ch <- Response{Data: data}
	} else {
		// Unexpected response
		c.handleFatalError(fmt.Errorf("received response for slot %d with no pending request", slot))
	}
}

// handleErrorResponse processes an error response from the server
func (c *Client) handleErrorResponse(message string) {
	// Error responses have format: e000
	if len(message) < 4 {
		c.handleFatalError(fmt.Errorf("invalid error response format: %s", message))
		return
	}

	errorCode := message[1:4]
	
	// For auth errors, we don't have a slot to forward to
	if errorCode == "004" || errorCode == "005" {
		// Authentication errors, log them
		fmt.Printf("Authentication error: %s\n", errorCode)
		return
	}

	// For other errors, we need to determine which request this is for
	// This is a simplification - in a real implementation, you'd need to track
	// which request this error is for
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Since we don't know which slot this error is for, we'll just forward it to all pending requests
	// In a real implementation, you'd want to be more precise
	for slot, ch := range c.pendingRequests {
		ch <- Response{Error: model.NewGhotiError(errorCode)}
		delete(c.pendingRequests, slot)
	}
}

// handleBroadcastMessage processes a broadcast message from the server
func (c *Client) handleBroadcastMessage(message string) {
	// Broadcast messages have format: a000data
	if len(message) < 4 {
		c.handleFatalError(fmt.Errorf("invalid broadcast message format: %s", message))
		return
	}

	// Extract slot number
	slotStr := message[1:4]
	slot, err := strconv.Atoi(slotStr)
	if err != nil {
		c.handleFatalError(fmt.Errorf("invalid slot number in broadcast: %s", slotStr))
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

// handleFatalError handles a fatal error in the client
func (c *Client) handleFatalError(err error) {
	// For critical errors, close the connection
	fmt.Printf("Fatal client error: %v\n", err)
	c.Close()
}

// Auth authenticates with the server using the configured credentials
func (c *Client) Auth() error {
	// Send user command
	userCmd := fmt.Sprintf("u%s\n", c.config.Auth().User())
	_, err := c.conn.Write([]byte(userCmd))
	if err != nil {
		return fmt.Errorf("failed to send user command: %w", err)
	}

	// Wait a bit for the server to process
	time.Sleep(100 * time.Millisecond)

	// Send password command
	passCmd := fmt.Sprintf("p%s\n", c.config.Auth().Pass())
	_, err = c.conn.Write([]byte(passCmd))
	if err != nil {
		return fmt.Errorf("failed to send password command: %w", err)
	}

	// Wait a bit for the server to process
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Read reads the value from a slot
func (c *Client) Read(slot int) (string, error) {
	if slot < 0 || slot > 999 {
		return "", fmt.Errorf("invalid slot number: %d", slot)
	}

	// Create a channel to receive the response
	responseCh := make(chan Response, 1)

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

	// Wait for the response with a timeout
	select {
	case response := <-responseCh:
		if response.Error != nil {
			return "", response.Error
		}
		return response.Data, nil
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("timeout waiting for response")
	case <-c.done:
		return "", fmt.Errorf("client closed")
	}
}

// Write writes a value to a slot
func (c *Client) Write(slot int, data string) error {
	if slot < 0 || slot > 999 {
		return fmt.Errorf("invalid slot number: %d", slot)
	}

	if len(data) > 36 {
		return fmt.Errorf("data too long: maximum length is 36 characters")
	}

	// Create a channel to receive the response
	responseCh := make(chan Response, 1)

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

	// Wait for the response with a timeout
	select {
	case response := <-responseCh:
		return response.Error
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for response")
	case <-c.done:
		return fmt.Errorf("client closed")
	}
}

// Broadcast sends a message to all connected clients
func (c *Client) Broadcast(slot int, data string) (int, int, int, error) {
	if slot < 0 || slot > 999 {
		return 0, 0, 0, fmt.Errorf("invalid slot number: %d", slot)
	}

	if len(data) > 36 {
		return 0, 0, 0, fmt.Errorf("data too long: maximum length is 36 characters")
	}

	// Create a channel to receive the response
	responseCh := make(chan Response, 1)

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

	// Wait for the response with a timeout
	select {
	case response := <-responseCh:
		if response.Error != nil {
			return 0, 0, 0, response.Error
		}

		// Parse the response format: a/b/c
		parts := strings.Split(response.Data, "/")
		if len(parts) != 3 {
			return 0, 0, 0, fmt.Errorf("invalid broadcast response format: %s", response.Data)
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
	case <-time.After(5 * time.Second):
		return 0, 0, 0, fmt.Errorf("timeout waiting for response")
	case <-c.done:
		return 0, 0, 0, fmt.Errorf("client closed")
	}
}
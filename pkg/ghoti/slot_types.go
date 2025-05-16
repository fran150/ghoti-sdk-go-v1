package ghoti

import (
	"fmt"
	"strconv"
)

// SlotType represents the type of a slot
type SlotType string

const (
	// SimpleMemory is a simple memory slot
	SimpleMemory SlotType = "simple_memory"
	// TimeoutMemory is a memory slot with a timeout
	TimeoutMemory SlotType = "timeout_memory"
	// TokenBucket is a token bucket rate limiter
	TokenBucket SlotType = "token_bucket"
	// LeakyBucket is a leaky bucket rate limiter
	LeakyBucket SlotType = "leaky_bucket"
	// Broadcast is a broadcast signal propagation slot
	Broadcast SlotType = "broadcast"
	// Ticker is a watchdog ticker
	Ticker SlotType = "ticker"
	// AtomicCounter is an atomic counter slot
	AtomicCounter SlotType = "atomic_counter"
)

// SimpleMemorySlot provides methods for interacting with a simple memory slot
type SimpleMemorySlot struct {
	client *Client
	slot   int
}

// Read reads the value from the slot
func (s *SimpleMemorySlot) Read() (string, error) {
	return s.client.Read(s.slot)
}

// Write writes a value to the slot
func (s *SimpleMemorySlot) Write(data string) error {
	return s.client.Write(s.slot, data)
}

// TimeoutMemorySlot provides methods for interacting with a timeout memory slot
type TimeoutMemorySlot struct {
	client *Client
	slot   int
}

// Read reads the value from the slot
func (s *TimeoutMemorySlot) Read() (string, error) {
	return s.client.Read(s.slot)
}

// Write writes a value to the slot
func (s *TimeoutMemorySlot) Write(data string) error {
	return s.client.Write(s.slot, data)
}

// TokenBucketSlot provides methods for interacting with a token bucket slot
type TokenBucketSlot struct {
	client *Client
	slot   int
}

// GetTokens gets tokens from the bucket
func (s *TokenBucketSlot) GetTokens() (int, error) {
	data, err := s.client.Read(s.slot)
	if err != nil {
		return 0, err
	}

	tokens, err := strconv.Atoi(data)
	if err != nil {
		return 0, fmt.Errorf("invalid token count: %s", data)
	}

	return tokens, nil
}

// LeakyBucketSlot provides methods for interacting with a leaky bucket slot
type LeakyBucketSlot struct {
	client *Client
	slot   int
}

// TryAcquire tries to acquire a token from the bucket
func (s *LeakyBucketSlot) TryAcquire() (bool, error) {
	data, err := s.client.Read(s.slot)
	if err != nil {
		return false, err
	}

	result, err := strconv.Atoi(data)
	if err != nil {
		return false, fmt.Errorf("invalid result: %s", data)
	}

	return result == 1, nil
}

// BroadcastSlot provides methods for interacting with a broadcast slot
type BroadcastSlot struct {
	client *Client
	slot   int
}

// Read reads the last value sent to the broadcast slot
func (s *BroadcastSlot) Read() (string, error) {
	return s.client.Read(s.slot)
}

// Send sends a message to all connected clients
func (s *BroadcastSlot) Send(data string) (int, int, int, error) {
	return s.client.Broadcast(s.slot, data)
}

// TickerSlot provides methods for interacting with a ticker slot
type TickerSlot struct {
	client *Client
	slot   int
}

// Read reads the current value of the ticker
func (s *TickerSlot) Read() (int, error) {
	data, err := s.client.Read(s.slot)
	if err != nil {
		return 0, err
	}

	value, err := strconv.Atoi(data)
	if err != nil {
		return 0, fmt.Errorf("invalid ticker value: %s", data)
	}

	return value, nil
}

// Reset resets the ticker to the specified value
func (s *TickerSlot) Reset(value int) error {
	return s.client.Write(s.slot, strconv.Itoa(value))
}

// AtomicCounterSlot provides methods for interacting with an atomic counter slot
type AtomicCounterSlot struct {
	client *Client
	slot   int
}

// Read reads the current value of the counter
func (s *AtomicCounterSlot) Read() (int, error) {
	data, err := s.client.Read(s.slot)
	if err != nil {
		return 0, err
	}

	value, err := strconv.Atoi(data)
	if err != nil {
		return 0, fmt.Errorf("invalid counter value: %s", data)
	}

	return value, nil
}

// Increment increments the counter by the specified value
func (s *AtomicCounterSlot) Increment(value int) error {
	return s.client.Write(s.slot, strconv.Itoa(value))
}

// Decrement decrements the counter by the specified value
func (s *AtomicCounterSlot) Decrement(value int) error {
	return s.client.Write(s.slot, strconv.Itoa(-value))
}

// GetSlot returns a typed slot interface based on the slot type
func (c *Client) GetSlot(slotType SlotType, slot int) (interface{}, error) {
	if slot < 0 || slot > 999 {
		return nil, fmt.Errorf("invalid slot number: %d", slot)
	}

	switch slotType {
	case SimpleMemory:
		return &SimpleMemorySlot{client: c, slot: slot}, nil
	case TimeoutMemory:
		return &TimeoutMemorySlot{client: c, slot: slot}, nil
	case TokenBucket:
		return &TokenBucketSlot{client: c, slot: slot}, nil
	case LeakyBucket:
		return &LeakyBucketSlot{client: c, slot: slot}, nil
	case Broadcast:
		return &BroadcastSlot{client: c, slot: slot}, nil
	case Ticker:
		return &TickerSlot{client: c, slot: slot}, nil
	case AtomicCounter:
		return &AtomicCounterSlot{client: c, slot: slot}, nil
	default:
		return nil, fmt.Errorf("unknown slot type: %s", slotType)
	}
}

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fran150/ghoti-sdk-go-v1/internal/config"
	"github.com/fran150/ghoti-sdk-go-v1/pkg/ghoti"
)

func main() {
	// Load default configuration
	cfg := config.LoadDefaultConfig()

	// Create a new client
	client, err := ghoti.NewClient(cfg)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer client.Close()

	// Set up a broadcast handler
	client.SetBroadcastHandler(func(slot int, data string) {
		fmt.Printf("Received broadcast on slot %d: %s\n", slot, data)
	})

	// Authenticate
	err = client.Auth()
	if err != nil {
		fmt.Printf("Failed to authenticate: %v\n", err)
		return
	}
	fmt.Println("Authentication successful")

	// Get a simple memory slot
	simpleSlot, err := client.GetSlot(ghoti.SimpleMemory, 0)
	if err != nil {
		fmt.Printf("Failed to get slot: %v\n", err)
		return
	}
	
	// Type assertion
	memorySlot, ok := simpleSlot.(*ghoti.SimpleMemorySlot)
	if !ok {
		fmt.Println("Failed to cast to SimpleMemorySlot")
		return
	}

	// Write to the slot
	err = memorySlot.Write("Hello, Ghoti!")
	if err != nil {
		fmt.Printf("Failed to write to slot: %v\n", err)
		return
	}
	fmt.Println("Write successful")

	// Read from the slot
	value, err := memorySlot.Read()
	if err != nil {
		fmt.Printf("Failed to read from slot: %v\n", err)
		return
	}
	fmt.Printf("Read value: %s\n", value)

	// Get a broadcast slot
	broadcastSlot, err := client.GetSlot(ghoti.Broadcast, 1)
	if err != nil {
		fmt.Printf("Failed to get broadcast slot: %v\n", err)
		return
	}
	
	// Type assertion
	bcastSlot, ok := broadcastSlot.(*ghoti.BroadcastSlot)
	if !ok {
		fmt.Println("Failed to cast to BroadcastSlot")
		return
	}

	// Send a broadcast message
	received, total, failed, err := bcastSlot.Send("Broadcast message")
	if err != nil {
		fmt.Printf("Failed to send broadcast: %v\n", err)
		return
	}
	fmt.Printf("Broadcast sent: %d/%d clients received, %d failed\n", received, total, failed)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Client running. Press Ctrl+C to exit.")
	
	// Keep the client running to receive broadcasts
	select {
	case <-sigCh:
		fmt.Println("Shutting down...")
	case <-time.After(60 * time.Second):
		fmt.Println("Timeout reached, shutting down...")
	}
}
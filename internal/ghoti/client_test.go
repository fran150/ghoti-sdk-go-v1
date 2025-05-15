package ghoti

import (
	"testing"

	"github.com/fran150/ghoti-sdk-go-v1/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	cfg := config.LoadDefaultConfig()

	client, err := NewFromConfig(cfg)
	if err != nil {
		t.Errorf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Write(1, "This is a test")
	if err != nil {
		t.Errorf("Failed to write to client: %v", err)
	}

	value, err := client.Read(1)
	if err != nil {
		t.Errorf("Failed to read from client: %v", err)
	}

	assert.Equal(t, "This is a test", value)
}

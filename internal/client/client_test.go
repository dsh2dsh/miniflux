package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient_IncorrectEndpoint(t *testing.T) {
	client := NewClient("incorrect url")
	require.NotNil(t, client)
	_, err := client.Users()
	require.Error(t, err, "Using an incorrect URL should raise an error")

	client = NewClient("")
	require.NotNil(t, client)
	_, err = client.Users()
	require.Error(t, err, "Using an empty URL should raise an error")
}

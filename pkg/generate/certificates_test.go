package generate

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestHasAllNodesInSAN(t *testing.T) {
	// Call certHasAllNodeNames with a set of hosts
	certDir := os.Getenv("PWD") + "/../../testdata"
	result := certHasAllNodeNames([]string{"host1", "host2", "host3"}, "public", certDir)

	// Check if the function returns the expected result
	assert.True(t, result, "certHasAllNodeNames should return true when all hosts are in SAN")

	// Call certHasAllNodeNames with a set of hosts not all in SAN
	result = certHasAllNodeNames([]string{"host1", "host2", "host4"}, "public", certDir)

	// Check if the function returns the expected result
	assert.False(t, result, "certHasAllNodeNames should return false when not all hosts are in SAN")
}

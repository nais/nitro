package generate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasAllNodesInSAN(t *testing.T) {
	// Call certHasAllNodeNames with a set of hosts
	certDir := os.Getenv("PWD") + "/../../testdata"
	result := verifySubjectAltNames([]string{"host1", "host2", "host3"}, "public", certDir)

	// Check if the function returns the expected result
	assert.True(t, result, "certHasAllNodeNames should return true when all hosts are in SAN")

	// Call certHasAllNodeNames with a set of hosts not all in SAN
	result = verifySubjectAltNames([]string{"host1", "host2", "host4"}, "public", certDir)

	// Check if the function returns the expected result
	assert.False(t, result, "certHasAllNodeNames should return false when not all hosts are in SAN")
}

func TestHasOneTooManyInSAN(t *testing.T) {
	certDir := os.Getenv("PWD") + "/../../testdata"
	result := verifySubjectAltNames([]string{"host1", "host2"}, "public", certDir)
	assert.False(t, result, "certHasAllNodeNames should return false when there are too few hosts in SAN")

	result = verifySubjectAltNames([]string{"host1", "host2", "host3", "host4"}, "public", certDir)
	assert.False(t, result, "certHasAllNodeNames should return false when there are too many hosts in SAN")
}

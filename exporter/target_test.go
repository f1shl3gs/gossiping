package exporter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTarget(t *testing.T) {
	target, err := NewTarget("127.0.0.1")
	require.NoError(t, err)

	err = target.Start()
	require.NoError(t, err)
}

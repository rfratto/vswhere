//+build windows

package vswhere

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFind(t *testing.T) {
	timeout, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	installs, err := Find(timeout, WithAll(true))
	require.NoError(t, err)
	require.True(t, len(installs) > 0)
}

func TestGet(t *testing.T) {
	timeout, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	installs, err := Find(timeout, WithAll(true))
	require.NoError(t, err)
	require.True(t, len(installs) > 0)

	for _, install := range installs {
		i, err := Get(timeout, install.InstallationPath)
		require.NoError(t, err)
		require.Equal(t, install, i)
	}
}

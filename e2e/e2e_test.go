package e2e_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/cli/clitest"
	"github.com/coder/coder/pty/ptytest"
)

func TestGoogleCloud(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	root, cfg := clitest.New(t, "server", "--in-memory", "--tunnel", "--address", ":0", "--provisioner-daemons", "1")
	serverErr := make(chan error)
	go func() {
		serverErr <- root.ExecuteContext(ctx)
	}()
	var url string
	require.Eventually(t, func() bool {
		var err error
		url, err = cfg.URL().Read()
		return url != "" && err == nil
	}, 15*time.Second, 25*time.Millisecond)

	err := clitest.NewWithConfig(t, cfg, "login", url, "--username", "example", "--email", "test@coder.com", "--password", "password").Execute()
	require.NoError(t, err)

	tmp := t.TempDir()
	data, err := os.ReadFile("e2e_gcp.tf")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmp, "main.tf"), data, 0600)
	require.NoError(t, err)

	err = clitest.NewWithConfig(t, cfg, "templates", "create", "my-template", "--directory", tmp, "-y").Execute()
	require.NoError(t, err)

	err = clitest.NewWithConfig(t, cfg, "create", "dev", "--template", "my-template", "-y").Execute()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = clitest.NewWithConfig(t, cfg, "delete", "dev", "-y").Execute()
	})

	pty := ptytest.New(t)
	cmd := clitest.NewWithConfig(t, cfg, "ssh", "dev")
	cmd.SetIn(pty.Input())
	cmd.SetErr(pty.Output())
	cmd.SetOut(pty.Output())
	go func() {
		err := cmd.Execute()
		assert.NoError(t, err)
	}()
	// Wait for a prompt!
	pty.ExpectMatchWithTimeout(":~$", 5*time.Minute)
	pty.WriteLine("echo test")
	pty.ExpectMatch("test")
}

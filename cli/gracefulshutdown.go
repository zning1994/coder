package cli

import (
	"bytes"
	"context"
	"fmt"
	"github.com/coder/coder/cli/cliui"
	"github.com/spf13/cobra"
	"io"
	"os"
	"sync"
	"time"
)

type gracefulShutdown struct {
	outBuf []byte
	cmd *cobra.Command
	procedures []func(io.Writer) error
	promptUp bool
	mu sync.Mutex
}

func newGracefulShutdown(cmd *cobra.Command) *gracefulShutdown {
	return &gracefulShutdown{
		cmd:    cmd,
	}
}

func (g *gracefulShutdown) Write(p []byte) (n int, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.outBuf = append(g.outBuf, p...)
	return len(p), nil
}

func (g *gracefulShutdown) flusher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// pass
		}
		g.flushLine()
	}
}

func (g *gracefulShutdown) runProcedures(done chan error) {
	for _, f := range g.procedures {
		if err := f(g); err != nil {
			done <- err
		}
	}
	close(done)
}

func (g *gracefulShutdown) shutdown(stopChan <-chan os.Signal ) error {
	_, _ = fmt.Fprintln(g, "\n\n"+cliui.Styles.Bold.Render("Interrupt caught. Gracefully exiting..."))
	ctx, cancel := context.WithCancel(g.cmd.Context())
	defer cancel()
	go g.flusher(ctx)

	done := make(chan error)
	go g.runProcedures(done)

	return g.waitForInterruptOrDone(stopChan, done)
}

func (g *gracefulShutdown) waitForInterruptOrDone(stopChan <-chan os.Signal, done chan error) error {
	for {
		select {
		case err := <-done:
			return err
		case <-stopChan:
			g.mu.Lock()
			g.promptUp = true
			g.mu.Unlock()
			_, err := cliui.Prompt(g.cmd, cliui.PromptOptions{
				Text:      "Coder is still shutting down, if you close Coder now, your workspaces may remain up. Would you like to exit?",
				IsConfirm: true,
			})
			if err != nil {
				g.mu.Lock()
				g.promptUp = false
				g.mu.Unlock()
				continue
			}
			os.Exit(1)
		}
	}
}

func (g *gracefulShutdown) flushLine() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.promptUp {
		time.Sleep(10*time.Millisecond) // avoid tightlooping on idle
		return
	}
	if i := bytes.IndexByte(g.outBuf, '\n'); i>=0 {
		line := g.outBuf[0:i+1]
		_, _ = g.cmd.OutOrStdout().Write(line)
		g.outBuf = g.outBuf[i+1:]
		return
	}
	time.Sleep(10*time.Millisecond) // avoid tightlooping on idle
}

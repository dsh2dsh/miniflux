package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"miniflux.app/v2/internal/config"
)

func Listener() (net.Listener, error) {
	if !config.HasHTTPService() {
		return nil, nil
	}

	var listener net.Listener
	listenAddr := config.ListenAddr()

	switch {
	case os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()):
		f := os.NewFile(3, "systemd socket")
		l, err := net.FileListener(f)
		if err != nil {
			return nil, fmt.Errorf(
				"http/server: create listener from systemd socket: %w", err)
		}
		listener = l
	case strings.HasPrefix(listenAddr, "/"):
		l, err := unixListener(listenAddr, 0o666)
		if err != nil {
			return nil, fmt.Errorf("create unix listener on %q: %w", listenAddr, err)
		}
		listener = l
	}
	return listener, nil
}

func unixListener(path string, mode uint32) (*net.UnixListener, error) {
	if err := unlinkStaleUnix(path); err != nil {
		return nil, err
	}

	laddr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, fmt.Errorf("http/server: resolve unix address: %w", err)
	}

	l, err := net.ListenUnix("unix", laddr)
	if err != nil {
		return nil, fmt.Errorf("http/server: listen unix: %w", err)
	}

	l.SetUnlinkOnClose(true)
	if mode == 0 {
		return l, nil
	}

	if err := os.Chmod(path, os.FileMode(mode)); err != nil {
		return nil, fmt.Errorf(
			"http/server: change socket mode to %O: %w", mode, err)
	}
	return l, nil
}

func unlinkStaleUnix(path string) error {
	sockdir := filepath.Dir(path)
	stat, err := os.Stat(sockdir)
	switch {
	case err != nil && os.IsNotExist(err):
		if err := os.MkdirAll(sockdir, 0o755); err != nil {
			return fmt.Errorf("http/server: cannot mkdir %q: %w", sockdir, err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("http/server: cannot stat(2) %q: %w", sockdir, err)
	case !stat.IsDir():
		return fmt.Errorf("http/server: not a directory: %q", sockdir)
	}

	_, err = os.Stat(path)
	switch {
	case err == nil:
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("http/server: cannot remove stale socket: %w", err)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("http/server: cannot stat(2): %w", err)
	}
	return nil
}

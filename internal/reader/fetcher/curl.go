package fetcher

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"slices"
	"strconv"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
)

var defaultProtoMinor = []byte(".0 ")

type curlTransport struct {
	args []string
}

var _ http.RoundTripper = (*curlTransport)(nil)

func (self *curlTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(req.Context())
	cmd := self.cmd(ctx, req.URL.String())

	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("fetcher: curl stdout pipe: %w", err)
	}

	log := logging.FromContext(ctx)
	command := cmd.String()
	log.Info(command)

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("fetcher: start curl %s: %w", cmd.Path, err)
	}

	waitCurl := func() {
		defer cancel()
		_, _ = io.Copy(io.Discard, stdout)
		err := cmd.Wait()
		if err == nil {
			return
		}

		log.Error("curl command failed",
			slog.Int("stderr", stderr.Len()),
			slog.Any("error", err))

		if stderr.Len() == 0 {
			return
		}

		s := bufio.NewScanner(stderr)
		for s.Scan() {
			log.Error("stderr: " + s.Text())
		}
	}

	r, err := withProtoMinorVersion(stdout)
	if err != nil {
		cancel()
		waitCurl()
		return nil, fmt.Errorf("fetcher: fix HTTP proto minor version: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(r), req)
	if err != nil {
		cancel()
		waitCurl()
		return nil, fmt.Errorf("fetcher: reading curl output: %w", err)
	}
	return beforeBodyClose(resp, waitCurl), nil
}

func (self *curlTransport) cmd(ctx context.Context, u string) *exec.Cmd {
	maxBodySize := strconv.FormatInt(config.HTTPClientMaxBodySize(), 10)
	timeout := strconv.FormatFloat(config.HTTPClientTimeout().Seconds(),
		'f', -1, 64)

	args := slices.Concat(self.args, []string{
		"--compressed",
		"--connect-timeout", timeout,
		"--disable",
		"--location",
		"--max-filesize", maxBodySize,
		"--max-time", timeout,
		"--no-buffer",
		"--show-error",
		"--show-headers",
		"--silent",
		u,
	})
	return exec.CommandContext(ctx, args[0], args[1:]...)
}

func withProtoMinorVersion(r io.Reader) (io.Reader, error) {
	buf := make([]byte, 4096)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read buf unexpected: %w", err)
	}
	// err == nil || err == io.EOF

	if n < len(buf) {
		buf = buf[:n]
	}

	proto, after, ok := bytes.Cut(buf, []byte(" "))
	fixable := ok && bytes.HasPrefix(proto, []byte("HTTP/")) &&
		len(proto) == len("HTTP/X")
	if !fixable {
		if err == io.EOF {
			return bytes.NewReader(buf), nil
		}
		return io.MultiReader(bytes.NewReader(buf), r), nil

	}

	readers := append(make([]io.Reader, 0, 4),
		bytes.NewReader(proto),
		bytes.NewReader(defaultProtoMinor),
		bytes.NewReader(after))
	if err != io.EOF {
		readers = append(readers, r)
	}
	return io.MultiReader(readers...), nil
}

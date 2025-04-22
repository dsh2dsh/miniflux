package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	dotenv "github.com/dsh2dsh/expx-dotenv"
	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/client"
	"miniflux.app/v2/internal/cli"
)

var (
	localMode   bool
	logFileName string
	pidFileName string
)

func init() {
	flag.BoolVar(&localMode, "local", false,
		"demonize api server for running e2e test locally")
	flag.StringVar(&logFileName, "log", "e2e_api.log",
		"name of api server's log file")
	flag.StringVar(&pidFileName, "pid", "e2e_api.pid",
		"name of api server's pid file")
}

func main() {
	if err := dotenv.New().WithDepth(1).Load(); err != nil {
		log.Fatal(fmt.Errorf("failed parse .env file(s): %w", err))
	}

	flag.Parse()
	if localMode {
		if err := execLocalServer(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	cli.Execute()
}

func execLocalServer() error {
	cmd, err := demonize()
	if err != nil {
		return err
	}
	log.Printf("API server started, pid %d...\n", cmd.Process.Pid)
	log.Printf("all output redirected to %s\n", logFileName)

	if err := waitReady(cmd); err != nil {
		return err
	} else if err := writePid(cmd.Process.Pid); err != nil {
		return err
	}

	log.Printf("API server ready, %s created\n", pidFileName)
	return nil
}

func demonize() (*exec.Cmd, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed obtain executable: %w", err)
	}

	outFile, err := os.Create(logFileName)
	if err != nil {
		return nil, fmt.Errorf("failed open log file: %w", err)
	}

	cmd := exec.Command(path)
	cmd.Stdout = outFile
	cmd.Stderr = outFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		outFile.Close()
		return nil, fmt.Errorf("failed start %q: %w", path, err)
	}
	return cmd, nil
}

func waitReady(cmd *exec.Cmd) error {
	baseURL := "http://" + os.Getenv("LISTEN_ADDR")
	api := client.NewClient(baseURL, os.Getenv("ADMIN_USERNAME"),
		os.Getenv("ADMIN_PASSWORD"))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := cmd.Wait(); err != nil {
			state := cmd.ProcessState
			if state.Exited() {
				return fmt.Errorf("API server exited with status %d: %w",
					state.ExitCode(), err)
			}
			return fmt.Errorf("API server was terminated by: %w", err)
		}
		return nil
	})

	if err := waitHealth(ctx, api); err != nil {
		return err
	}
	return nil
}

func waitHealth(ctx context.Context, api *client.Client) error {
	if ok, err := healthCheck(api); err != nil {
		return err
	} else if ok {
		return nil
	}

	log.Print("API server isn't ready yet")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	startTime := time.Now()

	for {
		select {
		case <-ticker.C:
			if ok, err := healthCheck(api); err != nil {
				return err
			} else if ok {
				log.Print("Got OK health status")
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("waiting for api server: %w", context.Cause(ctx))
		}
		log.Printf("Still waiting... (%s)", time.Since(startTime))
	}
}

func healthCheck(api *client.Client) (bool, error) {
	if err := api.Healthcheck(); err != nil {
		var errno syscall.Errno
		if errors.As(err, &errno) && errno == syscall.ECONNREFUSED {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func writePid(pid int) error {
	f, err := os.Create(pidFileName)
	if err != nil {
		return fmt.Errorf("failed write pid: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, strconv.FormatInt(int64(pid), 10)); err != nil {
		return fmt.Errorf("failed write pid: %w", err)
	}
	return nil
}

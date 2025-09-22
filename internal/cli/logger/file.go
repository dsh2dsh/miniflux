package logger

import (
	"fmt"
	"os"
	"syscall"
)

func NewLogFile(filename string) (f *LogFile, err error) {
	f = &LogFile{filename: filename}
	if err := f.open(); err != nil {
		return nil, err
	}
	return f, nil
}

type LogFile struct {
	*os.File

	filename string
}

func (self *LogFile) open() (err error) {
	self.File, err = os.OpenFile(self.filename,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	return nil
}

func (self *LogFile) Write(p []byte) (int, error) {
	if err := self.reopenIfNotExists(); err != nil {
		return 0, fmt.Errorf("reopen file %q: %w", self.filename, err)
	}
	n, err := self.File.Write(p)
	if err != nil {
		return n, fmt.Errorf("write to %q: %w", self.filename, err)
	}
	return n, nil
}

func (self *LogFile) reopenIfNotExists() error {
	if ok, err := self.exists(); err != nil {
		return err
	} else if ok {
		return nil
	}
	return self.reopen()
}

func (self *LogFile) exists() (bool, error) {
	finfo, err := self.Stat()
	if err != nil {
		return false, fmt.Errorf("stat of %q: %w", self.filename, err)
	}

	if sys := finfo.Sys(); sys != nil {
		if stat, ok := sys.(*syscall.Stat_t); ok {
			return stat.Nlink > 0, nil
		}
	}
	return false, nil
}

func (self *LogFile) reopen() error {
	if err := self.Close(); err != nil {
		return fmt.Errorf("close %q: %w", self.filename, err)
	}
	return self.open()
}

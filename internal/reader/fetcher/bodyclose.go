package fetcher

import (
	"fmt"
	"io"
	"net/http"
)

type bodyCloser struct {
	io.ReadCloser

	handler func()
	closer  func() error
}

var _ io.ReadCloser = (*bodyCloser)(nil)

func newBodyCloser(body io.ReadCloser, handler func()) *bodyCloser {
	return &bodyCloser{ReadCloser: body, handler: handler}
}

func afterBodyClose(resp *http.Response, handler func()) *http.Response {
	return newBodyCloser(resp.Body, handler).closeAfter(resp)
}

func beforeBodyClose(resp *http.Response, handler func()) *http.Response {
	return newBodyCloser(resp.Body, handler).closeBefore(resp)
}

func (self *bodyCloser) closeAfter(resp *http.Response) *http.Response {
	self.closer = self.after
	resp.Body = self
	return resp
}

func (self *bodyCloser) closeBefore(resp *http.Response) *http.Response {
	self.closer = self.before
	resp.Body = self
	return resp
}

func (self *bodyCloser) Close() error { return self.closer() }

func (self *bodyCloser) after() error {
	err := self.ReadCloser.Close()
	self.handler()
	if err != nil {
		return fmt.Errorf("fetcher: after body close: %w", err)
	}
	return nil
}

func (self *bodyCloser) before() error {
	self.handler()
	err := self.ReadCloser.Close()
	if err != nil {
		return fmt.Errorf("fetcher: before body close: %w", err)
	}
	return nil
}

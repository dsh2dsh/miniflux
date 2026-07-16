package server

import (
	"os"
	"strconv"
	"strings"

	"miniflux.app/v2/internal/config"
)

const (
	plainServer int = iota
	tlsServer
	systemdServer
	unixServer
	autoCertServer
)

func (self *Server) detectServerType() {
	certDomain := config.CertDomain()
	listenAddr := config.ListenAddr()

	switch {
	case os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()):
		self.serverType = systemdServer
	case strings.HasPrefix(listenAddr, "/"):
		self.serverType = unixServer
	case certDomain != "":
		self.serverType = autoCertServer
	case self.tlsConfigured():
		self.serverType = tlsServer
	}
}

func (self *Server) tlsConfigured() bool {
	return self.certFile != "" && self.keyFile != ""
}

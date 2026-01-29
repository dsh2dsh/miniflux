package sites

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/template"
)

type SitesTestSuite struct {
	suite.Suite

	templates *template.Engine
	user      *model.User
}

func (self *SitesTestSuite) withConfig(envs map[string]string) {
	self.T().Helper()
	for k, v := range envs {
		self.T().Setenv(k, v)
	}
	self.Require().NoError(config.Load(""))
}

func (self *SitesTestSuite) rewrite(entry *model.Entry) *model.Entry {
	self.T().Helper()

	Rewrite(self.T().Context(), entry)

	b, err := json.Marshal(entry)
	self.Require().NoError(err)
	self.Require().NotEmpty(b)

	entry = &model.Entry{}
	self.Require().NoError(json.Unmarshal(b, entry))
	return entry
}

func (self *SitesTestSuite) render(entry *model.Entry) string {
	self.T().Helper()
	b, err := Render(self.T().Context(), self.user, entry, self.templates)
	self.Require().NoError(err)
	self.Require().NotEmpty(b)
	return string(b)
}

func TestSites(t *testing.T) {
	if testing.Verbose() {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	templates := template.NewEngine(mux.New())
	require.NoError(t, templates.ParseTemplates())

	suite.Run(t, &SitesTestSuite{
		templates: templates,
		user:      &model.User{Language: "en_US"},
	})
}

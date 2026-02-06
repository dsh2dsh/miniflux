package session

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

func New(store *storage.Storage, r *http.Request) *Session {
	return &Session{
		Session: request.Session(r),
		store:   store,
	}
}

type Session struct {
	*model.Session

	store   *storage.Storage
	changed bool
}

func (self *Session) SetLastForceRefresh() *Session {
	self.Data.LastForceRefresh = time.Now().UTC().Unix()
	self.changed = true
	return self
}

func (self *Session) SetOAuth2State(state string) *Session {
	self.Data.OAuth2State = state
	self.changed = true
	return self
}

func (self *Session) SetOAuth2CodeVerifier(codeVerfier string) *Session {
	self.Data.OAuth2CodeVerifier = codeVerfier
	self.changed = true
	return self
}

func (self *Session) NewFlashMessage(message string) *Session {
	self.Data.FlashMessage = message
	self.changed = true
	return self
}

func (self *Session) FlashMessage(message string) string {
	if message != "" {
		self.NewFlashMessage("")
	}
	return message
}

func (self *Session) NewFlashErrorMessage(message string) *Session {
	self.Data.FlashErrorMessage = message
	return self
}

func (self *Session) FlashErrorMessage(message string) string {
	if message != "" {
		self.NewFlashErrorMessage("")
	}
	return message
}

func (self *Session) SetLanguage(language string) *Session {
	self.Data.Language = language
	self.changed = true
	return self
}

func (self *Session) SetTheme(theme string) *Session {
	self.Data.Theme = theme
	self.changed = true
	return self
}

func (self *Session) SetWebAuthnSessionData(sessionData *model.WebAuthnSession,
) *Session {
	self.Data.WebAuthnSessionData = *sessionData
	self.changed = true
	return self
}

func (self *Session) Commit(ctx context.Context) {
	if !self.changed {
		return
	}

	if err := self.store.UpdateAppSession(ctx, self.Session); err != nil {
		logging.FromContext(ctx).Error("unable update session",
			slog.String("id", self.ID),
			slog.Any("error", err))
	}
}

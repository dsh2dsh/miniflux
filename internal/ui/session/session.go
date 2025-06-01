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
		store:  store,
		s:      request.Session(r),
		values: make(map[string]any),
	}
}

type Session struct {
	store  *storage.Storage
	s      *model.Session
	values map[string]any
}

func (self *Session) SetLastForceRefresh() *Session {
	self.values["last_force_refresh"] = time.Now().UTC().Unix()
	return self
}

func (self *Session) SetOAuth2State(state string) *Session {
	self.values["oauth2_state"] = state
	return self
}

func (self *Session) SetOAuth2CodeVerifier(codeVerfier string) *Session {
	self.values["oauth2_code_verifier"] = codeVerfier
	return self
}

func (self *Session) NewFlashMessage(message string) *Session {
	self.values["flash_message"] = message
	return self
}

func (self *Session) FlashMessage(message string) string {
	if message != "" {
		self.values["flash_message"] = ""
	}
	return message
}

func (self *Session) NewFlashErrorMessage(message string) *Session {
	self.values["flash_error_message"] = message
	return self
}

func (self *Session) FlashErrorMessage(message string) string {
	if message != "" {
		self.values["flash_error_message"] = ""
	}
	return message
}

func (self *Session) SetLanguage(language string) *Session {
	self.values["language"] = language
	return self
}

func (self *Session) SetTheme(theme string) *Session {
	self.values["theme"] = theme
	return self
}

func (self *Session) SetPocketRequestToken(requestToken string) *Session {
	self.values["pocket_request_token"] = requestToken
	return self
}

func (self *Session) SetWebAuthnSessionData(sessionData *model.WebAuthnSession,
) *Session {
	self.values["webauthn_session_data"] = sessionData
	return self
}

func (self *Session) Commit(ctx context.Context) {
	if len(self.values) == 0 {
		return
	}

	err := self.store.UpdateAppSession(ctx, self.s, self.values)
	if err != nil {
		logging.FromContext(ctx).Error("unable update session",
			slog.String("id", self.id()),
			slog.Any("error", err))
	}
}

func (self *Session) id() string { return self.s.ID }

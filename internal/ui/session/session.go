package session

import (
	"context"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"time"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

func New(store *storage.Storage, r *http.Request) *Session {
	return &Session{
		store:   store,
		s:       request.Session(r),
		updated: make(map[string]any),
		deleted: make(map[string]struct{}),
	}
}

type Session struct {
	store   *storage.Storage
	s       *model.Session
	updated map[string]any
	deleted map[string]struct{}
}

func (self *Session) update(k string, v any) *Session {
	self.updated[k] = v
	delete(self.deleted, k)
	return self
}

func (self *Session) delete(k string) {
	delete(self.updated, k)
	self.deleted[k] = struct{}{}
}

func updateOrDelete[T comparable](self *Session, k string, value T) *Session {
	var zeroValue T
	if value != zeroValue {
		self.update(k, value)
	} else {
		self.delete(k)
	}
	return self
}

func (self *Session) SetLastForceRefresh() *Session {
	return self.update("last_force_refresh", time.Now().UTC().Unix())
}

func (self *Session) SetOAuth2State(state string) *Session {
	return updateOrDelete(self, "oauth2_state", state)
}

func (self *Session) SetOAuth2CodeVerifier(codeVerfier string) *Session {
	return updateOrDelete(self, "oauth2_code_verifier", codeVerfier)
}

func (self *Session) NewFlashMessage(message string) *Session {
	return updateOrDelete(self, "flash_message", message)
}

func (self *Session) FlashMessage(message string) string {
	if message != "" {
		self.delete("flash_message")
	}
	return message
}

func (self *Session) NewFlashErrorMessage(message string) *Session {
	return updateOrDelete(self, "flash_error_message", message)
}

func (self *Session) FlashErrorMessage(message string) string {
	if message != "" {
		self.delete("flash_error_message")
	}
	return message
}

func (self *Session) SetLanguage(language string) *Session {
	return updateOrDelete(self, "language", language)
}

func (self *Session) SetTheme(theme string) *Session {
	return updateOrDelete(self, "theme", theme)
}

func (self *Session) SetWebAuthnSessionData(sessionData *model.WebAuthnSession,
) *Session {
	return updateOrDelete(self, "webauthn_session_data", sessionData)
}

func (self *Session) Commit(ctx context.Context) {
	if len(self.updated) == 0 && len(self.deleted) == 0 {
		return
	}

	var deleted []string
	if len(self.deleted) != 0 {
		deleted = slices.AppendSeq(make([]string, 0, len(self.deleted)),
			maps.Keys(self.deleted))
	}

	err := self.store.UpdateAppSession(ctx, self.s, self.updated, deleted)
	if err != nil {
		logging.FromContext(ctx).Error("unable update session",
			slog.String("id", self.id()),
			slog.Any("error", err))
	}
}

func (self *Session) id() string { return self.s.ID }

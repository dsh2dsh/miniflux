// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"crypto/md5"
	"fmt"
	"net/http"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) updateIntegration(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	i := user.Integration()
	printer := locale.NewPrinter(request.UserLanguage(r))
	ctx := r.Context()
	sess := session.New(h.store, r)
	defer sess.Commit(ctx)

	f := form.NewIntegrationForm(r)
	f.Merge(i)

	if i.FeverEnabled {
		if f.FeverPassword != "" {
			i.FeverToken = fmt.Sprintf("%x",
				md5.Sum([]byte(user.Username+":"+f.FeverPassword)))
		}
	} else {
		i.FeverToken = ""
	}

	if i.GoogleReaderEnabled {
		if f.GoogleReaderPassword != "" {
			pw, err := crypto.HashPassword(f.GoogleReaderPassword)
			if err != nil {
				html.ServerError(w, r, err)
				return
			}
			i.GoogleReaderPassword = pw
		}
	} else {
		i.GoogleReaderPassword = ""
	}

	if f.WebhookEnabled {
		if f.WebhookURL == "" {
			i.WebhookEnabled = false
			i.WebhookSecret = ""
		} else if i.WebhookSecret == "" {
			i.WebhookSecret = crypto.GenerateRandomStringHex(32)
		}
	} else {
		i.WebhookURL = ""
		i.WebhookSecret = ""
	}

	if f.LinktacoEnabled {
		if f.LinktacoAPIToken == "" || f.LinktacoOrgSlug == "" {
			sess.NewFlashErrorMessage(printer.Print("error.linktaco_missing_required_fields"))
			h.redirect(w, r, "integrations")
			return
		}
		if i.LinktacoVisibility == "" {
			i.LinktacoVisibility = "PUBLIC"
		}
	}

	if err := h.store.UpdateUser(ctx, user); err != nil {
		html.ServerError(w, r, err)
		return
	}

	sess.NewFlashMessage(printer.Print("alert.prefs_saved"))
	h.redirect(w, r, "integrations")
}

// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"github.com/gorilla/mux"

	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/template"
	"miniflux.app/v2/internal/worker"
)

type handler struct {
	router *mux.Router
	store  *storage.Storage
	tpl    *template.Engine
	pool   *worker.Pool
}

// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package route // import "miniflux.app/v2/internal/http/route"

import (
	"fmt"
	"strconv"

	"miniflux.app/v2/internal/http/mux"
)

// Path returns the defined route based on given arguments.
func Path(m *mux.ServeMux, name string, args ...any) string {
	pairs := make([]string, len(args))
	for i, arg := range args {
		switch param := arg.(type) {
		case string:
			pairs[i] = param
		case int:
			pairs[i] = strconv.Itoa(param)
		case int64:
			pairs[i] = strconv.FormatInt(param, 10)
		default:
			pairs[i] = fmt.Sprint(param)
		}
	}

	result := m.NamedPath(name, pairs...)
	if result == "" {
		panic("route not found: " + name)
	}
	return result
}

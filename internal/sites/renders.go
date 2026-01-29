package sites

import (
	"context"
	"log/slog"
	"strings"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/template"
)

type entryRender interface {
	Render(ctx context.Context, user *model.User, entry *model.Entry,
		t *template.Engine) ([]byte, error)
}

var renders = make(map[string]entryRender, 1)

func addRender(render entryRender, domains ...string) {
	for _, domain := range domains {
		renders[domain] = render
	}
}

type RenderFunc func(ctx context.Context, user *model.User, entry *model.Entry,
	t *template.Engine) ([]byte, error)

func (self RenderFunc) Render(ctx context.Context, user *model.User,
	entry *model.Entry, t *template.Engine,
) ([]byte, error) {
	return self(ctx, user, entry, t)
}

func addRenderFunc(render RenderFunc, domains ...string) {
	addRender(render, domains...)
}

func Render(ctx context.Context, user *model.User, entry *model.Entry,
	t *template.Engine,
) ([]byte, error) {
	hostname := entry.Hostname()
	log := logging.FromContext(ctx).With(
		slog.String("hostname", hostname),
		slog.String("entry_url", entry.URL))
	log.Debug("Looking for site specific entry render")

	for {
		if render, ok := renders[hostname]; ok {
			log.Debug("Applying site specific entry render",
				slog.String("domain", hostname))
			b, err := render.Render(ctx, user, entry, t)
			if err != nil {
				log.Error("site specific render failed", slog.Any("error", err))
				return nil, err
			}
			return b, nil
		}
		_, domain, ok := strings.Cut(hostname, ".")
		if !ok {
			return nil, nil
		}
		hostname = domain
	}
}

func renderTemplate(t *template.Engine, name string, user *model.User,
	entry *model.Entry, data map[string]any,
) []byte {
	data["language"] = user.Language
	data["Entry"] = entry
	return t.Render(name, data)
}

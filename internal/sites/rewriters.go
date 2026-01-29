package sites

import (
	"context"
	"log/slog"
	"strings"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

type entryRewriter interface {
	Rewrite(ctx context.Context, entry *model.Entry)
}

var rewriters = make(map[string]entryRewriter, 1)

func addRewriter(rewriter entryRewriter, domains ...string) {
	for _, domain := range domains {
		rewriters[domain] = rewriter
	}
}

type RewriterFunc func(ctx context.Context, entry *model.Entry)

func (self RewriterFunc) Rewrite(ctx context.Context, entry *model.Entry) {
	self(ctx, entry)
}

func addRewriterFunc(rewriter RewriterFunc, domains ...string) {
	addRewriter(rewriter, domains...)
}

func Rewrite(ctx context.Context, entry *model.Entry) {
	hostname := entry.Hostname()
	log := logging.FromContext(ctx).With(
		slog.String("hostname", hostname),
		slog.String("entry_url", entry.URL))
	log.Debug("Applying site specific content rewriters")

	for {
		if rewriter, ok := rewriters[hostname]; ok {
			log.Debug("Applying site specific content rewriter",
				slog.String("domain", hostname))
			rewriter.Rewrite(ctx, entry)
			return
		}
		_, domain, ok := strings.Cut(hostname, ".")
		if !ok {
			return
		}
		hostname = domain
	}
}

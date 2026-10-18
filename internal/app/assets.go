package app

import (
	"errors"
	"net/http"
	"strings"

	"github.com/unclesp1d3r/redtracks/internal/evidence"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

const (
	routeEvidence = "/evidence/"
	routeInbox    = "/inbox/"
	// Headers that keep a served evidence file inert in the webview.
	cspSandbox   = "sandbox; default-src 'none'"
	cacheControl = "private, max-age=0"
)

// EvidenceAssets returns the asset middleware that serves /evidence/<ip>/<file>
// from the open engagement and /inbox/<file> from the inbox, and passes every
// other path to next. Files are confined by evidence.Resolve and typed by their
// sniffed content only.
func EvidenceAssets(a *App) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, routeEvidence):
				serveEvidence(a, w, r)
			case strings.HasPrefix(r.URL.Path, routeInbox):
				serveFile(a, w, r, a.inboxDir(), splitSegments(r.URL.Path, routeInbox), 1)
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

func splitSegments(urlPath, prefix string) []string {
	return strings.Split(strings.TrimPrefix(urlPath, prefix), "/")
}

func serveEvidence(a *App, w http.ResponseWriter, r *http.Request) {
	segs := splitSegments(r.URL.Path, routeEvidence)
	if len(segs) != 2 { //nolint:mnd // <ip>/<file>
		http.NotFound(w, r)
		return
	}
	st, err := a.Store()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	root, err := store.EvidenceDir(st.Engagement().Dir, segs[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	serveFile(a, w, r, root, segs[1:], 1)
}

// serveFile resolves segs (which must number want) under root and streams the file.
func serveFile(a *App, w http.ResponseWriter, r *http.Request, root string, segs []string, want int) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(segs) != want {
		http.NotFound(w, r)
		return
	}
	f, res, err := evidence.OpenResolved(root, segs...)
	if err != nil {
		writeResolveError(a, w, r, err)
		return
	}
	defer func() { _ = f.Close() }()
	h := w.Header()
	h.Set("Content-Type", res.ContentType)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", cspSandbox)
	h.Set("Cache-Control", cacheControl)
	http.ServeContent(w, r, "", res.ModTime, f)
}

// writeResolveError maps a Resolve error to a status without echoing any path.
func writeResolveError(a *App, w http.ResponseWriter, r *http.Request, err error) {
	var notFound *evidence.NotFoundError
	var refused *evidence.RefusedError
	switch {
	case errors.As(err, &notFound):
		http.NotFound(w, r)
	case errors.As(err, &refused):
		http.Error(w, "forbidden", http.StatusForbidden)
	default:
		a.log.ErrorContext(r.Context(), "EvidenceAssets", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

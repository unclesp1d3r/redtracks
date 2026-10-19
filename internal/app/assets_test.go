package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const markupBody = "<html><script>alert(1)</script></html>"

// assetsHarness serves requests through EvidenceAssets and records fall-through.
type assetsHarness struct {
	handler http.Handler
	nextHit bool
}

func newAssetsHarness(a *App) *assetsHarness {
	h := &assetsHarness{}
	h.handler = EvidenceAssets(a)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		h.nextHit = true
		w.WriteHeader(http.StatusTeapot)
	}))
	return h
}

func (h *assetsHarness) do(method, path string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, "http://wails.localhost"+path, http.NoBody)
	req.URL.Path = path // keep ".." as sent; NewRequest would not clean it but be explicit
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func evidenceDirOf(f fixture) string {
	return filepath.Join(f.dir, "hosts", sampleHost, "evidence")
}

func putAsset(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceAssetsServesImageWithHardeningHeaders(t *testing.T) {
	f := openSample(t)
	putAsset(t, evidenceDirOf(f), "shot.png", pngBytes("x"))
	h := newAssetsHarness(f.app)

	rec := h.do(http.MethodGet, "/evidence/"+sampleHost+"/shot.png", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	want := map[string]string{
		"Content-Type":            "image/png",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "sandbox; default-src 'none'",
		"Cache-Control":           "private, max-age=0",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestEvidenceAssetsHonoursRange(t *testing.T) {
	f := openSample(t)
	putAsset(t, evidenceDirOf(f), "shot.png", pngBytes("0123456789"))
	h := newAssetsHarness(f.app)

	rec := h.do(http.MethodGet, "/evidence/"+sampleHost+"/shot.png", map[string]string{"Range": "bytes=0-3"})

	if rec.Code != http.StatusPartialContent || rec.Body.Len() != 4 {
		t.Fatalf("status = %d, body = %d bytes", rec.Code, rec.Body.Len())
	}
}

func TestEvidenceAssetsServesMarkupAsPlainText(t *testing.T) {
	f := openSample(t)
	putAsset(t, f.app.inboxDir(), "x.svg", []byte(markupBody))
	putAsset(t, f.app.inboxDir(), "x.html", []byte(markupBody))
	h := newAssetsHarness(f.app)

	for _, name := range []string{"x.svg", "x.html"} {
		rec := h.do(http.MethodGet, "/inbox/"+name, nil)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" ||
			rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: status %d headers %v", name, rec.Code, rec.Header())
		}
	}
}

func TestEvidenceAssetsRefusesTrueSVG(t *testing.T) {
	f := openSample(t)
	putAsset(t, f.app.inboxDir(), "x.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`))

	rec := newAssetsHarness(f.app).do(http.MethodGet, "/inbox/x.svg", nil)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestEvidenceAssetsWithoutEngagementIs404(t *testing.T) {
	f := newTestApp(t)

	rec := newAssetsHarness(f.app).do(http.MethodGet, "/evidence/"+sampleHost+"/shot.png", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestEvidenceAssetsMissingAndMalformedAre404(t *testing.T) {
	f := openSample(t)
	h := newAssetsHarness(f.app)
	for _, p := range []string{
		"/evidence/" + sampleHost + "/nope.png",
		"/evidence/10.9.9.9/nope.png",
		"/evidence/not-an-ip/x.png",
		"/evidence/" + sampleHost,
		"/evidence/" + sampleHost + "/a/b.png",
	} {
		if rec := h.do(http.MethodGet, p, nil); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d", p, rec.Code)
		}
	}
	if h.nextHit {
		t.Fatal("an evidence route fell through to the next handler")
	}
}

func TestEvidenceAssetsTraversalIsRefusedWithoutNext(t *testing.T) {
	f := openSample(t)
	putAsset(t, f.app.inboxDir(), "ok.png", pngBytes("x"))
	h := newAssetsHarness(f.app)

	for _, p := range []string{"/inbox/..", "/evidence/" + sampleHost + "/..", "/inbox/%2e%2e"} {
		rec := h.do(http.MethodGet, p, nil)
		if rec.Code < 400 || rec.Code >= 500 {
			t.Errorf("%s: status = %d", p, rec.Code)
		}
		if strings.Contains(rec.Body.String(), f.app.Root) {
			t.Errorf("%s: body leaks a path: %q", p, rec.Body.String())
		}
	}
	if h.nextHit {
		t.Fatal("traversal reached the next handler")
	}
}

func TestEvidenceAssetsUnrelatedPathsFallThrough(t *testing.T) {
	f := openSample(t)
	h := newAssetsHarness(f.app)

	rec := h.do(http.MethodGet, "/assets/index.js", nil)

	if !h.nextHit || rec.Code != http.StatusTeapot {
		t.Fatalf("nextHit=%v status=%d", h.nextHit, rec.Code)
	}
}

func TestEvidenceAssetsRejectsWrites(t *testing.T) {
	f := openSample(t)
	putAsset(t, f.app.inboxDir(), "ok.png", pngBytes("x"))

	rec := newAssetsHarness(f.app).do(http.MethodPost, "/inbox/ok.png", nil)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

package http

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed testdata/dist
var testAssetsRaw embed.FS

func testAssets(t *testing.T) fs.FS {
	t.Helper()
	sub, err := fs.Sub(testAssetsRaw, "testdata/dist")
	if err != nil {
		t.Fatal(err)
	}
	return sub
}

func TestSPAHandler_ServesStaticFile(t *testing.T) {
	handler := SPAHandler(testAssets(t))

	req := httptest.NewRequest("GET", "/test.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != "console.log('test');\n" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestSPAHandler_FallbackToIndex(t *testing.T) {
	handler := SPAHandler(testAssets(t))

	req := httptest.NewRequest("GET", "/some/spa/route", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != "<html><body>test</body></html>\n" {
		t.Errorf("unexpected body: %q", body)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache for SPA fallback, got %q", cc)
	}
}

func TestSPAHandler_RootServesIndex(t *testing.T) {
	handler := SPAHandler(testAssets(t))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != "<html><body>test</body></html>\n" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestSPAHandler_HashedAssetsCached(t *testing.T) {
	handler := SPAHandler(testAssets(t))

	req := httptest.NewRequest("GET", "/assets/app.abc123.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	cc := rec.Header().Get("Cache-Control")
	if cc != "public, max-age=31536000, immutable" {
		t.Errorf("expected long cache for hashed asset, got %q", cc)
	}
}

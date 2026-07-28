package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	handleHealthz(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleRenderRequiresToken(t *testing.T) {
	h := handleRender("secret")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader("x -> y"))
	h(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status without token = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/render", strings.NewReader("x -> y"))
	req.Header.Set("Authorization", "Bearer wrong")
	h(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status with wrong token = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleRenderSuccess(t *testing.T) {
	h := handleRender("")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader("x -> y: hi"))
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	if !strings.HasPrefix(rec.Body.String(), "\x89PNG") {
		t.Error("response body does not start with the PNG magic bytes")
	}
}

func TestHandleRenderEmptyBody(t *testing.T) {
	h := handleRender("")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader(""))
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleRenderInvalidD2(t *testing.T) {
	h := handleRender("")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader("x -> y: {"))
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleRenderInvalidScale(t *testing.T) {
	h := handleRender("")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/render?scale=notanumber", strings.NewReader("x -> y"))
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

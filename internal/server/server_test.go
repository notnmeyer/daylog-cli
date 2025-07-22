package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestShowHandler(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	req := httptest.NewRequest("GET", "/show", nil)
	w := httptest.NewRecorder()

	handler := showHandler(&wg)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	expectedContentType := "text/html"
	if contentType := w.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected HTML content, got non-HTML response")
	}

	// we could be more specific but whatever
	if !strings.Contains(body, "Daylog") {
		t.Error("expected title 'Daylog' in response")
	}

	wg.Wait()
}

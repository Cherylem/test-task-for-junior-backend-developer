package handlers

import (
	"net/http"
	"strings"
	"testing"
)

func TestDecodeJSONAcceptsSingleObject(t *testing.T) {
	t.Parallel()

	request, err := http.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"Call patients"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	var payload taskMutationDTO
	if err := decodeJSON(request, &payload); err != nil {
		t.Fatalf("decodeJSON returned error: %v", err)
	}
	if payload.Title != "Call patients" {
		t.Fatalf("unexpected title: %q", payload.Title)
	}
}

func TestDecodeJSONRejectsTrailingPayload(t *testing.T) {
	t.Parallel()

	request, err := http.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"Call patients"}{"title":"Extra"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	var payload taskMutationDTO
	err = decodeJSON(request, &payload)
	if err == nil {
		t.Fatal("expected decodeJSON to reject trailing payload")
	}
	if !strings.Contains(err.Error(), "single JSON object") {
		t.Fatalf("unexpected error: %v", err)
	}
}

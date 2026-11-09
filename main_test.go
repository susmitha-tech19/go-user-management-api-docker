package main

import (
	"net/http/httptest"
	"testing"
)

func TestParseBoundedInt(t *testing.T) {
	tests := []struct {
		name, input string
		fallback, min, max, want int
	}{
		{"default", "", 20, 1, 100, 20},
		{"valid", "25", 20, 1, 100, 25},
		{"too small", "0", 20, 1, 100, 20},
		{"too large", "999", 20, 1, 100, 100},
		{"invalid", "abc", 20, 1, 100, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseBoundedInt(tt.input, tt.fallback, tt.min, tt.max); got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, 400, "bad request")
	if rec.Code != 400 {
		t.Fatalf("got status %d, want 400", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got == "" {
		t.Fatal("expected Content-Type header")
	}
}

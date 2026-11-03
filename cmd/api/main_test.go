package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func TestHealthz(t *testing.T) {
	for _, tc := range []struct {
		nome string
		err  error
		want int
	}{{"ok", nil, 200}, {"db fora", errors.New("x"), 503}} {
		t.Run(tc.nome, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router(fakeDB{tc.err}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			if rec.Code != tc.want {
				t.Fatalf("code=%d want=%d", rec.Code, tc.want)
			}
		})
	}
}

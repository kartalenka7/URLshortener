package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPOST(t *testing.T) {
	type want struct {
		statusCode int
		URL        string
	}
	testsPost := []struct {
		name    string
		links   SavedLinks
		want    want
		request string
	}{
		{
			name: "POST test",
			links: SavedLinks{
				LinksMap: map[string]string{
					" ": " ",
				},
				gToken: "AsDfGhJkLl",
			},
			want: want{
				statusCode: 201,
				URL:        "http://localhost:8080/AsDfGhJkLl",
			},
			request: "/",
		},
	}

	for _, tt := range testsPost {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.request, nil)
			w := httptest.NewRecorder()
			handler1 := SavedLinks{
				LinksMap: tt.links.LinksMap,
				gToken:   tt.links.gToken,
			}
			h := http.HandlerFunc(handler1.ServeHTTP)
			h(w, request)
			result := w.Result()
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			defer result.Body.Close()
			body, err := io.ReadAll(result.Body)
			if err != nil {
				os.Exit(1)
			}
			URL := string(body)
			assert.Equal(t, tt.want.URL, URL)
		})
	}
}

func TestGET(t *testing.T) {
	type want struct {
		statusCode int
		URL        string
	}
	testsGet := []struct {
		name    string
		links   SavedLinks
		want    want
		request string
	}{
		{
			name: "POST test",
			links: SavedLinks{
				LinksMap: map[string]string{
					"AsDfGhJkLl": "http://localhost:8080/AsDfGhJkLl",
				},
			},
			want: want{
				statusCode: 307,
				URL:        "http://localhost:8080/AsDfGhJkLl",
			},
			request: "/AsDfGhJkLl",
		},
	}

	for _, tt := range testsGet {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)
			w := httptest.NewRecorder()
			handler1 := SavedLinks{
				LinksMap: tt.links.LinksMap,
			}
			h := http.HandlerFunc(handler1.ServeHTTP)
			h(w, request)
			result := w.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			URL := result.Header.Get("Location")
			assert.Equal(t, tt.want.URL, URL)
		})
	}
}

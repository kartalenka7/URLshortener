package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPOST(t *testing.T) {
	links := SavedLinks{
		LinksMap: map[string]string{
			" ": " ",
		},
		gToken: "AsDfGhJkLl",
	}

	r := NewRouter(links)
	ts := httptest.NewServer(r)
	defer ts.Close()

	statusCode, body, _ := testRequest(t, ts, "POST", "/")
	assert.Equal(t, 201, statusCode)
	assert.Equal(t, "http://localhost:8080/AsDfGhJkLl", body)
}

func TestGET(t *testing.T) {
	links := SavedLinks{
		LinksMap: map[string]string{
			"AsDfGhJkLl": "http://testtest/AsDfGhJkLl",
		},
		gToken: " ",
	}

	r := NewRouter(links)
	ts := httptest.NewServer(r)
	defer ts.Close()

	statusCode, _, location := testRequest(t, ts, "GET", "/AsDfGhJkLl")
	assert.Equal(t, 307, statusCode)
	assert.Equal(t, "http://testtest/AsDfGhJkLl", location)
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (int, string, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	defer resp.Body.Close()

	return resp.StatusCode, string(respBody), string(resp.Header.Get("Location"))
}

/* func TestPOST(t *testing.T) {
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
} */

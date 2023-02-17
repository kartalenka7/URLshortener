package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"errors"

	handlers "example.com/shortener/internal/app/handlers"
	storage "example.com/shortener/internal/app/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoints(t *testing.T) {

	type want struct {
		statusCode int
		URLshort   string
		err        string
	}
	testsGet := []struct {
		name    string
		storage storage.StorageLinks
		gToken  string
		want    want
		method  string
		request string
	}{
		{
			name: "POST positive test",
			storage: storage.StorageLinks{
				LinksMap: map[string]string{
					" ": " ",
				},
			},
			gToken: "AsDfGhJkLl",
			want: want{
				statusCode: http.StatusCreated,
				URLshort:   "http://localhost:8080/AsDfGhJkLl",
				err:        "",
			},
			method:  http.MethodPost,
			request: "/",
		},
		{
			name: "POST negative test",
			storage: storage.StorageLinks{
				LinksMap: map[string]string{
					"AsDfGhJkLl": "http://test/AsDfGhJkLl",
				},
			},
			gToken: "AsDfGhJkLl",
			want: want{
				statusCode: http.StatusInternalServerError,
				URLshort:   "Link already exists\n",
				err:        "",
			},
			method:  http.MethodPost,
			request: "/",
		},
		{
			name: "GET positive test",
			storage: storage.StorageLinks{
				LinksMap: map[string]string{
					"AsDfGhJkLl": "https://www.youtube.com",
				},
			},
			gToken: "",
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				URLshort:   "",
				err:        "Get \"https://www.youtube.com\": Redirect",
			},
			method:  http.MethodGet,
			request: "/AsDfGhJkLl",
		},
		{
			name: "GET negative test",
			storage: storage.StorageLinks{
				LinksMap: map[string]string{
					"AsDfGhJkLl": "https://www.youtube.com",
				},
			},
			gToken: "",
			want: want{
				statusCode: http.StatusMethodNotAllowed,
				URLshort:   "",
				err:        "",
			},
			method:  http.MethodGet,
			request: "/",
		},
	}

	for _, tt := range testsGet {
		t.Run(tt.name, func(t *testing.T) {
			r := handlers.NewRouter(tt.storage, tt.gToken)
			ts := httptest.NewServer(r)
			defer ts.Close()
			fmt.Println(tt.method, tt.request)
			statusCode, body, err := testRequest(t, ts, tt.method, tt.request)
			assert.Equal(t, tt.want.statusCode, statusCode)
			assert.Equal(t, tt.want.URLshort, body)
			if err != nil {
				fmt.Println(err.Error())
				assert.Equal(t, tt.want.err, err.Error())
			}
		})
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method, request string) (int, string, error) {
	var respBody []byte
	var err error

	req, err := http.NewRequest(method, ts.URL+request, nil)
	require.NoError(t, err)

	client := new(http.Client)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("Redirect")
	}
	resp, err := client.Do(req)

	if err == nil {
		respBody, err = io.ReadAll(resp.Body)
		defer resp.Body.Close()
		require.NoError(t, err)
	}
	return resp.StatusCode, string(respBody), err
}

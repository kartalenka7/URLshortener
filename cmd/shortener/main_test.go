package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"errors"

	handlers "example.com/shortener/internal/app/handlers"
	"example.com/shortener/internal/app/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPOST(t *testing.T) {

	type want struct {
		statusCode int
	}
	testsPost := []struct {
		name    string
		want    want
		method  string
		request string
	}{
		{
			name: "POST positive test",
			want: want{
				statusCode: http.StatusCreated,
			},
			method:  http.MethodPost,
			request: "/",
		},
	}

	for _, tt := range testsPost {
		t.Run(tt.name, func(t *testing.T) {
			s := storage.NewStorage()
			r := handlers.NewRouter(s)
			ts := httptest.NewServer(r)
			defer ts.Close()

			statusCode, body, err := testRequest(t, ts, tt.method, tt.request)
			assert.Equal(t, tt.want.statusCode, statusCode)
			assert.NoError(t, err)
			assert.NotNil(t, body)
			fmt.Println(body)
		})
	}
}

func TestGET(t *testing.T) {

	type want struct {
		statusCode int
		err        string
	}
	testsGet := []struct {
		name    string
		longURL string
		want    want
		method  string
	}{
		{
			name:    "GET positive test",
			longURL: "https://www.youtube.com",
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				err:        "Get \"https://www.youtube.com\": Redirect",
			},
			method: http.MethodGet,
		},
	}

	for _, tt := range testsGet {
		t.Run(tt.name, func(t *testing.T) {
			s := storage.NewStorage()
			// Добавить в хранилище URL, получить сгененированный токен
			gToken, err := s.AddLink(tt.longURL, "")
			assert.NoError(t, err)

			r := handlers.NewRouter(s)
			ts := httptest.NewServer(r)
			defer ts.Close()

			// Запрос = / + токен
			request := fmt.Sprintf("/%s", gToken)
			statusCode, _, err := testRequest(t, ts, tt.method, request)
			assert.Equal(t, tt.want.statusCode, statusCode)
			require.Error(t, err)
			fmt.Println(err.Error())
			assert.Equal(t, tt.want.err, err.Error())

		})
	}
}

func TestJSON(t *testing.T) {

	type want struct {
		statusCode  int
		contentType string
	}
	testsPost := []struct {
		name    string
		want    want
		method  string
		request string
	}{
		{
			name: "POST JSON test",
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "application/json",
			},
			method:  http.MethodPost,
			request: "/api/shorten",
		},
	}

	for _, tt := range testsPost {
		t.Run(tt.name, func(t *testing.T) {
			s := storage.NewStorage()
			r := handlers.NewRouter(s)
			ts := httptest.NewServer(r)
			defer ts.Close()

			statusCode, body, contentType, err := jsonRequest(t, ts, tt.method, tt.want.contentType, tt.request)
			assert.Equal(t, tt.want.statusCode, statusCode)
			assert.Equal(t, tt.want.contentType, contentType)
			assert.NoError(t, err)
			assert.NotNil(t, body)
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
	}
	return resp.StatusCode, string(respBody), err
}

func jsonRequest(t *testing.T, ts *httptest.Server, method, contentType, request string) (int, []byte, string, error) {
	var err error

	bodyStr := struct {
		LongURL string `json:"url"`
	}{
		LongURL: "https://www.youtube.com",
	}
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(bodyStr)
	req, err := http.NewRequest(method, ts.URL+request, buf)
	require.NoError(t, err)

	req.Header.Set("Content-Type", contentType)
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	require.NoError(t, err)
	return resp.StatusCode, body, resp.Header.Get("Content-Type"), err
}

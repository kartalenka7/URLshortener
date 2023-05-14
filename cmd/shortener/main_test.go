package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"time"

	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"strings"
	"testing"

	"net/http/cookiejar"

	handlers "example.com/shortener/internal/app/handlers"
	database "example.com/shortener/internal/app/storage/database"
	memory "example.com/shortener/internal/app/storage/memory"
	service "example.com/shortener/internal/app/storage/service"
	"example.com/shortener/internal/config"

	"net/url"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/publicsuffix"
)

var cfg config.Config
var jar *cookiejar.Jar

func init() {
	cfg = config.Config{
		BaseURL: "http://localhost:8080/",
		Server:  "localhost:8080",
		File:    "link.log",
		//Database: "postgres://habruser:habr@localhost:5432/habrdb",
		Database: "user=habruser password=habr host=localhost port=5432 database=habrdb sslmode=disable",
	}

	jar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
}

func TestPOST(t *testing.T) {
	var storer service.Storer
	var err error

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
			//s := storage.NewStorage(cfg)
			storer, err = database.New(cfg)
			if err != nil {
				storer = memory.New(cfg)
			}
			service := service.New(cfg, storer)
			r := handlers.NewRouter(service)
			ts := httptest.NewServer(r)
			defer ts.Close()

			var respBody []byte

			var buf bytes.Buffer
			zw := gzip.NewWriter(&buf)
			_, _ = zw.Write([]byte("https://www.pinterest20.com"))
			_ = zw.Close()

			data := url.Values{}
			data.Set("url", "https://www.pinterest20.com")

			req, err := http.NewRequest(tt.method, ts.URL+tt.request, bytes.NewBufferString(buf.String()))
			require.NoError(t, err)
			if err != nil {
				log.Println(err.Error())
			}
			req.Header.Add("Content-Encoding", "gzip")
			req.Header.Add("Accept-Encoding", "gzip")

			client := new(http.Client)
			client.Jar = jar
			resp, err := client.Do(req)
			require.NoError(t, err)

			fmt.Println("After 1st request:")
			for _, cookie := range jar.Cookies(req.URL) {
				fmt.Printf("куки  %s: %s\n", cookie.Name, cookie.Value)
			}

			respBody, err = io.ReadAll(resp.Body)
			defer resp.Body.Close()
			require.NoError(t, err)

			//statusCode, body, err := testRequest(t, ts, tt.method, tt.request)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
			assert.NoError(t, err)
			assert.NotNil(t, string(respBody))
			fmt.Println(string(respBody))
		})
	}
}

func TestGET(t *testing.T) {
	var storer service.Storer
	var err error

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
			longURL: "https://www.pinterest21.com",
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				err:        "Get \"https://www.pinterest21.com\": Redirect",
			},
			method: http.MethodGet,
		},
	}

	for _, tt := range testsGet {
		t.Run(tt.name, func(t *testing.T) {
			//s := storage.NewStorage(cfg)
			storer, err = database.New(cfg)
			if err != nil {
				storer = memory.New(cfg)
			}
			service := service.New(cfg, storer)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// Добавить в хранилище URL, получить сгененированный токен
			gToken, err := service.Storage.AddLink(tt.longURL, "", ctx)
			sToken := strings.Replace(gToken, cfg.BaseURL, "", 1)
			assert.NoError(t, err)

			r := handlers.NewRouter(service)
			ts := httptest.NewServer(r)
			defer ts.Close()

			// Запрос = / + токен
			request := fmt.Sprintf("/%s", sToken)
			log.Println(request)
			req, err := http.NewRequest(tt.method, ts.URL+request, nil)
			require.NoError(t, err)
			if err != nil {
				log.Println(err.Error())
			}
			req.Header.Add("Accept-Encoding", "no")
			client := new(http.Client)
			client.Jar = jar
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return errors.New("Redirect")
			}
			resp, err := client.Do(req)
			resp.Body.Close()

			log.Println("After 2st request:")
			for _, cookie := range jar.Cookies(req.URL) {
				log.Printf("куки  %s: %s\n", cookie.Name, cookie.Value)
			}
			//statusCode, _, err := testRequest(t, ts, tt.method, request)
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
			require.Error(t, err)
			log.Println(err.Error())
			assert.Equal(t, tt.want.err, err.Error())

		})
	}
}

func TestJSON(t *testing.T) {
	var storer service.Storer
	var err error

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
			//s := storage.NewStorage(cfg)
			storer, err = database.New(cfg)
			if err != nil {
				storer = memory.New(cfg)
			}
			service := service.New(cfg, storer)
			r := handlers.NewRouter(service)
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

func jsonRequest(t *testing.T, ts *httptest.Server, method, contentType, request string) (int, []byte, string, error) {
	var err error

	bodyStr := struct {
		LongURL string `json:"url"`
	}{
		LongURL: "https://www.pinterest22.com",
	}
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(bodyStr)
	req, err := http.NewRequest(method, ts.URL+request, buf)
	require.NoError(t, err)

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept-Encoding", "no")
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

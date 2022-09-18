package testserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

/*
Test cases:

1. Simple one request
2. Multiple requests
3. RespWithHeaders
4. RestWithBody
  4.1. JSON
  4.2. []byte
5. ReqBody
6. ReqBodyContains
7. ReqPathParam
8. ReqQueryParam
  8.1. Query string
  8.2. POST form param
9. Req headers

10. ALL IN ONE

*/

type testBody struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type suite struct {
	name   string
	mockFn func() *TestServer
	testFn func(host string) (*http.Response, error)
	want   want
}

type want struct {
	statusCode int
	respHeader http.Header
	respBody   []byte
}

func TestTestServer(t *testing.T) {
	suites := []suite{
		{
			name: "Simple one request",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("GET").Path("/sample").
					RespWithStatus(200))
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				return http.Get(host + "/sample")
			},
			want: want{
				statusCode: 200,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
		{
			name: "Multiple requests",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("GET").Path("/sample").Times(5).
					RespWithStatus(200))
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				var resp *http.Response
				var err error

				for i := 0; i < 5; i++ {
					resp, err = http.Get(host + "/sample")
					if err != nil {
						return resp, err
					}
				}
				return resp, err
			},
			want: want{
				statusCode: 200,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
		{
			name: "RespWithHeaders",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("GET").Path("/sample").
					RespWithStatus(200).
					RespWithHeader("X-Test-Header-1", "x-test-value-1").
					RespWithHeader("X-Test-Header-1", "x-test-value-2").
					RespWithHeader("X-Test-Header-3", "x-test-value-3"),
				)
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				return http.Get(host + "/sample")
			},
			want: want{
				statusCode: 200,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length":  {"0"},
					"X-Test-Header-1": {"x-test-value-1", "x-test-value-2"},
					"X-Test-Header-3": {"x-test-value-3"},
				},
			},
		},
		{
			name: "RestWithBody-JSON",
			mockFn: func() *TestServer {
				ts := New(t)

				body := testBody{
					Name: "John Doe",
					Age:  24,
				}

				ts.Add(EXPECT().Method("GET").Path("/sample").
					RespWithStatus(200).
					RespWithHeader("Content-Type", "application/json").
					RespWithBody(body),
				)
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				return http.Get(host + "/sample")
			},
			want: want{
				statusCode: 200,
				respBody:   []byte(`{"name":"John Doe","age":24}`),
				respHeader: http.Header{
					"Content-Length": {"28"},
					"Content-Type":   {"application/json"},
				},
			},
		},
		{
			name: "RestWithBody-[]byte",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("GET").Path("/sample").
					RespWithStatus(200).
					RespWithBody([]byte("assume html")),
				)
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				return http.Get(host + "/sample")
			},
			want: want{
				statusCode: 200,
				respBody:   []byte("assume html"),
				respHeader: http.Header{
					"Content-Length": {"11"},
					"Content-Type":   {"text/plain; charset=utf-8"},
				},
			},
		},
		{
			name: "ReqBody",
			mockFn: func() *TestServer {
				ts := New(t)

				carl := testBody{
					Name: "Carl Cox",
					Age:  60}

				ts.Add(EXPECT().Method("POST").Path("/sample").
					ReqBody(carl).
					RespWithStatus(200),
				)
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				carl := testBody{
					Name: "Carl Cox",
					Age:  60,
				}
				jsonBody, _ := json.Marshal(carl)
				body := bytes.NewBuffer(jsonBody)

				return http.Post(host+"/sample", "application/json", body)
			},
			want: want{
				statusCode: 200,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
		{
			name: "ReqBodyContains",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("POST").Path("/sample").
					ReqBodyContains("important info").
					RespWithStatus(200),
				)
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				body := bytes.NewBufferString("...........vary long string with important info...........")

				return http.Post(host+"/sample", "text/plain", body)
			},
			want: want{
				statusCode: 200,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
		{
			name: "ReqPathParam",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("GET").Path("/api/v1/client/{client_id}").
					ReqPathParam("client_id", "1337").
					RespWithStatus(200),
				)
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				return http.Get(host + "/api/v1/client/1337")
			},
			want: want{
				statusCode: 200,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
		{
			name: "ReqQueryParam-query_string",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("GET").Path("/api/v1/clients").
					ReqQueryParam("employee", "false").
					ReqQueryParam("active", "true").
					RespWithStatus(200),
				)
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				return http.Get(host + "/api/v1/clients?employee=false&active=true")
			},
			want: want{
				statusCode: 200,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
		{
			name: "ReqQueryParam-form-params",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("POST").Path("/api/v1/clients").
					ReqQueryParam("employee", "false").
					ReqQueryParam("active", "true").
					RespWithStatus(200),
				)
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				body := url.Values{}
				body.Set("employee", "false")
				body.Set("active", "true")

				data := strings.NewReader(body.Encode())
				return http.Post(host+"/api/v1/clients", "application/x-www-form-urlencoded", data)
			},
			want: want{
				statusCode: 200,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
		{
			name: "ReqQueryParam-form-params",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("GET").Path("/sample").
					ReqHeader("X-Rq-Header-1", "x-rq-value-1").
					ReqHeader("X-Rq-Header-2", "x-rq-value-2").
					ReqHeader("X-Rq-Header-3", "x-rq-value-3").
					RespWithStatus(200),
				)
				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				uri, _ := url.Parse(host + "/sample")
				req := http.Request{
					Method: "GET",
					URL:    uri,
					Header: map[string][]string{
						"X-Rq-Header-1": {"x-rq-value-1"},
						"X-Rq-Header-2": {"x-rq-value-2"},
						"X-Rq-Header-3": {"x-rq-value-3"},
					}}

				return http.DefaultClient.Do(&req)
			},
			want: want{
				statusCode: 200,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
		{
			name: "ALL IN",
			mockFn: func() *TestServer {
				ts := New(t)

				ts.Add(EXPECT().Method("POST").Path("/1").
					ReqHeader("Header-1", "value-1").
					ReqBody([]byte("1")).
					RespWithStatus(200).
					RespWithBody([]byte("1")).
					Times(3),
				)

				ts.Add(EXPECT().Method("GET").Path("/2").
					ReqQueryParam("active", "true").
					ReqQueryParam("gt", "8").
					RespWithStatus(200).
					RespWithBody([]byte("1")).
					Times(2),
				)

				ts.Add(EXPECT().Method("PUT").Path("/3").
					RespWithStatus(500).
					Times(1),
				)

				ts.Start()

				return ts
			},
			testFn: func(host string) (*http.Response, error) {
				uri1, _ := url.Parse(host + "/1")
				req1 := http.Request{
					Method: "POST",
					URL:    uri1,
					Header: map[string][]string{
						"Header-1": {"value-1"},
					},
					Body: io.NopCloser(bytes.NewBufferString("1")),
				}

				for i := 0; i < 3; i++ {
					_, err := http.DefaultClient.Do(&req1)
					if err != nil {
						t.Errorf("fail")
					}
				}

				uri2, _ := url.Parse(host + "/2?active=true&gt=8")
				req2 := http.Request{
					Method: "GET",
					URL:    uri2,
				}

				for i := 0; i < 2; i++ {
					_, err := http.DefaultClient.Do(&req2)
					if err != nil {
						t.Errorf("fail")
					}
				}

				uri3, _ := url.Parse(host + "/3")
				req3 := http.Request{
					Method: "PUT",
					URL:    uri3,
				}

				return http.DefaultClient.Do(&req3)
			},
			want: want{
				statusCode: 500,
				respBody:   nil,
				respHeader: http.Header{
					"Content-Length": {"0"},
				},
			},
		},
	}

	for _, s := range suites {
		s := s
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			ts := s.mockFn()
			defer ts.Stop()

			resp, err := s.testFn(ts.URL)
			if err != nil {
				t.Errorf("test failed: %v", err)
			}
			defer resp.Body.Close()
			gotBody, _ := io.ReadAll(resp.Body)

			if bytes.Compare(s.want.respBody, gotBody) != 0 {
				t.Errorf("body doesn't match expectation\nGot: %s\nWant: %s\n", gotBody, s.want.respBody)
			}
			if !reflect.DeepEqual(s.want.respHeader, resp.Header) {
				t.Errorf("headers doesn't match expectation\nGot: %s\nWant: %s\n", resp.Header, s.want.respHeader)
			}
			if resp.StatusCode != s.want.statusCode {
				t.Errorf("resp status doesn't match expectation\nGot: %d\nWant: %d\n", resp.StatusCode, s.want.statusCode)
			}
		})
	}
}

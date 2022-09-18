package testserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ExpectationBuilder help to build expectations for http request and response
type ExpectationBuilder struct {
	method     string
	urlPattern string

	reqHeaders     http.Header
	reqExactBody   []byte
	reqContainBody []byte
	reqPathParams  map[string]string
	reqQueryParams map[string]string

	respStatus  int
	respHeaders http.Header
	respBody    []byte

	times int
}

// EXPECT initiate ExpectationBuilder object
func EXPECT() *ExpectationBuilder {
	return &ExpectationBuilder{times: 1}
}

// Times allows you to set how many times endpoint
// should be called with specified parameters
// by default EXPECT is waiting for 1 call
func (eb *ExpectationBuilder) Times(t int) *ExpectationBuilder {
	eb.times = t
	return eb
}

// Method allows you to set method for Path
func (eb *ExpectationBuilder) Method(method string) *ExpectationBuilder {
	eb.method = method
	return eb
}

// Path allows you to set request path in terms of http.ServeMux
// Path is required
func (eb *ExpectationBuilder) Path(url string) *ExpectationBuilder {
	eb.urlPattern = url
	return eb
}

// RespWithStatus specify http response status
func (eb *ExpectationBuilder) RespWithStatus(s int) *ExpectationBuilder {
	eb.respStatus = s
	return eb
}

// RespWithHeader allows you to set response headers in key value format
// it can be called multiple times.
//
// For example:
//
// EXPECT().Method("Get").Path("/api").
// RespWithHeader("x-request-id", "123").
// RespWithHeader("Server" "Apache")
func (eb *ExpectationBuilder) RespWithHeader(k, v string) *ExpectationBuilder {
	if eb.respHeaders == nil {
		eb.respHeaders = make(http.Header)
	}
	eb.respHeaders[k] = append(eb.respHeaders[k], v)
	return eb
}

// RespWithBody allows you to set response body. You can use 2 time of args:
//  1. []byte type allow to set byte slice that will return to the response body
//  2. An any object that can be marshall to json
func (eb *ExpectationBuilder) RespWithBody(b interface{}) *ExpectationBuilder {
	if body, ok := b.([]byte); ok {
		eb.respBody = body
		return eb
	}
	if body, err := json.Marshal(b); err != nil {
		panic(err)
	} else {
		eb.respBody = body
	}
	return eb
}

// ReqBody allows you to set the request body to which the actual
// request body should correspond exactly. You can use 2 time of args:
//  1. []byte type allow to set byte slice that will return to the response body
//  2. An any object that can be marshall to json
func (eb *ExpectationBuilder) ReqBody(b interface{}) *ExpectationBuilder {
	if body, ok := b.([]byte); ok {
		eb.respBody = body
		return eb
	}
	if body, err := json.Marshal(b); err != nil {
		panic(err)
	} else {
		eb.reqExactBody = body
	}
	return eb
}

// ReqBodyContains allows you to specify the string that should be found in the request body
func (eb *ExpectationBuilder) ReqBodyContains(s string) *ExpectationBuilder {
	eb.reqContainBody = []byte(s)
	return eb
}

// ReqPathParam allows you to specify the params of request path.
//
// For example if you use builder like this:
//
// expectation := EXPECT().Method("GET").Path("/api/v1/client/{client_id}")
//
// then
//
// expectation.ReqParam("client_id", "123")
// ts.Add(expectation)
//
// allow you to check that the request path parameter client_id is 123
func (eb *ExpectationBuilder) ReqPathParam(k, v string) *ExpectationBuilder {
	if eb.reqPathParams == nil {
		eb.reqPathParams = make(map[string]string)
	}
	eb.reqPathParams[k] = v
	return eb
}

// ReqQueryParam allows you to specify the params of request query string
// or request body form data.
// POST and PUT body parameters take precedence over URL query string values.
func (eb *ExpectationBuilder) ReqQueryParam(k, v string) *ExpectationBuilder {
	if eb.reqQueryParams == nil {
		eb.reqQueryParams = make(map[string]string)
	}
	eb.reqQueryParams[k] = v
	return eb
}

// ReqHeader allows you to set request headers expectation in key value format
// it can be called multiple times.
func (eb *ExpectationBuilder) ReqHeader(k, v string) *ExpectationBuilder {
	if eb.reqHeaders == nil {
		eb.reqHeaders = make(http.Header)
	}
	eb.reqHeaders[k] = append(eb.reqHeaders[k], v)
	return eb
}

/*
TestServer is a structure that contains a set of expectations and a server from the httptest package.
You should use a new instance of the test server in each test.
The algorithm for using the test server is as follows:

	  func TestFunc(t *testing.T) {
	    // Create new instance
	    ts := testserver.New(t)

		// Add some expectations
	    ts.Add(EXPECT().Method("GET").Path("/api/profile").RespWithStatus(200).RespWithBody(profile))
	    ts.Add(EXPECT().Method("POST").Path("/api/order").RespWithStatus(200).RespWithBody(done).Times(2))
	    ...

		// Start test Server
	    ts.Start()
	    defer ts.Stop()

		// Create an instance of the object under test
	    service := order.New(ts.URL)

		// Call the method under test
		gotRs, gotErr := service.CreateOrder(args...)

		// Check the results
		if wantRs != gotRs {
			t.Error()
		}
		if wantErr != gotErr {
			t.Error()
		}
	  }
*/
type TestServer struct {
	t *testing.T
	*httptest.Server
	cases map[string]*testcase
}

type testcase struct {
	n map[int]http.HandlerFunc

	wantedCalls int
	actualCalls int
	fails       []string
}

func (tc *testcase) checkMethod(path, want, got string) {
	if got != want {
		tc.fails = append(tc.fails,
			fmt.Sprintf("wrong request method\nGot: %s %s\nWant: %s %s\n",
				got, path, want, path))
	}
}

func (tc *testcase) checkBody(method, path string, want, got []byte) {
	if want != nil {
		if bytes.Compare(got, want) != 0 {
			tc.fails = append(tc.fails,
				fmt.Sprintf("%s %s expect different reqeust body\nGot: %s\nWant: %s\n",
					method, path, got, want))
		}
	}
}

func (tc *testcase) checkBodyContains(method, path string, want, got []byte) {
	if want != nil {
		if !bytes.Contains(got, want) {
			tc.fails = append(tc.fails,
				fmt.Sprintf("%s %s reqeust body doesnt contain expected value \nGot: %s\nWant contains: %s\n",
					method, path, got, want))
		}
	}
}

func (tc *testcase) checkPathParams(method, path string, want, got map[string]string) {
	if want != nil {
		for wantK, wantV := range want {
			if gotV, ok := got[wantK]; !ok {
				tc.fails = append(tc.fails,
					fmt.Sprintf("%s %s check your expectations: there is no %s path param", method, path, wantK))
			} else if gotV != wantV {
				tc.fails = append(tc.fails,
					fmt.Sprintf("%s %s path param %q doesn't match expectation \nGot: %s\nWant: %s",
						method, path, wantK, gotV, wantV))
			}
		}
	}
}

func (tc *testcase) checkQueryParams(method, path string, want map[string]string, r *http.Request) {
	if want != nil {
		for wantK, wantV := range want {
			if gotV := r.FormValue(wantK); gotV != wantV {
				tc.fails = append(tc.fails,
					fmt.Sprintf("%s %s query param %s doesn't match expectation \nGot: %s\nWant: %s",
						method, path, wantK, gotV, wantV))
			}
		}
	}
}

func (tc *testcase) checkHeaders(method, path string, want, got http.Header) {
	if want != nil {
		for wantK, wantVV := range want {
			if gotVV, ok := got[wantK]; !ok {
				tc.fails = append(tc.fails,
					fmt.Sprintf("%s %s there is no %s request header", method, path, wantK))
			} else {
				for _, wantV := range wantVV {
					if !contains(gotVV, wantV) {
						tc.fails = append(tc.fails,
							fmt.Sprintf("%s %s request header %q doesn't match expectation \nGot: %s\nWant: %s",
								method, path, wantK, gotVV, wantV))
					}
				}
			}
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// New create new instance of TestServer
func New(t *testing.T) *TestServer {
	return &TestServer{
		t:     t,
		cases: make(map[string]*testcase),
	}
}

// Add new expectation on TestServer instance
func (ts *TestServer) Add(eb *ExpectationBuilder) {
	tc, ok := ts.cases[eb.urlPattern]
	if !ok {
		tc = &testcase{n: make(map[int]http.HandlerFunc)}
		ts.cases[eb.urlPattern] = tc
	}

	for i := 0; i < eb.times; i++ {
		tc.n[tc.wantedCalls] = func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			reqBody, _ := io.ReadAll(r.Body)
			m := r.Method
			p := r.URL.Path

			tc.checkMethod(p, eb.method, m)
			tc.checkBody(m, p, eb.reqExactBody, reqBody)
			tc.checkBodyContains(m, p, eb.reqContainBody, reqBody)
			tc.checkPathParams(m, p, eb.reqPathParams, mux.Vars(r))
			tc.checkQueryParams(m, p, eb.reqQueryParams, r)
			tc.checkHeaders(m, p, eb.reqHeaders, r.Header)

			for k, vv := range eb.respHeaders {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.Header()["Date"] = nil
			w.WriteHeader(eb.respStatus)
			w.Write(eb.respBody)
		}

		tc.wantedCalls++
	}
}

// Start TestServer
func (ts *TestServer) Start() {
	router := mux.NewRouter()
	router.NotFoundHandler = ts.notFound()
	for url, c := range ts.cases {
		url := url
		c := c
		router.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
			if handler, ok := c.n[c.actualCalls]; !ok {
				c.fails = append(c.fails,
					fmt.Sprintf("unexpected call %s %s\nWant calls: %d\nGot calls: %d\n",
						r.Method, url, c.wantedCalls, c.actualCalls+1))
			} else {
				handler(w, r)
				c.actualCalls++
			}
		})
	}
	ts.Server = httptest.NewServer(router)
}

func (ts *TestServer) notFound() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.t.Errorf("unexpected call %s %s", r.Method, r.URL)
	})
}

// Stop TestServer and check expectations
func (ts *TestServer) Stop() {
	for url, c := range ts.cases {
		if c.wantedCalls != c.actualCalls {
			ts.t.Errorf("there is %d calls %s, wanted: %d calls", c.actualCalls, url, c.wantedCalls)
		}
		for _, fail := range c.fails {
			ts.t.Error(fail)
		}
	}
	ts.Close()
}

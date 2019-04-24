package bugsnag

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRequestInformationGetsExtracted(t *testing.T) {
	contexts := make(chan context.Context, 1)
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = AttachRequestData(ctx, r)
		contexts <- ctx
	})
	ts := httptest.NewServer(hf)
	defer ts.Close()
	http.Get(ts.URL + "/1234abcd?fish=bird")

	reqJSON, req := extractRequestInfo(<-contexts)
	if reqJSON.ClientIP == "" {
		t.Errorf("expected to find an IP address for the request but was blank")
	}
	if got, exp := reqJSON.HTTPMethod, "GET"; got != exp {
		t.Errorf("expected HTTP method to be '%s' but was '%s'", exp, got)
	}
	if got, exp := req.URL.Path, "/1234abcd"; got != exp {
		t.Errorf("expected request URL to be '%s' but was '%s'", exp, got)
	}
	if got, exp := reqJSON.URL, "/1234abcd?fish=bird"; !strings.Contains(got, exp) {
		t.Errorf("expected request URL to contain '%s' but was '%s'", exp, got)
	}
	if got, exp := reqJSON.Referer, ""; got != exp {
		t.Errorf("expected request referer to be '%s' but was '%s'", exp, got)
	}
	if got, exp := reqJSON.Headers["Accept-Encoding"], "gzip"; got != exp {
		t.Errorf("expected Accept-Encoding to be '%s' but was '%s'", exp, got)
	}
	if got, exp := reqJSON.Headers["User-Agent"], "Go-http-client"; !strings.Contains(got, exp) {
		t.Errorf("expected user agent to contain '%s' but was '%s'", exp, got)
	}
}

func TestRequestExtractorCanHandleAbsentContext(t *testing.T) {
	if got, _ := extractRequestInfo(nil); got != nil {
		//really just testing that nothing panics here
		t.Errorf("expected nil contexts to give nil sub-objects, but was '%s'", got)
	}
	if got, _ := extractRequestInfo(context.Background()); got != nil {
		//really just testing that nothing panics here
		t.Errorf("expected contexts without requst info to give nil sub-objects, but was '%s'", got)
	}
}

func TestExtractRequestInfoFromReq_RedactURL(t *testing.T) {
	testCases := []struct{ in, exp string }{
		{in: "", exp: "http://example.com"},
		{in: "/", exp: "http://example.com/"},
		{in: "/foo.html", exp: "http://example.com/foo.html"},
		{in: "/foo.html?q=something&bar=123", exp: "http://example.com/foo.html?q=something&bar=123"},
		{in: "/foo.html?foo=1&foo=2&foo=3", exp: "http://example.com/foo.html?foo=1&foo=2&foo=3"},

		// Invalid query string.
		{in: "/foo?%", exp: "http://example.com/foo?%"},

		// Query params contain secrets
		{in: "/foo.html?access_token=something", exp: "http://example.com/foo.html?access_token=FILTERED"},
		{in: "/foo.html?access_token=something&access_token=", exp: "http://example.com/foo.html?access_token=FILTERED&access_token="},
	}

	for _, tc := range testCases {
		parsedURL, err := url.Parse(tc.in)
		if err != nil {
			t.Fatalf("error parsing originalURI (bad test): %v", err)
		}

		req := &http.Request{
			Host: "example.com",
			URL:  parsedURL,
		}
		result := extractRequestInfoFromReq(req)
		if result.URL != tc.exp {
			t.Errorf("expected URL to be '%s' but was '%s'", tc.exp, result.URL)
		}
	}
}

func TestParseHeadersWillSanitiseIllegalParams(t *testing.T) {
	headers := make(map[string][]string)
	headers["password"] = []string{"correct horse battery staple"}
	headers["secret"] = []string{"I am Banksy"}
	headers["authorization"] = []string{"licence to kill -9"}
	headers["custom-made-secret"] = []string{"I'm the insider at Sotheby's"}
	for k, v := range parseRequestHeaders(headers) {
		if v != "[FILTERED]" {
			t.Errorf("expected '%s' to be [FILTERED], but was '%s'", k, v)
		}
	}
}

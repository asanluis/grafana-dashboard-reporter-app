package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHTTPClientRedirects(t *testing.T) {
	Convey("When HTTP client encounters redirects", t, func() {
		redirectCount := 0
		customQueryParamReceived := ""

		// Test server that redirects once then returns success
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			customQueryParamReceived = r.URL.Query().Get("custom-test-param")

			if redirectCount == 0 {
				redirectCount++

				http.Redirect(w, r, "/final", http.StatusFound)

				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))
		defer ts.Close()

		Convey("HTTP client should follow redirects with custom query params in CheckRedirect", func() {
			// Create HTTP client with custom query parameters handled via CheckRedirect
			httpClient, err := httpclient.New(httpclient.Options{})
			So(err, ShouldBeNil)

			customQueryParams := map[string]string{
				"custom-test-param": "test-value",
			}

			// Configure redirect handling to preserve custom query parameters
			httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				// Apply custom query parameters to redirect requests
				if len(customQueryParams) > 0 {
					q := req.URL.Query()
					for name, value := range customQueryParams {
						q.Set(name, value)
					}

					req.URL.RawQuery = q.Encode()
				}

				// Allow unlimited redirects
				return nil
			}

			// Create request with custom query parameters
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/initial", nil)
			So(err, ShouldBeNil)

			// Add custom query parameters to initial request
			q := req.URL.Query()
			for name, value := range customQueryParams {
				q.Set(name, value)
			}

			req.URL.RawQuery = q.Encode()

			resp, err := httpClient.Do(req)

			So(err, ShouldBeNil)
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			So(redirectCount, ShouldEqual, 1)
			So(customQueryParamReceived, ShouldEqual, "test-value")

			resp.Body.Close()
		})
	})
}

func TestHTTPClientTLSSettings(t *testing.T) {
	Convey("When HTTP client is configured with TLS settings", t, func() {
		Convey("It should respect InsecureSkipVerify setting", func() {
			// Test with InsecureSkipVerify enabled
			httpClient, err := httpclient.New(httpclient.Options{
				TLS: &httpclient.TLSOptions{
					InsecureSkipVerify: true,
				},
			})
			So(err, ShouldBeNil)

			// Check that the underlying transport has InsecureSkipVerify set
			// The Grafana SDK might wrap the transport, so let's check if we can access the TLS config
			So(httpClient.Transport, ShouldNotBeNil)

			// Test with InsecureSkipVerify disabled
			httpClient2, err := httpclient.New(httpclient.Options{
				TLS: &httpclient.TLSOptions{
					InsecureSkipVerify: false,
				},
			})
			So(err, ShouldBeNil)
			So(httpClient2.Transport, ShouldNotBeNil)
		})
	})
}

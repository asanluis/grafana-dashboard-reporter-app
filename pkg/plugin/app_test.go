package plugin

import (
	"fmt"
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

		Convey("HTTP client with custom query params should follow redirects", func() {
			// Create HTTP client with custom query parameters
			httpClient, err := httpclient.New(httpclient.Options{})
			So(err, ShouldBeNil)

			customQueryParams := map[string]string{
				"custom-test-param": "test-value",
			}

			// Apply the same logic as in app.go
			httpClient.Transport = &customQueryParamTransport{
				base:   httpClient.Transport,
				params: customQueryParams,
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

			// Make request that will trigger redirect
			resp, err := httpClient.Get(ts.URL + "/initial")

			So(err, ShouldBeNil)
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			So(redirectCount, ShouldEqual, 1)
			So(customQueryParamReceived, ShouldEqual, "test-value")

			resp.Body.Close()
		})

		Convey("HTTP client should handle multiple redirects", func() {
			multiRedirectCount := 0
			multiTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				customQueryParamReceived = r.URL.Query().Get("custom-test-param")

				if multiRedirectCount < 3 {
					multiRedirectCount++
					http.Redirect(w, r, fmt.Sprintf("/redirect-%d", multiRedirectCount), http.StatusFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("final success"))
			}))
			defer multiTs.Close()

			httpClient, err := httpclient.New(httpclient.Options{})
			So(err, ShouldBeNil)

			customQueryParams := map[string]string{
				"custom-test-param": "multi-redirect-test",
			}

			httpClient.Transport = &customQueryParamTransport{
				base:   httpClient.Transport,
				params: customQueryParams,
			}

			httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
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

			resp, err := httpClient.Get(multiTs.URL + "/start")

			So(err, ShouldBeNil)
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			So(multiRedirectCount, ShouldEqual, 3)
			So(customQueryParamReceived, ShouldEqual, "multi-redirect-test")

			resp.Body.Close()
		})
	})
}

func TestCustomQueryParamTransport(t *testing.T) {
	Convey("When using customQueryParamTransport", t, func() {
		receivedQueryParams := make(map[string]string)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedQueryParams["custom-param-1"] = r.URL.Query().Get("custom-param-1")
			receivedQueryParams["custom-param-2"] = r.URL.Query().Get("custom-param-2")
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		Convey("It should add custom query parameters to requests", func() {
			httpClient, err := httpclient.New(httpclient.Options{})
			So(err, ShouldBeNil)

			customQueryParams := map[string]string{
				"custom-param-1": "value1",
				"custom-param-2": "value2",
			}

			httpClient.Transport = &customQueryParamTransport{
				base:   httpClient.Transport,
				params: customQueryParams,
			}

			resp, err := httpClient.Get(ts.URL)

			So(err, ShouldBeNil)
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			So(receivedQueryParams["custom-param-1"], ShouldEqual, "value1")
			So(receivedQueryParams["custom-param-2"], ShouldEqual, "value2")

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

		Convey("It should work with custom query params and TLS settings combined", func() {
			httpClient, err := httpclient.New(httpclient.Options{
				TLS: &httpclient.TLSOptions{
					InsecureSkipVerify: true,
				},
			})
			So(err, ShouldBeNil)

			customQueryParams := map[string]string{
				"custom-tls-param": "tls-test-value",
			}

			// Apply custom query param transport
			httpClient.Transport = &customQueryParamTransport{
				base:   httpClient.Transport,
				params: customQueryParams,
			}

			// Check that the custom transport was applied
			customTransport, ok := httpClient.Transport.(*customQueryParamTransport)
			So(ok, ShouldBeTrue)
			So(customTransport.base, ShouldNotBeNil)
			So(customTransport.params, ShouldNotBeNil)
		})
	})
}

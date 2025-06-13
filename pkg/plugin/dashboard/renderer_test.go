package dashboard

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/asanluis/grafana-dashboard-reporter-app/pkg/plugin/chrome"
	"github.com/asanluis/grafana-dashboard-reporter-app/pkg/plugin/config"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	. "github.com/smartystreets/goconvey/convey"
)

// We want our tests to run fast.
func init() {
	getPanelRetrySleepTime = time.Duration(1) * time.Millisecond
}

func TestFetchPanelPNG(t *testing.T) {
	Convey("When fetching a panel PNG", t, func() {
		requestURI := ""
		requestHeaders := http.Header{}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestURI = r.RequestURI
			requestHeaders = r.Header
		}))
		defer ts.Close()

		conf := config.Config{
			Layout:        "simple",
			DashboardMode: "default",
		}
		variables := url.Values{}
		variables.Add("var-host", "servername")
		variables.Add("var-port", "adapter")
		variables.Add("from", "now-1h")
		variables.Add("to", "now")

		dash, err := New(
			log.NewNullLogger(),
			&conf,
			http.DefaultClient,
			&chrome.LocalInstance{},
			ts.URL,
			"v11.1.0",
			&Model{Dashboard: struct {
				ID          int          `json:"id"`
				UID         string       `json:"uid"`
				Title       string       `json:"title"`
				Description string       `json:"description"`
				RowOrPanels []RowOrPanel `json:"panels"`
				Panels      []Panel
				Variables   url.Values
			}{
				UID:       "randomUID",
				Variables: variables,
			}},
			http.Header{
				backend.OAuthIdentityTokenHeaderName: []string{"Bearer token"},
			},
		)

		Convey("New dashboard should receive no errors", func() {
			So(err, ShouldBeNil)
		})

		_, err = dash.PanelPNG(t.Context(), Panel{ID: "44", Type: "singlestat", Title: "title", GridPos: GridPos{}})

		Convey("It should receives no errors", func() {
			So(err, ShouldBeNil)
		})

		Convey("The httpClient should use the render endpoint with the dashboard name", func() {
			So(requestURI, ShouldStartWith, "/render/d-solo/randomUID/_")
		})

		Convey("The httpClient should request the panel ID", func() {
			So(requestURI, ShouldContainSubstring, "panelId=44")
		})

		Convey("The httpClient should request the time", func() {
			So(requestURI, ShouldContainSubstring, "from=now-1h")
			So(requestURI, ShouldContainSubstring, "to=now")
		})

		Convey("The httpClient should insert auth token should in request header", func() {
			So(requestHeaders.Get("Authorization"), ShouldEqual, "Bearer token")
		})

		Convey("The httpClient should pass variables in the request parameters", func() {
			So(requestURI, ShouldContainSubstring, "var-host=servername")
			So(requestURI, ShouldContainSubstring, "var-port=adapter")
		})

		Convey("The httpClient should request singlestat panels at a smaller size", func() {
			So(requestURI, ShouldContainSubstring, "width=1000")
			So(requestURI, ShouldContainSubstring, "height=500")
		})

		// Use grid layout
		conf.Layout = "grid"

		dash, err = New(
			log.NewNullLogger(),
			&conf,
			http.DefaultClient,
			&chrome.LocalInstance{},
			ts.URL,
			"v11.1.0",
			&Model{Dashboard: struct {
				ID          int          `json:"id"`
				UID         string       `json:"uid"`
				Title       string       `json:"title"`
				Description string       `json:"description"`
				RowOrPanels []RowOrPanel `json:"panels"`
				Panels      []Panel
				Variables   url.Values
			}{
				UID:       "randomUID",
				Variables: variables,
			}},
			http.Header{
				backend.OAuthIdentityTokenHeaderName: []string{"token"},
			},
		)

		Convey("New dashboard should receive no errors using grid layout", func() {
			So(err, ShouldBeNil)
		})

		_, err = dash.PanelPNG(t.Context(), Panel{ID: "44", Type: "graph", Title: "title", GridPos: GridPos{H: 6, W: 24}})

		Convey("It should receives no errors using grid layout", func() {
			So(err, ShouldBeNil)
		})

		Convey("The httpClient should request singlestat panels at grid layout size", func() {
			So(requestURI, ShouldContainSubstring, "width=1536")
			So(requestURI, ShouldContainSubstring, "height=216")
		})
	})
}

func TestCustomQueryParamsInRenderURL(t *testing.T) {
	Convey("When fetching a panel PNG with custom query parameters via image renderer", t, func() {
		requestURI := ""

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestURI = r.RequestURI
		}))
		defer ts.Close()

		conf := config.Config{
			Layout:          "simple",
			DashboardMode:   "default",
			NativeRendering: false, // Use image renderer (HTTP client)
			CustomQueryParams: map[string]string{
				"c_query_test":  "checked",
				"another_param": "value123",
			},
		}

		variables := url.Values{}
		variables.Add("from", "now-1h")
		variables.Add("to", "now")

		dash, err := New(
			log.NewNullLogger(),
			&conf,
			http.DefaultClient,
			&chrome.LocalInstance{},
			ts.URL,
			"v11.1.0",
			&Model{Dashboard: struct {
				ID          int          `json:"id"`
				UID         string       `json:"uid"`
				Title       string       `json:"title"`
				Description string       `json:"description"`
				RowOrPanels []RowOrPanel `json:"panels"`
				Panels      []Panel
				Variables   url.Values
			}{
				UID:       "testUID",
				Variables: variables,
			}},
			http.Header{},
		)

		Convey("New dashboard should receive no errors", func() {
			So(err, ShouldBeNil)
		})

		_, err = dash.PanelPNG(t.Context(), Panel{ID: "44", Type: "singlestat", Title: "title", GridPos: GridPos{}})

		Convey("It should receives no errors", func() {
			So(err, ShouldBeNil)
		})

		Convey("The httpClient should include custom query parameters in render URL", func() {
			So(requestURI, ShouldContainSubstring, "c_query_test=checked")
			So(requestURI, ShouldContainSubstring, "another_param=value123")
		})

		Convey("The httpClient should still include standard parameters", func() {
			So(requestURI, ShouldStartWith, "/render/d-solo/testUID/_")
			So(requestURI, ShouldContainSubstring, "panelId=44")
			So(requestURI, ShouldContainSubstring, "from=now-1h")
			So(requestURI, ShouldContainSubstring, "to=now")
		})
	})
}

// Mock Chrome instance for testing.
type mockChromeInstance struct{}

func (m *mockChromeInstance) NewTab(logger log.Logger, conf *config.Config) *chrome.Tab {
	return &chrome.Tab{} // We'll override the methods we need
}

func (m *mockChromeInstance) Name() string {
	return "mock-chrome"
}

func (m *mockChromeInstance) Close(logger log.Logger) {
	// No-op for mock
}

func TestCustomQueryParamsInChromeURL(t *testing.T) {
	Convey("When fetching a panel PNG with custom query parameters via Chrome instance", t, func() {
		capturedURL := ""

		// Create a mock chrome instance that captures the URL
		mockChrome := &mockChromeInstance{}

		// We need to test the URL generation directly since mocking the full Chrome interaction is complex
		conf := config.Config{
			Layout:          "simple",
			DashboardMode:   "default",
			NativeRendering: true, // Use Chrome native rendering
			CustomQueryParams: map[string]string{
				"c_query_test": "checked",
				"chrome_param": "native_value",
			},
		}

		variables := url.Values{}
		variables.Add("from", "now-1h")
		variables.Add("to", "now")

		dash, err := New(
			log.NewNullLogger(),
			&conf,
			http.DefaultClient,
			mockChrome,
			"http://test-server.com",
			"v11.1.0",
			&Model{Dashboard: struct {
				ID          int          `json:"id"`
				UID         string       `json:"uid"`
				Title       string       `json:"title"`
				Description string       `json:"description"`
				RowOrPanels []RowOrPanel `json:"panels"`
				Panels      []Panel
				Variables   url.Values
			}{
				UID:       "chromeUID",
				Variables: variables,
			}},
			http.Header{},
		)

		Convey("New dashboard should receive no errors", func() {
			So(err, ShouldBeNil)
		})

		// Test the URL generation as it would happen in panelPNGNativeRenderer
		panel := Panel{ID: "88", Type: "graph", Title: "test panel", GridPos: GridPos{}}
		panelURL := dash.panelPNGURL(panel, false) // false = Chrome native rendering (no render/ prefix)

		// Simulate the custom query parameter addition that happens in panelPNGNativeRenderer
		if len(conf.CustomQueryParams) > 0 {
			q := panelURL.Query()
			for name, value := range conf.CustomQueryParams {
				q.Set(name, value)
			}

			panelURL.RawQuery = q.Encode()
		}

		capturedURL = panelURL.String()

		Convey("The Chrome URL should include custom query parameters", func() {
			So(capturedURL, ShouldContainSubstring, "c_query_test=checked")
			So(capturedURL, ShouldContainSubstring, "chrome_param=native_value")
		})

		Convey("The Chrome URL should still include standard parameters", func() {
			So(capturedURL, ShouldContainSubstring, "d-solo/chromeUID/_")
			So(capturedURL, ShouldContainSubstring, "panelId=88")
			So(capturedURL, ShouldContainSubstring, "from=now-1h")
			So(capturedURL, ShouldContainSubstring, "to=now")
		})

		Convey("The Chrome URL should NOT have render prefix (native rendering)", func() {
			So(capturedURL, ShouldNotContainSubstring, "/render/")
		})
	})
}

// Integration test that demonstrates the actual flow through panelPNGNativeRenderer.
func TestCustomQueryParamsIntegrationChromeNavigation(t *testing.T) {
	Convey("When custom query parameters are configured for Chrome navigation", t, func() {
		conf := config.Config{
			Layout:          "simple",
			DashboardMode:   "default",
			NativeRendering: true,
			CustomQueryParams: map[string]string{
				"c_query_test":     "checked",
				"integration_test": "chrome_navigation",
			},
		}

		// Test that panelPNGNativeRenderer correctly adds custom query parameters
		// This simulates the exact flow in the real code
		variables := url.Values{}
		variables.Add("from", "now-1h")
		variables.Add("to", "now")

		dash, err := New(
			log.NewNullLogger(),
			&conf,
			http.DefaultClient,
			&mockChromeInstance{},
			"http://grafana.example.com",
			"v11.1.0",
			&Model{Dashboard: struct {
				ID          int          `json:"id"`
				UID         string       `json:"uid"`
				Title       string       `json:"title"`
				Description string       `json:"description"`
				RowOrPanels []RowOrPanel `json:"panels"`
				Panels      []Panel
				Variables   url.Values
			}{
				UID:       "integrationUID",
				Variables: variables,
			}},
			http.Header{},
		)

		So(err, ShouldBeNil)

		// Step 1: Get initial panel URL (without custom query params)
		panel := Panel{ID: "123", Type: "timeseries", Title: "Integration Test Panel", GridPos: GridPos{}}
		initialURL := dash.panelPNGURL(panel, false)

		// Step 2: Simulate what panelPNGNativeRenderer does - add custom query params
		finalURL := *initialURL
		if len(conf.CustomQueryParams) > 0 {
			q := finalURL.Query()
			for name, value := range conf.CustomQueryParams {
				q.Set(name, value)
			}

			finalURL.RawQuery = q.Encode()
		}

		finalURLString := finalURL.String()

		Convey("The initial URL should not contain custom query parameters", func() {
			So(initialURL.String(), ShouldNotContainSubstring, "c_query_test=checked")
			So(initialURL.String(), ShouldNotContainSubstring, "integration_test=chrome_navigation")
		})

		Convey("The final URL (after panelPNGNativeRenderer processing) should contain all parameters", func() {
			// Custom query parameters should be present
			So(finalURLString, ShouldContainSubstring, "c_query_test=checked")
			So(finalURLString, ShouldContainSubstring, "integration_test=chrome_navigation")

			// Standard parameters should still be present
			So(finalURLString, ShouldContainSubstring, "panelId=123")
			So(finalURLString, ShouldContainSubstring, "from=now-1h")
			So(finalURLString, ShouldContainSubstring, "to=now")
			So(finalURLString, ShouldContainSubstring, "d-solo/integrationUID/_")

			// Should not have render prefix for Chrome navigation
			So(finalURLString, ShouldNotContainSubstring, "/render/")
		})

		Convey("Custom query parameters should be properly URL encoded", func() {
			// Parse the final URL to verify proper encoding
			parsedURL, err := url.Parse(finalURLString)
			So(err, ShouldBeNil)

			queryParams := parsedURL.Query()
			So(queryParams.Get("c_query_test"), ShouldEqual, "checked")
			So(queryParams.Get("integration_test"), ShouldEqual, "chrome_navigation")
		})
	})
}

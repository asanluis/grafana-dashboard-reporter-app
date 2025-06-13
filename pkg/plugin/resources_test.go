package plugin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strings"
	"testing"

	"github.com/asanluis/grafana-dashboard-reporter-app/pkg/plugin/config"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	. "github.com/smartystreets/goconvey/convey"
)

// mockCallResourceResponseSender implements backend.CallResourceResponseSender
// for use in tests.
type mockCallResourceResponseSender struct {
	response *backend.CallResourceResponse
}

// Send sets the received *backend.CallResourceResponse to s.response.
func (s *mockCallResourceResponseSender) Send(response *backend.CallResourceResponse) error {
	s.response = response

	return nil
}

// Test report resource.
func TestReportResource(t *testing.T) {
	var execPath string

	locations := []string{
		// Mac
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		// Windows
		"chrome.exe",
		// Linux
		"google-chrome",
		"chrome",
	}

	for _, path := range locations {
		found, err := exec.LookPath(path)
		if err == nil {
			execPath = found

			break
		}
	}

	// Skip test if chrome is not available
	if execPath == "" {
		t.Skip("Chrome not found. Skipping test")
	}

	// Initialize app
	inst, err := NewDashboardReporterApp(t.Context(), backend.AppInstanceSettings{
		DecryptedSecureJSONData: map[string]string{
			config.SaToken: "token",
		},
	})
	if err != nil {
		t.Fatalf("new app: %s", err)
	}

	if inst == nil {
		t.Fatal("inst must not be nil")
	}

	app, ok := inst.(*App)
	if !ok {
		t.Fatal("inst must be of type *App")
	}

	Convey("When the report handler is called", t, func() {
		Convey("It should extract dashboard ID from the URL and forward it to the new reporter ", func() {
			var repDashName string

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/dashboards/") {
					urlParts := strings.Split(r.URL.Path, "/")
					repDashName = urlParts[len(urlParts)-1]
				}

				if _, err := w.Write([]byte(`{"dashboard": {"title": "foo","panels":[{"type":"singlestat", "id":0}]}}`)); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)

					return
				}
			}))
			defer ts.Close()

			ctx := backend.WithGrafanaConfig(t.Context(), backend.NewGrafanaCfg(map[string]string{
				backend.AppURL: ts.URL,
			}))

			var r mockCallResourceResponseSender
			err = app.CallResource(ctx, &backend.CallResourceRequest{
				PluginContext: backend.PluginContext{
					OrgID:    3,
					PluginID: "my-plugin",
					User:     &backend.User{Name: "foobar", Email: "foo@bar.com", Login: "foo@bar.com"},
				},
				Method: http.MethodGet,
				Path:   "report?dashUid=testDash",
			}, &r)

			So(repDashName, ShouldEqual, "testDash")
		})
	})
}

func TestFilterTemplateVariables(t *testing.T) {
	Convey("When filtering template variables from query parameters", t, func() {
		app := &App{
			conf: config.Config{},
		}

		Convey("It should exclude system parameters", func() {
			// Create query parameters with both system and template variables
			queryParams := url.Values{}
			queryParams.Add("dashUid", "test-dashboard-uid")
			queryParams.Add("theme", "dark")
			queryParams.Add("layout", "grid")
			queryParams.Add("orientation", "landscape")
			queryParams.Add("from", "now-6h")
			queryParams.Add("to", "now")
			queryParams.Add("panelId", "2")
			queryParams.Add("access_id", "admin")
			queryParams.Add("orgId", "1")
			// Template variables (should be preserved)
			queryParams.Add("var-server", "web01")
			queryParams.Add("var-environment", "production")

			filteredParams := app.filterTemplateVariables(queryParams)

			// System parameters should be filtered out
			So(filteredParams.Has("dashUid"), ShouldBeFalse)
			So(filteredParams.Has("theme"), ShouldBeFalse)
			So(filteredParams.Has("layout"), ShouldBeFalse)
			So(filteredParams.Has("orientation"), ShouldBeFalse)
			So(filteredParams.Has("from"), ShouldBeFalse)
			So(filteredParams.Has("to"), ShouldBeFalse)
			So(filteredParams.Has("panelId"), ShouldBeFalse)
			So(filteredParams.Has("access_id"), ShouldBeFalse)
			So(filteredParams.Has("orgId"), ShouldBeFalse)

			// Template variables should be preserved
			So(filteredParams.Has("var-server"), ShouldBeTrue)
			So(filteredParams.Get("var-server"), ShouldEqual, "web01")
			So(filteredParams.Has("var-environment"), ShouldBeTrue)
			So(filteredParams.Get("var-environment"), ShouldEqual, "production")
		})

		Convey("It should preserve custom parameters that are not system parameters", func() {
			queryParams := url.Values{}
			queryParams.Add("dashUid", "test-uid")          // system parameter - should be filtered
			queryParams.Add("custom-param", "custom-value") // custom parameter - should be preserved
			queryParams.Add("my_var", "my_value")           // custom parameter - should be preserved

			filteredParams := app.filterTemplateVariables(queryParams)

			So(filteredParams.Has("dashUid"), ShouldBeFalse)
			So(filteredParams.Has("custom-param"), ShouldBeTrue)
			So(filteredParams.Get("custom-param"), ShouldEqual, "custom-value")
			So(filteredParams.Has("my_var"), ShouldBeTrue)
			So(filteredParams.Get("my_var"), ShouldEqual, "my_value")
		})

		Convey("It should handle empty query parameters", func() {
			queryParams := url.Values{}

			filteredParams := app.filterTemplateVariables(queryParams)

			So(len(filteredParams), ShouldEqual, 0)
		})

		Convey("It should handle multiple values for the same parameter", func() {
			queryParams := url.Values{}
			queryParams.Add("var-hosts", "host1")
			queryParams.Add("var-hosts", "host2")
			queryParams.Add("dashUid", "test-uid") // system parameter

			filteredParams := app.filterTemplateVariables(queryParams)

			So(filteredParams.Has("dashUid"), ShouldBeFalse)
			So(filteredParams.Has("var-hosts"), ShouldBeTrue)
			hosts := filteredParams["var-hosts"]
			So(len(hosts), ShouldEqual, 2)
			So(hosts[0], ShouldEqual, "host1")
			So(hosts[1], ShouldEqual, "host2")
		})
	})
}

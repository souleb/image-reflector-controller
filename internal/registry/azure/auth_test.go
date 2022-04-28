package azure

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	. "github.com/onsi/gomega"
)

func TestGetAzureLoginAuth(t *testing.T) {
	tests := []struct {
		name            string
		tokenCredential azcore.TokenCredential
		responseBody    string
		statusCode      int
		wantErr         bool
		wantAuthConfig  authn.AuthConfig
	}{
		{
			name:            "success",
			tokenCredential: &FakeTokenCredential{Token: "foo"},
			responseBody:    `{"refresh_token": "bbbbb"}`,
			statusCode:      http.StatusOK,
			wantAuthConfig: authn.AuthConfig{
				Username: "00000000-0000-0000-0000-000000000000",
				Password: "bbbbb",
			},
		},
		{
			name:            "fail to get access token",
			tokenCredential: &FakeTokenCredential{Err: errors.New("no access token")},
			wantErr:         true,
		},
		{
			name:            "error from exchange service",
			tokenCredential: &FakeTokenCredential{Token: "foo"},
			responseBody:    `[{"code": "111","message": "error message 1"}]`,
			statusCode:      http.StatusInternalServerError,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Run a test server.
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}
			srv := httptest.NewServer(http.HandlerFunc(handler))
			t.Cleanup(func() {
				srv.Close()
			})

			// Construct an image repo name against the test server.
			u, err := url.Parse(srv.URL)
			g.Expect(err).ToNot(HaveOccurred())
			image := path.Join(u.Host, "foo/bar:v1")
			ref, err := name.ParseReference(image)
			g.Expect(err).ToNot(HaveOccurred())

			// Configure new client with test token credential.
			c := NewClient().
				WithTokenCredential(tt.tokenCredential).
				WithScheme("http")

			auth, err := c.getLoginAuth(context.TODO(), ref)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			if tt.statusCode == http.StatusOK {
				g.Expect(auth).To(Equal(tt.wantAuthConfig))
			}
		})
	}
}

func TestValidHost(t *testing.T) {
	tests := []struct {
		host   string
		result bool
	}{
		{"foo.azurecr.io", true},
		{"foo.azurecr.cn", true},
		{"foo.azurecr.de", true},
		{"foo.azurecr.us", true},
		{"gcr.io", false},
		{"docker.io", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(ValidHost(tt.host)).To(Equal(tt.result))
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name       string
		autoLogin  bool
		statusCode int
		wantErr    bool
	}{
		{
			name:       "no auto login",
			autoLogin:  false,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "with auto login",
			autoLogin:  true,
			statusCode: http.StatusOK,
		},
		{
			name:       "login failure",
			autoLogin:  true,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Run a test server.
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"refresh_token": "bbbbb"}`))
			}
			srv := httptest.NewServer(http.HandlerFunc(handler))
			t.Cleanup(func() {
				srv.Close()
			})

			// Construct an image repo name against the test server.
			u, err := url.Parse(srv.URL)
			g.Expect(err).ToNot(HaveOccurred())
			image := path.Join(u.Host, "foo/bar:v1")
			ref, err := name.ParseReference(image)
			g.Expect(err).ToNot(HaveOccurred())

			ac := NewClient().
				WithTokenCredential(&FakeTokenCredential{Token: "foo"}).
				WithScheme("http")

			_, err = ac.Login(context.TODO(), tt.autoLogin, image, ref)
			g.Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

package azure

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
)

func TestExchanger_ExchangeACRAccessToken(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		statusCode   int
		wantErr      bool
		wantToken    string
	}{
		{
			name: "successful",
			responseBody: `{
	"access_token": "aaaaa",
	"refresh_token": "bbbbb",
	"resource": "ccccc",
	"token_type": "ddddd"
}`,
			statusCode: http.StatusOK,
			wantToken:  "bbbbb",
		},
		{
			name:       "fail",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:         "invalid response",
			responseBody: "foo",
			statusCode:   http.StatusOK,
			wantErr:      true,
		},
		{
			name: "error response",
			responseBody: `[
	{
		"code": "111",
		"message": "error message 1"
	},
	{
		"code": "112",
		"message": "error message 2"
	}
]`,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}
			srv := httptest.NewServer(http.HandlerFunc(handler))
			t.Cleanup(func() {
				srv.Close()
			})

			ex := newExchanger(srv.URL)
			token, err := ex.ExchangeACRAccessToken("some-access-token")
			g.Expect(err != nil).To(Equal(tt.wantErr))
			if tt.statusCode == http.StatusOK {
				g.Expect(token).To(Equal(tt.wantToken))
			}
		})
	}
}

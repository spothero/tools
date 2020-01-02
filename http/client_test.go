package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// This roundtripper is embedded within the MiddlewareRoundTripper as a noop
type MockRoundTripper struct {
	responseStatusCode int
	createErr          bool
}

func (mockRT MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if mockRT.createErr {
		return nil, fmt.Errorf("error in roundtripper")
	}
	return &http.Response{StatusCode: mockRT.responseStatusCode}, nil
}

func TestNewDefaultClient(t *testing.T) {
	client := NewDefaultClient(NewMetrics(prometheus.NewRegistry(), true), nil)
	assert.NotNil(t, client)
	mrt, ok := client.Transport.(MiddlewareRoundTripper)
	assert.True(t, ok)
	assert.Equal(t, 4, len(mrt.Middleware))
	assert.Equal(t, http.DefaultTransport, mrt.RoundTripper)
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name           string
		roundTripper   http.RoundTripper
		middlewareErr  bool
		respHandlerErr bool
		expectErr      bool
	}{
		{
			"no round tripper results in an error",
			nil,
			false,
			false,
			true,
		},
		{
			"round tripper with no error invokes middleware correctly",
			MockRoundTripper{200, false},
			false,
			false,
			false,
		},
		{
			"round tripper with middleware error returns an error",
			MockRoundTripper{200, false},
			true,
			false,
			true,
		},
		{
			"round tripper with internal roundtripper error returns an error",
			MockRoundTripper{200, true},
			false,
			false,
			true,
		},
		{
			"round tripper with resp handler error returns an error",
			MockRoundTripper{200, false},
			false,
			true,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			middlewareCalled := false
			respHandlerCalled := false
			mockMiddleware := func(req *http.Request) (*http.Request, func(*http.Response) error, error) {
				middlewareCalled = true
				var mwErr error
				if test.middlewareErr {
					mwErr = fmt.Errorf("middleware error")
				}
				return req,
					func(*http.Response) error {
						respHandlerCalled = true
						var rhErr error
						if test.respHandlerErr {
							rhErr = fmt.Errorf("response handler error")
						}
						return rhErr
					}, mwErr
			}
			mrt := MiddlewareRoundTripper{
				RoundTripper: test.roundTripper,
				Middleware: []ClientMiddleware{
					mockMiddleware,
				},
			}
			mockReq := httptest.NewRequest("GET", "/path", nil)
			resp, err := mrt.RoundTrip(mockReq)
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.True(t, middlewareCalled)
				assert.True(t, respHandlerCalled)
			}
		})
	}
}

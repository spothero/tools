package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// This roundtripper is embedded within the MiddlewareRoundTripper as a noop
type MockRoundTripper struct {
	responseStatusCodes []int
	createErr           bool
	callNumber          int
}

func (mockRT *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	currCallNumber := mockRT.callNumber
	mockRT.callNumber++
	if mockRT.createErr {
		return nil, fmt.Errorf("error in roundtripper")
	}
	return &http.Response{StatusCode: mockRT.responseStatusCodes[currCallNumber]}, nil
}

func TestNewDefaultClient(t *testing.T) {
	client := NewDefaultClient(NewMetrics(prometheus.NewRegistry(), true), nil)
	assert.NotNil(t, client)
	mrt, ok := client.Transport.(MiddlewareRoundTripper)
	assert.True(t, ok)
	assert.Equal(t, 4, len(mrt.Middleware))

	rrt, ok := mrt.RoundTripper.(RetryRoundTripper)
	assert.True(t, ok)
	assert.Equal(t, http.DefaultTransport, rrt.RoundTripper)
}

func TestMiddlewareRoundTrip(t *testing.T) {
	tests := []struct {
		name           string
		roundTripper   http.RoundTripper
		middlewareErr  bool
		respHandlerErr bool
		expectErr      bool
		expectPanic    bool
	}{
		{
			"no round tripper results in a panic",
			nil,
			false,
			false,
			false,
			true,
		},
		{
			"round tripper with no error invokes middleware correctly",
			&MockRoundTripper{responseStatusCodes: []int{http.StatusOK}, createErr: false},
			false,
			false,
			false,
			false,
		},
		{
			"round tripper with middleware error returns an error",
			&MockRoundTripper{responseStatusCodes: []int{http.StatusOK}, createErr: false},
			true,
			false,
			true,
			false,
		},
		{
			"round tripper with internal roundtripper error returns an error",
			&MockRoundTripper{responseStatusCodes: []int{http.StatusOK}, createErr: true},
			false,
			false,
			true,
			false,
		},
		{
			"round tripper with resp handler error returns an error",
			&MockRoundTripper{responseStatusCodes: []int{http.StatusOK}, createErr: false},
			false,
			true,
			true,
			false,
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
			if !test.expectPanic {
				resp, err := mrt.RoundTrip(mockReq)
				if test.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, resp)
					assert.True(t, middlewareCalled)
					assert.True(t, respHandlerCalled)
				}
			} else {
				assert.Panics(t, func() {
					_, _ = mrt.RoundTrip(mockReq)
				})
			}
		})
	}
}

func TestRetryRoundTrip(t *testing.T) {
	tests := []struct {
		name               string
		roundTripper       http.RoundTripper
		expectedStatusCode int
		numRetries         uint64
		expectErr          bool
		expectPanic        bool
	}{
		{
			"no round tripper results in a panic",
			nil,
			http.StatusOK, // doesn't matter
			0,
			false,
			true,
		},
		{
			"round tripper with no error invokes middleware correctly",
			&MockRoundTripper{responseStatusCodes: []int{http.StatusOK}, createErr: false},
			http.StatusOK,
			0,
			false,
			false,
		},
		{
			"round tripper with an unresolved error returns an error",
			&MockRoundTripper{
				responseStatusCodes: []int{
					http.StatusInternalServerError,
					http.StatusInternalServerError,
				},
				createErr: false,
			},
			http.StatusInternalServerError,
			1,
			false,
			false,
		},
		{
			"round tripper with an unretriable error returns an error",
			&MockRoundTripper{
				responseStatusCodes: []int{
					http.StatusNotImplemented,
				},
				createErr: false,
			},
			http.StatusNotImplemented,
			1,
			false,
			false,
		},
		{
			"round tripper that encounters an http err is retried",
			&MockRoundTripper{
				responseStatusCodes: []int{
					http.StatusBadRequest,
					http.StatusBadRequest,
				},
				createErr: true,
			},
			http.StatusBadRequest,
			1,
			true,
			false,
		},
		{
			"retries are stopped when a successful or non-retriable status code is given",
			&MockRoundTripper{
				responseStatusCodes: []int{
					http.StatusInternalServerError,
					http.StatusOK,
					http.StatusInternalServerError,
				},
				createErr: true,
			},
			http.StatusOK,
			2,
			true,
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rrt := RetryRoundTripper{
				RetriableStatusCodes: []int{http.StatusInternalServerError},
				MaxRetries:           test.numRetries,
				InitialInterval:      1 * time.Nanosecond,
				RoundTripper:         test.roundTripper,
			}
			mockReq := httptest.NewRequest("GET", "/path", nil)
			if !test.expectPanic {
				resp, err := rrt.RoundTrip(mockReq)
				if test.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, resp)
					assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
				}
			} else {
				assert.Panics(t, func() {
					_, _ = rrt.RoundTrip(mockReq)
				})
			}
		})
	}
}

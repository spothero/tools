package <%=appName%>

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelloWorld(t *testing.T) {
	req, err := http.NewRequest("GET", "/hello-world", nil)
	require.NoError(t, err)
	writer := httptest.NewRecorder()
	tests := []struct {
		name            string
		expectedOutcome string
	}{
		{
			"Hello World writes output",
			"<html>Hello World. What's the <a href='/best-language'>best language?</a></html>",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			helloWorld(writer, req)
			assert.Equal(t, test.expectedOutcome, writer.Body.String())
		})
	}
}

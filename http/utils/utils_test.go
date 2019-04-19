// Copyright 2019 SpotHero
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestWriteHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	sr := StatusRecorder{recorder, http.StatusNotImplemented}
	sr.WriteHeader(http.StatusOK)
	assert.Equal(t, sr.StatusCode, http.StatusOK)
	assert.Equal(t, recorder.Result().StatusCode, http.StatusOK)
}

func TestFetchRoutePathTemplate(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		requestPath     string
		expectedOutcome string
	}{
		{
			"when the path is registered we should receive the path template",
			"/",
			"/",
			"/",
		},
		{
			"when the path is not registered we should receive an empty string",
			"/",
			"/does/not/exist",
			"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pathTemplate := ""
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				pathTemplate = FetchRoutePathTemplate(r)
			}
			router := mux.NewRouter()
			router.HandleFunc(test.path, mockHandler)
			server := httptest.NewServer(router)
			defer server.Close()

			client := server.Client()
			_, err := client.Get(fmt.Sprintf("%s/%s", server.URL, test.requestPath))
			assert.NoError(t, err)
			assert.Equal(t, test.expectedOutcome, pathTemplate)
		})
	}
}

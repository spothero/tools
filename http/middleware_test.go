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

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/spothero/tools/http/utils"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	middlewareCalled := false
	deferableCalled := false
	mw := Middleware{
		func(sr *utils.StatusRecorder, r *http.Request) (func(), *http.Request) {
			middlewareCalled = true
			return func() { deferableCalled = true }, r
		},
	}
	httpRec := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	http.HandlerFunc(mw.handler(mux.NewRouter())).ServeHTTP(httpRec, req)
	assert.True(t, middlewareCalled)
	assert.True(t, deferableCalled)
}

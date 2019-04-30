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

package sql

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func generateStartMiddleware(callCount *int, returnErr bool) MiddlewareStart {
	return func(ctx context.Context, query string, args ...interface{}) (context.Context, MiddlewareEnd, error) {
		*callCount += 1
		var err error
		if returnErr {
			err = fmt.Errorf("fake error")
		}
		return ctx, generateEndMiddleware(callCount, returnErr), err
	}
}

func generateEndMiddleware(callCount *int, returnErr bool) MiddlewareEnd {
	return func(ctx context.Context, query string, queryErr error, args ...interface{}) (context.Context, error) {
		var err error
		if returnErr {
			err = fmt.Errorf("fake error")
		}
		*callCount += 1
		return ctx, err
	}
}

func TestBefore(t *testing.T) {
	tests := []struct {
		name      string
		numMW     int
		expectErr bool
	}{
		{
			"middleware is called successfully and callbacks are added to the context",
			2,
			false,
		},
		{
			"errors in middleware are propagated immediately",
			1,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mw := make(Middleware, 0)
			callCounters := make([]int, test.numMW)
			for idx := 0; idx < test.numMW; idx++ {
				mw = append(mw, generateStartMiddleware(&callCounters[idx], test.expectErr))
			}
			ctx := context.Background()
			ctx, err := mw.Before(ctx, "query", "arg1", "arg2")
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				deferables, ok := ctx.Value(deferableKey).([]MiddlewareEnd)
				assert.True(t, ok)
				assert.Len(t, deferables, 2)
				for _, count := range callCounters {
					assert.Equal(t, 1, count)
				}
			}
		})
	}

}

func TestAfter(t *testing.T) {
	startCtx := context.Background()
	endCtx, err := Middleware{}.After(startCtx, "", "")
	assert.NoError(t, err)
	assert.Equal(t, startCtx, endCtx)
}

func TestOnError(t *testing.T) {
	startCtx := context.Background()
	assert.NoError(t, Middleware{}.OnError(startCtx, nil, "", ""))
}

func TestEnd(t *testing.T) {
	tests := []struct {
		name      string
		numMW     int
		expectErr bool
	}{
		{
			"middlewares are called successfully from context",
			2,
			false,
		},
		{
			"errors in middleware are propagated immediately",
			1,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mw := make([]MiddlewareEnd, 0)
			callCounters := make([]int, test.numMW)
			for idx := 0; idx < test.numMW; idx++ {
				mw = append(mw, generateEndMiddleware(&callCounters[idx], test.expectErr))
			}
			ctx := context.Background()
			ctx = context.WithValue(ctx, deferableKey, mw)
			ctx, err := Middleware{}.end(ctx, nil, "query", "arg1", "arg2")
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				deferables, ok := ctx.Value(deferableKey).([]MiddlewareEnd)
				assert.True(t, ok)
				assert.Len(t, deferables, 2)
				for _, count := range callCounters {
					assert.Equal(t, 1, count)
				}
			}
		})
	}
}

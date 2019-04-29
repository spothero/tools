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
)

// ctxKey is the type used to uniquely place the middleware within context.Context
type ctxKey int

// deferableKey is the value used to uniquely place the middleware end callback in context.Context
const deferableKey ctxKey = iota

// MiddlewareEnd is called after the SQL query has completed
type MiddlewareEnd func(ctx context.Context, query string, queryErr error, args ...interface{}) (context.Context, error)

// MiddlewareStart is called before the SQL query has started
type MiddlewareStart func(ctx context.Context, query string, args ...interface{}) (context.Context, MiddlewareEnd, error)

// Middleware aliases a list of SQL Middleware
type Middleware []MiddlewareStart

// Before satisfies the sqlhooks interface for hooks called before the query is executed
func (m Middleware) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	var err error
	var deferableFunc MiddlewareEnd
	deferables := make([]MiddlewareEnd, len(m))
	for idx, mw := range m {
		ctx, deferableFunc, err = mw(ctx, query, args)
		if err != nil {
			return ctx, err
		}
		deferables[idx] = deferableFunc
	}
	return context.WithValue(ctx, deferableKey, deferables), nil
}

// After satisfies the sqlhooks interface for hooks called after the query has completed
func (m Middleware) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	deferables, ok := ctx.Value(deferableKey).([]MiddlewareEnd)
	if !ok {
		return ctx, nil
	}
	var err error
	for _, mw := range deferables {
		if ctx, err = mw(ctx, query, nil, args); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// OnError satisfies the sqlhooks interface for hooks called when a query errors
func (m Middleware) OnError(ctx context.Context, queryErr error, query string, args ...interface{}) error {
	deferables, ok := ctx.Value(deferableKey).([]MiddlewareEnd)
	if !ok {
		return nil
	}
	var err error
	for _, mw := range deferables {
		if ctx, err = mw(ctx, query, queryErr, args); err != nil {
			return err
		}
	}
	return nil
}

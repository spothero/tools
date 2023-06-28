// Copyright 2023 SpotHero
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

package middleware

import (
	"context"
)

// ctxCallbackKey is the type used to place the list of middleware within context.Context
type ctxCallbackKey int

// ctxCallbackValue is the value used to place the list of End in context.Context
const ctxCallbackValue ctxCallbackKey = iota

// ctxQueryNameKey is the type used to place the query name in context.Context
type ctxQueryNameKey int

// ctxQueryNameValue is the value used to place the query name in context.Context
const ctxQueryNameValue ctxQueryNameKey = iota

// End is called after the SQL query has completed
type End func(ctx context.Context, queryName, query string, queryErr error, args ...interface{}) (context.Context, error)

// Start is called before the SQL query has started
type Start func(ctx context.Context, queryName, query string, args ...interface{}) (context.Context, End, error)

// Middleware aliases a list of SQL Middleware
type Middleware []Start

// NewContext returns a SQL middleware context with the query name embedded in it for downstream
// middleware. All contexts passed into the SQL middleware chain must have first called this
// function
func NewContext(ctx context.Context, queryName string) context.Context {
	return context.WithValue(ctx, ctxQueryNameValue, queryName)
}

// Before satisfies the sqlhooks interface for hooks called before the query is executed
func (m Middleware) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	queryName, _ := ctx.Value(ctxQueryNameValue).(string)
	var err error
	var mwEnd End
	mwEndCallbacks := make([]End, len(m))
	for idx, mw := range m {
		ctx, mwEnd, err = mw(ctx, queryName, query, args)
		if err != nil {
			return ctx, err
		}
		mwEndCallbacks[idx] = mwEnd
	}
	return context.WithValue(ctx, ctxCallbackValue, mwEndCallbacks), nil
}

// After satisfies the sqlhooks interface for hooks called after the query has completed
func (m Middleware) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	return m.end(ctx, nil, query, args)
}

// OnError satisfies the sqlhooks interface for hooks called when a query errors
func (m Middleware) OnError(ctx context.Context, queryErr error, query string, args ...interface{}) error {
	_, err := m.end(ctx, queryErr, query, args)
	return err
}

// end provides a common function for closing out SQL query middleware
func (m Middleware) end(ctx context.Context, queryErr error, query string, args ...interface{}) (context.Context, error) {
	queryName, _ := ctx.Value(ctxQueryNameValue).(string)
	mwEndCallbacks, ok := ctx.Value(ctxCallbackValue).([]End)
	if !ok {
		return ctx, nil
	}
	var err error
	for _, mw := range mwEndCallbacks {
		if ctx, err = mw(ctx, queryName, query, queryErr, args); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

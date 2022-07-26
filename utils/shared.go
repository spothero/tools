package utils

// ContextKey this type is created to avoid the linting error:
// SA1029: should not use built-in type string as key for value; define your own type to avoid collisions (staticcheck)
type ContextKey string

// AuthenticatedClientKey is the key used in the request context to pass the client to metrics
const AuthenticatedClientKey = ContextKey("authenticated_client")

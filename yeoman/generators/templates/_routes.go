package <%=appName%>

import (
	"fmt"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
)

// RegisterMuxes is a callback used to register HTTP endpoints to the default server
// NOTE: The HTTP server automatically registers /health and /metrics -- Have a look in your
// browser!
func RegisterMuxes(mux *http.ServeMux) {
	mux.HandleFunc("/", helloWorld)
	mux.HandleFunc("/best-language", bestLanguage)
}

// helloWorld simply writes "hello world" to the caller. It is ended for use as an HTTP callback.
func helloWorld(w http.ResponseWriter, r *http.Request) {
	// NOTE: This is an example of an opentracing span
	span, _ := opentracing.StartSpanFromContext(r.Context(), "example-hello-world")
	span = span.SetTag("Key", "Value")
	defer span.Finish()

	// NOTE: Here we write out some artisanal HTML. There are many other (better) ways to output data.
	fmt.Fprintf(w, "<html>Hello World. What's the <a href='/best-language'>best language?</a></html>")
}

// bestLanguage tells the caller what the best language is. It is inteded for use as an HTTP callback.
func bestLanguage(w http.ResponseWriter, r *http.Request) {
	// NOTE: This is an example of an opentracing span
	span, _ := opentracing.StartSpanFromContext(r.Context(), "example-hello-world")
	span = span.SetTag("best.language", "golang")
	span = span.SetTag("best.mascot", "gopher")
	defer span.Finish()

	// NOTE: Here we write out some artisanal HTML. There are many other (better) ways to output data.
	fmt.Fprintf(w, "<html><a href='//golang.org/'>Golang</a>, of course! \\ʕ◔ϖ◔ʔ/</br> Say <a href='/'>hello</a> again.</html>")
}

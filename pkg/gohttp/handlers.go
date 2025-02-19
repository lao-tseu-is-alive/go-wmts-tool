package gohttp

import (
	"fmt"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"net/http"
	"time"
)

func GetReadinessHandler(l golog.MyLogger) http.HandlerFunc {
	handlerName := "GetReadinessHandler"
	l.Debug(initCallMsg, handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		TraceRequest(handlerName, r, l)
		w.WriteHeader(http.StatusOK)
	}
}

func GetHealthHandler(l golog.MyLogger) http.HandlerFunc {
	handlerName := "GetHealthHandler"
	l.Debug(initCallMsg, handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		TraceRequest(handlerName, r, l)
		w.WriteHeader(http.StatusOK)
	}
}

func GetStaticPageHandler(title string, description string, l golog.MyLogger) http.HandlerFunc {
	handlerName := fmt.Sprintf("GetStaticPageHandler[%s]", title)
	l.Debug(initCallMsg, handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		TraceRequest(handlerName, r, l)
		w.Header().Set(HeaderContentType, MIMEHtml)
		w.WriteHeader(http.StatusOK)
		n, err := fmt.Fprintf(w, getHtmlPage(title, description))
		if err != nil {
			l.Error("💥💥 ERROR: [%s]  was unable to Fprintf. path:'%s', from IP: [%s], send_bytes:%d\n", handlerName, r.URL.Path, r.RemoteAddr, n)
			http.Error(w, "Internal server error. GetStaticPageHandler was unable to Fprintf", http.StatusInternalServerError)
		}
	}
}

func GetTimeHandler(l golog.MyLogger) http.HandlerFunc {
	handlerName := "GetTimeHandler"
	l.Debug(initCallMsg, handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		TraceRequest(handlerName, r, l)
		now := time.Now()
		w.Header().Set(HeaderContentType, MIMEAppJSONCharsetUTF8)
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, "{\"time\":\"%s\"}", now.Format(time.RFC3339))
		if err != nil {
			l.Error("Error doing fmt.Fprintf err: %s", err)
		}
	}
}

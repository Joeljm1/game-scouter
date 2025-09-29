// Package customrespwriter provide a custom response writer to wrap
// an http.ResponseWriter to get details like status and duration for
// logging. If operations like Flush or Hijack are needed,
// then wrap the response writer in http.NewResponseController
// to access those methods
package customrespwriter

import (
	"net/http"
	"time"
)

//TODO: instead of just write do check session chamges and write to db if required and then send resp

type MetricsResponseWriter struct {
	http.ResponseWriter
	Status        int
	StartTime     time.Time
	HeaderWritten bool
}

func New(w http.ResponseWriter) *MetricsResponseWriter {
	return &MetricsResponseWriter{
		ResponseWriter: w,
		Status:         http.StatusOK,
		StartTime:      time.Now().UTC(),
		HeaderWritten:  false,
	}
}

func (mw *MetricsResponseWriter) WriteHeader(status int) {
	mw.ResponseWriter.WriteHeader(status)
	if !mw.HeaderWritten {
		mw.Status = status
		mw.HeaderWritten = true
	}
}

func (mw *MetricsResponseWriter) Write(b []byte) (int, error) {
	mw.HeaderWritten = true
	return mw.ResponseWriter.Write(b)
}

func (mw *MetricsResponseWriter) UnWrap() http.ResponseWriter {
	return mw.ResponseWriter
}

package http

import (
	"io"
	"net/http"
)

// Discard is an helper function that discards the response body and closes the underlying connection
func Discard(req *Request, resp *http.Response, respReadLimit int64) {
	_, err := io.Copy(io.Discard, io.LimitReader(resp.Body, respReadLimit))
	if err != nil {
		req.Metrics.DrainErrors++
	}
	resp.Body.Close()
}

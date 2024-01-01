package http

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hetiansu5/urlquery"
	"github.com/oarkflow/errors"
)

// Client is used to make HTTP requests. It adds additional functionality
// like automatic retries to tolerate minor outages.
type Client struct {
	// HTTPClient is the pkg HTTP client.
	HTTPClient *http.Client

	// RequestLogHook allows a user-supplied function to be called
	// before each retry.
	RequestLogHook RequestLogHook
	// ResponseLogHook allows a user-supplied function to be called
	// with the response from each HTTP request executed.
	ResponseLogHook ResponseLogHook
	// ErrorHandler specifies the custom error handler to use, if any
	ErrorHandler ErrorHandler

	// CheckRetry specifies the policy for handling retries, and is called
	// after each request. The default policy is DefaultRetryPolicy.
	CheckRetry CheckRetry
	// Backoff specifies the policy for how long to wait between retries
	Backoff Backoff

	options *Options
}

// Options contains configuration options for the client
type Options struct {
	// RetryWaitMin is the minimum time to wait for retry
	RetryWaitMin time.Duration
	// RetryWaitMax is the maximum time to wait for retry
	RetryWaitMax time.Duration
	// Timeout is the maximum time to wait for the request
	Timeout time.Duration
	// RetryMax is the maximum number of retries
	RetryMax int
	// RespReadLimit is the maximum HTTP response size to read for
	// connection being reused.
	RespReadLimit int64
	// Verbose specifies if debug messages should be printed
	Verbose bool
	// KillIdleConn specifies if all keep-alive connections gets killed
	KillIdleConn     bool
	MaxPoolSize      int
	ReqPerSec        int
	Semaphore        chan int
	RateLimiter      <-chan time.Time
	Auth             Auth
	Headers          map[string]string
	AuthType         string                 `json:"auth_type"`
	URL              string                 `json:"url"`
	Method           string                 `json:"method"`
	Data             map[string]interface{} `json:"data"`
	DataField        string                 `json:"data_field"`
	ResponseCallback func(response []byte, dataField ...string) (interface{}, error)
	MU               *sync.RWMutex
}

// DefaultOptionsSpraying contains the default options for host spraying
// scenarios where lots of requests need to be sent to different hosts.
var DefaultOptionsSpraying = Options{
	RetryWaitMin:  1 * time.Second,
	RetryWaitMax:  30 * time.Second,
	Timeout:       30 * time.Second,
	RetryMax:      5,
	RespReadLimit: 4096,
	KillIdleConn:  true,
	MaxPoolSize:   100,
	ReqPerSec:     10,
}

// DefaultOptionsSingle contains the default options for host bruteforce
// scenarios where lots of requests need to be sent to a single host.
var DefaultOptionsSingle = Options{
	RetryWaitMin:  1 * time.Second,
	RetryWaitMax:  30 * time.Second,
	Timeout:       30 * time.Second,
	RetryMax:      5,
	RespReadLimit: 4096,
	KillIdleConn:  false,
	MaxPoolSize:   100,
	ReqPerSec:     10,
}

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewClient creates a new Client with default settings.
func NewClient(opt ...*Options) *Client {
	var options *Options
	if len(opt) > 0 {
		options = opt[0]
	}

	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}
	httpclient := DefaultClient()
	// if necessary adjusts per-request timeout proportionally to general timeout (30%)
	if options.Timeout > time.Second*15 {
		httpclient.Timeout = time.Duration(options.Timeout.Seconds()*0.3) * time.Second
	}
	var semaphore chan int = nil
	if options.MaxPoolSize > 0 {
		semaphore = make(chan int, options.MaxPoolSize) // Buffered channel to act as a semaphore
	}

	var emitter <-chan time.Time = nil
	if options.ReqPerSec > 0 {
		emitter = time.NewTicker(time.Second / time.Duration(options.ReqPerSec)).C // x req/s == 1s/x req (inverse)
	}
	options.Semaphore = semaphore
	options.RateLimiter = emitter

	c := &Client{
		HTTPClient: httpclient,
		CheckRetry: DefaultRetryPolicy(),
		Backoff:    ExponentialJitterBackoff(),
		options:    options,
	}

	c.setKillIdleConnections()
	return c
}

// NewWithHTTPClient creates a new Client with default settings and provided http.Client
func NewWithHTTPClient(client *http.Client, options Options) *Client {
	c := &Client{
		HTTPClient: client,
		CheckRetry: DefaultRetryPolicy(),
		Backoff:    DefaultBackoff(),
		options:    &options,
	}

	c.setKillIdleConnections()
	return c
}

// setKillIdleConnections sets the kill idle conns switch in two scenarios
//  1. If the http.Client has settings that require us to do so.
//  2. The user has enabled it by default, in which case we have nothing to do.
func (c *Client) setKillIdleConnections() {
	if c.HTTPClient != nil || !c.options.KillIdleConn {
		if b, ok := c.HTTPClient.Transport.(*http.Transport); ok {
			c.options.KillIdleConn = b.DisableKeepAlives || b.MaxConnsPerHost < 0
		}
	}
}

func (c *Client) AddHeader(key, value string) {
	c.options.Headers[key] = value
}

// Get is a convenience helper for doing simple GET requests.
func (c *Client) Get(url string, body interface{}, headers ...map[string]string) (*http.Response, error) {
	var err error
	if body != nil {
		bts, err := urlquery.Marshal(body)
		if err != nil {
			return nil, err
		}
		url = url + "?" + string(bts)
	}
	_, headers, err = prepareHeaderContentType(body, headers, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, err
	}
	return c.Request(http.MethodGet, url, strings.NewReader(""), headers...)
}

// Post is a convenience method for doing simple POST requests.
func (c *Client) Post(url string, body interface{}, headers ...map[string]string) (*http.Response, error) {
	var err error
	body, headers, err = prepareHeaderContentType(body, headers, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, err
	}
	return c.Request(http.MethodPost, url, body, headers...)
}

// Put is a convenience method for doing simple POST requests.
func (c *Client) Put(url string, body interface{}, headers ...map[string]string) (*http.Response, error) {
	var err error
	body, headers, err = prepareHeaderContentType(body, headers, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, err
	}
	return c.Request(http.MethodPut, url, body, headers...)
}

// PutForm is a convenience method for doing simple POST requests.
func (c *Client) PutForm(url string, body interface{}, headers ...map[string]string) (*http.Response, error) {
	var err error
	body, headers, err = prepareHeaderContentType(body, headers, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return nil, err
	}
	return c.Request(http.MethodPut, url, body, headers...)
}

// Form is a convenience method for doing simple POST operations using
// pre-filled url.Values form data.
func (c *Client) Form(url string, body interface{}, headers ...map[string]string) (*http.Response, error) {
	var err error
	body, headers, err = prepareHeaderContentType(body, headers, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return nil, err
	}
	return c.Request(http.MethodPost, url, body, headers...)
}

// Delete is a convenience helper for doing simple GET requests.
func (c *Client) Delete(url string, headers ...map[string]string) (*http.Response, error) {
	return c.Request(http.MethodDelete, url, nil, headers...)
}

// Head is a convenience method for doing simple HEAD requests.
func (c *Client) Head(url string, headers ...map[string]string) (*http.Response, error) {
	return c.Request(http.MethodHead, url, nil, headers...)
}

func (c *Client) Request(method string, url string, body interface{}, headers ...map[string]string) (*http.Response, error) {
	req, err := NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	for key, val := range c.options.Headers {
		req.Header.Set(key, val)
	}
	for _, heads := range headers {
		for key, val := range heads {
			req.Header.Set(key, val)
		}
	}
	return c.Do(req)
}

// Do wraps calling an HTTP method with retries.
func (c *Client) Do(req *Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// Create a main context that will be used as the main timeout
	mainCtx, cancel := context.WithTimeout(context.Background(), c.options.Timeout)
	defer cancel()

	for i := 0; ; i++ {
		// Always rewind the request body when non-nil.
		if req.body != nil {
			body, err := req.body()
			if err != nil {
				c.closeIdleConnections()
				return resp, err
			}
			if c, ok := body.(io.ReadCloser); ok {
				req.Body = c
			} else {
				req.Body = io.NopCloser(body)
			}
		}

		if c.RequestLogHook != nil {
			c.RequestLogHook(req.Request, i)
		}
		if c.options.MaxPoolSize > 0 {
			c.options.Semaphore <- 1 // Grab a connection from our pool
			defer func() {
				<-c.options.Semaphore // Defer release our connection back to the pool
			}()
		}

		if c.options.ReqPerSec > 0 {
			<-c.options.RateLimiter // Block until a signal is emitted from the rateLimiter
		}
		// Attempt the request
		resp, err = c.HTTPClient.Do(req.Request)

		// Check if we should continue with retries.
		checkOK, checkErr := c.CheckRetry(req.Context(), resp, err)

		if err != nil {
			// Increment the failure counter as the request failed
			req.Metrics.Failures++
		} else if c.ResponseLogHook != nil {
			// Call the response logger function if provided.
			c.ResponseLogHook(resp)
		}

		// Now decide if we should continue.
		if !checkOK {
			if checkErr != nil {
				err = checkErr
			}
			c.closeIdleConnections()
			return resp, err
		}

		// We do this before drainBody beause there's no need for the I/O if
		// we're breaking out
		remain := c.options.RetryMax - i
		if remain <= 0 {
			break
		}

		// Increment the retries counter as we are going to do one more retry
		req.Metrics.Retries++

		// We're going to retry, consume any response to reuse the connection.
		if err == nil && resp != nil {
			c.drainBody(req, resp)
		}

		// Wait for the time specified by backoff then retry.
		// If the context is cancelled however, return.
		wait := c.Backoff(c.options.RetryWaitMin, c.options.RetryWaitMax, i, resp)
		log.Println(fmt.Sprintf("Retrying for URL %s after %d for error %s", req.Host, wait, err))
		// Exit if the main context or the request context is done
		// Otherwise, wait for the duration and try again.
		select {
		case <-mainCtx.Done():
			break
		case <-req.Context().Done():
			c.closeIdleConnections()
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}

	if c.ErrorHandler != nil {
		c.closeIdleConnections()
		return c.ErrorHandler(resp, err, c.options.RetryMax+1)
	}

	// By default, we close the response body and return an error without
	// returning the response
	if resp != nil {
		resp.Body.Close()
	}
	c.closeIdleConnections()
	return nil, fmt.Errorf("%s %s giving up after %d attempts: %w", req.Method, req.URL, c.options.RetryMax+1, err)
}

// Try to read the response body, so we can reuse this connection.
func (c *Client) drainBody(req *Request, resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, io.LimitReader(resp.Body, c.options.RespReadLimit))
	if err != nil {
		req.Metrics.DrainErrors++
	}
	resp.Body.Close()
}

func (c *Client) closeIdleConnections() {
	if c.options.KillIdleConn {
		c.HTTPClient.CloseIdleConnections()
	}
}

func prepareHeaderContentType(body interface{}, headers []map[string]string, head map[string]string) ([]byte, []map[string]string, error) {
	for _, header := range headers {
		for key, val := range header {
			if strings.ToLower(key) == "content-type" {
				switch val {
				case "text/xml":
					bt, err := xml.Marshal(body)
					return bt, headers, errors.NewE(err, "Unable to marshal xml", "")
				default:
					bt, err := json.Marshal(body)
					return bt, headers, errors.NewE(err, "Unable to marshal json", "")
				}
			}
		}
	}
	headers = append(headers, head)
	bt, err := json.Marshal(body)
	return bt, headers, err
}

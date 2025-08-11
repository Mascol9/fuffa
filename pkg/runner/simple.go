package runner

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Mascol9/fuffa/pkg/ffuf"

	"github.com/andybalholm/brotli"
)

// Download results < 5MB
const MAX_DOWNLOAD_SIZE = 5242880

type SimpleRunner struct {
	config        *ffuf.Config
	client        *http.Client
	firstRequest  bool
}

func NewSimpleRunner(conf *ffuf.Config, replay bool) ffuf.RunnerProvider {
	var simplerunner SimpleRunner
	proxyURL := http.ProxyFromEnvironment
	customProxy := ""

	if replay {
		customProxy = conf.ReplayProxyURL
	} else {
		customProxy = conf.ProxyURL
	}
	if len(customProxy) > 0 {
		pu, err := url.Parse(customProxy)
		if err == nil {
			proxyURL = http.ProxyURL(pu)
		}
	}
	cert := []tls.Certificate{}

	if conf.ClientCert != "" && conf.ClientKey != "" {
		tmp, _ := tls.LoadX509KeyPair(conf.ClientCert, conf.ClientKey)
		cert = []tls.Certificate{tmp}
	}

	simplerunner.config = conf
	simplerunner.firstRequest = true
	simplerunner.client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
		Timeout:       time.Duration(time.Duration(conf.Timeout) * time.Second),
		Transport: &http.Transport{
			ForceAttemptHTTP2:   conf.Http2,
			Proxy:               proxyURL,
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 500,
			MaxConnsPerHost:     500,
			DialContext: (&net.Dialer{
				Timeout: time.Duration(time.Duration(conf.Timeout) * time.Second),
			}).DialContext,
			TLSHandshakeTimeout: time.Duration(time.Duration(conf.Timeout) * time.Second),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS10,
				Renegotiation:      tls.RenegotiateOnceAsClient,
				ServerName:         conf.SNI,
				Certificates:       cert,
			},
		}}

	if conf.FollowRedirects {
		simplerunner.client.CheckRedirect = nil
	}
	return &simplerunner
}

func (r *SimpleRunner) Prepare(input map[string][]byte, basereq *ffuf.Request) (ffuf.Request, error) {
	req := ffuf.CopyRequest(basereq)

	for keyword, inputitem := range input {
		req.Method = strings.ReplaceAll(req.Method, keyword, string(inputitem))
		headers := make(map[string]string, len(req.Headers))
		for h, v := range req.Headers {
			var CanonicalHeader string = textproto.CanonicalMIMEHeaderKey(strings.ReplaceAll(h, keyword, string(inputitem)))
			headers[CanonicalHeader] = strings.ReplaceAll(v, keyword, string(inputitem))
		}
		req.Headers = headers
		req.Url = r.replaceKeywordInURL(req.Url, keyword, string(inputitem))
		req.Data = []byte(strings.ReplaceAll(string(req.Data), keyword, string(inputitem)))
	}

	req.Input = input
	return req, nil
}

// replaceKeywordInURL replaces keyword in URL while avoiding double slashes
func (r *SimpleRunner) replaceKeywordInURL(url, keyword, replacement string) string {
	result := strings.ReplaceAll(url, keyword, replacement)
	// Fix double slashes but preserve protocol slashes (://)
	return regexp.MustCompile(`([^:])/+`).ReplaceAllString(result, "$1/")
}

func (r *SimpleRunner) Execute(req *ffuf.Request) (ffuf.Response, error) {
	var httpreq *http.Request
	var err error
	var rawreq []byte
	data := bytes.NewReader(req.Data)

	var start time.Time
	var firstByteTime time.Duration

	trace := &httptrace.ClientTrace{
		WroteRequest: func(wri httptrace.WroteRequestInfo) {
			start = time.Now() // begin the timer after the request is fully written
		},
		GotFirstResponseByte: func() {
			firstByteTime = time.Since(start) // record when the first byte of the response was received
		},
	}

	httpreq, err = http.NewRequestWithContext(r.config.Context, req.Method, req.Url, data)

	if err != nil {
		return ffuf.Response{}, err
	}

	// set default User-Agent header if not present
	if _, ok := req.Headers["User-Agent"]; !ok {
		req.Headers["User-Agent"] = fmt.Sprintf("%s v%s", "FUFFA - FFUF Using Fantastic Formats And colors", ffuf.Version())
	}

	// Handle Go http.Request special cases
	if _, ok := req.Headers["Host"]; ok {
		httpreq.Host = req.Headers["Host"]
	}

	req.Host = httpreq.Host
	httpreq = httpreq.WithContext(httptrace.WithClientTrace(r.config.Context, trace))

	if r.config.Raw {
		httpreq.URL.Opaque = req.Url
	}

	for k, v := range req.Headers {
		httpreq.Header.Set(k, v)
	}

	if len(r.config.OutputDirectory) > 0 || len(r.config.AuditLog) > 0 {
		rawreq, _ = httputil.DumpRequestOut(httpreq, true)
		req.Raw = string(rawreq)
	}

	httpresp, err := r.client.Do(httpreq)
	if err != nil {
		return ffuf.Response{}, err
	}

	req.Timestamp = start

	resp := ffuf.NewResponse(httpresp, req)
	defer httpresp.Body.Close()

	// Debug first request/response if enabled, or if forced
	if (r.config.DebugFirstRequest && r.firstRequest) || r.config.ForceDebugNext {
		r.printDebugRequest(httpreq, httpresp)
		r.firstRequest = false
		// Reset the force debug flag after using it
		r.config.ForceDebugNext = false
	}

	// Check if we should download the resource or not
	size, err := strconv.Atoi(httpresp.Header.Get("Content-Length"))
	if err == nil {
		resp.ContentLength = int64(size)
		if (r.config.IgnoreBody) || (size > MAX_DOWNLOAD_SIZE) {
			resp.Cancelled = true
			return resp, nil
		}
	}

	if len(r.config.OutputDirectory) > 0 || len(r.config.AuditLog) > 0 {
		rawresp, _ := httputil.DumpResponse(httpresp, true)
		resp.Request.Raw = string(rawreq)
		resp.Raw = string(rawresp)
	}
	var bodyReader io.ReadCloser
	if httpresp.Header.Get("Content-Encoding") == "gzip" {
		bodyReader, err = gzip.NewReader(httpresp.Body)
		if err != nil {
			// fallback to raw data
			bodyReader = httpresp.Body
		}
	} else if httpresp.Header.Get("Content-Encoding") == "br" {
		bodyReader = io.NopCloser(brotli.NewReader(httpresp.Body))
		if err != nil {
			// fallback to raw data
			bodyReader = httpresp.Body
		}
	} else if httpresp.Header.Get("Content-Encoding") == "deflate" {
		bodyReader = flate.NewReader(httpresp.Body)
		if err != nil {
			// fallback to raw data
			bodyReader = httpresp.Body
		}
	} else {
		bodyReader = httpresp.Body
	}

	if respbody, err := io.ReadAll(bodyReader); err == nil {
		resp.ContentLength = int64(len(string(respbody)))
		resp.Data = respbody
	}

	wordsSize := len(strings.Split(string(resp.Data), " "))
	linesSize := len(strings.Split(string(resp.Data), "\n"))
	resp.ContentWords = int64(wordsSize)
	resp.ContentLines = int64(linesSize)
	resp.Duration = firstByteTime
	resp.Timestamp = start.Add(firstByteTime)

	return resp, nil
}

func (r *SimpleRunner) Dump(req *ffuf.Request) ([]byte, error) {
	var httpreq *http.Request
	var err error
	data := bytes.NewReader(req.Data)
	httpreq, err = http.NewRequestWithContext(r.config.Context, req.Method, req.Url, data)
	if err != nil {
		return []byte{}, err
	}

	// set default User-Agent header if not present
	if _, ok := req.Headers["User-Agent"]; !ok {
		req.Headers["User-Agent"] = fmt.Sprintf("%s v%s", "FUFFA - FFUF Using Fantastic Formats And colors", ffuf.Version())
	}

	// Handle Go http.Request special cases
	if _, ok := req.Headers["Host"]; ok {
		httpreq.Host = req.Headers["Host"]
	}

	req.Host = httpreq.Host
	for k, v := range req.Headers {
		httpreq.Header.Set(k, v)
	}
	return httputil.DumpRequestOut(httpreq, true)
}


// printDebugRequest prints the first HTTP request and response for debugging
func (r *SimpleRunner) printDebugRequest(httpreq *http.Request, httpresp *http.Response) {
	// ANSI color constants for debug output
	const (
		ANSI_CLEAR  = "\x1b[0m"
		ANSI_CYAN   = "\x1b[36m"
		ANSI_YELLOW = "\x1b[33m"
		ANSI_GREEN  = "\x1b[32m"
		ANSI_BLUE   = "\x1b[34m"
		ANSI_BOLD   = "\x1b[1m"
	)
	
	// Maximum response body length to display
	const MAX_RESPONSE_BODY = 2000
	
	fmt.Printf("\n%s%s%s\n", ANSI_CYAN, strings.Repeat("═", 60), ANSI_CLEAR)
	fmt.Printf("%s%s🐛 DEBUG: FIRST HTTP REQUEST AND RESPONSE%s\n", ANSI_BOLD, ANSI_CYAN, ANSI_CLEAR)
	fmt.Printf("%s%s%s\n\n", ANSI_CYAN, strings.Repeat("═", 60), ANSI_CLEAR)
	
	// Print request
	fmt.Printf("%s%s📤 REQUEST:%s\n", ANSI_BOLD, ANSI_GREEN, ANSI_CLEAR)
	fmt.Printf("%s%s%s\n", ANSI_GREEN, strings.Repeat("─", 30), ANSI_CLEAR)
	reqDump, err := httputil.DumpRequestOut(httpreq, true)
	if err != nil {
		fmt.Printf("%sError dumping request: %v%s\n", ANSI_YELLOW, err, ANSI_CLEAR)
	} else {
		fmt.Printf("%s\n", string(reqDump))
	}
	
	// Print response
	fmt.Printf("%s%s📥 RESPONSE:%s\n", ANSI_BOLD, ANSI_BLUE, ANSI_CLEAR)
	fmt.Printf("%s%s%s\n", ANSI_BLUE, strings.Repeat("─", 30), ANSI_CLEAR)
	respDump, err := httputil.DumpResponse(httpresp, true)
	if err != nil {
		fmt.Printf("%sError dumping response: %v%s\n", ANSI_YELLOW, err, ANSI_CLEAR)
	} else {
		respStr := string(respDump)
		
		// Limit response body length
		if len(respStr) > MAX_RESPONSE_BODY {
			// Find the end of headers (double newline)
			headerEnd := strings.Index(respStr, "\r\n\r\n")
			if headerEnd == -1 {
				headerEnd = strings.Index(respStr, "\n\n")
			}
			
			if headerEnd != -1 && headerEnd < MAX_RESPONSE_BODY {
				// Show headers + truncated body
				truncatedBody := respStr[headerEnd:headerEnd+4] + respStr[headerEnd+4:min(headerEnd+4+MAX_RESPONSE_BODY-headerEnd-4, len(respStr))]
				if len(respStr) > headerEnd+4+MAX_RESPONSE_BODY-headerEnd-4 {
					truncatedBody += fmt.Sprintf("\n\n%s... [TRUNCATED - %d more chars] ...%s", ANSI_YELLOW, len(respStr)-len(truncatedBody), ANSI_CLEAR)
				}
				respStr = respStr[:headerEnd] + truncatedBody
			} else {
				// Just truncate everything
				respStr = respStr[:MAX_RESPONSE_BODY] + fmt.Sprintf("\n\n%s... [TRUNCATED - %d more chars] ...%s", ANSI_YELLOW, len(string(respDump))-MAX_RESPONSE_BODY, ANSI_CLEAR)
			}
		}
		
		fmt.Printf("%s\n", respStr)
	}
	
	fmt.Printf("%s%s%s\n", ANSI_CYAN, strings.Repeat("═", 60), ANSI_CLEAR)
	fmt.Printf("%s%s✅ END OF DEBUG OUTPUT%s\n", ANSI_BOLD, ANSI_CYAN, ANSI_CLEAR)
	fmt.Printf("%s%s%s\n\n", ANSI_CYAN, strings.Repeat("═", 60), ANSI_CLEAR)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

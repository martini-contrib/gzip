package gzip

import (
	"bufio"
	"compress/gzip"
	"github.com/go-martini/martini"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_GzipAll(t *testing.T) {
	// Set up
	recorder := httptest.NewRecorder()
	before := false

	bM := martini.New()
	bM.Use(All())
	bM.Use(func(r http.ResponseWriter) {
		r.(martini.ResponseWriter).Before(func(rw martini.ResponseWriter) {
			before = true
		})
	})
	router := martini.NewRouter()
	bM.MapTo(router, (*martini.Routes)(nil))
	bM.Action(router.Handle)
	m := &martini.ClassicMartini{
		Martini: bM,
		Router:  router,
	}
	//Create tests for
	m.Get("/", (func(w http.ResponseWriter) {
		w.Write([]byte("data!"))
	}))
	m.Get("/blank", (func(w http.ResponseWriter) (int, []byte) {
		return http.StatusOK, nil
	}))

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Error(err)
	}

	m.ServeHTTP(recorder, r)

	t.Logf("recorder: %#v", recorder)

	// Make our assertions
	_, ok := recorder.HeaderMap[HeaderContentEncoding]
	if ok {
		t.Error(HeaderContentEncoding + " present")
	}

	ce := recorder.Header().Get(HeaderContentEncoding)
	if strings.EqualFold(ce, "gzip") {
		t.Error(HeaderContentEncoding + " is 'gzip'")
	}

	recorder = httptest.NewRecorder()
	r.Header.Set(HeaderAcceptEncoding, "gzip")
	m.ServeHTTP(recorder, r)

	// Make our assertions
	_, ok = recorder.HeaderMap[HeaderContentEncoding]
	if !ok {
		t.Error(HeaderContentEncoding + " not present")
	}

	ce = recorder.Header().Get(HeaderContentEncoding)
	if !strings.EqualFold(ce, "gzip") {
		t.Error(HeaderContentEncoding + " is not 'gzip'")
	}

	bodyReader, err := gzip.NewReader(recorder.Body)
	if err != nil {
		t.Errorf("Unable to read body as gzip: %+v", err)
	}
	d, err := ioutil.ReadAll(bodyReader)
	if err != nil && err != io.EOF {
		t.Errorf("Unable to read gzip body: %+v", err)
	}
	if string(d) != "data!" {
		t.Errorf("Body not decoded correctly: %s, %s", recorder.Body.Bytes(), d)
	}

	if before == false {
		t.Error("Before hook was not called")
	}

	//Setup case where we request gzip but no response body is given
	recorder = httptest.NewRecorder()
	r, err = http.NewRequest("GET", "/blank", nil)
	if err != nil {
		t.Errorf("Error creating request", err)
		t.FailNow()
	}
	r.Header.Set(HeaderAcceptEncoding, "gzip")
	m.ServeHTTP(recorder, r)

	// Make our assertions
	_, ok = recorder.HeaderMap[HeaderContentEncoding]
	if ok {
		t.Error(HeaderContentEncoding + " is present")
	}

	ce = recorder.Header().Get(HeaderContentEncoding)
	if strings.EqualFold(ce, "gzip") {
		t.Error(HeaderContentEncoding + " is 'gzip'")
	}

	data := recorder.Body.Bytes()
	if len(data) > 0 {
		t.Error("Body has content " + string(data))
	}

	if before == false {
		t.Error("Before hook was not called")
	}
}

type hijackableResponse struct {
	Hijacked bool
	header   http.Header
}

func newHijackableResponse() *hijackableResponse {
	return &hijackableResponse{header: make(http.Header)}
}

func (h *hijackableResponse) Header() http.Header           { return h.header }
func (h *hijackableResponse) Write(buf []byte) (int, error) { return 0, nil }
func (h *hijackableResponse) WriteHeader(code int)          {}
func (h *hijackableResponse) Flush()                        {}
func (h *hijackableResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.Hijacked = true
	return nil, nil, nil
}

func Test_ResponseWriter_Hijack(t *testing.T) {
	hijackable := newHijackableResponse()

	m := martini.New()
	m.Use(All())
	m.Use(func(rw http.ResponseWriter) {
		if hj, ok := rw.(http.Hijacker); !ok {
			t.Error("Unable to hijack")
		} else {
			hj.Hijack()
		}
	})

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Error(err)
	}

	r.Header.Set(HeaderAcceptEncoding, "gzip")
	m.ServeHTTP(hijackable, r)

	if !hijackable.Hijacked {
		t.Error("Hijack was not called")
	}
}

package short

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

const (
	testbase = "http://te.st/"
	testdb   = "testdata/test.db"
)

var testHandler Handler

func newTestHandler() Handler {
	db, err := initDB(testdb, shortenerBucket)
	if err != nil {
		log.Fatalln("init initDB:", err)
	}
	h := Handler{
		shortBase: testbase,
		db:        db,
	}
	return h
}

type test struct {
	long           string
	short          string
	wanterr        bool
	shortenResCode int
	expandResCode  int
}

var cases = map[string]test{
	// Shorten and Expand
	"Good 1": {
		long:           "http://testlong.com/x",
		short:          testbase + "1",
		wanterr:        false,
		shortenResCode: http.StatusOK,
		expandResCode:  http.StatusSeeOther,
	},
	"Good 2": {
		long:           "https://www.google.com/maps/place/Lush+Studio+Soho/@51.5127403,-0.1367994,20z/data=!4m5!3m4!1s0x487604d4f30976bb:0xaac8a5e5c74c71fe!8m2!3d51.5127963!4d-0.1367095",
		short:          testbase + "2",
		wanterr:        false,
		shortenResCode: http.StatusOK,
		expandResCode:  http.StatusSeeOther,
	},

	// Shorten (ServeHTTP only). Shorten itself doesn't check the long URL
	"Invalid long URL": {
		long:           "$this[shouldn't%%%%work.com/wow",
		short:          "",
		wanterr:        true,
		shortenResCode: http.StatusBadRequest,
	},

	// Expand Only
	"Short URL too short": {
		long:          "",
		short:         "http://d",
		wanterr:       true,
		expandResCode: http.StatusBadRequest,
	},
	"Short URL wrong base": {
		long:          "",
		short:         "http://wo.mp/2",
		wanterr:       true,
		expandResCode: http.StatusBadRequest,
	},
	"Valid but not found": {
		long:          "",
		short:         testbase + "1h1gtj",
		wanterr:       false,
		expandResCode: http.StatusNotFound,
	},
}

// Ordered test assignments
var shortenTests = []string{
	"Good 1", "Good 2",
}
var expandTests = []string{
	"Good 1", "Good 2", "Valid but not found",
	"Short URL too short", "Short URL wrong base",
}

func TestShortenerHandler(t *testing.T) {
}

func TestHandler_Shorten(t *testing.T) {
	// Needs a fresh db
	err := os.Remove("testdata/test.db")
	if err != nil && !os.IsNotExist(err) {
		log.Fatalln("Removing test db:", err)
	}
	h := newTestHandler()
	defer h.db.Close()

	for _, k := range shortenTests {
		c := cases[k]
		short, err := h.Shorten(c.long)
		if short != c.short {
			t.Errorf("%s, Shorten: got %s, want %s", k, short, c.short)
		}
		if c.wanterr != (err != nil) {
			t.Errorf("%s, errors: got err %v, want err %v", k, err != nil, c.wanterr)
		}
	}
}

func TestHandler_Expand(t *testing.T) {
	h := newTestHandler()
	defer h.db.Close()

	for _, k := range expandTests {
		c := cases[k]
		long, err := h.Expand(c.short)
		if long != c.long {
			t.Errorf("%s, Expand: got %q, want %q", k, long, c.long)
		}
		if c.wanterr != (err != nil) {
			t.Errorf("%s, errors: got err %v, want err %v", k, err != nil, c.wanterr)
		}
	}
}

// Test the Handler's responses, integrates shorten/expand methods
func TestHandler_ServeHTTP(t *testing.T) {
	// Needs a fresh db
	err := os.Remove("testdata/test.db")
	if err != nil && !os.IsNotExist(err) {
		log.Fatalln("Removing test db:", err)
	}
	h := newTestHandler()
	defer h.db.Close()

	serveShortenTests := append(shortenTests, "Invalid long URL")
	serveExpandTests := expandTests

	for _, k := range serveShortenTests {
		c := cases[k]
		req := httptest.NewRequest("POST", "/", strings.NewReader(c.long))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)
		// If we are expecting an error, don't check the body for the short URL.
		// The error text, which we aren't currently testing :), will be there.
		if !c.wanterr {
			short := string(body)
			if short != c.short {
				t.Errorf("%s, Serve Shorten: got %q, want %q", k, short, c.short)
			}
		}
		if resp.StatusCode != c.shortenResCode {
			t.Errorf("%s, response code: got %d, want %d", k, resp.StatusCode, c.shortenResCode)
		}
	}
	for _, k := range serveExpandTests {
		c := cases[k]
		req := httptest.NewRequest("GET", "http://nomatter?short="+c.short, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		resp := w.Result()
		long := resp.Header.Get("Location")
		if long != c.long {
			t.Errorf("%s, Serve Expand: got %q, want %q", k, long, c.long)
		}
		if resp.StatusCode != c.expandResCode {
			t.Errorf("%s, response code: got %d, want %d", k, resp.StatusCode, c.expandResCode)
		}
	}
}

package short

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

const shortenerBucket string = "longs"

// Handler responds to HTTP Requests, as appopriate (See ServeHTTP method).
// Satisfies the http.Handler interface
type Handler struct {
	shortBase string
	db        *bolt.DB
}

func initDB(dbfile, bucket string) (*bolt.DB, error) {

	// DB will be created if it doesn't exist.
	db, err := bolt.Open(dbfile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return db, fmt.Errorf("initDB open (%s): %s", dbfile, err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(shortenerBucket))
		return err
	})
	if err != nil {
		return db, fmt.Errorf("initDB create bucket: %s", err)
	}

	return db, nil
}

// ShortenerHandler initialises and returns a new Handler
func ShortenerHandler(shortBase string, dbfile string) (Handler, error) {
	h := Handler{}

	// Validate the short URL, assuming it is absolute.
	_, err := url.ParseRequestURI(shortBase)
	if err != nil {
		return h, fmt.Errorf("Invalid shortBase (%s): %s", shortBase, err)
	}

	// Ensuring the trailing slash is present simplifies adding the key to the
	// URL because we can now confidently use string concatenation.
	if shortBase[len(shortBase)-1:] != "/" {
		shortBase = shortBase + "/"
	}

	db, err := initDB(dbfile, shortenerBucket)
	if err != nil {
		return h, fmt.Errorf("ShortenerHandler initDB: %s", err)
	}

	h.shortBase = shortBase
	h.db = db
	return h, nil
}

// Close closes the Handler's the database connection.
func (h *Handler) Close() {
	h.db.Close()
}

// Shorten creates a URL from shortBase by adding a unique key
// which will refer back the long URL. The key is generated by
// incrementing the db's sequence number, then formatting that
// number in base 36. Shorten does not validate the long URL.
func (h *Handler) Shorten(long string) (string, error) {
	var key, short string

	err := h.db.Update(func(tx *bolt.Tx) error {

		// This bucket is created when the Handler is created.
		b := tx.Bucket([]byte(shortenerBucket))

		// Generate a unique sequential number for making the key.
		// This returns an error only if the Tx is closed or not writeable.
		// Per the boltdb author: That can't happen in an Update() call,
		// so I ignore the error check.
		seq, _ := b.NextSequence()

		// Convert the unique number to base 36 (a-z0-9), append to base.
		key = strconv.FormatUint(seq, 36)

		// Save key -> long pair to the bucket.
		return b.Put([]byte(key), []byte(long))
	})
	if err != nil {
		err := fmt.Errorf("Shorten from %s: %s", long, err)
		return short, err
	}

	// We can join the URL/path by concatenation safely because
	// we prepared the shortBase when creating the Handler.
	short = h.shortBase + key
	return short, nil
}

// Expand checks for a reference from the given short URL
// to a long URL.
func (h *Handler) Expand(short string) (string, error) {
	var long string

	// Ensure short URL has correct base
	if len(short) <= len(h.shortBase) || short[:len(h.shortBase)] != h.shortBase {
		return long, fmt.Errorf("Bad short URL, %s (base=%s)", short, h.shortBase)
	}

	// Slice key from short url.
	key := short[len(h.shortBase):]

	err := h.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(shortenerBucket))
		v := b.Get([]byte(key))
		long = string(v)
		return nil
	})
	if err != nil {
		err := fmt.Errorf("db error, Expand from %s: %s", short, err)
		return long, err
	}

	return long, nil
}

// ServeHTTP writes reply headers and data to the ResponseWriter.
//
// GET or HEAD requests will expand the short URL and redirect
// to the original long URL, if found.
// Clients should encode the short URL value with the key "short".
// On success, the response will redirect to long with status 303.
//
// POST requests will shorten a long URL and return the short URL.
// Clients should write only the long URL to the request body.
// On Success, the respone body will be the short URL, and the
// status 200.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	// Expand
	case http.MethodGet, http.MethodHead:
		short := r.URL.Query().Get("short")
		long, err := h.Expand(short)
		if err != nil {
			log.Println("Shortener Handler:", err)
			if strings.Contains(err.Error(), "Bad short URL") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(long) < 1 {
			http.NotFound(w, r)
			return
		}

		log.Printf("Expanded %s -> %s\n", short, long)
		http.Redirect(w, r, long, http.StatusSeeOther)
		return

	// Shorten
	case http.MethodPost:
		raw, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Shortener Handler: Reading request body:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Validate the long URL. Parse accepts almost anything as valid.
		// The stricter alternative, ParseRequestURI, removes #fragments.
		// If that is an acceptable loss, it may be worth switching.
		// This serves as a weak URL sanitiser, and may alter a valid URL
		// in some cases.
		// Need to assess whether this is neccessary/sufficient/appropriate.
		longu, err := url.Parse(string(raw))
		if err != nil {
			log.Printf("Shortener Handler: parsing long raw: %s %s\n", raw, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		short, err := h.Shorten(longu.String())
		if err != nil {
			log.Printf("Shortener Handler: %s %s\n", longu.String(), err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Shortened %s -> %s\n", longu.String(), short)
		io.WriteString(w, short)
		return

	default:
		// TODO(zdjones) Do we want to log this?
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}
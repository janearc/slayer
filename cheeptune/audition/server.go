package audition

import (
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

// maxBody is the most a vote's body may hold, in bytes. Picked, 2026-11-12:
// a vote is a name and a word.
const maxBody = 4096

// jsonType is the content type a vote must be sent with. A page on another
// site cannot send it without a preflight request, which this server never
// answers.
const jsonType = "application/json"

// Server serves a library folder and the votes API.
type Server struct {
	root   *os.Root
	ledger *ledger
	hosts  map[string]bool
}

// hostsOf is the Host headers a request to this server may carry: this
// machine's two names with the port the server listens on.
func hostsOf(port string) map[string]bool {
	return map[string]bool{"localhost:" + port: true,
		"127.0.0.1:" + port: true}
}

// NewServer checks the folder, lists its tunes, and opens it for serving
// on a port. Only files inside the folder are served: a path that climbs
// out, and a link that leads out, are refused. A request whose Host header
// is not this machine with that port is refused.
func NewServer(dir, port string) (*Server, error) {
	if n, err := strconv.Atoi(port); err != nil || n < 0 || n > maxPort {
		return nil, fmt.Errorf("port %q is not a port number; use "+
			"0 to %d", port, maxPort)
	}
	names, err := Names(dir)
	if err != nil {
		return nil, fmt.Errorf("the folder %s cannot be read: %w", dir, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("the folder %s holds no tunes; name the "+
			"folder that has NAME.json files", dir)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &Server{root: root, ledger: newLedger(dir, time.Now),
		hosts: hostsOf(port)}, nil
}

// maxPort is the largest port number. Defined by TCP.
const maxPort = 65535

// Close lets the folder go. A server closed serves nothing more.
func (s *Server) Close() error { return s.root.Close() }

// Handler is the folder's files at their own paths and the votes at /votes,
// behind the Host check.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(s.root.FS()))
	mux.HandleFunc("/votes", s.votes)
	return s.guard(mux)
}

// guard refuses, with 403, a request whose Host header is not this machine
// with the server's port: a hostile name that resolves here would otherwise
// be let in.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.hosts[r.Host] {
			http.Error(w, fmt.Sprintf("host %q is refused; this server "+
				"answers to localhost and 127.0.0.1 with its own port "+
				"only", r.Host), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isJSON says a request is sent as JSON, with or without parameters such as
// a charset.
func isJSON(r *http.Request) bool {
	kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && kind == jsonType
}

// ballot is the body of a POST to /votes.
type ballot struct {
	Name string `json:"name"`
	Vote string `json:"vote"`
}

// votes answers GET with the record and POST with the record after a vote.
func (s *Server) votes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rec, err := s.ledger.Record()
		reply(w, rec, err, http.StatusInternalServerError)
	case http.MethodPost:
		if !isJSON(r) {
			http.Error(w, "a vote is sent with Content-Type: "+jsonType,
				http.StatusUnsupportedMediaType)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var b ballot
		if err := dec.Decode(&b); err != nil {
			reply(w, Record{}, fmt.Errorf("a vote is {\"name\": ..., "+
				"\"vote\": \"keep\"|\"toss\"|\"clear\"}: %w", err),
				http.StatusBadRequest)
			return
		}
		rec, err := s.ledger.cast(b.Name, b.Vote)
		reply(w, rec, err, http.StatusBadRequest)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "votes takes GET and POST", http.StatusMethodNotAllowed)
	}
}

// reply writes a record as JSON, or the error with the status given.
func reply(w http.ResponseWriter, rec Record, err error, status int) {
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}

// Loopback reports whether an address names this machine and no other: the
// host is 127.0.0.1 or localhost, with a port. An empty host would listen
// on every interface and is refused.
func Loopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("address %q is not host:port: %w", addr, err)
	}
	if host != "127.0.0.1" && host != "localhost" {
		return fmt.Errorf("address %q is refused: the votes are served on "+
			"this machine only, so use 127.0.0.1:PORT or localhost:PORT",
			addr)
	}
	return nil
}

// Serve serves the folder on a loopback address until it fails.
func Serve(dir, addr string) error {
	if err := Loopback(addr); err != nil {
		return err
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return err
	}
	s, err := NewServer(dir, port)
	if err != nil {
		return err
	}
	return http.Serve(ln, s.Handler())
}

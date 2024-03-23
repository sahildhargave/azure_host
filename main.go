// 😎😎😍😍🥰🥰😴😴
package main

import (
	"expvar"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	httpAddr   = flag.String("http", "localhost:8080", "Listen Address")
	pollPeriod = flag.Duration("poll", 5*time.Second, "Poll Period")
	version    = flag.String("version", "1.4", "Go Version")
)

const baseChangeURL = "https://go.googlesource.com/go/+/"

func main() {
	flag.Parse()
	changeURL := fmt.Sprintf("%sgo%s", baseChangeURL, *version)
	http.Handle("/", NewServer(*version, changeURL, *pollPeriod))

	// Start HTTP server
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

var (
	hitCount       = expvar.NewInt("hitCount")
	pollCount      = expvar.NewInt("pollCount")
	pollError      = expvar.NewString("pollError")
	pollErrorCount = expvar.NewInt("pollErrorCount")
)

type Server struct {
	version string
	url     string
	period  time.Duration

	mu  sync.RWMutex
	yes bool
}

func NewServer(version, url string, period time.Duration) *Server {
	s := &Server{version: version, url: url, period: period}
	go s.poll()
	return s
}

func (s *Server) poll() {
	for !s.yes {
		pollCount.Add(1)
		if isTagged(s.url) {
			s.mu.Lock()
			s.yes = true
			s.mu.Unlock()
			pollDone()
			return
		}
		pollSleep(s.period)
	}
}

var (
	pollSleep = time.Sleep
	pollDone  = func() {}
)

func isTagged(url string) bool {
	r, err := http.Head(url)
	if err != nil {
		log.Print(err)
		pollError.Set(err.Error())
		pollErrorCount.Add(1)
		return false
	}
	return r.StatusCode == http.StatusOK
}

// ServeHTTP implements the HTTP user interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hitCount.Add(1)
	s.mu.RLock()
	data := struct {
		URL     string
		Version string
		Yes     bool
	}{
		s.url,
		s.version,
		s.yes,
	}

	s.mu.RUnlock()
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Print(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

var tmpl = template.Must(template.New("tmpl").Parse(
	`
	<!DOCTYPE html>
	<html>
	<body>
	<center>
	<h2> Is Go {{.Version}}</h2>
	<h1>
	{{if .Yes}}
	<a href="{{.URL}}">Yes!</a>
	{{else}}
	No. 
	{{end}}
	</h1>
	</center>
	</body>
	</html>
	`))

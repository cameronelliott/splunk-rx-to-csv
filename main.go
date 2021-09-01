package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/caddyserver/certmagic"
	"github.com/spf13/pflag"
)

func checkFatal(err error) {
	if err != nil {
		_, fileName, fileLine, _ := runtime.Caller(1)
		log.Fatalf("FATAL %s:%d %v", filepath.Base(fileName), fileLine, err)
		// log.Fatalf calls os.Exit(1)
	}
}

func checkNotFatal(err error) bool {
	if err != nil {
		_, fileName, fileLine, _ := runtime.Caller(1)
		log.Printf("NON-FATAL %s:%d %v", filepath.Base(fileName), fileLine, err)
		return true
	}
	return false
}

// func httpError(w http.ResponseWriter, err error) {
// 	m := time.Now().UTC().Format(time.RFC3339) + " :: " + err.Error()
// 	log.Println(m)
// 	http.Error(w, m, http.StatusInternalServerError)
// }

const testStream = `
{
	"event": {
		"line": "",
		"source": "stdout",
		"tag": "da8b33636b41"
	},
	"time": "1630529225.902264",
	"host": "docker-desktop"
}{
	"event": {
		"line": "Hello from Docker!",
		"source": "stdout",
		"tag": "da8b33636b41"
	},
	"time": "1630529225.902291",
	"host": "docker-desktop"
}
`

var domain = pflag.String("domain", "", "https domain name")
var testFlag = pflag.Bool("test", false, "testing")

type Message struct {
	Time, Host string

	Event struct{ Line, Source, Tag string }
}

func test() {
	for _, m := range parse(testStream) {
		fmt.Println(m.Time, m.Host, m.Event.Line, m.Event.Source, m.Event.Tag)
	}
}

func parse(splunkJson string) []Message {

	x := make([]Message, 0)
	dec := json.NewDecoder(strings.NewReader(splunkJson))
	for {
		var m Message
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		x = append(x, m)
	}
	return x
}

func main() {
	pflag.Parse()
	if *testFlag {
		test()
		return
	}

	if *domain == "" {
		checkFatal(fmt.Errorf("--domain not set, fatal"))
	}

	f, err := os.OpenFile("splunklog.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	checkFatal(err)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if checkNotFatal(err) {
			return
		}
		for _, m := range parse(string(body)) {
			fmt.Fprintf(f, "%s,%s,%s,%s,%s\n", m.Time, m.Host, m.Event.Source, m.Event.Tag, m.Event.Line)
		}

	})

	err = certmagic.HTTPS([]string{*domain}, mux)
	checkFatal(err)

}

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var nerrors int
var failedToOpen = make([]string, 0, 32)
var fetchedShort = make([]string, 0, 128)
var fetchedLong = make([]string, 0, 32)

func report(format string, a ...interface{}) {
	log.Printf(format, a...)
	nerrors++
	if nerrors > 9 {
		panic(errors.New("too many errors"))
	}
}

func openLocal(localfile string) *os.File {
	if finfo, err := os.Stat(localfile); err == nil && finfo.Size() > 0 {
		return nil
	}

	if alternate := strings.TrimSuffix(localfile, "o.gz"); alternate != localfile {
		alternate = alternate + "d.bz2"
		if finfo, err := os.Stat(alternate); err == nil {
			if finfo.Size() > 0 {
				return nil
			}
			os.Remove(alternate)
		}
	}

	out, err := os.Create(localfile)
	if err != nil {
		log.Printf("Unable to create %s: %s", localfile, err.Error())
		return nil
	}

	return out
}

func fetch(client *http.Client, url, localfile, name string) bool {
	var out *os.File
	var err error
	var req *http.Request
	var resp *http.Response

	if localfile != "" {
		if out = openLocal(localfile); out == nil {
			return false
		}
		defer func() {
			out.Close()
		}()
	} else {
		log.Fatalln("Don't know what to do with fetch of", url)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 480*time.Second)
	req, _ = http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err = client.Do(req)
	if err != nil || resp.StatusCode >= 300 {
		cancel()
		os.Remove(localfile)
		if err == nil {
			if resp.StatusCode == 404 {
				failedToOpen = append(failedToOpen, name)
			} else {
				report("Unable to GET %s: %s", url, resp.Status)
			}
		} else if strings.Contains(err.Error(), "550 Failed to open file") ||
			strings.Contains(err.Error(), "TLS handshake timeout") {
			failedToOpen = append(failedToOpen, name)
		} else if err.Error() == "i/o timeout" { // an internal/poll.TimeoutError
			panic(err)
		} else {
			report("Unable to GET %s: %s", url, err.Error())
		}
		return false
	}
	defer func() {
		resp.Body.Close()
		cancel()
	}()

	if _, err := io.Copy(out, resp.Body); err != nil {
		report("Failed to GET %s into %s: %s", url, localfile, err.Error())
		os.Remove(localfile)
		return false
	}

	return true
}

func getenv(name, defaultValue string) string {
	if evar := os.Getenv(name); evar != "" {
		return evar
	}
	return defaultValue
}

func getNameList(response *http.Response) []string {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		report("Unable to ready body for %s: %s",
			response.Request.URL.String(), err.Error())
		return nil
	}

	res := make([]string, 0, 2048)
	start := []byte("<a href=\"")
	for len(body) > 0 {
		// Find the text inside an <a href=\".*\" block.
		idx := bytes.Index(body, start)
		if idx < 0 {
			break
		}
		body = body[idx+len(start):]
		idx = bytes.IndexByte(body, '"')
		if idx < 0 {
			break
		}
		url := body[:idx]
		body = body[idx+1:]

		// Filter urls: allow ????/, or *.gz, ignore sum_gz/ and \?C=* and /*.
		if idx == 5 && url[4] == '/' { // "abcd/" becomes "abcd"
			res = append(res, string(url[:4]))
		} else if idx > 4 && bytes.Equal(url[idx-3:], []byte(".gz")) { // keep "*.gz"
			res = append(res, string(url))
		} else if idx == 7 && bytes.Equal(url, []byte("sum_gz/")) {
			// ignore
		} else if idx > 3 && bytes.Equal(url[0:3], []byte("?C=")) {
			// ignore
		} else if idx > 18 && bytes.Equal(url[idx-11:idx], []byte(".files.list")) {
			// ignore (yyyy.ddd.files.list)
			//         0123456789012345678
		} else if idx > 0 && url[0] == '/' {
			// ignore
		} else {
			log.Printf("Unexpected URL in directory listing: %s", url)
		}
	}

	return res
}

func fetchDay(client *http.Client, url, year, dnum string) {
	var resp *http.Response
	var err error

	localdir := fmt.Sprintf("%s/%s", year, dnum)
	if err = os.MkdirAll(localdir, os.ModePerm); err != nil {
		log.Printf("Unable to mkdir %s: %s", localdir, err.Error())
		return
	}

	dayURL := fmt.Sprintf("%s%s/%s/", url, year, dnum)
	if resp, err = client.Get(dayURL); err != nil {
		report("Unable to GET %s: %s", dayURL, err.Error())
		return
	}

	names := getNameList(resp)
	if names == nil || len(names) < 1 {
		return
	}
	// log.Printf("%s: %d entries", dirname, len(names))

	defer func() {
		if len(failedToOpen) > 0 {
			log.Printf("%s failed to open: %s", localdir,
				strings.Join(failedToOpen, " "))
			failedToOpen = failedToOpen[:0]
		}
		var reportText string
		if len(fetchedShort) > 0 {
			reportText = reportText + " " + strings.Join(fetchedShort, " ")
			fetchedShort = fetchedShort[:0]
		}
		if len(fetchedLong) > 0 {
			reportText = reportText + " " + strings.Join(fetchedLong, " ")
			fetchedLong = fetchedLong[:0]
		}
		if reportText == "" {
			reportText = " All up to date"
		}
		log.Printf("%s fetched: %s", localdir, reportText[1:])
	}()

	for _, name := range names {
		if len(name) == 4 {
			filename := fmt.Sprintf("/%s%s0.%so.gz", name, dnum, year[2:4])
			if fetch(client, dayURL+name+filename, localdir+filename, name) {
				fetchedShort = append(fetchedShort, name)
			}
		} else {
			if fetch(client, dayURL+name, localdir+"/"+name, name) {
				fetchedLong = append(fetchedLong, name)
			}
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <year> <dnum> ...", os.Args[0])
	}

	// What is the server URL?
	url := getenv("CORS_SERVER", "https://geodesy.noaa.gov/corsdata/rinex/")

	// Sanity-check arguments before we connect.
	yearRE := regexp.MustCompile("^20[0-9][0-9]$")
	year := os.Args[1]
	if yearRE.Find([]byte(year)) == nil {
		log.Fatalf("Expected year to be 2000-2099, got '%s'", year)
	}
	dnumRE := regexp.MustCompile("^[0-3][0-9][0-9]$")
	for _, dnum := range os.Args[2:] {
		if dnumRE.Find([]byte(dnum)) == nil {
			log.Fatalf("Expected day number to be 000-365, got '%s'\n", dnum)
		}
	}

	// Create our HTTP client object.
	client := new(http.Client)
	defer func() {
		if r := recover(); r != nil {
			log.Fatalln(r)
		}
	}()

	// Fetch files for each specified day.
	nerrors = 0
	for _, dnum := range os.Args[2:] {
		fetchDay(client, url, year, dnum)
	}
}

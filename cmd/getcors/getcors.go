package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

var nerrors int
var md5File *os.File

func report(format string, a ...interface{}) {
	log.Printf(format, a...)
	nerrors++
	if nerrors > 9 {
		log.Fatalln(" ... bailing out")
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
		log.Printf("Unable to create %s: %s\n", localfile, err.Error())
		return nil
	}

	return out
}

func fetch(cors *ftp.ServerConn, file, localfile string) bool {
	var out *os.File
	var err error

	if localfile != "" {
		if out = openLocal(localfile); out == nil {
			return false
		}
	} else if strings.HasSuffix(file, ".md5") {
		out = md5File
	} else {
		log.Fatalln("Don't know what to do with fetch of", file)
	}

	in, err := cors.Retr(file)
	if err != nil {
		if out != nil && out != md5File {
			out.Close()
		}
		os.Remove(localfile)
		if strings.Contains(err.Error(), "550 Failed to open file") {
			log.Printf("Unable to RETR %s: %s\n", file, err.Error())
		} else if err.Error() == "i/o timeout" { // an internal/poll.TimeoutError
			log.Fatalln(" ... bailing out")
		} else {
			report("Unable to RETR %s: %s\n", file, err.Error())
		}
		return false
	}
	defer in.Close()

	in.SetDeadline(time.Now().Add(480 * time.Second))
	if nbytes, err := io.Copy(out, in); err != nil {
		report("Failed to RETR %s into %s: %s\n", file, localfile, err.Error())
		os.Remove(localfile)
	} else if !strings.HasSuffix(file, ".md5") {
		log.Printf("%s: %d bytes\n", file, nbytes)
	}
	if out != nil && out != md5File {
		out.Close()
	}

	return true
}

func getenv(name, defaultValue string) string {
	if evar := os.Getenv(name); evar != "" {
		return evar
	}
	return defaultValue
}

func fetchDay(cors *ftp.ServerConn, spath, year, dnum string) {
	var err error

	localdir := fmt.Sprintf("%s/%s", year, dnum)
	if err = os.MkdirAll(localdir, os.ModePerm); err != nil {
		log.Printf("Unable to mkdir %s: %s\n", localdir, err.Error())
		return
	}

	md5File, err = os.OpenFile(localdir+"/md5sums", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		report("Unable to open %s/md5sums: %s\n", localdir, err.Error())
		return
	}

	dirname := fmt.Sprintf("%s/rinex/%s/%s", spath, year, dnum)
	if err = cors.ChangeDir(dirname); err != nil {
		report("Unable to CWD %s: %s\n", dirname, err.Error())
		md5File.Close()
		return
	}

	names, err := cors.NameList(".")
	if err != nil {
		report("Unable to NLST %s: %s\n", dirname, err.Error())
		md5File.Close()
		return
	}
	log.Printf("%s: %d entries\n", dirname, len(names))

	for _, name := range names {
		if len(name) == 4 {
			filename := fmt.Sprintf("/%s%s0.%so.gz", name, dnum, year[2:4])
			if fetch(cors, name+filename, localdir+filename) {
				filename = filename + ".md5"
				fetch(cors, name+filename, "")
			}
		} else if name != "sum_gz" {
			fetch(cors, name, localdir+"/"+name)
		}
	}

	md5File.Close()
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <year> <dnum> ...\n", os.Args[0])
	}

	// What is the FTP server name and path?
	server := getenv("CORS_SERVER", "geodesy.noaa.gov:ftp")
	spath := getenv("CORS_PATH", "/cors")
	user := getenv("CORS_USER", "anonymous")
	password := getenv("CORS_PASS", "")
	if password == "" {
		fmt.Println("Please set the CORS_PASS environment variable to your email address")
		os.Exit(1)
	}

	// Sanity-check arguments before we connect.
	yearRE := regexp.MustCompile("^20[0-9][0-9]$")
	year := os.Args[1]
	if yearRE.Find([]byte(year)) == nil {
		log.Fatalf("Expected year to be 2000-2099, got '%s'\n", year)
	}
	dnumRE := regexp.MustCompile("^[0-3][0-9][0-9]$")
	for _, dnum := range os.Args[2:] {
		if dnumRE.Find([]byte(dnum)) == nil {
			log.Fatalf("Expected day number to be 000-365, got '%s'\n", dnum)
		}
	}

	// Connect to the server.
	cors, err := ftp.Dial(server)
	if err != nil {
		log.Fatalln("Cannot connect to server:", err)
	}
	if err := cors.Login(user, password); err != nil {
		log.Fatalln("Unable to log in to FTP server:", err)
	}

	// Fetch files for each specified day.
	nerrors = 0
	for _, dnum := range os.Args[2:] {
		fetchDay(cors, spath, year, dnum)
	}
}

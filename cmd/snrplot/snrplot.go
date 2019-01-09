package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/entrope/gnss/rinex"
)

var templ = template.Must(template.New("").Parse(`<!DOCTYPE html><html>
<style type="text/css">table { border: 1px outset grey; padding: 1px }
td { border: thin inset grey; margin: 1; text-align: center }</style>
<title>{{ .Basename }} SNRs</title><body>
<table><caption>{{ .Basename }} SNRs; interval = {{ .Interval }} seconds;
vertical range is 20 to 60 (dB-Hz assumed)</caption>
<thead><tr><th><th>L1<th>L5<th>L2</thead><tbody>
{{range $svid, $map := .SNRs}}
<tr><td>{{$svid}}
<td>{{if $map.L1}}<img src="{{ $map.L1 }}">{{else}}no data{{end}}
<td>{{if $map.L5}}<img src="{{ $map.L5 }}">{{else}}no data{{end}}
<td>{{if $map.L2}}<img src="{{ $map.L2 }}">{{else}}no data{{end}}
{{- end}}
</tbody></table></body></html>`))

type TemplateData struct {
	// Basename contains the four-character site code, followed by the
	// four-digit Julian day and hour indicator.
	Basename string

	// Interval is the interval between samples at this site.
	Interval int

	// SNRs maps from three-character satellite name (G01, E04, etc.)
	// to frequency name (L1, L2, L5) to the URI for the SNR image.
	SNRs map[string]map[string]string
}

type SignalDay struct {
	// snr is a 2-D histogram of SNR values.  The first index is time,
	// in two-minute units.  The second index is scaled SNR, as
	// 2 * (SNR - 20).
	snr [720][80]byte
}

type SiteDay struct {
	// Basename contains the four-character site code, followed by the
	// four-digit Julian day and hour indicator.
	Basename string

	// Interval is the interval between samples at this site.
	Interval int

	// Sats maps from a satellite-signal identifier to SNR values for it.
	// The key's first byte is the frequency number ('1', '2', '5') and
	// the other three bytes are a RINEX satellite identifier.
	Sats map[[4]byte]*SignalDay
}

var (
	palettes = make(map[int][]color.NRGBA)
	njobs    = flag.Uint("j", 1, "number of concurrent jobs to launch")
	suffix   *regexp.Regexp
)

func makePalette(g, r, t int) []color.NRGBA {
	res := make([]color.NRGBA, t)
	res[0] = color.NRGBA{0, 0, 0, 0}
	for i := 1; i < g; i++ {
		res[i] = color.NRGBA{24, 90, 169, 255} // dark blue
	}
	for i := g; i < r; i++ {
		res[i] = color.NRGBA{0, 140, 72, 255} // dark green
	}
	for i := r; i < t; i++ {
		res[i] = color.NRGBA{238, 46, 47, 255} // dark red
	}
	return res
}

func makePalettes() {
	palettes[1] = makePalette(7, 22, 120)
	palettes[5] = makePalette(4, 10, 24)
	palettes[15] = makePalette(2, 4, 8)
	palettes[30] = makePalette(1, 2, 4)
}

func loadDay(fname string) (*SiteDay, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	var r io.Reader = f

	if strings.HasSuffix(fname, ".gz") {
		if r, err = gzip.NewReader(r); err != nil {
			return nil, err
		}
	}
	basename := suffix.ReplaceAllString(fname, "")
	if idx := strings.LastIndexByte(basename, '/'); idx >= 0 {
		basename = basename[idx+1:]
	}
	res := &SiteDay{
		Basename: basename,
		Interval: 0,
		Sats:     make(map[[4]byte]*SignalDay, 64),
	}
	var day byte
	first := 0
	or := &rinex.ObsReader{}
	or.HeaderFunc = func(label, value string) error {
		if strings.TrimSpace(label) == "INTERVAL" {
			flt, err := strconv.ParseFloat(strings.TrimSpace(value[:11]), 64)
			if err != nil {
				return err
			}
			res.Interval = int(math.Round(flt))
		}
		return nil
	}
	or.ObsFunc = func(rec rinex.ObservationRecord) error {
		if rec.EpochFlag > 1 {
			return nil
		}
		if day == 0 {
			day = rec.Day
		} else if day != rec.Day {
			return nil
		}
		if res.Interval == 0 {
			seconds := int(rec.Hour)*3600 + int(rec.Minute)*60 + int(rec.Second)
			if first == 0 {
				first = seconds
			} else {
				res.Interval = seconds - first
			}
		}
		horiz := (int(rec.Hour)*60 + int(rec.Minute)) / 2
		for _, sv := range rec.Sat {
			if sv.PRN[0] != 'G' && sv.PRN[0] != 'E' {
				continue
			}
			var key [4]byte
			copy(key[1:4], sv.PRN[:])
			obsCodes := or.Observations[sv.PRN[0]]
			if obsCodes == nil {
				obsCodes = or.Observations[' ']
			}
			for j, o := range sv.Obs {
				obsCode := obsCodes[j]
				if obsCode[0] != 'S' || o.Value == 0 {
					continue
				}
				key[0] = obsCode[1]
				s := res.Sats[key]
				if s == nil {
					s = new(SignalDay)
					res.Sats[key] = s
				}
				y := math.Round(2 * (o.Value - 20))
				y = math.Max(0, math.Min(float64(len(s.snr[0])-1), y))
				s.snr[horiz][int(y)]++
				if s.snr[horiz][int(y)] > 120 {
					panic("snr counter got too big")
				}
			}
		}
		return nil
	}

	if err = or.Parse(r); err != nil {
		return nil, err
	}

	return res, nil
}

func loadDays(wg *sync.WaitGroup, filenames <-chan string, sitedays chan<- *SiteDay) {
	defer wg.Done()
	for {
		fname, ok := <-filenames
		if !ok {
			break
		}
		siteDay, err := loadDay(fname)
		if err != nil {
			fmt.Printf("%s: %s\n", fname, err.Error())
			continue
		}
		sitedays <- siteDay
	}
}

func drawGrid(img *image.NRGBA) {
	grey := color.NRGBA{R: 119, G: 136, B: 153, A: 255} // light slate grey
	if img.Rect.Min.X != 0 || img.Rect.Min.Y != 0 {
		panic("Can only draw grid for zero-origin image")
	}
	width := img.Rect.Max.X
	height := img.Rect.Max.Y
	for i := 1; i < 24; i++ {
		x := width * i / 24
		for y := 0; y < 4; y++ {
			img.Set(x, height-1-y, grey)
		}
	}
	for i := 1; i < 6; i++ {
		x := width * i / 6
		for y := 0; y < height; y++ {
			img.Set(x, y, grey)
		}
	}
	for x := 0; x < width; x++ {
		for y := 20; y < height; y += 20 {
			img.Set(x, height-1-y, grey)
		}
	}
}

func plotDay(siteDay *SiteDay) error {
	colors := palettes[siteDay.Interval]
	if colors == nil {
		return fmt.Errorf("unexpected interval %d", siteDay.Interval)
	}

	td := TemplateData{
		Basename: siteDay.Basename,
		Interval: siteDay.Interval,
		SNRs:     make(map[string]map[string]string),
	}
	for k, v := range siteDay.Sats {
		svid := string(k[1:])
		inner := td.SNRs[svid]
		if inner == nil {
			inner = make(map[string]string)
			td.SNRs[svid] = inner
		}
		bb := bytes.Buffer{}
		width := len(v.snr)
		height := len(v.snr[0])
		img := image.NewNRGBA(image.Rect(0, 0, width, height))
		drawGrid(img)
		for x := range v.snr {
			for y, h := range v.snr[x] {
				if h == 0 {
					continue
				}
				z := int(h)
				if z >= len(colors) {
					z = len(colors) - 1
				}
				img.Set(x, height-1-y, colors[z])
			}
		}
		if err := png.Encode(&bb, img); err != nil {
			return err
		}
		freq := fmt.Sprintf("L%c", k[0])
		inner[freq] = "data:image/png;base64," +
			base64.StdEncoding.EncodeToString(bb.Bytes())
	}

	f, err := os.Create(fmt.Sprintf("%ds/%s.html", siteDay.Interval,
		siteDay.Basename))
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(fmt.Sprintf("%ds", siteDay.Interval), os.ModePerm)
		if err == nil {
			f, err = os.Create(fmt.Sprintf("%ds/%s.html",
				siteDay.Interval, siteDay.Basename))
		}
	}
	if err != nil {
		return err
	}
	templ.Execute(f, td)
	return f.Close()
}

func plotDays(wg *sync.WaitGroup, sitedays <-chan *SiteDay) {
	defer wg.Done()
	for {
		siteday, ok := <-sitedays
		if !ok {
			break
		}
		if err := plotDay(siteday); err != nil {
			fmt.Printf("%s: %s\n", siteday.Basename, err.Error())
		}
	}
}

func main() {
	flag.Parse()
	makePalettes()
	suffix = regexp.MustCompile(`\.(rnx|\d\do)(\.gz)?$`)

	filenames := make(chan string, 8)
	sitedays := make(chan *SiteDay, 8)

	wg1 := sync.WaitGroup{}
	wg1.Add(1)
	go plotDays(&wg1, sitedays)

	jobCount := int(*njobs)
	if jobCount == 0 {
		jobCount = runtime.NumCPU()
	}
	wg2 := sync.WaitGroup{}
	for i := 0; i < jobCount; i++ {
		wg2.Add(1)
		go loadDays(&wg2, filenames, sitedays)
	}

	for _, fname := range flag.Args() {
		filenames <- fname
	}

	close(filenames)
	wg2.Wait()
	close(sitedays)
	wg1.Wait()
}

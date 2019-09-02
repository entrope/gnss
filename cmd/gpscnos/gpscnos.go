package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/entrope/gnss/rinex"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// SignalDay holds one day worth of statistics for a single signal.
type SignalDay struct {
	// snr is a 2-D histogram of SNR values.  The first index is time,
	// in two-minute units.  The second index is scaled SNR, as
	// 2 * (SNR - 20).
	snr [720][80]byte
}

// SiteDay holds the statistics for one day at one site.
type SiteDay struct {
	// Basename contains the four-character site code, followed by the
	// four-digit Julian day and hour indicator.
	Basename string

	// Interval is the interval between samples at this site.
	Interval int

	// Year is the (four-digit) year of the data.
	Year int

	// Month is the (one-based) month of the data.
	Month int

	// Day is the (one-based, within the month) day of the data.
	Day int

	// Sats maps a gpsIdx value to the SignalDay structure for that SV.
	Sats []*SignalDay
}

var (
	palette  []color.NRGBA
	njobs    = flag.Uint("j", 1, "number of concurrent jobs to launch")
	linkFlag = flag.Int("link", 1, "link number to plot (1, 2 or 5)")
	link     byte
	suffix   *regexp.Regexp
	gpsIdx   = [...]int{-1, 0, 1, 2, -1, 3, 4, 5, 6, 7, 8, 9, 10, 11,
		12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27,
		28, 29, 30}
	satNames = []string{
		"G01", "G02", "G03", "G05", "G06", "G07", "G08", "G09",
		"G10", "G11", "G12", "G13", "G14", "G15", "G16", "G17",
		"G18", "G19", "G20", "G21", "G22", "G23", "G24", "G25",
		"G26", "G27", "G28", "G29", "G30", "G31", "G32",
	}
)

func rgb(r, g, b byte) color.NRGBA {
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}

func makePalette() {
	palette = []color.NRGBA{
		rgb(167, 206, 227),
		rgb(31, 120, 180),
		rgb(178, 223, 138),
		rgb(51, 160, 44),
		rgb(251, 154, 153),
		rgb(227, 26, 28),
		rgb(253, 191, 111),
		rgb(255, 127, 0),
		rgb(202, 178, 214),
		rgb(106, 61, 154),
		rgb(177, 89, 40),
	}
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
		Sats:     make([]*SignalDay, 31),
	}
	last := -1
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
		if res.Day == 0 {
			res.Year = int(rec.Year)
			res.Month = int(rec.Month)
			res.Day = int(rec.Day)
		} else if res.Day != int(rec.Day) {
			return nil
		}
		seconds := int(rec.Hour)*3600 + int(rec.Minute)*60 + int(rec.Second)
		if last >= 0 {
			interval := seconds - last
			if res.Interval == 0 || interval < res.Interval {
				res.Interval = interval
			}
		}
		last = seconds
		horiz := seconds / 120
		for _, sv := range rec.Sat {
			if sv.PRN[0] != 'G' {
				continue
			}
			prn := (sv.PRN[1]-'0')*10 + sv.PRN[2] - '0'
			idx := gpsIdx[prn]
			if idx < 0 {
				continue
			}
			obsCodes := or.Observations[sv.PRN[0]]
			if obsCodes == nil {
				obsCodes = or.Observations[' ']
			}
			for j, o := range sv.Obs {
				obsCode := obsCodes[j]
				if obsCode[0] != 'S' || obsCode[1] != '1' {
					continue
				}
				s := res.Sats[idx]
				if s == nil {
					s = new(SignalDay)
					res.Sats[idx] = s
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
	if img.Rect.Min.X != 0 || img.Rect.Min.Y != 0 {
		panic("Can only draw grid for zero-origin image")
	}
	white := rgb(255, 255, 255)
	grey := rgb(119, 136, 153) // light slate grey
	height := img.Rect.Max.Y
	width := img.Rect.Max.X
	draw.Draw(img, img.Rect, &image.Uniform{C: white},
		image.Point{0, 0}, draw.Src)
	for i := 1; i < 24; i++ {
		x := width * i / 24
		for y := 0; y < 4; y++ {
			img.SetNRGBA(x, height-1-y, grey)
		}
	}
	for i := 1; i < 6; i++ {
		x := width * i / 6
		for y := 0; y < height; y++ {
			img.SetNRGBA(x, y, grey)
		}
	}
	for x := 0; x < width; x++ {
		for y := 20; y < height; y += 20 {
			img.SetNRGBA(x, height-1-y, grey)
		}
	}
}

func addLabel(img *image.NRGBA, x, y int, label string, c color.NRGBA) {
	d := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot: fixed.Point26_6{
			X: fixed.Int26_6(x * 64),
			Y: fixed.Int26_6(y * 64),
		},
	}
	d.DrawString(label)
}

func plotDay(siteDay *SiteDay) error {
	var img [3]*image.NRGBA
	width := 720
	height := 480
	date := fmt.Sprintf("%04d-%02d-%02d", siteDay.Year, siteDay.Month, siteDay.Day)
	for i := range img {
		img[i] = image.NewNRGBA(image.Rect(0, 0, width, height))
		drawGrid(img[i])
		addLabel(img[i], 645, 14, date, rgb(0, 0, 0))
	}

	for idx, v := range siteDay.Sats {
		if v == nil {
			continue
		}
		i, j := idx/11, idx%11
		ofs := 40 * j
		c := palette[j]
		addLabel(img[i], 2, height-41-ofs, satNames[idx], c)
		for x := range v.snr {
			for y, h := range v.snr[x] {
				if h == 0 {
					continue
				}
				img[i].SetNRGBA(x, height-1-ofs-y, c)
			}
		}
	}

	for i := range img {
		f, err := os.Create(fmt.Sprintf("%s_G%d_L%c_%04d%02d%02d.png",
			siteDay.Basename[0:4], i, link, siteDay.Year, siteDay.Month,
			siteDay.Day))
		if err == nil {
			err = png.Encode(f, img[i])
			if err == nil {
				err = f.Close()
			}
		}
		if err != nil {
			return err
		}
	}

	return nil
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
	makePalette()
	suffix = regexp.MustCompile(`\.(rnx|\d\do)(\.gz)?$`)
	link = '0' + byte(*linkFlag)

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

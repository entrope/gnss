package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/entrope/gnss/rinex"
)

// This is a rather ad hoc program to convert RINEX observation files
// into Version 6 MAT-files.  Each output MAT-file contains one variable,
// a structure with the same name as the input file's basename.  If the
// input file name starts with a digit, the corresponding structure
// field is prefixed with an underscore.
//
// The structure contains one field for each satellite/signal pair.
// Each field is an Nx4 matrix, where each row is one observation, and
// the columns are respectively time, SNR (usually in units of dB-Hz),
// code-based pseudorange, and carrier phase.
//
// The data file looks like this:
// Bytes 0-127: MAT-file header
// Bytes 128-135: miMATRIX, length of data
// Bytes 136-151: miUINT32, 8, array flags value (mxSTRUCT_CLASS)
// Bytes 152-159: miINT32, 8, 1, 1 ("array" dimensions)
// Bytes 160-175: miINT8, 8, input file basename
// Bytes 176-183: miINT32, 4, 8 (compressed form of field name length)
// Bytes 184-???: miINT8, N*8, names of satellite/signal pairs ...
// Bytes N+0-N+56: miMATRIX, mxDOUBLE_CLASS, Nx4 dimensions, empty name,
//  miDOUBLE of N*32 bytes size (pr)

// nolint
const (
	miINT8       = 1
	miUINT8      = 2
	miINT16      = 3
	miUINT16     = 4
	miINT32      = 5
	miUINT32     = 6
	miSINGLE     = 7
	miDOUBLE     = 9
	miINT64      = 12
	miUINT64     = 13
	miMATRIX     = 14
	miCOMPRESSED = 15
	miUTF8       = 16
	miUTF16      = 17
	miUTF32      = 18

	mxCELL_CLASS   = 1
	mxSTRUCT_CLASS = 2
	mxOBJECT_CLASS = 3
	mxCHAR_CLASS   = 4
	mxDOUBLE_CLASS = 6
	mxSINGLE_CLASS = 7
	mxINT8_CLASS   = 8
	mxUINT8_CLASS  = 9
	mxINT16_CLASS  = 10
	mxUINT16_CLASS = 11
	mxINT32_CLASS  = 12
	mxUINT32_CLASS = 13
	mxINT64_CLASS  = 14
	mxUINT64_CLASS = 15
)

type observation struct {
	time    float32
	snr     float32
	code    float64
	carrier float64
}

func putUint32s(s []byte, v ...uint32) int {
	for i, x := range v {
		binary.LittleEndian.PutUint32(s[4*i:4*i+4], x)
	}
	return 4 * len(v)
}

func putFloat64(s []byte, v float64) {
	binary.LittleEndian.PutUint64(s[:8], math.Float64bits(v))
}

func saveMatrix(out io.Writer, varname string, series map[[4]byte][]observation) error {
	// Sort our observation codes, and decide how long each one is.
	snames := make([]string, 0, len(series))
	totalRows := 0
	for k, v := range series {
		snames = append(snames, string(k[:]))
		totalRows += len(v)
	}
	sort.Strings(snames)
	// Each "field" has an 8-byte name and a 56-byte miMATRIX header.
	totalBytes := 64 + 64*len(snames) + 32*totalRows

	// Write the global header.
	// TODO: If necessary, add an underscore to varname, but this will
	// make it 9 characters long.
	var header [128]byte
	pos := putUint32s(header[:],
		miMATRIX, uint32(totalBytes),
		miUINT32, 8, mxSTRUCT_CLASS, 0,
		miINT32, 8, 1, 1,
		miINT8, 8, 0, 0,
		miINT32+4<<16, 8,
		miINT8, uint32(8*len(snames)))
	copy(header[48:56], varname)
	if _, err := out.Write(header[:pos]); err != nil {
		return err
	}
	for _, name := range snames {
		for i := copy(header[:8], name); i < 8; i++ {
			header[i] = 0
		}
		if _, err := out.Write(header[:8]); err != nil {
			return err
		}
	}

	// Write the data for each satellite/frequency pair.
	for _, name := range snames {
		var key [4]byte
		copy(key[:], name)
		o := series[key]
		pos := putUint32s(header[:],
			miMATRIX, uint32(48+32*len(o)),
			miUINT32, 8, mxDOUBLE_CLASS, 0,
			miINT32, 8, uint32(len(o)), 4,
			miINT8, 0,
			miDOUBLE, uint32(len(o)*32))
		if _, err := out.Write(header[:pos]); err != nil {
			return err
		}

		// Write the time column.
		for i, v := range o {
			x := (i & 15) * 8
			putFloat64(header[x:x+8], float64(v.time))
			if x == 120 || i+1 == len(o) {
				if _, err := out.Write(header[:x+8]); err != nil {
					return err
				}
			}
		}

		// Repeat for SNR column.
		for i, v := range o {
			x := (i & 15) * 8
			putFloat64(header[x:x+8], float64(v.snr))
			if x == 120 || i+1 == len(o) {
				if _, err := out.Write(header[:x+8]); err != nil {
					return err
				}
			}
		}

		// Repeat for code-based pseudorange column.
		for i, v := range o {
			x := (i & 15) * 8
			putFloat64(header[x:x+8], v.code)
			if x == 120 || i+1 == len(o) {
				if _, err := out.Write(header[:x+8]); err != nil {
					return err
				}
			}
		}

		// Repeat for carrier phase column.
		for i, v := range o {
			x := (i & 15) * 8
			putFloat64(header[x:x+8], v.carrier)
			if x == 120 || i+1 == len(o) {
				if _, err := out.Write(header[:x+8]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func save(varname string, series map[[4]byte][]observation) error {
	// Write the header.
	bb := &bytes.Buffer{}
	var header [136]byte
	copy(header[:116], "MATLAB 5.0 MAT-file, created by rnx2mat")
	copy(header[124:], []byte{0, 1, 'I', 'M'})
	binary.LittleEndian.PutUint32(header[128:132], miCOMPRESSED)
	binary.LittleEndian.PutUint32(header[132:136], uint32(bb.Len()))
	if _, err := bb.Write(header[:]); err != nil {
		return err
	}

	// Write the compressed
	gzw, err := zlib.NewWriterLevel(bb, zlib.BestCompression)
	if err != nil {
		return nil
	}
	if err := saveMatrix(gzw, varname, series); err != nil {
		return err
	}
	if err := gzw.Close(); err != nil {
		return err
	}

	// Patch the header and create the file.
	s := bb.Bytes()
	binary.LittleEndian.PutUint32(s[132:136], uint32(bb.Len())-136)
	return ioutil.WriteFile(varname+".mat", s, 0666)
}

func main() {
	var series map[[4]byte][]observation

	suffix := regexp.MustCompile(`\.(rnx|\d\do)(\.gz)?$`)

	for _, fname := range os.Args[1:] {
		f, err := os.Open(fname)
		if err != nil {
			fmt.Println(err)
			continue
		}
		var r io.Reader = f

		if strings.HasSuffix(fname, ".gz") {
			if r, err = gzip.NewReader(r); err != nil {
				fmt.Println("Creating gzip reader: ", err)
				continue
			}
		}
		or := &rinex.ObsReader{}
		or.ObsFunc = func(rec rinex.ObservationRecord) error {
			if rec.EpochFlag > 1 {
				return nil
			}
			time := float32(rec.Hour)*3600 + float32(rec.Minute)*60 + rec.Second
			for _, sv := range rec.Sat {
				var key [4]byte
				copy(key[0:3], sv.PRN[:])
				obsCodes := or.Observations[sv.PRN[0]]
				if obsCodes == nil {
					obsCodes = or.Observations[' ']
				}
				for j, o := range sv.Obs {
					if o.Value == 0 {
						continue
					}
					obsCode := obsCodes[j]
					key[3] = obsCode[1]
					s := series[key]
					if len(s) == 0 || s[len(s)-1].time != time {
						s = append(s, observation{time: time})
					}
					idx := len(s) - 1
					switch obsCode[0] {
					case 'L':
						s[idx].carrier = o.Value
					case 'C':
						s[idx].code = o.Value
					case 'S':
						s[idx].snr = float32(o.Value)
					case 'D', 'P':
						continue
					default:
						panic("unexpected observation code " + string(obsCode[:]))
					}
					series[key] = s
				}
			}
			return nil
		}

		series = make(map[[4]byte][]observation, 256)
		if err = or.Parse(r); err != nil {
			fmt.Println(fname, ":", err)
			continue
		}

		varname := suffix.ReplaceAllString(fname, "")
		if idx := strings.LastIndexByte(varname, '/'); idx >= 0 {
			varname = varname[idx+1:]
		}
		if err = save(varname, series); err != nil {
			fmt.Println(err)
			continue
		}
	}
}

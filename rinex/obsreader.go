// Package rinex provides readers (and eventually writers) for the RINEX
// v2.11 and v3.04 file formats.  Notably, it does not attempt to parse
// all the defined header lines, but allows an application to receive
// and process each header line separately.
package rinex

import (
	"bufio"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

// Observation describes a single RINEX-style observation.
type Observation struct {
	// Value is the value of the observation, with three decimal digits
	// of fractional precision.  If the observation is not present in
	// the file, Value is 0.
	Value float64

	// LLI is the "loss of lock indicator", which is a three-bit mask.
	// This is the value of the field, not the ASCII code from the
	// input; blanks are mapped to 0.
	LLI byte

	// SignalStrength is a projection of signal strength onto the
	// interval 1-9.  RINEX 2 did not specify the exact mapping from
	// SNR or CNR to this value; RINEX 3 does.
	SignalStrength byte
}

// SVObservation combines the satellite identification with its actual
// observations for an epoch.
type SVObservation struct {
	// PRN identifies the satellite. PRN[0] identifies the GNSS. PRN[1]
	// and PRN[2] identify the satellite within the constellation.
	PRN [3]byte

	// Obs contains the observations from this satellite during the
	// current epoch.  This slice has the same length as the parent
	// ObsReader.Observations[PRN[0]], and is in the same order as that.
	Obs []Observation
}

// ObservationRecord represents one top-level data record in a GNSS
// observation data stream.  It includes a timestamp, an "epoch flag",
// and a variable number of satellite records.
type ObservationRecord struct {
	// Year is the year number of the observation record.
	// Two-digit years from RINEX are combined with the header field
	// "TIME OF FIRST OBS" to produce a full four-digit year.
	Year uint16

	// Month, Day, Hour and Minute indicate the time of the record in a
	// GNSS time scale.  (The RINEX header indicates which GNSS time
	// scale is used for this)
	Month, Day, Hour, Minute byte

	// EpochFlag is a value from 0 to 6 that indicate any unusual event
	// that occurred before this epoch.  0 means no unusual event; 1
	// means a power failure between the last epoch and this one; 6
	// means the observation records actually reflect cycle slips rather
	// than the nominal observation value; and 2 through 5 indicate
	// special events as described in the RINEX specification.
	EpochFlag byte

	// Second is the seconds-with-minute value for the measurement
	// epoch. It has up to seven decimal digits of significant fraction.
	Second float32

	// Offset is the receiver clock offset.  If this value was not
	// given in the input, Offset holds 0.
	Offset float64

	// Sat
	Sat []SVObservation
}

// ObsReader reads RINEX data that contain satellite observable values.
// In particular, it handles the RINEX 2.11 format that is associated
// with file extensions .yyo (where yy is a two-digit year number) and
// the RINEX 3.04 format that is associated with file extension .rnx.
type ObsReader struct {
	// HeaderFunc is a function that is called for each header line.
	// label starts at the 61st column, and is always 20 bytes long.
	// If HeaderFunc returns non-nil, parsing stops.
	HeaderFunc func(label, value string) error

	// ObsFunc is a function that is called for each observation record.
	// If it returns non-nil, parsing stops.
	ObsFunc func(rec ObservationRecord) error

	// Observations lists the types of observations for a given GNSS.
	// The map index is the first character of a satellite ID ('G' for
	// GPS, 'R' for GLONASS, 'S' for SBAS, 'E' for Galileo, etc., as
	// listed in RINEX 3.04, Figure 1.)  RINEX 2.11 data have the same
	// set of observables for all GNSSes; this uses a map index of ' '.
	//
	// The [3]byte values are two- or three-ASCII-character observation
	// types.  RINEX 2 uses two-character identifiers, with a NUL third
	// byte; RINEX 3 uses three-character identifiers.  (There is not a
	// one-to-one mapping between them, so this preserves the original
	// identifiers.)
	Observations map[byte][][3]byte

	// version is the RINEX version number for the stream.
	version int

	// year is, for RINEX 2.11 streams, the year of the most recent
	// observation that has been read.  It is initialized from the
	// TIME OF FIRST OBS header record.
	year uint16

	// count is the "Number of satellites" field from the first line of
	// an observation record.  For some epoch flags, this counts the
	// number of special records or header lines instead of satellites.
	count uint16

	// prnIndex is the PRN for which we are currently reading observations.
	prnIndex uint16

	// obsIndex is the next observation that we will read for prnIndex.
	obsIndex uint16

	// inHeader is true when we are parsing the RINEX header.
	inHeader bool

	// lastSystem is a scratch variable used for RINEX 3 continuation
	// lines for "SYS / # / OBS TYPE" header lines.
	lastSystem byte

	// obsRec holds the observation record that is currently being read.
	obsRec ObservationRecord

	// lineBuf holds the line currently being processed.
	lineBuf [80]byte
}

/************************ TOP LEVEL FUNCTIONS ************************/

// Parse reads RINEX data from r and runs the callback functions in or.
func (or *ObsReader) Parse(r io.Reader) error {
	or.inHeader = true
	or.version = 0
	or.lastSystem = 0
	or.Observations = make(map[byte][][3]byte)
	s := bufio.NewScanner(r)

	for s.Scan() {
		var line string
		if or.inHeader || or.version == 2 {
			// Space-pad the input to 80 characters.
			b := s.Bytes()
			if len(b) > 80 {
				return errors.New("Oversized input line")
			}
			for i := copy(or.lineBuf[:], b); i < 80; i++ {
				or.lineBuf[i] = ' '
			}
			line = string(or.lineBuf[:]) // yuck!
		} else {
			line = s.Text()
		}

		// Handle the line depending on our format.
		if or.inHeader {
			if err := or.handleHeader(line); err != nil {
				return err
			}
		} else if or.version == 2 {
			if err := or.parseV2(line); err != nil {
				return err
			}
		} else if or.version == 3 {
			if err := or.parseV3(line); err != nil {
				return err
			}
		} else {
			panic("RINEX header did not declare its version")
		}
	}

	return s.Err()
}

// handleHeader parse a RINEX 2.11 or 3.04 format header line.
func (or *ObsReader) handleHeader(line string) error {
	// Is this an embedded header for epoch/event flag 4?
	if or.count > 0 {
		or.count--
		if or.count == 0 {
			or.inHeader = false
		}
	}

	// Split the line into label and value.
	var err error
	value := line[:60]
	label := line[60:]

	// Is it one of the known labels that we treat specially?
	if handler := specialHeaders[label]; handler != nil {
		err = handler(or, value)
	}

	if err == nil && or.HeaderFunc != nil {
		err = or.HeaderFunc(label, value)
	}

	return err
}

/************************** HELPER FUNCTIONS **************************/

func parseUint(text string, bitSize int) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(text), 10, bitSize)
}

func parseFloat(text string, bitSize int) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(text), bitSize)
}

/************************* RINEX v2 FUNCTIONS *************************/

func (or *ObsReader) parseV2(line string) error {
	// parseV2 uses or.lastSystem as a state flag: 0 means the first
	// line of an EPOCH/SAT or EVENT FLAG, 1 means a PRN continuation
	// line, ' ' means between observations, and 'G', 'R', 'S', 'E'
	// mean mid-observation for that type.

	if or.lastSystem == 0 {
		if err := or.parseV2ObsIntro(line); err != nil {
			return err
		}

		switch or.obsRec.EpochFlag {
		case 0, 1, 6:
			// continue to parse PRNs
		default:
			return nil
		}
	}

	if or.lastSystem < 2 {
		return or.parseV2PRNs(line)
	}

	return or.parseV2Observations(line)
}

// parseV2Epoch parses the date/time stamp in an EPOCH/SAT or EVENT FLAG
// line from a RINEX 2.11 file.
func (or *ObsReader) parseV2Epoch(line string, flag byte) error {
	var year, month, day, hour, minute uint64
	var second float64

	if line[2] != ' ' {
		// Parse the fields.
		var err error
		if year, err = parseUint(line[1:3], 16); err != nil {
			return err
		}
		if month, err = parseUint(line[4:6], 8); err != nil {
			return err
		}
		if day, err = parseUint(line[7:9], 8); err != nil {
			return err
		}
		if hour, err = parseUint(line[10:12], 8); err != nil {
			return err
		}
		if minute, err = parseUint(line[13:15], 8); err != nil {
			return err
		}
		if second, err = parseFloat(line[15:26], 32); err != nil {
			return err
		}

		// Extend "year" to four digits.
		if year < uint64(or.year%100) {
			or.year += 100
		}
		or.year = or.year/100*100 + uint16(year)
		year = uint64(or.year)
	} else if flag == '0' || flag == '1' {
		return errors.New("RINEX 2 observation requires epoch: " + line)
	} // else no epoch, but none is needed

	or.obsRec.Year = uint16(year)
	or.obsRec.Month = byte(month)
	or.obsRec.Day = byte(day)
	or.obsRec.Hour = byte(hour)
	or.obsRec.Minute = byte(minute)
	or.obsRec.Second = float32(second)
	return nil
}

// parseV2ObsIntro parses an EPOCH/SAT or EVENT FLAG line.
func (or *ObsReader) parseV2ObsIntro(line string) error {
	// Start of observation record: EPOCH/SAT or EVENT FLAG line.
	var count uint64

	// Parse epoch flag and "number of satellites" field.
	flag := line[28]
	or.obsRec.EpochFlag = flag - '0'
	or.obsRec.Offset = 0
	or.obsRec.Sat = or.obsRec.Sat[:0]
	count, err := parseUint(line[29:32], 16)
	if err != nil {
		return err
	}
	or.count = uint16(count)

	// Are the epoch fields present?
	if err = or.parseV2Epoch(line, flag); err != nil {
		return err
	}

	// Is the epoch flag 2-5?
	if flag != '0' && flag != '1' && flag != '6' {
		or.inHeader = or.count > 0
		if or.ObsFunc != nil {
			return or.ObsFunc(or.obsRec)
		}
		return nil
	}

	// Parse the receiver time offset.
	if line[79] != ' ' {
		offset, err := parseFloat(line[68:80], 64)
		if err != nil {
			return err
		}
		or.obsRec.Offset = offset
	}

	// Get ready to read PRNs.
	if cap(or.obsRec.Sat) < int(count) {
		or.obsRec.Sat = make([]SVObservation, 0, count)
	}

	return nil
}

// parseV2PRNs parses the PRN list for a set of observations, either in
// the EPOCH/SAT line or in a continuation line.
func (or *ObsReader) parseV2PRNs(line string) error {
	// Either epoch flag 0, 1, 5, or a continuation line: PRNs.
	for i := 0; i < 12; i++ {
		prn := line[3*i+32 : 3*i+35]
		idx := len(or.obsRec.Sat)
		if prn[2] == ' ' {
			if idx < int(or.count) {
				return errors.New("PRN list terminated early")
			}
			break
		}
		or.obsRec.Sat = or.obsRec.Sat[:idx+1]
		copy(or.obsRec.Sat[idx].PRN[:], prn)
		if prn[0] == ' ' {
			or.obsRec.Sat[idx].PRN[0] = 'G'
		}
		or.obsRec.Sat[idx].Obs = or.obsRec.Sat[idx].Obs[:0]
	}

	// Are we at the end of the PRN list?
	if len(or.obsRec.Sat) == int(or.count) {
		or.lastSystem = 2
		or.prnIndex = 0
		or.obsIndex = 0
	} else {
		or.lastSystem = 1
	}

	return nil
}

// parseV2Observations parses an "OBSERVATIONS" line, either the first
// line for a PRN or a continuation record.
func (or *ObsReader) parseV2Observations(line string) error {
	// Read each observation on the current line.
	or.lastSystem = or.obsRec.Sat[or.prnIndex].PRN[0]
	nObs := len(or.Observations[' ']) - int(or.obsIndex)
	for i := 0; i < 5 && i < nObs; i++ {
		var err error
		entry := line[i*16 : (i+1)*16]

		// Parse Observation field.
		value := 0.0
		if entry[10] == '.' {
			if value, err = parseFloat(entry[0:14], 64); err != nil {
				return err
			}
		}

		// Translate LLI field.
		lli := byte(0)
		if entry[14] != ' ' {
			lli = entry[14] - '0'
		}

		// Translate Signal strength field.
		signalStrength := byte(0)
		if entry[15] != ' ' {
			signalStrength = entry[15] - '0'
		}

		// Save the values.
		s := or.obsRec.Sat[or.prnIndex].Obs
		for len(s) <= int(or.obsIndex)+i {
			s = append(s, Observation{})
		}
		s[int(or.obsIndex)+i] = Observation{
			Value:          value,
			LLI:            lli,
			SignalStrength: signalStrength,
		}
		or.obsRec.Sat[or.prnIndex].Obs = s
	}
	or.obsIndex += 5

	// Are we now at the end of the observations for this PRN?
	if int(or.obsIndex) >= len(or.Observations[' ']) {
		or.lastSystem = ' '
		or.obsIndex = 0
		or.prnIndex++

		if or.prnIndex == or.count {
			or.prnIndex = 0
			or.lastSystem = 0
			if or.ObsFunc != nil {
				return or.ObsFunc(or.obsRec)
			}
		}
	}

	return nil
}

/************************* RINEX v3 FUNCTIONS *************************/

func (or *ObsReader) parseV3(line string) error {
	if line[0] == '>' {
		return or.parseV3ObsIntro(line)
	}

	obslist, ok := or.Observations[line[0]]
	if !ok {
		return errors.New("Unexpected GNSS type: " + line)
	}
	idx := len(or.obsRec.Sat)
	or.obsRec.Sat = or.obsRec.Sat[:idx+1]
	svo := or.obsRec.Sat[idx]
	copy(svo.PRN[:], line[0:3])
	if cap(svo.Obs) < len(obslist) {
		svo.Obs = make([]Observation, 0, len(obslist))
	} else {
		svo.Obs = svo.Obs[:0]
	}
	for i := 0; i < len(obslist); i++ {
		obs := Observation{}

		if len(line) >= 17+16*i {
			v, err := parseFloat(line[3+16*i:17+16*i], 64)
			if err != nil {
				return err
			}
			obs.Value = v
		}

		if len(line) >= 18+16*i {
			obs.LLI = line[18+16*i] - '0'
		}

		if len(line) >= 19+16*i {
			obs.SignalStrength = line[19+16*i] - '0'
		}

		svo.Obs = append(svo.Obs, obs)
	}
	or.obsRec.Sat[idx] = svo
	return nil
}

func (or *ObsReader) parseV3Epoch(line string, flag byte) error {
	var year, month, day, hour, minute uint64
	var second float64

	if line[2] != ' ' {
		// Parse the fields.
		var err error
		if year, err = parseUint(line[2:6], 16); err != nil {
			return err
		}
		if month, err = parseUint(line[7:9], 8); err != nil {
			return err
		}
		if day, err = parseUint(line[10:12], 8); err != nil {
			return err
		}
		if hour, err = parseUint(line[13:15], 8); err != nil {
			return err
		}
		if minute, err = parseUint(line[16:18], 8); err != nil {
			return err
		}
		if second, err = parseFloat(line[19:30], 32); err != nil {
			return err
		}
	} else if flag == '0' || flag == '1' {
		return errors.New("RINEX 3 observation requires epoch: " + line)
	} // else no epoch, but none is needed

	or.obsRec.Year = uint16(year)
	or.obsRec.Month = byte(month)
	or.obsRec.Day = byte(day)
	or.obsRec.Hour = byte(hour)
	or.obsRec.Minute = byte(minute)
	or.obsRec.Second = float32(second)
	return nil
}

func (or *ObsReader) parseV3ObsIntro(line string) error {
	flag := line[28]
	or.obsRec.EpochFlag = flag - '0'
	or.obsRec.Offset = 0
	or.obsRec.Sat = or.obsRec.Sat[:0]
	count, err := parseUint(line[32:35], 16)
	if err != nil {
		return err
	}
	or.count = uint16(count)

	if err := or.parseV3Epoch(line, flag); err != nil {
		return err
	}

	// Does it declare a special event?
	if flag != '0' && flag != '1' && flag != '6' {
		or.inHeader = or.count > 0
		if or.ObsFunc != nil {
			return or.ObsFunc(or.obsRec)
		}
		return nil
	}

	// Parse the receiver time offset.
	if len(line) >= 56 {
		offset, err := parseFloat(line[41:56], 64)
		if err != nil {
			return err
		}
		or.obsRec.Offset = offset
	}

	// Make sure or.obsRec.Sat has enough capacity.
	if cap(or.obsRec.Sat) < int(or.count) {
		or.obsRec.Sat = make([]SVObservation, 0, int(or.count))
	}

	return nil
}

/********************** HEADER PARSING FUNCTIONS **********************/

// specialHeaders lists the headers that this file treats specially.
var specialHeaders = map[string]func(*ObsReader, string) error{
	"RINEX VERSION / TYPE": (*ObsReader).handleRINEXVersion,
	"END OF HEADER       ": (*ObsReader).handleEndOfHeader,
	"TIME OF FIRST OBS   ": (*ObsReader).handleTimeOfFirstObs,
	"# / TYPES OF OBSERV ": (*ObsReader).handleNumTypesOfObserv,
	"SYS / # / OBS TYPES ": (*ObsReader).handleSysNumObsTypes,
}

// handleRINEXVersion handles a RINEX VERSION / TYPE header.
func (or *ObsReader) handleRINEXVersion(value string) error {
	fltVersion, err := parseFloat(value[0:9], 32)
	if err != nil {
		return err
	}
	or.version = int(math.Round(fltVersion))
	if or.version != 2 && or.version != 3 {
		return errors.New("Invalid RINEX version " + value[0:9])
	}

	if value[20] != 'O' {
		return errors.New("Expected observation file, but got " + value[20:20])
	}

	return nil
}

// handleEndOfHeader handles a END OF HEADER header.
func (or *ObsReader) handleEndOfHeader(_ string) error {
	or.inHeader = false
	return nil
}

// handleTimeOfFirstObs handles a RINEX 2 TIME OF FIRST OBS header.
func (or *ObsReader) handleTimeOfFirstObs(value string) error {
	if or.version != 2 {
		return nil
	}

	year, err := parseUint(value[0:6], 16)
	if err != nil {
		return err
	}
	or.year = uint16(year)

	return nil
}

// handleNumTypesOfObserv handles a RINEX 2 # / TYPES OF OBSERV header.
func (or *ObsReader) handleNumTypesOfObserv(value string) error {
	if or.version != 2 {
		return nil
	}

	// Is this the first line?
	if len(or.Observations) == 0 {
		count, err := parseUint(value[0:6], 32)
		if err != nil {
			return err
		}

		or.Observations[' '] = make([][3]byte, 0, count)
	}

	// Read up to 9 observables from this line.
	s := or.Observations[' ']
	for i := 0; i < 9; i++ {
		s = append(s, [3]byte{value[10+6*i], value[11+6*i], 32})
		if len(s) == cap(s) {
			break
		}
	}
	or.Observations[' '] = s

	return nil
}

// handleSysNumObsTYpes handles a RINEX 3 SYS / # / OBS TYPES header.
func (or *ObsReader) handleSysNumObsTypes(value string) error {
	if or.version != 3 {
		return nil
	}
	var s [][3]byte

	// Is this the first line for the system?
	if or.lastSystem == 0 {
		or.lastSystem = value[0]
		count, err := parseUint(value[3:6], 32)
		if err != nil {
			return err
		}
		s = make([][3]byte, 0, count)
	} else {
		s = or.Observations[or.lastSystem]
	}

	// Read up to 13 observables from this line.
	for i := 0; i < 13 && len(s) < cap(s); i++ {
		idx := len(s)
		s = s[:idx+1]
		copy(s[idx][:], value[7+i*4:11+i*4])
	}

	or.Observations[or.lastSystem] = s
	if len(s) == cap(s) {
		or.lastSystem = 0
	}

	return nil
}

// Time converts the date and time in rec to a standard Go time.
func (rec ObservationRecord) Time() time.Time {
	fSec := float64(rec.Second)
	iSec := math.Floor(fSec)
	nSec := int(math.Round(1e9 * (fSec - iSec)))
	return time.Date(int(rec.Year), time.Month(rec.Month), int(rec.Day),
		int(rec.Hour), int(rec.Minute), int(iSec), nSec, time.UTC)
}

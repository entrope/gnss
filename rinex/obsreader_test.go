package rinex

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type expectation interface {
	checkHeader(label, value string) error
	checkObs(rec ObservationRecord) error
}

type checker struct {
	r []expectation
}

func TestParseV2(t *testing.T) {
	r := bytes.NewReader([]byte(
		`     2.11           OBSERVATION DATA    M (MIXED)           RINEX VERSION / TYPE
BLANK OR G = GPS,  R = GLONASS,  E = GALILEO,  M = MIXED    COMMENT
XXRINEXO V9.9       AIUB                24-MAR-01 14:43     PGM / RUN BY / DATE
EXAMPLE OF A MIXED RINEX FILE (NO FEATURES OF V 2.11)       COMMENT
A 9080                                                      MARKER NAME
9080.1.34                                                   MARKER NUMBER
BILL SMITH          ABC INSTITUTE                           OBSERVER / AGENCY
X1234A123           XX                  ZZZ                 REC # / TYPE / VERS
234                 YY                                      ANT # / TYPE
  4375274.       587466.      4589095.                      APPROX POSITION XYZ
         .9030         .0000         .0000                  ANTENNA: DELTA H/E/N
     1     1                                                WAVELENGTH FACT L1/2
     1     2     6   G14   G15   G16   G17   G18   G19      WAVELENGTH FACT L1/2
     0                                                      RCV CLOCK OFFS APPL
     5    P1    L1    L2    P2    L5                        # / TYPES OF OBSERV
    18.000                                                  INTERVAL
  2005     3    24    13    10   36.0000000                 TIME OF FIRST OBS
                                                            END OF HEADER
 05  3 24 13 10 36.0000000  0  4G12G09G06E11                         -.123456789
  23629347.915            .300 8         -.353    23629364.158
  20891534.648           -.120 9         -.358    20891541.292
  20607600.189           -.430 9          .394    20607605.848
                          .324 8                                          .178 7
 05  3 24 13 10 50.0000000  4  4
     1     2     2   G 9   G12                              WAVELENGTH FACT L1/2
  *** WAVELENGTH FACTOR CHANGED FOR 2 SATELLITES ***        COMMENT
      NOW 8 SATELLITES HAVE WL FACT 1 AND 2!                COMMENT
                                                            COMMENT
 05  3 24 13 10 54.0000000  0  6G12G09G06R21R22E11                   -.123456789
  23619095.450      -53875.632 8    -41981.375    23619112.008
  20886075.667      -28688.027 9    -22354.535    20886082.101
  20611072.689       18247.789 9     14219.770    20611078.410
  21345678.576       12345.567 5
  22123456.789       23456.789 5
                     65432.123 5                                     48861.586 7
 05  3 24 13 11  0.0000000  2  1
            *** FROM NOW ON KINEMATIC DATA! ***             COMMENT
 05  3 24 13 11 48.0000000  0  4G16G12G09G06                         -.123456789
  21110991.756       16119.980 7     12560.510    21110998.441
  23588424.398     -215050.557 6   -167571.734    23588439.570
  20869878.790     -113803.187 8    -88677.926    20869884.938
  20621643.727       73797.462 7     57505.177    20621649.276
                            3  4
A 9080                                                      MARKER NAME
9080.1.34                                                   MARKER NUMBER
         .9030         .0000         .0000                  ANTENNA: DELTA H/E/N
          --> THIS IS THE START OF A NEW SITE <--           COMMENT
 05  3 24 13 12  6.0000000  0  4G16G12G06G09                         -.123456987
  21112589.384       24515.877 6     19102.763 3  21112596.187
  23578228.338     -268624.234 7   -209317.284 4  23578244.398
  20625218.088       92581.207 7     72141.846 4  20625223.795
  20864539.693     -141858.836 8   -110539.435 5  20864545.943
 05  3 24 13 13  1.2345678  5  0
                            4  1
        (AN EVENT FLAG WITH SIGNIFICANT EPOCH)              COMMENT
 05  3 24 13 14 12.0000000  0  4G16G12G09G06                         -.123456012
  21124965.133       89551.30216     69779.62654  21124972.2754
  23507272.372     -212616.150 7   -165674.789 5  23507288.421
  20828010.354     -333820.093 6   -260119.395 5  20828017.129
  20650944.902      227775.130 7    177487.651 4  20650950.363
                            4  1
           *** ANTISPOOFING ON G 16 AND LOST LOCK           COMMENT
 05  3 24 13 14 12.0000000  6  2G16G09
                 123456789.0      -9876543.5
                         0.0            -0.5
                            4  2
           ---> CYCLE SLIPS THAT HAVE BEEN APPLIED TO       COMMENT
                THE OBSERVATIONS                            COMMENT
 05  3 24 13 14 48.0000000  0  4G16G12G09G06                         -.123456234
  21128884.159      110143.144 7     85825.18545  21128890.7764
  23487131.045     -318463.297 7   -248152.72824  23487146.149
  20817844.743     -387242.571 6   -301747.22925  20817851.322
  20658519.895      267583.67817    208507.26234  20658525.869
                            4  3
         ***   SATELLITE G 9   THIS EPOCH ON WLFACT 1 (L2)  COMMENT
         *** G 6 LOST LOCK AND THIS EPOCH ON WLFACT 2 (L2)  COMMENT
                (OPPOSITE TO PREVIOUS SETTINGS)             COMMENT`))

	c := checker{
		r: []expectation{
			expectHeader{
				label: "RINEX VERSION / TYPE",
				value: "     2.11           OBSERVATION DATA    M (MIXED)           ",
			},
			expectHeader{
				label: "COMMENT",
				value: "BLANK OR G = GPS,  R = GLONASS,  E = GALILEO,  M = MIXED    ",
			},
			expectHeader{label: "PGM / RUN BY / DATE"},
			expectHeader{label: "COMMENT"},
			expectHeader{label: "MARKER NAME"},
			expectHeader{label: "MARKER NUMBER"},
			expectHeader{label: "OBSERVER / AGENCY"},
			expectHeader{label: "REC # / TYPE / VERS"},
			expectHeader{label: "ANT # / TYPE"},
			expectHeader{label: "APPROX POSITION XYZ"},
			expectHeader{label: "ANTENNA: DELTA H/E/N"},
			expectHeader{label: "WAVELENGTH FACT L1/2"},
			expectHeader{label: "WAVELENGTH FACT L1/2"},
			expectHeader{label: "RCV CLOCK OFFS APPL"},
			expectHeader{label: "# / TYPES OF OBSERV"},
			expectHeader{label: "INTERVAL"},
			expectHeader{label: "TIME OF FIRST OBS"},
			expectHeader{label: "END OF HEADER"},
			expectObs{},
			expectObs{},
			expectHeader{label: "WAVELENGTH FACT L1/2"},
			expectHeader{label: "COMMENT"},
			expectHeader{label: "COMMENT"},
			expectHeader{label: "COMMENT"},
			expectObs{},
			expectObs{},
			expectHeader{label: "COMMENT"},
			expectObs{},
			expectObs{},
			expectHeader{label: "MARKER NAME"},
			expectHeader{label: "MARKER NUMBER"},
			expectHeader{label: "ANTENNA: DELTA H/E/N"},
			expectHeader{label: "COMMENT"},
			expectObs{},
			expectObs{},
			expectObs{},
			expectHeader{label: "COMMENT"},
			expectObs{},
			expectObs{},
			expectHeader{label: "COMMENT"},
			expectObs{},
			expectObs{},
			expectHeader{label: "COMMENT"},
			expectHeader{label: "COMMENT"},
			expectObs{},
			expectObs{},
			expectHeader{label: "COMMENT"},
			expectHeader{label: "COMMENT"},
			expectHeader{label: "COMMENT"},
		},
	}
	or := ObsReader{
		HeaderFunc: c.checkHeader,
		ObsFunc:    c.checkObs,
	}
	err := or.Parse(r)
	if err != nil {
		t.Error(err)
	}
}

func TestParseV3(t *testing.T) {
	r := bytes.NewReader([]byte(
		`     3.02           OBSERVATION DATA    M                   RINEX VERSION / TYPE
ssrcrin-10.1.1x                         20190110 000000 LCL PGM / RUN BY / DATE 
(0930225631113) Septentrio specific, please ignore.         COMMENT             
TWTF                                                        MARKER NAME         
Septentrio                                                  MARKER NUMBER       
PolaRx4Pro                                                  MARKER TYPE         
Pseudonym Doe       TL                                      OBSERVER / AGENCY   
3008040             SEPT POLARX4        2.9.0               REC # / TYPE / VERS 
CR620012101         ASH701945C_M    SCIS                    ANT # / TYPE        
 -2994427.6478  4951307.5755  2674496.0997                  APPROX POSITION XYZ 
        0.0000        0.0000        0.0000                  ANTENNA: DELTA H/E/N
G   18 C1C L1C D1C S1C C1W S1W C2W L2W D2W S2W C2L L2L D2L  SYS / # / OBS TYPES 
       S2L C5Q L5Q D5Q S5Q                                  SYS / # / OBS TYPES 
E   16 C1C L1C D1C S1C C5Q L5Q D5Q S5Q C7Q L7Q D7Q S7Q C8Q  SYS / # / OBS TYPES 
       L8Q D8Q S8Q                                          SYS / # / OBS TYPES 
S    8 C1C L1C D1C S1C C5I L5I D5I S5I                      SYS / # / OBS TYPES 
R   16 C1C L1C D1C S1C C2P L2P D2P S2P C2C L2C D2C S2C C3Q  SYS / # / OBS TYPES 
       L3Q D3Q S3Q                                          SYS / # / OBS TYPES 
C    8 C1I L1I D1I S1I C7I L7I D7I S7I                      SYS / # / OBS TYPES 
J   12 C1C L1C D1C S1C C2L L2L D2L S2L C5Q L5Q D5Q S5Q      SYS / # / OBS TYPES 
SEPTENTRIO RECEIVERS OUTPUT ALIGNED CARRIER PHASES.         COMMENT             
NO FURTHER PHASE SHIFT APPLIED IN THE RINEX ENCODER.        COMMENT             
G 1C                                                        SYS / PHASE SHIFT   
G 2W                                                        SYS / PHASE SHIFT   
G 2L   0.00000                                              SYS / PHASE SHIFT   
G 5Q   0.00000                                              SYS / PHASE SHIFT   
E 1C   0.00000                                              SYS / PHASE SHIFT   
E 5Q   0.00000                                              SYS / PHASE SHIFT   
E 7Q   0.00000                                              SYS / PHASE SHIFT   
E 8Q   0.00000                                              SYS / PHASE SHIFT   
S 1C                                                        SYS / PHASE SHIFT   
S 5I                                                        SYS / PHASE SHIFT   
R 1C                                                        SYS / PHASE SHIFT   
R 2P   0.00000                                              SYS / PHASE SHIFT   
R 2C                                                        SYS / PHASE SHIFT   
R 3Q   0.00000                                              SYS / PHASE SHIFT   
C 1I                                                        SYS / PHASE SHIFT   
C 7I                                                        SYS / PHASE SHIFT   
J 1C                                                        SYS / PHASE SHIFT   
J 2L   0.00000                                              SYS / PHASE SHIFT   
J 5Q   0.00000                                              SYS / PHASE SHIFT   
  2019     1    10     0     0    0.0000000     GPS         TIME OF FIRST OBS   
 C1C    0.000 C2C    0.000 C2P    0.000                     GLONASS COD/PHS/BIS 
DBHZ                                                        SIGNAL STRENGTH UNIT
                                                            END OF HEADER       
> 2019 01 10 00 00  0.0000000  0 25
S22  36968522.053 7 194271247.78607       -39.024 7        43.970
S37  36925330.673 6 194043956.65206         4.286 6        40.205
R20  21470076.145 6 114810173.20606      3090.169 6        36.450    21470085.162 7  89296810.30607      2403.428 7        43.482    21470085.883 7  89296817.31207      2403.475 7        42.724
G24  23660058.191 5 124334441.97205      3536.009 5        35.948    23660058.183 3        22.609    23660062.087 3  96883999.47003      2755.327 3        22.609    23660061.636 6  96884000.48606      2755.368 6        39.024
R05  21053486.921 7 112542934.51107       518.143 7        43.007    21053494.583 7  87533419.24207       403.061 7        43.788    21053495.148 7  87533425.25907       402.968 7        43.469
S28  37768235.394 7 198474690.61107         0.286 7        43.412
G29  21745189.830 7 114271733.77807      -625.538 7        47.307    21745189.759 5        35.257    21745189.555 5  89042902.21605      -487.428 5        35.257    21745189.917 6  89042904.21006      -487.383 6        41.108
S32  37120920.343 6 195071749.62306         1.694 6        38.541
G21  25068559.482 5 131736161.07005      3023.889 5        33.245    25068558.628 2        15.190    25068558.065 2 102651550.01902      2356.263 2        15.190
R18  21837707.116 6 116571123.61506     -4111.833 6        41.023    21837715.505 7  90666468.54207     -3198.123 7        43.264    21837715.794 7  90666472.53407     -3198.009 7        42.919
S26  39258577.813 5 206304606.82705       -43.680 5        34.733
J02  37022437.538 6 194554238.77606       284.251 6        40.687    37022440.522 6 151600740.00006       221.410 6        41.449
G05  21557855.617 8 113287291.64908      -936.262 8        48.876    21557855.670 6        39.396    21557855.238 6  88275809.25406      -729.553 6        39.396    21557855.214 7  88275811.25007      -729.607 7        43.075
S29  36925306.276 6 194043821.67406         4.452 6        39.069
G02  21368724.778 7 112293405.61407     -1893.303 7        46.782    21368724.121 6        38.420    21368722.575 6  87501332.34006     -1475.296 6        38.420
S40  37094099.434 7 194931031.85707        -9.025 7        45.035
R04  20716420.381 7 110935479.03307     -1576.407 7        42.663    20716426.919 7  86283173.89207     -1226.096 7        44.579    20716427.020 7  86283171.89907     -1226.190 7        44.126
S30-262832343.036 7-381193270.63107        60.308 7        46.056
R19  19302375.961 6 103254681.43606     -1155.669 6        40.784    19302382.279 7  80309222.10807      -898.879 7        45.609    19302382.910 7  80309229.10807      -898.926 7        45.169
J03  36170763.840 6 190078651.44206      -234.439 6        41.800    36170766.211 7 148113245.64007      -182.668 7        42.427
G15  20565309.840 8 108071421.78108      1175.407 8        50.805    20565309.889 7        42.476    20565309.469 7  84211487.59607       915.903 7        42.476    20565309.613 7  84211485.59107       915.948 7        45.279
G13  20204184.687 8 106173705.70908      -996.799 8        48.676    20204184.459 7        43.185    20204184.090 7  82732742.84807      -776.727 7        43.185
J01  39393044.929 7 207011852.97307        23.584 7        45.907    39393045.278 7 161307966.81407        18.388 7        44.650
R14  22698361.677 6 120995042.36406      2057.955 6        40.217    22698370.128 7  94107307.11907      1600.686 7        42.002    22698370.731 6  94107298.10406      1600.647 6        41.402
G30  24083967.488 5 126562091.90505     -1573.881 5        35.905    24083967.037 3        20.762    24083970.512 3  98619811.62103     -1226.399 3        20.762    24083970.019 6  98619814.63006     -1226.374 6        36.934`))

	c := checker{
		r: []expectation{
			expectHeader{
				label: "RINEX VERSION / TYPE",
				value: "     3.02           OBSERVATION DATA    M                   ",
			},
			expectHeader{label: "PGM / RUN BY / DATE"},
			expectHeader{label: "COMMENT"},
			expectHeader{label: "MARKER NAME"},
			expectHeader{label: "MARKER NUMBER"},
			expectHeader{label: "MARKER TYPE"},
			expectHeader{label: "OBSERVER / AGENCY"},
			expectHeader{label: "REC # / TYPE / VERS"},
			expectHeader{label: "ANT # / TYPE"},
			expectHeader{label: "APPROX POSITION XYZ"},
			expectHeader{label: "ANTENNA: DELTA H/E/N"},
			expectHeader{label: "SYS / # / OBS TYPES"},
			expectHeader{label: "SYS / # / OBS TYPES"},
			expectHeader{label: "SYS / # / OBS TYPES"},
			expectHeader{label: "SYS / # / OBS TYPES"},
			expectHeader{label: "SYS / # / OBS TYPES"},
			expectHeader{label: "SYS / # / OBS TYPES"},
			expectHeader{label: "SYS / # / OBS TYPES"},
			expectHeader{label: "SYS / # / OBS TYPES"},
			expectHeader{label: "SYS / # / OBS TYPES"},
			expectHeader{label: "COMMENT"},
			expectHeader{label: "COMMENT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "SYS / PHASE SHIFT"},
			expectHeader{label: "TIME OF FIRST OBS"},
			expectHeader{label: "GLONASS COD/PHS/BIS"},
			expectHeader{label: "SIGNAL STRENGTH UNIT"},
			expectHeader{label: "END OF HEADER"},
			expectObs{},
		},
	}
	or := ObsReader{
		HeaderFunc: c.checkHeader,
		ObsFunc:    c.checkObs,
	}
	err := or.Parse(r)
	if err != nil {
		t.Error(err)
	}
}

/********************* CONCRETE EXPECTATION TYPES *********************/

type expectHeader struct {
	label, value string
}

func (x expectHeader) checkHeader(label, value string) error {
	if x.label != "" && x.label != strings.TrimSpace(label) {
		return fmt.Errorf("Header label mismatch: expected %s, got %s", x.label, label)
	}
	if x.value != "" && x.value != value {
		return fmt.Errorf("Header value mismatch: expected %s, got %s", x.value, value)
	}
	return nil
}

func (x expectHeader) checkObs(rec ObservationRecord) error {
	return errors.New("Expected header line, but got observation")
}

type expectObs struct {
}

func (x expectObs) checkHeader(label, value string) error {
	return fmt.Errorf("Expected observation, but got header (%s%s)", value, label)
}

func (x expectObs) checkObs(rec ObservationRecord) error {
	return nil
}

func (c *checker) checkHeader(label, value string) error {
	if len(c.r) == 0 {
		return errors.New("Got unexpected header line")
	}
	expect := c.r[0]
	c.r = c.r[1:]
	return expect.checkHeader(label, value)
}

func (c *checker) checkObs(rec ObservationRecord) error {
	if len(c.r) == 0 {
		return errors.New("Got unexpected observation record")
	}
	expect := c.r[0]
	c.r = c.r[1:]
	return expect.checkObs(rec)
}

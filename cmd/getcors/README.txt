
Welcome to the National Oceanic and Atmospheric Administration's (NOAA)
Continuously Operating Reference Station (CORS) Network  

                  http://geodesy.noaa.gov/CORS
                  http://alt.ngs.noaa.gov/CORS

Last Updated: 18 June 2015

CONTENTS OF THIS FILE
======================

INTRODUCTION
ACCESS TO DATA
DIRECTORY STRUCTURE OF FTP SITE
BRIEF DESCRIPTION OF MAIN DIRECTORIES
DETAILED DESCRIPTION OF THE CONTENTS OF THE DIRECTORY coord
DETAILED DESCRIPTION OF THE CONTENTS OF THE DIRECTORY rinex
FILE RETENTION POLICY
CONTACT INFORMATION

INTRODUCTION
============

NOAA's National Geodetic Survey (NGS) operates the Continuously Operating
Reference Station (CORS) network that consists of group of Global Navigation 
Satellite System (GNSS) reference stations which provide code range and carrier 
phase data to users in support of postprocessing applications. The stations are 
owned and operated by federal, state, local agencies, private companies, and 
university groups, and NGS redistributes their data to the public free of charge.

The GNSS data collected at these stations are made available to the public by NGS 
in Receiver INdependent EXchange (RINEX) format. The data include observation, 
meteorological, navigation/ephemeris, station logs and NGS coordinate files for 
the stations. Most data are available within 1 hour from when they were recorded 
at the remote site, and a few sites have a delay of 24 hours.

All data since 9 February (040) 1994 are available.

ACCESS TO DATA
=================
 Data can be retrieved via anonymous ftp from 
       geodesy.noaa.gov  or alt.ngs.noaa.gov

or 
       ftp://geodesy.noaa.gov/cors

or 
       ftp://alt.ngs.noaa.gov/cors

Alternatively a customized data request service is available at:
       http://geodesy.noaa.gov/UFCORS


DIRECTORY STRUCTURE OF FTP SITE
=================================

All characters within {} refer to variable names. For an explanation of these
  please see next section EXPLANATION OF DIRECTORY CONTENTS 
              |
              |
              |--coord--|--coord_{yy}--|--{ssss} (individual sites) 
              |                        |--nad_geo.txt
              |                        |--nad_xyz.txt
              |                        |--itrf_geo.txt
              |                        |--itrf_xyz.txt
              |                        |
              |                        |--Old--|{ssss}.{yy}{Mmm}{dd}
              |                        |
              |                        | for coord_08 file names have changed 
              |                        |--{ssss}_08.coord.txt (individual sites)
              |                        |
              |                        |  composite coordinate files
              |                        |--igs08_geo.comp.txt
              |                        |--igs08_xyz.comp.txt
              |                        |--igs08_geo.htdp.txt
              |                        |--igs08_xyz.htdp.txt
              |                        |--nad83_2011_geo.comp.txt
              |                        |--nad83_2011_xyz.comp.txt
              |                        |--nad83_2011_geo.htdp.txt
              |                        |--nad83_2011_xyz.htdp.txt
              |                        |--nad83_MA11_geo.comp.txt
              |                        |--nad83_MA11_xyz.comp.txt
              |                        |--nad83_MA11_geo.htdp.txt
              |                        |--nad83_MA11_xyz.htdp.txt
              |                        |--nad83_PA11_geo.comp.txt
              |                        |--nad83_PA11_xyz.comp.txt
              |                        |--nad83_PA11_geo.htdp.txt
              |                        |--nad83_PA11_xyz.htdp.txt
              |                        
              |
              |--Plots--|--{ssss}_08.short.png (short-term plots of daily positions)
              |         |
              |         |--Longterm--|--{ssss}_08.long.png (long-term plots of
              |         |                                    daily positions)
              |
              |        
              |-- README.txt  (This file)
              |-- RINEX-211.txt (RINEX 2.11 definition file)
              | 
              |          |          |         |          |--{ssssdddh}.{yyt}.Z
              |          |--{yyyy}--|--{ddd}--|--{ssss}--|--{ssssdddh}.{yyt}.gz
              |--rinex --|          |-{mmmdd}-|          |--{ssssdddh}.{yy}S
              |          |          |         |          |--{ssssdddh}.{yy}m.Z
      --cors--|                               | 
              |                               |   Orbit files
              |                               |--brdc{ddd}0.{yy}n.gz
              |                               |--brdc{ddd}0.{yy}g.gz
              |                               |--igl{wwwwd}.{yy}.sp3.gz
              |                               |--igs{wwwwd}.{yy}.sp3.gz
              |                               |--igr{wwwwd}.{yy}.sp3.gz
              |                               |--igu{wwwwd}_{hh}.{yy}.sp3.gz
              |                               |
              |                               |
              |                               |--sum_gz--|--{ssssdddh}.{yy}S
              |                               |
              |
              |             |
              |--spec_prod--|--Various directories to support special products
              |             |
              |
              |--station_log--|--{ssss}.log.txt
              |               |--cumulative.station.info.cors
              |               |
              |
              |--tst_rnx--    test subdirectories (not for general use)
              |


BRIEF DESCRIPTION OF MAIN DIRECTORIES
==================================

  coord          NAD 83, IGS08, and ITRF coordinates and velocities for each 
                   CORS site (see next section for a description of the files
                   and their formats) 
 
  Plots          Short-term timeseries plots of CORS sites. These files
                   show the difference between the calculated daily position
                   and the published coordinate listed in the coord file for the 
                   previous few months.
                 Long-term timeseries plots derived from our "multiyear"/MYCS1 
                   project. Last updated April 2012. 

  rinex          RINEX data files, orbit files 
                   (see next section for a description of the files and their 
                   formats) 

  station_log    Station information, including current and historical 
                   equipment used at a site. 
                   File format {ssss}.log.txt
                   Where ssss - 4-character site id

  spec_prod      Special products not for operational use 

  tst_rnx        Test data sets not for operational use

DETAILED DESCRIPTION OF THE CONTENTS OF THE DIRECTORY coord
========================================= 

For detailed explanation of the files contained in this directory
 and their formats see:
http://geodesy.noaa.gov/CORS/coords.shtml

in /cors/coord the following directories and files are available

  Directories:

  coord_{yy}  where yy is a 2-digit year that corresponds to a particular
    Global reference frame. Currently NGS supports IGS08 which is located
    in coord_08, prior to this NGS supported global frames in the International 
    Terrestrial Reference Frame (ITRF) e.g. in coord_00 any 
    ITRF coordinates are in ITRF2000. 
   
  Within each directory up to 6 types of files are present:

  With the introduction of IGS08 directory file naming changed:
  In the directory coord08
  {ssss}_08.coord.txt - is the coordinate file that contains the 
           current IGS08 and NAD 83(2011,MA11,PA11) xyz and 
           geographic coordinates for a particular site
  Old/{ssss}.{yy}{Mmm}{dd} - are older coordinate file that applies to the time
                           prior to the specified data ({yy}{Mmm}{dd})

  ssss  - 4-character id
  yy    - 2-digit year
  Mmm   - 3-letter month
  dd    - 2-digit day of month

The following files contain composite listings of all coordinates for CORS including
 decommissioned sites.

  igs08_geo_comp.txt - coordinates are computed IGS08 geographic coordinates
  igs08_geo_htdp.txt - coordinates are modeled IGS08 geographic coordinates
  igs08_xyz.comp.txt - coordinates are computed IGS08 xyz coordinates
  igs08_xyz.htdp.txt - coordinates are modeled IGS08 xyz coordinates
  nad83_2011_geo.comp.txt - coordinates are NAD_83(2011) computed geographic coordinates
  nad83_2011_geo.htdp.txt - coordinates are NAD_83(2011) modeled geographic coordinates
  nad83_2011_xyz.comp.txt - coordinates are NAD_83(2011) computed xyz coordinates
  nad83_2011_xyz.htdp.txt - coordinates are NAD_83(2011) modeled xyz coordinates
  nad83_MA11_geo.comp.txt - coordinates are NAD_83(MA11) computed geographic coordinates
  nad83_MA11_geo.htdp.txt - coordinates are NAD_83(MA11) modeled geographic coordinates
  nad83_MA11_xyz.comp.txt - coordinates are NAD_83(MA11) computed xyz coordinates
  nad83_MA11_xyz.htdp.txt - coordinates are NAD_83(MA11) modeled xyz coordinates
  nad83_PA11_geo.comp.txt - coordinates are NAD_83(PA11) computed geographic coordinates
  nad83_PA11_geo.htdp.txt - coordinates are NAD_83(PA11) modeled geographic coordinates
  nad83_PA11_xyz.comp.txt - coordinates are NAD_83(PA11) computed xyz coordinates
  nad83_PA11_xyz.htdp.txt - coordinates are NAD_83(MA11) modeled xyz coordinates

  In the directories coord94, coord96, coord97, coord00 the following files 
  may be found:
  {ssss} - is the coordinate file that contains the current ITRF and NAD3 
           xyz and geographic coordinates and velocities for a particular
           site
  Old/{ssss}.{yy}{Mmm}{dd} - are older coordinate file that applies to the time 
                           prior to the specified data ({yy}{Mmm}{dd})

  ssss  - 4-character id
  yy    - 2-digit year
  Mmm   - 3-letter month
  dd    - 2-digit day of month
  
The following files are a composite listing of all CORS including
 decommissioned sites.

  itrf_geo.txt - coordinates are ITRF geographic coordinates 
  itrf_xyz.txt - coordinates are ITRF xyz coordinates
  nad_geo.txt  - coordinates are NAD83 geographic coordinates   
  nad_xyz.txt  - coordinates are NAD83 xyz coordinates

DETAILED DESCRIPTION OF THE CONTENTS OF THE DIRECTORY rinex
========================================= 

in /cors/rinex/yyyy/ddd the following orbit/ephemeris files are available
  this is the same as:
  /cors/rinex/yyyy/mmmdd

Global broadcast orbit:
brdc{ddd}0.{yy}n.gz - GPS global broadcast/navigation/ephemeris file gzip compressed 
brdc{ddd}0.{yy}g.gz - GLONASS global broadcast/navigation/ephemeris file gzip compressed 
                       ddd - is 3-digit day-of-year
                       yy  - is 2-digit year
                        These files are built cumulatively every hour till the end 
                         of the day

High-precision orbits published by IGS:
For detailed information on IGS orbit products see
http://igs.org/products
               
               wwww - is the 4-digit GPS week
               d    - is the 1-digit GPS day 0-Sunday,...6-Saturday
               hh   - is the 2-digit hour 00,06,12,18

IGS ultrarapid orbits are produced every 6 hours
igu{wwww}{d}_{hh}.sp3.gz  - IGS ultrarapid GPS orbits in sp3 format gzip compressed

IGS rapid orbits are produced with a latency of approximately 18 hours
igr{wwww}{d}.sp3.gz  - IGS rapid GPS orbits in sp3c format gzip compressed

IGS precise/final orbits are produced with a 14-16 day latency
igs{wwww}{d}.sp3.gz  - IGS final GPS orbits in sp3c format gzip compressed
igl{wwww}{d}.sp3.gz  - IGS final GLONASS orbits in sp3c format gzip compressed
 
 Please note: 
  Prior to 2007-357 only IGS final orbits are available 
  All orbits between 1994.001 to 2007-357 are from the IGS reprocessing solution
  and are aligned with IGS05
  All orbits between 2007-357 to 2011-106 are aligned with IGS05
  All orbits after 2011-107 or 17 April 2011 are aligned to IGS08

Notice effective 3 February 2014
 2007-250-present can contain:
   GPS L1+L2+L2C+L5 and GLONASS L1+L2
 Prior to 2007-250 contain only 
   GPS L1+L2
    
in /cors/rinex/yyyy/hold_sum 
    1 or 2 directories are present
    ddz - always present
    dgz - only present for the current year
      In each of these directories is a single file for each day that contains a modified
      SUM line from the teqc summary files (see next section). Each file has the following name
      summary.{yyyy}.{ddd}.fileinfo
         yyyy - is a 4-digit year
         ddd  - is a 3-digit day-of-year
    Each file contains the following columns
 obs typ SSSS  first epoch    last epoch    St-En dt   #expt     #exp    #actu  e/a e/a      multi    Obs per  Tot  Group/           File
              yyyy-ddd hh:mm yyyy-ddd hh:mm  hrs  sec  24obs     obs      obs   24%   %    mp1   mp2    slip  Slps  Agency        Time Stamp

  obs         - Observable type: gp - GPS only; gn - GPS + GLONASS
  typ         - Type of compresssion: dz - daily Hatanaka and UNIX compressed; gz - gzip compressed
  SSSS        - 4-alphanumeric site identifier
  first epoch - Date and time of first epoch of data contained in the file 
  last epoch  - Date and time of last epoch of data contained in the file
  St-En       - Number of hours of data contained in the file excludes data gaps
  dt          - Sampling rate of file in seconds
  #expt 24obs - Expected number of observations that could be collected with a cutoff angle of 5 degrees over a 24hr period 
  #act obs    - Actual number of observations that were collected with a cutoff angle of 5 degrees 
  e/a 24%     - Percentage of actual number of observations compared to expected number of observation (see two previous columns)
  e/a %       - Percentage of actual number of observations compared to expected number of observation given 
                    actual start and end time, but no data gaps
  multi        - Pseudorange MP1 and MP2 multipath
  Obs per slip - Number of observation collected before a cycle slip is detected
  Tot Slps     - Total number of cycle slips detected in the file
  Group/Agency - NGS internal 1-6 alphanumeric code representing Group/Agency contributing data for the site. 
  File Time Stamp - Time stamp in UTC when the teqc summary file (see next section) was created

 in /cors/rinex/{yyyy}/{ddd}/{ssss} 1 or more of the following data files are available
   this is the same as:
   /cors/rinex/{yyyy}/{mmmdd}/{ssss}

{ssss}{ddd}{h}.{yy}{t}.{C}

       ssss - is the 4-character site identifier
       ddd  - is a 3-digit day-of-year
       h    - for 1 hour long (60 minute duration) files this is a letter 
               a through x that corresponds to the start hour as shown below
         00 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 18 19 20 21 22 23
          a  b  c  d  e  f  g  h  i  j  k  l  m  n  o  p  q  r  s  t  u  v  w  x

              for 24 hour (daily) long files the number 0 (zero) is used

       yy   - is a 2-digit year
       t    - is the type of file. 6 types exist
                d - is a RINEX observation file that has been decimated to a
                    sampling rate of 30 seconds and that is Hatanaka compressed
                    Hatanaka uncompress software is available at:
                        http://terras.gsi.go.jp/ja/crx2rnx.html    
                g - is a RINEX GLONASS navigation data file
                m - is a RINEX meterological data file
                n - is a RINEX GPS navigation data file
                o - is a RINEX observation file with the receiver sampling rate
                      of 1, 5, 15, 30 seconds, after 30 days this file is 
                      decimated to 30 seconds
                S - is a summary file of the Hatanaka compressed file (type d.Z)
                      created using the TEQC software

       C    - is compression type 
               gz - gzip compression format
               Z  - UNIX compression format

in /cors/rinex/{yyyy}/{ddd}/sum_gz the following type of files are available
   this is the same as:
   /cors/rinex/{yyyy}/{mmmdd}/{sum_gz}

{ssss}{ddd}{h}.{yy}S

    This is a summary file of the file (type o.gz) created using the TEQC
      software


FILE RETENTION POLICY
======================

RINEX observation, meteorological, navigation and TEQC summary files:
Hourly files are only kept for 2 days
Daily files are kept permanently 
However after 30 days the 24hr (daily) RINEX observation files that are gzipped
 ({ssss}{ddd}0.{yy}o.gz) are decimated to a 30 second sampling rate equivalent
  to the {ssss}{ddd}0.{yy}d.Z
site specific GPS navigation files are only kept for sites that NGS submits to 
the IGS network 
Meteorological files are available only for sites with independent meteorological 
sensors

After six-months to one year the second copy of the RINEX observation files that are gzipped
is removed i.e. file name {ssss}{ddd}0.{yy}o.gz

All orbit files are kept permanently

Coord files are updated as needed

Logs files are updated as needed

Plot files are updated daily 


CONTACT INFORMATION
==================================

If you have questions about this file or about CORS in general please check
  http://www.ngs.noaa.gov/CORS
  http://alt.ngs.noaa.gov/CORS
or email ngs.cors @ noaa.gov

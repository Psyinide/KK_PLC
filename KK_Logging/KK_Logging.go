package KK_Logging

import (
    "io"
    "io/ioutil"
    "log"
    "os"
    "fmt"
    "strings"
    "time"
    "errors"
    "path/filepath"
)

/*
 *
 *	Logging module for my project
 * 
 *	to use, include then call Start( loggingLevel ) ie:
 *	KK_Logging.Start( "warning" )
 *	then to log something call:
 *	KK_Logging.QueueLogWrite( txtToLog, severityLevel )
 *
 *	Log writes are thread safe, writes are queued in a channel
 */

// these should be configurable, or args to Start()
type loggerConfig struct {
	logDirectory string
	logPrefix string
	logSuffix string
	timestampFormat string
	daysToKeep int
}

var lc loggerConfig


// logger pointers
var (
    Trace   *log.Logger
    Info    *log.Logger
    Event    *log.Logger
    Warning *log.Logger
    Error   *log.Logger
)

var loggingLevel string
var logFileOpened bool
var logFileDate string
var logWriteQueue chan logStruct
type logStruct struct{
	str string
	severity string
}



// must call Start() first to create loggers
// Start() returns, it doesn't need to be called
// as a Go Routine
func Start( logLevel string ) {


	lc.logDirectory = "./logs"
	lc.logPrefix = "log"
	lc.logSuffix = "txt"
	lc.timestampFormat = "2006-01-02"
	lc.daysToKeep = 7

	// validate and set the logging level
	ChangeLogLevel(logLevel)

	// run purge on init
	if lc.daysToKeep > 0 {
		kk := purgeOldLogs()
		if kk != nil {
			fmt.Println( kk.Error() )
		}

		// also run the purge every 8 hours
	    purgeTimer := time.NewTimer( time.Hour * 8 )
	    go func() {
	        <- purgeTimer.C
			purge := purgeOldLogs()
			if purge != nil {
				fmt.Println( purge.Error() )
			}
	    }()
	}

	// track if the log file was able to be opened for writting
	logFileOpened = false

	// create the log write queue for threading friendly log writting
	logWriteQueue = make(chan logStruct, 100)

	// open log file, rotate if date flips
	err := getLogFile()
	if err != nil{
		fmt.Fprintln(os.Stderr, err.Error())
	}
	    
    // start the processor
    go processLogQueue()
}


func ChangeLogLevel( newLogLevel string ){
	
	// validate and set the logging level
	newLogLevel = strings.ToLower(newLogLevel)
	switch newLogLevel {
	case "info", "event", "warning", "error":
		loggingLevel = newLogLevel
	default:
		loggingLevel = "warning"
	}
}


//
// Public function to write to the log file
// this is a Variadic function, so here is the function defination:
// QueueLogWrite(txt as string, optional severity as sting = "info" )
//
//
//func QueueLogWrite( txt, severity string ){
func QueueLogWrite( args ...string ){

	// validate args
	if len(args) < 1 {return}

	// populate local vars from args
	txt := args[0]
	severity := "info"
	if len(args) > 1 { severity = args[1] }

	// build the log stuct
	var logObj logStruct
	logObj.str = txt
	logObj.severity = severity

	// queue the log write
	logWriteQueue <- logObj
}

/*
 *	All internal functions below
 */


// internal function to create the loggers
func initLogger( infoHandle, warningHandle, errorHandle io.Writer) {

	//flags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
	flags := log.Ldate | log.Ltime | log.Lmicroseconds

    Info = log.New(infoHandle, "INFO: ",flags)
    Event = log.New(infoHandle, "EVENT: ",flags)
    Warning = log.New(warningHandle, "WARNING: ",flags)
    Error = log.New(errorHandle, "ERROR: ",flags)
}


// internal function to process the log write queue
// forwards log writes to the correct logger
// default logger is logInfo()
func processLogQueue(){
	for {
		queuedLogObj := <-logWriteQueue
		queuedLogObj.severity = strings.ToLower(queuedLogObj.severity)


		// open log file, rotate if date flips
		err := getLogFile()
		if err != nil{
			fmt.Fprintln(os.Stderr, err.Error())
			continue			
		}
		

		// don't try writting to the log file if it's not opened
		if ( !logFileOpened ){
			fmt.Fprintln(os.Stderr, "Cannot write to log file, log file not opened")
			continue
		}

		// handle severity/loglevel checking
		if ( queuedLogObj.severity == "info" ){
		
			// only log info severity if we are at the lowest logging level
			if ( loggingLevel == "info") {Info.Println(queuedLogObj.str + "\r")}

		}else if ( queuedLogObj.severity == "error" ){

			// always log errors
			Error.Println(queuedLogObj.str + "\r")

		}else if ( queuedLogObj.severity == "warning" ){

			// log warnings are warning level or lower
			if ( loggingLevel == "info" || loggingLevel == "event" || loggingLevel == "warning") {
				Warning.Println(queuedLogObj.str + "\r")
			}

		}else if ( queuedLogObj.severity == "event" ){

			// events come from the JS event engine's log function
			if ( loggingLevel == "info" || loggingLevel == "event") {
				Event.Println(queuedLogObj.str + "\r")
			}

		}else{

			// default
			Info.Println(queuedLogObj.str + "\r")
		}
	}
}



// creates a new log file if the current log file being used
// is not for the current date
func getLogFile() (err error) {
	
	// check if the date has changed
	now := getDate()

	if now != logFileDate{
		// update the stored current date
		logFileDate = now

		// close the handle to the old log file

		// open handle to the new log file

		// track if the log file was able to be opened for writting
		logFileOpened = false


		// create the log directory if it doesn't exist
		_, err = ioutil.ReadDir( lc.logDirectory )
		if err != nil {
			fmt.Fprintln(os.Stderr, "Log directory doesn't exist, creating")
			
			err = os.MkdirAll(lc.logDirectory, 0744)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		}

		// build the log path from the config properties
		//logPath := lc.logDirectory + "/" + lc.logPrefix + "_" + now + "." + lc.logSuffix
		fileName := lc.logPrefix + "_" + now + "." + lc.logSuffix
		logPath := filepath.Join(lc.logDirectory, fileName)

		// If the file doesn't exist, create it, or append to the file
		logFile, er := os.OpenFile( logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			
			fmt.Fprintln(os.Stderr, "Error opening log file", err)
			// failed to open the log file
			// all calls to log writters will gracefully fail

			return er
		}else{

			// success
			logFileOpened = true

			// log data will be written to 2 io.writters,
			// the log file and stdout/stderr
			//infoWarningIO := io.MultiWriter(logFile, os.Stdout)
			errorIO := io.MultiWriter(logFile, os.Stderr)

			// create the log writters
		    //initLogger(infoWarningIO, infoWarningIO, errorIO)
		    initLogger(logFile, logFile, errorIO)
		}
	}
	return err
}


func purgeOldLogs() (err error) {

	// get the existing log files
	existingLogs, er := getLogFiles()
	if er != nil {
		return er
	}else{
		for _, l := range existingLogs {
			fmt.Println("Existing log file: " + l)

			// get the time stamp on the log files
			// time stamp is encoded in the file name
			tsStr := l[ len(lc.logPrefix) + 1 : len(l) ]
			//tsStr = tsStr[: len(tsStr) - ( len(lc.logSuffix) + 1) ]

			// slicing the suffix off is giving me out of bounds,
			// so whatever, use the strings lib to replace it away
			tsStr = strings.Replace(tsStr, "." + lc.logSuffix , "", 1)
			//fmt.Println("Suffix stripped: " + tsStr)

			// convert to time stamp if possible
			timestamp, err := time.Parse(lc.timestampFormat, tsStr)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				continue
			}

			duration := time.Since(timestamp)
			hoursOld := duration.Hours()

			//fmt.Println("Hours Old: " + hoursOld)
			//fmt.Sprintf("Hours Old: %i", int64(hoursOld))

			// less than a day old, keep it
			if hoursOld < 24 {
				continue
			}

			daysOld := hoursOld / 24
			if int64(daysOld) > int64(lc.daysToKeep) {
				// purge
				fmt.Println("Purging log file: " + l)
				os.Remove(filepath.Join(lc.logDirectory, l))
			}
		}
	}
	return err
}




func getLogFiles() ( logNames []string, err error) {

	// bind to log dir
	logDirChildren, err := ioutil.ReadDir( lc.logDirectory )
	if err != nil {
		err = errors.New("getLogFiles() Error: Failed to open log directory")
		return nil, err
	}

	// loop dir contents
	for _, f := range logDirChildren {
		
		// skip sub dirs
		if f.IsDir() {
			continue
		}

		// skip if prefix is wrong
		if !strings.HasPrefix(f.Name(), lc.logPrefix) {
			continue
		}

		// skip if suffix is wrong
		if !strings.HasSuffix(f.Name(), lc.logSuffix) {
			continue
		}

		// what remains are files with the correct prefix and suffix of our log files
		logNames = append(logNames, f.Name())
	}

	return logNames, nil
}


// returns yyyy-mm-dd as a string
func getDate() string {
	current_time := time.Now().Local()
	return current_time.Format(lc.timestampFormat)
}
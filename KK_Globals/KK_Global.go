package KK_Globals

import(
	"time"
	"fmt"
	"strings"
	"os"
	"os/exec"
	"runtime"
	//"errors"
	//"text/tabwriter"
	"io/ioutil"
	"net/http"
	//"bytes"
	"kellestine.com/KKPLC_Gateway/KK_Logging"
)


/*
 *	Misc functions and globals live here
 */


var CLIcode chan string

// controlls the current CLI mode
var mode string

// controls updates to the cli mode
func SetMode(newMode string){

	// handle case
	newMode = strings.ToLower(newMode)

	// validate the new mode and update if valid
	switch newMode {
	case "tagdb", "plc":
	
	case "javascript":
		ClearTerminal()
		fmt.Println("JS Mode:")		

	case "debugging":
		ClearTerminal()
		fmt.Println("Debug Mode:")

	case "command" :
		ClearTerminal()
		fmt.Println("Command Mode:")
		fmt.Println("select a PLC with: set plc plcName")
		fmt.Println("List PLC's with: list plcs")

	default:
		Dbg("SetMode() Error: Invalid mode of [" + newMode +"] ", "error")

		// make the new mode the current mode
		// preventing the invalid mode from activating
		newMode = mode
	}
	mode = newMode
}

// returns the current CLI Mode
func Mode() string{
	return mode
}

//
// prints to the console if in debugging mode
// also forwards everything to the logging function,
// which will write to the log file based on the severity
//
// this is a Variadic function, so here is the function defination:
// Dbg(txt as string, optional severity as sting = "info" )
//
//func Dbg( txt, severity string ){
func Dbg( args ...string ){

	// validate args
	if len(args) < 1 {return}

	// populate local vars from args
	txt := args[0]
	severity := "info"
	if len(args) > 1 { severity = args[1] }

	// print the debug text if we are in debugging mode
	if ( mode == "debugging" ){fmt.Println(txt)}

	// send the text to the logger which will
	// log it depending on the severity and current log level
	KK_Logging.QueueLogWrite( txt, severity )
}


//
// gets a url
// returns the body of the url as a string
//
func HTTPGet( url string ) (content string, err error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	}

	content = ByteSliceToString(body)

	return content, err
}


// this doesn't need to be a function
// but I keep forgetting how to do this cast
func ByteSliceToString( byteSlice []byte ) (str string){
	s := string(byteSlice[:])
	return s
}


// returns a formated date string
func GetNow() string {

	// RFC1123     = "Mon, 11 Dec 2017 15:04:05 EST"
	now := time.Now()
	t := now.Format(time.RFC1123)
	
	return t
}


// clears the terminal window for windows or linx
// and probably mac
func ClearTerminal(){

	//runtime.GOOS -> linux, windows, darwin etc.
    if runtime.GOOS == "windows" {
	    cmd := exec.Command("cmd", "/c", "cls")
	    cmd.Stdout = os.Stdout
	    cmd.Run()
	}else{
        cmd := exec.Command("clear")
        cmd.Stdout = os.Stdout
        cmd.Run()
	}
}
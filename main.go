package main

import (
	"io"
	"strings"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"errors"
	//"time"
	"text/tabwriter"
	"kellestine.com/KKPLC_Gateway/KK_Rest"
)

//
// this is a rough first attempt to migrate some
// node.js code into a Go program
//



//
// basic struct for spawing a gateway instance
//
type PLCGateway struct {
	plcName string
	cmdBind *exec.Cmd
	stdin io.Writer 		// stdin pipe
	stdout io.Reader 		// stdout pipe
}

func (p PLCGateway) SendCommand( cmd string ) {
    io.WriteString(p.stdin, cmd + "\n")
}

func (p PLCGateway) AddTags( tagNames []string ) {

	for _, tagName := range tagNames {
		p.SendCommand("tagadd " + tagName)
		p.SendCommand("tagread " + tagName)		
	}
}



// 
// structure for tags
// there will be a slice of these
//
type tagObj struct {
	plcName string
	tagName string
	tagAlias string
	tagType string
	tagValue string
	writeable bool
}


// this slice of tags is the global tag database
var tagDatabase []tagObj


var mode string


//
//	Main
//
func main() {


	// command mode is the default mode
	mode = "command"

	fmt.Println("command mode:")
	fmt.Println("select a PLC with: set plc plcName")
	fmt.Println("List PLC's with: list plcs")
	fmt.Println("View tag database with: tagdb")

	// fire up the rest server
	//go KK_Rest.StartRest();
	KK_Rest.Placeholder();

	// OS stdin bind
	// so we can handle user input
	osScanner := bufio.NewScanner(os.Stdin)
	go func() {
		for osScanner.Scan() {
			inputHandler(osScanner.Text())
		}
	}()

	// in prod this will be replaced with a config load and init
	runTest()
	// currently runTest is blocking so we wont get past it

	fmt.Fprintln(os.Stdout, "System Running")

	// when the process is done show the exit code
	fmt.Fprintln(os.Stdout, "No more open child gateways, exiting.")


}//end main





func runTest(){
		//
	// TESTING
	//
	// define the external app that will simulate the PLC gateway layer
	cmdPath := "KK_PLC_Svr.exe"
	cmdArgs := []string{""}

	// instiate a new gateway process
	plcGatewayApp := plcGatewayBinder("PLC_Name", cmdPath, cmdArgs)

	// start the gateway child process
	err := plcGatewayApp.cmdBind.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
	}

	// connect to the PLC
	plcGatewayApp.SendCommand("connect 10.0.0.50")

	// disable the tag event logging
	plcGatewayApp.SendCommand("logging disable")
	
	// add non-string tags using member functions
	tagNames := []string{"KK.0", "KK.1"}
	plcGatewayApp.AddTags(tagNames)

	// start monitoring all tags in the default tag group
	plcGatewayApp.SendCommand("groupenable default_group")

	// switch to tag database monitoring
	mode = "tagdb"

	// hold the go app open until the nested app closes
	err = plcGatewayApp.cmdBind.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
	}

	// when the process is done show the exit code
	fmt.Fprintln(os.Stdout, "App exited: ", plcGatewayApp.cmdBind.ProcessState )
}





//
// starts an external app
// binds the app's stdin and stdout to the go app's
//
func plcGatewayBinder( plcName, exeFileName string, cmdArgs []string ) (gatewayObj PLCGateway) {


	// set the exec command with args
	//cmd := exec.Command(exeFileName, cmdArgs...)
	cmd := exec.Command(exeFileName, cmdArgs...)
	
	// cmd stdout bind
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		os.Exit(1)
	}

	// cmd stdin bind
	cmdIn, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdinPipe for Cmd", err)
		os.Exit(1)
	}

	// action on stdout from the cmd
	// pass the out data to the handler
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {

			// send the data to the handler with the PLC name
			handlePLCGatewaySTDOut( plcName, scanner.Text() )
		}
	}()

	gatewayObj = PLCGateway{ plcName, cmd, cmdIn, cmdReader }

	return gatewayObj
}// plcGatewayBinder



//
// this will be the handler for the stdout from PLCGateways
//
func handlePLCGatewaySTDOut( plcGatewayName, stdout string ){

	// send all the stdout to the go console if debugging
	if ( mode == "debugging" ){
		fmt.Printf(plcGatewayName +  " sent  | %s\n", stdout)
	}

	// check for tag updates
	if ( strings.Contains(stdout, "TAGUPDATE: ") ){
		
		// tag update messages should be in the form: tagname = value
		if ( mode == "debugging" ){ fmt.Println("Tag update recieved")}

		// split so we can ditch the TAGUPDATE: prefix
		splitter := strings.Split( stdout, "TAGUPDATE: " )

		// get the index of the equals sign
		index := strings.Index(splitter[1], " = ")
		if ( index > -1 ){

			// split on the equals
			splitter = strings.Split( splitter[1], " = " )

			// get the tag and val
			tagName := splitter[0]
			tagVal := splitter[1]

			if ( mode == "debugging" ){
				fmt.Printf("Tag: %s \n", tagName)
				fmt.Printf("Value : %s \n", tagVal)
			}

			// update the tag database
			tagBind, err := getTag( plcGatewayName, tagName )
			if ( err == nil ){

				// update the tag value
				tagDatabase[tagBind].tagValue = tagVal


				if ( mode == "tagdb" ){

					// clear the cmd.exe shell
				    cmd := exec.Command("cmd", "/c", "cls")
				    cmd.Stdout = os.Stdout
				    cmd.Run()				

				    // reprint the tag database
					printTagDB()
				}

			}else{

				// add the tag to the DB
				tagObj := tagObj{ plcGatewayName, tagName, tagName, "auto", tagVal, false }
				tagDatabase = append( tagDatabase, tagObj )
			}
		}

	}else{
		//fmt.Println("something other than a tag update recieved")
	}
}// handlePLCGatewaySTDOut



//
// takes a plcName and tagName
// returns the index in the tag DB of the matching tagObj if it exists
// else -1 and a non-nil error
//
func getTag( plcName, tagName string) (tagDBIndex int, err error){

	for i, v := range tagDatabase {
		if ( v.plcName == plcName && v.tagName == tagName ) {return i, err}
	}

	//fmt.Println("Tag not found in DB")

	// return the dummy tag and the error
	return -1, errors.New("tag doesn't exit")
}// getTag





//
// prints the tag database to the console
//
func printTagDB(){

	fmt.Println("---------------------------------------------------------------")
	fmt.Println("     Tag Database   [Enter any key and enter to exit]          ")
	fmt.Println("---------------------------------------------------------------")
	fmt.Println("            PLC Name|            Tag Name|           Tag Value|")
	fmt.Println("---------------------------------------------------------------")

	w := tabwriter.NewWriter(os.Stdout, 20, 1, 0, ' ', tabwriter.AlignRight|tabwriter.Debug)
	for _, v := range tagDatabase {
		fmt.Fprintln(w,  v.plcName + "\t" + v.tagName + "\t" + v.tagValue + "\v")
	}

	w.Flush()
	fmt.Println("---------------------------------------------------------------")
}// printTagDB




//
// handles stdin input to the go app
//
func inputHandler( rawInput string ){

	// convert to lowercase for command handling
	input := strings.ToLower(rawInput)

	// command hanlder
	if ( input == "tagdb" || input == "tdb" || input == "db" ){
		
		// tag database mode
		printTagDB()
		mode = "tagdb"

	}else if ( input == "d" || input == "debug" ){
		
		// debugging mode
		mode = "debugging"

	}else if ( input == "q" || input == "c" || input == "cmd" ){
		
		// command mode
		mode = "command"
		fmt.Println("command mode:")
		fmt.Println("select a PLC with: set plc plcName")
		fmt.Println("List PLC's with: list plcs")

	}else if ( mode == "command" ){

		if ( input == "list plcs" ){
			
			// store unique plcs from the tagDB into a map
			plcs := make(map[string]int)

			// loop the tag DB
			for i, v := range tagDatabase {

				// add the plc to the map if it isnt' already in there
				_, ok := plcs[v.plcName]
				if ( !ok ){
					plcs[v.plcName] = i
				}
			}

			// loop and print the PLC map
			for j, _ := range plcs {
				fmt.Printf("PLC: %s \n", j)
			}

		}else if ( strings.Index(input, "set plc") > -1 ){
			
			// a plc has been selected, parse it out
			// watch out for case, input is lcased
			splitter := strings.Split( input, "set plc " )
			fmt.Println( splitter[1] + " is set")
			mode = "plc"
		}


	}else if ( mode == "plc" ){
		// add logic here for plc_mode

	}else{
		
		// for debugging send all other commands to kkplc
		//io.WriteString(plcGatewayApp.cmdIn, rawInput + "\n" )
	}
}// inputHandler

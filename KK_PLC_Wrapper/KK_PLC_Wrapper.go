package KK_PLC_Wrapper

import (
	"io"
	"fmt"
	"os"
	"os/exec"
	"bufio"
	"strings"
	"errors"
	"kellestine.com/KKPLC_Gateway/KK_Globals"
)


//
// basic struct for spawing a gateway instance
//
type PLCGateway struct {
	plcName string
	cmdBind *exec.Cmd
	stdin io.Writer 		// stdin pipe
	stdout io.Reader 		// stdout pipe
}

// struct used to start a new PLCGateway
type gatewayInit struct{
	plcName string
	connectAddress string
	tagSlice []gatewayTag
}

type gatewayTag struct{
	tagName string
	alias string
	isString bool
	events []KK_Globals.TagEvent
}


// sends a command to the process stdin
// this can be remapped to send the command to the rest server
func (p PLCGateway) SendCommand( cmd string ) {
    io.WriteString(p.stdin, cmd + "\n")
}

// adds a slice of tags to the default group
func (p PLCGateway) AddTags( tagNames []string ) {

	for _, tagName := range tagNames {
		p.SendCommand("tagadd " + tagName)
		p.SendCommand("tagread " + tagName)		
	}
}

// adds a slice of tags to the default group
func (p PLCGateway) AddStringTags( tagNames []string ) {

	for _, tagName := range tagNames {
		p.SendCommand("tagstringadd " + tagName)
		p.SendCommand("tagread " + tagName)		
	}
}




// each gateway will be stored here
var PLCGateways []PLCGateway

// used when a PLC Gateway needs to be returned but we failed to find one
var nilGateway PLCGateway



func RunTest(){
	//
	// TESTING
	//

	// create a tag object
	var tagSlice []gatewayTag

	var tag gatewayTag
	tag.tagName = "KK.0"
	tag.alias = "KK.0"
	tag.isString = false
	
	// create event
	var event KK_Globals.TagEvent
	event.EventName = "Test_Event"
	event.TriggerType = "transition high"

	// add events to the tag
	var tagEvents []KK_Globals.TagEvent
	tagEvents = append(tagEvents, event)	
	tag.events = tagEvents

	tagSlice = append( tagSlice, tag)

	// make a second tag, no events on this one
	var tag2 gatewayTag
	tag2.tagName = "KK.2"
	tag2.alias = "KK.2"
	tag2.isString = false
	
	tagSlice = append( tagSlice, tag2)


	// create the initilization arguments for the gateway
	var initObj gatewayInit
	initObj.plcName = "cushions"
	initObj.connectAddress = "10.34.17.50"
	initObj.tagSlice = tagSlice
	

	// switch to tag database monitoring
	//KK_Globals.Mode = "tagdb"
	KK_Globals.Mode = "debugging"

	// this call blocks
	go StartAGateway(initObj)

}



//
// starts a new PLC Gateway, adds the tags to monitor, connects and starts monitoring
// this function will block until the PLC Gateway exe exits
//
func StartAGateway( init gatewayInit ){

	// if enabled, a new gateway instance will spawn up if this one exits
	autoReconnect := true

	// Process the tag slice
	var tagSlice [] string
	var strTagSlice [] string
	for _, t := range init.tagSlice {

		// build struct expected by the AddTag method
		var tag KK_Globals.TagObj
		tag.PLCName = init.plcName
		tag.TagName = t.tagName
		tag.Events = t.events

		// add the tag to the tag database
		KK_Globals.TagDatabase.AddTag( tag )

		// parse out the Tag and strTag names for the gateway
		if ( t.isString ) {
			strTagSlice = append( strTagSlice, t.tagName )
		}else{
			tagSlice = append( tagSlice, t.tagName )
		}
	}


	// instiate a new gateway process
	plcGatewayApp := plcGatewayBinder(init.plcName, KK_Globals.GatewayPath, KK_Globals.GatewayArgs)

	// add this gateway to the global slice
	PLCGateways = append(PLCGateways, plcGatewayApp)

	// start the gateway child process
	err := plcGatewayApp.cmdBind.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
	}

	// connect to the PLC
	plcGatewayApp.SendCommand("connect " + init.connectAddress)

	// disable the tag event logging
	plcGatewayApp.SendCommand("logging disable")

	// add tags to the gateway monitor groups
	plcGatewayApp.AddTags(tagSlice)
	plcGatewayApp.AddStringTags(strTagSlice)

	// start monitoring all tags in the default tag group
	plcGatewayApp.SendCommand("groupenable default_group")

	// hold the go app open until the nested app closes
	err = plcGatewayApp.cmdBind.Wait()
	if err != nil {
		if ( KK_Globals.Mode == "debugging" ){
			fmt.Println("gateway has exited unexpectedly", err)
		}
	}

	/////////////////////////////////////////////////////////////////////////////
	/////////// code flow only passes this point when the exe exits /////////////
	/////////////////////////////////////////////////////////////////////////////

	// clean up the plcGatewayApp slice to removed the exited gateway
	i, err := getPLCGatewayIndex(init.plcName)
	if err != nil {
		if ( KK_Globals.Mode == "debugging" ){
			fmt.Println("Error finding " + init.plcName + " in Gateway Slice", err)
		}
	}else{

		// remove the gateway from the slice
		PLCGateways = append(PLCGateways[:i], PLCGateways[i+1:]...)
	}

	// reconnect if auto reconnect is enabled
	if autoReconnect {
		StartAGateway(init)
	}
}




//
// this function takes a tag update string in the format:
// tagName=newValue@plcName
// a tag update message is then sent to the correct PLC gateway
//
func HandleTagUpdateString( tagUpdateStr string ) (response string, err error) {

	if ( !strings.Contains(tagUpdateStr, "@") ){
		err = errors.New("expecting tagName=newValue@plcName")
		response = "Error, expecting tagName=newValue@plcName"
	}else{
	
		// break string into the parts
		// expected format: tagName=newValue@plcName
		splitter := strings.Split(tagUpdateStr, "@")
		plcName := splitter[1]
		tagUpdateCmd := splitter[0]

		//fmt.Fprintln(os.Stdout, "PLC Name: ", plcName )

		if plcName == "virtual"{

			// add to the tag update queue
			KK_Globals.TagDatabase.QueueTagUpdate( "TAGUPDATE: " + tagUpdateStr )
			response = "virtual tag queued for update"

		}else{

			// find the PLC Gateway by name
			gwPt, errr := getPLCGateway(plcName)
			if errr != nil {
				err = errr
				//fmt.Fprintln(os.Stdout, "getPLCGateway returned error: ", errr )
				response = "Error finding " + plcName + " in Gateway Slice"
			}else{

				// get the GW object
				gatewayObj := *gwPt

				// update the tag update command to match what the GW expects
				tagUpdateCmd = "tagset " + tagUpdateCmd
				//fmt.Fprintln(os.Stdout, "Sending Command : ", tagUpdateCmd )

				// send the formated command
				gatewayObj.SendCommand(tagUpdateCmd)
				response = tagUpdateCmd + " sent to " + plcName
				//fmt.Fprintln(os.Stdout, "getPLCGateway returned a GW pointer: ", gatewayObj.plcName )
			}
		}
	}
	return response, err
}



//
// takes a PLC Gateway name and returns a pointer to that Gateway
//
func getPLCGateway( PLCName string ) ( plcGW *PLCGateway, err error ){

	// find the PLC Gateway by name and return a pointer to it
	for _, v := range PLCGateways {
		if ( v.plcName == PLCName ) {
			return &v, err
		}
	}

	// failed to find PLCName
	err = errors.New("PLCName " + PLCName + " not found in gateway slice")
	return plcGW, err
}


//
// takes a PLC Gateway name and returns it's index in the gateway slice
//
func getPLCGatewayIndex( PLCName string ) ( i int, err error ){

	// find the PLC Gateway by name and return a pointer to it
	for j, v := range PLCGateways {
		if ( v.plcName == PLCName ) {
			return j, err
		}
	}

	// failed to find PLCName
	err = errors.New("PLCName " + PLCName + " not found in gateway slice")
	return -1, err
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
	if ( KK_Globals.Mode == "debugging" ){
		fmt.Printf(plcGatewayName +  " sent  | %s\n", stdout)
	}

	// check for tag updates
	if ( strings.Contains(stdout, "TAGUPDATE: ") ){
		
		stdout = stdout + "@" + plcGatewayName

		// add to the tag update queue
		KK_Globals.TagDatabase.QueueTagUpdate( stdout )

	}else{
		//fmt.Println("something other than a tag update recieved")
	}


}// handlePLCGatewaySTDOut


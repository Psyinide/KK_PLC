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
	"kellestine.com/KKPLC_Gateway/KK_Tag_DB"
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
type GatewayInit struct{
	PlcName string
	ConnectAddress string
	TagSlice []GatewayTag
	Path string
	Args []string
}

type GatewayTag struct{
	TagName string
	Alias string
	IsString bool
	IsWritable bool
	Events []KK_Tag_DB.TagEvent
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



//
// handle write requests to the PLCs
//
func ProcessGatewayTagSetQueue(){
	for {
		queuedTagSetStr := <-KK_Tag_DB.GatewayTagSetQueue
		
		KK_Globals.Dbg("ProcessGatewayTagSetQueue() Processing Queue", "info")
		_, err := HandleTagUpdateString(queuedTagSetStr)
		if err != nil {
			KK_Globals.Dbg(err.Error(), "error")
		}
	}
}



// this function blocks
//
// starts a new PLC Gateway, adds the tags to monitor, connects and starts monitoring
// this function will block until the PLC Gateway exe exits
// the function will attempt to auto relaunch the gateway if the gateway proc ends
// if the proc fails to start the function exits
//
func StartAGateway( init GatewayInit ) {

	// if enabled, a new gateway instance will spawn up if this one exits
	// todo: \/ configurable \/
	autoReconnect := true

	// Process the tag slice
	var tagSlice [] string
	var strTagSlice [] string
	for _, t := range init.TagSlice {

		// build struct expected by the AddTag method
		var tag KK_Tag_DB.TagObj
		tag.PLCName = init.PlcName
		tag.TagAddress = t.TagName 	// yes these seems backwards
		tag.TagName = t.Alias		// clean up later
		tag.IsWritable = t.IsWritable
		tag.Events = t.Events

		// add the tag to the tag database
		KK_Tag_DB.TagDatabase.AddTag( tag )

		// parse out the Tag and strTag names for the gateway
		if ( t.IsString ) {
			strTagSlice = append( strTagSlice, t.TagName )
		}else{
			tagSlice = append( tagSlice, t.TagName )
		}
	}


	// instiate a new gateway process
	plcGatewayApp := plcGatewayBinder(init.PlcName, init.Path, init.Args)

	// add this gateway to the global slice
	PLCGateways = append(PLCGateways, plcGatewayApp)

	// start the gateway child process
	err := plcGatewayApp.cmdBind.Start()
	if err != nil {
		KK_Globals.Dbg( init.PlcName + " fatal gateway error, cannot start gateway: " + err.Error(), "error" )


		// todo: come back to this idea of raising events in the system
		KK_Tag_DB.TagDatabase.SetSystemTag( "plcTagInitFailure", init.PlcName)
		return
	}

	// connect to the PLC
	plcGatewayApp.SendCommand("connect " + init.ConnectAddress)

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
		KK_Globals.Dbg( init.PlcName + " gateway has exited unexpectedly", "warning" )
		KK_Globals.Dbg( init.PlcName + " gateway Error: " + err.Error(), "error" )
	}

	/////////////////////////////////////////////////////////////////////////////
	/////////// code flow only passes this point when the exe exits /////////////
	/////////////////////////////////////////////////////////////////////////////

	// clean up the plcGatewayApp slice to removed the exited gateway
	i, err := getPLCGatewayIndex(init.PlcName)
	if err != nil {
		KK_Globals.Dbg( "Error finding " + init.PlcName + " in Gateway Slice", "warning" )
		KK_Globals.Dbg( err.Error(), "warning" )
	}else{

		// remove the gateway from the slice
		PLCGateways = append(PLCGateways[:i], PLCGateways[i+1:]...)
	}

	// reconnect if auto reconnect is enabled
	if autoReconnect {
		KK_Globals.Dbg( "Restarting " + init.PlcName, "info" )
		StartAGateway(init)
	}
}




//
// this function takes a tag update string in the format:
// tagName=newValue@plcName
// a tag update message is then sent to the correct PLC gateway
//
func HandleTagUpdateString( tagUpdateStr string ) (response string, err error) {

	KK_Globals.Dbg("HandleTagUpdateString() Processing:" + tagUpdateStr, "info")

	// validate the input string
	if ( !strings.Contains(tagUpdateStr, "@") ){
		err = errors.New("expecting tagName=newValue@plcName")
		response = "Error, expecting tagName=newValue@plcName"
	}else{
	
		// break string into the parts
		// expected format: tagName=newValue@plcName
		splitter := strings.Split(tagUpdateStr, "@")

		// validate we have content after the @ sign
		if len(splitter) < 2 {
			err = errors.New("expecting tagName=newValue@plcName")
			response = "Error, expecting tagName=newValue@plcName"
			return response, err
		}

		// assign friendly names
		plcName := splitter[1]
		tagUpdateCmd := splitter[0]

		// find the PLC Gateway by name
		gwPt, errr := getPLCGateway(plcName)
		if errr != nil {
			err = errr
			response = "Error finding " + plcName + " in Gateway Slice"
		}else{

			// get the GW object
			gatewayObj := *gwPt

			// update the tag update command to match what the GW expects
			tagUpdateCmd = "tagset " + tagUpdateCmd

			// send the formated command
			gatewayObj.SendCommand(tagUpdateCmd)
			response = tagUpdateCmd + " sent to " + plcName
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
	KK_Globals.Dbg(plcGatewayName +  " sent  |" + stdout, "info")

	// check for tag updates
	if ( strings.Contains(stdout, "TAGUPDATE: ") ){
	
/*	
		// break string into the parts
		// expected format: tagName=newValue@plcName
		splitter := strings.Split(stdout, "@")

		// validate we have content after the @ sign
		if len(splitter) < 2 {
			// bad
		}else{

			// assign friendly names
			tagName := splitter[0]

			// try to replace the tag address with the tag name
			tagBind, err := KK_Tag_DB.TagDatabase.GetTagIndexByAddress( plcGatewayName, tagName )
			if ( err == nil ){
				tagName := KK_Tag_DB.TagDatabase.Tags[tagBind].TagName
				tagAddress := KK_Tag_DB.TagDatabase.Tags[tagBind].TagAddress
				stdout = strings.Replace(stdout, tagAddress, tagName, 1)
			}
		}
*/
		// finish build the tag update string
		stdout = stdout + "@" + plcGatewayName

		// add to the tag update queue
		KK_Tag_DB.TagDatabase.QueueTagUpdate( stdout )

	}else{
		//fmt.Println("something other than a tag update recieved")
	}
}// handlePLCGatewaySTDOut



/*

func RunTest(){
	//
	// TESTING
	//

	// create a slice tag objects
	var tagSlice []GatewayTag

	// create first tag
	var tag GatewayTag
	tag.TagName = "KK.0"
	tag.Alias = "KK.0"
	tag.IsString = false
	
	// create event for the tag
	var event KK_Globals.TagEvent
	event.EventName = "Test_Event"
	event.TriggerType = "transition high"

	// add events to the tag
	var tagEvents []KK_Globals.TagEvent
	tagEvents = append(tagEvents, event)	
	tag.Events = tagEvents

	// add tag to slice
	tagSlice = append( tagSlice, tag)

	// make a second tag, no events on this one
	var tag2 GatewayTag
	tag2.TagName = "KK.2"
	tag2.Alias = "KK.2"
	tag2.IsString = false
	
	// add tag to slice
	tagSlice = append( tagSlice, tag2)


	//
	// create the initilization arguments for the gateway
	// this is a vbs gateway
	//
	var initObj GatewayInit
	initObj.PlcName = "vbs"
	initObj.ConnectAddress = "10.34.17.50"
	initObj.TagSlice = tagSlice
	initObj.Path = "C:\\Windows\\System32\\cscript.exe"
	initObj.Args = []string{"test.vbs"}


	// switch to tag database monitoring
	//KK_Globals.Mode = "tagdb"
	KK_Globals.Mode = "debugging"

	// this call blocks
	go StartAGateway(initObj)


	//
	// lets connect to a real PLC
	//
	var initObj2 GatewayInit
	initObj2.PlcName = "cushions"
	initObj2.ConnectAddress = "10.34.17.50"
	initObj2.TagSlice = tagSlice
	initObj2.Path = "C:\\Go\\gocode\\src\\kellestine.com\\KKPLC_Gateway\\KK_PLC\\KK_PLC_Svr.exe"
	initObj2.Args = []string{"ignoreScript=true"}

	// this call blocks
	//go StartAGateway(initObj2)
}
*/
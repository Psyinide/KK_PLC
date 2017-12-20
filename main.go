package main

import (
	//"io"
	"strings"
	"bufio"
	"fmt"
	"os"
	//"os/exec"
	//"errors"
	"time"
	//"text/tabwriter"
	//"log"
	"kellestine.com/KKPLC_Gateway/KK_Globals"
	"kellestine.com/KKPLC_Gateway/KK_Rest"
	"kellestine.com/KKPLC_Gateway/KK_PLC_Wrapper"
	"kellestine.com/KKPLC_Gateway/KK_Events"
)

//
// this is a rough first attempt to migrate some
// node.js code into a Go program
//



//			//
//	Main 	//
//			//
func main() {


	// command mode is the default mode
	KK_Globals.Mode = "command"

	fmt.Println("command mode:")
	fmt.Println("select a PLC with: set plc plcName")
	fmt.Println("List PLC's with: list plcs")
	fmt.Println("View tag database with: tagdb")


	// init the tag database update queue
	KK_Globals.TagDatabase.TagUpdateQueue = make(chan string, 100)
	KK_Globals.EventQueue = make(chan KK_Globals.EventQueueObj, 100)

	go KK_Globals.TagDatabase.ProcessTagUpdateQueue()
	go KK_Events.InitEventEngine()

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
	KK_PLC_Wrapper.RunTest()


	// config loaded, system up
	fmt.Fprintln(os.Stdout, "System Running")

	// hold the app open
    for {
        time.Sleep(300 * time.Millisecond)
    }

}//end main




//
// handles stdin input to the go app
//
func inputHandler( rawInput string ){

	// convert to lowercase for command handling
	input := strings.ToLower(rawInput)

	// command hanlder
	if ( input == "tagdb" || input == "tdb" || input == "db" ){
		
		// tag database mode
		KK_Globals.TagDatabase.PrintTagDB()
		KK_Globals.Mode = "tagdb"

	}else if ( input == "d" || input == "debug" ){
		
		// debugging mode
		KK_Globals.Mode = "debugging"

	}else if ( input == "q" || input == "c" || input == "cmd" ){
		
		// command mode
		KK_Globals.Mode = "command"
		fmt.Println("command mode:")
		fmt.Println("select a PLC with: set plc plcName")
		fmt.Println("List PLC's with: list plcs")


	}else if ( input == "kk" ){
		// this is for testing

		var tag KK_Globals.TagObj
		tag.PLCName = "virtual"
		tag.TagName = "VKK"

		err:= KK_Globals.TagDatabase.AddTag(tag)
		if ( err == nil ){
			// this error is raised if the tag already exists
		}


	}else if ( KK_Globals.Mode == "command" ){

		if ( input == "list plcs" ){
			
			// store unique plcs from the tagDB into a map
			plcs := make(map[string]int)

			// loop the tag DB
			for i, v := range KK_Globals.TagDatabase.Tags {

				// add the plc to the map if it isnt' already in there
				_, ok := plcs[v.PLCName]
				if ( !ok ){
					plcs[v.PLCName] = i
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
			KK_Globals.Mode = "plc"
		}

	}else if ( KK_Globals.Mode == "plc" ){
		// add logic here for plc_mode

	}else{
		
		// for debugging send all other commands to kkplc
		//io.WriteString(plcGatewayApp.cmdIn, rawInput + "\n" )
	}
}// inputHandler

package KK_Events
import (
	"io/ioutil"
	"fmt"
	//"unicode/utf8"
	"path/filepath"
	"kellestine.com/KKPLC_Gateway/KK_Globals"
	"kellestine.com/KKPLC_Gateway/KK_Tag_DB"
	"github.com/robertkrimen/otto"
)


/*
 *
 *	this is the javascript event handler module
 * 	the init function should be run as a go routine:
 *	go KK_Events.InitEventEngine()
 *
 */

func RunEventEngine() {

	// Start the VM
	vm := otto.New()

	// expose a function to allow writing to the log file
	vm.Set("log", func(call otto.FunctionCall) otto.Value {
	    
	    // get the JS func call arguments into Go
	    strToLog := call.Argument(0).String()
	    severity := call.Argument(1).String()

	    // make sure we have a proper value for the severity, default to "event"
	    if ( severity != "info" && severity != "warning" && severity != "error") {
	    	severity = "event"
	    }

	    // log it
	    KK_Globals.Dbg("Event Log: " + strToLog, severity)

	    // this is a null return
	    return otto.Value{}
	})


	// expose a function to get the contents of a web page
	vm.Set("HTTPGet", func(call otto.FunctionCall) otto.Value {
	    
	    // get the JS func call arguments into Go
	    url := call.Argument(0).String()

	    d, err := KK_Globals.HTTPGet(url)
	    if err != nil{
			result, _ := vm.ToValue(err.Error())
			return result
    	}else{
			result, _ := vm.ToValue(d)
			return result
    	}
	})


	// create the getTagValue function in the VM
	vm.Set("getTagValue", func(call otto.FunctionCall) otto.Value {
	    
	    // get the JS func call arguments into Go
	    plcName := call.Argument(0).String()
	    tagName := call.Argument(1).String()

	    // get the tag's value
		tagVal, err := KK_Tag_DB.TagDatabase.GetTagValue( plcName, tagName )
		if ( err == nil ){

			// return the tag's value to the JS caller
			result, _ := vm.ToValue(tagVal)
			return result
		}else{

			// return a generic error to the JS caller
			result, _ := vm.ToValue("KK_ERROR")
			return result
		}
	})


	// create the setTagValue function in the VM
	vm.Set("setTagValue", func(call otto.FunctionCall) otto.Value {
	    
	    // get the JS func call arguments into Go
	    plcName := call.Argument(0).String()
	    tagName := call.Argument(1).String()
	    tagValue := call.Argument(2).String()

	    // build the tag update string
		// TAGUPDATE: tagName=newValue@plcName
	    tagUpdateStr := "TAGUPDATE: " + tagName + "=" + tagValue + "@" + plcName

	    // queue the tag update
		KK_Tag_DB.TagDatabase.QueueTagSet( tagUpdateStr )

		// must return, this returns nothing for a method call
		return otto.Value{}
	})


	// events should be in a sub dir relative to the current dir
	eventsConfigDir := filepath.Join(".", "KK_Events")

	// validate that the events directory exists
	_, er := ioutil.ReadDir( eventsConfigDir )
	if er != nil {
		KK_Globals.Dbg(er.Error(), "error")
		KK_Globals.Dbg("KK_Events directory read error, can't read events", "error")
	}else{

		// load in the optional JS Event Environment if it exists
		eventsEnvPath := filepath.Join(eventsConfigDir, "events_env.js")
		byteSlice, err := ioutil.ReadFile(eventsEnvPath)
		if err == nil {
			// execute the config file in the JS VM
			_,err:= vm.Run( byteSlice )
			if err != nil{
				KK_Globals.Dbg("Error executing events_env.js: ", "error")
				KK_Globals.Dbg(err.Error(), "error")
			}
		}

		// load in the JS Event Config
		eventsConfigPath := filepath.Join(eventsConfigDir, "config_events.js")
		byteSlice, err = ioutil.ReadFile(eventsConfigPath)
		if err != nil {
			KK_Globals.Dbg("Error loading config_events.js: ", "error")
			KK_Globals.Dbg(err.Error(), "error")
		}else{

			// execute the config file in the JS VM
			_,err:= vm.Run( byteSlice )
			if err != nil{
				KK_Globals.Dbg("Error executing config_events.js: ", "error")
				KK_Globals.Dbg(err.Error(), "error")
			}
		}
	}


	// CLI processor loop
	go func(){
		for {
			// block until we have work to do
			jsCode := <-KK_Globals.CLIcode
			
			// execute it in the VM
			_,err:= vm.Run( jsCode )
			if err != nil{
				fmt.Println("JS Error: ", err)
			}
		}
	}()


	// event processor loop
	jsCode := ""
	for {
		// block until we have work to do
		eventObj := <-KK_Tag_DB.EventQueue
		
		// build the JS code to execute
		// $EventName( $PLCName, $TagName, $OldValue, $NewValue )
		jsCode = eventObj.EventName + "('" + eventObj.PLCName
		jsCode += "', '" + eventObj.TagName + "', '"
		jsCode += eventObj.OldValue + "', '" + eventObj.NewValue + "')"

		// execute it in the VM
		vm.Run(jsCode) 
	}

}// RunEventEngine



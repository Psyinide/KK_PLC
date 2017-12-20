package KK_Events
import (
	"io/ioutil"
	"fmt"
	//"unicode/utf8"
	"kellestine.com/KKPLC_Gateway/KK_Globals"
	"github.com/robertkrimen/otto"
)


func InitEventEngine() {

	// Start the VM
	vm := otto.New()

	// create the getTagValue function in the VM
	vm.Set("getTagValue", func(call otto.FunctionCall) otto.Value {
	    
	    // get the JS func call arguments into Go
	    plcName := call.Argument(0).String()
	    tagName := call.Argument(1).String()

	    // get the tag's value
		tagVal, err := KK_Globals.TagDatabase.GetTagValue( plcName, tagName )
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
		KK_Globals.TagDatabase.QueueTagUpdate( tagUpdateStr )

		// must return, this returns nothing for a method call
		return otto.Value{}
	})


	// load in the JS Event Config
	byteSlice, err := ioutil.ReadFile("./KK_Events/config_events.js")
	if err != nil {
		fmt.Println("Err: ", err)
	}else{


		// execute the config file in the JS VM
		_,err:= vm.Run( byteSlice )
		if err != nil{
			fmt.Println("Error executing config_events.js: ", err)
		}

/*
		// once the config is loaded block on an empty queue channel
		vm.Run(`
		    Test_Event("cushions", "KK.2", "False", "True");
		`) 
*/
	}

	jsCode := ""
	for {

		//fmt.Println("Ready for events")

		// block until we have work to do
		eventObj := <-KK_Globals.EventQueue
		
		// build the JS code to execute
		jsCode = eventObj.EventName + "('" + eventObj.PLCName
		jsCode += "', '" + eventObj.TagName + "', '"
		jsCode += eventObj.OldValue + "', '" + eventObj.NewValue + "')"

		//fmt.Println(jsCode)
		vm.Run(jsCode) 
	}


	// test the function
	/*
	vm.Run(`
	    console.log(getTagValue("cushions", "KK.0"));
	    setTagValue("cushions", "KK.2", "False");
	`) 
	*/



/*

	// get a value from the VM
	value, err := vm.Get("abc"); 
	if err != nil {
		fmt.Printf("Go: abc = %s\n", value)
	}

	// Set a number
	vm.Set("def", 11)
	vm.Run(`
	    console.log("JS: " + "The value of def is " + def);
	    // The value of def is 11
	`)

	// Set a string
	vm.Set("xyzzy", "Nothing happens.")
	vm.Run(`
	    console.log("JS: " + xyzzy.length); // 16
	`)

	// Get the value of an expression
	value, err = vm.Run("xyzzy.length")
	{
	    // value is an int64 with a value of 16
	    value, _ := value.ToInteger()
	    value = value
	}

	// An error happens
	value, err = vm.Run("abcdefghijlmnopqrstuvwxyz.length")
	if err != nil {
	    // err = ReferenceError: abcdefghijlmnopqrstuvwxyz is not defined
	    // If there is an error, then value.IsUndefined() is true
	    fmt.Println("Go: JS Error in Console")
	}

	// Set a Go functions
	vm.Set("sayHello", func(call otto.FunctionCall) otto.Value {
	    fmt.Printf("Hello, %s.\n", call.Argument(0).String())
	    return otto.Value{}
	})

	vm.Set("twoPlus", func(call otto.FunctionCall) otto.Value {
	    right, _ := call.Argument(0).ToInteger()
	    result, _ := vm.ToValue(2 + right)
	    return result
	})

	//Use the functions in JavaScript
	vm.Run(`
	    sayHello("Xyzzy");      // Hello, Xyzzy.
	    sayHello();             // Hello, undefined

	    result = twoPlus(2.0); // 4
	`) 

	vm.Set("getTag", func(call otto.FunctionCall) otto.Value {
	    vm.Set("tagName", "KK_Tag")
	    vm.Set("tagValue", "KK")
	    return otto.Value{}
	})

	vm.Run(`
	    getTag("Xyzzy");
	    console.log("JS: tagName = " + tagName)
	    console.log("JS: tagValue = " + tagValue)
	`) 

	*/
}

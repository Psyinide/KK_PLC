console.log("Loading JS Config");

/*
	this script should contain a function for
	each defined event in config.json

	code that is not directly an event should be
	put in events_env.js to keep this script as 
	streamlined as possible
*/

//
function log_barcode( plcName, tagName, oldValue, newValue ){
	if ( newValue != 0 ){ log("Barcode Read = " + newValue, "event"); }
}

// this is a test
function Test_Event( plcName, tagName, oldValue, newValue ){
	//console.log("Test_Event_Running")

	if ( plcName == "cushions" ) {

		checkValue = "False"
		if ( getTagValue( "cushions", "KK.3" ) === "False" ) {checkValue = "True"}

		setTagValue("cushions", "KK.3", checkValue)
	}
}


function virtual_event( plcName, tagName, oldValue, newValue ){
	console.log("virtual_event Running")

	if ( plcName == "vbs" ) {
		if ( newValue === "True" ){
			
			newVirtualValue = "False"
			if ( getTagValue( "virtual", "KK_Flag" ) === "False" ) {newVirtualValue = "True"}
			setTagValue("virtual", "KK_Flag", newVirtualValue)
		}
	}
}


//
console.log("JS Config Loaded");
//
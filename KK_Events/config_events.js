console.log("Loading JS Config");



function Awesome_Event( plcName, tagName, oldValue, newValue ){
	
	console.log("Awesome_Event")

	// check for transition high
	if ( newValue === "True" ){
	
		// toggle a tag	
		newVirtualValue = "False"
		if ( getTagValue( "virtual", "VKK" ) === "False" ) {newVirtualValue = "True"}
		setTagValue("virtual", "VKK", newVirtualValue)

	}
}



function Test_Event( plcName, tagName, oldValue, newValue ){
	
	//console.log("Test_Event_Running")
	if ( newValue === "True" ){
		
		newVirtualValue = "False"
		if ( getTagValue( "virtual", "VKK" ) === "False" ) {newVirtualValue = "True"}
		setTagValue("virtual", "VKK", newVirtualValue)
	}
}
console.log("Loading JS Environment");

/*
	this script should contain helper functions for the JS VM
	2 example functions provided
*/

function dbg(str){ console.log(str) }

function help(){
	dbg("")
	dbg("Built in functions:")
	dbg("")
	dbg( "gtv getTagValue( \"PLCName\", \"TagName\" )" )
	dbg( "stv setTagValue( \"PLCName\", \"TagName\", \"NewValue\" )" )
	dbg( "log( \"string to log\", \"levelLevel=event\" )" )
	dbg( "HTTPGet(\"url\")" )
}

function gtv(a,b){ getTagValue(a,b) }
function stv(a,b){ setTagValue(a,b) }

//
console.log("JS Environment Loaded");
//
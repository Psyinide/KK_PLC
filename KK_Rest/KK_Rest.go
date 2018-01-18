package KK_Rest

// this is the bare bones framework for the server side component of my KKPLC application
// the dotnet app will send all point updates to this app which can act as the event handler

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"encoding/json"
	"strings"
	"kellestine.com/KKPLC_Gateway/KK_Globals"
	"kellestine.com/KKPLC_Gateway/KK_Tag_DB"
	//"kellestine.com/KKPLC_Gateway/KK_PLC_Wrapper"
)

// should load this from config:
var listeningPort string// = "3001"

/*
 *
 *	this is the web server
 *
 */
func StartRest( port string ) {

	listeningPort = port

	fmt.Println("Starting HTTP Server Router")
	mx := mux.NewRouter()


	//
	// json output
	//

	// all tags in the tag DB as JSON
	mx.HandleFunc("/api/v1/json/alltags", sendAllTagsAsJSON)
	// http://127.0.0.1:3000/api/v1/json/alltags

	// all tags in {plcname} as JSON
	mx.HandleFunc("/api/v1/json/plctags/{plcname}", sendAllTagsByPLCAsJSON)
	// http://127.0.0.1:3000/api/v1/json/plctags/cushions

	// one specific tag in {plcname}/{tagname} as JSON
	mx.HandleFunc("/api/v1/json/plctags/{plcname}/{tagname}", sendTagAsJSON)
	// http://127.0.0.1:3000/api/v1/json/plctags/cushions/KK.2

	// return specific tags from the tagdb as JSON
	mx.HandleFunc("/api/v1/json/specifictags/{tagsarray}", sendSpecificTagsAsJSON)
	// http://127.0.0.1:3000/api/v1/json/specifictags/[KK.2@cushions,flag@virtual]

	
	//
	// API Input
	//

	// sets a tag using tagname=tagvalue@plcname syntax
	mx.HandleFunc("/api/v1/tags/set/{tagdata}", tagSet)
	// http://127.0.0.1:3000/api/v1/tags/set/Ken=False@virtual


	//
	// testing stuff
	//
	mx.HandleFunc("/test/{val}", testResponse)
	mx.HandleFunc("/error/{error}", errorEvent)

	// route the root dir to the html folder
	mx.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("html/")))) 
	// http://127.0.0.1:3000

	// listen on all addresses
	http.ListenAndServe(":" + listeningPort, mx)

} //end startRest


// I call this is place of starting the rest server in main
// for testing, that way I don't get an included and not used error
func Placeholder() {
	return
}



// handler for a tag update/set
func tagSet(w http.ResponseWriter, r *http.Request) {
	tagdata := mux.Vars(r)["tagdata"]
	fmt.Println("New tag data ")
	fmt.Println(tagdata)
	fmt.Println(" ")

	// validate the input string format
	if ( !strings.Contains(tagdata, "@") ){
		w.Write([]byte("Error, expecting tagName=newValue@plcName"))
	}else{
	
		// tagName=newValue@plcName

		// this needs to be tested
		KK_Tag_DB.TagDatabase.QueueTagSet( "TAGUPDATE: " + tagdata)
		w.Write([]byte("Tag Set queued"))
	}
}



//
// returns the entire tag database as JSON
//
func sendAllTagsAsJSON(w http.ResponseWriter, r *http.Request) {

// r.Queries("key", "value")

	json, err := json.Marshal(KK_Tag_DB.TagDatabase.Tags)
	if err != nil {
		fmt.Println("error:", err)
		w.Write([]byte("JSON Marshal Error"))
	}

	w.Write(json)
}



//
// takes a plc name as a parameter
// returns all tags in that gateway as json
//
func sendAllTagsByPLCAsJSON(w http.ResponseWriter, r *http.Request) {

	// get the PLC name from the url
	plcname := mux.Vars(r)["plcname"]

	// create a new slice of tags to return
	var tmpTagDB []KK_Tag_DB.TagObj

	// populate the new slice with the correct tags
	for _, v := range KK_Tag_DB.TagDatabase.Tags {
		if v.PLCName == plcname {
			tmpTagDB = append(tmpTagDB, v)
		}
	}

	// return it as JSON
	json, err := json.Marshal(tmpTagDB)
	if err != nil {
		fmt.Println("error:", err)
		w.Write([]byte("JSON Marshal Error"))
	}
	w.Write(json)
}


//
// takes a json array of tagName@plcName
// returns the tags as json
//
func sendSpecificTagsAsJSON(w http.ResponseWriter, r *http.Request) {

	// get the json from the url
	jsonTagArray := mux.Vars(r)["tagsarray"]
	jsonByteArray := []byte(jsonTagArray)

	// validate the json
	if !json.Valid(jsonByteArray){
		KK_Globals.Dbg("sendSpecificTagsAsJSON: Invalid JSON tag array","warning")
		w.Write([]byte("Invalid JSON tag array"))
		return
	}

	// attempt to parse the validated json into a string slice
	var tagRequestSlice []string
	err := json.Unmarshal(jsonByteArray, &tagRequestSlice)
	if err != nil {
		KK_Globals.Dbg("sendSpecificTagsAsJSON: Invalid JSON tag array","warning")
		KK_Globals.Dbg( err.Error(), "warning" )
		w.Write([]byte("Invalid JSON tag array"))
		return
	}

	// create a new slice of tags to return
	var tmpTagDB []KK_Tag_DB.TagObj

	// loop each tag in the json tag slice
	for _, tagStr := range tagRequestSlice {

		KK_Globals.Dbg( tagStr, "info")

		if ( !strings.Contains(tagStr, "@") ){
			KK_Globals.Dbg("sendSpecificTagsAsJSON: Error, expecting tagName@plcName","warning")
			w.Write([]byte("Error, expecting tagName@plcName"))
		}else{
		
			// break string into the parts
			// expected format: tagName=newValue@plcName
			splitter := strings.Split(tagStr, "@")

			// validate we have content after the @ sign
			if len(splitter) < 2 {
				KK_Globals.Dbg("sendSpecificTagsAsJSON: Error, expecting tagName@plcName","warning")
				w.Write([]byte("Error, expecting tagName@plcName"))
				return
			}

			// assign friendly names
			tagName := splitter[0]
			plcName := splitter[1]
			
			KK_Globals.Dbg( "Tag Name" + tagName, "info")
			KK_Globals.Dbg( "PLC Name" + plcName, "info")

			// populate the new slice with the correct tags
			for _, v := range KK_Tag_DB.TagDatabase.Tags {
				if (v.PLCName == plcName && ( v.TagName == tagName || v.TagAddress == tagName ) ) {
					tmpTagDB = append(tmpTagDB, v)
				}
			}
		}
	}


	// return it as JSON
	json, err := json.Marshal(tmpTagDB)
	if err != nil {
		fmt.Println("error:", err)
		w.Write([]byte("JSON Marshal Error"))
	}
	w.Write(json)
}



//
// takes a plcname/tagname as 2 parameters
// returns the tag as json
//
func sendTagAsJSON(w http.ResponseWriter, r *http.Request) {

	// get the PLC name from the url
	plcname := mux.Vars(r)["plcname"]
	tagname := mux.Vars(r)["tagname"]

	// create a new slice of tags to return
	var tmpTagDB []KK_Tag_DB.TagObj

	// populate the new slice with the correct tags
	for _, v := range KK_Tag_DB.TagDatabase.Tags {
		if (v.PLCName == plcname && v.TagName == tagname ) {
			tmpTagDB = append(tmpTagDB, v)
		}
	}

	// return it as JSON
	json, err := json.Marshal(tmpTagDB)
	if err != nil {
		fmt.Println("error:", err)
		w.Write([]byte("JSON Marshal Error"))
	}
	w.Write(json)
}




// Handler for the server root
// this will eventually display an html page like the Node version
func displayMain(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Event Server"))
}




// handler for the test
// Echos back whatever was nested in the /test virtual dir
func testResponse(w http.ResponseWriter, r *http.Request) {
	val := mux.Vars(r)["val"]
	w.Write([]byte(fmt.Sprintf("Response %s", val)))
}



// handler for an error event
func errorEvent(w http.ResponseWriter, r *http.Request) {
	error := mux.Vars(r)["error"]
	fmt.Println("Error Event ")
	fmt.Println(error)
	fmt.Println(" ")
	w.Write([]byte(fmt.Sprintf("ErrorAckd %s", error)))
}

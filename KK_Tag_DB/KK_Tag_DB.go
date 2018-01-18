package KK_Tag_DB

import(
	//"time"
	"fmt"
	"strings"
	"os"
	//"os/exec"
	"errors"
	"text/tabwriter"
	//"kellestine.com/KKPLC_Gateway/KK_Logging"
	"kellestine.com/KKPLC_Gateway/KK_Globals"
)


/*
 *	This is the tag database
 *	the actual instance of it, and all methods,
 *	structs and definations
 */


// 
// structure for tags
// there will be a slice of these
//
type TagObj struct {
	PLCName string
	TagName string
	TagAddress string
	TagType string
	TagValue string
	IsWritable bool
	TimeStamp string
	Events []TagEvent
}

type TagEvent struct {
	EventName string
	TriggerType string
}


type EventQueueObj struct{
	EventName string
	PLCName string
	TagName string
	OldValue string
	NewValue string
}
var EventQueue chan EventQueueObj

// tag Database stuct, there will only be one instance of this
type tagDB struct{
	Tags []TagObj
	TagUpdateQueue chan string // processes tag updates to the tagDB
	TagSetQueue chan string // sets tags [called by external]
}


// this is the global tag database object
var TagDatabase tagDB



// apply updates to Tags
// figures out if they are gateway tags or virtual
// forwards the update to the correct place
// the Gateways SHOULD NOT ADD TO THIS QUEUE
// everything else should
func (tdb tagDB) QueueTagSet( updateStr string ) {
   tdb.TagSetQueue <- updateStr
}
// method to process the tag update queue
func ( tdb tagDB ) ProcessTagSetQueue(){
	for {
		queuedTagSetStr := <-tdb.TagSetQueue
		err := tdb.ProcessTagSet(queuedTagSetStr)
		if err != nil{
			KK_Globals.Dbg(err.Error(), "error")
		}
	}
}


// updates for the Gateways
// these update requests get sent to the gateways
// so they can be written to the PLCs
func QueueGatewayTagSet(updateStr string){
	GatewayTagSetQueue <- updateStr
}
var GatewayTagSetQueue chan string


// method to handle updating tags in the tag database
// this queue is populated by the PLC Gateways when 
// they get tag updates fromt their PLCS
// also the ProcessTagSetQueue() can add to this queue
func (tdb tagDB) QueueTagUpdate( updateStr string ) {
   tdb.TagUpdateQueue <- updateStr
}
// method to process the tag update queue
func ( tdb tagDB ) ProcessTagUpdateQueue(){
	for {
		queuedTagUpdateStr := <-tdb.TagUpdateQueue
		tdb.updateTagDB(queuedTagUpdateStr)
	}
}

// sets a system tag
func (tdb tagDB) SetSystemTag( tagName, tagValue string ) {
	updateStr := "TAGUPDATE: "+tagName+"="+tagValue+"@system"
	TagDatabase.QueueTagUpdate(updateStr)
}

// gets a tag's index from the tag database
func (tdb tagDB) GetTagIndex( plcName, tagName string) (tagDBIndex int, err error){

	for i, v := range TagDatabase.Tags {
		if ( v.PLCName == plcName && v.TagName == tagName ) {return i, err}
	}

	// return the dummy tag and the error
	return -1, errors.New("tag doesn't exit: " + plcName + " " + tagName)
}// getTagIndex


// gets a tag's index by tag Address
func (tdb tagDB) GetTagIndexByAddress( plcName, tagAddress string) (tagDBIndex int, err error){

	for i, v := range TagDatabase.Tags {
		if ( v.PLCName == plcName && v.TagAddress == tagAddress ) {return i, err}
	}

	// return the dummy tag and the error
	return -1, errors.New("tag doesn't exit: " + plcName + " " + tagAddress)
}// getTagIndex


func (tdb tagDB) GetTagNameFromAddress( plcName, tagAddress string) (tagName string, err error){

	for _, v := range TagDatabase.Tags {
		if ( v.PLCName == plcName && v.TagAddress == tagAddress ) {
			return v.TagName, err
		}
	}

	// not found
	return "-1", errors.New("tag doesn't exit: " + plcName + " " + tagAddress)
}//


//
// returns the value of a tag (as a string) or an error
//
func (tdb tagDB) GetTagValue( plcName, tagName string) (value string, err error){
	tagBind, err := tdb.GetTagIndex( plcName, tagName )
	if ( err == nil ){
		return tdb.Tags[tagBind].TagValue, err
	}else{
		return "", err
	}
}


// adds a new tag to the tag database
func (tdb tagDB) AddTag( tag TagObj) (err error) {
	
	// first check if tag already exists
	_, err = tdb.GetTagIndex( tag.PLCName, tag.TagName )
	if ( err == nil ){
		
		// tag exists, create a new error as the error
		// returned by GetTagIndex is worded in the opposite
		err = errors.New(tag.TagName + " Already exists")
		return err
	}

	// Add the new tag to the slice
	TagDatabase.Tags = append(TagDatabase.Tags, tag)
	return err
}


//
// adds a named event to a Tag's event []string
//
func (tdb tagDB) AddTagEvent( plcName, tagName string, event TagEvent) (err error) {
	
	// first check if tag exists
	i, er := tdb.GetTagIndex( plcName, tagName )
	if ( er != nil ){ return er }

	// check if the event already exists on this tag
	for _,e := range TagDatabase.Tags[i].Events {
		if e.EventName == event.EventName {

			// event exists, error out
			err = errors.New(event.EventName + " on " + tagName + " already exists")
			return err
		}
	}

	// Add the new event to the tag
	TagDatabase.Tags[i].Events = append(TagDatabase.Tags[i].Events, event)
	return err
}



//
// anything that wants to make an update to the tag database
// other than Gateways that are reporting their updates;
// goes thru this interface
// this is the only way to write to the PLCs and vTags
//
func (tdb tagDB) ProcessTagSet( updateStr string ) (err error) {

	// tag update messages should be in the form: tagname = value
	KK_Globals.Dbg("ProcessTagSet: " + updateStr, "info")

	val := strings.Index(updateStr, "TAGUPDATE: ")
	if ( val < 0 ){
		err = errors.New("ProcessTagSet: malformed update string: " + updateStr)
		return err
	}

	// split so we can ditch the TAGUPDATE: prefix
	splitter := strings.Split( updateStr, "TAGUPDATE: " )

	// get the index of the @ sign
	indexi := strings.Index(splitter[1], "@")
	if ( indexi > -1 ){
		// split on the @
		splitter = strings.Split( splitter[1], "@" )

		if len(splitter) < 2 {
			err = errors.New("ProcessTagSet: malformed update string: " + updateStr)
			return err	
		}

		// get the PLC name and the upate command
		updateCommand := splitter[0]
		plcGatewayName := splitter[1]

		// so tag updates could be in the form tagName = newValue
		// or tagName=newValue
		// switch to a single form here
		updateCommand = strings.Replace(updateCommand, " = ", "=", 1)

		// get the index of the equals sign
		index := strings.Index(updateCommand, "=")
		if ( index > -1 ){

			// split on the equals
			splitter = strings.Split( updateCommand, "=" )

			if len(splitter) < 2 {
				err = errors.New("ProcessTagSet: malformed update string: " + updateStr)
				return err	
			}

			// get the tag and val
			tagName := splitter[0]
			//tagVal := splitter[1]

/*
			if ( KK_Globals.Mode() == "debugging" ){
				fmt.Printf("Tag: %s \n", tagName)
				fmt.Printf("Value : %s \n", tagVal)
			}
*/

			// Add the tag to the db if it doesnt exists in the database yet
			// this means it's a non-configured tag
			tagBind, er := tdb.GetTagIndex( plcGatewayName, tagName )
			if ( er != nil ){

				var tag TagObj
				tag.PLCName = plcGatewayName
				tag.TagName = tagName

				if plcGatewayName == "virtual"{
					tag.IsWritable = true					
				}

				er = tdb.AddTag(tag)
				if ( err == nil ){
					// this error is raised if the tag already exists
					// which shouldn't happen because we are here because
					// the tag doesn't exist yet
				}
			}

			// get the tag
			tagBind, err = tdb.GetTagIndex( plcGatewayName, tagName )
			if ( err == nil ){

				// don't process tag sets for non-writable tags
				if TagDatabase.Tags[tagBind].IsWritable {

					// if this tag belongs to a Gateway send the update to the gateway
					if strings.Index(strings.ToLower(plcGatewayName), "virtual") > -1 {
						// virtual tag, queue a tagDB update
						TagDatabase.QueueTagUpdate(updateStr)
					}else{
						//Dbg("Sending tagset to PLC Gateway: " + updateStr, "info")

						// swap out the tag name for the tag address
						// because the PLC Gateway uses tag addresses
						updateCommand = strings.Replace(updateCommand, TagDatabase.Tags[tagBind].TagName, TagDatabase.Tags[tagBind].TagAddress, 1)

						tagSetStr := updateCommand + "@" + plcGatewayName
						QueueGatewayTagSet(tagSetStr)
					}
				}else{
					// error, tag is not writable
					//KK_Globals.Dbg("ProcessTagSet: " + tagName + " is not writable", "warning")
					err = errors.New("ProcessTagSet: " + tagName + " is not writable")
					return err					
				}
			}else{
				err = errors.New("ProcessTagSet: Tag: " + tagName + " doesn't exist")
				return err	
			}
		}
	}else{
		// malformed tagset command
		err = errors.New("ProcessTagSet: malformed update string: " + updateStr)
		return err
	}
	return err
}



// updates the tag database
// called by TagDatabase.ProcessTagUpdateQueue()
// tag updates from the PLC gateway are queued there
// fires tag events
// does not update PLC tags, only the tag database
//
// updateStr can contain a tagAddress or a tag Name
//
func (tdb tagDB) updateTagDB( updateStr string ){

	// tag update messages should be in the form: tagname = value
	KK_Globals.Dbg("updateTagDB: " + updateStr, "info")

	// split so we can ditch the TAGUPDATE: prefix
	splitter := strings.Split( updateStr, "TAGUPDATE: " )


	// get the index of the @ sign
	indexi := strings.Index(splitter[1], "@")
	if ( indexi > -1 ){
		// split on the @
		splitter = strings.Split( splitter[1], "@" )

		// get the PLC name and the upate command
		updateCommand := splitter[0]
		plcGatewayName := splitter[1]

		// so tag updates could be in the form tagName = newValue
		// or tagName=newValue
		// switch to a single form here
		updateCommand = strings.Replace(updateCommand, " = ", "=", 1)

		// get the index of the equals sign
		index := strings.Index(updateCommand, "=")
		if ( index > -1 ){

			// split on the equals
			splitter = strings.Split( updateCommand, "=" )

			// get the tag and val
			tagName := splitter[0]
			tagVal := splitter[1]

/*
			if ( KK_Globals.Mode() == "debugging" ){
				fmt.Printf("Tag: %s \n", tagName)
				fmt.Printf("Value : %s \n", tagVal)
			}
*/

			// check if we got a tag addresss, if so translate it to a tag name
			maybeTagName, er := tdb.GetTagNameFromAddress(plcGatewayName, tagName)
			if ( er == nil ){
				tagName = maybeTagName
			}

			// Add the tag to the db if it doesnt exists in the database yet
			// this means it's a non-configured tag
			tagBind, err := tdb.GetTagIndex( plcGatewayName, tagName )
			if ( err != nil ){

				var tag TagObj
				tag.PLCName = plcGatewayName
				tag.TagName = tagName

				err:= tdb.AddTag(tag)
				if ( err == nil ){
					// this error is raised if the tag already exists
				}
			}

			// update the tag value in the database
			tagBind, err = tdb.GetTagIndex( plcGatewayName, tagName )
			if ( err == nil ){

				// process all tag events bound to this tag
				for _,e := range TagDatabase.Tags[tagBind].Events {
					go processTagEvent(plcGatewayName, tagName, e, TagDatabase.Tags[tagBind].TagValue, tagVal)
				}

				// update the tag value
				TagDatabase.Tags[tagBind].TagValue = tagVal
				TagDatabase.Tags[tagBind].TimeStamp = KK_Globals.GetNow()


				if ( KK_Globals.Mode() == "tagdb" ){

				    // reprint the tag database
					tdb.PrintTagDB()
				}
			}
		}
	}
}


//
// every tag that has one or more events in their event slice
// will call this function on tag update
//
func processTagEvent( plcName, tagName string, event TagEvent, previousValue, newValue string ){
	if ( KK_Globals.Mode() == "debugging" ){
		fmt.Println(event.EventName + " triggered on " + tagName + "@" + plcName)
		fmt.Println("Previous Value: [" + previousValue + "]")
		fmt.Println("New Value: [" + newValue + "]")
	}

	// check if the correct condition occured, example: transition high
	processEvent := false

	// check for init
	if previousValue == "" {
		// Tag Init
		if ( KK_Globals.Mode() == "debugging" ){fmt.Println("Tag Init Detected")}
		if strings.ToLower(event.TriggerType) == "init"{
			//fmt.Println("Tag Init event firing")
			processEvent = true
		}

	}else if previousValue == newValue {
		// previousValue == newValue
		if ( KK_Globals.Mode() == "debugging" ){fmt.Println("previousValue = newValue, likely missed a transition")}

	}else if newValue == "True" {
		// Transition high
		if ( KK_Globals.Mode() == "debugging" ){fmt.Println("Tag transition high detected")}
		if strings.ToLower(event.TriggerType) == "transition high"{
			//fmt.Println("Tag transition high event firing")
			processEvent = true
		}

	}else if newValue == "False" {
		// Transition low
		if ( KK_Globals.Mode() == "debugging" ){fmt.Println("Tag transition low detected")}
		if strings.ToLower(event.TriggerType) == "transition low"{
			//fmt.Println("Tag transition low event firing")
			processEvent = true
		}

	}else{
		// none of the above
		if ( KK_Globals.Mode() == "debugging" ){fmt.Println("Tag transition of non-boolean tag detected")}
		processEvent = true
	}

	// queue the event for processing if the condition was met
	if processEvent {
		var eventQueueObj EventQueueObj
		eventQueueObj.EventName = event.EventName
		eventQueueObj.PLCName = plcName
		eventQueueObj.TagName = tagName
		eventQueueObj.OldValue = previousValue
		eventQueueObj.NewValue = newValue
		EventQueue <- eventQueueObj
	}
}



//
// prints the tag database to the console
//
func (tdb tagDB) PrintTagDB(){

	// clear the terminal window
    KK_Globals.ClearTerminal()

	fmt.Println("---------------------------------------------------------------")
	fmt.Println("     Tag Database        [press c then enter to exit]          ")
	fmt.Println("---------------------------------------------------------------")
	fmt.Println("            PLC Name|                 Tag|           Tag Value|")
	fmt.Println("---------------------------------------------------------------")

	w := tabwriter.NewWriter(os.Stdout, 20, 1, 0, ' ', tabwriter.AlignRight|tabwriter.Debug)
	for _, v := range TagDatabase.Tags {

		displayName := v.TagAddress
		if v.TagName != "" { displayName = v.TagName }

		fmt.Fprintln(w,  v.PLCName + "\t" + displayName + "\t" + v.TagValue + "\v")
	}

	w.Flush()
	fmt.Println("---------------------------------------------------------------")
}// PrintTagDB



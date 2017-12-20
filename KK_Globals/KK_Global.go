package KK_Globals

import(
	"time"
	"fmt"
	"strings"
	"os"
	"os/exec"
	"errors"
	"text/tabwriter"
)


// consts to start the gateway process
const GatewayPath = "C:\\Windows\\System32\\cscript.exe"
var GatewayArgs = []string{"test.vbs"}


// 
// structure for tags
// there will be a slice of these
//
type TagObj struct {
	PLCName string
	TagName string
	TagAlias string
	TagType string
	TagValue string
	Writeable bool
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
	TagUpdateQueue chan string
}

// method to handle updating tags
func (tdb tagDB) QueueTagUpdate( updateStr string ) {
   tdb.TagUpdateQueue <- updateStr
}

// method to process the tag update queue
func ( tdb tagDB ) ProcessTagUpdateQueue(){
	for {
		queuedTagUpdateStr := <-tdb.TagUpdateQueue
		//fmt.Println("Processing: " + queuedTagUpdateStr)
		tdb.updateTagDB(queuedTagUpdateStr)
	}
}

// gets a tag's index from the tag database
func (tdb tagDB) GetTagIndex( plcName, tagName string) (tagDBIndex int, err error){

	for i, v := range TagDatabase.Tags {
		if ( v.PLCName == plcName && v.TagName == tagName ) {return i, err}
	}

	// return the dummy tag and the error
	//fmt.Println("tag doesn't exit: " + plcName + " " + tagName)
	return -1, errors.New("tag doesn't exit: " + plcName + " " + tagName)
}// getTagIndex


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
	
	// first check if tag already exists
	i, er := tdb.GetTagIndex( plcName, tagName )
	if ( er != nil ){ return er }

	// check if the event already exists on this tag
	found := false
	for _,e := range TagDatabase.Tags[i].Events {
		if e.EventName == event.EventName {
			found = true
			break;
		}
	}
	if found {
		err = errors.New(event.EventName + " on " + tagName + " already exists")
		return err
	}

	// Add the new tag to the slice
	TagDatabase.Tags[i].Events = append(TagDatabase.Tags[i].Events, event)
	return err
}



// this is the global tag database object
var TagDatabase tagDB


// controlls the current CLI mode
var Mode string




// updates the tag database
// called by TagDatabase.ProcessTagUpdateQueue()
func (tdb tagDB) updateTagDB( updateStr string ){

	// tag update messages should be in the form: tagname = value
	if ( Mode == "debugging" ){ fmt.Println("Tag update recieved")}

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

			if ( Mode == "debugging" ){
				fmt.Printf("Tag: %s \n", tagName)
				fmt.Printf("Value : %s \n", tagVal)
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
				TagDatabase.Tags[tagBind].TimeStamp = GetNow()


				if ( Mode == "tagdb" ){

					// clear the cmd.exe shell
				    cmd := exec.Command("cmd", "/c", "cls")
				    cmd.Stdout = os.Stdout
				    cmd.Run()				

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
	if ( Mode == "debugging" ){
		fmt.Println(event.EventName + " triggered on " + tagName + "@" + plcName)
		fmt.Println("Previous Value: [" + previousValue + "]")
		fmt.Println("New Value: [" + newValue + "]")
	}

	processEvent := false

	// check for init
	if previousValue == "" {
		// Tag Init
		if ( Mode == "debugging" ){fmt.Println("Tag Init Detected")}
		if strings.ToLower(event.TriggerType) == "init"{
			//fmt.Println("Tag Init event firing")
			processEvent = true
		}

	}else if previousValue == newValue {
		// previousValue == newValue
		if ( Mode == "debugging" ){fmt.Println("previousValue = newValue, likely missed a transition")}

	}else if newValue == "True" {
		// Transition high
		if ( Mode == "debugging" ){fmt.Println("Tag transition high detected")}
		if strings.ToLower(event.TriggerType) == "transition high"{
			//fmt.Println("Tag transition high event firing")
			processEvent = true
		}

	}else if newValue == "False" {
		// Transition low
		if ( Mode == "debugging" ){fmt.Println("Tag transition low detected")}
		if strings.ToLower(event.TriggerType) == "transition low"{
			//fmt.Println("Tag transition low event firing")
			processEvent = true
		}

	}else{
		// none of the above
		if ( Mode == "debugging" ){fmt.Println("Tag transition of non-boolean tag detected")}
		processEvent = true
	}

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

	fmt.Println("---------------------------------------------------------------")
	fmt.Println("     Tag Database        [press c then enter to exit]          ")
	fmt.Println("---------------------------------------------------------------")
	fmt.Println("            PLC Name|            Tag Name|           Tag Value|")
	fmt.Println("---------------------------------------------------------------")

	w := tabwriter.NewWriter(os.Stdout, 20, 1, 0, ' ', tabwriter.AlignRight|tabwriter.Debug)
	for _, v := range TagDatabase.Tags {
		fmt.Fprintln(w,  v.PLCName + "\t" + v.TagName + "\t" + v.TagValue + "\v")
	}

	w.Flush()
	fmt.Println("---------------------------------------------------------------")
}// PrintTagDB




func GetNow() string {

	// RFC1123     = "Mon, 11 Dec 2017 15:04:05 EST"
	now := time.Now()
	t := now.Format(time.RFC1123)
	
	return t
}


func Dbg( txt, severity string ){
	if ( Mode == "debugging" ){fmt.Println(txt)}
}

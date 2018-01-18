package KK_Config
import (
	"io/ioutil"
	"errors"
	"strings"
	"encoding/json"
	"kellestine.com/KKPLC_Gateway/KK_PLC_Wrapper"
	"kellestine.com/KKPLC_Gateway/KK_Globals"
	"kellestine.com/KKPLC_Gateway/KK_Tag_DB"
)


/*
 *
 *	this package loads the JSON config file
 * 	and uses that data to initilize the system
 *	the loaded config is stored in the exported "Config"
 *
 */


// load the JSON into this
var Config configFile
var ConfigLoaded bool = false
var SystemInitilized bool = false

// structs to match the config structure
// courtesy of https://mholt.github.io/json-to-go/
type configFile struct {
	KeyVals []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"keyVals"`
/*	
	Virtuals []struct {
		VirtualPLC string `json:"virtualPLC"`
		TagName    string `json:"tagName"`
		IsWritable bool   `json:"isWritable"`
		Events     []struct {
			Name    string `json:"name"`
			Trigger string `json:"trigger"`
		} `json:"events"`
	} `json:"virtuals"`
*/	
	Gateways []struct {
		PlcName        string   `json:"plcName"`
		Enabled        bool   `json:"enabled"`
		ConnectAddress string   `json:"connectAddress"`
		Path           string   `json:"path"`
		Args           []string `json:"args"`
		Tags           []struct {
			TagName   string `json:"tagName"`
			TagAddress     string `json:"tagAddress"`
			TagType   string `json:"tagType"`
			IsWritable bool   `json:"isWritable"`
			Events    []struct {
				Name    string `json:"name"`
				Trigger string `json:"trigger"`
			} `json:"events,omitempty"`
		} `json:"tags"`
	} `json:"gateways"`
}



//
// Loads ./KK_Config/config.json to "Config"
// calls initSystem() if the system hasn't be initilized yet
// returns an error or nil error
//
func LoadConfig() (err error) {

	// load in the JSON Config file
	byteSlice, err := ioutil.ReadFile("./KK_Config/config.json")
	if err != nil {
		return errors.New("LoadConfig File Read Error: " + err.Error())
	}else{


		// validate
		if json.Valid(byteSlice){

			err := json.Unmarshal(byteSlice, &Config)
			if err != nil {

				return errors.New("LoadConfig JSON Parse Error: " + err.Error())
			}else{

				KK_Globals.Dbg("LoadConfig: Config Loaded", "info")
				ConfigLoaded = true

				// init the system
				if !SystemInitilized {
					err = initSystem()
				}

				// all good, return nil error
				return err
			}
		}else{
			return errors.New("LoadConfig Error: config.json is not valid JSON")
		}
	}
}


func GetConfigValue( configKey string ) ( configVal string, err error ){
	if !ConfigLoaded {
		return "", errors.New("GetConfigValue Error: config has not been loaded yet")
	}

	for _, keyVal := range Config.KeyVals {

		if keyVal.Key == configKey{
			return keyVal.Value, nil
		}
	}

	return "", errors.New("GetConfigValue Error: configKey not found")
}


//
// initilizes the system using the data from the config file
// todo: get the function to only initilized new stuff,
// export the function. Then it can be called to load new config 
// stuff live
//
func initSystem() (err error) {

	if !ConfigLoaded {
		return errors.New("initSystem Error: config has not been loaded yet")
	}

/*
	//
	// load the virtual tags and events
	//
	for _, virtual := range Config.Virtuals {

		KK_Globals.Dbg("Load Virtual Tags: " + virtual.VirtualPLC, "info")
		KK_Globals.Dbg("Load Virtual Tags: " + virtual.TagName, "info")

		// loop the events and build the expected event slice
		var eventSlice []KK_Tag_DB.TagEvent
		for _, event := range virtual.Events {

			// new event of correct type
			var e KK_Tag_DB.TagEvent
			e.EventName = event.Name
			e.TriggerType = event.Trigger

			// add to slice
			eventSlice = append( eventSlice, e )
		}

		// build struct expected by the AddTag method
		var tag KK_Tag_DB.TagObj
		tag.PLCName = virtual.VirtualPLC
		tag.TagName = virtual.TagName
		//tag.Alias = virtual.TagName
		tag.Events = eventSlice
		tag.IsWritable = true
		if !virtual.IsWritable{
			tag.IsWritable = false
		}

		// add the tag to the tag database
		KK_Tag_DB.TagDatabase.AddTag( tag )
	}
*/

	//
	// load the Gateways
	//
	for _, gateway := range Config.Gateways {

		KK_Globals.Dbg("Load Gateway: PLC: " + gateway.PlcName, "info")

		// skip if disabled	
		if gateway.Enabled {
			KK_Globals.Dbg("load Gateway: Address: " + gateway.ConnectAddress, "info")

			// create a slice tag objects
			var tagSlice []KK_PLC_Wrapper.GatewayTag

			// loop each tag in this gateway
			for _, tagObj := range gateway.Tags {
				var tag KK_PLC_Wrapper.GatewayTag
				
				tag.Alias = tagObj.TagName
				tag.TagName = tagObj.TagAddress
				tag.IsString = false
				if strings.ToLower(tagObj.TagType) == "string"{
					tag.IsString = true
				}
				tag.IsWritable = true
				if !tagObj.IsWritable{
					tag.IsWritable = false
				}
				// loop each event on this tag
				var eventSlice []KK_Tag_DB.TagEvent
				for _, eventObj := range tagObj.Events {
					var e KK_Tag_DB.TagEvent
					e.EventName = eventObj.Name
					e.TriggerType = eventObj.Trigger

					// add event to slice
					eventSlice = append( eventSlice, e )
				}

				// add event slice to tag
				tag.Events = eventSlice

				// add tag to tag slice
				tagSlice = append( tagSlice, tag )
			}

			//
			// create the initilization arguments for the gateway
			//
			var initObj KK_PLC_Wrapper.GatewayInit
			initObj.PlcName = gateway.PlcName
			initObj.ConnectAddress = gateway.ConnectAddress
			initObj.TagSlice = tagSlice
			initObj.Path = gateway.Path
			initObj.Args = gateway.Args

			if strings.ToLower(gateway.ConnectAddress) != "virtual"{
				
				// start the gateway
				// this call blocks
				go KK_PLC_Wrapper.StartAGateway(initObj)					
			}else{
				

				// virtual gateway, just added the tags to the db
				KK_Globals.Dbg("Load Gateway:" + gateway.PlcName + " is virtual", "info")
				for _, tag := range initObj.TagSlice {

					var tagDBTag KK_Tag_DB.TagObj
					tagDBTag.PLCName = gateway.PlcName
					tagDBTag.TagName = tag.TagName
					tagDBTag.IsWritable = tag.IsWritable
					tagDBTag.Events = tag.Events

					KK_Tag_DB.TagDatabase.AddTag( tagDBTag )
				}
			}

		}else{
			KK_Globals.Dbg("Load Gateway: PLC " + gateway.PlcName + " is disabled. Skipping", "info")
		}
	}

	SystemInitilized = true
	return err
}
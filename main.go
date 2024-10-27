package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("no .env file found")
	}
}

func main() {
	http.HandleFunc("/lab", getLabStatus)
	http.HandleFunc("/door", getDoorStatus)
	http.HandleFunc("/favicon.ico", doNothing)

	err := http.ListenAndServe(":3333", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

type EntityAttributes struct {
	Editable     bool   `json:"editable"`
	FriendlyName string `json:"friendly_name"`
}

type EntityContext struct {
	Id       string `json:"id"`
	ParentId string `json:"parent_id"`
	UserId   string `json:"user_id"`
}

type HassEntityState struct {
	EntityId    string           `json:"entity_id"`
	State       string           `json:"state"`
	Attributes  EntityAttributes `json:"attributes"`
	LastChanged string           `json:"last_changed"`
	LastUpdated string           `json:"last_updated"`
	Context     EntityContext    `json:"context"`
}

type StrippedHassEntityState struct {
	State       string `json:"state"`
	LastChanged string `json:"last_changed_utc"`
	LastUpdated string `json:"last_updated_utc"`
}

var date_lastupdated_utc time.Time
var last_request time.Time
var cached_hass_state StrippedHassEntityState

func getLabStatus(w http.ResponseWriter, r *http.Request) {
	if (strings.Split(r.RemoteAddr, ":")[0] == "127.0.0.1" || strings.Split(r.RemoteAddr, "::1]")[0] == "[") && r.Header.Get("X-Forwarded-For") != "" { //ipv4 and ipv6 localhost
		fmt.Printf("got / request from %v\n", r.Header.Get("X-Forwarded-For"))
	} else {
		fmt.Printf("got / request from %v\n", r.RemoteAddr)
	}

	if (time.Now().UTC().Unix() - last_request.Unix()) <= 30 {
		fmt.Printf("cache hit, last used at %v\n", last_request.Format(time.RFC3339))
		p, _ := json.Marshal(cached_hass_state)

		fmt.Printf("returning %+v\n", cached_hass_state)

		//set response headers and send it back
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(p)
		return
	}

	//prep http get to metalab homeassistant
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://10.20.30.97/api/states/input_boolean.lab_is_on", nil)
	if err != nil {
		fmt.Printf("error while building rest request to hass: %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	//set required headers
	req.Header.Set("Authorization", "Bearer "+os.Getenv("HOMEASSISTANT_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	//actually send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error while sending request to hass: %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	//close request and read the body
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error while reading response body from hass: %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	//create and fill the new object
	var hassEntityStateResponse HassEntityState
	json.Unmarshal(body, &hassEntityStateResponse)

	date_lastchanged_utc, err := time.Parse(time.RFC3339, hassEntityStateResponse.LastChanged)
	if err != nil {
		fmt.Printf("error while parsing lastchanged date from hass: %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	} else {
		date_lastupdated_utc = time.Now().UTC()
		fmt.Printf("got response from hass at %v\n", date_lastupdated_utc.Format(time.RFC3339))
	}

	//cached hass state is used to globally store it
	cached_hass_state = StrippedHassEntityState{hassEntityStateResponse.State, date_lastchanged_utc.Format(time.RFC3339), date_lastupdated_utc.Format(time.RFC3339)}
	p, _ := json.Marshal(cached_hass_state)

	fmt.Printf("returning %+v\n", cached_hass_state)

	last_request = time.Now().UTC()

	//set response headers and send it back
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(p)
}

func getDoorStatus(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func doNothing(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

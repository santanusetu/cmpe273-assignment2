package main

import (
"io"
"fmt"
"log"
"strings"
"errors"
"net/url"
"net/http"
"io/ioutil"
"encoding/json"
"gopkg.in/mgo.v2"
"gopkg.in/mgo.v2/bson"
"github.com/julienschmidt/httprouter"
)

type (
location struct {
Id                           bson.ObjectId              `json:"id" bson:"id"`
Name                         string                     `json:"name" bson:"name"`
Address                      string                     `json:"address" bson:"address"`
City                         string                     `json:"city" bson:"city"`
State                        string                     `json:"state" bson:"state"`
Zip                          string                     `json:"zip" bson:"zip"`
Coordinate points                      `json:"coordinate" bson:"coordinate"`
}

points struct {
Lat                         float64                     `json:"lat" bson:"lat"`
Lng                         float64                     `json:"lng" bson:"lng"`
}
)



type locationController struct{
session *mgo.Session
}

 
func newLocationController(s *mgo.Session) *locationController {
return &locationController{s}
}


// Function to initiate Mongo Session
func getMgoSession() *mgo.Session {
session, err := mgo.Dial("mongodb://santanu:santanu@ds045054.mongolab.com:45054/assignment2")
if err != nil {
panic(err)
}
return session
}


// Function to update customer location
func updateConsumerLocation(cc locationController, id string, contents io.Reader) (location, error) {

consumer, err := fetchLocationById(cc, id)
if err != nil {
return location{}, err
}

updLocation := location{}
updLocation.Id = consumer.Id
updLocation.Name = consumer.Name
json.NewDecoder(contents).Decode(&updLocation)

err = getCoordinates(&updLocation)
if err != nil {
return location{}, err
}

objId := bson.ObjectIdHex(id)
conn := cc.session.DB("assignment2").C("locations")
err = conn.Update(bson.M{"id": objId}, updLocation)
if err != nil {
log.Println(err)
return location{}, errors.New("Invalid id provided")
}
return updLocation, nil
}



// Function to fetch the location based upon the id
func fetchLocationById(cc locationController, id string) (location, error) {

if !bson.IsObjectIdHex(id) {
return location{}, errors.New("Invalid Consumer ID")
}
objId := bson.ObjectIdHex(id)
LocationModel := location{}
conn := cc.session.DB("assignment2").C("locations")
err := conn.Find(bson.M{"id": objId}).One(&LocationModel)
if err != nil {
return location{}, errors.New("Provided Id doesn't exists in the database")
}

return LocationModel, nil
}



// Function to fetch the location of the customer 
func getCoordinates(LocationModel *location) error {
client := &http.Client{}
address := LocationModel.Address + "+" + LocationModel.City + "+" + LocationModel.State + "+" + LocationModel.Zip;

// Building the google url for querying
urlNew := "http://maps.google.com/maps/api/geocode/json?address="

urlNew += url.QueryEscape(address)
urlNew += "&sensor=false"
req, err := http.NewRequest("GET", urlNew , nil)
res, err := client.Do(req)

if err != nil {
return err
}

body, err := ioutil.ReadAll(res.Body)
if err != nil {
return err
}
defer res.Body.Close()

var contents map[string]interface{}
err = json.Unmarshal(body, &contents)
if err != nil {
return err
}

if !strings.EqualFold(contents["status"].(string), "OK") {
return errors.New("Location not available for this particular customer")
}

results := contents["results"].([]interface{})
location := results[0].(map[string]interface{})["geometry"].(map[string]interface{})["location"]

LocationModel.Coordinate.Lat = location.(map[string]interface{})["lat"].(float64)
LocationModel.Coordinate.Lng = location.(map[string]interface{})["lng"].(float64)

if err != nil {
return err
}

return nil
}




//Function to put location object in collection 
func (cc locationController) CreateLocation(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
LocationModel := location{}
json.NewDecoder(req.Body).Decode(&LocationModel)
LocationModel.Id = bson.NewObjectId()
err := getCoordinates(&LocationModel)
if err != nil {
log.Println(err)
}

//Saving data in mongoDB  
conn := cc.session.DB("assignment2").C("locations")
err = conn.Insert(LocationModel)
if err != nil {
log.Println(err)
}

locationJSON, _ := json.Marshal(LocationModel)
if err != nil {
rw.Header().Set("Content-Type", "plain/text")
rw.WriteHeader(400)
fmt.Fprintf(rw, "%s\n", err)
} else {
rw.Header().Set("Content-Type", "application/json")
rw.WriteHeader(201)
fmt.Fprintf(rw, "%s\n", locationJSON)
}
}


// Function to get location from the json
func (cc locationController) GetLocation(rw http.ResponseWriter, _ *http.Request, param httprouter.Params) {
location, err := fetchLocationById(cc, param.ByName("id"))
if err != nil {
log.Println(err)
}

locationJSON, _ := json.Marshal(location)
if err != nil {
rw.Header().Set("Content-Type", "plain/text")
rw.WriteHeader(400)
fmt.Fprintf(rw, "%s\n", err)
} else {
rw.Header().Set("Content-Type", "application/json")
rw.WriteHeader(200)
fmt.Fprintf(rw, "%s\n", locationJSON)
}
}




// Function to update customer location 
func (cc locationController) UpdateLocation(rw http.ResponseWriter, req *http.Request, param httprouter.Params) {
updatedUsr, err := updateConsumerLocation(cc, param.ByName("id"), req.Body)
if err != nil {
log.Println(err)
}

locationJSON, _ := json.Marshal(updatedUsr)
if err != nil {
rw.Header().Set("Content-Type", "plain/text")
rw.WriteHeader(400)
fmt.Fprintf(rw, "%s\n", err)
} else {
rw.Header().Set("Content-Type", "application/json")
rw.WriteHeader(201)
fmt.Fprintf(rw, "%s\n", locationJSON)
}
}




// Function  to remove customer from collection
func (cc locationController) RemoveLocation(rw http.ResponseWriter, _ *http.Request, param httprouter.Params) {

location, err := fetchLocationById(cc, param.ByName("id"))
if err != nil {
log.Println(err)
log.Println(location)
}

objId := bson.ObjectIdHex(param.ByName("id"))
conn := cc.session.DB("assignment2").C("locations")
err = conn.Remove(bson.M{"id": objId})
if err != nil {
log.Println(err)
}
rw.Header().Set("Content-Type", "plain/text")
if err != nil {
rw.WriteHeader(400)
fmt.Fprintf(rw, "%s\n", err)
} else {
rw.WriteHeader(200)
fmt.Fprintf(rw, "Customer ID=%s is deleted from the database", param.ByName("id"))
}
}




func main() {
router := httprouter.New()
locationController := newLocationController(getMgoSession())

router.GET("/locations/:id", locationController.GetLocation)
router.POST("/locations", locationController.CreateLocation)
router.PUT("/locations/:id", locationController.UpdateLocation)
router.DELETE("/locations/:id", locationController.RemoveLocation)

fmt.Println("Server running on port 8080")
log.Fatal(http.ListenAndServe(":8080", router))
}

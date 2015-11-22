package main
import (
   	"fmt"
    "httprouter"
    "net/http"
    "encoding/json"
    "log"
    "io/ioutil"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "math/rand"
    "strconv"
    "strings"
    "time"
    "bytes"
)

type User struct{
	UserId string
	Name string
	UserAddress Address
}

type Address struct{
	Address string
	City string
	State string
	Zip string
	Coordinates Location
}

type Location struct{
	Latitude float64
	Longitude float64
}

type AddLocationRequest struct{
	Name string
	Address string
	City string
	State string
	Zip string
}

type UpdateLocationRequest struct{
	Address string
	City string
	State string
	Zip string
}

type CreateTripRequest struct{
	Starting_from_location_id string
	Location_ids []string
}

type Trip struct{
	Id string
	Status string
	Starting_from_location_id string
	Best_route_location_ids []string
	Total_uber_costs float64
	Total_uber_duration float64
	Total_distance float64
	Uber_wait_time_eta int
	Next int
}

type node struct{
	Fare float64
	Distance float64
	Duration float64
}

type CreateTripResponse struct{
	Id string
	Status string
	Starting_from_location_id string
	Best_route_location_ids []string
	Total_uber_costs float64
	Total_uber_duration float64
	Total_distance float64
}

type UpdateTripResponse struct{
	Id string
	Status string
	Starting_from_location_id string
	Next_destination_location_id string
	Best_route_location_ids []string
	Total_uber_costs float64
	Total_uber_duration float64
	Total_distance float64
	Uber_wait_time_eta int
}

type JsonUber struct{
	Start_latitude string `json:"start_latitude"`
	Start_longitude string `json:"start_longitude"`
	End_latitude string `json:"end_latitude"`
	End_longitude string `json:"end_longitude"`
	Product_id string `json:"product_id"`
}

func getCoordinates(a *Address){
	addressString := strings.Replace(a.Address+"+"+a.City+"+"+a.State+"+"+a.Zip, " ", "%20", -1)
   	resp, err := http.Get("http://maps.google.com/maps/api/geocode/json?address="+addressString+"&sensor=false")
	if(err == nil){
	    body, err := ioutil.ReadAll(resp.Body)
	    if(err == nil) {
	        var data interface{}
	        json.Unmarshal(body, &data)
	        var m = data.(map[string] interface{})            
	        var articles = m["results"].([]interface{})[0].(map[string]interface{})["geometry"].(map[string]interface{})["location"]
	        lat := articles.(map[string]interface{})["lat"].(float64)
	        lng := articles.(map[string]interface{})["lng"].(float64)
	        a.Coordinates.Latitude = lat
	        a.Coordinates.Longitude = lng
	    } else {
	        fmt.Println(err)
	    }
	} else {
	    fmt.Println(err)
	}
}

func random(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}

func arrContains(arr []int, element int)(string){
	for i:=0;i<len(arr);i++{
		if arr[i] == element{
			return "true"
		}
	}
	return "false"
}

func createTrip(rw http.ResponseWriter, req *http.Request, p httprouter.Params){
	CreateTripRequest := new(CreateTripRequest)
	decoder := json.NewDecoder(req.Body)
	error := decoder.Decode(&CreateTripRequest)
	if error != nil {
		log.Println(error.Error())
		http.Error(rw, error.Error(), http.StatusInternalServerError)
		return
	}
	destinationsNum := len(CreateTripRequest.Location_ids)+1
	nodeMatrix := make([][]node, destinationsNum)
	var lat,lng float64
	locId := ""
	locArr := make([]Location, destinationsNum)
	for i:=0;i<destinationsNum;i++{
		location := new(Location)
		if(i==0){
			locId = CreateTripRequest.Starting_from_location_id
		}else{
			locId = CreateTripRequest.Location_ids[i-1]
		}
		resp, err := http.Get("http://localhost:8000/locations/"+locId)
		if(err == nil){
			body, err := ioutil.ReadAll(resp.Body)
		    if(err == nil) {
		        var data interface{}
		        json.Unmarshal(body, &data)
		        var m = data.(map[string] interface{})
		        var coordinates = m["UserAddress"].(map[string]interface{})["Coordinates"]
		        lat = coordinates.(map[string]interface{})["Latitude"].(float64)
		        lng = coordinates.(map[string]interface{})["Longitude"].(float64)
		        location.Latitude = lat
		        location.Longitude = lng
		    } else {
		        fmt.Println(err)
		    }
		} else {
		    fmt.Println(err)
		}
		locArr[i] = *location
	}
	for i:=0;i<destinationsNum;i++{
		nodeMatrix[i] = make([]node, destinationsNum)
		for j:=0;j<destinationsNum;j++ {
			if(i!=j){
				fare, distance ,duration := uberGetCost(locArr[i].Latitude, locArr[i].Longitude, locArr[j].Latitude, locArr[j].Longitude)
				nodeMatrix[i][j].Fare = fare
				nodeMatrix[i][j].Distance = distance
				nodeMatrix[i][j].Duration = duration
			}
		}
	}
	pathArr := make([]string,destinationsNum)
	var min float64 = 0
	var minIndex int = 0
	k := 0
	l:= 0
	var totalFare float64 = 0
	var totalDuration float64 = 0
	var totalDistance float64 =0
	pathCoveredArr := make([]int,destinationsNum)
	for k<destinationsNum{
		pathCoveredArr[k] = l
		min = 10000
		for j:=0;j<destinationsNum;j++ {
			if(minIndex!=j && arrContains(pathCoveredArr,j)=="false"){
				if nodeMatrix[l][j].Distance < min{
					min = nodeMatrix[l][j].Distance	
					totalFare = totalFare + nodeMatrix[l][j].Fare
					totalDuration = totalDuration + nodeMatrix[l][j].Duration
					totalDistance = totalDistance + nodeMatrix[l][j].Distance
					minIndex = j
				}			
			}
		}
		l = minIndex
		k = k+1
	}
	for i:=1;i<destinationsNum;i++{
		if(i==0){
			pathArr[i] = CreateTripRequest.Starting_from_location_id
		}else{
			pathArr[i] = CreateTripRequest.Location_ids[pathCoveredArr[i]-1]	
		}
	}
    trip := Trip{}
    trip.Id = strconv.Itoa(random(1, 100))
    trip.Status = "planning"
	trip.Starting_from_location_id = CreateTripRequest.Starting_from_location_id
	trip.Best_route_location_ids = pathArr[1:destinationsNum]
	trip.Total_uber_costs = totalFare
	trip.Total_uber_duration = totalDuration
	trip.Total_distance = totalDistance
	trip.Uber_wait_time_eta = 0
	trip.Next = 0
	session, err := mgo.Dial("mongodb://sejal:1234@ds045064.mongolab.com:45064/planner")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("planner").C("trip")
	err = c.Insert(&trip)
	if err != nil {
	        log.Fatal(err)
	}
	result := Trip{}
	err = c.Find(bson.M{"id":trip.Id}).One(&result)
	if err != nil {
	        log.Fatal(err)
	}
	createTripResponse := CreateTripResponse{}
    createTripResponse.Id = trip.Id
    createTripResponse.Status = trip.Status
	createTripResponse.Starting_from_location_id = CreateTripRequest.Starting_from_location_id
	createTripResponse.Best_route_location_ids = pathArr[1:destinationsNum]
	createTripResponse.Total_uber_costs = totalFare
	createTripResponse.Total_uber_duration = totalDuration
	createTripResponse.Total_distance = totalDistance
	
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusCreated)
	tripResponse, err := json.Marshal(createTripResponse)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprint(rw, string(tripResponse))
}

func getTrip(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	tripId := p.ByName("trip_id")
	session, err := mgo.Dial("mongodb://sejal:1234@ds045064.mongolab.com:45064/planner")
    if err != nil {
            panic(err)
    }
    defer session.Close()
    session.SetMode(mgo.Monotonic, true)
    c := session.DB("planner").C("trip")
    trip := Trip{}
    err = c.Find(bson.M{"id":tripId}).One(&trip)
    if err != nil {
            fmt.Fprint(rw, "Data not found")
            return
    }
    
	createTripResponse := CreateTripResponse{}
    createTripResponse.Id = trip.Id
    createTripResponse.Status = trip.Status
	createTripResponse.Starting_from_location_id = trip.Starting_from_location_id
	createTripResponse.Best_route_location_ids = trip.Best_route_location_ids
	createTripResponse.Total_uber_costs = trip.Total_uber_costs
	createTripResponse.Total_uber_duration = trip.Total_uber_duration
	createTripResponse.Total_distance = trip.Total_distance
	getTripJson, err := json.Marshal(createTripResponse)
	if err != nil {
		log.Fatal(err)
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusCreated)
    fmt.Fprint(rw, string(getTripJson))
}

func createLocation(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	addLocationRequest := new(AddLocationRequest)
	decoder := json.NewDecoder(req.Body)
	error := decoder.Decode(&addLocationRequest)
	if error != nil {
		log.Println(error.Error())
		http.Error(rw, error.Error(), http.StatusInternalServerError)
		return
	}
	
	location := Location{}
	address := Address{addLocationRequest.Address,addLocationRequest.City,addLocationRequest.State,addLocationRequest.Zip,location}
	user := User{"0",addLocationRequest.Name,address}
	user.UserId = strconv.Itoa(random(1, 100))
	getCoordinates(&user.UserAddress)
	session, err := mgo.Dial("mongodb://sejal:1234@ds045064.mongolab.com:45064/planner")
	if err != nil {
	        panic(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("planner").C("user")
	err = c.Insert(&user)
	if err != nil {
	        log.Fatal(err)
	}
	result := User{}
	err = c.Find(bson.M{"userid":user.UserId}).One(&result)
	if err != nil {
	        log.Fatal(err)
	}
	outgoingJSON, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusCreated)
	fmt.Fprint(rw, string(outgoingJSON))
}

func getLocation(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	locationId := p.ByName("location_id")
	session, err := mgo.Dial("mongodb://sejal:1234@ds045064.mongolab.com:45064/planner")
    if err != nil {
            panic(err)
    }
    defer session.Close()
    session.SetMode(mgo.Monotonic, true)
    c := session.DB("planner").C("user")
    result := User{}
    err = c.Find(bson.M{"userid":locationId}).One(&result)
    if err != nil {
            fmt.Fprint(rw, "Data not found")
            return
    }
    outgoingJSON, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusCreated)
    fmt.Fprint(rw, string(outgoingJSON))
}

func putLocation(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	locationId := p.ByName("location_id")
	updateLocationRequest := new(UpdateLocationRequest)
	decoder := json.NewDecoder(req.Body)
	error := decoder.Decode(&updateLocationRequest)
	if error != nil {
		log.Println(error.Error())
		http.Error(rw, error.Error(), http.StatusInternalServerError)
		return
	}
	
	location := Location{}
	address := Address{updateLocationRequest.Address,updateLocationRequest.City,updateLocationRequest.State,updateLocationRequest.Zip,location}
	
	session, err := mgo.Dial("mongodb://sejal:1234@ds045064.mongolab.com:45064/planner")
	if err != nil {
	        panic(err)
	}
	
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("planner").C("user")
	
	result := User{}
	err = c.Find(bson.M{"userid":locationId}).One(&result)
    if err != nil {
            log.Fatal(err)
    }
    
    userName := result.Name    
	user := User{locationId,userName,address}
	getCoordinates(&user.UserAddress)
	
	colQuerier := bson.M{"userid":locationId}
	change := bson.M{"$set": bson.M{"useraddress": user.UserAddress}}
	err = c.Update(colQuerier, change)
	if err != nil {
		panic(err)
	}
	
	result2 := User{}
	err = c.Find(bson.M{"userid":locationId}).One(&result2)
	if err != nil {
	        log.Fatal(err)
	}
	outgoingJSON, err := json.Marshal(result2)
	if err != nil {
		log.Fatal(err)
	}
	
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusCreated)
	fmt.Fprint(rw, string(outgoingJSON))
}

func deleteLocation(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	locationId,_ := strconv.Atoi(p.ByName("location_id"))
	session, err := mgo.Dial("mongodb://sejal:1234@ds045064.mongolab.com:45064/planner")
    if err != nil {
            panic(err)
    }
    defer session.Close()
    session.SetMode(mgo.Monotonic, true)
    c := session.DB("planner").C("user")
    err = c.Remove(bson.M{"userid":locationId})
    if err != nil {
            log.Fatal(err)
    }
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusCreated)
    fmt.Fprint(rw, "User Location deleted successfully")
}

func uberGetCost(iLatitude float64, ilongitude float64, fLatitude float64, flongitude float64) (float64, float64, float64){
	server_token := "IcVqT_KKraVNMaNs3fwF5nDHSs-86_72vm2kOfH_"
   	resp, err := http.Get("https://api.uber.com/v1/estimates/price?server_token="+server_token+"&start_longitude="+strconv.FormatFloat(ilongitude, 'f', 6, 64)+"&end_longitude="+strconv.FormatFloat(flongitude, 'f', 6, 64)+"&start_latitude="+strconv.FormatFloat(iLatitude, 'f', 6, 64)+"&end_latitude="+strconv.FormatFloat(fLatitude, 'f', 6, 64))
	if(err == nil){
	    body, err := ioutil.ReadAll(resp.Body)
	    if(err == nil) {
	        var data interface{}
	        json.Unmarshal(body, &data)
	        var m = data.(map[string] interface{})            
	        var articles = m["prices"].([]interface{})[0]
	        fare := articles.(map[string]interface{})["high_estimate"].(float64)
	        distance := articles.(map[string]interface{})["distance"].(float64)
	        duration := articles.(map[string]interface{})["duration"].(float64)
	        return fare, distance ,duration
	    } else {
	        fmt.Println(err)
	    }
	} else {
	    fmt.Println(err)
	}
	return 0,0,0
}

func getCost(costArr [][]float64,path []int) (float64){
	var sum float64
	sum = 0
	for i:=0 ;i<len(path)-1;i++{
		sum = sum + costArr[path[i]][path[i+1]]
	}
	return sum;
}
func updateTrip(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	tripId := p.ByName("trip_id")
	session, err := mgo.Dial("mongodb://sejal:1234@ds045064.mongolab.com:45064/planner")
    if err != nil {
    	panic(err)
    }
    defer session.Close()
    session.SetMode(mgo.Monotonic, true)
    c := session.DB("planner").C("trip")
    tripResult := Trip{}
    err = c.Find(bson.M{"id":tripId}).One(&tripResult)
    if err != nil {
            fmt.Fprint(rw, "Data not found")
            return
    }
    if(tripResult.Next == len(tripResult.Best_route_location_ids)){
    	fmt.Println("h");
    	//rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		/*updateTripJson, err := json.Marshal(updateTripResponse)
		if err != nil {
			log.Fatal(err)
		}*/
		fmt.Fprint(rw, "Trip is Finised. You have covered all the locations.")
		return
    }
    locationArr := make([]Location,2)
    var locId string
    for i:=0;i<2;i++{
    	location := new(Location)
    	if(i == 0){
    		if(tripResult.Next==0){
    			locId = tripResult.Starting_from_location_id
	    	}else{
	    		locId = tripResult.Best_route_location_ids[tripResult.Next-1]
	    	}	
    	}else{
    		locId = tripResult.Best_route_location_ids[tripResult.Next]
    	}
    	resp, err := http.Get("http://localhost:8000/locations/"+locId)
		if(err == nil){
			body, err := ioutil.ReadAll(resp.Body)
		    if(err == nil) {
		        var data interface{}
		        json.Unmarshal(body, &data)
		        var m = data.(map[string] interface{})
		        var coordinates = m["UserAddress"].(map[string]interface{})["Coordinates"]
		        location.Latitude = coordinates.(map[string]interface{})["Latitude"].(float64)
		        location.Longitude = coordinates.(map[string]interface{})["Longitude"].(float64)
		        locationArr[i] = *location
		    } else {
		        fmt.Println(err)
		    }
		} else {
		    fmt.Println(err)
		}	
    }
	url := "https://sandbox-api.uber.com/v1/requests"
	jsonUber := JsonUber{}
	jsonUber.Start_latitude = strconv.FormatFloat(locationArr[0].Latitude, 'E', -1, 64)
	jsonUber.Start_longitude = strconv.FormatFloat(locationArr[0].Longitude, 'E', -1, 64)
	jsonUber.End_latitude = strconv.FormatFloat(locationArr[1].Latitude, 'E', -1, 64)
	jsonUber.End_longitude = strconv.FormatFloat(locationArr[1].Longitude, 'E', -1, 64)
	jsonUber.Product_id = "04a497f5-380d-47f2-bf1b-ad4cfdcb51f2"
	jsonUberMar, err := json.Marshal(jsonUber)
	req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonUberMar))
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicHJvZmlsZSIsInJlcXVlc3QiLCJyZXF1ZXN0X3JlY2VpcHQiLCJoaXN0b3J5X2xpdGUiXSwic3ViIjoiMDFiNTkwNzktYTIyMy00NmRiLTk2NTktZjYwNGFkMWY1OWI4IiwiaXNzIjoidWJlci11czEiLCJqdGkiOiI1MzU5MzIzMi0wNmYzLTQyNWItODI1ZC1lYzk5NTU0Yjg1ZmMiLCJleHAiOjE0NTA3NTcwNDEsImlhdCI6MTQ0ODE2NTA0MSwidWFjdCI6InJmUTNmWXB3elRPNVNuRU5vMkIzbE1UVVpDUEVudCIsIm5iZiI6MTQ0ODE2NDk1MSwiYXVkIjoiWmhsaVZsWEhka3U2UlFiSEFqdmM4aUh6bjhjQXk1TDkifQ.oDzvT5QnpEHYgLvzRHxoLoyIYuu7fEkkuq7Q5kbTWOqr34esAJGryayhXYFNKUME56dOWifbHeJDXw0VolffCNiEqvQVkAFHTqnrJJs-hEhJFORY6b-p78837leOlHGWnP56U-7jM1oSvd_preUOL2ud5FhHGmj3ZCXU4gGkP3Qo2uDGCfHLz3meX_nL15aHqCpIS0I_STPm5dTL3bI2AW0QoKK0X5sEXanaG5AxaRg5RxkKmRXYG7LWuO2tT-JW8hpFxYlPG-wwei_Ws7STqxoaaM7b1XpJTT5q7nOU0CDXPkR5n-iJjm2LP9WQzodFQg70mlZugHPkKST4wov3WQ")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var data interface{}
    json.Unmarshal(body, &data)
    var m = data.(map[string] interface{})
    eta := (m["eta"].(float64))
	session, err = mgo.Dial("mongodb://sejal:1234@ds045064.mongolab.com:45064/planner")
	if err != nil {
		panic(err)
	}
	
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c = session.DB("planner").C("trip")
	
	trip := Trip{}
    trip.Id = tripId
    var status string
    if tripResult.Next == len(tripResult.Best_route_location_ids)-1{
    	status = "finished"	
    }else{
    	status = "requesting"	
    }
    counter := tripResult.Next+1
	colQuerier := bson.M{"id":tripId}
	change := bson.M{"$set": bson.M{"uber_wait_time_eta":0 ,"next":counter,"status":status}}
	err = c.Update(colQuerier, change)
	if err != nil {
		panic(err)
	}
	
	updateTripResponse := UpdateTripResponse{}
    updateTripResponse.Id = tripId
    updateTripResponse.Status = status
	updateTripResponse.Starting_from_location_id = tripResult.Starting_from_location_id
	updateTripResponse.Next_destination_location_id = tripResult.Best_route_location_ids[counter-1]
	updateTripResponse.Best_route_location_ids = tripResult.Best_route_location_ids
	updateTripResponse.Total_uber_costs = tripResult.Total_uber_costs
	updateTripResponse.Total_uber_duration = tripResult.Total_uber_duration
	updateTripResponse.Total_distance = tripResult.Total_distance
	updateTripResponse.Uber_wait_time_eta = int(eta)
	
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusCreated)
	updateTripJson, err := json.Marshal(updateTripResponse)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprint(rw, string(updateTripJson))
}

func main() {
    router := httprouter.New()
    router.POST("/locations", createLocation)
    router.GET("/locations/:location_id", getLocation)
    router.PUT("/locations/:location_id", putLocation)
    router.DELETE ("/locations/:location_id", deleteLocation)
    router.POST("/trips", createTrip)
    router.GET("/trips/:trip_id", getTrip)
	router.PUT("/trips/:trip_id/requestUber", updateTrip)
    server := http.Server{
            Addr:        "0.0.0.0:8000",
            Handler: router,
    }
    server.ListenAndServe()
}


Trip Planner
======
The trip planner is a feature that takes a set of locations from the database and will then check against UBER’s price estimates API to suggest the best possible route in terms of costs and duration.
UBER Price Estimates Resource used: GET /v1/estimates/price
Its using UBER Sandbox environment for all API calls.
 
##Plan a trip
* POST   /trips  
```
Request
{
    "starting_from_location_id: "999999",
    "location_ids" : [ "10000", "10001", "20004", "30003" ] 
}
```
```
Response
{
     "id" : "1122",
     “status” : “planning”,
     "starting_from_location_id: "999999",
     "best_route_location_ids" : [ "30003", "10001", "10000", "20004" ],
  "total_uber_costs" : 125,
  "total_uber_duration" : 640,
  "total_distance" : 25.05 
}

```
* GET  /trips/{trip_id} # Check the trip details and status
```
Request:
GET             /trips/1122
```
```
Response:
{
     "id" : "1122",
     "status" : "planning",
     "starting_from_location_id: "999999",
     "best_route_location_ids" : [ "30003", "10001", "10000", "20004" ],
  "total_uber_costs" : 125,
  "total_uber_duration" : 640,
  "total_distance" : 25.05 
}

```
* PUT /v1/sandbox/requests/{request_id}
Once a destination is reached, the subsequent call the API will request a car for the next destination.
```
Request:
 /trips/1122/request
```
```
{
     "id" : "1122",
     "status" : "requesting",
     "starting_from_location_id”: "999999",
     "next_destination_location_id”: "30003",
     "best_route_location_ids" : [ "30003", "10001", "10000", "20004" ],
  "total_uber_costs" : 125,
  "total_uber_duration" : 640,
  "total_distance" : 25.05,
  "uber_wait_time_eta" : 5 
}

```

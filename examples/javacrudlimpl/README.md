# Java implementation of the crudl service

Assuming you generate java code for [examples/crudl.sadl](https://github.com/boynton/sadl/blob/master/examples/crudl.sadl) like this:

    sadl2java -jsonutil -dir gen -package model -pom -server examples/crudl.sadl

Then, the Main.java file that gets generated can be replaced with the one in this directory, to demonstrate a simple memory-based implementation
of the service.

To build and run the generated service:

    $ cd gen
    $ mvn exec:java

Then you can test it. This session uses a "json" utility to pretty-print the results. To get it:

    $ go get github.com/boynton/hacks/json

A session against the server launched as above follows. Note the test of conditional get based on modified time.

    $ curl -s 'http://localhost:8080/items' | json
    {
       "items": []
    }
    $ curl -s -X POST -H "Content-type: application/json" -d '{"id":"1ce437b0-1dd2-11b2-beb7-003ee1be85f8","data":"Hi there!"}' 'http://localhost:8080/items' | json
    {
       "data": "Hi there!",
       "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f8",
       "modified": "2019-02-16T23:14:21.103Z"
    }
    $ curl -s 'http://localhost:8080/items/1ce437b0-1dd2-11b2-beb7-003ee1be85f8' | json
    {
       "data": "Hi there!",
       "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f8",
       "modified": "2019-02-16T23:14:21.103Z"
    }
    $ curl -s -X POST -H "Content-type: application/json" -d '{"id":"1ce437b0-1dd2-11b2-beb7-003ee1be85f8","data":"Hi there!"}' 'http://localhost:8080/items' | json
    {
       "error": "Already exists: 1ce437b0-1dd2-11b2-beb7-003ee1be85f8"
    }
    $ curl -s -X POST -H "Content-type: application/json" -d '{"id":"1ce437b0-1dd2-11b2-beb7-003ee1be85f9","data":"Hi there again"}' 'http://localhost:8080/items' | json
    {
       "data": "Hi there again",
       "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f9",
       "modified": "2019-02-16T23:15:55.894Z"
    }
    $ curl -s 'http://localhost:8080/items' | json
    {
       "items": [
          {
             "data": "Hi there again",
             "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f9",
             "modified": "2019-02-16T23:15:55.894Z"
          },
          {
             "data": "Hi there!",
             "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f8",
             "modified": "2019-02-16T23:14:21.103Z"
          }
       ]
    }
    $ curl -s 'http://localhost:8080/items?limit=1' | json
    {
       "items": [
          {
             "data": "Hi there again",
             "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f9",
             "modified": "2019-02-16T23:15:55.894Z"
          }
       ],
       "next": "1ce437b0-1dd2-11b2-beb7-003ee1be85f8"
    }
    $ curl -s 'http://localhost:8080/items?limit=1&skip=1ce437b0-1dd2-11b2-beb7-003ee1be85f8' | json
    {
       "items": [
          {
             "data": "Hi there!",
             "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f8",
             "modified": "2019-02-16T23:14:21.103Z"
          }
       ]
    }
    $ curl -v -H 'If-Modified-Since: 2019-02-16T23:14:21.103Z' -s 'http://localhost:8080/items/1ce437b0-1dd2-11b2-beb7-003ee1be85f9'
     to localhost (::1) port 8080 (#0)
    > GET /items/1ce437b0-1dd2-11b2-beb7-003ee1be85f9 HTTP/1.1
    > Host: localhost:8080
    > User-Agent: curl/7.54.0
    > Accept: */*
    > If-Modified-Since: 2019-02-16T23:14:21.103Z
    > 
    < HTTP/1.1 304 Not Modified
    < Date: Sat, 16 Feb 2019 23:15:02 GMT
    < Server: Jetty(9.4.7.v20170914)
    < 
    * Connection #0 to host localhost left intact
    $ curl -s -X PUT -H "Content-type: application/json" -d '{"id":"1ce437b0-1dd2-11b2-beb7-003ee1be85f9","data":"Hi there again!!!!!"}' 'http://localhost:8080/items/1ce437b0-1dd2-11b2-beb7-003ee1be85f9' | json
    {
       "data": "Hi there again!!!!!",
       "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f9",
       "modified": "2019-02-16T23:21:52.614Z"
    }
    $ curl -v -H 'If-Modified-Since: 2019-02-16T23:14:21.103Z' -s 'http://localhost:8080/items/1ce437b0-1dd2-11b2-beb7-003ee1be85f9' && echo
    *   Trying ::1...
    * TCP_NODELAY set
    * Connected to localhost (::1) port 8080 (#0)
    > GET /items/1ce437b0-1dd2-11b2-beb7-003ee1be85f9 HTTP/1.1
    > Host: localhost:8080
    > User-Agent: curl/7.54.0
    > Accept: */*
    > If-Modified-Since: 2019-02-16T23:14:21.103Z
    > 
    < HTTP/1.1 200 OK
    < Date: Sat, 16 Feb 2019 23:22:12 GMT
    < Content-Type: application/json
    < Transfer-Encoding: chunked
    < Server: Jetty(9.4.7.v20170914)
    < 
    * Connection #0 to host localhost left intact
    {"id":"1ce437b0-1dd2-11b2-beb7-003ee1be85f9","modified":"2019-02-16T23:21:52.614Z","data":"Hi there again!!!!!"}
    $ curl -s 'http://localhost:8080/items' | json
    {
       "items": [
          {
             "data": "Hi there again!!!!!",
             "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f9",
             "modified": "2019-02-16T23:21:52.614Z"
          },
          {
             "data": "Hi there!",
             "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f8",
             "modified": "2019-02-16T23:14:21.103Z"
          }
       ]
    }
    $ curl -v -X DELETE 'http://localhost:8080/items/1ce437b0-1dd2-11b2-beb7-003ee1be85f9'
    *   Trying ::1...
    * TCP_NODELAY set
    * Connected to localhost (::1) port 8080 (#0)
    > DELETE /items/1ce437b0-1dd2-11b2-beb7-003ee1be85f9 HTTP/1.1
    > Host: localhost:8080
    > User-Agent: curl/7.54.0
    > Accept: */*
    > 
    < HTTP/1.1 204 No Content
    < Date: Sat, 16 Feb 2019 23:23:26 GMT
    < Server: Jetty(9.4.7.v20170914)
    < 
    * Connection #0 to host localhost left intact
    $ curl -s 'http://localhost:8080/items' | json
    {
       "items": [
          {
             "data": "Hi there!",
             "id": "1ce437b0-1dd2-11b2-beb7-003ee1be85f8",
             "modified": "2019-02-16T23:14:21.103Z"
          }
       ]
    }

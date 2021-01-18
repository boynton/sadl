# Java implementation of the crudl service

    This is a simple implementation in Java for the server side of  [examples/crudl.sadl](https://github.com/boynton/sadl/blob/master/examples/crudl.sadl). To build and run the server:

    $ make

Then you can test it. This session uses a "json" utility to pretty-print the results. To get it:

    $ go get github.com/boynton/hacks/json

Or you can just omit the `| json` for each example and parse the one-line JSON yourself.
    
Here is an example session against the server launched as above. Note the test of conditional get based on modified time.

    $ curl -s 'http://localhost:8080/items' | json
    {
       "items": []
    }
    $ curl -s -X POST -H "Content-type: application/json" -d '{"id":"item1","data":"Hi there!"}' 'http://localhost:8080/items' | json
    {
       "data": "Hi there!",
       "id": "item1",
       "modified": "2019-02-16T23:14:21.103Z"
    }
    $ curl -s 'http://localhost:8080/items/item1' | json
    {
       "data": "Hi there!",
       "id": "item1",
       "modified": "2019-02-16T23:14:21.103Z"
    }
    $ curl -s -X POST -H "Content-type: application/json" -d '{"id":"item1","data":"Hi there!"}' 'http://localhost:8080/items' | json
    {
       "error": "Already exists: item1"
    }
    $ curl -s -X POST -H "Content-type: application/json" -d '{"id":"item2","data":"Hi there again"}' 'http://localhost:8080/items' | json
    {
       "data": "Hi there again",
       "id": "item2",
       "modified": "2019-02-16T23:15:55.894Z"
    }
    $ curl -s 'http://localhost:8080/items' | json
    {
       "items": [
          {
             "data": "Hi there again",
             "id": "item2",
             "modified": "2019-02-16T23:15:55.894Z"
          },
          {
             "data": "Hi there!",
             "id": "item1",
             "modified": "2019-02-16T23:14:21.103Z"
          }
       ]
    }
    $ curl -s 'http://localhost:8080/items?limit=1' | json
    {
       "items": [
          {
             "data": "Hi there again",
             "id": "item2",
             "modified": "2019-02-16T23:15:55.894Z"
          }
       ],
       "next": "item1"
    }
    $ curl -s 'http://localhost:8080/items?limit=1&skip=item1' | json
    {
       "items": [
          {
             "data": "Hi there!",
             "id": "item1",
             "modified": "2019-02-16T23:14:21.103Z"
          }
       ]
    }
    $ curl -v -H 'If-Modified-Since: 2019-02-16T23:14:21.103Z' -s 'http://localhost:8080/items/item2'
     to localhost (::1) port 8080 (#0)
    > GET /items/item2 HTTP/1.1
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
    $ curl -s -X PUT -H "Content-type: application/json" -d '{"id":"item2","data":"Hi there again!!!!!"}' 'http://localhost:8080/items/item2' | json
    {
       "data": "Hi there again!!!!!",
       "id": "item2",
       "modified": "2019-02-16T23:21:52.614Z"
    }
    $ curl -v -H 'If-Modified-Since: 2019-02-16T23:14:21.103Z' -s 'http://localhost:8080/items/item2' && echo
    *   Trying ::1...
    * TCP_NODELAY set
    * Connected to localhost (::1) port 8080 (#0)
    > GET /items/item2 HTTP/1.1
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
    {"id":"item2","modified":"2019-02-16T23:21:52.614Z","data":"Hi there again!!!!!"}
    $ curl -s 'http://localhost:8080/items' | json
    {
       "items": [
          {
             "data": "Hi there again!!!!!",
             "id": "item2",
             "modified": "2019-02-16T23:21:52.614Z"
          },
          {
             "data": "Hi there!",
             "id": "item1",
             "modified": "2019-02-16T23:14:21.103Z"
          }
       ]
    }
    $ curl -v -X DELETE 'http://localhost:8080/items/item2'
    *   Trying ::1...
    * TCP_NODELAY set
    * Connected to localhost (::1) port 8080 (#0)
    > DELETE /items/item2 HTTP/1.1
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
             "id": "item1",
             "modified": "2019-02-16T23:14:21.103Z"
          }
       ]
    }

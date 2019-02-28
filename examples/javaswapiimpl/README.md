#javaswapiimpl

This minimal example shows how graphql can be utilized on top of generated web service code.

The main.java file in this directory is a dumb in-memory implementation of the specified web service. With just that code,
both traditional REST style access works as expected, as well as a GraphQL endpoint with the specified queries, using the
same types, and utilizing the generated HTTP operations as data resolvers for GraphQL.


## build

    $ ./bin/sadl2java -jsonutil -dir gen -package model -pom -server -graphql examples/swapi.sadl
    $ cp examples/javaswapiimpl/Main.java gen/src/main/java
    $ cd gen
    $ mvn compile
    $ mvn exec:java

Then, from another terminal, try some queries:

    $ curl -s -H "Content-type: application/json" http://localhost:8080/graphql -d '{"query": "{films { id, name, cast {id, name} } }"}' | json
    {
       "data": {
          "films": [
             {
                "cast": [
                   {
                      "id": "1",
                      "name": "R2-D2"
                   },
                   {
                      "id": "2",
                      "name": "Han Solo"
                   }
                ],
                "id": "4",
                "name": "A New Hope"
             },
             {
                "cast": null,
                "id": "5",
                "name": "The Empire Strikes Back"
             }
          ]
       }
    }





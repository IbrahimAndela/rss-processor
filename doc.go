// Package rss-processor

/*
Pacakge rss-processor provides RSS feeds processing for clients
It has the search, fetchrss and serverss packages to enable it get store and provide clients with the ability to
search stored feeds and present the results via a HTTP interface.

Basics

The application uses already stored RSS sources (i.e links) on the sources table in the database, thus for more RSS feeds one only needs to add the source and the application will pick up in the next wake up (at the moment every 5 minutes).

MySQL database was chosen as the data backend to leverage it's FULL TEXT search capabilities to enable clients query for various topics get results.

N.B the added RSS sources in the DB are not exhaustive and others from the publishers can be added to the table and get processed on.
the next run.

Architecture

The application has three main packages.
    1. fetchrss
    2. serverss
    3. searchrss
The packages are called and composed in the main function to start up the application.
First the application fetches feeds for RSS sources defined in the Database and stores them locally.
Thereafter periodically after 5 minutes the fetchrss package fetches the RSS feeds and updates the local database with newer feeds.

The main programme also starts a http server (on port 9000) on a separate goroutine to listen to and process client requests to search the current locally stored feeds.

The server package utilizes the search package to search and return matching results from the feeds in the database. The search is dependent on FULL TEXT search for MySQL databases.

# Security

To make requests to the server one needs to abtain a JWT token string and attach it to the following requests to the server.
To get a valid token, issue the request below.
N.B This is an open endpoint to get the token and is for (demo security puposes)
    $ curl http://localhost:9000/get-token
    Sample response
    ```
    {
       "JWTTokenValue" : "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdXRob3JpemVkIjp0cnVlLCJjbGllbnQiOiJEZW1vIEF1dGggVXNlciIsImV4cCI6MTU4MjYzNTg5MX0.Izgu_9b1eS8HIEFPzYeRK7nFJruYNGnZMGqlyF8l3mY"
    }```


Documentation

The documentation can be read using the godoc tool by running the command

    $ godoc -http=":<port>" # e.g godoc -http=":10000"
and thereafter visiting
    http://localhost:10000/pkg/github.com/gideon-maina/rss-processor/

Deployment

To enable quick running, a docker-compose file is present in the directory and the app can be started with.
    $ docker-compose up
The docker file creates two containers one for the MySQL db and one for the RSS processor app (this app).

Then you can  query to get feeds for a given query
    $ curl http://localhost:9000/search?q=lewis+hamiliton  -H "Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdXRob3JpemVkIjp0cnVlLCJjbGllbnQiOiJEZW1vIEF1dGggVXNlciIsImV4cCI6MTU4MjYzNTg5MX0.Izgu_9b1eS8HIEFPzYeRK7nFJruYNGnZMGqlyF8l3mY"

Schema

The schema is under the sql/ directory and is automatically used by the docker containers to bootstrap the DB.

Running

NB.Since docker containers need a prefix of their names in order to connect to them via TCP, you might need to update `db/db.go` to remove docker specific fields

Pass the duration to refresh the feeds as an arg to the program, the default is 5 minutes.

    $ go run main.go -refresh=<minutes>
*/
package main

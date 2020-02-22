//Package provides the types and functions to fetch RSS xml files from sources (stored in a DB) and updating newer feeds with
// each subsequent run.
package fetchrss

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/gideon-maina/rss-processor/db"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	// The layouts mimmick the time.layout constants for time formats
	MySQLToTimeLayout = "2006-01-02 15:04:05"
	TimeToMySQLLayout = "2006-01-02 15:04:05"
)

// For each of the sources of the rss feeds have a struct for them containing all the XML as a string and whether it's done read
type RSSXML struct {
	Content []byte
}

func GetRSSSources() ([]Source, error) {
	conn := db.Conn()
	defer conn.Close()

	sources := []Source{}
	sourceRows, err := conn.Query("SELECT id,publisher,url,topic,description,lastBuildDate,dateModified,dateCreated FROM sources;")
	if err != nil {
		return nil, err
	}
	for sourceRows.Next() {
		var source Source
		err = sourceRows.Scan(&source.Id, &source.Publisher, &source.Url, &source.Topic, &source.Description, &source.LastBuildDate, &source.DateModified, &source.DateCreated)
		sources = append(sources, source)
	}
	return sources, nil
}
func GetRSSXML(url string) (*RSSXML, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &RSSXML{Content: body}, nil
}
func StoreFeeds(sourceId int, xmlContent *RSSXML) error {
	conn := db.Conn()
	defer conn.Close()

	transaction, err := conn.Begin()
	saveFeed := false

	if err != nil {
		fmt.Println("Error in creating transaction:> ", err)
		log.Fatal(err)
	}

	// parse the xmlcontent for this particular rss topic
	feedSourceResults := []RSSData{}
	// Since NewDecoder needs something that is a reader convert our content []byte into a reader
	newReader := bytes.NewReader(xmlContent.Content)
	decoder := xml.NewDecoder(newReader)
	err = decoder.Decode(&feedSourceResults)
	if err != nil {
		fmt.Println("Error in decoding XML: >", err)
		log.Fatal(err)
	}
	// Check if the latest feed for a given source topic/ url has been updated
	latestLocalFeed, err := getLastSavedFeed(conn, sourceId)
	if err != nil {
		fmt.Println("Error in decoding XML: >", err)
		log.Fatal(err)
	}

	for _, val := range feedSourceResults {
		value := val // force value evaluation on each loop to enable go routines work
		// fmt.Printf("%T", value.Channel.Item, " Is the TYPE of the news Items\n")
		// For all the news items in the feed items slice loop over and save them if they are newer than last record
		for _, newsItemVal := range value.Channel.Item {
			newsItem := newsItemVal // force evaluation of newsItem each time in the loop
			title := newsItem.Title
			description := newsItem.Description
			link := newsItem.Link
			guid := newsItem.Guid
			// guid is a struct so it has to be converted to a json string
			guidString, err := json.Marshal(&guid)
			if err != nil {
				log.Fatal(err)
			}
			pubDate := newsItem.PubDate
			// Sometimes feeds don't have pubDates given so skip them don't ASSUME
			if pubDate == "" {
				continue
			}
			// Convert this pubDate to someething MySQL likes
			pubDateTime, err := time.Parse(time.RFC1123, pubDate)
			pubDateMysql := pubDateTime.Format(TimeToMySQLLayout)
			if err != nil {
				fmt.Println("Error in convert pubDate to time.Time for newsItem")
				log.Fatal(err)
			}
			if latestLocalFeed.PubDate == "" {
				//First time encountering this source of feeds so save all feeds
				saveFeed = true
			}
			latestLocalFeedTime, err := time.Parse(MySQLToTimeLayout, latestLocalFeed.PubDate)
			if err != nil {
				// Don't exit on this error , ssume it's the first time and we don't have a latestlocalfeed for topic
				fmt.Errorf("FAiled to parse latest feed for topic pubDate. %v", err)
			}
			// If this pubDate is after the latest record we have then save it to the DB
			if pubDateTime.After(latestLocalFeedTime) || saveFeed {
				// Save this feed as it's new
				dateCreated := time.Now().Format(TimeToMySQLLayout)
				insertS, err := transaction.Prepare("INSERT INTO feeds (source_id, title, description, link, guid, pubDate, dateCreated) VALUES(?,?,?,?,?,?,?)")
				if err != nil {
					fmt.Println("Error in storing feeds :>", err)
					log.Fatal(err)
				}
				defer insertS.Close()
				if _, err := insertS.Exec(sourceId, title, description, link, guidString, pubDateMysql, dateCreated); err != nil {
					transaction.Rollback()
					log.Fatal(err)
				}
				log.Println("Save Feed", title, " for source Id ", sourceId)
			} else {
				// This feed is already saved in the DB so skip
				continue
			}
		}
	}
	return transaction.Commit()
}

func getLastSavedFeed(conn *sql.DB, sourceID int) (Feed, error) {
	transaction, err := conn.Begin()
	if err != nil {
		fmt.Println("Error in creating transaction:> ", err)
		log.Fatal(err)
	}
	lasFeedRow, err := transaction.Query("SELECT id,source_id,title,description,link,guid,pubDate,dateCreated, dateModified FROM feeds WHERE source_id = ? ORDER BY pubDate DESC LIMIT 1;", sourceID)
	if err != nil {
		fmt.Println("Error in creating transaction:> ", err)
		log.Fatal(err)
	}
	var feed Feed
	for lasFeedRow.Next() {
		err = lasFeedRow.Scan(&feed.Id, &feed.SourceId, &feed.Title, &feed.Description, &feed.Link, &feed.Guid, &feed.PubDate, &feed.DateCreated, &feed.DateModified)
		if err != nil {
			fmt.Println("Error in the get latest feed", err)
			log.Fatal(err)
		}
	}
	return feed, nil
}

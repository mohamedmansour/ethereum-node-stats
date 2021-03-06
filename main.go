package main

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/MariusVanDerWijden/node-crawler-backend/api"
	"github.com/MariusVanDerWijden/node-crawler-backend/input"
)

var (
	inputDBName = "nodetable"
	myDBName    = "nodes"
)

func main() {
	crawlerDB, err := sql.Open("sqlite3", inputDBName)
	if err != nil {
		panic(err)
	}
	shouldInit := false
	if _, err := os.Stat(myDBName); os.IsNotExist(err) {
		shouldInit = true
	}
	nodeDB, err := sql.Open("sqlite3", myDBName)
	if err != nil {
		panic(err)
	}
	if shouldInit {
		fmt.Println("DB did not exist, init")
		if err := createDB(nodeDB); err != nil {
			panic(err)
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	// Start reading deamon
	go deamon(&wg, crawlerDB, nodeDB)
	// Start the API deamon
	apiDeamon := api.New(nodeDB)
	go apiDeamon.HandleRequests(&wg)
	wg.Wait()
}

// Deamon reads new nodes from the crawler and puts them in the db
// Might trigger the invalidation of caches for the api in the future
func deamon(wg *sync.WaitGroup, crawlerDB, nodeDB *sql.DB) {
	defer wg.Done()
	lastCheck := time.Time{}
	for {
		nodes, err := input.ReadRecentNodes(crawlerDB, lastCheck)
		if err != nil {
			fmt.Printf("Error reading nodes: %v\n", err)
			return
		}
		lastCheck = time.Now()
		if len(nodes) > 0 {
			err := InsertCrawledNodes(nodeDB, nodes)
			if err != nil {
				fmt.Printf("Error inserting nodes: %v\n", err)
			}
		}
		time.Sleep(time.Second)
	}
}

package main

import (
	elementsModel "backend/models"
	"backend/routes"
	"backend/scraper"
	"log"
	"net/http"
)

func main() {
	filePath := "data/elements.json"

	log.Println("Scraping data...")
	scraper.Scrape(filePath)

	log.Println("Initializing elements model...")
	err := elementsModel.GetInstance().Initialize(filePath)
	if err != nil {
		log.Fatalf("Error initializing elements service: %v", err)
	}

	log.Println("Starting server on http://localhost:8080")
	router := routes.InitRoutes()
	log.Fatal(http.ListenAndServe(":8080", router))
}

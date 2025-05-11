package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	elementsModel "backend/models"
	"backend/routes"
	"backend/scraper"
)

func getDataPath() string {
    cwd, err := os.Getwd()
    if err != nil {
        log.Fatalf("Cannot get working directory: %v", err)
    }
    // return filepath.Join(cwd, "src","backend","data", "elements.json") //for docker build
    return filepath.Join(cwd,"data", "elements.json") // if not using docker
}

func main() {
	filePath := os.Getenv("ELEMENTS_JSON_PATH")
	if filePath == "" {
		filePath = getDataPath()
	}

	log.Println("Scraping data...")
	scraper.Scrape(filePath)

	log.Println("Initializing elements model...")
	err := elementsModel.GetInstance().Initialize(filePath)
	if err != nil {
		log.Fatalf("Error initializing elements service: %v", err)
	}

	log.Println("Starting server on http://0.0.0.0:8080")
	router := routes.InitRoutes()
	log.Fatal(http.ListenAndServe(":8080", router))
}


	// package main

	// import (
	// 	elementsController "backend/controllers"
	// 	elementsModel "backend/models"
	// 	"backend/scraper"
	// 	"fmt"
	// )

	// func main() {
	// 	filePath := "data/elements.json"
	// 	target := "Barn"
	// 	n := 10
	// 	useBFS := true


	// 	fmt.Println("Recipe Tree:")
	// 	elementsController.PrintRecipeTree(tree, "", true)

	// 	fmt.Printf("\nNodes visited: %d\n", visited)
	// 	fmt.Printf("Duration: %s\n", duration)
	// }

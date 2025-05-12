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

func main() {
	cwd, err := os.Getwd()
    if err != nil {
        log.Fatalf("Cannot get working directory: %v", err)
    }

	// filepath:=filepath.Join(cwd, "src","backend","data", "elements.json")
	filepath:=filepath.Join(cwd, "data", "elements.json")

	log.Println("Scraping data...")
	scraper.Scrape(filepath)

	log.Println("Initializing elements model...")
	errr := elementsModel.GetInstance().Initialize(filepath)
	if errr != nil {
		log.Fatalf("error initializing elements service: %v", errr)
	}

	log.Println("Starting server on http://0.0.0.0:4003")
	router := routes.InitRoutes()
	log.Fatal(http.ListenAndServe(":4003", router))
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

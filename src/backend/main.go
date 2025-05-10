// package main

// import (
// 	elementsModel "backend/models"
// 	"backend/routes"
// 	"backend/scraper"
// 	"log"
// 	"net/http"
// )

// func main() {
// 	filePath := "data/elements.json"

// 	log.Println("Scraping data...")
// 	scraper.Scrape(filePath)

// 	log.Println("Initializing elements model...")
// 	err := elementsModel.GetInstance().Initialize(filePath)
// 	if err != nil {
// 		log.Fatalf("Error initializing elements service: %v", err)
// 	}

// 	log.Println("Starting server on http://localhost:8080")
// 	router := routes.InitRoutes()
// 	log.Fatal(http.ListenAndServe(":8080", router))
// }

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

// 	fmt.Println("🔄 Scraping data...")
// 	scraper.Scrape(filePath)

// 	fmt.Println("🔧 Initializing model...")
// 	err := elementsModel.GetInstance().Initialize(filePath)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println("🚀 Finding recipes...")
// 	controller := elementsController.ElementController{}
// 	tree, visited, duration, err := controller.FindNRecipes(target, n, useBFS)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println("📦 Recipe Tree:")
// 	elementsController.PrintRecipeTree(tree, "", true)

// 	fmt.Printf("\nNodes visited: %d\n", visited)
// 	fmt.Printf("Duration: %s\n", duration)
// }

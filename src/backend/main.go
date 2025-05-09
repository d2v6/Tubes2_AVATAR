package main

import (
	ElementsController "backend/controllers"
	ElementsModel "backend/models"
	Scraper "backend/scraper"
	"fmt"
)

func main() {
	filePath := "data/elements.json"

	fmt.Println("Scraping data...")
	Scraper.Scrape(filePath)

	elementsService := ElementsModel.GetInstance()
	err := elementsService.Initialize(filePath)
	if err != nil {
		fmt.Println("Error initializing elements service:", err)
		return
	}

	_, err = ElementsController.NewElementController(filePath)
	if err != nil {
		fmt.Println("Error creating controller:", err)
		return
	}

	targetElement := "Lake"
	fmt.Printf("\n=== Example: Finding multiple recipe trees for %s ===\n", targetElement)

	targetNode, err := elementsService.GetElementNode(targetElement)
	if err != nil {
		fmt.Println("Error getting element node:", err)
		return
	}

	// BFS
	fmt.Println("\n-- BFS Trees --")
	bfsResults := ElementsController.FindNRecipesForElementBFS(targetNode)
	for i, tree := range bfsResults {
		fmt.Printf("\nBFS Tree %d:\n", i+1)
		ElementsController.PrintTree(tree, "", true)
	}

	// DFS
	fmt.Println("\n-- DFS Trees --")
	dfsResults := ElementsController.FindNRecipesForElementDFS(targetNode)
	for i, tree := range dfsResults {
		fmt.Printf("\nDFS Tree %d:\n", i+1)
		ElementsController.PrintTree(tree, "", true)
	}
}

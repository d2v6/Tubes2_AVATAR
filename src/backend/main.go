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

    controller, err := ElementsController.NewElementController(filePath)
    if err != nil {
        fmt.Println("Error creating controller:", err)
        return
    }

    targetElement := "Brick"
    fmt.Printf("\n=== Example 1: Finding path to create %s ===\n", targetElement)
    
    path, err := controller.FindPathToElement(targetElement)
    if err != nil {
        fmt.Printf("Error finding path for %s: %v\n", targetElement, err)
    } else {
        fmt.Printf("Path to create %s:\n", targetElement)
        
        if len(path.Steps) == 0 {
            fmt.Printf("%s is a tier 0 element and cannot be created.\n", targetElement)
        } else {
            for i, step := range path.Steps {
                fmt.Printf("Step %d: Combine %s and %s to create %s\n", 
                    i+1, 
                    step.Ingredients[0], 
                    step.Ingredients[1], 
                    step.Element)
            }
        }
    }

    fmt.Printf("\n=== Example 2: Getting formatted instructions for %s ===\n", targetElement)
    
    instructions, err := controller.GetElementCreationInstructions(targetElement)
    if err != nil {
        fmt.Printf("Error getting instructions for %s: %v\n", targetElement, err)
    } else {
        fmt.Println(instructions)
    }

    fmt.Printf("\n=== Example 3: Getting dependency tree for %s ===\n", targetElement)
    
    tree, err := controller.GetElementDependencyTree(targetElement)
    if err != nil {
        fmt.Printf("Error getting dependency tree for %s: %v\n", targetElement, err)
    } else {
        fmt.Println(tree)
    }
}
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// Recipe represents a combination of ingredients to create an element
type Recipe struct {
	Ingredients []string `json:"ingredients"`
}

// Element represents a Little Alchemy 2 element and its recipes
type Element struct {
	Name    string   `json:"name"`
	Tier    int      `json:"tier"`
	Recipes []Recipe `json:"recipes"`
}

func main() {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"),
		colly.AllowedDomains("little-alchemy.fandom.com"),
		colly.AllowURLRevisit(),
	)

	// Create a slice to store all elements
	var elements []Element

	// Use a map to track the tier for each section
	sectionTiers := make(map[string]int)

	// First pass: Collect all section tiers
	sectionCollector := c.Clone()
	sectionCollector.OnHTML("h3 .mw-headline", func(e *colly.HTMLElement) {
		sectionTitle := e.Text
		sectionID := e.Attr("id") // Get the section ID
		fmt.Println("Found section:", sectionTitle, "with ID:", sectionID)

		// Set tier based on section title
		var tier int
		switch sectionTitle {
		case "Starting elements":
			tier = 0
		case "Special element":
			tier = 0
		default:
			if sectionTitle != "" {
				rawSectionTitle := strings.Split(sectionTitle, " ")
				if len(rawSectionTitle) == 3 {
					// Extract the tier number from the section title
					tierStr := rawSectionTitle[1]
					if t, err := strconv.Atoi(tierStr); err == nil {
						tier = t
					} else {
						fmt.Println("Error converting tier string to int:", err)
						tier = 999 // Default high value for unknown
					}
				} else {
					fmt.Println("Unexpected section title format:", sectionTitle)
					tier = 999 // Default high value for unknown
				}
			}
		}

		// Store the tier for this section ID
		sectionTiers[sectionID] = tier
		fmt.Printf("Section %s (ID: %s) set to tier: %d\n", sectionTitle, sectionID, tier)
	})

	// Process the sections first
	if err := sectionCollector.Visit("https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)"); err != nil {
		log.Fatal(err)
	}

	// Second pass: Process content with known tiers
	c.OnHTML("h3", func(h3 *colly.HTMLElement) {
		sectionID := h3.ChildAttr(".mw-headline", "id")
		tier, exists := sectionTiers[sectionID]
		if !exists {
			tier = 999 // Default if not found
		}

		// Find the table that follows this header
		h3.DOM.NextUntil("h3").Each(func(i int, s *goquery.Selection) {
			if s.Is("table.list-table") {
				// Process each row in this table
				s.Find("tr").Each(func(_ int, row *goquery.Selection) {
					// Skip header rows or empty rows
					if row.Find("td:nth-of-type(1)").Text() == "" {
						return
					}

					// Extract element name (first column)
					name := row.Find("td:nth-of-type(1)").Text()

					// Extract recipes (second column)
					var recipes []Recipe

					recipeText := row.Find("td:nth-of-type(2)").Text()
					if recipeText != "" {
						rawRecipes := strings.Split(recipeText, "\n")
						for _, recipe := range rawRecipes {
							recipe = strings.TrimSpace(recipe)

							// Only process recipes that contain "+"
							if strings.Contains(recipe, "+") {
								parts := strings.Split(recipe, "+")
								var ingredients []string
								for _, part := range parts {
									ingredient := strings.TrimSpace(part)
									if ingredient != "" {
										ingredients = append(ingredients, ingredient)
									}
								}

								if len(ingredients) >= 2 {
									recipes = append(recipes, Recipe{Ingredients: ingredients})
								}
							}
						}
					}

					// Add the element to our collection if name is not empty
					if name != "" {
						elements = append(elements, Element{
							Name:    strings.TrimSpace(name),
							Tier:    tier,
							Recipes: recipes,
						})
					}
				})
			}
		})
	})

	if err := c.Visit("https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)"); err != nil {
		log.Fatal(err)
	}

	// Marshal the data to JSON
	jsonData, err := json.MarshalIndent(elements, "", "  ")
	if err != nil {
		log.Fatal("Error marshalling to JSON:", err)
	}

	// Write to a file
	if err := os.WriteFile("elements.json", jsonData, 0644); err != nil {
		log.Fatal("Error writing JSON to file:", err)
	}

	fmt.Println("Data successfully saved to elements.json")
}

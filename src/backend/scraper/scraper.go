package scraper

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

type Recipe struct {
    Ingredients []string `json:"ingredients"`
}

type Element struct {
    Name    string   `json:"name"`
    Tier    int      `json:"tier"`
    Recipes []Recipe `json:"recipes"`
}

func Scrape(filePath string) {
    c := colly.NewCollector(
        colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"),
        colly.AllowedDomains("little-alchemy.fandom.com"),
        colly.AllowURLRevisit(),
    )

    var elements []Element
    sectionTiers := make(map[string]int)

    sectionCollector := c.Clone()
    sectionCollector.OnHTML("h3 .mw-headline", func(e *colly.HTMLElement) {
        sectionTitle := e.Text
        sectionID := e.Attr("id")

        var tier int
        switch sectionTitle {
        case "Starting elements", "Special element":
            tier = 0
        default:
            if sectionTitle != "" {
                rawSectionTitle := strings.Split(sectionTitle, " ")
                if len(rawSectionTitle) == 3 {
                    tierStr := rawSectionTitle[1]
                    if t, err := strconv.Atoi(tierStr); err == nil {
                        tier = t
                    } else {
                        fmt.Println("Error converting tier string to int:", err)
                        tier = 999
                    }
                } else {
                    fmt.Println("Unexpected section title format:", sectionTitle)
                    tier = 999
                }
            }
        }

        sectionTiers[sectionID] = tier
    })

    if err := sectionCollector.Visit("https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)"); err != nil {
        log.Fatal(err)
    }

    c.OnHTML("h3", func(h3 *colly.HTMLElement) {
        sectionID := h3.ChildAttr(".mw-headline", "id")
        tier, exists := sectionTiers[sectionID]
        if !exists {
            tier = 999
        }

        h3.DOM.NextUntil("h3").Each(func(i int, s *goquery.Selection) {
            if s.Is("table.list-table") {
                s.Find("tr").Each(func(_ int, row *goquery.Selection) {
                    if row.Find("td:nth-of-type(1)").Text() == "" {
                        return
                    }

                    name := row.Find("td:nth-of-type(1)").Text()
                    var recipes []Recipe

                    recipeText := row.Find("td:nth-of-type(2)").Text()
                    if recipeText != "" {
                        rawRecipes := strings.Split(recipeText, "\n")
                        for _, recipe := range rawRecipes {
                            recipe = strings.TrimSpace(recipe)
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

    jsonData, err := json.MarshalIndent(elements, "", "  ")
    if err != nil {
        log.Fatal("Error marshalling to JSON:", err)
    }

    if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
        log.Fatal("Error writing JSON to file:", err)
    }

    fmt.Println("Data successfully saved to", filePath)
}
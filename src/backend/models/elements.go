package models

import (
	"encoding/json"
	"errors"
	"os"
)

type Recipe struct {
	Ingredients []string `json:"ingredients"`
}

type Element struct {
	Name          string   `json:"name"`
	Tier          int      `json:"tier"`
	Recipes       []Recipe `json:"recipes"`
	ChildElements []Recipe
}

func LoadElements(filePath string) ([]Element, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var elements []Element
	err = json.Unmarshal(data, &elements)
	if err != nil {
		return nil, err
	}

	for i := range elements {
		elements[i].ChildElements = []Recipe{}
	}

	populateChildElements(elements)

	for _, element := range elements {
		jsonElement, _ := json.MarshalIndent(element, "", "  ")
		println(string(jsonElement))
	}

	return elements, nil
}

func populateChildElements(elements []Element) {
	// Create a map for faster lookups
	elementMap := make(map[string]*Element)
	for i := range elements {
		elementMap[elements[i].Name] = &elements[i]
	}

	for _, element := range elements {
		if element.Recipes == nil {
			continue
		}

		for _, recipe := range element.Recipes {
			for _, ingredient := range recipe.Ingredients {
				if childElement, exists := elementMap[ingredient]; exists {
					childRecipe := Recipe{
						Ingredients: []string{element.Name, "Recipe: " + ingredient + " + " + getOtherIngredient(recipe.Ingredients, ingredient)},
					}
					childElement.ChildElements = append(childElement.ChildElements, childRecipe)
				}
			}
		}
	}
}

func getOtherIngredient(ingredients []string, currentIngredient string) string {
	for _, ing := range ingredients {
		if ing != currentIngredient {
			return ing
		}
	}
	return ""
}

func GetElementByName(elements []Element, name string) (*Element, error) {
	for _, element := range elements {
		if element.Name == name {
			return &element, nil
		}
	}
	return nil, errors.New("element not found")
}
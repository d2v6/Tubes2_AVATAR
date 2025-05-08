package main

import (
	"backend/models"
	"fmt"
)

func main() {
    filePath := "data/elements.json"

    elements, err := models.LoadElements(filePath)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }

    for _, element := range elements {
        fmt.Printf("Element: %+v\n", element)
    }
}
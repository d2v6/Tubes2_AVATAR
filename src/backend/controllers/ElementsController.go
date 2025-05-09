package elementsController

import (
	elementsModel "backend/models"
	"errors"
	"fmt"
	"strings"
)

type ElementController struct {
}

func NewElementController(filePath string) (*ElementController, error) {
	err := elementsModel.GetInstance().Initialize(filePath)
	if err != nil {
		return nil, err
	}
	
	return &ElementController{}, nil
}

type RecipeStep struct {
	Element     string
	Ingredients []string
}

type ElementPath struct {
	TargetElement string
	Steps         []RecipeStep
}

func (ec *ElementController) FindPathToElement(targetName string) (*ElementPath, error) {
	elementsService := elementsModel.GetInstance()
	
	targetNode, err := elementsService.GetElementNode(targetName)
	if err != nil {
		return nil, err
	}
	
	if targetNode.Element.Tier == 0 {
		return &ElementPath{
			TargetElement: targetName,
			Steps:         []RecipeStep{},
		}, nil
	}
	
	path, err := findOptimalPath(targetNode)
	if err != nil {
		return nil, err
	}
	
	return path, nil
}

func findOptimalPath(targetNode *elementsModel.ElementNode) (*ElementPath, error) {
	path := &ElementPath{
		TargetElement: targetNode.Element.Name,
		Steps:         []RecipeStep{},
	}
	
	processedElements := make(map[string]bool)
	
	queue := []*elementsModel.ElementNode{targetNode}
	
	elementToStep := make(map[string]RecipeStep)
	
	elementToTierSum := make(map[string]int)
	elementToTierSum[targetNode.Element.Name] = targetNode.Element.Tier
	
	for len(queue) > 0 {
		currentNode := queue[0]
		queue = queue[1:]
		
		if processedElements[currentNode.Element.Name] {
			continue
		}
		processedElements[currentNode.Element.Name] = true
		
		if currentNode.Element.Tier == 0 {
			continue
		}
		
		var bestRelation *elementsModel.ElementRelation
		lowestTierSum := 9999999
		
		for _, relation := range currentNode.Parents {
			tierSum := 0
			allIngredientsExist := true
			
			for _, sourceNode := range relation.SourceNodes {
				if sourceNode == nil || sourceNode.Element == nil {
					allIngredientsExist = false
					break
				}
				tierSum += sourceNode.Element.Tier
			}
			
			if allIngredientsExist && tierSum < lowestTierSum {
				lowestTierSum = tierSum
				bestRelation = relation
			}
		}
		
		if bestRelation != nil {
			step := RecipeStep{
				Element:     currentNode.Element.Name,
				Ingredients: bestRelation.Recipe.Ingredients,
			}
			
			elementToStep[currentNode.Element.Name] = step
			
			for _, sourceNode := range bestRelation.SourceNodes {
				if sourceNode.Element.Tier > 0 {
					queue = append(queue, sourceNode)
				}
			}
		}
	}
	
	elementStack := []string{targetNode.Element.Name}
	visitedElements := make(map[string]bool)
	
	for len(elementStack) > 0 {
		currentElement := elementStack[len(elementStack)-1]
		elementStack = elementStack[:len(elementStack)-1]
		
		if visitedElements[currentElement] {
			continue
		}
		visitedElements[currentElement] = true
		
		step, exists := elementToStep[currentElement]
		if exists {
			path.Steps = append(path.Steps, step)
			
			for _, ingredient := range step.Ingredients {
				if elementToStep[ingredient].Element != "" { 
					elementStack = append(elementStack, ingredient)
				}
			}
		}
	}
	
	sortSteps(path)
	
	return path, nil
}

func sortSteps(path *ElementPath) {
	elementToStepIndex := make(map[string]int)
	for i, step := range path.Steps {
		elementToStepIndex[step.Element] = i
	}
	
	availableElements := make(map[string]bool)
	
	for _, step := range path.Steps {
		for _, ingredient := range step.Ingredients {
			if _, exists := elementToStepIndex[ingredient]; !exists {
				availableElements[ingredient] = true 
			}
		}
	}
	
	sortedSteps := []RecipeStep{}
	for len(sortedSteps) < len(path.Steps) {
		stepAdded := false
		for _, step := range path.Steps {
			alreadyAdded := false
			for _, sortedStep := range sortedSteps {
				if sortedStep.Element == step.Element {
					alreadyAdded = true
					break
				}
			}
			if alreadyAdded {
				continue
			}
			
			canCreate := true
			for _, ingredient := range step.Ingredients {
				if !availableElements[ingredient] {
					canCreate = false
					break
				}
			}
			
			if canCreate {
				sortedSteps = append(sortedSteps, step)
				availableElements[step.Element] = true
				stepAdded = true
			}
		}
		
		if !stepAdded {
			break
		}
	}
	
	path.Steps = sortedSteps
}

func (ec *ElementController) GetElementCreationInstructions(targetName string) (string, error) {
	path, err := ec.FindPathToElement(targetName)
	if err != nil {
		return "", err
	}
	
	if len(path.Steps) == 0 {
		return fmt.Sprintf("%s is a tier 0 element and cannot be created from other elements.", targetName), nil
	}
	
	instructions := fmt.Sprintf("To create %s:\n", targetName)
	
	for i, step := range path.Steps {
		instructions += fmt.Sprintf("%d. Combine %s and %s to create %s\n", 
			i+1, 
			step.Ingredients[0], 
			step.Ingredients[1],
			step.Element)
	}
	
	return instructions, nil
}

func (ec *ElementController) FindAllPossiblePaths(targetName string) ([]ElementPath, error) {
	return nil, errors.New("not implemented yet")
}

func (ec *ElementController) GetElementDependencyTree(targetName string) (string, error) {
	path, err := ec.FindPathToElement(targetName)
	if err != nil {
		return "", err
	}
	
	if len(path.Steps) == 0 {
		return fmt.Sprintf("%s (tier 0)", targetName), nil
	}
	
	elementToIngredients := make(map[string][]string)
	for _, step := range path.Steps {
		elementToIngredients[step.Element] = step.Ingredients
	}
	
	treeText := buildTreeRepresentation(targetName, elementToIngredients, "", true)
	
	return treeText, nil
}

func buildTreeRepresentation(elementName string, elementToIngredients map[string][]string, indent string, isLast bool) string {
	var sb strings.Builder
	
	if isLast {
		sb.WriteString(indent + "└── " + elementName + "\n")
		indent += "    "
	} else {
		sb.WriteString(indent + "├── " + elementName + "\n")
		indent += "│   "
	}
	
	ingredients, exists := elementToIngredients[elementName]
	if !exists {
		return sb.String()
	}
	
	for i, ingredient := range ingredients {
		isLastIngredient := (i == len(ingredients)-1)
		sb.WriteString(buildTreeRepresentation(ingredient, elementToIngredients, indent, isLastIngredient))
	}
	
	return sb.String()
}
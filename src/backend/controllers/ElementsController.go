package ElementsController

import (
	ElementsModel "backend/models"
	"fmt"
)

type ElementController struct {
}

type TreeNode struct {
	Element     string      `json:"element"`
	Ingredients []*TreeNode `json:"ingredients"`
	Recipe      []string    `json:"recipe"`
}

func NewElementController(filePath string) (*ElementController, error) {
	err := ElementsModel.GetInstance().Initialize(filePath)
	if err != nil {
		return nil, err
	}
	return &ElementController{}, nil
}

func (ec *ElementController) FindNRecipes(targetName string, n int, useBFS bool) ([]*TreeNode, error) {
	node, err := ElementsModel.GetInstance().GetElementNode(targetName)
	if err != nil {
		return nil, err
	}
	if useBFS {
		return FindNRecipesForElementBFS(node, n), nil
	} else {
		return FindNRecipesForElementDFS(node, n), nil
	}
}

func FindNRecipesForElementDFS(target *ElementsModel.ElementNode, maxCount int) []*TreeNode {
	type Frame struct {
		Tree     *TreeNode
		Node     *ElementsModel.ElementNode
		Relation *ElementsModel.ElementRelation
		ChildIdx int
	}

	var results []*TreeNode
	seen := map[string]bool{}
	stack := []Frame{}

	rootName := target.Element.Name

	// Initialize stack with all top-level recipes for the target
	for _, rel := range target.Parents {
		if len(rel.Recipe.Ingredients) != 2 {
			continue
		}
		tree := &TreeNode{
			Element:     rootName,
			Recipe:      rel.Recipe.Ingredients,
			Ingredients: make([]*TreeNode, 2),
		}
		stack = append(stack, Frame{
			Tree:     tree,
			Node:     target,
			Relation: rel,
			ChildIdx: 0,
		})
	}

	for len(stack) > 0 && len(results) < maxCount {
		frame := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if frame.ChildIdx == len(frame.Relation.SourceNodes) {
			// Finished building this tree
			if isTreeComplete(frame.Tree) && frame.Tree.Element == rootName {
				key := TreeKey(frame.Tree)
				if !seen[key] {
					seen[key] = true
					results = append(results, frame.Tree)
				}
			}
			continue
		}

		childNode := frame.Relation.SourceNodes[frame.ChildIdx]
		if childNode == nil {
			continue
		}

		// Push parent frame back to continue after resolving this child
		stack = append(stack, Frame{
			Tree:     frame.Tree,
			Node:     frame.Node,
			Relation: frame.Relation,
			ChildIdx: frame.ChildIdx + 1,
		})

		if childNode.Element.Tier == 0 {
			// Base element: create leaf
			frame.Tree.Ingredients[frame.ChildIdx] = &TreeNode{
				Element:     childNode.Element.Name,
				Recipe:      nil,
				Ingredients: []*TreeNode{},
			}
			continue
		}

		// Expand all recipes for this childNode
		for _, childRel := range childNode.Parents {
			if len(childRel.Recipe.Ingredients) != 2 {
				continue
			}
			subTree := &TreeNode{
				Element:     childNode.Element.Name,
				Recipe:      childRel.Recipe.Ingredients,
				Ingredients: make([]*TreeNode, 2),
			}
			// Attach to current tree
			frame.Tree.Ingredients[frame.ChildIdx] = subTree
			// Push child subtree exploration
			stack = append(stack, Frame{
				Tree:     subTree,
				Node:     childNode,
				Relation: childRel,
				ChildIdx: 0,
			})
		}
	}

	return results
}

func isTreeComplete(tree *TreeNode) bool {
	if tree == nil {
		return false
	}

	stack := []*TreeNode{tree}

	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if curr == nil {
			return false
		}

		if curr.Recipe == nil {
			continue // Base element, complete
		}

		if len(curr.Ingredients) != 2 {
			return false
		}

		for _, child := range curr.Ingredients {
			if child == nil {
				return false
			}
			stack = append(stack, child)
		}
	}

	return true
}

func FindNRecipesForElementBFS(target *ElementsModel.ElementNode, maxCount int) []*TreeNode {
	type Frame struct {
		Tree *TreeNode
	}

	var results []*TreeNode
	seen := map[string]bool{}
	queue := []Frame{}

	rootName := target.Element.Name

	// Seed the queue with all recipes for the target element
	for _, relation := range target.Parents {
		if len(relation.Recipe.Ingredients) != 2 {
			continue
		}
		tree := &TreeNode{
			Element: rootName,
			Recipe:  relation.Recipe.Ingredients,
			Ingredients: []*TreeNode{
				{Element: relation.Recipe.Ingredients[0]},
				{Element: relation.Recipe.Ingredients[1]},
			},
		}
		queue = append(queue, Frame{Tree: tree})
	}

	for len(queue) > 0 && len(results) < maxCount {
		frame := queue[0]
		queue = queue[1:]

		tree := frame.Tree
		stack := []*TreeNode{tree}
		complete := true

		// BFS-like traversal to expand non-base ingredients
		for len(stack) > 0 {
			curr := stack[0]
			stack = stack[1:]

			if curr.Recipe == nil || len(curr.Ingredients) != 2 {
				continue
			}

			for i := 0; i < 2; i++ {
				child := curr.Ingredients[i]
				childNode, err := ElementsModel.GetInstance().GetElementNode(child.Element)
				if err != nil || childNode == nil {
					complete = false
					continue
				}

				// If base element, convert to full leaf node
				if childNode.Element.Tier == 0 {
					curr.Ingredients[i] = &TreeNode{
						Element:     child.Element,
						Recipe:      nil,
						Ingredients: []*TreeNode{},
					}
					continue
				}

				// Expand with first valid recipe
				found := false
				for _, rel := range childNode.Parents {
					if len(rel.Recipe.Ingredients) != 2 {
						continue
					}
					subTree := &TreeNode{
						Element: child.Element,
						Recipe:  rel.Recipe.Ingredients,
						Ingredients: []*TreeNode{
							{Element: rel.Recipe.Ingredients[0]},
							{Element: rel.Recipe.Ingredients[1]},
						},
					}
					curr.Ingredients[i] = subTree
					stack = append(stack, subTree)
					found = true
					break
				}
				if !found {
					complete = false
				}
			}
		}

		if complete && tree.Element == rootName && isTreeComplete(tree) {
			key := TreeKey(tree)
			if !seen[key] {
				seen[key] = true
				results = append(results, tree)
			}
		}
	}

	return results
}

func TreeKey(tree *TreeNode) string {
	if tree == nil {
		return ""
	}
	if tree.Recipe == nil || len(tree.Ingredients) == 0 {
		return tree.Element
	}
	left := TreeKey(tree.Ingredients[0])
	right := TreeKey(tree.Ingredients[1])
	if left > right {
		left, right = right, left
	}
	return fmt.Sprintf("%s(%s,%s)", tree.Element, left, right)
}

func PrintRecipeTree(tree *TreeNode, indent string) {
	if tree == nil {
		return
	}
	if tree.Recipe != nil {
		fmt.Printf("%s%s = %s + %s\n", indent, tree.Element, tree.Recipe[0], tree.Recipe[1])
	} else {
		fmt.Printf("%s%s\n", indent, tree.Element)
	}
	for _, child := range tree.Ingredients {
		PrintRecipeTree(child, indent+"  ")
	}
}

// func FindNRecipesForElementDFS(target *ElementsModel.ElementNode) []*TreeNode {
// 	type StackState struct {
// 		Node   *ElementsModel.ElementNode
// 		Tree   *TreeNode
// 		Parent *TreeNode
// 	}

// 	var finalResults []*TreeNode
// 	stack := []*StackState{}

// 	for _, rel := range target.Parents {
// 		root := &TreeNode{
// 			Element: target.Element.Name,
// 			Recipe:  rel.Recipe.Ingredients,
// 		}
// 		stack = append(stack, &StackState{Node: target, Tree: root, Parent: nil})
// 	}

// 	for len(stack) > 0 {
// 		current := stack[len(stack)-1]
// 		stack = stack[:len(stack)-1]

// 		for _, rel := range current.Node.Parents {
// 			children := []*TreeNode{}

// 			for _, src := range rel.SourceNodes {
// 				child := &TreeNode{
// 					Element: src.Element.Name,
// 				}
// 				children = append(children, child)
// 				stack = append(stack, &StackState{
// 					Node:   src,
// 					Tree:   child,
// 					Parent: current.Tree,
// 				})
// 			}

// 			current.Tree.Ingredients = children
// 			if current.Parent == nil {
// 				finalResults = append(finalResults, current.Tree)
// 				return finalResults
// 			}
// 		}
// 	}
// 	return finalResults
// }

// func FindNRecipesForElementBFS(target *ElementsModel.ElementNode) []*TreeNode {
// 	type QueueState struct {
// 		Node   *ElementsModel.ElementNode
// 		Tree   *TreeNode
// 		Parent *TreeNode
// 	}

// 	var finalResults []*TreeNode
// 	queue := []*QueueState{}

// 	for _, rel := range target.Parents {
// 		root := &TreeNode{
// 			Element: target.Element.Name,
// 			Recipe:  rel.Recipe.Ingredients,
// 		}
// 		queue = append(queue, &QueueState{Node: target, Tree: root, Parent: nil})
// 	}

// 	for len(queue) > 0 {
// 		current := queue[0]
// 		queue = queue[1:]

// 		for _, rel := range current.Node.Parents {
// 			children := []*TreeNode{}

// 			for _, src := range rel.SourceNodes {
// 				child := &TreeNode{
// 					Element: src.Element.Name,
// 				}
// 				children = append(children, child)
// 				queue = append(queue, &QueueState{
// 					Node:   src,
// 					Tree:   child,
// 					Parent: current.Tree,
// 				})
// 			}

// 			current.Tree.Ingredients = children
// 			if current.Parent == nil {
// 				finalResults = append(finalResults, current.Tree)
// 				return finalResults
// 			}
// 		}
// 	}
// 	return finalResults
// }

// func PrintTree(node *TreeNode, prefix string, isLast bool) {
// 	if node == nil {
// 		return
// 	}

// 	connector := "├──"
// 	if isLast {
// 		connector = "└──"
// 	}

// 	// Show element name + recipe if available
// 	if len(node.Recipe) == 2 {
// 		fmt.Printf("%s%s %s (from: %s + %s)\n", prefix, connector, node.Element, node.Recipe[0], node.Recipe[1])
// 	} else {
// 		fmt.Printf("%s%s %s\n", prefix, connector, node.Element)
// 	}

// 	newPrefix := prefix
// 	if isLast {
// 		newPrefix += "    "
// 	} else {
// 		newPrefix += "│   "
// 	}

// 	for i, child := range node.Ingredients {
// 		PrintTree(child, newPrefix, i == len(node.Ingredients)-1)
// 	}
// }

// type RecipeStep struct {
// 	Element     string
// 	Ingredients []string
// }

// type ElementPath struct {
// 	TargetElement string
// 	Steps         []RecipeStep
// }

// func (ec *ElementController) FindPathToElement(targetName string) (*ElementPath, error) {
// 	elementsService := ElementsModel.GetInstance()

// 	targetNode, err := elementsService.GetElementNode(targetName)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if targetNode.Element.Tier == 0 {
// 		return &ElementPath{
// 			TargetElement: targetName,
// 			Steps:         []RecipeStep{},
// 		}, nil
// 	}

// 	path, err := findOptimalPath(targetNode)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return path, nil
// }

// func findOptimalPath(targetNode *ElementsModel.ElementNode) (*ElementPath, error) {
// 	path := &ElementPath{
// 		TargetElement: targetNode.Element.Name,
// 		Steps:         []RecipeStep{},
// 	}

// 	processedElements := make(map[string]bool)

// 	queue := []*ElementsModel.ElementNode{targetNode}

// 	elementToStep := make(map[string]RecipeStep)

// 	elementToTierSum := make(map[string]int)
// 	elementToTierSum[targetNode.Element.Name] = targetNode.Element.Tier

// 	for len(queue) > 0 {
// 		currentNode := queue[0]
// 		queue = queue[1:]

// 		if processedElements[currentNode.Element.Name] {
// 			continue
// 		}
// 		processedElements[currentNode.Element.Name] = true

// 		if currentNode.Element.Tier == 0 {
// 			continue
// 		}

// 		var bestRelation *ElementsModel.ElementRelation
// 		lowestTierSum := 9999999

// 		for _, relation := range currentNode.Parents {
// 			tierSum := 0
// 			allIngredientsExist := true

// 			for _, sourceNode := range relation.SourceNodes {
// 				if sourceNode == nil || sourceNode.Element == nil {
// 					allIngredientsExist = false
// 					break
// 				}
// 				tierSum += sourceNode.Element.Tier
// 			}

// 			if allIngredientsExist && tierSum < lowestTierSum {
// 				lowestTierSum = tierSum
// 				bestRelation = relation
// 			}
// 		}

// 		if bestRelation != nil {
// 			step := RecipeStep{
// 				Element:     currentNode.Element.Name,
// 				Ingredients: bestRelation.Recipe.Ingredients,
// 			}

// 			elementToStep[currentNode.Element.Name] = step

// 			for _, sourceNode := range bestRelation.SourceNodes {
// 				if sourceNode.Element.Tier > 0 {
// 					queue = append(queue, sourceNode)
// 				}
// 			}
// 		}
// 	}

// 	elementStack := []string{targetNode.Element.Name}
// 	visitedElements := make(map[string]bool)

// 	for len(elementStack) > 0 {
// 		currentElement := elementStack[len(elementStack)-1]
// 		elementStack = elementStack[:len(elementStack)-1]

// 		if visitedElements[currentElement] {
// 			continue
// 		}
// 		visitedElements[currentElement] = true

// 		step, exists := elementToStep[currentElement]
// 		if exists {
// 			path.Steps = append(path.Steps, step)

// 			for _, ingredient := range step.Ingredients {
// 				if elementToStep[ingredient].Element != "" {
// 					elementStack = append(elementStack, ingredient)
// 				}
// 			}
// 		}
// 	}

// 	sortSteps(path)

// 	return path, nil
// }

// func sortSteps(path *ElementPath) {
// 	elementToStepIndex := make(map[string]int)
// 	for i, step := range path.Steps {
// 		elementToStepIndex[step.Element] = i
// 	}

// 	availableElements := make(map[string]bool)

// 	for _, step := range path.Steps {
// 		for _, ingredient := range step.Ingredients {
// 			if _, exists := elementToStepIndex[ingredient]; !exists {
// 				availableElements[ingredient] = true
// 			}
// 		}
// 	}

// 	sortedSteps := []RecipeStep{}
// 	for len(sortedSteps) < len(path.Steps) {
// 		stepAdded := false
// 		for _, step := range path.Steps {
// 			alreadyAdded := false
// 			for _, sortedStep := range sortedSteps {
// 				if sortedStep.Element == step.Element {
// 					alreadyAdded = true
// 					break
// 				}
// 			}
// 			if alreadyAdded {
// 				continue
// 			}

// 			canCreate := true
// 			for _, ingredient := range step.Ingredients {
// 				if !availableElements[ingredient] {
// 					canCreate = false
// 					break
// 				}
// 			}

// 			if canCreate {
// 				sortedSteps = append(sortedSteps, step)
// 				availableElements[step.Element] = true
// 				stepAdded = true
// 			}
// 		}

// 		if !stepAdded {
// 			break
// 		}
// 	}

// 	path.Steps = sortedSteps
// }

// func (ec *ElementController) GetElementCreationInstructions(targetName string) (string, error) {
// 	path, err := ec.FindPathToElement(targetName)
// 	if err != nil {
// 		return "", err
// 	}

// 	if len(path.Steps) == 0 {
// 		return fmt.Sprintf("%s is a tier 0 element and cannot be created from other elements.", targetName), nil
// 	}

// 	instructions := fmt.Sprintf("To create %s:\n", targetName)

// 	for i, step := range path.Steps {
// 		instructions += fmt.Sprintf("%d. Combine %s and %s to create %s\n",
// 			i+1,
// 			step.Ingredients[0],
// 			step.Ingredients[1],
// 			step.Element)
// 	}

// 	return instructions, nil
// }

// func (ec *ElementController) FindAllPossiblePaths(targetName string) ([]ElementPath, error) {
// 	return nil, errors.New("not implemented yet")
// }

// func (ec *ElementController) GetElementDependencyTree(targetName string) (string, error) {
// 	path, err := ec.FindPathToElement(targetName)
// 	if err != nil {
// 		return "", err
// 	}

// 	if len(path.Steps) == 0 {
// 		return fmt.Sprintf("%s (tier 0)", targetName), nil
// 	}

// 	elementToIngredients := make(map[string][]string)
// 	for _, step := range path.Steps {
// 		elementToIngredients[step.Element] = step.Ingredients
// 	}

// 	treeText := buildTreeRepresentation(targetName, elementToIngredients, "", true)

// 	return treeText, nil
// }

// func buildTreeRepresentation(elementName string, elementToIngredients map[string][]string, indent string, isLast bool) string {
// 	var sb strings.Builder

// 	if isLast {
// 		sb.WriteString(indent + "└── " + elementName + "\n")
// 		indent += "    "
// 	} else {
// 		sb.WriteString(indent + "├── " + elementName + "\n")
// 		indent += "│   "
// 	}

// 	ingredients, exists := elementToIngredients[elementName]
// 	if !exists {
// 		return sb.String()
// 	}

// 	for i, ingredient := range ingredients {
// 		isLastIngredient := (i == len(ingredients)-1)
// 		sb.WriteString(buildTreeRepresentation(ingredient, elementToIngredients, indent, isLastIngredient))
// 	}

// 	return sb.String()
// }

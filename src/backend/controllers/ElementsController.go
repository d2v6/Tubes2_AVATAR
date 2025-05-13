package elementsController

import (
	elementsModel "backend/models"
	"fmt"
	"sort"
	"strconv"
	"time"
)

type ElementController struct {
}

type TreeNode struct {
	Name   string
	Recipe []*TreeNode
}

var NodesVisited int64

func NewElementController(filePath string) (*ElementController, error) {
	err := elementsModel.GetInstance().Initialize(filePath)
	if err != nil {
		return nil, err
	}
	return &ElementController{}, nil
}

func (ec *ElementController) GetAllElementsTiers() (map[string][]string, error) {
	elements := elementsModel.GetInstance().GetAllElements()

	tierGroups := make(map[string][]string)

	for _, element := range elements {
		tierStr := strconv.Itoa(element.Tier)
		tierGroups[tierStr] = append(tierGroups[tierStr], element.Name)
	}

	for _, elements := range tierGroups {
		sort.Strings(elements)
	}

	return tierGroups, nil
}

func StartDFS(targetName string, n int, treeChan chan *TreeNode) (*TreeNode, time.Duration) {
	start := time.Now()
	node, err := elementsModel.GetInstance().GetElementNode(targetName)
	if err != nil {
		return nil, 0
	}

	trees := dfs(node, int64(n), treeChan)
	return mergeTree(trees), time.Since(start)
}

func StartBFS(targetName string, n int, treeChan chan *TreeNode) (*TreeNode, time.Duration) {
	start := time.Now()
	node, err := elementsModel.GetInstance().GetElementNode(targetName)
	if err != nil {
		return nil, 0
	}

	trees := bfs(node, int64(n), treeChan)
	return mergeTree(trees), time.Since(start)
}

func dfs(target *elementsModel.ElementNode, n int64, treeChan chan *TreeNode) []*TreeNode {
	if target == nil {
		return nil
	}

	if len(target.Parents) <= 0 || target.Element.Tier == 0 {
		node := &TreeNode{
			Name: target.Element.Name,
		}
		return []*TreeNode{node}
	}

	var results []*TreeNode

	for _, recipe := range target.Parents {
		leftTrees := dfs(recipe.SourceNodes[0], n, treeChan)
		rightTrees := dfs(recipe.SourceNodes[1], n, treeChan)

		for _, left := range leftTrees {
			for _, right := range rightTrees {
				node := &TreeNode{
					Name:   target.Element.Name,
					Recipe: []*TreeNode{left, right},
				}

				treeChan <- node

				results = append(results, node)
				if len(results) >= int(n) {
					return results
				}
			}
			if len(results) >= int(n) {
				return results
			}
		}
		if len(results) >= int(n) {
			return results
		}
	}

	return results
}

func bfs(target *elementsModel.ElementNode, n int64, treeChan chan *TreeNode) []*TreeNode {
	if target == nil {
		return nil
	}

	type QueueItem struct {
		Element string
	}

	if len(target.Parents) <= 0 || target.Element.Tier == 0 {
		node := &TreeNode{
			Name: target.Element.Name,
		}
		return []*TreeNode{node}
	}

	elementToTree := make(map[string][]*TreeNode)
	processedElements := make(map[string]bool)

	currentQueue := []*QueueItem{}

	for _, recipe := range target.Parents {
		currentQueue = append(currentQueue,
			&QueueItem{Element: recipe.Recipe.Ingredients[0]},
			&QueueItem{Element: recipe.Recipe.Ingredients[1]},
		)
	}

	var results []*TreeNode

	for len(currentQueue) > 0 {
		nextQueue := []*QueueItem{}
		for len(currentQueue) > 0 {
			current := currentQueue[0]
			currentQueue = currentQueue[1:]

			if processedElements[current.Element] {
				continue
			}

			currentNode, err := elementsModel.GetInstance().GetElementNode(current.Element)
			if err != nil || currentNode == nil {
				continue
			}

			if currentNode.Element.Tier == 0 || len(currentNode.Parents) == 0 {
				tree := &TreeNode{
					Name: currentNode.Element.Name,
				}
			
				treeChan <- tree
			
				elementToTree[currentNode.Element.Name] = []*TreeNode{tree}
				processedElements[currentNode.Element.Name] = true
			}

			allReady := true
			for _, recipe := range currentNode.Parents {
				if !processedElements[recipe.Recipe.Ingredients[0]] {
					nextQueue = append(nextQueue, &QueueItem{Element: recipe.Recipe.Ingredients[0]})
					allReady = false
				}
				if !processedElements[recipe.Recipe.Ingredients[1]] {
					nextQueue = append(nextQueue, &QueueItem{Element: recipe.Recipe.Ingredients[1]})
					allReady = false
				}
			}

			if !allReady {
				nextQueue = append(nextQueue, current)
				continue
			}

			var trees []*TreeNode
			for _, recipe := range currentNode.Parents {
				leftTrees := elementToTree[recipe.Recipe.Ingredients[0]]
				rightTrees := elementToTree[recipe.Recipe.Ingredients[1]]
				if leftTrees == nil || rightTrees == nil {
					continue
				}
				for _, left := range leftTrees {
					for _, right := range rightTrees {
						node := &TreeNode{
							Name:   currentNode.Element.Name,
							Recipe: []*TreeNode{left, right},
						}

						treeChan <- node

						trees = append(trees, node)
					}
				}
			}
			if len(trees) > 0 {
				elementToTree[currentNode.Element.Name] = trees
				processedElements[currentNode.Element.Name] = true
			}
		}
		currentQueue = nextQueue
	}

	for _, recipe := range target.Parents {
		leftTrees := elementToTree[recipe.Recipe.Ingredients[0]]
		rightTrees := elementToTree[recipe.Recipe.Ingredients[1]]
		if leftTrees == nil || rightTrees == nil {
			continue
		}

		for _, left := range leftTrees {
			for _, right := range rightTrees {
				node := &TreeNode{
					Name:   target.Element.Name,
					Recipe: []*TreeNode{left, right},
				}
				results = append(results, node)
				if len(results) >= int(n) {
					return results
				}
			}
		}
	}

	return results
}

func mergeTree(trees []*TreeNode) *TreeNode {
	root := &TreeNode{
		Name:   "Root",
		Recipe: trees,
	}
	return root
}

func PrintRecipeTree(tree *TreeNode, prefix string, isLast bool) {
	if tree == nil {
		return
	}

	connector := "├── "
	if isLast {
		connector = "└── "
	}

	fmt.Printf("%s%s%s", prefix, connector, tree.Name)

	// Print the recipe if there are exactly 2 ingredients (left and right)
	if len(tree.Recipe) == 2 {
		fmt.Printf(" = %s + %s", tree.Recipe[0].Name, tree.Recipe[1].Name)
	}
	fmt.Println()

	// Update the prefix for the next level of the tree
	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}

	for i, child := range tree.Recipe {
		PrintRecipeTree(child, newPrefix, i == len(tree.Recipe)-1)
	}
}

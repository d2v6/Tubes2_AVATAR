package elementsController

import (
	elementsModel "backend/models"
	"fmt"
	"sort"
	"strconv"
	"sync"
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

func dfs(target *elementsModel.ElementNode, limit int64, treeChan chan *TreeNode) []*TreeNode {
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
	var resultsMutex sync.Mutex
	var wg sync.WaitGroup

	for _, recipe := range target.Parents {
		wg.Add(1)
		go func(recipe *elementsModel.ElementRelation) {
			defer wg.Done()
			if(len(recipe.SourceNodes)<2){
				return;
			}
			leftTrees := dfs(recipe.SourceNodes[0], limit, treeChan)
			rightTrees := dfs(recipe.SourceNodes[1], limit, treeChan)

			localResults := []*TreeNode{}
			for _, left := range leftTrees {
				for _, right := range rightTrees {
					node := &TreeNode{
						Name:   target.Element.Name,
						Recipe: []*TreeNode{left, right},
					}

					treeChan <- node
					localResults = append(localResults, node)
					
					resultsMutex.Lock()
					if len(results) >= int(limit) {
						resultsMutex.Unlock()
						return
					}
					resultsMutex.Unlock()
				}
			}

			resultsMutex.Lock()
			results = append(results, localResults...)
			if len(results) > int(limit) {
				results = results[:int(limit)]
			}
			resultsMutex.Unlock()
		}(recipe)
	}

	wg.Wait()
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
	var elementToTreeMutex sync.RWMutex
	
	processedElements := make(map[string]bool)
	var processedMutex sync.RWMutex

	currentQueue := []*QueueItem{}

	for _, recipe := range target.Parents {
		currentQueue = append(currentQueue,
			&QueueItem{Element: recipe.Recipe.Ingredients[0]},
			&QueueItem{Element: recipe.Recipe.Ingredients[1]},
		)
	}

	var results []*TreeNode
	var resultsMutex sync.Mutex

	for len(currentQueue) > 0 {
		nextQueue := []*QueueItem{}
		var nextQueueMutex sync.Mutex
		
		var wg sync.WaitGroup
		currentQueueCopy := make([]*QueueItem, len(currentQueue))
		copy(currentQueueCopy, currentQueue)
		
		for _, current := range currentQueueCopy {
			wg.Add(1)
			go func(current *QueueItem) {
				defer wg.Done()
				
				processedMutex.RLock()
				alreadyProcessed := processedElements[current.Element]
				processedMutex.RUnlock()
				
				if alreadyProcessed {
					return
				}

				currentNode, err := elementsModel.GetInstance().GetElementNode(current.Element)
				if err != nil || currentNode == nil {
					return
				}

				if currentNode.Element.Tier == 0 || len(currentNode.Parents) == 0 {
					tree := &TreeNode{
						Name: currentNode.Element.Name,
					}
				
					treeChan <- tree
				
					elementToTreeMutex.Lock()
					elementToTree[currentNode.Element.Name] = []*TreeNode{tree}
					elementToTreeMutex.Unlock()
					
					processedMutex.Lock()
					processedElements[currentNode.Element.Name] = true
					processedMutex.Unlock()
					return
				}

				allReady := true
				nextItems := []*QueueItem{}
				
				for _, recipe := range currentNode.Parents {
					processedMutex.RLock()
					leftProcessed := processedElements[recipe.Recipe.Ingredients[0]]
					rightProcessed := processedElements[recipe.Recipe.Ingredients[1]]
					processedMutex.RUnlock()
					
					if !leftProcessed {
						nextItems = append(nextItems, &QueueItem{Element: recipe.Recipe.Ingredients[0]})
						allReady = false
					}
					if !rightProcessed {
						nextItems = append(nextItems, &QueueItem{Element: recipe.Recipe.Ingredients[1]})
						allReady = false
					}
				}

				if !allReady {
					nextItems = append(nextItems, current)
					
					nextQueueMutex.Lock()
					nextQueue = append(nextQueue, nextItems...)
					nextQueueMutex.Unlock()
					return
				}

				var trees []*TreeNode
				for _, recipe := range currentNode.Parents {
					elementToTreeMutex.RLock()
					leftTrees := elementToTree[recipe.Recipe.Ingredients[0]]
					rightTrees := elementToTree[recipe.Recipe.Ingredients[1]]
					elementToTreeMutex.RUnlock()
					
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
					elementToTreeMutex.Lock()
					elementToTree[currentNode.Element.Name] = trees
					elementToTreeMutex.Unlock()
					
					processedMutex.Lock()
					processedElements[currentNode.Element.Name] = true
					processedMutex.Unlock()
				}
			}(current)
		}
		
		wg.Wait()
		currentQueue = nextQueue
	}

	var targetWg sync.WaitGroup
	for _, recipe := range target.Parents {
		targetWg.Add(1)
		go func(recipe *elementsModel.ElementRelation) {
			defer targetWg.Done()
			
			elementToTreeMutex.RLock()
			leftTrees := elementToTree[recipe.Recipe.Ingredients[0]]
			rightTrees := elementToTree[recipe.Recipe.Ingredients[1]]
			elementToTreeMutex.RUnlock()
			
			if leftTrees == nil || rightTrees == nil {
				return
			}

			localResults := []*TreeNode{}
			for _, left := range leftTrees {
				for _, right := range rightTrees {
					node := &TreeNode{
						Name:   target.Element.Name,
						Recipe: []*TreeNode{left, right},
					}
					localResults = append(localResults, node)
					
					resultsMutex.Lock()
					if len(results) >= int(n) {
						resultsMutex.Unlock()
						return
					}
					resultsMutex.Unlock()
				}
			}
			
			resultsMutex.Lock()
			results = append(results, localResults...)
			if len(results) > int(n) {
				results = results[:int(n)]
			}
			resultsMutex.Unlock()
		}(recipe)
	}
	
	targetWg.Wait()
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

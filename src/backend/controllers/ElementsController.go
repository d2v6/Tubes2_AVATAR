package elementsController

import (
	elementsModel "backend/models"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ElementController struct {
}

type TreeNode struct {
	Element     string      `json:"element"`
	Ingredients []*TreeNode `json:"ingredients"`
	Recipe      []string    `json:"recipe"`
}

func NewElementController(filePath string) (*ElementController, error) {
	err := elementsModel.GetInstance().Initialize(filePath)
	if err != nil {
		return nil, err
	}
	return &ElementController{}, nil
}

func (ec *ElementController) FindNRecipes(targetName string, n int, useBFS bool) (*TreeNode, int, time.Duration, error) {
	node, err := elementsModel.GetInstance().GetElementNode(targetName)
	if err != nil {
		return nil, 0, 0, err
	}

	start := time.Now()
	var trees []*TreeNode
	var nodesVisited int

	if useBFS {
		trees, nodesVisited = findRecipesBFS(node, n)
	} else {
		trees, nodesVisited = findRecipesDFS(node, n)
	}

	duration := time.Since(start)
	result := mergeTrees(trees)
	return result, nodesVisited, duration, nil
}

// func findRecipesDFS(target *elementsModel.ElementNode, maxCount int) ([]*TreeNode, int) {
// 	type Frame struct {
// 		Tree  *TreeNode
// 		Stack []*TreeNode
// 	}

// 	var results []*TreeNode
// 	seen := sync.Map{}
// 	stack := []Frame{}
// 	nodesVisited := 0

// 	// Initialize with all root-level recipes
// 	for _, rel := range target.Parents {
// 		if len(rel.Recipe.Ingredients) != 2 {
// 			continue
// 		}
// 		tree := &TreeNode{
// 			Element: target.Element.Name,
// 			Recipe:  rel.Recipe.Ingredients,
// 			Ingredients: []*TreeNode{
// 				{Element: rel.Recipe.Ingredients[0]},
// 				{Element: rel.Recipe.Ingredients[1]},
// 			},
// 		}
// 		stack = append(stack, Frame{
// 			Tree:  tree,
// 			Stack: []*TreeNode{tree},
// 		})
// 	}

// 	for len(stack) > 0 && len(results) < maxCount {
// 		frame := stack[len(stack)-1]
// 		stack = stack[:len(stack)-1]

// 		if len(frame.Stack) == 0 {
// 			key := treeKey(frame.Tree)
// 			if _, loaded := seen.LoadOrStore(key, true); !loaded {
// 				results = append(results, frame.Tree)
// 			}
// 			continue
// 		}

// 		curr := frame.Stack[len(frame.Stack)-1]
// 		rest := frame.Stack[:len(frame.Stack)-1]
// 		nodesVisited++

// 		if curr.Recipe == nil || len(curr.Ingredients) != 2 {
// 			stack = append(stack, Frame{Tree: frame.Tree, Stack: rest})
// 			continue
// 		}

// 		leftName := curr.Ingredients[0].Element
// 		rightName := curr.Ingredients[1].Element

// 		leftNode, _ := elementsModel.GetInstance().GetElementNode(leftName)
// 		rightNode, _ := elementsModel.GetInstance().GetElementNode(rightName)

// 		leftTrees := expandIngredient(leftNode)
// 		rightTrees := expandIngredient(rightNode)

// 		for _, l := range leftTrees {
// 			for _, r := range rightTrees {
// 				clone := cloneTree(frame.Tree)
// 				node := findNodeByElement(clone, curr.Element)
// 				if node == nil || len(node.Ingredients) != 2 {
// 					continue
// 				}
// 				node.Ingredients[0] = l
// 				node.Ingredients[1] = r
// 				stack = append(stack, Frame{
// 					Tree:  clone,
// 					Stack: append([]*TreeNode{l, r}, rest...),
// 				})
// 			}
// 		}
// 	}

// 	return results, nodesVisited
// }

func findRecipesDFS(target *elementsModel.ElementNode, maxCount int) ([]*TreeNode, int) {
	type Frame struct {
		Tree  *TreeNode
		Stack []*TreeNode
	}

	var (
		resultsMu    sync.Mutex
		seen         sync.Map
		results      []*TreeNode
		nodesVisited int64
		wg           sync.WaitGroup
		resultChan   = make(chan *TreeNode, maxCount)
	)

	done := make(chan struct{})
	go func() {
		for tree := range resultChan {
			resultsMu.Lock()
			if len(results) < maxCount {
				results = append(results, tree)
			}
			resultsMu.Unlock()
		}
		close(done)
	}()

	for _, rel := range target.Parents {
		if len(rel.Recipe.Ingredients) != 2 {
			continue
		}

		wg.Add(1)
		go func(rel elementsModel.ElementRelation) {
			defer wg.Done()

			tree := &TreeNode{
				Element: target.Element.Name,
				Recipe:  rel.Recipe.Ingredients,
				Ingredients: []*TreeNode{
					{Element: rel.Recipe.Ingredients[0]},
					{Element: rel.Recipe.Ingredients[1]},
				},
			}
			stack := []Frame{{Tree: tree, Stack: []*TreeNode{tree}}}

			for len(stack) > 0 {
				resultsMu.Lock()
				if len(results) >= maxCount {
					resultsMu.Unlock()
					break
				}
				resultsMu.Unlock()

				frame := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				if len(frame.Stack) == 0 {
					key := treeKey(frame.Tree)
					if _, loaded := seen.LoadOrStore(key, true); !loaded {
						select {
						case resultChan <- frame.Tree:
						default:
						}
					}
					continue
				}

				curr := frame.Stack[len(frame.Stack)-1]
				rest := frame.Stack[:len(frame.Stack)-1]
				atomic.AddInt64(&nodesVisited, 1)

				if curr.Recipe == nil || len(curr.Ingredients) != 2 {
					stack = append(stack, Frame{Tree: frame.Tree, Stack: rest})
					continue
				}

				leftNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[0].Element)
				rightNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[1].Element)
				leftTrees := expandIngredient(leftNode)
				rightTrees := expandIngredient(rightNode)

				for _, l := range leftTrees {
					for _, r := range rightTrees {
						clone := cloneTree(frame.Tree)
						ptr := findNodeByElement(clone, curr.Element)
						if ptr == nil || len(ptr.Ingredients) != 2 {
							continue
						}
						ptr.Ingredients[0] = l
						ptr.Ingredients[1] = r

						stack = append(stack, Frame{
							Tree:  clone,
							Stack: append([]*TreeNode{l, r}, rest...),
						})
					}
				}
			}
		}(*rel)
	}

	wg.Wait()
	close(resultChan)
	<-done

	return results, int(nodesVisited)
}

func expandIngredient(node *elementsModel.ElementNode) []*TreeNode {
	if node == nil {
		return nil
	}
	if node.Element.Tier == 0 {
		return []*TreeNode{{Element: node.Element.Name}}
	}
	var out []*TreeNode
	for _, rel := range node.Parents {
		if len(rel.Recipe.Ingredients) != 2 {
			continue
		}
		out = append(out, &TreeNode{
			Element: node.Element.Name,
			Recipe:  rel.Recipe.Ingredients,
			Ingredients: []*TreeNode{
				{Element: rel.Recipe.Ingredients[0]},
				{Element: rel.Recipe.Ingredients[1]},
			},
		})
	}
	return out
}

func cloneTree(n *TreeNode) *TreeNode {
	if n == nil {
		return nil
	}
	copy := &TreeNode{
		Element:     n.Element,
		Recipe:      append([]string{}, n.Recipe...),
		Ingredients: make([]*TreeNode, len(n.Ingredients)),
	}
	for i, c := range n.Ingredients {
		copy.Ingredients[i] = cloneTree(c)
	}
	return copy
}

func findNodeByElement(n *TreeNode, target string) *TreeNode {
	stack := []*TreeNode{n}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur.Element == target {
			return cur
		}
		stack = append(stack, cur.Ingredients...)
	}
	return nil
}

func treeKey(n *TreeNode) string {
	if n == nil {
		return ""
	}
	if n.Recipe == nil || len(n.Ingredients) != 2 {
		return n.Element
	}

	left := treeKey(n.Ingredients[0])
	right := treeKey(n.Ingredients[1])

	return fmt.Sprintf("%s[%s+%s](%s,%s)", n.Element, n.Recipe[0], n.Recipe[1], left, right)
}

// func findRecipesBFS(target *elementsModel.ElementNode, maxCount int) ([]*TreeNode, int) {
// 	type Frame struct {
// 		Tree  *TreeNode
// 		Queue []*TreeNode
// 	}

// 	var results []*TreeNode
// 	seen := sync.Map{}
// 	nodesVisited := 0
// 	queue := []Frame{}

// 	// Initialize with all recipes for target
// 	for _, rel := range target.Parents {
// 		if len(rel.Recipe.Ingredients) != 2 {
// 			continue
// 		}
// 		tree := &TreeNode{
// 			Element: target.Element.Name,
// 			Recipe:  rel.Recipe.Ingredients,
// 			Ingredients: []*TreeNode{
// 				{Element: rel.Recipe.Ingredients[0]},
// 				{Element: rel.Recipe.Ingredients[1]},
// 			},
// 		}
// 		queue = append(queue, Frame{
// 			Tree:  tree,
// 			Queue: []*TreeNode{tree},
// 		})
// 	}

// 	for len(queue) > 0 && len(results) < maxCount {
// 		frame := queue[0]
// 		queue = queue[1:]

// 		if len(frame.Queue) == 0 {
// 			key := treeKey(frame.Tree)
// 			if _, loaded := seen.LoadOrStore(key, true); !loaded {
// 				results = append(results, frame.Tree)
// 			}
// 			continue
// 		}

// 		curr := frame.Queue[0]
// 		rest := frame.Queue[1:]
// 		nodesVisited++

// 		if curr.Recipe == nil || len(curr.Ingredients) != 2 {
// 			queue = append(queue, Frame{Tree: frame.Tree, Queue: rest})
// 			continue
// 		}

// 		leftName := curr.Ingredients[0].Element
// 		rightName := curr.Ingredients[1].Element

// 		leftNode, _ := elementsModel.GetInstance().GetElementNode(leftName)
// 		rightNode, _ := elementsModel.GetInstance().GetElementNode(rightName)

// 		leftTrees := expandIngredient(leftNode)
// 		rightTrees := expandIngredient(rightNode)

// 		for _, l := range leftTrees {
// 			for _, r := range rightTrees {
// 				clone := cloneTree(frame.Tree)
// 				node := findNodeByElement(clone, curr.Element)
// 				if node == nil || len(node.Ingredients) != 2 {
// 					continue
// 				}
// 				node.Ingredients[0] = l
// 				node.Ingredients[1] = r

// 				queue = append(queue, Frame{
// 					Tree:  clone,
// 					Queue: append(rest, l, r),
// 				})
// 			}
// 		}
// 	}

// 	return results, nodesVisited
// }

func findRecipesBFS(target *elementsModel.ElementNode, maxCount int) ([]*TreeNode, int) {
	type Frame struct {
		Tree  *TreeNode
		Queue []*TreeNode
	}

	var (
		results      []*TreeNode
		resultsMu    sync.Mutex
		nodesVisited int64
		wg           sync.WaitGroup
		seen         sync.Map
		resultChan   = make(chan *TreeNode, maxCount)
	)

	done := make(chan struct{})
	go func() {
		for tree := range resultChan {
			resultsMu.Lock()
			if len(results) < maxCount {
				results = append(results, tree)
			}
			resultsMu.Unlock()
		}
		close(done)
	}()

	for _, rel := range target.Parents {
		if len(rel.Recipe.Ingredients) != 2 {
			continue
		}

		wg.Add(1)
		go func(rel elementsModel.ElementRelation) {
			defer wg.Done()

			tree := &TreeNode{
				Element: target.Element.Name,
				Recipe:  rel.Recipe.Ingredients,
				Ingredients: []*TreeNode{
					{Element: rel.Recipe.Ingredients[0]},
					{Element: rel.Recipe.Ingredients[1]},
				},
			}
			queue := []Frame{{Tree: tree, Queue: []*TreeNode{tree}}}

			for len(queue) > 0 {
				resultsMu.Lock()
				if len(results) >= maxCount {
					resultsMu.Unlock()
					return
				}
				resultsMu.Unlock()

				frame := queue[0]
				queue = queue[1:]

				if len(frame.Queue) == 0 {
					key := treeKey(frame.Tree)
					if _, loaded := seen.LoadOrStore(key, true); !loaded {
						resultsMu.Lock()
						if len(results) < maxCount {
							resultChan <- frame.Tree
						}
						resultsMu.Unlock()
					}
					continue
				}

				curr := frame.Queue[0]
				rest := frame.Queue[1:]

				atomic.AddInt64(&nodesVisited, 1)

				if curr.Recipe == nil || len(curr.Ingredients) != 2 {
					queue = append(queue, Frame{Tree: frame.Tree, Queue: rest})
					continue
				}

				leftNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[0].Element)
				rightNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[1].Element)
				leftTrees := expandIngredient(leftNode)
				rightTrees := expandIngredient(rightNode)

				for _, l := range leftTrees {
					for _, r := range rightTrees {
						clone := cloneTree(frame.Tree)
						ptr := findNodeByElement(clone, curr.Element)
						if ptr == nil || len(ptr.Ingredients) != 2 {
							continue
						}
						ptr.Ingredients[0] = l
						ptr.Ingredients[1] = r
						queue = append(queue, Frame{
							Tree:  clone,
							Queue: append(rest, l, r),
						})
					}
				}
			}
		}(*rel)
	}

	wg.Wait()
	close(resultChan)
	<-done

	return results, int(nodesVisited)
}

func mergeTrees(trees []*TreeNode) *TreeNode {
	if len(trees) == 0 {
		return nil
	}
	return &TreeNode{
		Element:     "Root",
		Ingredients: trees,
		Recipe:      nil,
	}
}

func PrintRecipeTree(tree *TreeNode, prefix string, isLast bool) {
	if tree == nil {
		return
	}

	connector := "├── "
	if isLast {
		connector = "└── "
	}
	fmt.Printf("%s%s%s", prefix, connector, tree.Element)

	if len(tree.Recipe) == 2 {
		fmt.Printf(" = %s + %s", tree.Recipe[0], tree.Recipe[1])
	}
	fmt.Println()

	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}

	for i, child := range tree.Ingredients {
		PrintRecipeTree(child, newPrefix, i == len(tree.Ingredients)-1)
	}
}

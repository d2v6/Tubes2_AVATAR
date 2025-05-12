package elementsController

import (
	elementsModel "backend/models"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
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

type RecipeStats struct {
	NodesVisited int           `json:"nodesVisited"`
	Duration     time.Duration `json:"duration"`
}

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

func StreamRecipesDFS(target *elementsModel.ElementNode, maxCount int, resultChan chan<- *TreeNode) ([]*TreeNode, int, time.Duration) {
	start := time.Now()
	trees, nodesVisited := streamDFS(target, maxCount, resultChan)
	duration := time.Since(start)
	return trees, nodesVisited, duration
}

func StreamRecipesBFS(target *elementsModel.ElementNode, maxCount int, resultChan chan<- *TreeNode) ([]*TreeNode, int, time.Duration) {
	start := time.Now()
	trees, nodesVisited := streamBFS(target, maxCount, resultChan)
	duration := time.Since(start)
	return trees, nodesVisited, duration
}

func MergeTreesFromChannel(treeChan <-chan *TreeNode) *TreeNode {
	var trees []*TreeNode
	for tree := range treeChan {
		trees = append(trees, tree)
	}
	return mergeTrees(trees)
}

func streamDFS(target *elementsModel.ElementNode, maxCount int, resultChan chan<- *TreeNode) ([]*TreeNode, int) {
	type Frame struct {
		Tree  *TreeNode
		Stack []*TreeNode
	}

	var (
		resultsMu    sync.Mutex
		seen         sync.Map
		results      = make([]*TreeNode, 0, maxCount)
		nodesVisited int64
		wg           sync.WaitGroup
		localChan    = make(chan *TreeNode, maxCount*2) 
	)

	maxWorkers := runtime.GOMAXPROCS(0) * 2
	workChan := make(chan elementsModel.ElementRelation, maxWorkers*4)

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			localStack := make([]Frame, 0, 100)

			for rel := range workChan {
				if len(rel.Recipe.Ingredients) != 2 {
					continue
				}

				tree := &TreeNode{
					Element: target.Element.Name,
					Recipe:  rel.Recipe.Ingredients,
					Ingredients: []*TreeNode{
						{Element: rel.Recipe.Ingredients[0]},
						{Element: rel.Recipe.Ingredients[1]},
					},
				}

				localStack = localStack[:0]
				localStack = append(localStack, Frame{Tree: tree, Stack: []*TreeNode{tree}})

				for len(localStack) > 0 {
					resultsMu.Lock()
					full := len(results) >= maxCount
					resultsMu.Unlock()

					if full {
						break
					}

					lastIdx := len(localStack) - 1
					frame := localStack[lastIdx]
					localStack = localStack[:lastIdx]

					if len(frame.Stack) == 0 {
						key := treeKey(frame.Tree)
						if _, loaded := seen.LoadOrStore(key, true); !loaded {
							select {
							case localChan <- frame.Tree:
								resultChan <- frame.Tree
							default:
								resultsMu.Lock()
								if len(results) < maxCount {
									results = append(results, frame.Tree)
									resultChan <- frame.Tree
								}
								resultsMu.Unlock()
							}
						}
						continue
					}

					lastIdx = len(frame.Stack) - 1
					curr := frame.Stack[lastIdx]
					rest := frame.Stack[:lastIdx]
					atomic.AddInt64(&nodesVisited, 1)

					if curr.Recipe == nil || len(curr.Ingredients) != 2 {
						localStack = append(localStack, Frame{Tree: frame.Tree, Stack: rest})
						continue
					}

					leftNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[0].Element)
					rightNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[1].Element)

					if (leftNode == nil || leftNode.Element.Tier == 0) &&
						(rightNode == nil || rightNode.Element.Tier == 0) {
						localStack = append(localStack, Frame{Tree: frame.Tree, Stack: rest})
						continue
					}

					leftTrees := expandIngredient(leftNode)
					rightTrees := expandIngredient(rightNode)

					newFrames := make([]Frame, 0, len(leftTrees)*len(rightTrees))

					for _, l := range leftTrees {
						for _, r := range rightTrees {
							clone := cloneTree(frame.Tree)
							ptr := findNodeByElement(clone, curr.Element)
							if ptr == nil || len(ptr.Ingredients) != 2 {
								continue
							}
							ptr.Ingredients[0] = l
							ptr.Ingredients[1] = r

							newStack := make([]*TreeNode, len(rest)+2)
							copy(newStack, append([]*TreeNode{l, r}, rest...))

							newFrames = append(newFrames, Frame{
								Tree:  clone,
								Stack: newStack,
							})
						}
					}

					for i := len(newFrames) - 1; i >= 0; i-- {
						localStack = append(localStack, newFrames[i])
					}
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		for tree := range localChan {
			resultsMu.Lock()
			if len(results) < maxCount {
				results = append(results, tree)
			}
			resultsMu.Unlock()
		}
		close(done)
	}()

	go func() {
		for _, rel := range target.Parents {
			workChan <- *rel
		}
		close(workChan)
	}()

	wg.Wait()
	close(localChan)
	<-done

	sort.Slice(results, func(i, j int) bool {
		return treeComplexity(results[i]) < treeComplexity(results[j])
	})

	return results, int(nodesVisited)
}

func treeComplexity(node *TreeNode) int {
	if node == nil {
		return 0
	}

	count := 1
	for _, child := range node.Ingredients {
		count += treeComplexity(child)
	}
	return count
}

var expandCache sync.Map

func expandIngredient(node *elementsModel.ElementNode) []*TreeNode {
	if node == nil {
		return nil
	}

	if node.Element.Tier == 0 {
		return []*TreeNode{{Element: node.Element.Name}}
	}

	cacheKey := fmt.Sprintf("%s-%s", node.Element.Name, strconv.Itoa(node.Element.Tier))
	if cached, found := expandCache.Load(cacheKey); found {
		return cached.([]*TreeNode)
	}

	out := make([]*TreeNode, 0, len(node.Parents))
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

	expandCache.Store(cacheKey, out)

	return out
}

func cloneTree(n *TreeNode) *TreeNode {
	if n == nil {
		return nil
	}

	copy := &TreeNode{
		Element:     n.Element,
		Recipe:      make([]string, len(n.Recipe)),
		Ingredients: make([]*TreeNode, len(n.Ingredients)),
	}

	if len(n.Recipe) > 0 {
		copy.Recipe[0] = n.Recipe[0]
		if len(n.Recipe) > 1 {
			copy.Recipe[1] = n.Recipe[1]
		}
	}

	for i, c := range n.Ingredients {
		copy.Ingredients[i] = cloneTree(c)
	}
	return copy
}

func findNodeByElement(n *TreeNode, target string) *TreeNode {
	if n == nil {
		return nil
	}

	if n.Element == target {
		return n
	}

	stack := make([]*TreeNode, 0, 32)
	stack = append(stack, n.Ingredients...)

	for len(stack) > 0 {
		lastIdx := len(stack) - 1
		cur := stack[lastIdx]
		stack = stack[:lastIdx]

		if cur == nil {
			continue
		}

		if cur.Element == target {
			return cur
		}

		for i := len(cur.Ingredients) - 1; i >= 0; i-- {
			stack = append(stack, cur.Ingredients[i])
		}
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

	var sb strings.Builder
	sb.Grow(len(n.Element) + len(n.Recipe[0]) + len(n.Recipe[1]) + 20)

	sb.WriteString(n.Element)
	sb.WriteString("[")
	sb.WriteString(n.Recipe[0])
	sb.WriteString("+")
	sb.WriteString(n.Recipe[1])
	sb.WriteString("](")

	left := treeKey(n.Ingredients[0])	
	sb.WriteString(left)
	sb.WriteString(",")

	right := treeKey(n.Ingredients[1])
	sb.WriteString(right)
	sb.WriteString(")")

	return sb.String()
}

func streamBFS(target *elementsModel.ElementNode, maxCount int, resultChan chan<- *TreeNode) ([]*TreeNode, int) {
	type Frame struct {
		Tree  *TreeNode
		Queue []*TreeNode
	}

	var (
		results      = make([]*TreeNode, 0, maxCount)
		resultsMu    sync.Mutex
		nodesVisited int64
		wg           sync.WaitGroup
		seen         sync.Map
		localChan    = make(chan *TreeNode, maxCount*2) 
	)

	maxWorkers := runtime.GOMAXPROCS(0)
	workChan := make(chan elementsModel.ElementRelation, maxWorkers*2)

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			localQueue := make([]Frame, 0, 100)

			for rel := range workChan {
				if len(rel.Recipe.Ingredients) != 2 {
					continue
				}

				tree := &TreeNode{
					Element: target.Element.Name,
					Recipe:  rel.Recipe.Ingredients,
					Ingredients: []*TreeNode{
						{Element: rel.Recipe.Ingredients[0]},
						{Element: rel.Recipe.Ingredients[1]},
					},
				}

				localQueue = localQueue[:0]
				localQueue = append(localQueue, Frame{Tree: tree, Queue: []*TreeNode{tree}})

				for len(localQueue) > 0 {
					resultsMu.Lock()
					full := len(results) >= maxCount
					resultsMu.Unlock()

					if full {
						break
					}

					frame := localQueue[0]
					localQueue = localQueue[1:]

					if len(frame.Queue) == 0 {
						key := treeKey(frame.Tree)
						if _, loaded := seen.LoadOrStore(key, true); !loaded {
							select {
							case localChan <- frame.Tree:
								resultChan <- frame.Tree
							default:
								resultsMu.Lock()
								if len(results) < maxCount {
									results = append(results, frame.Tree)
									resultChan <- frame.Tree
								}
								resultsMu.Unlock()
							}
						}
						continue
					}

					curr := frame.Queue[0]
					rest := frame.Queue[1:]
					atomic.AddInt64(&nodesVisited, 1)

					if curr.Recipe == nil || len(curr.Ingredients) != 2 {
						localQueue = append(localQueue, Frame{Tree: frame.Tree, Queue: rest})
						continue
					}

					leftNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[0].Element)
					rightNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[1].Element)

					if (leftNode == nil || leftNode.Element.Tier == 0) &&
						(rightNode == nil || rightNode.Element.Tier == 0) {
						localQueue = append(localQueue, Frame{Tree: frame.Tree, Queue: rest})
						continue
					}

					leftTrees := expandIngredient(leftNode)
					rightTrees := expandIngredient(rightNode)

					newFrames := make([]Frame, 0, len(leftTrees)*len(rightTrees))

					for _, l := range leftTrees {
						for _, r := range rightTrees {
							clone := cloneTree(frame.Tree)
							ptr := findNodeByElement(clone, curr.Element)
							if ptr == nil || len(ptr.Ingredients) != 2 {
								continue
							}
							ptr.Ingredients[0] = l
							ptr.Ingredients[1] = r

							newQueue := make([]*TreeNode, 0, len(rest)+2)
							newQueue = append(newQueue, rest...)
							newQueue = append(newQueue, l, r)

							newFrames = append(newFrames, Frame{
								Tree:  clone,
								Queue: newQueue,
							})
							resultChan <- clone
						}
					}

					localQueue = append(localQueue, newFrames...)
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		for tree := range localChan {
			resultsMu.Lock()
			if len(results) < maxCount {
				results = append(results, tree)
			}
			resultsMu.Unlock()
		}
		close(done)
	}()

	go func() {
		for _, rel := range target.Parents {
			workChan <- *rel
		}
		close(workChan)
	}()

	wg.Wait()
	close(localChan)
	<-done

	return results, int(nodesVisited)
}

func mergeTrees(trees []*TreeNode) *TreeNode {
	if len(trees) == 0 {
		return nil
	}

	ingredients := make([]*TreeNode, len(trees))
	copy(ingredients, trees)

	return &TreeNode{
		Element:     "Root",
		Ingredients: ingredients,
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
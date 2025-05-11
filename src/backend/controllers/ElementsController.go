package elementsController

import (
	elementsModel "backend/models"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"strconv"
	"sync/atomic"
	"time"
)

type ElementController struct {
	// Add a cache for expensive lookups
	nodeCache sync.Map
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

func (ec *ElementController) getElementNodeCached(name string) (*elementsModel.ElementNode, error) {
	// Use cached node if available
	if node, ok := ec.nodeCache.Load(name); ok {
		return node.(*elementsModel.ElementNode), nil
	}

	// Otherwise get from model and cache it
	node, err := elementsModel.GetInstance().GetElementNode(name)
	if err != nil {
		return nil, err
	}

	ec.nodeCache.Store(name, node)
	return node, nil
}

func findRecipesDFS(target *elementsModel.ElementNode, maxCount int) ([]*TreeNode, int) {
	type Frame struct {
		Tree  *TreeNode
		Stack []*TreeNode
	}

	var (
		resultsMu    sync.Mutex
		seen         sync.Map
		results      = make([]*TreeNode, 0, maxCount) // Pre-allocate with capacity
		nodesVisited int64
		wg           sync.WaitGroup
		resultChan   = make(chan *TreeNode, maxCount*2) // Larger buffer to reduce blocking
	)

	// Use a buffered channel for work distribution
	maxWorkers := runtime.GOMAXPROCS(0) * 2
	workChan := make(chan elementsModel.ElementRelation, maxWorkers*4)

	// Worker goroutines pool
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Reusable working memory
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

				// Clear and reuse localStack
				localStack = localStack[:0]
				localStack = append(localStack, Frame{Tree: tree, Stack: []*TreeNode{tree}})

				for len(localStack) > 0 {
					resultsMu.Lock()
					full := len(results) >= maxCount
					resultsMu.Unlock()

					if full {
						break
					}

					// Pop from stack
					lastIdx := len(localStack) - 1
					frame := localStack[lastIdx]
					localStack = localStack[:lastIdx]

					if len(frame.Stack) == 0 {
						key := treeKey(frame.Tree)
						if _, loaded := seen.LoadOrStore(key, true); !loaded {
							select {
							case resultChan <- frame.Tree:
							default:
								// If channel is full, try direct insertion
								resultsMu.Lock()
								if len(results) < maxCount {
									results = append(results, frame.Tree)
								}
								resultsMu.Unlock()
							}
						}
						continue
					}

					// Pop from frame stack
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

					// Fast path for base elements
					if (leftNode == nil || leftNode.Element.Tier == 0) &&
						(rightNode == nil || rightNode.Element.Tier == 0) {
						localStack = append(localStack, Frame{Tree: frame.Tree, Stack: rest})
						continue
					}

					leftTrees := expandIngredient(leftNode)
					rightTrees := expandIngredient(rightNode)

					// Pre-allocate estimated capacity
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

							// Create new stack with calculated capacity
							newStack := make([]*TreeNode, len(rest)+2)
							copy(newStack, append([]*TreeNode{l, r}, rest...))

							newFrames = append(newFrames, Frame{
								Tree:  clone,
								Stack: newStack,
							})
						}
					}

					// Add frames in reverse order for depth-first search
					for i := len(newFrames) - 1; i >= 0; i-- {
						localStack = append(localStack, newFrames[i])
					}
				}
			}
		}()
	}

	// Collector goroutine
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

	// Distribute initial work
	go func() {
		for _, rel := range target.Parents {
			workChan <- *rel
		}
		close(workChan)
	}()

	wg.Wait()
	close(resultChan)
	<-done

	// Sort results by complexity (optional - helps with deterministic output)
	sort.Slice(results, func(i, j int) bool {
		return treeComplexity(results[i]) < treeComplexity(results[j])
	})

	return results, int(nodesVisited)
}

// Calculate tree complexity for sorting
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

// Optimized expandIngredient with context-aware result caching
var expandCache sync.Map

func expandIngredient(node *elementsModel.ElementNode) []*TreeNode {
	if node == nil {
		return nil
	}

	// Check if this node has been expanded before for a particular parent
	if node.Element.Tier == 0 {
		// For base elements, always return them (no expansion needed)
		return []*TreeNode{{Element: node.Element.Name}}
	}

	// Use the parent's name and the element's name as a unique key for caching
	cacheKey := fmt.Sprintf("%s-%s", node.Element.Name, node.Element.Tier)
	if cached, found := expandCache.Load(cacheKey); found {
		// Return the cached result if it's already expanded for this context
		return cached.([]*TreeNode)
	}

	// If not cached, expand the node and store it
	out := make([]*TreeNode, 0, len(node.Parents))
	for _, rel := range node.Parents {
		if len(rel.Recipe.Ingredients) != 2 {
			continue
		}

		// Expand node's ingredients and store them in the result
		out = append(out, &TreeNode{
			Element: node.Element.Name,
			Recipe:  rel.Recipe.Ingredients,
			Ingredients: []*TreeNode{
				{Element: rel.Recipe.Ingredients[0]},
				{Element: rel.Recipe.Ingredients[1]},
			},
		})
	}

	// Store the result in the cache with the context (i.e., parent-child relationship)
	expandCache.Store(cacheKey, out)

	return out
}

// Optimized cloneTree with preallocated slice
func cloneTree(n *TreeNode) *TreeNode {
	if n == nil {
		return nil
	}

	copy := &TreeNode{
		Element:     n.Element,
		Recipe:      make([]string, len(n.Recipe)),
		Ingredients: make([]*TreeNode, len(n.Ingredients)),
	}

	// Fast copy for small slices
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

// Optimized findNodeByElement with early stopping
func findNodeByElement(n *TreeNode, target string) *TreeNode {
	if n == nil {
		return nil
	}

	if n.Element == target {
		return n
	}

	// Use a pre-allocated stack with reasonable capacity
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

		// Append in reverse for depth-first behavior
		for i := len(cur.Ingredients) - 1; i >= 0; i-- {
			stack = append(stack, cur.Ingredients[i])
		}
	}
	return nil
}

// Optimized treeKey with string builder
func treeKey(n *TreeNode) string {
	if n == nil {
		return ""
	}

	if n.Recipe == nil || len(n.Ingredients) != 2 {
		return n.Element
	}

	// Pre-calculate sizes to avoid reallocations
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

func findRecipesBFS(target *elementsModel.ElementNode, maxCount int) ([]*TreeNode, int) {
	type Frame struct {
		Tree  *TreeNode
		Queue []*TreeNode
	}

	var (
		results      = make([]*TreeNode, 0, maxCount) // Pre-allocate with capacity
		resultsMu    sync.Mutex
		nodesVisited int64
		wg           sync.WaitGroup
		seen         sync.Map
		resultChan   = make(chan *TreeNode, maxCount*2) // Larger buffer
	)

	// Use a worker pool pattern
	maxWorkers := runtime.GOMAXPROCS(0)
	workChan := make(chan elementsModel.ElementRelation, maxWorkers*2)

	// Start worker pool
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Local queue for each worker
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

				// Reset local queue
				localQueue = localQueue[:0]
				localQueue = append(localQueue, Frame{Tree: tree, Queue: []*TreeNode{tree}})

				for len(localQueue) > 0 {
					resultsMu.Lock()
					full := len(results) >= maxCount
					resultsMu.Unlock()

					if full {
						break
					}

					// Process first item in queue (BFS)
					frame := localQueue[0]
					localQueue = localQueue[1:]

					if len(frame.Queue) == 0 {
						key := treeKey(frame.Tree)
						if _, loaded := seen.LoadOrStore(key, true); !loaded {
							select {
							case resultChan <- frame.Tree:
							default:
								// Direct insertion if channel is full
								resultsMu.Lock()
								if len(results) < maxCount {
									results = append(results, frame.Tree)
								}
								resultsMu.Unlock()
							}
						}
						continue
					}

					// Extract first item from queue
					curr := frame.Queue[0]
					rest := frame.Queue[1:]
					atomic.AddInt64(&nodesVisited, 1)

					if curr.Recipe == nil || len(curr.Ingredients) != 2 {
						localQueue = append(localQueue, Frame{Tree: frame.Tree, Queue: rest})
						continue
					}

					leftNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[0].Element)
					rightNode, _ := elementsModel.GetInstance().GetElementNode(curr.Ingredients[1].Element)

					// Fast path for base elements
					if (leftNode == nil || leftNode.Element.Tier == 0) &&
						(rightNode == nil || rightNode.Element.Tier == 0) {
						localQueue = append(localQueue, Frame{Tree: frame.Tree, Queue: rest})
						continue
					}

					leftTrees := expandIngredient(leftNode)
					rightTrees := expandIngredient(rightNode)

					// Pre-calculate capacity
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

							// Create new queue with calculated capacity
							newQueue := make([]*TreeNode, 0, len(rest)+2)
							newQueue = append(newQueue, rest...)
							newQueue = append(newQueue, l, r)

							newFrames = append(newFrames, Frame{
								Tree:  clone,
								Queue: newQueue,
							})
						}
					}

					// Append all new frames to queue
					localQueue = append(localQueue, newFrames...)
				}
			}
		}()
	}

	// Start collector goroutine
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

	// Distribute initial work
	go func() {
		for _, rel := range target.Parents {
			workChan <- *rel
		}
		close(workChan)
	}()

	wg.Wait()
	close(resultChan)
	<-done

	return results, int(nodesVisited)
}

// Pool for reusing TreeNode slices
var treeNodePool = sync.Pool{
	New: func() interface{} {
		return make([]*TreeNode, 0, 16)
	},
}

func mergeTrees(trees []*TreeNode) *TreeNode {
	if len(trees) == 0 {
		return nil
	}

	// Take a copy to avoid modifying original slice
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

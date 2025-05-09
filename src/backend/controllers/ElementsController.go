package elementsController

import (
	elementsModel "backend/models"
	"fmt"
	"sync"
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
    var result *TreeNode
    var nodesVisited int

    if useBFS {
        result, nodesVisited = FindNRecipesForElementBFS(node, n)
    } else {
        result, nodesVisited = FindNRecipesForElementDFS(node, n)
    }

    duration := time.Since(start)
    return result, nodesVisited, duration, nil
}

func FindNRecipesForElementDFS(target *elementsModel.ElementNode, maxCount int) (*TreeNode, int) {
    var results []*TreeNode
    seen := sync.Map{}
    var mu sync.Mutex
    var wg sync.WaitGroup
    ch := make(chan *TreeNode, maxCount)

    rootName := target.Element.Name
    nodesVisited := 0
    var nodesVisitedMu sync.Mutex

    for _, rel := range target.Parents {
        if len(rel.Recipe.Ingredients) != 2 {
            continue
        }
        tree := &TreeNode{
            Element:     rootName,
            Recipe:      rel.Recipe.Ingredients,
            Ingredients: make([]*TreeNode, 2),
        }
        wg.Add(1)
        go func(tree *TreeNode, rel *elementsModel.ElementRelation) {
            defer wg.Done()
            expandTreeDFS(tree, rootName, &seen, ch, &nodesVisited, &nodesVisitedMu)
        }(tree, rel)
    }

    go func() {
        wg.Wait()
        close(ch)
    }()

    for tree := range ch {
        mu.Lock()
        results = append(results, tree)
        if len(results) >= maxCount {
            mu.Unlock()
            break
        }
        mu.Unlock()
    }

    return MergeTrees(results), nodesVisited
}

func expandTreeDFS(tree *TreeNode, rootName string, seen *sync.Map, ch chan *TreeNode, nodesVisited *int, nodesVisitedMu *sync.Mutex) {
    stack := []*TreeNode{tree}
    complete := true

    for len(stack) > 0 {
        curr := stack[len(stack)-1]
        stack = stack[:len(stack)-1]

        nodesVisitedMu.Lock()
        *nodesVisited++
        nodesVisitedMu.Unlock()

        if curr.Recipe == nil || len(curr.Ingredients) != 2 {
            continue
        }

        for i := 0; i < 2; i++ {
            child := curr.Ingredients[i]
            childNode, err := elementsModel.GetInstance().GetElementNode(child.Element)
            if err != nil || childNode == nil {
                complete = false
                continue
            }

            if childNode.Element.Tier == 0 {
                curr.Ingredients[i] = &TreeNode{
                    Element:     child.Element,
                    Recipe:      nil,
                    Ingredients: []*TreeNode{},
                }
                continue
            }

            found := false
            for _, childRel := range childNode.Parents {
                if len(childRel.Recipe.Ingredients) != 2 {
                    continue
                }
                subTree := &TreeNode{
                    Element: child.Element,
                    Recipe:  childRel.Recipe.Ingredients,
                    Ingredients: []*TreeNode{
                        {Element: childRel.Recipe.Ingredients[0]},
                        {Element: childRel.Recipe.Ingredients[1]},
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
        if _, loaded := seen.LoadOrStore(key, true); !loaded {
            ch <- tree
        }
    }
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
			continue 
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

func FindNRecipesForElementBFS(target *elementsModel.ElementNode, maxCount int) (*TreeNode, int) {
    rootName := target.Element.Name
    seen := sync.Map{}
    var results []*TreeNode
    var mu sync.Mutex
    var wg sync.WaitGroup
    ch := make(chan *TreeNode, maxCount)

    nodesVisited := 0
    var nodesVisitedMu sync.Mutex

    for _, relation := range target.Parents {
        if len(relation.Recipe.Ingredients) != 2 {
            continue
        }

        wg.Add(1)
        go func(rel *elementsModel.ElementRelation) {
            defer wg.Done()

            tree := &TreeNode{
                Element: rootName,
                Recipe:  rel.Recipe.Ingredients,
                Ingredients: []*TreeNode{
                    {Element: rel.Recipe.Ingredients[0]},
                    {Element: rel.Recipe.Ingredients[1]},
                },
            }

            if expandTreeBFS(tree, rootName, &nodesVisited, &nodesVisitedMu) {
                key := TreeKey(tree)
                if _, loaded := seen.LoadOrStore(key, true); !loaded {
                    ch <- tree
                }
            }
        }(relation)
    }

    go func() {
        wg.Wait()
        close(ch)
    }()

    for tree := range ch {
        mu.Lock()
        results = append(results, tree)
        if len(results) >= maxCount {
            mu.Unlock()
            break
        }
        mu.Unlock()
    }

    return MergeTrees(results), nodesVisited
}

func expandTreeBFS(tree *TreeNode, rootName string, nodesVisited *int, nodesVisitedMu *sync.Mutex) bool {
    stack := []*TreeNode{tree}
    complete := true

    for len(stack) > 0 {
        curr := stack[0]
        stack = stack[1:]

        nodesVisitedMu.Lock()
        *nodesVisited++
        nodesVisitedMu.Unlock()

        if curr.Recipe == nil || len(curr.Ingredients) != 2 {
            continue
        }

        for i := 0; i < 2; i++ {
            child := curr.Ingredients[i]
            childNode, err := elementsModel.GetInstance().GetElementNode(child.Element)
            if err != nil || childNode == nil {
                complete = false
                continue
            }

            if childNode.Element.Tier == 0 {
                curr.Ingredients[i] = &TreeNode{
                    Element:     child.Element,
                    Recipe:      nil,
                    Ingredients: []*TreeNode{},
                }
                continue
            }

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

    return complete && tree.Element == rootName && isTreeComplete(tree)
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

func MergeTrees(trees []*TreeNode) *TreeNode {
    if len(trees) == 0 {
        return nil
    }

    root := &TreeNode{
        Element:     "Root",
        Ingredients: []*TreeNode{},
        Recipe:      nil,
    }

    root.Ingredients = append(root.Ingredients, trees...)

    return root
}
package elementsController

import (
	elementsModel "backend/models"
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
	err := elementsModel.GetInstance().Initialize(filePath)
	if err != nil {
		return nil, err
	}
	return &ElementController{}, nil
}

func (ec *ElementController) FindNRecipes(targetName string, n int, useBFS bool) (*TreeNode, error) {
	node, err := elementsModel.GetInstance().GetElementNode(targetName)
	if err != nil {
		return nil, err
	}
	if useBFS {
		return FindNRecipesForElementBFS(node, n), nil
	} else {
		return FindNRecipesForElementDFS(node, n), nil
	}
}

func FindNRecipesForElementDFS(target *elementsModel.ElementNode, maxCount int) *TreeNode {
    type Frame struct {
        Tree     *TreeNode
        Node     *elementsModel.ElementNode
        Relation *elementsModel.ElementRelation
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

    // Merge all trees into a single root node
    return MergeTrees(results)
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

func FindNRecipesForElementBFS(target *elementsModel.ElementNode, maxCount int) *TreeNode {
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
                childNode, err := elementsModel.GetInstance().GetElementNode(child.Element)
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

    // Merge all trees into a single root node
    return MergeTrees(results)
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

// func PrintRecipeTree(tree *TreeNode) {
// 	if tree == nil {
// 		return
// 	}

// 	type StackFrame struct {
// 		Node   *TreeNode
// 		Prefix string // accumulated prefix (e.g. "│   ")
// 		IsLast bool   // whether this node is the last child
// 	}

// 	stack := []StackFrame{{Node: tree, Prefix: "", IsLast: true}}

// 	for len(stack) > 0 {
// 		// Pop from the back for correct DFS order
// 		frame := stack[len(stack)-1]
// 		stack = stack[:len(stack)-1]

// 		node := frame.Node
// 		connector := "├── "
// 		if frame.IsLast {
// 			connector = "└── "
// 		}
// 		fmt.Printf("%s%s", frame.Prefix, connector)
// 		if node.Recipe != nil {
// 			fmt.Printf("%s = %s + %s\n", node.Element, node.Recipe[0], node.Recipe[1])
// 		} else {
// 			fmt.Printf("%s\n", node.Element)
// 		}

// 		// Prepare next children in reverse order (so first shows on top)
// 		if len(node.Ingredients) > 0 {
// 			newPrefix := frame.Prefix
// 			if frame.IsLast {
// 				newPrefix += "    "
// 			} else {
// 				newPrefix += "│   "
// 			}
// 			for i := len(node.Ingredients) - 1; i >= 0; i-- {
// 				child := node.Ingredients[i]
// 				isLast := i == len(node.Ingredients)-1
// 				stack = append(stack, StackFrame{
// 					Node:   child,
// 					Prefix: newPrefix,
// 					IsLast: isLast,
// 				})
// 			}
// 		}
// 	}
// }

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
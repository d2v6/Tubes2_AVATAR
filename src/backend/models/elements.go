package elementsModel

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type ElementNode struct {
	Element  *Element
	Children []*ElementRelation
	Parents  []*ElementRelation
}

type ElementRelation struct {
	TargetNode  *ElementNode
	SourceNodes []*ElementNode
	Recipe      Recipe
}

type Recipe struct {
	Ingredients []string `json:"ingredients"`
}

type Element struct {
	Name    string   `json:"name"`
	Tier    int      `json:"tier"`
	Recipes []Recipe `json:"recipes"`
}

type ElementGraph struct {
	RootNode   *ElementNode
	AllNodes   map[string]*ElementNode
	Tier0Nodes []*ElementNode
}

type ElementsService struct {
	elements    []Element
	elementsMap map[string]*Element
	graph       *ElementGraph
	filePath    string
	initialized bool
	mutex       sync.RWMutex
}

var (
	instance *ElementsService
	once     sync.Once
)

func GetInstance() *ElementsService {
	once.Do(func() {
		instance = &ElementsService{
			elementsMap: make(map[string]*Element),
		}
	})
	return instance
}

func (es *ElementsService) Initialize(filePath string) error {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	if es.initialized {
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var elements []Element
	err = json.Unmarshal(data, &elements)
	if err != nil {
		return err
	}

	for i := range elements {
		es.elementsMap[elements[i].Name] = &elements[i]
	}

	es.elements = elements
	es.filePath = filePath

	es.buildElementGraph()

	es.initialized = true
	return nil
}

func (es *ElementsService) buildElementGraph() {
	graph := &ElementGraph{
		RootNode: &ElementNode{
			Element:  nil,
			Children: []*ElementRelation{},
			Parents:  []*ElementRelation{},
		},
		AllNodes:   make(map[string]*ElementNode),
		Tier0Nodes: []*ElementNode{},
	}

	for name, element := range es.elementsMap {
		node := &ElementNode{
			Element:  element,
			Children: []*ElementRelation{},
			Parents:  []*ElementRelation{},
		}
		graph.AllNodes[name] = node

		if element.Tier == 0 {
			graph.Tier0Nodes = append(graph.Tier0Nodes, node)
		}
	}

	for _, node := range graph.AllNodes {
		if node.Element.Recipes == nil {
			continue
		}

		for _, recipe := range node.Element.Recipes {
			relation := &ElementRelation{
				TargetNode:  node,
				SourceNodes: []*ElementNode{},
				Recipe:      recipe,
			}

			shouldAddParent := true
			for _, ingredientName := range recipe.Ingredients {
				if ingredientNode, exists := graph.AllNodes[ingredientName]; exists {
					if ingredientNode.Element.Tier >= node.Element.Tier {
						shouldAddParent = false
					} else {
						relation.SourceNodes = append(relation.SourceNodes, ingredientNode)
						ingredientNode.Children = append(ingredientNode.Children, relation)
					}
				}
			}

			if shouldAddParent {
				node.Parents = append(node.Parents, relation)
			}
		}
	}

	for _, tier0Node := range graph.Tier0Nodes {
		rootRelation := &ElementRelation{
			TargetNode:  tier0Node,
			SourceNodes: []*ElementNode{graph.RootNode},
			Recipe:      Recipe{Ingredients: []string{"BasicElement"}},
		}
		graph.RootNode.Children = append(graph.RootNode.Children, rootRelation)
	}

	es.graph = graph
}

func (es *ElementsService) GetElementGraph() *ElementGraph {
	es.mutex.RLock()
	defer es.mutex.RUnlock()
	return es.graph
}

func (es *ElementsService) GetAllElements() []Element {
	es.mutex.RLock()
	defer es.mutex.RUnlock()

	return es.elements
}

func (es *ElementsService) GetElementByName(name string) (*Element, error) {
	es.mutex.RLock()
	defer es.mutex.RUnlock()

	if element, exists := es.elementsMap[name]; exists {
		return element, nil
	}

	return nil, errors.New("element not found")
}

func (es *ElementsService) GetElementNode(name string) (*ElementNode, error) {
	es.mutex.RLock()
	defer es.mutex.RUnlock()

	if node, exists := es.graph.AllNodes[name]; exists {
		return node, nil
	}

	return nil, errors.New("element node not found")
}
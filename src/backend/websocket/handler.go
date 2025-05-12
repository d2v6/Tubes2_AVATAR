package websocket

import (
	elementsController "backend/controllers"
	elementsModel "backend/models"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type TreeMessage struct {
	Tree         *elementsController.TreeNode `json:"tree"`
	NodesVisited int                          `json:"nodesVisited"`
	Duration     time.Duration                `json:"duration,omitempty"`
	Done         bool                         `json:"done"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleTreeWebSocket(controller *elementsController.ElementController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}
		defer conn.Close()

		var req struct {
			Target string `json:"target"`
			Count  int    `json:"count"`
			UseBFS bool   `json:"useBfs"`
			Delay  int    `json:"delay"`
		}

		if err := conn.ReadJSON(&req); err != nil {
			log.Println("Failed to read JSON:", err)
			return
		}

		node, err := elementsModel.GetInstance().GetElementNode(req.Target)
		if err != nil {
			log.Println("Invalid target element:", err)
			conn.WriteMessage(websocket.TextMessage, []byte("Invalid element"))
			return
		}

		treeChan := make(chan *elementsController.TreeNode, req.Count)

		var trees []*elementsController.TreeNode
		var nodesVisited int
		var duration time.Duration

		go func() {
			if req.UseBFS {
				trees, nodesVisited, duration = elementsController.StartBFS(node, req.Count, treeChan)
			} else {
				trees, nodesVisited, duration = elementsController.StartDFS(node, req.Count, treeChan)
			}
			close(treeChan)
		}()

		var delay time.Duration = time.Duration(req.Delay) * time.Millisecond
		
		for tree := range treeChan {
			time.Sleep(delay) 

			// log.Println("Debug: Printing tree structure:")
			// elementsController.PrintRecipeTree(tree, "", true)

			msg := TreeMessage{
				Tree:         tree,
				NodesVisited: nodesVisited,
				Done:         false,
			}

			if err := conn.WriteJSON(msg); err != nil {
				log.Println("Error sending intermediate result:", err)
				return
			}
		}
		
		// for _, tree := range trees {
		// 	log.Println("Tree:")
		// 	elementsController.PrintRecipeTree(tree, "", true)
		// }

		treeChanForMerge := make(chan *elementsController.TreeNode, len(trees))
		for _, tree := range trees {
			treeChanForMerge <- tree
		}
		close(treeChanForMerge)

		finalTree := elementsController.MergeTrees(treeChanForMerge)
		if finalTree == nil && len(trees) > 0 {
			finalTree = &elementsController.TreeNode{
				Element:     "Root",
				Ingredients: trees,
			}
		}

		finalMsg := TreeMessage{
			Tree:         finalTree,
			NodesVisited: nodesVisited,
			Duration:     duration,
			Done:         true,
		}

		// if finalTree != nil {
		// 	log.Println("Debug: Printing final merged tree structure:")
		// 	elementsController.PrintRecipeTree(finalTree, "", true)
		// }

		if err := conn.WriteJSON(finalMsg); err != nil {
			log.Println("Error sending final result:", err)
		}
	}
}
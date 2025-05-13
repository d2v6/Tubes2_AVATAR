package websocket

import (
	elementsController "backend/controllers"
	"log"
	"net/http"
	"sync"
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

        treeChan := make(chan *elementsController.TreeNode, req.Count)

        var tree *elementsController.TreeNode
        var nodesVisited int
        var duration time.Duration
		var wg sync.WaitGroup
		wg.Add(1)

        go func() {
			defer wg.Done()
            if req.UseBFS {
                tree, duration = elementsController.StartBFS(req.Target, req.Count, treeChan)
            } else {
                tree, duration = elementsController.StartDFS(req.Target, req.Count, treeChan)
            }
            close(treeChan)
        }()

        if req.Delay > 0 {
            var delay time.Duration = time.Duration(req.Delay) * time.Millisecond

            for intermediateTree := range treeChan {
                time.Sleep(delay)

                msg := TreeMessage{
                    Tree:         intermediateTree,
                    NodesVisited: nodesVisited,
                    Done:         false,
                }

                if err := conn.WriteJSON(msg); err != nil {
                    log.Println("Error sending intermediate result:", err)
                    return
                }
            }
        }

		wg.Wait()
        finalMsg := TreeMessage{
            Tree:         tree,
            NodesVisited: nodesVisited,
            Duration:     duration,
            Done:         true,
        }

        if err := conn.WriteJSON(finalMsg); err != nil {
            log.Println("Error sending final result:", err)
        }
    }
}
package websocket

import (
	elementsController "backend/controllers"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type TreeMessage struct {
    Tree             *elementsController.TreeNode `json:"tree"`
    NodesVisited     int64                        `json:"nodesVisited"`
    SearchDuration   time.Duration                `json:"searchDuration,omitempty"`
    ProgramDuration  time.Duration                `json:"programDuration,omitempty"`
    Done             bool                         `json:"done"`
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
            Target         string `json:"target"`
            Count          int    `json:"count"`
            UseBFS         bool   `json:"useBfs"`
            Delay          int    `json:"delay"`
            UseMultiThread bool   `json:"useMultiThread"`
        }

        if err := conn.ReadJSON(&req); err != nil {
            log.Println("Failed to read JSON:", err)
            return
        }

        treeChan := make(chan *elementsController.TreeNode, req.Count)

        var tree *elementsController.TreeNode
        var nodesVisited int64
        var searchDuration time.Duration
		startProgram := time.Now()

        go func() {
            if req.UseBFS {
                if req.UseMultiThread {
                    tree, nodesVisited, searchDuration = elementsController.StartBFSMulti(req.Target, req.Count, treeChan)
                } else {
                    tree, nodesVisited, searchDuration = elementsController.StartBFS(req.Target, req.Count, treeChan)
                }
            } else {
                if req.UseMultiThread {
                    tree, nodesVisited, searchDuration = elementsController.StartDFSMulti(req.Target, req.Count, treeChan)
                } else {
                    tree, nodesVisited, searchDuration = elementsController.StartDFS(req.Target, req.Count, treeChan)
                }
            }
            close(treeChan)
        }()

		var delay time.Duration = time.Duration(req.Delay) * time.Millisecond

		for intermediateTree := range treeChan {
			time.Sleep(delay)

            msg := TreeMessage{
                Tree:            intermediateTree,
                NodesVisited:    nodesVisited,
                SearchDuration:  searchDuration,
                ProgramDuration: time.Since(startProgram),
                Done:            false,
            }

			if err := conn.WriteJSON(msg); err != nil {
				log.Println("Error sending intermediate result:", err)
				return
			}
		}

        finalMsg := TreeMessage{
            Tree:            tree,
            NodesVisited:    nodesVisited,
            SearchDuration:  searchDuration,
            ProgramDuration: time.Since(startProgram),
            Done:            true,
        }

        if err := conn.WriteJSON(finalMsg); err != nil {
            log.Println("Error sending final result:", err)
        }
    }
}
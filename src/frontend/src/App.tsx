import { useEffect, useState, useCallback, useRef } from "react";
import ReactFlow, { type Node, type Edge, Background, Controls, MiniMap, useNodesState, useEdgesState, Position } from "reactflow";
import "reactflow/dist/style.css";
import "./App.css";

type TreeNode = {
  Name: string;
  Recipe?: TreeNode[];
};

type TreeWebSocketMessage = {
  tree: TreeNode;
  nodesVisited: number;
  searchDuration?: string;
  programDuration?: string;
  done: boolean;
};

type Element = {
  name: string;
  tier: number;
  recipes: { ingredients: string[] }[];
};

function App() {
  const [recipeTree, setRecipeTree] = useState<TreeNode | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [target, setTarget] = useState("Metal");
  const [method, setMethod] = useState("bfs");
  const [count, setCount] = useState(2);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [nodesVisited, setNodesVisited] = useState<number | null>(null);
  const [timeTaken, setTimeTaken] = useState<string | null>(null);
  const [searchTime, setSearchTime] = useState<string | null>(null);
  const [wsDelay, setWsDelay] = useState(100);
  const [useMultiThread, setUseMultiThread] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);

  const wsUrl = window.location.hostname === "localhost" ? "ws://localhost:4003" : `ws://${window.location.host}`;

  const convertToReactFlowFormat = useCallback((treeNode: TreeNode, parentId?: string, depth = 0, xOffset = 0): { nodes: Node[]; edges: Edge[]; width: number } => {
    const nodeId = `${treeNode.Name}-${depth}-${xOffset}`;
    const nodeSpacingX = 200;
    const nodeSpacingY = 100;

    const nodes: Node[] = [];
    const edges: Edge[] = [];

    const childResults: { nodes: Node[]; edges: Edge[]; width: number }[] = [];

    let totalChildWidth = 0;
    if (treeNode.Recipe) {
      for (const child of treeNode.Recipe) {
        const result = convertToReactFlowFormat(child, nodeId, depth + 1, xOffset + totalChildWidth);
        childResults.push(result);
        totalChildWidth += result.width;
      }
    }

    if (totalChildWidth === 0) {
      totalChildWidth = 1;
    }

    const centerX = xOffset + totalChildWidth / 2;

    nodes.push({
      id: nodeId,
      data: { label: treeNode.Name },
      position: {
        x: centerX * nodeSpacingX,
        y: depth * nodeSpacingY,
      },
      type: "default",
      targetPosition: Position.Top,
      sourcePosition: Position.Bottom,
    });

    if (parentId) {
      edges.push({
        id: `${parentId}-${nodeId}`,
        source: parentId,
        target: nodeId,
      });
    }

    for (const result of childResults) {
      nodes.push(...result.nodes);
      edges.push(...result.edges);
    }

    return { nodes, edges, width: totalChildWidth };
  }, []);

  const fetchRecipes = async () => {
    setLoading(true);
    setError(null);
    setRecipeTree(null);
    setNodesVisited(null);
    setTimeTaken(null);
    setSearchTime(null);
    setNodes([]);
    setEdges([]);

    try {
      // const response = await fetch(`https://localhost:4003/api/elements/${target}`); //for local
      const response = await fetch(`/api/elements/${target}`);
      if (!response.ok) {
        throw new Error("Failed to fetch target element");
      }
      const targetElement: Element = await response.json();

      if (method === "bfs" && targetElement.tier > 5 && window.location.hostname !== "localhost") {
        setError("Error: BFS is not allowed for elements with a tier above 5.");
        setLoading(false);
        return;
      }
    } catch (err) {
      console.error("Error fetching target element:", err);
      setError("Failed to find element.");
      setLoading(false);
      return;
    }

    if (wsRef.current) {
      wsRef.current.close();
    }

    const ws = new WebSocket(`${wsUrl.replace("http", "ws")}/ws/tree`);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log("WebSocket connection established");
      ws.send(
        JSON.stringify({
          target: target,
          count: count,
          useBfs: method === "bfs",
          delay: wsDelay,
          useMultiThread: useMultiThread,
        })
      );
    };

    ws.onmessage = (event) => {
      setTimeout(() => {
        try {
          const data = JSON.parse(event.data) as TreeWebSocketMessage;
          console.log("Data:", data);

          if (data.tree) {
            setRecipeTree(data.tree);
          }

          if (data.nodesVisited) {
            setNodesVisited(data.nodesVisited);
          }

          if (data.searchDuration) {
            setSearchTime(`${(parseInt(data.searchDuration) / 1e6).toFixed(2)} ms`);
          }

          if (data.programDuration) {
            setTimeTaken(`${(parseInt(data.programDuration) / 1e6).toFixed(2)} ms`);
          }

          if (data.done) {
            setLoading(false);
            ws.close();
            wsRef.current = null;
          }
        } catch (err) {
          console.error("Error parsing WebSocket message:", err);
          setError("Failed to parse WebSocket response");
          setLoading(false);
        }
      }, wsDelay);
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      setError("WebSocket connection error");
      setLoading(false);
    };

    ws.onclose = () => {
      console.log("WebSocket connection closed");
    };
  };

  useEffect(() => {
    if (recipeTree) {
      const { nodes: flowNodes, edges: flowEdges } = convertToReactFlowFormat(recipeTree);
      setNodes(flowNodes);
      setEdges(flowEdges);
    }
  }, [recipeTree, convertToReactFlowFormat, setNodes, setEdges]);

  useEffect(() => {
    fetchRecipes();
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSearch = () => {
    fetchRecipes();
  };

  return (
    <div className="p-4">
      <div className="mb-4 flex flex-col space-y-4">
        <div className="flex space-x-2 items-center">
          <input type="text" value={target} onChange={(e) => setTarget(e.target.value)} placeholder="Element name" className="border p-2 rounded" />
          <select value={method} onChange={(e) => setMethod(e.target.value)} className="border p-2 rounded">
            <option value="bfs">BFS</option>
            <option value="dfs">DFS</option>
          </select>
          <input
            type="number"
            min="1"
            value={count || ""}
            onChange={(e) => setCount(e.target.value === "" ? 0 : parseInt(e.target.value))}
            placeholder="Max recipes"
            className="border p-2 rounded w-24"
          />
          <input
            type="number"
            min="0"
            value={wsDelay || ""}
            onChange={(e) => setWsDelay(Math.max(0, parseInt(e.target.value) || 0))}
            placeholder="WebSocket delay (ms)"
            className="border p-2 rounded w-36"
          />
          <button onClick={() => setUseMultiThread((prev) => !prev)} className={`p-2 rounded ${useMultiThread ? "bg-green-500 hover:bg-green-600" : "bg-gray-500 hover:bg-gray-600"} text-white`}>
            {useMultiThread ? "Disable Multithreading" : "Enable Multithreading"}
          </button>
          <button
            onClick={() => {
              if (!count) setCount(1);
              if (!wsDelay) setWsDelay(0);
              handleSearch();
            }}
            className="bg-blue-500 text-white p-2 rounded hover:bg-blue-600"
            disabled={loading}
          >
            Find Recipes
          </button>
        </div>
      </div>

      <div className="rounded-lg overflow-hidden">
        {error && <p className="p-4 text-center text-red-500">Error: {error}</p>}
        {!error && !recipeTree && <p className="p-4 text-center">No recipes found for "{target}"</p>}
        {recipeTree && (
          <div className="p-4">
            <h2 className="text-xl font-bold mb-4">Recipe Tree for {recipeTree.Recipe && recipeTree.Recipe[0]?.Name ? recipeTree.Recipe[0].Name : ""}</h2>
            <div className="mb-4">
              <p>Nodes Visited: {nodesVisited}</p>
              <p>Search Duration: {searchTime || "0ms"}</p>
              <p>Program Duration: {timeTaken || "0ms"}</p>
            </div>
            <div style={{ width: "100%", height: "600px", border: "1px solid #ddd" }}>
              <ReactFlow nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} defaultViewport={{ x: 0, y: 10, zoom: 0.7 }}>
                <Background />
                <Controls />
                <MiniMap />
              </ReactFlow>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;

import { useEffect, useState, useCallback, useRef } from "react";
import { Tree } from "react-tree-graph";
import "react-tree-graph/dist/style.css";
import "./App.css";

type RecipeInfo = {
  ingredients: string[];
};

type TreeNode = {
  element: string;
  ingredients: Record<string, TreeNode>;
  recipes: RecipeInfo[];
};

type TreeGraphNode = {
  name: string;
  children: TreeGraphNode[];
};

type TreeWebSocketMessage = {
  tree: TreeNode;
  nodesVisited: number;
  duration?: string;
  done: boolean;
};

function getTreeSize(node: TreeGraphNode): { depth: number; maxBreadth: number } {
  let maxDepth = 0;
  let maxBreadth = 0;

  function traverse(n: TreeGraphNode, depth: number) {
    if (depth > maxDepth) maxDepth = depth;
    if (n.children && n.children.length > maxBreadth) {
      maxBreadth = n.children.length;
    }
    n.children?.forEach((child) => traverse(child, depth + 1));
  }

  traverse(node, 1);
  return { depth: maxDepth, maxBreadth };
}

function App() {
  const [recipeTree, setRecipeTree] = useState<TreeNode | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [target, setTarget] = useState("Metal");
  const [method, setMethod] = useState("bfs");
  const [count, setCount] = useState(2);
  const [treeData, setTreeData] = useState<TreeGraphNode | null>(null);
  const [nodesVisited, setNodesVisited] = useState<number | null>(null);
  const [timeTaken, setTimeTaken] = useState<string | null>(null);
  const [wsDelay, setWsDelay] = useState(500);

  const wsRef = useRef<WebSocket | null>(null);

  const wsUrl = window.location.hostname === "localhost" ? "ws://localhost:4003" : `ws://${window.location.host}`;

  const fetchRecipes = () => {
    setLoading(true);
    setError(null);
    setTreeData(null);
    setRecipeTree(null);

    if (wsRef.current) {
      wsRef.current.close();
    }

    const ws = new WebSocket(`${wsUrl}/ws/tree`);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log("WebSocket connection established");
      ws.send(
        JSON.stringify({
          target: target,
          count: count,
          useBfs: method === "bfs",
          delay: wsDelay,
        })
      );
    };

    ws.onmessage = (event) => {
      setTimeout(() => {
        try {
          const data = JSON.parse(event.data) as TreeWebSocketMessage;
          console.log("Data:", data);

          if (data.tree) {
            setRecipeTree(null);
            setRecipeTree(data.tree);
          }

          if (data.nodesVisited) {
            setNodesVisited(data.nodesVisited);
          }

          if (data.duration) {
            setTimeTaken(`${(parseInt(data.duration) / 1e6).toFixed(2)} ms`);
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

  const convertToTreeGraphFormat = useCallback((node: TreeNode): TreeGraphNode => {
    const children: TreeGraphNode[] = node.ingredients ? Object.values(node.ingredients).map(convertToTreeGraphFormat) : [];

    return {
      name: node.element,
      children: children,
    };
  }, []);

  useEffect(() => {
    if (recipeTree) {
      setTreeData(null);
      const data = convertToTreeGraphFormat(recipeTree);
      setTreeData(data);
    }
  }, [recipeTree, convertToTreeGraphFormat]);

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
            max="10"
            value={count || ""}
            onChange={(e) => setCount(e.target.value === "" ? 0 : Math.min(10, Math.max(1, parseInt(e.target.value) || 1)))}
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
          <button
            onClick={() => {
              if (!count) setCount(1);
              if (!wsDelay) setWsDelay(500);
              handleSearch();
            }}
            className="bg-blue-500 text-white p-2 rounded hover:bg-blue-600"
            disabled={loading}
          >
            Find Recipes
          </button>
        </div>
      </div>

      <div className="border border-gray-300 rounded-lg overflow-hidden">
        {error && <p className="p-4 text-center text-red-500">Error: {error}</p>}
        {!error && !recipeTree && <p className="p-4 text-center">No recipes found for "{target}"</p>}
        {recipeTree && treeData && (
          <div className="p-4">
            <h2 className="text-xl font-bold mb-4">Recipe Tree for {recipeTree.element}</h2>
            <div className="mb-4">
              <p>Nodes Visited: {nodesVisited}</p>
              <p>Time Taken: {timeTaken}</p>
            </div>
            <div id="treeWrapper" className="overflow-auto border p-4" style={{ width: "100%", height: "600px" }}>
              {treeData &&
                (() => {
                  const { depth, maxBreadth } = getTreeSize(treeData);
                  const height = Math.max(300, depth * 150);
                  const width = Math.max(600, maxBreadth * 300);
                  return <Tree data={treeData} height={height} width={width} svgProps={{ className: "custom" }} />;
                })()}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;

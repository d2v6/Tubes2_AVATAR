import { useEffect, useState, useCallback } from "react";
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
  children?: TreeGraphNode[];
};

function App() {
  const [recipeTree, setRecipeTree] = useState<TreeNode | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [target, setTarget] = useState("Brick");
  const [method, setMethod] = useState("bfs");
  const [count, setCount] = useState(1);
  const [treeData, setTreeData] = useState<TreeGraphNode | null>(null);
  const [nodesVisited, setNodesVisited] = useState<number | null>(null);
  const [timeTaken, setTimeTaken] = useState<number | null>(null);

  const fetchRecipes = () => {
    setLoading(true);
    setError(null);
    setTreeData(null);
    setRecipeTree(null);

    fetch(`http://localhost:8080/api/recipes?target=${target}&method=${method}&count=${count}`)
      .then((res) => {
        if (!res.ok) {
          return res.text().then((text) => {
            throw new Error(text || `Failed to fetch: ${res.status}`);
          });
        }
        return res.json();
      })
      .then((data) => {
        setRecipeTree(data.recipes);
        setNodesVisited(data.nodesVisited);
        setTimeTaken(parseFloat(data.duration) * 1000);
        setLoading(false);
        console.log("Recipe tree data:", data);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
        console.error("Error fetching recipes:", err);
      });
  };

  const convertToTreeGraphFormat = useCallback((node: TreeNode): TreeGraphNode => {
    const children: TreeGraphNode[] = node.ingredients ? Object.values(node.ingredients).map(convertToTreeGraphFormat) : [];
    return {
      name: node.element,
      children: children.length > 0 ? children : undefined,
    };
  }, []);

  useEffect(() => {
    if (recipeTree) {
      const data = convertToTreeGraphFormat(recipeTree);
      setTreeData(data);
    }
  }, [recipeTree, convertToTreeGraphFormat]);

  useEffect(() => {
    fetchRecipes();
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
            value={count}
            onChange={(e) => setCount(Math.min(10, Math.max(1, parseInt(e.target.value) || 1)))}
            placeholder="Max recipes"
            className="border p-2 rounded w-24"
          />
          <button onClick={handleSearch} className="bg-blue-500 text-white p-2 rounded hover:bg-blue-600">
            Find Recipes
          </button>
        </div>
      </div>

      <div className="border border-gray-300 rounded-lg overflow-hidden">
        {loading && <p className="p-4 text-center">Loading...</p>}
        {error && <p className="p-4 text-center text-red-500">Error: {error}</p>}
        {!loading && !error && !recipeTree && <p className="p-4 text-center">No recipes found for "{target}"</p>}
        {!loading && recipeTree && treeData && (
          <div className="p-4">
            <h2 className="text-xl font-bold mb-4">Recipe Tree for {recipeTree.element}</h2>
            <div className="mb-4">
              <p>Nodes Visited: {nodesVisited}</p>
              <p>Time Taken: {timeTaken} ms</p>
            </div>
            <div id="treeWrapper" className="overflow-auto border p-4" style={{ width: "100%", height: "600px" }}>
              <Tree data={treeData} height={500} width={1000} animated svgProps={{ className: "custom" }} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;

import { useEffect, useState } from "react";
import React from "react";
import "./App.css";

// Component to recursively render the recipe tree
type RecipeNode = {
  element: string;
  recipe?: [string, string];
  ingredients?: RecipeNode[];
};

const RecipeTreeNode = ({ node, depth = 0 }: { node: RecipeNode; depth?: number }) => {
  if (!node) return null;
  
  const indentClass = depth > 0 ? 'tree-node-indented' : '';
  
  return (
    <div className={`tree-node ${indentClass}`} style={{ marginLeft: `${depth * 20}px` }}>
      <div className="element-info">
        <span className="element-name">{node.element}</span>
        {node.recipe && node.recipe.length === 2 && (
          <span className="recipe-formula"> = {node.recipe[0]} + {node.recipe[1]}</span>
        )}
      </div>
      
      {node.ingredients && node.ingredients.length > 0 && (
        <div className="ingredients">
          {node.ingredients.map((ingredient, idx) => (
            <RecipeTreeNode 
              key={`${ingredient.element}-${idx}`} 
              node={ingredient} 
              depth={depth + 1} 
            />
          ))}
        </div>
      )}
    </div>
  );
};

function App() {
  const [recipes, setRecipes] = useState<RecipeNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [target, setTarget] = useState("Brick");
  const [method, setMethod] = useState("bfs");
  const [count, setCount] = useState(3);
  const [activeTab, setActiveTab] = useState("single");

  const fetchSingleRecipe = () => {
    setLoading(true);
    setError(null);
    
    fetch(`http://localhost:8080/api/path?target=${target}&method=${method}`)
      .then((res) => {
        if (!res.ok) {
          return res.text().then(text => {
            throw new Error(text || `Failed to fetch: ${res.status}`);
          });
        }
        return res.json();
      })
      .then((data) => {
        setRecipes([data]);
        setLoading(false);
        console.log("Recipe data:", data);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
        console.error("Error fetching recipe:", err);
      });
  };

  const fetchMultipleRecipes = () => {
    setLoading(true);
    setError(null);
    
    fetch(`http://localhost:8080/api/recipes?target=${target}&method=${method}&count=${count}`)
      .then((res) => {
        if (!res.ok) {
          return res.text().then(text => {
            throw new Error(text || `Failed to fetch: ${res.status}`);
          });
        }
        return res.json();
      })
      .then((data) => {
        setRecipes(data);
        setLoading(false);
        console.log("Recipes data:", data);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
        console.error("Error fetching recipes:", err);
      });
  };

  useEffect(() => {
    if (activeTab === "single") {
      fetchSingleRecipe();
    } else {
      fetchMultipleRecipes();
    }
  }, [activeTab]);

  const handleSearch = () => {
    if (activeTab === "single") {
      fetchSingleRecipe();
    } else {
      fetchMultipleRecipes();
    }
  };

  return (
    <div className="app">
      <div className="header">
        <h1>Element Recipe Explorer</h1>
        
        <div className="tabs">
          <button 
            className={`tab-button ${activeTab === "single" ? "active-tab" : ""}`}
            onClick={() => setActiveTab("single")}
          >
            Single Recipe
          </button>
          <button 
            className={`tab-button ${activeTab === "multiple" ? "active-tab" : ""}`}
            onClick={() => setActiveTab("multiple")}
          >
            Multiple Recipes
          </button>
        </div>

        <div className="search-controls">
          <input
            className="input"
            type="text" 
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder="Element name"
          />
          <select 
            className="select"
            value={method} 
            onChange={(e) => setMethod(e.target.value)}
          >
            <option value="bfs">BFS</option>
            <option value="dfs">DFS</option>
          </select>
          
          {activeTab === "multiple" && (
            <input
              className="count-input"
              type="number"
              min="1"
              max="10"
              value={count}
              onChange={(e) => setCount(Math.min(10, Math.max(1, parseInt(e.target.value) || 1)))}
              placeholder="Count"
            />
          )}
          
          <button className="button" onClick={handleSearch}>
            Find {activeTab === "multiple" ? "Recipes" : "Recipe"}
          </button>
        </div>
      </div>
      
      <div>
        {loading && <p className="loading">Loading...</p>}
        {error && <p className="error">Error: {error}</p>}
        
        {!loading && !error && recipes.length === 0 && (
          <p className="no-results">No recipes found for "{target}"</p>
        )}
        
        {recipes.length > 0 && (
          <div className="recipes-container">
            <h2>
              {recipes.length > 1 ? 
                `${recipes.length} Recipes for ${recipes[0].element}` : 
                `Recipe for ${recipes[0].element}`}
            </h2>
            
            {recipes.map((recipe, index) => (
              <div key={index} className="recipe-tree">
                <h3>Recipe {index + 1}</h3>
                <div className="tree-container">
                  <RecipeTreeNode node={recipe} />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default App
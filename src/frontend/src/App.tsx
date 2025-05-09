import { useEffect, useState } from "react";
import "./App.css";

function App() {
  const [path, setPath] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch("http://localhost:8080/api/path?target=Brick")
      .then((res) => {
        if (!res.ok) throw new Error("Failed to fetch");
        return res.json();
      })
      .then((data) => {
        setPath(data);
        setLoading(false);
        console.log(data);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  return (
    <div className="App">
      <h1>Element Path to Create "Brick"</h1>
      {loading && <p>Loading...</p>}
      {error && <p style={{ color: "red" }}>Error: {error}</p>}
      {path && (
        <div>
          <p>{/* <strong>Target:</strong> {path.TargetElement} */}</p>
          <ul>
            {/* {path.Steps.map((step, index) => (
              <li key={index}>
                Step {index + 1}: Combine <b>{step.Ingredients[0]}</b> and <b>{step.Ingredients[1]}</b> to make <b>{step.Element}</b>
              </li>
            ))} */}
          </ul>
        </div>
      )}
    </div>
  );
}

export default App;

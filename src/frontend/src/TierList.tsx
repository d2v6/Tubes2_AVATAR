import { useEffect, useState } from "react";

interface TierListData {
  tiers: number[];
  elements: Record<string, string[]>;
}

function TierList() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tierData, setTierData] = useState<TierListData | null>(null);

  useEffect(() => {
    const fetchTierList = async () => {
      try {
        setLoading(true);
        const response = await fetch(`http://localhost:4003/api/tiers`); //for local
        // const response = await fetch(`/api/tiers`);

        if (!response.ok) {
          throw new Error(`Failed to fetch tier list: ${response.status}`);
        }

        const data = await response.json();
        setTierData(data);
        setLoading(false);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
        setLoading(false);
      }
    };

    fetchTierList();
  }, []);

  const getTierColor = (tierNum: number): string => {
    const colors = [
      "#44bd32",
      "#0097e6",
      "#8e44ad",
      "#e1b12c",
      "#e67e22", //
      "#d35400", //
      "#c23616", //
      "#e84118", //
      "#273c75", //
      "#574b90", //
      "#786fa6", //
      "#f368e0", //
      "#ff9ff3", //
      "#341f97", //
      "#5f27cd", //
      "#222f3e", //
    ];
    return tierNum < colors.length ? colors[tierNum] : "#000";
  };

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <h1 className="text-4xl font-extrabold mb-8 text-center">🌟 Element Tier List 🌟</h1>

      {loading && <p className="text-gray-600 text-center">Loading tier data...</p>}
      {error && <p className="text-red-500 text-center">Error: {error}</p>}

      {tierData && (
        <div className="space-y-6">
          {tierData.tiers.map((tier) => (
            <div key={tier} className="rounded-2xl shadow-lg overflow-hidden border border-gray-200">
              <div
                className="py-4 px-6 text-xl font-bold"
                style={{
                  backgroundColor: getTierColor(tier),
                  color: "#fff",
                }}
              >
                Tier {tier}
              </div>
              <div className="p-6 grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 xl:grid-cols-8 gap-3">
                {tierData.elements[tier.toString()]?.map((element) => (
                  <div key={element} className="px-3 py-1 text-sm font-medium bg-gray-100 border border-gray-300 rounded-full text-center hover:bg-gray-200 transition duration-150">
                    {element}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default TierList;

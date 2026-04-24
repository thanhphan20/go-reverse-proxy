"use client";

import { useEffect, useState } from "react";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function Home() {
  const [metrics, setMetrics] = useState({
    total_requests: 0,
    error_count: 0,
    cache_hits: 0,
    cache_misses: 0,
  });
  const [events, setEvents] = useState([]);
  const [routes, setRoutes] = useState({});
  const [newPath, setNewPath] = useState("");
  const [newTarget, setNewTarget] = useState("");
  const [testPath, setTestPath] = useState("");
  const [testResponse, setTestResponse] = useState<any>(null);
  const [isTesting, setIsTesting] = useState(false);

  const fetchData = async () => {
    try {
      const mRes = await fetch(`${API_URL}/api/metrics`);
      setMetrics(await mRes.json());
      const eRes = await fetch(`${API_URL}/api/events`);
      setEvents(await eRes.json());
      const rRes = await fetch(`${API_URL}/api/routes`);
      setRoutes(await rRes.json());
    } catch (e) {
      console.error("Failed to fetch proxy API", e);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 2000);
    return () => clearInterval(interval);
  }, []);

  const handleDeleteRoute = async (path: string) => {
    await fetch(`${API_URL}/api/routes?path=${encodeURIComponent(path)}`, {
      method: "DELETE",
    });
    fetchData();
  };

  const handleEditRoute = (path: string, target: string) => {
    setNewPath(path);
    setNewTarget(target);
  };

  const handleAddRoute = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPath || !newTarget) return;
    await fetch(`${API_URL}/api/routes`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: newPath, target: newTarget }),
    });
    setNewPath("");
    setNewTarget("");
    fetchData();
  };

  const handleTestRoute = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!testPath) return;
    setIsTesting(true);
    setTestResponse(null);
    try {
      const res = await fetch(`${API_URL}${testPath.startsWith("/") ? testPath : "/" + testPath}`);
      
      let data;
      const contentType = res.headers.get("content-type");
      if (contentType && contentType.includes("application/json")) {
        data = await res.json();
      } else {
        data = await res.text();
      }

      setTestResponse({
        status: res.status,
        data: data,
        headers: Object.fromEntries(res.headers.entries())
      });
    } catch (err: any) {
      setTestResponse({ error: err.message });
    } finally {
      setIsTesting(false);
      fetchData(); // Update metrics/events after test
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-8 font-sans">
      <div className="max-w-6xl mx-auto space-y-8">
        <h1 className="text-4xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500">
          Go Reverse Proxy Dashboard
        </h1>
        
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-gray-800 p-6 rounded-xl border border-gray-700 shadow-lg transition-transform hover:scale-105">
            <h3 className="text-gray-400 mb-2">Total Requests</h3>
            <p className="text-3xl font-bold text-blue-400">{metrics.total_requests}</p>
          </div>
          <div className="bg-gray-800 p-6 rounded-xl border border-gray-700 shadow-lg transition-transform hover:scale-105">
            <h3 className="text-gray-400 mb-2">Errors</h3>
            <p className="text-3xl font-bold text-red-400">{metrics.error_count}</p>
          </div>
          <div className="bg-gray-800 p-6 rounded-xl border border-gray-700 shadow-lg transition-transform hover:scale-105">
            <h3 className="text-gray-400 mb-2">Cache Hits</h3>
            <p className="text-3xl font-bold text-green-400">{metrics.cache_hits}</p>
          </div>
          <div className="bg-gray-800 p-6 rounded-xl border border-gray-700 shadow-lg transition-transform hover:scale-105">
            <h3 className="text-gray-400 mb-2">Cache Misses</h3>
            <p className="text-3xl font-bold text-yellow-400">{metrics.cache_misses}</p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          <div className="bg-gray-800 p-6 rounded-xl border border-gray-700 shadow-lg">
            <h2 className="text-2xl font-bold mb-4 border-b border-gray-700 pb-2">Dynamic Routes</h2>
            <div className="space-y-4 mb-6">
              {Object.entries(routes).map(([path, target]) => (
                <div key={path} className="group flex justify-between items-center bg-gray-900 p-3 rounded border border-gray-700 hover:border-blue-500/50 transition-colors">
                  <div className="flex items-center gap-3 overflow-hidden">
                    <span className="font-mono text-purple-400 font-semibold">{path}</span>
                    <span className="text-gray-600">→</span>
                    <span className="font-mono text-green-400 text-sm truncate">{target as string}</span>
                  </div>
                  <div className="flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button 
                      onClick={() => handleEditRoute(path, target as string)}
                      className="p-1.5 hover:bg-gray-700 rounded text-gray-400 hover:text-blue-400 transition-colors"
                      title="Edit"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-5M16.243 3.757a4.5 4.5 0 116.364 6.364L12 20.364l-7.682-7.682L16.243 3.757z" />
                      </svg>
                    </button>
                    <button 
                      onClick={() => handleDeleteRoute(path)}
                      className="p-1.5 hover:bg-gray-700 rounded text-gray-400 hover:text-red-400 transition-colors"
                      title="Delete"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
              ))}
              {Object.keys(routes).length === 0 && <p className="text-gray-500">No routes configured.</p>}
            </div>
            <form onSubmit={handleAddRoute} className="flex gap-2 bg-gray-900/50 p-4 rounded-xl border border-gray-700/50 backdrop-blur-sm">
              <div className="flex-1 space-y-1">
                <label className="text-[10px] font-bold text-gray-500 uppercase tracking-wider ml-1">Path</label>
                <input 
                  type="text" 
                  placeholder="/api/users" 
                  className="bg-gray-900 border border-gray-700 p-2 rounded-lg w-full focus:ring-2 focus:ring-blue-500 outline-none transition-all placeholder:text-gray-600"
                  value={newPath}
                  onChange={e => setNewPath(e.target.value)}
                />
              </div>
              <div className="flex-1 space-y-1">
                <label className="text-[10px] font-bold text-gray-500 uppercase tracking-wider ml-1">Target URL</label>
                <input 
                  type="text" 
                  placeholder="https://target.com" 
                  className="bg-gray-900 border border-gray-700 p-2 rounded-lg w-full focus:ring-2 focus:ring-blue-500 outline-none transition-all placeholder:text-gray-600"
                  value={newTarget}
                  onChange={e => setNewTarget(e.target.value)}
                />
              </div>
              <div className="flex items-end">
                <button type="submit" className="bg-blue-600 hover:bg-blue-500 text-white px-6 py-2 rounded-lg transition-all font-bold shadow-lg shadow-blue-900/20 active:scale-95 h-[42px]">
                  {Object.keys(routes).includes(newPath) ? "Update" : "Add"}
                </button>
              </div>
            </form>
          </div>

          <div className="bg-gray-800 p-6 rounded-xl border border-gray-700 shadow-lg h-[500px] flex flex-col">
            <h2 className="text-2xl font-bold mb-4 border-b border-gray-700 pb-2 flex justify-between items-center">
              <span>Live Events</span>
              <span className="text-xs font-normal text-gray-400 bg-gray-900 px-2 py-1 rounded-full flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></span>
                Polling 2s
              </span>
            </h2>
            <div className="overflow-y-auto flex-1 space-y-2 pr-2 scrollbar-thin scrollbar-thumb-gray-600 scrollbar-track-transparent">
              {events.map((ev: any, idx) => (
                <div key={idx} className="bg-gray-900 p-3 rounded border border-gray-700 text-sm flex items-center justify-between hover:bg-gray-850 transition-colors">
                  <div className="flex items-center gap-3">
                    <span className={`px-2 py-1 rounded text-xs font-bold ${ev.status >= 400 ? 'bg-red-900 text-red-300' : 'bg-green-900 text-green-300'}`}>
                      {ev.status}
                    </span>
                    <span className="font-mono text-gray-200">{ev.path}</span>
                  </div>
                  <div className="flex items-center gap-3 text-gray-400">
                    {ev.from_cache && <span className="bg-yellow-900/50 text-yellow-300 px-2 py-0.5 rounded border border-yellow-800 text-[10px] font-bold tracking-wider">CACHE</span>}
                    <span className="font-mono">{ev.latency}ms</span>
                  </div>
                </div>
              ))}
              {events.length === 0 && (
                <div className="flex flex-col items-center justify-center h-full text-gray-500">
                  <p>No events yet...</p>
                  <p className="text-sm">Make requests through the proxy to see them here.</p>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="bg-gray-800 p-6 rounded-xl border border-gray-700 shadow-lg">
          <h2 className="text-2xl font-bold mb-4 border-b border-gray-700 pb-2 flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            Proxy Terminal
          </h2>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-1 space-y-4">
              <p className="text-sm text-gray-400">Enter a path to test your proxy routing and caching.</p>
              <form onSubmit={handleTestRoute} className="space-y-3">
                <div className="relative">
                  <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 font-mono text-sm">GET</span>
                  <input 
                    type="text" 
                    placeholder="/posts" 
                    className="bg-gray-900 border border-gray-700 p-2 pl-12 rounded-lg w-full focus:ring-2 focus:ring-blue-500 outline-none transition-all font-mono text-sm"
                    value={testPath}
                    onChange={e => setTestPath(e.target.value)}
                  />
                </div>
                <button 
                  type="submit" 
                  disabled={isTesting}
                  className="w-full bg-blue-600 hover:bg-blue-500 disabled:bg-gray-700 text-white px-4 py-2 rounded-lg transition-all font-bold flex items-center justify-center gap-2"
                >
                  {isTesting ? (
                    <span className="w-4 h-4 border-2 border-white/20 border-t-white rounded-full animate-spin"></span>
                  ) : (
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  )}
                  Run Request
                </button>
              </form>
              <div className="space-y-2">
                <h4 className="text-xs font-bold text-gray-500 uppercase tracking-widest">Quick Tests</h4>
                <div className="flex flex-wrap gap-2">
                  {Object.keys(routes).slice(0, 5).map(path => (
                    <button 
                      key={path}
                      onClick={() => setTestPath(path)}
                      className="text-xs bg-gray-900 hover:bg-gray-700 border border-gray-700 px-2 py-1 rounded transition-colors font-mono"
                    >
                      {path}
                    </button>
                  ))}
                </div>
              </div>
            </div>
            <div className="lg:col-span-2">
              <div className="bg-black/50 rounded-xl border border-gray-700 h-[300px] flex flex-col overflow-hidden">
                <div className="bg-gray-900/80 px-4 py-2 border-b border-gray-700 flex justify-between items-center">
                  <span className="text-xs font-mono text-gray-400">Response Output</span>
                  {testResponse && (
                    <span className={`text-[10px] font-bold px-2 py-0.5 rounded ${testResponse.status < 400 ? 'bg-green-900 text-green-300' : 'bg-red-900 text-red-300'}`}>
                      HTTP {testResponse.status}
                    </span>
                  )}
                </div>
                <div className="flex-1 overflow-auto p-4 font-mono text-xs text-green-500 scrollbar-thin scrollbar-thumb-gray-800">
                  {testResponse ? (
                    <pre className="whitespace-pre-wrap">
                      {JSON.stringify(testResponse.data, null, 2)}
                    </pre>
                  ) : (
                    <div className="flex flex-col items-center justify-center h-full text-gray-700 italic">
                      <p>Waiting for request...</p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

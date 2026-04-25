"use client";

import { useEffect, useRef, useState } from "react";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Metrics {
  total_requests: number;
  error_count: number;
  cache_hits: number;
  cache_misses: number;
  avg_latency: number;
  error_rate: number;
}

interface ProxyEvent {
  path: string;
  status: number;
  latency: number;
  from_cache: boolean;
  request_id: string;
}

interface DashboardState {
  metrics: Metrics;
  events: ProxyEvent[];
  routes: Record<string, string[]>;
  health: Record<string, boolean>;
}

// ─── Stat Card ────────────────────────────────────────────────────────────────

function StatCard({
  label,
  value,
  color,
  icon,
}: {
  label: string;
  value: number;
  color: string;
  icon: React.ReactNode;
}) {
  return (
    <div
      className={`relative overflow-hidden bg-gray-800/60 backdrop-blur p-5 rounded-2xl border border-gray-700/50 shadow-xl transition-all duration-300 hover:-translate-y-1 hover:shadow-2xl hover:border-gray-600`}
    >
      <div className="flex items-start justify-between mb-3">
        <div className={`p-2 rounded-xl bg-gray-900/70 ${color}`}>{icon}</div>
        <span className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">{label}</span>
      </div>
      <p className={`text-4xl font-extrabold tabular-nums ${color}`}>{value.toLocaleString()}</p>
    </div>
  );
}

// ─── Status Badge ─────────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: number }) {
  const ok = status < 400;
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-bold tabular-nums ${
        ok
          ? "bg-emerald-900/60 text-emerald-300 border border-emerald-800/50"
          : "bg-red-900/60 text-red-300 border border-red-800/50"
      }`}
    >
      {status}
    </span>
  );
}

// ─── Main Dashboard ───────────────────────────────────────────────────────────

export default function Home() {
  const [metrics, setMetrics] = useState<Metrics>({
    total_requests: 0,
    error_count: 0,
    cache_hits: 0,
    cache_misses: 0,
    avg_latency: 0,
    error_rate: 0,
  });
  const [events, setEvents] = useState<ProxyEvent[]>([]);
  const [routes, setRoutes] = useState<Record<string, string[]>>({});
  const [health, setHealth] = useState<Record<string, boolean>>({});
  const [connected, setConnected] = useState(false);

  const [newPath, setNewPath] = useState("");
  const [newTarget, setNewTarget] = useState("");
  const [testPath, setTestPath] = useState("");
  const [testResponse, setTestResponse] = useState<any>(null);
  const [isTesting, setIsTesting] = useState(false);

  const eventsEndRef = useRef<HTMLDivElement>(null);

  // ── SSE connection (replaces polling) ───────────────────────────────────────
  useEffect(() => {
    let es: EventSource;
    let retryTimeout: ReturnType<typeof setTimeout>;

    const connect = () => {
      es = new EventSource(`${API_URL}/api/stream`);

      es.onopen = () => setConnected(true);

      es.onmessage = (e) => {
        try {
          const state: DashboardState = JSON.parse(e.data);
          setMetrics(state.metrics);
          setEvents(state.events ?? []);
          setRoutes(state.routes ?? {});
          setHealth(state.health ?? {});
        } catch {
          // malformed frame — ignore
        }
      };

      es.onerror = () => {
        setConnected(false);
        es.close();
        retryTimeout = setTimeout(connect, 3000);
      };
    };

    connect();
    return () => {
      es?.close();
      clearTimeout(retryTimeout);
    };
  }, []);

  // ── Route mutations (still use REST, then SSE will auto-refresh) ────────────

  const handleDeleteRoute = async (path: string) => {
    await fetch(`${API_URL}/api/routes?path=${encodeURIComponent(path)}`, {
      method: "DELETE",
    });
  };

  const handleEditRoute = (path: string, target: string) => {
    setNewPath(path);
    setNewTarget(target);
  };

  const handleAddRoute = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPath || !newTarget) return;
    const targets = newTarget
      .split(",")
      .map((t) => t.trim())
      .filter((t) => t);
    await fetch(`${API_URL}/api/routes`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: newPath, targets }),
    });
    setNewPath("");
    setNewTarget("");
  };

  const handleTestRoute = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!testPath) return;
    setIsTesting(true);
    setTestResponse(null);
    try {
      const res = await fetch(`${API_URL}${testPath.startsWith("/") ? testPath : "/" + testPath}`);
      const contentType = res.headers.get("content-type");
      const data = contentType && contentType.includes("application/json") ? await res.json() : await res.text();
      setTestResponse({
        status: res.status,
        data,
        headers: Object.fromEntries(res.headers.entries()),
      });
    } catch (err: any) {
      setTestResponse({ error: err.message });
    } finally {
      setIsTesting(false);
    }
  };

  const hitRate =
    metrics.cache_hits + metrics.cache_misses > 0
      ? Math.round((metrics.cache_hits / (metrics.cache_hits + metrics.cache_misses)) * 100)
      : 0;

  // ── Render ──────────────────────────────────────────────────────────────────

  return (
    <div className="min-h-screen bg-[#0d0f14] text-white font-sans">
      {/* Ambient background blobs */}
      <div aria-hidden className="pointer-events-none fixed inset-0 overflow-hidden">
        <div className="absolute -top-40 -left-40 w-[600px] h-[600px] rounded-full bg-blue-600/10 blur-[120px]" />
        <div className="absolute -bottom-40 -right-40 w-[600px] h-[600px] rounded-full bg-purple-600/10 blur-[120px]" />
      </div>

      <div className="relative max-w-7xl mx-auto px-6 py-10 space-y-8">
        {/* ── Header ── */}
        <header className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight bg-gradient-to-r from-blue-400 via-indigo-400 to-purple-500 bg-clip-text text-transparent">
              Go Reverse Proxy
            </h1>
            <p className="text-sm text-gray-500 mt-1">Live dashboard · real-time via SSE</p>
          </div>
          <div className="flex items-center gap-2 text-xs font-semibold">
            <span className={`w-2 h-2 rounded-full ${connected ? "bg-emerald-400 animate-pulse" : "bg-red-500"}`} />
            <span className={connected ? "text-emerald-400" : "text-red-400"}>
              {connected ? "Connected" : "Reconnecting…"}
            </span>
          </div>
        </header>

        {/* ── Stat Cards ── */}
        <section className="grid grid-cols-2 md:grid-cols-6 gap-4">
          <StatCard
            label="Total Requests"
            value={metrics.total_requests}
            color="text-blue-400"
            icon={
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            }
          />
          <StatCard
            label="Errors"
            value={metrics.error_count}
            color="text-red-400"
            icon={
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            }
          />
          <StatCard
            label="Cache Hits"
            value={metrics.cache_hits}
            color="text-emerald-400"
            icon={
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            }
          />
          <StatCard
            label="Cache Misses"
            value={metrics.cache_misses}
            color="text-amber-400"
            icon={
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            }
          />
          <StatCard
            label="Avg Latency"
            value={Math.round(metrics.avg_latency)}
            color="text-purple-400"
            icon={
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            }
          />
          <StatCard
            label="Error Rate"
            value={Math.round(metrics.error_rate * 100) / 100}
            color="text-red-400"
            icon={
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6"
                />
              </svg>
            }
          />
        </section>

        {/* ── Cache hit-rate bar ── */}
        <div className="bg-gray-800/50 backdrop-blur rounded-2xl border border-gray-700/50 p-5">
          <div className="flex justify-between items-center mb-2 text-sm">
            <span className="text-gray-400 font-medium">Cache Hit Rate</span>
            <span className="font-bold text-white">{hitRate}%</span>
          </div>
          <div className="h-2.5 rounded-full bg-gray-700 overflow-hidden">
            <div
              className="h-full rounded-full bg-gradient-to-r from-emerald-500 to-blue-500 transition-all duration-700"
              style={{ width: `${hitRate}%` }}
            />
          </div>
        </div>

        {/* ── Routes + Events ── */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Routes */}
          <div className="bg-gray-800/50 backdrop-blur rounded-2xl border border-gray-700/50 p-6 flex flex-col gap-4">
            <h2 className="text-lg font-bold text-white flex items-center gap-2">
              <svg className="w-5 h-5 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7"
                />
              </svg>
              Routes
              <span className="ml-auto text-xs text-gray-500 font-normal bg-gray-900/60 px-2 py-0.5 rounded-full">
                {Object.keys(routes).length} configured
              </span>
            </h2>

            {/* Route list */}
            <div className="space-y-2 max-h-60 overflow-y-auto pr-1 scrollbar-thin scrollbar-thumb-gray-700">
              {Object.entries(routes).map(([path, targets]) => (
                <div
                  key={path}
                  className="group flex items-start justify-between bg-gray-900/60 px-3 py-2.5 rounded-xl border border-gray-700/40 hover:border-indigo-500/40 transition-colors"
                >
                  <div className="flex flex-col gap-1 min-w-0 flex-1">
                    <span className="font-mono text-indigo-300 text-sm font-semibold shrink-0">{path}</span>
                    <div className="flex flex-wrap gap-1">
                      {targets.map((target, idx) => (
                        <span
                          key={idx}
                          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-mono ${
                            health[target]
                              ? "bg-emerald-900/40 text-emerald-300 border border-emerald-800/40"
                              : "bg-red-900/40 text-red-300 border border-red-800/40"
                          }`}
                        >
                          <span
                            className={`w-1.5 h-1.5 rounded-full ${health[target] ? "bg-emerald-400" : "bg-red-400"}`}
                          />
                          {target}
                        </span>
                      ))}
                    </div>
                  </div>
                  <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0 ml-2">
                    <button
                      onClick={() => handleEditRoute(path, targets.join(", "))}
                      className="p-1.5 rounded-lg hover:bg-gray-700 text-gray-500 hover:text-blue-400 transition-colors"
                      title="Edit"
                    >
                      <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M11 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-5M16.243 3.757a4.5 4.5 0 116.364 6.364L12 20.364l-7.682-7.682L16.243 3.757z"
                        />
                      </svg>
                    </button>
                    <button
                      onClick={() => handleDeleteRoute(path)}
                      className="p-1.5 rounded-lg hover:bg-gray-700 text-gray-500 hover:text-red-400 transition-colors"
                      title="Delete"
                    >
                      <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                        />
                      </svg>
                    </button>
                  </div>
                </div>
              ))}
              {Object.keys(routes).length === 0 && (
                <p className="text-sm text-gray-600 text-center py-4">No routes configured.</p>
              )}
            </div>

            {/* Add / update form */}
            <form
              onSubmit={handleAddRoute}
              className="grid grid-cols-[1fr_1fr_auto] gap-2 bg-gray-900/40 p-3 rounded-xl border border-gray-700/40"
            >
              <div className="flex flex-col gap-1">
                <label className="text-[10px] text-gray-500 font-bold uppercase tracking-widest">Path</label>
                <input
                  type="text"
                  placeholder="/api/users"
                  value={newPath}
                  onChange={(e) => setNewPath(e.target.value)}
                  className="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-indigo-500 transition-all placeholder:text-gray-600"
                />
              </div>
              <div className="flex flex-col gap-1">
                <label className="text-[10px] text-gray-500 font-bold uppercase tracking-widest">Targets</label>
                <input
                  type="text"
                  placeholder="https://api1.com, https://api2.com"
                  value={newTarget}
                  onChange={(e) => setNewTarget(e.target.value)}
                  className="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-indigo-500 transition-all placeholder:text-gray-600"
                />
              </div>
              <div className="flex items-end">
                <button
                  type="submit"
                  className="h-[38px] px-4 rounded-lg bg-indigo-600 hover:bg-indigo-500 active:scale-95 text-white font-bold text-sm transition-all shadow-lg shadow-indigo-900/30"
                >
                  {Object.keys(routes).includes(newPath) ? "Update" : "Add"}
                </button>
              </div>
            </form>
          </div>

          {/* Events Feed */}
          <div className="bg-gray-800/50 backdrop-blur rounded-2xl border border-gray-700/50 p-6 flex flex-col">
            <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-4">
              <svg className="w-5 h-5 text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 6h16M4 10h16M4 14h16M4 18h16"
                />
              </svg>
              Live Events
              <span className="ml-auto">
                <span
                  className={`text-[10px] font-semibold px-2 py-0.5 rounded-full flex items-center gap-1.5 ${
                    connected
                      ? "bg-emerald-900/40 text-emerald-400 border border-emerald-800/40"
                      : "bg-gray-800 text-gray-500 border border-gray-700"
                  }`}
                >
                  <span
                    className={`w-1.5 h-1.5 rounded-full ${connected ? "bg-emerald-400 animate-pulse" : "bg-gray-600"}`}
                  />
                  SSE
                </span>
              </span>
            </h2>

            <div className="flex-1 overflow-y-auto space-y-1.5 max-h-72 pr-1 scrollbar-thin scrollbar-thumb-gray-700">
              {events.map((ev, idx) => (
                <div
                  key={idx}
                  className="flex items-center justify-between bg-gray-900/60 px-3 py-2 rounded-xl text-sm border border-gray-700/30 hover:border-gray-600/50 transition-colors"
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <StatusBadge status={ev.status} />
                    <span className="font-mono text-gray-300 text-xs truncate">{ev.path}</span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0 ml-2">
                    {ev.from_cache && (
                      <span className="text-[9px] font-bold bg-amber-900/40 text-amber-300 border border-amber-800/40 px-1.5 py-0.5 rounded tracking-widest">
                        CACHE
                      </span>
                    )}
                    <span className="text-xs font-mono text-gray-500">{ev.latency}ms</span>
                  </div>
                </div>
              ))}
              {events.length === 0 && (
                <div className="flex flex-col items-center justify-center h-48 text-gray-600">
                  <svg className="w-8 h-8 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={1.5}
                      d="M8 12h.01M12 12h.01M16 12h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  <p className="text-sm">No events yet</p>
                  <p className="text-xs mt-1">Route a request to see it here.</p>
                </div>
              )}
              <div ref={eventsEndRef} />
            </div>
          </div>
        </div>

        {/* ── Proxy Terminal ── */}
        <div className="bg-gray-800/50 backdrop-blur rounded-2xl border border-gray-700/50 p-6">
          <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-5">
            <svg className="w-5 h-5 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
              />
            </svg>
            Proxy Terminal
          </h2>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Input side */}
            <div className="space-y-4">
              <p className="text-sm text-gray-500">Fire a test request through the proxy and inspect the response.</p>
              <form onSubmit={handleTestRoute} className="space-y-3">
                <div className="relative">
                  <span className="absolute left-3 top-1/2 -translate-y-1/2 text-xs font-mono text-gray-500 font-bold">
                    GET
                  </span>
                  <input
                    type="text"
                    placeholder="/api/posts"
                    value={testPath}
                    onChange={(e) => setTestPath(e.target.value)}
                    className="bg-gray-900 border border-gray-700 rounded-xl pl-12 pr-3 py-2.5 w-full font-mono text-sm outline-none focus:ring-2 focus:ring-blue-500 transition-all placeholder:text-gray-600"
                  />
                </div>
                <button
                  type="submit"
                  disabled={isTesting}
                  className="w-full flex items-center justify-center gap-2 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-700 rounded-xl py-2.5 font-bold text-sm transition-all active:scale-95 shadow-lg shadow-blue-900/20"
                >
                  {isTesting ? (
                    <span className="w-4 h-4 border-2 border-white/20 border-t-white rounded-full animate-spin" />
                  ) : (
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
                      />
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                      />
                    </svg>
                  )}
                  Run Request
                </button>
              </form>

              {/* Quick picks */}
              {Object.keys(routes).length > 0 && (
                <div className="space-y-1.5">
                  <p className="text-[10px] font-bold text-gray-600 uppercase tracking-widest">Quick test</p>
                  <div className="flex flex-wrap gap-1.5">
                    {Object.keys(routes)
                      .slice(0, 6)
                      .map((path) => (
                        <button
                          key={path}
                          onClick={() => setTestPath(path)}
                          className="text-xs bg-gray-900 hover:bg-gray-700 border border-gray-700 hover:border-gray-600 px-2.5 py-1 rounded-lg font-mono transition-all"
                        >
                          {path}
                        </button>
                      ))}
                  </div>
                </div>
              )}
            </div>

            {/* Output side */}
            <div className="lg:col-span-2">
              <div className="bg-black/60 rounded-2xl border border-gray-700/60 overflow-hidden h-[280px] flex flex-col">
                {/* Terminal toolbar */}
                <div className="flex items-center justify-between bg-gray-900/80 px-4 py-2 border-b border-gray-800">
                  <div className="flex items-center gap-1.5">
                    <span className="w-2.5 h-2.5 rounded-full bg-red-500/70" />
                    <span className="w-2.5 h-2.5 rounded-full bg-amber-500/70" />
                    <span className="w-2.5 h-2.5 rounded-full bg-emerald-500/70" />
                  </div>
                  <span className="text-xs font-mono text-gray-500">response.json</span>
                  {testResponse && (
                    <span
                      className={`text-[10px] font-bold px-2 py-0.5 rounded ${
                        testResponse.status < 400 ? "bg-emerald-900/60 text-emerald-300" : "bg-red-900/60 text-red-300"
                      }`}
                    >
                      HTTP {testResponse.status}
                    </span>
                  )}
                </div>
                {/* Content */}
                <div className="flex-1 overflow-auto p-4 font-mono text-xs text-emerald-400 scrollbar-thin scrollbar-thumb-gray-800">
                  {testResponse ? (
                    <pre className="whitespace-pre-wrap">
                      {testResponse.error ? `Error: ${testResponse.error}` : JSON.stringify(testResponse.data, null, 2)}
                    </pre>
                  ) : (
                    <div className="flex items-center justify-center h-full text-gray-700 italic select-none">
                      <p>Waiting for request…</p>
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

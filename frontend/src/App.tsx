import { useState } from "react";
import {
  Link,
  MoveRight,
  Loader2,
  Home,
  Globe,
  AlertTriangle,
} from "lucide-react";

// --- Types ---

type HeadingLevel = "h1" | "h2" | "h3" | "h4" | "h5" | "h6";

interface PageAnalysis {
  url: string;
  html_version: string;
  title: string;
  headings: Record<HeadingLevel, number>;
  links: {
    internal_count: number;
    external_count: number;
    inaccessible_count: number;
  };
  has_login_form: boolean;
}

interface ErrorResponse {
  error: string;
  status_code: number;
  message: string;
}

// --- API ---

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

async function analyzeUrl(url: string): Promise<PageAnalysis> {
  const res = await fetch(`${API_URL}/analyze`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  });
  if (!res.ok) {
    let err: ErrorResponse;
    try {
      err = await res.json();
    } catch {
      err = {
        error: "Error",
        status_code: res.status,
        message: res.statusText || "Unable to connect to analysis service",
      };
    }
    throw err;
  }
  return res.json();
}

// --- Validation ---

function isValidUrl(input: string): boolean {
  try {
    const parsed = new URL(input);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

// --- Component ---

const HEADING_LEVELS: HeadingLevel[] = ["h1", "h2", "h3", "h4", "h5", "h6"];

function App() {
  const [url, setUrl] = useState("");
  const [result, setResult] = useState<PageAnalysis | null>(null);
  const [error, setError] = useState<ErrorResponse | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!isValidUrl(url)) {
      setError({
        error: "Validation Error",
        status_code: 0,
        message: "Please enter a valid URL (e.g., https://example.com)",
      });
      return;
    }

    setError(null);
    setLoading(true);

    try {
      const data = await analyzeUrl(url);
      setResult(data);
      setError(null);
    } catch (err) {
      const apiError =
        err && typeof err === "object" && "status_code" in err
          ? (err as ErrorResponse)
          : {
              error: "Network Error",
              status_code: 0,
              message: "Unable to connect to analysis service",
            };
      setError(apiError);
      setResult(null);
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="min-h-screen bg-white text-gray-900">
      <div className="mx-auto max-w-2xl px-4 py-16">
        <h1 className="mb-8 text-center text-3xl font-bold">Page Insight</h1>

        {/* Search Form */}
        <form onSubmit={handleSubmit}>
          <div className="flex items-center border border-gray-300">
            <Link className="ml-3 h-5 w-5 shrink-0 text-gray-400" />
            <input
              type="text"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://example.com"
              disabled={loading}
              className="flex-1 px-3 py-3 text-sm outline-none disabled:bg-gray-50 disabled:text-gray-400"
            />
            <button
              type="submit"
              disabled={loading}
              className="flex items-center gap-2 bg-gray-900 px-4 py-3 text-sm font-medium text-white hover:bg-gray-800 disabled:opacity-50"
            >
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <MoveRight className="h-4 w-4" />
              )}
              Analyze URL
            </button>
          </div>
        </form>

        {/* Error Alert */}
        {error && (
          <div
            role="alert"
            className="mt-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800"
          >
            {error.status_code > 0 && (
              <span className="font-medium">Error {error.status_code}: </span>
            )}
            {error.message}
          </div>
        )}

        {/* Results */}
        <div aria-live="polite">
          {result && (
            <div className="mt-8 space-y-6">
              {/* General Info */}
              <div className="border border-gray-200 p-4">
                <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-gray-500">
                  General Info
                </h2>
                <dl className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <dt className="text-gray-500">HTML Version</dt>
                    <dd className="font-medium">{result.html_version}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-gray-500">Page Title</dt>
                    <dd className="font-medium">
                      {result.title || "(No title)"}
                    </dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-gray-500">Login Form</dt>
                    <dd className="font-medium">
                      {result.has_login_form ? "Yes" : "No"}
                    </dd>
                  </div>
                </dl>
              </div>

              {/* Headings Table */}
              <div className="border border-gray-200 p-4">
                <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-gray-500">
                  Headings
                </h2>
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 text-left text-gray-500">
                      <th className="pb-2 font-medium">Level</th>
                      <th className="pb-2 text-right font-medium">Count</th>
                    </tr>
                  </thead>
                  <tbody>
                    {HEADING_LEVELS.map((level) => (
                      <tr key={level} className="border-b border-gray-100">
                        <td className="py-2 uppercase">{level}</td>
                        <td className="py-2 text-right font-medium">
                          {result.headings[level]}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Link Metrics */}
              <div className="border border-gray-200 p-4">
                <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-gray-500">
                  Links
                </h2>
                <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                  <div className="flex items-center gap-3 rounded border border-gray-100 p-3">
                    <Home className="h-5 w-5 text-gray-600" />
                    <div>
                      <p className="text-xs text-gray-500">Internal</p>
                      <p className="text-lg font-semibold">
                        {result.links.internal_count}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3 rounded border border-gray-100 p-3">
                    <Globe className="h-5 w-5 text-gray-600" />
                    <div>
                      <p className="text-xs text-gray-500">External</p>
                      <p className="text-lg font-semibold">
                        {result.links.external_count}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3 rounded border border-gray-100 p-3">
                    <AlertTriangle className="h-5 w-5 text-red-500" />
                    <div>
                      <p className="text-xs text-gray-500">Inaccessible</p>
                      <p className="text-lg font-semibold text-red-600">
                        {result.links.inaccessible_count}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </main>
  );
}

export default App;

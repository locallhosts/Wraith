import { useEffect, useState } from "react";
import { getApiKey, setApiKey } from "../lib/api";

export default function ApiKeyBar() {
  const [key, setKey] = useState("");
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    setKey(getApiKey());
  }, []);

  return (
    <div className="flex items-center gap-2 text-xs">
      <label className="text-zinc-600">API key</label>
      <input
        type="password"
        value={key}
        onChange={(e) => {
          setKey(e.target.value);
          setSaved(false);
        }}
        placeholder="wraith_..."
        className="w-40 rounded border border-zinc-800 bg-zinc-900 px-2 py-1 font-mono text-zinc-300 focus:border-zinc-600 focus:outline-none"
      />
      <button
        onClick={() => {
          setApiKey(key);
          setSaved(true);
        }}
        className="rounded border border-zinc-700 px-2 py-1 text-zinc-400 hover:border-zinc-500 hover:text-zinc-200"
      >
        {saved ? "Saved" : "Save"}
      </button>
    </div>
  );
}

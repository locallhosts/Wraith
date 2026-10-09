import { useState } from "react";
import { fetchSession, getApiKey, setApiKey } from "../lib/api";

type SessionState = { kind: "idle" | "checking" | "valid" | "error"; message: string };

export default function ApiKeyBar() {
  const [key, setKey] = useState(() => getApiKey());
  const [session, setSession] = useState<SessionState>({ kind: "idle", message: "Key stays in memory and is cleared on page reload." });

  async function saveKey() {
    setApiKey(key);
    setSession({ kind: "checking", message: key.trim() ? "Checking API key…" : "API key cleared from this page session." });
    if (!key.trim()) return;
    try {
      const result = await fetchSession();
      if (result.authenticated === false) {
        setSession({ kind: "error", message: "The API did not authenticate this key. Check the key and try again." });
        return;
      }
      const role = result.role ? ` · role: ${result.role}` : "";
      const label = result.label ? ` · ${result.label}` : "";
      setSession({ kind: "valid", message: `API responded to the session check${role}${label}.` });
    } catch (error) {
      setSession({ kind: "error", message: error instanceof Error ? error.message : "Session check failed. Verify API connectivity and key permissions." });
    }
  }

  return (
    <div className="api-key-control" aria-label="API authentication">
      <label htmlFor="wraith-api-key">API key</label>
      <input id="wraith-api-key" type="password" autoComplete="off" value={key}
        onChange={(event) => { setKey(event.target.value); setSession({ kind: "idle", message: "Unsaved key changes." }); }}
        placeholder="Paste API key" />
      <button type="button" onClick={saveKey} disabled={session.kind === "checking"}>{session.kind === "checking" ? "Checking…" : "Apply key"}</button>
      <p role={session.kind === "error" ? "alert" : "status"} className={`api-key-status ${session.kind}`}>{session.message}</p>
    </div>
  );
}

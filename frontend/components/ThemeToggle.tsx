import { useEffect, useState } from "react";

type Theme = "dark" | "light";
const KEY = "wraith-theme";

function apply(theme: Theme) {
  document.documentElement.classList.toggle("wraith-light", theme === "light");
  document.body.classList.toggle("wraith-light", theme === "light");
}

export default function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>("dark");
  useEffect(() => { const saved = window.localStorage.getItem(KEY); const next: Theme = saved === "light" ? "light" : "dark"; setTheme(next); apply(next); }, []);
  const change = (next: Theme) => { setTheme(next); window.localStorage.setItem(KEY, next); apply(next); };
  return <div className="flex items-center rounded-md border border-zinc-800 bg-[#0a0e13] p-0.5" aria-label="Theme selection">
    {(["dark", "light"] as Theme[]).map(value => <button key={value} type="button" onClick={() => change(value)} aria-pressed={theme === value} className={`rounded px-2.5 py-1.5 text-[10px] font-medium ${theme === value ? "bg-zinc-800 text-zinc-100" : "text-zinc-600 hover:text-zinc-300"}`}>{value[0].toUpperCase() + value.slice(1)}</button>)}
  </div>;
}

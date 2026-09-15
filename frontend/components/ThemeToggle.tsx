import { useEffect, useState } from "react";

const STORAGE_KEY = "wraith-theme";
type Theme = "dark" | "light";

function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle("wraith-light", theme === "light");
  document.documentElement.dataset.theme = theme;
  document.body.classList.toggle("wraith-light", theme === "light");
}

export default function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>("dark");

  useEffect(() => {
    const saved = window.localStorage.getItem(STORAGE_KEY);
    const next: Theme = saved === "light" ? "light" : "dark";
    setTheme(next);
    applyTheme(next);
  }, []);

  const changeTheme = (next: Theme) => {
    setTheme(next);
    window.localStorage.setItem(STORAGE_KEY, next);
    applyTheme(next);
  };

  return (
    <div className="flex items-center rounded-md border border-zinc-800 bg-[#0a0e13] p-0.5" aria-label="Theme selection">
      <button
        type="button"
        onClick={() => changeTheme("dark")}
        aria-pressed={theme === "dark"}
        className={`rounded px-2.5 py-1.5 text-[10px] font-medium transition ${theme === "dark" ? "bg-zinc-800 text-zinc-100" : "text-zinc-600 hover:text-zinc-300"}`}
      >
        Dark
      </button>
      <button
        type="button"
        onClick={() => changeTheme("light")}
        aria-pressed={theme === "light"}
        className={`rounded px-2.5 py-1.5 text-[10px] font-medium transition ${theme === "light" ? "bg-zinc-100 text-zinc-900" : "text-zinc-600 hover:text-zinc-300"}`}
      >
        Light
      </button>
    </div>
  );
}

import { create } from "zustand";

export type ThemeMode = "dark" | "light";

const STORAGE_KEY = "agentic-rag-theme";

function readInitial(): ThemeMode {
  if (
    typeof document !== "undefined" &&
    document.documentElement.classList.contains("light")
  ) {
    return "light";
  }

  try {
    return localStorage.getItem(STORAGE_KEY) === "light" ? "light" : "dark";
  } catch {
    return "dark";
  }
}

function applyTheme(mode: ThemeMode): void {
  if (typeof document !== "undefined") {
    document.documentElement.classList.toggle("light", mode === "light");
  }

  try {
    localStorage.setItem(STORAGE_KEY, mode);
  } catch {
    // Storage may be unavailable in private browsing contexts.
  }
}

interface ThemeState {
  mode: ThemeMode;
  toggle: () => void;
}

export const useThemeStore = create<ThemeState>()((set, get) => {
  const mode = readInitial();
  applyTheme(mode);

  return {
    mode,
    toggle() {
      const nextMode = get().mode === "dark" ? "light" : "dark";
      applyTheme(nextMode);
      set({ mode: nextMode });
    },
  };
});

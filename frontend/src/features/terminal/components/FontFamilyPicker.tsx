import { useEffect, useRef, useState } from "react";
import { useTerminalThemeStore, DEFAULT_FONT_FAMILY } from "../stores/terminal-theme.store";

const CUSTOM_VALUE = "__custom__";

// Curated fallback list shown when queryLocalFonts() is unavailable
const BUILTIN_FONTS = [
  "MesloLGS NF",
  "Meslo LG S Nerd Font Mono",
  "Meslo LG S for Powerline",
  "Meslo LG M for Powerline",
  "JetBrainsMono Nerd Font Mono",
  "JetBrains Mono",
  "FiraCode Nerd Font Mono",
  "Fira Code",
  "Fira Mono for Powerline",
  "Hack Nerd Font Mono",
  "Source Code Pro for Powerline",
  "DejaVu Sans Mono for Powerline",
  "Inconsolata for Powerline",
  "Roboto Mono for Powerline",
  "Ubuntu Mono derivative Powerline",
  "Symbols Nerd Font Mono",
  "Menlo",
  "Monaco",
  "Courier New",
];

const MONOSPACE_KEYWORDS = [
  "mono", "code", "console", "terminal", "typewriter", "courier",
  "hack", "nerd", "powerline", "menlo", "monaco", "inconsolata",
  "fira", "droid", "source code", "jetbrains", "cascadia", "meslo",
  "ubuntu mono", "liberation mono", "roboto mono", "space mono",
];

function looksMonospace(family: string): boolean {
  const lower = family.toLowerCase();
  return MONOSPACE_KEYWORDS.some((kw) => lower.includes(kw));
}

interface FontData { family: string }

const inputStyle: React.CSSProperties = {
  backgroundColor: "var(--cmux-surface-hover)",
  border: "1px solid var(--cmux-border-light)",
  color: "var(--cmux-text)",
};

export function FontFamilyPicker() {
  const { fontFamily, setFontFamily } = useTerminalThemeStore();
  const [detectedFonts, setDetectedFonts] = useState<string[] | null>(null);
  const [customValue, setCustomValue] = useState("");
  const customInputRef = useRef<HTMLInputElement>(null);

  const fonts = detectedFonts ?? BUILTIN_FONTS;
  const isKnownFont = fonts.includes(fontFamily);
  const selectValue = isKnownFont ? fontFamily : CUSTOM_VALUE;

  useEffect(() => {
    if (!("queryLocalFonts" in window)) return;
    (window as unknown as { queryLocalFonts: () => Promise<FontData[]> })
      .queryLocalFonts()
      .then((raw) => {
        const seen = new Set<string>();
        const families: string[] = [];
        for (const font of raw) {
          if (!seen.has(font.family) && looksMonospace(font.family)) {
            seen.add(font.family);
            families.push(font.family);
          }
        }
        setDetectedFonts(families.sort());
      })
      .catch(() => { /* stay on builtin list */ });
  }, []);

  useEffect(() => {
    if (!isKnownFont) setCustomValue(fontFamily);
  }, [fontFamily, isKnownFont]);

  function handleSelectChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const val = e.target.value;
    if (val === CUSTOM_VALUE) {
      setTimeout(() => customInputRef.current?.focus(), 0);
    } else {
      setFontFamily(val);
    }
  }

  function commitCustom() {
    setFontFamily(customValue.trim() || DEFAULT_FONT_FAMILY);
  }

  return (
    <div className="space-y-2">
      <span
        className="block px-1 text-xs font-medium uppercase tracking-wider"
        style={{ color: "var(--cmux-text-muted)" }}
      >
        Terminal Font
      </span>

      <select
        value={selectValue}
        onChange={handleSelectChange}
        className="w-full rounded px-2.5 py-2 text-xs outline-none"
        style={inputStyle}
      >
        {fonts.map((f) => (
          <option key={f} value={f}>{f}</option>
        ))}
        <option value={CUSTOM_VALUE}>Custom…</option>
      </select>

      {selectValue === CUSTOM_VALUE && (
        <input
          ref={customInputRef}
          type="text"
          value={customValue}
          onChange={(e) => setCustomValue(e.target.value)}
          onBlur={commitCustom}
          onKeyDown={(e) => e.key === "Enter" && commitCustom()}
          placeholder="e.g. 'Fira Code', monospace"
          className="w-full rounded px-2.5 py-2 text-xs outline-none"
          style={inputStyle}
        />
      )}

      <button
        type="button"
        onClick={() => { setFontFamily(DEFAULT_FONT_FAMILY); setCustomValue(""); }}
        className="text-xs transition-colors"
        style={{ color: "var(--cmux-text-muted)" }}
      >
        Reset to default
      </button>
    </div>
  );
}

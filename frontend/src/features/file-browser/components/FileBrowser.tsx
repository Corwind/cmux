import { useEffect, useState } from "react";
import { useFileBrowser } from "../hooks/useFileBrowser";
import type { DirEntry } from "@/features/sessions";

interface FileBrowserProps {
  onSelect: (path: string) => void;
  onClose: () => void;
  showHidden?: boolean;
  showFiles?: boolean;
}

// SymlinkArrow is a small badge overlaid on an entry icon to mark it as a
// symbolic link.
function SymlinkArrow() {
  return (
    <svg
      className="absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5"
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M7 17L17 7M17 7H9M17 7V15" stroke="currentColor" strokeWidth={3} strokeLinecap="round" strokeLinejoin="round" fill="none" />
    </svg>
  );
}

function FolderIcon({ entry }: { entry: DirEntry }) {
  return (
    <span className="relative inline-flex shrink-0 text-yellow-500">
      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
        />
      </svg>
      {entry.is_symlink && <SymlinkArrow />}
    </span>
  );
}

function FileIcon({ entry }: { entry: DirEntry }) {
  return (
    <span className={`relative inline-flex shrink-0 ${entry.is_broken ? "text-red-400" : "text-gray-500"}`}>
      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
        />
      </svg>
      {entry.is_symlink && <SymlinkArrow />}
    </span>
  );
}

export function FileBrowser({ onSelect, onClose, showHidden, showFiles }: FileBrowserProps) {
  const [currentPath, setCurrentPath] = useState<string | undefined>(undefined);
  const { data, isLoading, error } = useFileBrowser(currentPath, showHidden);

  useEffect(() => {
    if (data && !currentPath) {
      setCurrentPath(data.path);
    }
  }, [data, currentPath]);

  const displayPath = currentPath ?? data?.path ?? "~";
  const entries = Array.isArray(data?.entries) ? data.entries : [];

  function childPath(name: string) {
    return `${displayPath === "/" ? "" : displayPath}/${name}`;
  }

  function navigateUp() {
    if (!currentPath) return;
    const parent = currentPath.replace(/\/[^/]+\/?$/, "") || "/";
    setCurrentPath(parent);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="flex h-[480px] w-[460px] flex-col rounded-lg border border-gray-700 bg-gray-900 shadow-xl">
        <div className="flex items-center justify-between border-b border-gray-700 px-4 py-3">
          <h2 className="text-sm font-semibold text-white">
            Select Directory
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1 text-gray-400 hover:bg-gray-700 hover:text-white"
          >
            <svg
              className="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        <div className="flex items-center gap-2 border-b border-gray-700 px-4 py-2">
          <button
            type="button"
            onClick={navigateUp}
            disabled={displayPath === "/"}
            className="rounded p-1 text-gray-400 hover:bg-gray-700 hover:text-white disabled:opacity-30"
            title="Go up"
          >
            <svg
              className="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M5 15l7-7 7 7"
              />
            </svg>
          </button>
          <span className="min-w-0 flex-1 truncate text-xs font-mono text-gray-300">
            {displayPath}
          </span>
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          {isLoading && (
            <div className="p-4 text-center text-sm text-gray-500">
              Loading...
            </div>
          )}
          {error && (
            <div className="p-4 text-center text-sm text-red-400">
              Failed to list directory
            </div>
          )}
          {entries.length > 0 && (
            <ul className="space-y-0.5">
              {entries
                .filter((entry) => entry.is_dir)
                .map((entry) => (
                  <li key={entry.name}>
                    <button
                      type="button"
                      onClick={() => setCurrentPath(childPath(entry.name))}
                      title={entry.is_symlink ? `${entry.name} (symlink)` : entry.name}
                      className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm text-gray-300 hover:bg-gray-800 hover:text-white"
                    >
                      <FolderIcon entry={entry} />
                      <span className="truncate">{entry.name}</span>
                    </button>
                  </li>
                ))}
              {entries
                .filter((entry) => !entry.is_dir && (showFiles || entry.is_broken))
                .map((entry) =>
                  entry.is_broken ? (
                    <li key={entry.name}>
                      <div
                        title={`${entry.name} (broken symlink)`}
                        className="flex w-full cursor-not-allowed items-center gap-2 rounded px-2 py-1.5 text-left text-sm text-red-400"
                      >
                        <FileIcon entry={entry} />
                        <span className="truncate">{entry.name}</span>
                      </div>
                    </li>
                  ) : (
                    <li key={entry.name}>
                      <button
                        type="button"
                        onClick={() => onSelect(childPath(entry.name))}
                        title={entry.is_symlink ? `${entry.name} (symlink)` : entry.name}
                        className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm text-gray-400 hover:bg-gray-800 hover:text-white"
                      >
                        <FileIcon entry={entry} />
                        <span className="truncate">{entry.name}</span>
                      </button>
                    </li>
                  ),
                )}
            </ul>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-gray-700 px-4 py-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded border border-gray-600 px-3 py-1.5 text-sm text-gray-400 hover:text-white"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => displayPath && onSelect(displayPath)}
            className="rounded bg-green-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-green-500"
          >
            Select: {displayPath}
          </button>
        </div>
      </div>
    </div>
  );
}

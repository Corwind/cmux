import { useState } from "react";
import { useFileBrowser } from "../hooks/useFileBrowser";

interface FileBrowserProps {
  onSelect: (path: string) => void;
  onClose: () => void;
}

export function FileBrowser({ onSelect, onClose }: FileBrowserProps) {
  const [currentPath, setCurrentPath] = useState("/");
  const { data: entries, isLoading, error } = useFileBrowser(currentPath);

  function navigateUp() {
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
            disabled={currentPath === "/"}
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
            {currentPath}
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
          {entries && (
            <ul className="space-y-0.5">
              {entries
                .filter((entry) => entry.is_dir)
                .map((entry) => (
                  <li key={entry.name}>
                    <button
                      type="button"
                      onClick={() => setCurrentPath(`${currentPath === "/" ? "" : currentPath}/${entry.name}`)}
                      className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm text-gray-300 hover:bg-gray-800 hover:text-white"
                    >
                      <svg
                        className="h-4 w-4 shrink-0 text-yellow-500"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        strokeWidth={2}
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
                        />
                      </svg>
                      <span className="truncate">{entry.name}</span>
                    </button>
                  </li>
                ))}
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
            onClick={() => onSelect(currentPath)}
            className="rounded bg-green-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-green-500"
          >
            Select: {currentPath}
          </button>
        </div>
      </div>
    </div>
  );
}

import { LoadingSpinner } from "@/components/common";
import { useExamples } from "../hooks/useExamples";

export function ExampleList() {
  const { data, isLoading, error } = useExamples();

  if (isLoading) {
    return <LoadingSpinner />;
  }

  if (error) {
    return (
      <div className="p-4 text-red-600" role="alert">
        Failed to load examples: {error.message}
      </div>
    );
  }

  const examples = data?.data ?? [];

  if (examples.length === 0) {
    return <div className="p-4 text-gray-500">No examples found.</div>;
  }

  return (
    <ul className="space-y-4">
      {examples.map((example) => (
        <li key={example.id} className="rounded-lg border border-gray-200 p-4">
          <h3 className="text-lg font-semibold">{example.title}</h3>
          <p className="text-gray-600">{example.description}</p>
        </li>
      ))}
    </ul>
  );
}

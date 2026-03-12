import { http, HttpResponse } from "msw";

const mockExamples = [
  {
    id: "1",
    title: "First Example",
    description: "This is the first example",
    createdAt: "2024-01-01T00:00:00Z",
  },
  {
    id: "2",
    title: "Second Example",
    description: "This is the second example",
    createdAt: "2024-01-02T00:00:00Z",
  },
];

export const handlers = [
  http.get("http://localhost:3001/api/examples", () => {
    return HttpResponse.json({ data: mockExamples });
  }),
  http.get("http://localhost:3001/api/examples/:id", ({ params }) => {
    const example = mockExamples.find((e) => e.id === params.id);
    if (!example) {
      return HttpResponse.json({ message: "Not found" }, { status: 404 });
    }
    return HttpResponse.json({ data: example });
  }),
  http.post("http://localhost:3001/api/examples", async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    const newExample = {
      id: String(mockExamples.length + 1),
      title: body.title as string,
      description: body.description as string,
      createdAt: new Date().toISOString(),
    };
    return HttpResponse.json({ data: newExample }, { status: 201 });
  }),
];

import { render, screen, waitFor } from "@/test/test-utils";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";
import { server } from "@/test/mocks/server";
import { ExampleList } from "./ExampleList";

describe("ExampleList", () => {
  it("shows loading spinner initially", () => {
    render(<ExampleList />);
    expect(
      screen.getByRole("status", { name: /loading/i }),
    ).toBeInTheDocument();
  });

  it("renders list of examples after loading", async () => {
    render(<ExampleList />);

    await waitFor(() => {
      expect(screen.getByText("First Example")).toBeInTheDocument();
    });
    expect(screen.getByText("Second Example")).toBeInTheDocument();
    expect(screen.getByText("This is the first example")).toBeInTheDocument();
    expect(screen.getByText("This is the second example")).toBeInTheDocument();
  });

  it("handles error state", async () => {
    server.use(
      http.get("http://localhost:3001/api/examples", () => {
        return HttpResponse.json(
          { message: "Internal Server Error" },
          { status: 500 },
        );
      }),
    );

    render(<ExampleList />);

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
    });
    expect(screen.getByText(/error/i)).toBeInTheDocument();
  });
});

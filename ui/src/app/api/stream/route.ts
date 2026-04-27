import { NextResponse } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function GET() {
  try {
    const response = await fetch(`${API_URL}/api/stream`, {
      headers: {
        "Accept": "text/event-stream",
        "Cache-Control": "no-cache",
      },
    });

    if (!response.ok) {
      return NextResponse.json({ error: "Failed to connect to stream" }, { status: response.status });
    }

    const stream = new ReadableStream({
      start(controller) {
        const reader = response.body!.getReader();
        function read() {
          reader
            .read()
            .then(({ done, value }) => {
              if (done) {
                controller.close();
                return;
              }
              controller.enqueue(value);
              read();
            })
            .catch((error) => {
              controller.error(error);
            });
        }
        read();
      },
    });

    return new Response(stream, {
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        "Connection": "keep-alive",
        "X-Accel-Buffering": "no",
      },
    });
  } catch {
    return NextResponse.json({ error: "Failed to fetch stream" }, { status: 500 });
  }
}

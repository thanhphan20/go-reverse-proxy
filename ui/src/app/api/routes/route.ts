import { checkBotId } from "botid/server";
import { NextRequest, NextResponse } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function GET() {
  try {
    const res = await fetch(`${API_URL}/api/routes`);
    const data = await res.json();
    return NextResponse.json(data);
  } catch {
    return NextResponse.json({ error: "Failed to fetch routes" }, { status: 500 });
  }
}

export async function POST(request: NextRequest) {
  const botResult = await checkBotId();

  if (botResult.isBot) {
    return NextResponse.json({ error: "Bot traffic is not allowed" }, { status: 403 });
  }

  try {
    const body = await request.json();
    const res = await fetch(`${API_URL}/api/routes`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      return NextResponse.json({ error: "Failed to add route" }, { status: res.status });
    }
    return NextResponse.json({ success: true });
  } catch {
    return NextResponse.json({ error: "Failed to add route" }, { status: 500 });
  }
}

export async function DELETE(request: NextRequest) {
  const botResult = await checkBotId();

  if (botResult.isBot) {
    return NextResponse.json({ error: "Bot traffic is not allowed" }, { status: 403 });
  }

  try {
    const { searchParams } = new URL(request.url);
    const path = searchParams.get("path");
    if (!path) {
      return NextResponse.json({ error: "Path is required" }, { status: 400 });
    }
    const res = await fetch(`${API_URL}/api/routes?path=${encodeURIComponent(path)}`, {
      method: "DELETE",
    });
    if (!res.ok) {
      return NextResponse.json({ error: "Failed to delete route" }, { status: res.status });
    }
    return NextResponse.json({ success: true });
  } catch {
    return NextResponse.json({ error: "Failed to delete route" }, { status: 500 });
  }
}

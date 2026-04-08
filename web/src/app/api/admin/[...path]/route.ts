import { type NextRequest } from "next/server";

const ADMIN_API_BASE = process.env.OPENCRAB_ADMIN_API_BASE ?? "http://127.0.0.1:8080";

async function proxy(request: NextRequest, params: { path: string[] }) {
  const upstreamUrl = new URL(`/api/admin/${params.path.join("/")}`, ADMIN_API_BASE);
  upstreamUrl.search = request.nextUrl.search;

  const init: RequestInit = {
    method: request.method,
    headers: {
      "Content-Type": request.headers.get("content-type") ?? "application/json"
    },
    cache: "no-store"
  };

  if (request.method !== "GET" && request.method !== "HEAD") {
    init.body = await request.text();
  }

  const response = await fetch(upstreamUrl, init);
  const body = await response.text();

  return new Response(body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("content-type") ?? "application/json; charset=utf-8"
    }
  });
}

export async function GET(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return proxy(request, await context.params);
}

export async function POST(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return proxy(request, await context.params);
}

export async function PUT(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return proxy(request, await context.params);
}

export async function DELETE(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return proxy(request, await context.params);
}

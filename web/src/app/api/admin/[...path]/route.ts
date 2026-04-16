import { type NextRequest } from "next/server";

const ADMIN_API_BASE = process.env.OPENCRAB_ADMIN_API_BASE ?? "http://127.0.0.1:8080";

async function proxy(request: NextRequest, params: { path: string[] }) {
  const upstreamUrl = new URL(`/api/admin/${params.path.join("/")}`, ADMIN_API_BASE);
  upstreamUrl.search = request.nextUrl.search;
  const headers = new Headers({
    "Content-Type": request.headers.get("content-type") ?? "application/json"
  });
  const cookie = request.headers.get("cookie");
  if (cookie) {
    headers.set("cookie", cookie);
  }
  const secondaryPassword = request.headers.get("x-opencrab-secondary-password");
  if (secondaryPassword) {
    headers.set("x-opencrab-secondary-password", secondaryPassword);
  }
  const forwardedProto = request.headers.get("x-forwarded-proto") ?? request.nextUrl.protocol.replace(":", "");
  if (forwardedProto) {
    headers.set("x-forwarded-proto", forwardedProto);
  }

  const init: RequestInit = {
    method: request.method,
    headers,
    cache: "no-store"
  };

  if (request.method !== "GET" && request.method !== "HEAD") {
    init.body = await request.text();
  }

  const response = await fetch(upstreamUrl, init);
  const body = await response.text();

  const responseHeaders = new Headers({
    "Content-Type": response.headers.get("content-type") ?? "application/json; charset=utf-8"
  });
  const setCookie = response.headers.get("set-cookie");
  if (setCookie) {
    responseHeaders.set("set-cookie", setCookie);
  }

  return new Response(body, {
    status: response.status,
    headers: responseHeaders
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

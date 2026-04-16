import { type NextRequest } from "next/server";

const ADMIN_API_BASE = process.env.OPENCRAB_ADMIN_API_BASE ?? "http://127.0.0.1:8080";

function buildUpstreamUrl(action: string) {
  return new URL(`/api/admin/auth/${action}`, ADMIN_API_BASE);
}

async function proxy(request: NextRequest, action: string) {
  const headers = new Headers();
  const contentType = request.headers.get("content-type");
  if (contentType) {
    headers.set("Content-Type", contentType);
  }
  const cookie = request.headers.get("cookie");
  if (cookie) {
    headers.set("cookie", cookie);
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

  const response = await fetch(buildUpstreamUrl(action), init);
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

export async function GET(request: NextRequest, context: { params: Promise<{ action: string }> }) {
  const { action } = await context.params;
  if (action !== "status") {
    return new Response("Not Found", { status: 404 });
  }
  return proxy(request, action);
}

export async function POST(request: NextRequest, context: { params: Promise<{ action: string }> }) {
  const { action } = await context.params;
  if (!["setup", "login", "logout"].includes(action)) {
    return new Response("Not Found", { status: 404 });
  }
  return proxy(request, action);
}

import { type NextRequest } from "next/server";

const OPENCRAB_API_BASE = process.env.OPENCRAB_ADMIN_API_BASE ?? "http://127.0.0.1:8080";

const FORWARDED_REQUEST_HEADERS = [
  "accept",
  "authorization",
  "content-type",
  "idempotency-key",
  "openai-beta",
  "prefer",
  "x-api-key",
  "x-goog-api-key",
  "x-opencrab-async",
  "x-session-id",
  "anthropic-beta",
  "anthropic-version",
  "anthropic-dangerous-direct-browser-access",
  "x-claude-code-session-id"
];

const FORWARDED_RESPONSE_HEADERS = [
  "cache-control",
  "content-type",
  "openai-model",
  "x-opencrab-channel",
  "x-opencrab-provider"
];

async function proxy(request: NextRequest, params: { path: string[] }) {
  const upstreamUrl = new URL(`/v1/${params.path.join("/")}`, OPENCRAB_API_BASE);
  upstreamUrl.search = request.nextUrl.search;

  const headers = new Headers();
  for (const key of FORWARDED_REQUEST_HEADERS) {
    const value = request.headers.get(key);
    if (value) {
      headers.set(key, value);
    }
  }

  const forwardedProto = request.headers.get("x-forwarded-proto") ?? request.nextUrl.protocol.replace(":", "");
  if (forwardedProto) {
    headers.set("x-forwarded-proto", forwardedProto);
  }

  const init: RequestInit = {
    method: request.method,
    headers,
    cache: "no-store",
    redirect: "manual"
  };

  if (request.method !== "GET" && request.method !== "HEAD") {
    init.body = await request.text();
  }

  const response = await fetch(upstreamUrl, init);
  const responseHeaders = new Headers();
  for (const key of FORWARDED_RESPONSE_HEADERS) {
    const value = response.headers.get(key);
    if (value) {
      responseHeaders.set(key, value);
    }
  }

  return new Response(response.body, {
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

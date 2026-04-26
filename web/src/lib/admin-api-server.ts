import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import type {
  AdminApiKey,
  AdminAuthStatus,
  AdminChannel,
  AdminDashboardSummary,
  AdminModel,
  AdminModelRoute,
  AdminRequestLogDetail,
  AdminRequestLogListResult,
  AdminSecondarySecurityState,
  AdminSettingGroup
} from "@/lib/admin-api";

const ADMIN_API_BASE = process.env.OPENCRAB_ADMIN_API_BASE ?? "http://127.0.0.1:8080";

type ListResponse<T> = {
  items: T[];
};

export function resolveAdminFetchFailure(status: number, message: string) {
  if (status === 401) {
    return { redirectTo: "/login", message };
  }
  if (status === 428) {
    return { redirectTo: "/init", message };
  }
  return { redirectTo: null, message };
}

async function adminFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const cookieStore = await cookies();
  const forwardedCookies = cookieStore
    .getAll()
    .map((item) => `${item.name}=${item.value}`)
    .join("; ");
  const headers = new Headers(init?.headers);

  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (forwardedCookies) {
    headers.set("Cookie", forwardedCookies);
  }

  const response = await fetch(`${ADMIN_API_BASE}${path}`, {
    ...init,
    cache: "no-store",
    headers
  });

  if (!response.ok) {
    const message = await response.text();
    const failure = resolveAdminFetchFailure(response.status, message || `请求失败: ${response.status}`);
    if (failure.redirectTo) {
      redirect(failure.redirectTo);
    }
    throw new Error(failure.message);
  }

  return response.json() as Promise<T>;
}

export async function getAdminChannels() {
  const response = await adminFetch<ListResponse<AdminChannel>>("/api/admin/channels");
  return response.items;
}

export async function getAdminApiKeys() {
  const response = await adminFetch<ListResponse<AdminApiKey>>("/api/admin/api-keys");
  return response.items;
}

export async function getAdminModels() {
  const response = await adminFetch<ListResponse<AdminModel>>("/api/admin/models");
  return response.items;
}

export async function getAdminModelRoutes() {
  const response = await adminFetch<ListResponse<AdminModelRoute>>("/api/admin/model-routes");
  return response.items;
}

export async function getAdminLogs(params?: { q?: string; category?: string }) {
  const searchParams = new URLSearchParams();
  if (params?.q?.trim()) {
    searchParams.set("q", params.q.trim());
  }
  if (params?.category?.trim() && params.category !== "all") {
    searchParams.set("category", params.category.trim());
  }
  const query = searchParams.toString();
  return adminFetch<AdminRequestLogListResult>(`/api/admin/logs${query ? `?${query}` : ""}`);
}

export async function getAdminLogDetail(id: number) {
  return adminFetch<AdminRequestLogDetail>(`/api/admin/logs/${id}`);
}

export async function getAdminSettings() {
  const response = await adminFetch<ListResponse<AdminSettingGroup>>("/api/admin/settings");
  return response.items;
}

export async function getAdminRoutingOverview() {
  return adminFetch<AdminDashboardSummary["routing_overview"]>("/api/admin/routing/overview");
}

export async function getAdminDashboardSummary() {
  return adminFetch<AdminDashboardSummary>("/api/admin/dashboard/summary");
}

export async function getAdminAuthStatus() {
  return adminFetch<AdminAuthStatus>("/api/admin/auth/status");
}

export async function getAdminSecondarySecurityState() {
  return adminFetch<AdminSecondarySecurityState>("/api/admin/auth/security");
}

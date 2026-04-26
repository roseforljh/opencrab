import { resolveAdminFetchFailure } from "./admin-api-server.ts";

function assert(condition: unknown, message: string) {
  if (!condition) {
    throw new Error(message);
  }
}

const unauthorized = resolveAdminFetchFailure(401, "未登录或登录已失效");
assert(unauthorized.redirectTo === "/login", "401 should redirect to /login");

const uninitialized = resolveAdminFetchFailure(428, "管理员密码尚未初始化");
assert(uninitialized.redirectTo === "/init", "428 should redirect to /init");

const generic = resolveAdminFetchFailure(500, "内部错误");
assert(generic.redirectTo === null, "500 should not redirect");
assert(generic.message === "内部错误", "500 should preserve upstream message");

console.log("admin-api-server tests passed");

import { redirect } from "next/navigation";

import { PasswordAuthForm } from "@/components/auth/password-auth-form";
import { getAdminAuthStatus } from "@/lib/admin-api-server";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export const dynamic = "force-dynamic";

export default async function LoginPage() {
  const authStatus = await getAdminAuthStatus();
  if (!authStatus.initialized) {
    redirect("/init");
  }
  if (authStatus.authenticated) {
    redirect("/");
  }

  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  return (
    <PasswordAuthForm
      mode="login"
      badge={t("auth.badge")}
      title={t("auth.login.title")}
      description={t("auth.login.description")}
      passwordLabel={t("auth.password")}
      confirmLabel={t("auth.password_confirm")}
      passwordHint={t("auth.password_hint")}
      mismatchMessage={t("auth.password_mismatch")}
      submitLabel={t("auth.login.submit")}
    />
  );
}

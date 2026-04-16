"use client";

import Image from "next/image";
import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type PasswordAuthFormProps = {
  mode: "setup" | "login";
  badge: string;
  title: string;
  description: string;
  passwordLabel: string;
  confirmLabel: string;
  passwordHint: string;
  mismatchMessage: string;
  submitLabel: string;
};

export function PasswordAuthForm({
  mode,
  badge,
  title,
  description,
  passwordLabel,
  confirmLabel,
  passwordHint,
  mismatchMessage,
  submitLabel
}: PasswordAuthFormProps) {
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (mode === "setup" && password !== confirmPassword) {
      setError(mismatchMessage);
      return;
    }

    try {
      setSubmitting(true);
      setError("");

      const response = await fetch(`/api/auth/${mode === "setup" ? "setup" : "login"}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ password })
      });

      if (!response.ok) {
        setError((await response.text()) || "请求失败");
        return;
      }

      window.location.href = "/";
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "请求失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-6 py-10 text-foreground">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.08),transparent_38%)] dark:bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.08),transparent_32%)]" />
      <div className="relative w-full max-w-5xl overflow-hidden rounded-[28px] border border-border/80 bg-background shadow-[0_24px_80px_rgba(0,0,0,0.18)]">
        <div className="grid lg:grid-cols-[1.05fr_0.95fr]">
          <section className="border-b border-border/60 px-8 py-10 lg:border-b-0 lg:border-r lg:px-10 lg:py-12">
            <Badge className="border-white/10 bg-white/5 text-foreground ring-white/10">{badge}</Badge>
            <div className="mt-8 flex items-center gap-4">
              <div className="relative flex h-14 w-14 items-center justify-center overflow-hidden rounded-2xl border border-border/70 bg-black text-white dark:bg-white dark:text-black">
                <Image src="/logo.png" alt="OpenCrab Logo" width={34} height={34} className="object-contain pixelated" />
              </div>
              <div>
                <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">OpenCrab</p>
                <h1 className="mt-2 text-3xl font-semibold tracking-tight">{title}</h1>
              </div>
            </div>
            <p className="mt-5 max-w-xl text-sm leading-7 text-muted-foreground">{description}</p>

            <div className="mt-10 grid gap-4 sm:grid-cols-2">
              <div className="rounded-2xl border border-border/70 bg-muted/20 p-5">
                <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">Security</p>
                <p className="mt-3 text-sm leading-6 text-foreground">单管理员、单密码、单控制台入口，适合个人部署环境。</p>
              </div>
              <div className="rounded-2xl border border-border/70 bg-muted/20 p-5">
                <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">Style</p>
                <p className="mt-3 text-sm leading-6 text-foreground">沿用现有高对比黑白壳层和紧凑控制台排版，不引入多余流程。</p>
              </div>
            </div>
          </section>

          <section className="px-8 py-10 lg:px-10 lg:py-12">
            <form className="space-y-5" onSubmit={handleSubmit}>
              <div>
                <label className="mb-2 block text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">{passwordLabel}</label>
                <Input
                  type="password"
                  autoFocus
                  autoComplete={mode === "setup" ? "new-password" : "current-password"}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder={passwordHint}
                />
              </div>

              {mode === "setup" ? (
                <div>
                  <label className="mb-2 block text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">{confirmLabel}</label>
                  <Input
                    type="password"
                    autoComplete="new-password"
                    value={confirmPassword}
                    onChange={(event) => setConfirmPassword(event.target.value)}
                    placeholder={passwordHint}
                  />
                </div>
              ) : null}

              <div className="rounded-2xl border border-border/70 bg-muted/20 px-4 py-3 text-sm leading-6 text-muted-foreground">{passwordHint}</div>

              {error ? <div className="rounded-2xl border border-danger/25 bg-danger/5 px-4 py-3 text-sm leading-6 text-danger">{error}</div> : null}

              <Button type="submit" size="lg" className="w-full" disabled={submitting}>
                {submitting ? `${submitLabel}...` : submitLabel}
              </Button>
            </form>
          </section>
        </div>
      </div>
    </main>
  );
}

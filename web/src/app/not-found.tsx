import Link from "next/link";

export default function NotFound() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-6 py-10 text-foreground">
      <div className="w-full max-w-md rounded-2xl border border-border bg-background p-8 text-center shadow-sm">
        <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">404</p>
        <h1 className="mt-3 text-2xl font-semibold tracking-tight">页面不存在</h1>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          当前访问的页面不存在，可能已经被移动，或者这个地址本身就是无效的。
        </p>
        <div className="mt-6">
          <Link
            href="/"
            className="inline-flex items-center justify-center rounded-lg border border-border bg-secondary px-4 py-2 text-sm font-medium text-foreground transition-[background-color,color,transform] duration-200 ease-[var(--ease-out-smooth)] hover:-translate-y-0.5 hover:bg-secondary/80"
          >
            返回首页
          </Link>
        </div>
      </div>
    </main>
  );
}

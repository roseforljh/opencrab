"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { FilterBar } from "@/components/shared/filter-bar";
import { SectionCard } from "@/components/shared/section-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { AdminSecondarySecurityState, AdminSettingGroup, AdminSettingItem } from "@/lib/admin-api";

const routingStrategyOptions = [
  { value: "sequential", label: "顺序" },
  { value: "round_robin", label: "轮询" }
];

const stickyEnabledOptions = [
  { value: "true", label: "启用" },
  { value: "false", label: "禁用" }
];

const stickyKeySourceOptions = [
  { value: "auto", label: "自动" },
  { value: "header", label: "Header" },
  { value: "metadata", label: "Metadata" }
];

const booleanSettingKeys = new Set([
  "gateway.sticky_enabled",
  "dispatch.redis_enabled",
  "dispatch.redis_tls_enabled",
  "dispatch.pause_dispatch",
  "dispatch.dead_letter_enabled",
  "dispatch.metrics_enabled",
  "dispatch.show_worker_status",
  "dispatch.show_queue_depth",
  "dispatch.show_retry_rate"
]);

const queueModeOptions = [
  { value: "single", label: "单队列" },
  { value: "priority", label: "优先级队列" }
];

const backoffModeOptions = [
  { value: "fixed", label: "固定退避" },
  { value: "exponential", label: "指数退避" }
];

const securityInputClassName = "bg-muted/30 dark:border-white/8 dark:bg-white/[0.03]";

export function SettingsClient({
  eyebrow,
  title,
  description,
  initialGroups,
  initialSecurityState
}: {
  eyebrow: string;
  title: string;
  description: string;
  initialGroups: AdminSettingGroup[];
  initialSecurityState: AdminSecondarySecurityState;
}) {
	const router = useRouter();
  const [groups, setGroups] = useState(initialGroups);
  const [savingKey, setSavingKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [securityState, setSecurityState] = useState(initialSecurityState);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordSaving, setPasswordSaving] = useState(false);
  const [passwordMessage, setPasswordMessage] = useState<string | null>(null);
  const [secondaryEnabled, setSecondaryEnabled] = useState(initialSecurityState.enabled ? "enabled" : "disabled");
  const [currentAdminPassword, setCurrentAdminPassword] = useState("");
  const [currentSecondaryPassword, setCurrentSecondaryPassword] = useState("");
  const [secondaryPassword, setSecondaryPassword] = useState("");
  const [secondaryConfirmPassword, setSecondaryConfirmPassword] = useState("");
  const [secondarySaving, setSecondarySaving] = useState(false);
  const [secondaryMessage, setSecondaryMessage] = useState<string | null>(null);
  const [searchValue, setSearchValue] = useState("");
  const [clearingLogs, setClearingLogs] = useState(false);

  const handleChange = (groupTitle: string, key: string, value: string) => {
    setGroups((current) =>
      current.map((group) =>
        group.title === groupTitle
          ? { ...group, items: group.items.map((item) => (item.key === key ? { ...item, value } : item)) }
          : group
      )
    );
  };

  const handleSave = async (key: string, value: string) => {
    const currentItem = groups.flatMap((group) => group.items).find((item) => item.key === key);
    if (currentItem?.sensitive && value.trim() === "") {
      return;
    }
    setError(null);
    setSavingKey(key);
    try {
      const response = await fetch("/api/admin/settings", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key, value })
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "设置保存失败");
    } finally {
      setSavingKey(null);
    }
  };

  const handleClearLogs = async () => {
    setError(null);
    setClearingLogs(true);
    try {
      const response = await fetch("/api/admin/logs", {
        method: "DELETE"
      });
      if (!response.ok) {
        throw new Error((await response.text()) || "清空日志失败");
      }
      router.refresh();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "清空日志失败");
    } finally {
      setClearingLogs(false);
    }
  };

  const renderField = (groupTitle: string, item: AdminSettingItem) => {
    if (item.key === "gateway.routing_strategy") {
      return (
        <Select value={item.value} onValueChange={(value) => handleChange(groupTitle, item.key, value)}>
          <SelectTrigger className="bg-muted/30">
            <SelectValue placeholder="选择路由策略" />
          </SelectTrigger>
          <SelectContent>
            {routingStrategyOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    }

    if (booleanSettingKeys.has(item.key)) {
      return (
        <Select value={item.value} onValueChange={(value) => handleChange(groupTitle, item.key, value)}>
          <SelectTrigger className="bg-muted/30">
            <SelectValue placeholder="选择状态" />
          </SelectTrigger>
          <SelectContent>
            {stickyEnabledOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    }

    if (item.key === "dispatch.queue_mode") {
      return (
        <Select value={item.value} onValueChange={(value) => handleChange(groupTitle, item.key, value)}>
          <SelectTrigger className="bg-muted/30">
            <SelectValue placeholder="选择队列模式" />
          </SelectTrigger>
          <SelectContent>
            {queueModeOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    }

    if (item.key === "dispatch.backoff_mode") {
      return (
        <Select value={item.value} onValueChange={(value) => handleChange(groupTitle, item.key, value)}>
          <SelectTrigger className="bg-muted/30">
            <SelectValue placeholder="选择退避模式" />
          </SelectTrigger>
          <SelectContent>
            {backoffModeOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    }

    if (item.key === "gateway.sticky_key_source") {
      return (
        <Select value={item.value} onValueChange={(value) => handleChange(groupTitle, item.key, value)}>
          <SelectTrigger className="bg-muted/30">
            <SelectValue placeholder="选择 sticky key 来源" />
          </SelectTrigger>
          <SelectContent>
            {stickyKeySourceOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    }

    if (item.sensitive) {
      return (
        <Input
          type="password"
          value={item.value}
          onChange={(event) => handleChange(groupTitle, item.key, event.target.value)}
          className="bg-muted/30"
          placeholder={item.configured ? "已配置，留空表示不修改" : "未配置"}
        />
      );
    }

    return <Input value={item.value} onChange={(event) => handleChange(groupTitle, item.key, event.target.value)} className="bg-muted/30" />;
  };

  const showSecondaryPasswordFields = secondaryEnabled === "enabled";
  const normalizedSearchValue = searchValue.trim().toLowerCase();
  const filteredGroups = groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => {
        if (!normalizedSearchValue) {
          return true;
        }

        return [group.title, item.label, item.description, item.key, item.value].some((field) =>
          field.toLowerCase().includes(normalizedSearchValue)
        );
      })
    }))
    .filter((group) => group.items.length > 0);

  const scrollToGroup = (groupTitle: string) => {
    const sections = Array.from(document.querySelectorAll("section"));
    const matchedSection = sections.find((section) => section.querySelector("h3")?.textContent === groupTitle);
    matchedSection?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  const handlePasswordChange = async () => {
    setPasswordMessage(null);
    setPasswordSaving(true);
    try {
      const response = await fetch("/api/admin/auth/password", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
          confirm_password: confirmPassword
        })
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      setPasswordMessage("主密码已更新，新会话已自动续期。")
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (requestError) {
      setPasswordMessage(requestError instanceof Error ? requestError.message : "主密码修改失败");
    } finally {
      setPasswordSaving(false);
    }
  };

  const handleSecondarySave = async () => {
    setSecondaryMessage(null);
    setSecondarySaving(true);
    try {
      const response = await fetch("/api/admin/auth/secondary", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          enabled: secondaryEnabled === "enabled",
          current_admin_password: currentAdminPassword,
          current_secondary_password: currentSecondaryPassword,
          new_password: secondaryPassword,
          confirm_password: secondaryConfirmPassword
        })
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      const updated = (await response.json()) as AdminSecondarySecurityState;
      setSecurityState(updated);
      setSecondaryEnabled(updated.enabled ? "enabled" : "disabled");
      setSecondaryMessage(updated.enabled ? "二级密码配置已更新。" : "二级密码已关闭。")
      setCurrentAdminPassword("");
      setCurrentSecondaryPassword("");
      setSecondaryPassword("");
      setSecondaryConfirmPassword("");
    } catch (requestError) {
      setSecondaryMessage(requestError instanceof Error ? requestError.message : "二级密码配置失败");
    } finally {
      setSecondarySaving(false);
    }
  };

  return (
    <PageContainer>
      <PageHeader eyebrow={eyebrow} title={title} description={description} />

      <FilterBar
        placeholder="搜索设置项..."
        searchValue={searchValue}
        onSearchValueChange={setSearchValue}
        showChipIcon={false}
        enableActiveStyle={false}
        chips={groups.map((group) => ({
          label: group.title,
          onClick: () => scrollToGroup(group.title)
        }))}
      />

      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}

      <div className="space-y-8">
        <SectionCard title="认证与安全" description="管理管理员主密码和敏感操作所需的二级密码。">
          <div className="grid gap-6 xl:grid-cols-2">
            <div className="rounded-2xl border border-border bg-card p-5">
              <div className="mb-4">
                <div className="text-sm font-medium text-foreground">修改管理员密码</div>
                <div className="mt-1 text-sm text-muted-foreground">修改主密码前必须先输入当前密码，新密码需要输入两次并保持一致。</div>
              </div>
              <div className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">当前密码</label>
                  <Input type="password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} className={securityInputClassName} />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">新密码</label>
                  <Input type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} className={securityInputClassName} />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">确认新密码</label>
                  <Input type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} className={securityInputClassName} />
                </div>
                {passwordMessage ? <div className="rounded-xl border border-border bg-muted/30 px-3 py-2 text-xs text-foreground">{passwordMessage}</div> : null}
                <div className="flex justify-end">
                  <Button onClick={() => void handlePasswordChange()} disabled={passwordSaving}>{passwordSaving ? "保存中..." : "更新主密码"}</Button>
                </div>
              </div>
            </div>

            <div className="rounded-2xl border border-border bg-card p-5">
              <div className="mb-4">
                <div className="flex items-center justify-between gap-3">
                  <div className="text-sm font-medium text-foreground">二级密码</div>
                  <span className={`rounded-full px-2.5 py-1 text-xs font-medium ring-1 ${securityState.enabled ? "bg-success/10 text-success ring-success/20" : "bg-muted text-muted-foreground ring-border"}`}>{securityState.enabled ? "已开启" : "已关闭"}</span>
                </div>
                <div className="mt-1 text-sm text-muted-foreground">用于创建或删除 API Key 的二次校验。首次开启必须设置，设置后可正常关闭和重新开启。</div>
              </div>
              <div className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">开关状态</label>
                  <Select value={secondaryEnabled} onValueChange={setSecondaryEnabled}>
                    <SelectTrigger className="bg-muted/30">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="enabled">开启</SelectItem>
                      <SelectItem value="disabled">关闭</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">当前管理员密码</label>
                  <Input type="password" value={currentAdminPassword} onChange={(event) => setCurrentAdminPassword(event.target.value)} className={securityInputClassName} />
                </div>
                {securityState.configured && showSecondaryPasswordFields ? (
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground">当前二级密码</label>
                    <Input
                      type="password"
                      value={currentSecondaryPassword}
                      onChange={(event) => setCurrentSecondaryPassword(event.target.value)}
                      placeholder="修改二级密码时需要输入，单独重新开启可留空"
                      className={securityInputClassName}
                    />
                  </div>
                ) : null}
                {showSecondaryPasswordFields ? (
                  <>
                    <div className="space-y-2">
                      <label className="text-sm font-medium text-foreground">新二级密码</label>
                      <Input
                        type="password"
                        value={secondaryPassword}
                        onChange={(event) => setSecondaryPassword(event.target.value)}
                        placeholder={securityState.configured ? "留空表示仅重新开启，不修改二级密码" : "首次开启时必须设置"}
                        className={securityInputClassName}
                      />
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium text-foreground">确认新二级密码</label>
                      <Input
                        type="password"
                        value={secondaryConfirmPassword}
                        onChange={(event) => setSecondaryConfirmPassword(event.target.value)}
                        className={securityInputClassName}
                      />
                    </div>
                  </>
                ) : null}
                {secondaryMessage ? <div className="rounded-xl border border-border bg-muted/30 px-3 py-2 text-xs text-foreground">{secondaryMessage}</div> : null}
                <div className="flex justify-end">
                  <Button onClick={() => void handleSecondarySave()} disabled={secondarySaving}>{secondarySaving ? "保存中..." : "保存二级密码设置"}</Button>
                </div>
              </div>
            </div>
          </div>
        </SectionCard>

        {filteredGroups.map((group) => (
          <SectionCard key={group.title} title={group.title} description={`管理${group.title}相关的配置项。`}>
            <div className="divide-y divide-border rounded-xl border border-border bg-card">
              {group.items.map((item) => (
                <div key={item.key} className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="space-y-1">
                    <div className="text-sm font-medium text-foreground">{item.label}</div>
                    <div className="text-sm text-muted-foreground">{item.description}</div>
                  </div>
                  <div className="flex items-center gap-3 sm:w-72">
                    {renderField(group.title, item)}
                    <Button variant="outline" className="shrink-0" onClick={() => void handleSave(item.key, item.value)}>
                      {savingKey === item.key ? "保存中..." : "保存"}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </SectionCard>
        ))}

        {filteredGroups.length === 0 ? (
          <SectionCard title="未找到匹配配置" description="试试更换关键字，或点击上方筛选标签切换分组。">
            <div className="rounded-xl border border-dashed border-border bg-muted/20 px-4 py-6 text-sm text-muted-foreground">
              当前没有匹配的设置项。
            </div>
          </SectionCard>
        ) : null}

        <SectionCard title="危险操作区" description="这些操作可能会导致数据丢失或服务中断，请谨慎操作。" className="border-danger/20">
          <div className="divide-y divide-danger/10 rounded-xl border border-danger/20 bg-danger/5">
            <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2 text-sm font-medium text-danger">
                  <AlertTriangle className="h-4 w-4" />
                  清空系统日志
                </div>
                <div className="text-sm text-danger/80">永久删除所有请求日志和异常记录，此操作不可恢复。</div>
              </div>
              <ConfirmDialog
                trigger={<Button variant="danger" className="shrink-0" disabled={clearingLogs}>{clearingLogs ? "清空中..." : "清空日志"}</Button>}
                title="确认清空系统日志"
                description="该操作会删除当前所有请求日志和异常记录，只建议在测试环境或明确需要时执行。"
                confirmLabel="确认清空"
                onConfirm={handleClearLogs}
              />
            </div>
            <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2 text-sm font-medium text-danger">
                  <AlertTriangle className="h-4 w-4" />
                  重置所有配置
                </div>
                <div className="text-sm text-danger/80">将所有系统设置恢复为默认值，不影响渠道和模型数据。</div>
              </div>
              <ConfirmDialog
                trigger={<Button variant="danger" className="shrink-0">重置配置</Button>}
                title="确认重置系统配置"
                description="该操作会把系统设置恢复到默认值。高风险操作必须经过二次确认。"
                confirmLabel="确认重置"
              />
            </div>
          </div>
        </SectionCard>
      </div>
    </PageContainer>
  );
}

"use client";

import { useState } from "react";

import { EditChannelForm } from "@/app/(console)/channels/edit-channel-form";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { Button } from "@/components/ui/button";

type ChannelRow = {
  name: string;
  provider: string;
  endpoint: string;
};

export function EditChannelDrawer({ row }: { row: ChannelRow }) {
  const [open, setOpen] = useState(false);

  return (
    <DetailDrawer
      title={`编辑渠道 · ${row.name}`}
      description="修改渠道名称、兼容类型、请求地址和默认密钥配置。"
      triggerLabel="编辑配置"
      trigger={<Button>编辑配置</Button>}
      open={open}
      onOpenChange={setOpen}
    >
      <EditChannelForm row={row} onCancel={() => setOpen(false)} onSave={() => setOpen(false)} />
    </DetailDrawer>
  );
}

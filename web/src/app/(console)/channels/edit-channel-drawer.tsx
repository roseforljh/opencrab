"use client";

import { useState } from "react";

import { EditChannelForm } from "@/app/(console)/channels/edit-channel-form";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { Button } from "@/components/ui/button";

type ChannelRow = {
	id: number;
	name: string;
	provider: string;
	endpoint: string;
	status: string;
	modelIds: string[];
	rpmLimit: number;
	maxInflight: number;
	safetyFactor: number;
	enabledForAsync: boolean;
	dispatchWeight: number;
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
		<EditChannelForm row={row} onCancel={() => setOpen(false)} onSave={() => window.location.reload()} />
	</DetailDrawer>
	);
}

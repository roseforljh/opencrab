/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Card, Chat, Typography, Button, Tag } from '@douyinfe/semi-ui';
import { MessageSquare, Eye, EyeOff, Sparkles } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import CustomInputRender from './CustomInputRender';

const ChatArea = ({
  chatRef,
  message,
  inputs,
  styleState,
  showDebugPanel,
  roleInfo,
  onMessageSend,
  onMessageCopy,
  onMessageReset,
  onMessageDelete,
  onStopGenerator,
  onClearMessages,
  onToggleDebugPanel,
  renderCustomChatContent,
  renderChatBoxAction,
}) => {
  const { t } = useTranslation();

  const renderInputArea = React.useCallback((props) => {
    return <CustomInputRender {...props} />;
  }, []);

  return (
    <Card
      className='h-full border-0 shadow-none'
      bordered={false}
      bodyStyle={{
        padding: 0,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        background: 'transparent',
      }}
    >
      <div className='border-b border-semi-color-border bg-semi-color-bg-1 px-5 py-4 md:px-6'>
        <div className='flex items-start justify-between gap-4'>
          <div className='min-w-0'>
            <div className='flex items-center gap-2'>
              <div className='rounded-xl bg-semi-color-fill-0 p-2 text-semi-color-primary'>
                <MessageSquare size={18} />
              </div>
              <div>
                <Typography.Title heading={5} className='!mb-0'>
                  {t('请求工作区')}
                </Typography.Title>
                <Typography.Text className='text-sm !text-semi-color-text-2'>
                  {t('在这里编排消息、发送请求并查看模型响应。')}
                </Typography.Text>
              </div>
            </div>
            <div className='mt-3 flex flex-wrap gap-2'>
              <Tag shape='circle' color='grey' size='small'>
                {inputs.group || t('未选择分组')}
              </Tag>
              <Tag shape='circle' color='blue' size='small'>
                {inputs.model || t('未选择模型')}
              </Tag>
              {inputs.stream && (
                <Tag shape='circle' color='green' size='small'>
                  {t('流式输出')}
                </Tag>
              )}
              {inputs.imageEnabled && (
                <Tag shape='circle' color='orange' size='small'>
                  {t('图片输入已启用')}
                </Tag>
              )}
            </div>
          </div>
          {!styleState.isMobile && (
            <Button
              icon={showDebugPanel ? <EyeOff size={14} /> : <Eye size={14} />}
              onClick={onToggleDebugPanel}
              theme='borderless'
              type='tertiary'
              className='!rounded-lg !border !border-semi-color-border !bg-transparent'
            >
              {showDebugPanel ? t('隐藏调试') : t('显示调试')}
            </Button>
          )}
        </div>
      </div>

      <div className='flex-1 overflow-hidden'>
        <Chat
          ref={chatRef}
          chatBoxRenderConfig={{
            renderChatBoxContent: renderCustomChatContent,
            renderChatBoxAction: renderChatBoxAction,
            renderChatBoxTitle: () => null,
          }}
          renderInputArea={renderInputArea}
          roleConfig={roleInfo}
          style={{
            height: '100%',
            maxWidth: '100%',
            overflow: 'hidden',
          }}
          chats={message}
          onMessageSend={onMessageSend}
          onMessageCopy={onMessageCopy}
          onMessageReset={onMessageReset}
          onMessageDelete={onMessageDelete}
          showClearContext
          showStopGenerate
          onStopGenerator={onStopGenerator}
          onClear={onClearMessages}
          className='h-full'
          placeholder={t('输入你的调试问题、提示词或测试请求...')}
        />
      </div>

      {styleState.isMobile && (
        <div className='border-t border-semi-color-border bg-semi-color-bg-1 px-4 py-3'>
          <Button
            icon={
              showDebugPanel ? <EyeOff size={14} /> : <Sparkles size={14} />
            }
            onClick={onToggleDebugPanel}
            type='primary'
            className='!w-full !rounded-lg'
          >
            {showDebugPanel ? t('关闭调试面板') : t('打开调试面板')}
          </Button>
        </div>
      )}
    </Card>
  );
};

export default ChatArea;

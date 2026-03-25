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

import React, { useEffect, useRef } from 'react';
import { Typography } from '@douyinfe/semi-ui';
import MarkdownRenderer from '../common/markdown/MarkdownRenderer';
import { ChevronRight, ChevronUp, Brain, Loader2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

const ThinkingContent = ({
  message,
  finalExtractedThinkingContent,
  thinkingSource,
  styleState,
  onToggleReasoningExpansion,
}) => {
  const { t } = useTranslation();
  const scrollRef = useRef(null);
  const lastContentRef = useRef('');

  const isThinkingStatus =
    message.status === 'loading' || message.status === 'incomplete';
  const headerText =
    isThinkingStatus && !message.isThinkingComplete
      ? t('思考中...')
      : t('思考过程');

  useEffect(() => {
    if (
      scrollRef.current &&
      finalExtractedThinkingContent &&
      message.isReasoningExpanded
    ) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [finalExtractedThinkingContent, message.isReasoningExpanded]);

  useEffect(() => {
    if (!isThinkingStatus) {
      lastContentRef.current = '';
    }
  }, [isThinkingStatus]);

  if (!finalExtractedThinkingContent) return null;

  let prevLength = 0;
  if (isThinkingStatus && lastContentRef.current) {
    if (finalExtractedThinkingContent.startsWith(lastContentRef.current)) {
      prevLength = lastContentRef.current.length;
    }
  }

  if (isThinkingStatus) {
    lastContentRef.current = finalExtractedThinkingContent;
  }

  return (
    <div className='rounded-xl sm:rounded-2xl mb-2 sm:mb-4 overflow-hidden shadow-sm border border-gray-200 dark:border-[#222] bg-white dark:bg-[#0a0a0a]'>
      <div
        className='flex items-center justify-between p-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-[#111] transition-all'
        onClick={() => onToggleReasoningExpansion(message.id)}
      >
        <div className='flex items-center gap-2 sm:gap-4 relative'>
          <div className='w-6 h-6 sm:w-8 sm:h-8 rounded-full bg-gray-100 dark:bg-[#1a1a1a] flex items-center justify-center border border-gray-200 dark:border-[#333]'>
            <Brain
              className='text-gray-700 dark:text-gray-300'
              size={styleState.isMobile ? 12 : 16}
            />
          </div>
          <div className='flex flex-col'>
            <Typography.Text
              strong
              className='text-sm sm:text-base text-gray-900 dark:text-gray-100'
            >
              {headerText}
            </Typography.Text>
            {thinkingSource && (
              <Typography.Text
                className='text-xs mt-0.5 text-gray-500 dark:text-gray-400 hidden sm:block'
              >
                来源: {thinkingSource}
              </Typography.Text>
            )}
          </div>
        </div>
        <div className='flex items-center gap-2 sm:gap-3 relative'>
          {isThinkingStatus && !message.isThinkingComplete && (
            <div className='flex items-center gap-1 sm:gap-2'>
              <Loader2
                className='animate-spin text-gray-600 dark:text-gray-400'
                size={styleState.isMobile ? 14 : 18}
              />
              <Typography.Text
                className='text-xs sm:text-sm font-medium text-gray-600 dark:text-gray-400'
              >
                思考中
              </Typography.Text>
            </div>
          )}
          {(!isThinkingStatus || message.isThinkingComplete) && (
            <div className='w-5 h-5 sm:w-6 sm:h-6 rounded-full bg-gray-100 dark:bg-[#1a1a1a] flex items-center justify-center border border-gray-200 dark:border-[#333]'>
              {message.isReasoningExpanded ? (
                <ChevronUp
                  size={styleState.isMobile ? 12 : 16}
                  className='text-gray-600 dark:text-gray-400'
                />
              ) : (
                <ChevronRight
                  size={styleState.isMobile ? 12 : 16}
                  className='text-gray-600 dark:text-gray-400'
                />
              )}
            </div>
          )}
        </div>
      </div>
      <div
        className={`transition-all duration-500 ease-out ${
          message.isReasoningExpanded
            ? 'max-h-96 opacity-100'
            : 'max-h-0 opacity-0'
        } overflow-hidden bg-gray-50 dark:bg-[#111] border-t border-gray-200 dark:border-[#222]`}
      >
        {message.isReasoningExpanded && (
          <div className='p-3 sm:p-5 pt-2 sm:pt-4'>
            <div
              ref={scrollRef}
              className='bg-white dark:bg-[#000] rounded-lg sm:rounded-xl p-3 shadow-inner overflow-x-auto overflow-y-auto thinking-content-scroll border border-gray-200 dark:border-[#333]'
              style={{
                maxHeight: '200px',
                scrollbarWidth: 'thin',
                scrollbarColor: 'rgba(128, 128, 128, 0.3) transparent',
              }}
            >
              <div className='prose prose-xs sm:prose-sm max-w-none text-xs sm:text-sm dark:prose-invert'>
                <MarkdownRenderer
                  content={finalExtractedThinkingContent}
                  className=''
                  animated={isThinkingStatus}
                  previousContentLength={prevLength}
                />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ThinkingContent;


import React, { useEffect, useRef, useState } from 'react';
import { Notification, Button, Space, Toast, Select } from '@douyinfe/semi-ui';
import { API, showError, getModelCategories, selectFilter } from '../../../helpers';
import CardPro from '../../common/ui/CardPro';
import TokensTable from './TokensTable';
import TokensActions from './TokensActions';
import TokensFilters from './TokensFilters';
import TokensDescription from './TokensDescription';
import EditTokenModal from './modals/EditTokenModal';
import CCSwitchModal from './modals/CCSwitchModal';
import { useTokensData } from '../../../hooks/tokens/useTokensData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

function TokensPage() {
  const openFluentNotificationRef = useRef(null);
  const openCCSwitchModalRef = useRef(null);
  const tokensData = useTokensData(
    (key) => openFluentNotificationRef.current?.(key),
    (key) => openCCSwitchModalRef.current?.(key),
  );
  const isMobile = useIsMobile();
  const latestRef = useRef({
    tokens: [],
    selectedKeys: [],
    t: (k) => k,
    selectedModel: '',
    prefillKey: '',
    fetchTokenKey: async () => '',
  });
  const [modelOptions, setModelOptions] = useState([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [prefillKey, setPrefillKey] = useState('');
  const [ccSwitchVisible, setCCSwitchVisible] = useState(false);
  const [ccSwitchKey, setCCSwitchKey] = useState('');

  useEffect(() => {
    latestRef.current = {
      tokens: tokensData.tokens,
      selectedKeys: tokensData.selectedKeys,
      t: tokensData.t,
      selectedModel,
      prefillKey,
      fetchTokenKey: tokensData.fetchTokenKey,
    };
  }, [
    tokensData.tokens,
    tokensData.selectedKeys,
    tokensData.t,
    selectedModel,
    prefillKey,
    tokensData.fetchTokenKey,
  ]);

  const loadModels = async () => {
    try {
      const res = await API.get('/api/user/models');
      const { success, message, data } = res.data || {};
      if (success) {
        const categories = getModelCategories(tokensData.t);
        const options = (data || []).map((model) => {
          let icon = null;
          for (const [key, category] of Object.entries(categories)) {
            if (key !== 'all' && category.filter({ model_name: model })) {
              icon = category.icon;
              break;
            }
          }
          return {
            label: (
              <span className='flex items-center gap-1'>
                {icon}
                {model}
              </span>
            ),
            value: model,
          };
        });
        setModelOptions(options);
      } else {
        showError(tokensData.t(message));
      }
    } catch (e) {
      showError(e.message || 'Failed to load models');
    }
  };

  function openFluentNotification(key) {
    const { t } = latestRef.current;
    const SUPPRESS_KEY = 'fluent_notify_suppressed';
    if (modelOptions.length === 0) {
      loadModels();
    }
    if (!key && localStorage.getItem(SUPPRESS_KEY) === '1') return;
    const container = document.getElementById('fluent-opencrab-container');
    if (!container) {
      Toast.warning(t('未检测到 FluentRead（流畅阅读），请确认扩展已启用'));
      return;
    }
    setPrefillKey(key || '');
    Notification.info({
      id: 'fluent-detected',
      title: t('检测到 FluentRead（流畅阅读）'),
      content: (
        <div>
          <div style={{ marginBottom: 8 }}>
            {key
              ? t('请选择模型。')
              : t('选择模型后可一键填充当前选中令牌（或本页第一个令牌）。')}
          </div>
          <div style={{ marginBottom: 8 }}>
            <Select
              placeholder={t('请选择模型')}
              optionList={modelOptions}
              onChange={setSelectedModel}
              filter={selectFilter}
              style={{ width: 320 }}
              showClear
              searchable
              emptyContent={t('暂无数据')}
            />
          </div>
          <Space>
            <Button theme='solid' type='primary' onClick={handlePrefillToFluent}>
              {t('一键填充到 FluentRead')}
            </Button>
            {!key && (
              <Button
                type='warning'
                onClick={() => {
                  localStorage.setItem(SUPPRESS_KEY, '1');
                  Notification.close('fluent-detected');
                  Toast.info(t('已关闭后续提醒'));
                }}
              >
                {t('不再提醒')}
              </Button>
            )}
            <Button type='tertiary' onClick={() => Notification.close('fluent-detected')}>
              {t('关闭')}
            </Button>
          </Space>
        </div>
      ),
      duration: 0,
    });
  }
  openFluentNotificationRef.current = openFluentNotification;

  function openCCSwitchModal(key) {
    if (modelOptions.length === 0) {
      loadModels();
    }
    setCCSwitchKey(key || '');
    setCCSwitchVisible(true);
  }
  openCCSwitchModalRef.current = openCCSwitchModal;

  const handlePrefillToFluent = async () => {
    const {
      tokens,
      selectedKeys,
      t,
      selectedModel: chosenModel,
      prefillKey: overrideKey,
      fetchTokenKey,
    } = latestRef.current;
    const container = document.getElementById('fluent-opencrab-container');
    if (!container) {
      Toast.error(t('未检测到 Fluent 容器'));
      return;
    }

    if (!chosenModel) {
      Toast.warning(t('请选择模型'));
      return;
    }

    let fullKey = overrideKey;
    if (!fullKey) {
      let target = null;
      if (selectedKeys.length > 0) {
        target = selectedKeys[0];
      }
      if (!target && tokens.length > 0) {
        target = tokens[0];
      }
      if (!target) {
        Toast.warning(t('未找到可用令牌'));
        return;
      }
      fullKey = await fetchTokenKey(target);
    }

    const payload = {
      serverUrl: window.location.origin,
      apiKey: `sk-${fullKey}`,
      model: chosenModel,
    };

    container.dispatchEvent(new CustomEvent('opencrab:prefill', { detail: payload }));
    Toast.success(t('已发送到 FluentRead'));
    Notification.close('fluent-detected');
  };

  useEffect(() => {
    loadModels();
  }, []);

  return (
    <>
      <EditTokenModal
        visible={tokensData.showEdit}
        visiable={tokensData.showEdit}
        handleClose={tokensData.closeEdit}
        editingToken={tokensData.editingToken}
        refresh={tokensData.refresh}
      />
      <CCSwitchModal
        visible={ccSwitchVisible}
        onClose={() => setCCSwitchVisible(false)}
        tokenKey={ccSwitchKey}
        modelOptions={modelOptions}
      />
      <CardPro
        descriptionArea={
          <TokensDescription
            compactMode={tokensData.compactMode}
            setCompactMode={tokensData.setCompactMode}
            t={tokensData.t}
          />
        }
        actionsArea={
          <TokensActions
            selectedKeys={tokensData.selectedKeys}
            setShowEdit={tokensData.setShowEdit}
            setEditingToken={tokensData.setEditingToken}
            batchCopyTokens={tokensData.batchCopyTokens}
            batchDeleteTokens={tokensData.batchDeleteTokens}
            t={tokensData.t}
          />
        }
        searchArea={
          <TokensFilters
            formInitValues={tokensData.formInitValues}
            setFormApi={tokensData.setFormApi}
            searchTokens={tokensData.searchTokens}
            loading={tokensData.loading}
            searching={tokensData.searching}
            t={tokensData.t}
          />
        }
        paginationArea={createCardProPagination({
          currentPage: tokensData.activePage,
          pageSize: tokensData.pageSize,
          total: tokensData.tokenCount,
          onPageChange: tokensData.handlePageChange,
          onPageSizeChange: tokensData.handlePageSizeChange,
          isMobile,
        })}
      >
        <TokensTable {...tokensData} />
      </CardPro>
    </>
  );
}

export default TokensPage;

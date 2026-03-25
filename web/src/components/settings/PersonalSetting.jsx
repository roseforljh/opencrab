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

import React, { useContext, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  API,
  copy,
  showError,
  showSuccess,
  setStatusData,
  setUserData,
} from '../../helpers';
import { UserContext } from '../../context/User';
import { useTranslation } from 'react-i18next';

// 导入子组件
import UserInfoHeader from './personal/components/UserInfoHeader';
import AccountManagement from './personal/cards/AccountManagement';
import NotificationSettings from './personal/cards/NotificationSettings';
import AccountDeleteModal from './personal/modals/AccountDeleteModal';
import ChangePasswordModal from './personal/modals/ChangePasswordModal';
import { Modal, Form } from '@douyinfe/semi-ui';
const PersonalSetting = () => {
  const [userState, userDispatch] = useContext(UserContext);
  let navigate = useNavigate();
  const { t } = useTranslation();

  const [inputs, setInputs] = useState({
    self_account_deletion_confirmation: '',
    original_password: '',
    set_new_password: '',
    set_new_password_confirmation: '',
  });
  const [status, setStatus] = useState({});
  const [showChangePasswordModal, setShowChangePasswordModal] = useState(false);
  const [showAccountDeleteModal, setShowAccountDeleteModal] = useState(false);
  const [showPinModal, setShowPinModal] = useState(false);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [systemToken, setSystemToken] = useState('');
  const [notificationSettings, setNotificationSettings] = useState({
    warningType: 'email',
    warningThreshold: 100000,
    webhookUrl: '',
    webhookSecret: '',
    notificationEmail: '',
    barkUrl: '',
    gotifyUrl: '',
    gotifyToken: '',
    gotifyPriority: 5,
    upstreamModelUpdateNotifyEnabled: false,
    recordIpLog: false,
  });
  const [pinInputs, setPinInputs] = useState({
    currentPin: '',
    pin: '',
    confirmPin: '',
  });

  useEffect(() => {
    let saved = localStorage.getItem('status');
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        setStatus(parsed);
        if (parsed.turnstile_check) {
          setTurnstileEnabled(true);
          setTurnstileSiteKey(parsed.turnstile_site_key);
        } else {
          setTurnstileEnabled(false);
          setTurnstileSiteKey('');
        }
      } catch {
        setStatus({});
        setTurnstileEnabled(false);
        setTurnstileSiteKey('');
      }
    }
    // Always refresh status from server to avoid stale flags (e.g., admin just enabled OAuth)
    (async () => {
      try {
        const res = await API.get('/api/status');
        const { success, data } = res.data;
        if (success && data) {
          setStatus(data);
          setStatusData(data);
          if (data.turnstile_check) {
            setTurnstileEnabled(true);
            setTurnstileSiteKey(data.turnstile_site_key);
          } else {
            setTurnstileEnabled(false);
            setTurnstileSiteKey('');
          }
        }
      } catch (e) {
        // ignore and keep local status
      }
    })();

    getUserData();
  }, []);

  useEffect(() => {
    if (userState?.user?.setting) {
      try {
        const settings = JSON.parse(userState.user.setting);
        setNotificationSettings({
          warningType: settings.notify_type || 'email',
          warningThreshold: settings.quota_warning_threshold || 500000,
          webhookUrl: settings.webhook_url || '',
          webhookSecret: settings.webhook_secret || '',
          notificationEmail: settings.notification_email || '',
          barkUrl: settings.bark_url || '',
          gotifyUrl: settings.gotify_url || '',
          gotifyToken: settings.gotify_token || '',
          gotifyPriority:
            settings.gotify_priority !== undefined
              ? settings.gotify_priority
              : 5,
          upstreamModelUpdateNotifyEnabled:
            settings.upstream_model_update_notify_enabled === true,
          recordIpLog: settings.record_ip_log || false,
        });
      } catch {
        setNotificationSettings({
          warningType: 'email',
          warningThreshold: 100000,
          webhookUrl: '',
          webhookSecret: '',
          notificationEmail: '',
          barkUrl: '',
          gotifyUrl: '',
          gotifyToken: '',
          gotifyPriority: 5,
          upstreamModelUpdateNotifyEnabled: false,
          recordIpLog: false,
        });
      }
    }
  }, [userState?.user?.setting]);

  const handleInputChange = (name, value) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const handlePinInputChange = (name, value) => {
    setPinInputs((prev) => ({ ...prev, [name]: value }));
  };

  const handleUpdatePin = async () => {
    if (!pinInputs.pin || pinInputs.pin !== pinInputs.confirmPin) {
      showError(t('PIN 输入不完整或两次输入不一致'));
      return;
    }
    try {
      const res = await API.put('/api/user/pin', {
        current_pin: pinInputs.currentPin,
        pin: pinInputs.pin,
        confirm_pin: pinInputs.confirmPin,
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('PIN 更新成功'));
        setShowPinModal(false);
        setPinInputs({ currentPin: '', pin: '', confirmPin: '' });
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('PIN 更新失败'));
    }
  };

  const generateAccessToken = async () => {
    const res = await API.get('/api/user/token');
    const { success, message, data } = res.data;
    if (success) {
      setSystemToken(data);
      await copy(data);
      showSuccess(t('令牌已重置并已复制到剪贴板'));
    } else {
      showError(message);
    }
  };

  const getUserData = async () => {
    let res = await API.get(`/api/user/self`);
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
      setUserData(data);
    } else {
      showError(message);
    }
  };

  const handleSystemTokenClick = async (e) => {
    e.target.select();
    await copy(e.target.value);
    showSuccess(t('系统令牌已复制到剪切板'));
  };

  const deleteAccount = async () => {
    if (inputs.self_account_deletion_confirmation !== userState.user.username) {
      showError(t('请输入你的账户名以确认删除！'));
      return;
    }

    const res = await API.delete('/api/user/self');
    const { success, message } = res.data;

    if (success) {
      showSuccess(t('账户已删除！'));
      await API.get('/api/user/logout');
      userDispatch({ type: 'logout' });
      localStorage.removeItem('user');
      navigate('/login');
    } else {
      showError(message);
    }
  };

  const changePassword = async () => {
    // if (inputs.original_password === '') {
    //   showError(t('请输入原密码！'));
    //   return;
    // }
    if (inputs.set_new_password === '') {
      showError(t('请输入新密码！'));
      return;
    }
    if (inputs.original_password === inputs.set_new_password) {
      showError(t('新密码需要和原密码不一致！'));
      return;
    }
    if (inputs.set_new_password !== inputs.set_new_password_confirmation) {
      showError(t('两次输入的密码不一致！'));
      return;
    }
    const res = await API.put(`/api/user/self`, {
      original_password: inputs.original_password,
      password: inputs.set_new_password,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('密码修改成功！'));
    } else {
      showError(message);
    }
    setShowChangePasswordModal(false);
  };

  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制：') + text);
    } else {
      // setSearchKeyword(text);
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  const handleNotificationSettingChange = (type, value) => {
    setNotificationSettings((prev) => ({
      ...prev,
      [type]: value.target
        ? value.target.value !== undefined
          ? value.target.value
          : value.target.checked
        : value, // handle checkbox properly
    }));
  };

  const saveNotificationSettings = async () => {
    try {
      const res = await API.put('/api/user/setting', {
        notify_type: notificationSettings.warningType,
        quota_warning_threshold: parseFloat(
          notificationSettings.warningThreshold,
        ),
        webhook_url: notificationSettings.webhookUrl,
        webhook_secret: notificationSettings.webhookSecret,
        notification_email: notificationSettings.notificationEmail,
        bark_url: notificationSettings.barkUrl,
        gotify_url: notificationSettings.gotifyUrl,
        gotify_token: notificationSettings.gotifyToken,
        gotify_priority: (() => {
          const parsed = parseInt(notificationSettings.gotifyPriority);
          return isNaN(parsed) ? 5 : parsed;
        })(),
        upstream_model_update_notify_enabled:
          notificationSettings.upstreamModelUpdateNotifyEnabled === true,
        record_ip_log: notificationSettings.recordIpLog,
      });

      if (res.data.success) {
        showSuccess(t('设置保存成功'));
        await getUserData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('设置保存失败'));
    }
  };

  return (
    <div className='mt-[72px]'>
      <div className='flex justify-center'>
        <div className='mx-auto w-full max-w-7xl px-2 md:px-3'>
          <div className='rounded-[32px] border border-white/10 bg-white/6 p-3 shadow-[0_30px_100px_rgba(0,0,0,0.34)] backdrop-blur-2xl md:p-5'>
          {/* 顶部用户信息区域 */}
          <UserInfoHeader t={t} userState={userState} />

          {/* 账户管理和其他设置 */}
          <div className='mt-4 grid grid-cols-1 items-start gap-4 md:mt-6 md:gap-6 xl:grid-cols-2'>
            {/* 左侧：账户管理设置 */}
            <div className='flex flex-col gap-4 md:gap-6'>
              <AccountManagement
                t={t}
                systemToken={systemToken}
                generateAccessToken={generateAccessToken}
                handleSystemTokenClick={handleSystemTokenClick}
                setShowChangePasswordModal={setShowChangePasswordModal}
                setShowPinModal={setShowPinModal}
                setShowAccountDeleteModal={setShowAccountDeleteModal}
              />
            </div>

            {/* 右侧：其他设置 */}
            <NotificationSettings
              t={t}
              notificationSettings={notificationSettings}
              handleNotificationSettingChange={handleNotificationSettingChange}
              saveNotificationSettings={saveNotificationSettings}
            />
          </div>
          </div>
        </div>
      </div>

      {/* 模态框组件 */}
      <AccountDeleteModal
        t={t}
        showAccountDeleteModal={showAccountDeleteModal}
        setShowAccountDeleteModal={setShowAccountDeleteModal}
        inputs={inputs}
        handleInputChange={handleInputChange}
        deleteAccount={deleteAccount}
        userState={userState}
        turnstileEnabled={turnstileEnabled}
        turnstileSiteKey={turnstileSiteKey}
        setTurnstileToken={setTurnstileToken}
      />

      <ChangePasswordModal
        t={t}
        showChangePasswordModal={showChangePasswordModal}
        setShowChangePasswordModal={setShowChangePasswordModal}
        inputs={inputs}
        handleInputChange={handleInputChange}
        changePassword={changePassword}
        turnstileEnabled={turnstileEnabled}
        turnstileSiteKey={turnstileSiteKey}
        setTurnstileToken={setTurnstileToken}
      />

      <Modal
        title={t('修改 PIN')}
        visible={showPinModal}
        onCancel={() => setShowPinModal(false)}
        onOk={handleUpdatePin}
        okText={t('保存 PIN')}
        cancelText={t('取消')}
        centered
      >
        <Form>
          <Form.Input
            field='currentPin'
            label={t('当前 PIN')}
            mode='password'
            value={pinInputs.currentPin}
            onChange={(value) => handlePinInputChange('currentPin', value)}
          />
          <Form.Input
            field='pin'
            label={t('新 PIN')}
            mode='password'
            value={pinInputs.pin}
            onChange={(value) => handlePinInputChange('pin', value)}
          />
          <Form.Input
            field='confirmPin'
            label={t('确认 PIN')}
            mode='password'
            value={pinInputs.confirmPin}
            onChange={(value) => handlePinInputChange('confirmPin', value)}
          />
        </Form>
      </Modal>
    </div>
  );
};

export default PersonalSetting;

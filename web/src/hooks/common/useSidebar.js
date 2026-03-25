import { useState, useEffect } from 'react';

export const useSidebar = () => {
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(false);
  }, []);

  const finalConfig = {
    console: {
      enabled: true,
      token: true,
    },
    personal: {
      enabled: true,
      personal: true,
    },
    admin: {
      enabled: true,
      channel: true,
      models: true,
      setting: true,
    },
  };

  const isModuleVisible = (sectionKey, moduleKey) => {
    if (!finalConfig[sectionKey]?.enabled) return false;
    if (!moduleKey) return true;
    return finalConfig[sectionKey]?.[moduleKey] === true;
  };

  const hasSectionVisibleModules = (sectionKey) => {
    const section = finalConfig[sectionKey];
    if (!section?.enabled) return false;
    return Object.entries(section).some(
      ([key, value]) => key !== 'enabled' && value === true,
    );
  };

  const refreshUserConfig = async () => {};

  return {
    config: finalConfig,
    userConfig: finalConfig,
    adminConfig: finalConfig,
    loading,
    isModuleVisible,
    hasSectionVisibleModules,
    refreshUserConfig,
  };
};

export default useSidebar;

import { useMemo } from 'react';

export const useUserPermissions = () => {
  const permissions = useMemo(
    () => ({
      sidebar_settings: true,
      sidebar_modules: {
        admin: {
          channel: true,
          models: true,
          setting: true,
        },
        console: {
          token: true,
        },
        personal: {
          personal: true,
        },
      },
    }),
    [],
  );

  const hasSidebarSettingsPermission = () => true;
  const isSidebarSectionAllowed = () => true;
  const isSidebarModuleAllowed = () => true;
  const getAllowedSidebarSections = () => Object.keys(permissions.sidebar_modules);
  const getAllowedSidebarModules = (sectionKey) =>
    Object.keys(permissions.sidebar_modules[sectionKey] || {});

  return {
    permissions,
    loading: false,
    error: null,
    loadPermissions: async () => permissions,
    hasSidebarSettingsPermission,
    isSidebarSectionAllowed,
    isSidebarModuleAllowed,
    getAllowedSidebarSections,
    getAllowedSidebarModules,
  };
};

export default useUserPermissions;

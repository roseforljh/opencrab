
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import zhCNTranslation from './locales/zh-CN.json';

i18n.use(initReactI18next).init({
  lng: 'zh-CN',
  fallbackLng: 'zh-CN',
  load: 'currentOnly',
  resources: {
    'zh-CN': zhCNTranslation,
  },
  nsSeparator: false,
  interpolation: {
    escapeValue: false,
  },
});

export default i18n;


import React, { useEffect, useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  verifyJSON,
} from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  CHANNEL_OPTIONS,
  MODEL_FETCHABLE_CHANNEL_TYPES,
} from '../../../../constants';
import {
  getChannelModels,
  copy,
  getChannelIcon,
  getModelCategories,
  selectFilter,
} from '../../../../helpers';
import ModelSelectModal from './ModelSelectModal';
import SingleModelSelectModal from './SingleModelSelectModal';
import OllamaModelModal from './OllamaModelModal';
import CodexOAuthModal from './CodexOAuthModal';
import ParamOverrideEditorModal from './ParamOverrideEditorModal';
import JSONEditor from '../../../common/ui/JSONEditor';
import StatusCodeRiskGuardModal from './StatusCodeRiskGuardModal';
import ChannelKeyDisplay from '../../../common/ui/ChannelKeyDisplay';
import {
  collectInvalidStatusCodeEntries,
  collectNewDisallowedStatusCodeRedirects,
} from './statusCodeRiskGuard';
import {
  Save,
  X,
  Server,
  Settings,
  Code2,
  Copy,
  Globe,
  Zap,
  Search,
  ChevronUp,
  ChevronDown,
} from 'lucide-react';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { Switch } from '@/components/ui/switch';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { Form, Highlight, Tooltip as SemiTooltip } from '@douyinfe/semi-ui';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import { IconSearch } from '@douyinfe/semi-icons';

const MODEL_MAPPING_EXAMPLE = {
  'gpt-3.5-turbo': 'gpt-3.5-turbo-0125',
};

const STATUS_CODE_MAPPING_EXAMPLE = {
  400: '500',
};

const REGION_EXAMPLE = {
  default: 'global',
  'gemini-1.5-pro-002': 'europe-west2',
  'gemini-1.5-flash-002': 'europe-west2',
  'claude-3-5-sonnet-20240620': 'europe-west1',
};
const UPSTREAM_DETECTED_MODEL_PREVIEW_LIMIT = 8;

const PARAM_OVERRIDE_LEGACY_TEMPLATE = {
  temperature: 0,
};

const PARAM_OVERRIDE_OPERATIONS_TEMPLATE = {
  operations: [
    {
      path: 'temperature',
      mode: 'set',
      value: 0.7,
      conditions: [
        {
          path: 'model',
          mode: 'prefix',
          value: 'openai/',
        },
      ],
      logic: 'AND',
    },
  ],
};

const SectionHeader = ({ icon: Icon, title, description, iconClassName }) => (
  <div className='mb-3 flex items-center gap-3'>
    <div
      className={`flex h-9 w-9 items-center justify-center rounded-full ${iconClassName}`}
    >
      <Icon className='h-4 w-4' />
    </div>
    <div>
      <div className='text-lg font-medium text-white'>{title}</div>
      {description ? (
        <div className='text-xs text-white/60'>{description}</div>
      ) : null}
    </div>
  </div>
);

const InlineAction = ({ children, onClick }) => (
  <button
    type='button'
    className='text-sm text-blue-300 transition hover:text-blue-200'
    onClick={onClick}
  >
    {children}
  </button>
);

const SectionIcon = ({ icon: Icon, className }) => (
  <Avatar
    size='sm'
    className={`border-0 shadow-md after:hidden ${className || ''}`}
  >
    <AvatarFallback className='bg-transparent text-current'>
      <Icon className='h-4 w-4' />
    </AvatarFallback>
  </Avatar>
);

const NoticeBanner = ({
  tone = 'info',
  description,
  actions,
  className = '',
}) => {
  const toneClasses = {
    info: 'border-blue-500/20 bg-blue-500/10 text-blue-100',
    warning: 'border-amber-500/20 bg-amber-500/10 text-amber-100',
  };

  return (
    <div
      className={`mb-4 rounded-xl border px-4 py-3 text-sm ${toneClasses[tone] || toneClasses.info} ${className}`}
    >
      <div className='space-y-3'>
        <div className='leading-6'>{description}</div>
        {actions ? <div className='flex flex-wrap gap-2'>{actions}</div> : null}
      </div>
    </div>
  );
};

const FieldLabel = ({ children }) => (
  <label className='text-sm font-medium text-white'>{children}</label>
);

const FieldHint = ({ children, tone = 'muted' }) => {
  const className =
    tone === 'warning' ? 'text-amber-300' : 'text-white/60';
  return <div className={`text-xs ${className}`}>{children}</div>;
};

const ToggleField = ({ label, checked, onChange, hint, disabled = false }) => (
  <div className='space-y-2'>
    <FieldLabel>{label}</FieldLabel>
    <div className='flex items-center gap-3'>
      <Switch checked={!!checked} onCheckedChange={onChange} disabled={disabled} />
      <span className='text-sm text-white/70'>{checked ? '开' : '关'}</span>
    </div>
    {hint ? <FieldHint>{hint}</FieldHint> : null}
  </div>
);

const UploadPlaceholder = ({ title, subtitle }) => (
  <div className='rounded-xl border border-dashed border-white/15 bg-white/5 px-4 py-6 text-center'>
    <div className='mb-2 flex justify-center text-white/70'>
      <Zap className='h-5 w-5' />
    </div>
    <div className='text-sm font-medium text-white'>{title}</div>
    <div className='mt-1 text-xs text-white/60'>{subtitle}</div>
  </div>
);

const readJsonSetting = (raw, key, fallback = '') => {
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw);
    return parsed?.[key] ?? fallback;
  } catch {
    return fallback;
  }
};

// 支持并且已适配通过接口获取模型列表的渠道类型
const MODEL_FETCHABLE_TYPES = new Set([
  1, 4, 14, 34, 17, 26, 27, 24, 47, 25, 20, 23, 31, 40, 42, 48, 43,
]);

function type2secretPrompt(type) {
  // inputs.type === 15 ? '按照如下格式输入：APIKey|SecretKey' : (inputs.type === 18 ? '按照如下格式输入：APPID|APISecret|APIKey' : '请输入渠道对应的鉴权密钥')
  switch (type) {
    case 15:
      return '按照如下格式输入：APIKey|SecretKey';
    case 18:
      return '按照如下格式输入：APPID|APISecret|APIKey';
    case 22:
      return '按照如下格式输入：APIKey-AppId，例如：fastgpt-0sp2gtvfdgyi4k30jwlgwf1i-64f335d84283f05518e9e041';
    case 23:
      return '按照如下格式输入：AppId|SecretId|SecretKey';
    case 33:
      return '按照如下格式输入：Ak|Sk|Region';
    case 45:
      return '请输入渠道对应的鉴权密钥, 豆包语音输入：AppId|AccessToken';
    case 50:
      return '按照如下格式输入: AccessKey|SecretKey, 如果上游是 OpenCrab 兼容网关，则直接输 ApiKey';
    case 51:
      return '按照如下格式输入: AccessKey|SecretAccessKey';
    case 57:
      return '请输入 JSON 格式的 OAuth 凭据（必须包含 access_token 和 account_id）';
    default:
      return '请输入渠道对应的鉴权密钥';
  }
}

const EditChannelModal = (props) => {
  const { t } = useTranslation();
  const channelId = props.editingChannel.id;
  const isEdit = channelId !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const isMobile = useIsMobile();
  const handleCancel = () => {
    props.handleClose();
  };
  const originInputs = {
    name: '',
    type: 1,
    key: '',
    openai_organization: '',
    max_input_tokens: 0,
    base_url: '',
    other: '',
    model_mapping: '',
    param_override: '',
    status_code_mapping: '',
    models: [],
    auto_ban: 1,
    test_model: '',
    groups: ['default'],
    priority: 0,
    weight: 0,
    tag: '',
    multi_key_mode: 'random',
    // 渠道额外设置的默认值
    force_format: false,
    thinking_to_content: false,
    proxy: '',
    pass_through_body_enabled: false,
    system_prompt: '',
    system_prompt_override: false,
    settings: '',
    // 仅 Vertex: 密钥格式（存入 settings.vertex_key_type）
    vertex_key_type: 'json',
    // 仅 AWS: 密钥格式和区域（存入 settings.aws_key_type 和 settings.aws_region）
    aws_key_type: 'ak_sk',
    // 企业账户设置
    is_enterprise_account: false,
    // 字段透传控制默认值
    allow_service_tier: false,
    disable_store: false, // false = 允许透传（默认开启）
    allow_safety_identifier: false,
    allow_include_obfuscation: false,
    allow_inference_geo: false,
    claude_beta_query: false,
    upstream_model_update_check_enabled: false,
    upstream_model_update_auto_sync_enabled: false,
    upstream_model_update_last_check_time: 0,
    upstream_model_update_last_detected_models: [],
    upstream_model_update_ignored_models: '',
  };
  const [batch, setBatch] = useState(false);
  const [multiToSingle, setMultiToSingle] = useState(false);
  const [multiKeyMode, setMultiKeyMode] = useState('random');
  const [autoBan, setAutoBan] = useState(true);
  const [inputs, setInputs] = useState(originInputs);
  const [originModelOptions, setOriginModelOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);
  const [groupOptions, setGroupOptions] = useState([]);
  const [basicModels, setBasicModels] = useState([]);
  const [fullModels, setFullModels] = useState([]);
  const [modelGroups, setModelGroups] = useState([]);
  const [customModel, setCustomModel] = useState('');
  const [modelSearchValue, setModelSearchValue] = useState('');
  const [modalImageUrl, setModalImageUrl] = useState('');
  const [isModalOpenurl, setIsModalOpenurl] = useState(false);
  const [modelModalVisible, setModelModalVisible] = useState(false);
  const [fetchedModels, setFetchedModels] = useState([]);
  const [modelMappingValueModalVisible, setModelMappingValueModalVisible] =
    useState(false);
  const [modelMappingValueModalModels, setModelMappingValueModalModels] =
    useState([]);
  const [modelMappingValueKey, setModelMappingValueKey] = useState('');
  const [modelMappingValueSelected, setModelMappingValueSelected] =
    useState('');
  const [ollamaModalVisible, setOllamaModalVisible] = useState(false);
  const formApiRef = useRef(null);
  const [vertexKeys, setVertexKeys] = useState([]);
  const [vertexFileList, setVertexFileList] = useState([]);
  const vertexErroredNames = useRef(new Set()); // 避免重复报错
  const [isMultiKeyChannel, setIsMultiKeyChannel] = useState(false);
  const [channelSearchValue, setChannelSearchValue] = useState('');
  const [useManualInput, setUseManualInput] = useState(false); // 是否使用手动输入模式
  const [keyMode, setKeyMode] = useState('append'); // 密钥模式：replace（覆盖）或 append（追加）
  const [isEnterpriseAccount, setIsEnterpriseAccount] = useState(false); // 是否为企业账户
  const [doubaoApiEditUnlocked, setDoubaoApiEditUnlocked] = useState(false); // 豆包渠道自定义 API 地址隐藏入口
  const redirectModelList = useMemo(() => {
    const mapping = inputs.model_mapping;
    if (typeof mapping !== 'string') return [];
    const trimmed = mapping.trim();
    if (!trimmed) return [];
    try {
      const parsed = JSON.parse(trimmed);
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
        return [];
      }
      const values = Object.values(parsed)
        .map((value) => (typeof value === 'string' ? value.trim() : undefined))
        .filter((value) => value);
      return Array.from(new Set(values));
    } catch (error) {
      return [];
    }
  }, [inputs.model_mapping]);
  const upstreamDetectedModels = useMemo(
    () =>
      Array.from(
        new Set(
          (inputs.upstream_model_update_last_detected_models || [])
            .map((model) => String(model || '').trim())
            .filter(Boolean),
        ),
      ),
    [inputs.upstream_model_update_last_detected_models],
  );
  const upstreamDetectedModelsPreview = useMemo(
    () =>
      upstreamDetectedModels.slice(0, UPSTREAM_DETECTED_MODEL_PREVIEW_LIMIT),
    [upstreamDetectedModels],
  );
  const upstreamDetectedModelsOmittedCount =
    upstreamDetectedModels.length - upstreamDetectedModelsPreview.length;
  const modelSearchMatchedCount = useMemo(() => {
    const keyword = modelSearchValue.trim();
    if (!keyword) {
      return modelOptions.length;
    }
    return modelOptions.reduce(
      (count, option) => count + (selectFilter(keyword, option) ? 1 : 0),
      0,
    );
  }, [modelOptions, modelSearchValue]);
  const modelSearchHintText = useMemo(() => {
    const keyword = modelSearchValue.trim();
    if (!keyword || modelSearchMatchedCount !== 0) {
      return '';
    }
    return t('未匹配到模型，按回车键可将「{{name}}」作为自定义模型名添加', {
      name: keyword,
    });
  }, [modelSearchMatchedCount, modelSearchValue, t]);
  const paramOverrideMeta = useMemo(() => {
    const raw =
      typeof inputs.param_override === 'string'
        ? inputs.param_override.trim()
        : '';
    if (!raw) {
      return {
        tagLabel: t('不更改'),
        tagColor: 'grey',
        preview: t('此项可选，用于覆盖请求参数。不支持覆盖 stream 参数'),
      };
    }
    if (!verifyJSON(raw)) {
      return {
        tagLabel: t('JSON格式错误'),
        tagColor: 'red',
        preview: raw,
      };
    }
    try {
      const parsed = JSON.parse(raw);
      const pretty = JSON.stringify(parsed, null, 2);
      if (
        parsed &&
        typeof parsed === 'object' &&
        !Array.isArray(parsed) &&
        Array.isArray(parsed.operations)
      ) {
        return {
          tagLabel: `${t('新格式模板')} (${parsed.operations.length})`,
          tagColor: 'cyan',
          preview: pretty,
        };
      }
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return {
          tagLabel: `${t('旧格式模板')} (${Object.keys(parsed).length})`,
          tagColor: 'blue',
          preview: pretty,
        };
      }
      return {
        tagLabel: t('自定义 JSON'),
        tagColor: 'orange',
        preview: pretty,
      };
    } catch (error) {
      return {
        tagLabel: t('JSON格式错误'),
        tagColor: 'red',
        preview: raw,
      };
    }
  }, [inputs.param_override, t]);
  const [isIonetChannel, setIsIonetChannel] = useState(false);
  const [ionetMetadata, setIonetMetadata] = useState(null);
  const [codexOAuthModalVisible, setCodexOAuthModalVisible] = useState(false);
  const [codexCredentialRefreshing, setCodexCredentialRefreshing] =
    useState(false);
  const [paramOverrideEditorVisible, setParamOverrideEditorVisible] =
    useState(false);

  // 密钥显示状态
  const [keyDisplayState, setKeyDisplayState] = useState({
    showModal: false,
    keyData: '',
  });

  // 专门的2FA验证状态（用于TwoFactorAuthModal）
  const [show2FAVerifyModal, setShow2FAVerifyModal] = useState(false);
  const [verifyCode, setVerifyCode] = useState('');

  useEffect(() => {
    if (!isEdit) {
      setIsIonetChannel(false);
      setIonetMetadata(null);
    }
  }, [isEdit]);

  const handleOpenIonetDeployment = () => {
    if (!ionetMetadata?.deployment_id) {
      return;
    }
    const targetUrl = `/console/deployment?deployment_id=${ionetMetadata.deployment_id}`;
    window.open(targetUrl, '_blank', 'noopener');
  };
  const [verifyLoading, setVerifyLoading] = useState(false);
  const statusCodeRiskConfirmResolverRef = useRef(null);
  const [statusCodeRiskConfirmVisible, setStatusCodeRiskConfirmVisible] =
    useState(false);
  const [statusCodeRiskDetailItems, setStatusCodeRiskDetailItems] = useState(
    [],
  );

  // 表单块导航相关状态
  const formSectionRefs = useRef({
    basicInfo: null,
    apiConfig: null,
    modelConfig: null,
    advancedSettings: null,
    channelExtraSettings: null,
  });
  const [currentSectionIndex, setCurrentSectionIndex] = useState(0);
  const formSections = [
    'basicInfo',
    'apiConfig',
    'modelConfig',
    'advancedSettings',
    'channelExtraSettings',
  ];
  const formContainerRef = useRef(null);
  const doubaoApiClickCountRef = useRef(0);
  const initialModelsRef = useRef([]);
  const initialModelMappingRef = useRef('');
  const initialStatusCodeMappingRef = useRef('');

  // 2FA状态更新辅助函数
  const updateTwoFAState = (updates) => {
    setTwoFAState((prev) => ({ ...prev, ...updates }));
  };

  // 重置密钥显示状态
  const resetKeyDisplayState = () => {
    setKeyDisplayState({
      showModal: false,
      keyData: '',
    });
  };

  // 重置2FA验证状态
  const reset2FAVerifyState = () => {
    setShow2FAVerifyModal(false);
    setVerifyCode('');
    setVerifyLoading(false);
  };

  // 表单导航功能
  const scrollToSection = (sectionKey) => {
    const sectionElement = formSectionRefs.current[sectionKey];
    if (sectionElement) {
      sectionElement.scrollIntoView({
        behavior: 'smooth',
        block: 'start',
        inline: 'nearest',
      });
    }
  };

  const navigateToSection = (direction) => {
    const availableSections = formSections.filter((section) => {
      if (section === 'apiConfig') {
        return showApiConfigCard;
      }
      return true;
    });

    let newIndex;
    if (direction === 'up') {
      newIndex =
        currentSectionIndex > 0
          ? currentSectionIndex - 1
          : availableSections.length - 1;
    } else {
      newIndex =
        currentSectionIndex < availableSections.length - 1
          ? currentSectionIndex + 1
          : 0;
    }

    setCurrentSectionIndex(newIndex);
    scrollToSection(availableSections[newIndex]);
  };

  const handleApiConfigSecretClick = () => {
    if (inputs.type !== 45) return;
    const next = doubaoApiClickCountRef.current + 1;
    doubaoApiClickCountRef.current = next;
    if (next >= 10) {
      setDoubaoApiEditUnlocked((unlocked) => {
        if (!unlocked) {
          showInfo(t('已解锁豆包自定义 API 地址编辑'));
        }
        return true;
      });
    }
  };

  // 渠道额外设置状态
  const [channelSettings, setChannelSettings] = useState({
    force_format: false,
    thinking_to_content: false,
    proxy: '',
    pass_through_body_enabled: false,
    system_prompt: '',
  });
  const showApiConfigCard = true; // 控制是否显示 API 配置卡片
  const getInitValues = () => ({ ...originInputs });

  // 处理渠道额外设置的更新
  const handleChannelSettingsChange = (key, value) => {
    // 更新内部状态
    setChannelSettings((prev) => ({ ...prev, [key]: value }));

    // 同步更新到表单字段
    if (formApiRef.current) {
      formApiRef.current.setValue(key, value);
    }

    // 同步更新inputs状态
    setInputs((prev) => ({ ...prev, [key]: value }));

    // 生成setting JSON并更新
    const newSettings = { ...channelSettings, [key]: value };
    const settingsJson = JSON.stringify(newSettings);
    handleInputChange('setting', settingsJson);
  };

  const handleChannelOtherSettingsChange = (key, value) => {
    // 更新内部状态
    setChannelSettings((prev) => ({ ...prev, [key]: value }));

    // 同步更新到表单字段
    if (formApiRef.current) {
      formApiRef.current.setValue(key, value);
    }

    // 同步更新inputs状态
    setInputs((prev) => ({ ...prev, [key]: value }));

    // 需要更新settings，是一个json，例如{"azure_responses_version": "preview"}
    let settings = {};
    if (inputs.settings) {
      try {
        settings = JSON.parse(inputs.settings);
      } catch (error) {
        console.error('解析设置失败:', error);
      }
    }
    settings[key] = value;
    const settingsJson = JSON.stringify(settings);
    handleInputChange('settings', settingsJson);
  };

  const isIonetLocked = isIonetChannel && isEdit;

  const handleInputChange = (name, value) => {
    if (
      isIonetChannel &&
      isEdit &&
      ['type', 'key', 'base_url'].includes(name)
    ) {
      return;
    }
    if (formApiRef.current) {
      formApiRef.current.setValue(name, value);
    }
    if (name === 'models' && Array.isArray(value)) {
      value = Array.from(new Set(value.map((m) => (m || '').trim())));
    }

    if (name === 'base_url' && value.endsWith('/v1')) {
      const confirmed = window.confirm(
        '不需要在末尾加/v1，OpenCrab 会自动处理，添加后可能导致请求失败，是否继续？',
      );
      if (confirmed) {
        setInputs((inputs) => ({ ...inputs, [name]: value }));
      }
      return;
    }
    setInputs((inputs) => ({ ...inputs, [name]: value }));
    if (name === 'type') {
      let localModels = [];
      switch (value) {
        case 2:
          localModels = [
            'mj_imagine',
            'mj_variation',
            'mj_reroll',
            'mj_blend',
            'mj_upscale',
            'mj_describe',
            'mj_uploads',
          ];
          break;
        case 5:
          localModels = [
            'swap_face',
            'mj_imagine',
            'mj_video',
            'mj_edits',
            'mj_variation',
            'mj_reroll',
            'mj_blend',
            'mj_upscale',
            'mj_describe',
            'mj_zoom',
            'mj_shorten',
            'mj_modal',
            'mj_inpaint',
            'mj_custom_zoom',
            'mj_high_variation',
            'mj_low_variation',
            'mj_pan',
            'mj_uploads',
          ];
          break;
        case 36:
          localModels = ['suno_music', 'suno_lyrics'];
          break;
        case 45:
          localModels = getChannelModels(value);
          setInputs((prevInputs) => ({
            ...prevInputs,
            base_url: 'https://ark.cn-beijing.volces.com',
          }));
          break;
        default:
          localModels = getChannelModels(value);
          break;
      }
      if (inputs.models.length === 0) {
        setInputs((inputs) => ({ ...inputs, models: localModels }));
      }
      setBasicModels(localModels);

      // 重置手动输入模式状态
      setUseManualInput(false);

      if (value === 57) {
        setBatch(false);
        setMultiToSingle(false);
        setMultiKeyMode('random');
        setVertexKeys([]);
        setVertexFileList([]);
        if (formApiRef.current) {
          formApiRef.current.setValue('vertex_files', []);
        }
        setInputs((prev) => ({ ...prev, vertex_files: [] }));
      }
    }
    //setAutoBan
  };

  const formatJsonField = (fieldName) => {
    const rawValue = (inputs?.[fieldName] ?? '').trim();
    if (!rawValue) return;

    try {
      const parsed = JSON.parse(rawValue);
      handleInputChange(fieldName, JSON.stringify(parsed, null, 2));
    } catch (error) {
      showError(`${t('JSON格式错误')}: ${error.message}`);
    }
  };

  const formatUnixTime = (timestamp) => {
    const value = Number(timestamp || 0);
    if (!value) {
      return t('暂无');
    }
    return new Date(value * 1000).toLocaleString();
  };

  const copyParamOverrideJson = async () => {
    const raw =
      typeof inputs.param_override === 'string'
        ? inputs.param_override.trim()
        : '';
    if (!raw) {
      showInfo(t('暂无可复制 JSON'));
      return;
    }

    let content = raw;
    if (verifyJSON(raw)) {
      try {
        content = JSON.stringify(JSON.parse(raw), null, 2);
      } catch (error) {
        content = raw;
      }
    }

    const ok = await copy(content);
    if (ok) {
      showSuccess(t('参数覆盖 JSON 已复制'));
    } else {
      showError(t('复制失败'));
    }
  };

  const parseParamOverrideInput = () => {
    const raw =
      typeof inputs.param_override === 'string'
        ? inputs.param_override.trim()
        : '';
    if (!raw) return null;
    if (!verifyJSON(raw)) {
      throw new Error(t('当前参数覆盖不是合法的 JSON'));
    }
    return JSON.parse(raw);
  };

  const applyParamOverrideTemplate = (
    templateType = 'operations',
    applyMode = 'fill',
  ) => {
    try {
      const parsedCurrent = parseParamOverrideInput();
      if (templateType === 'legacy') {
        if (applyMode === 'fill') {
          handleInputChange(
            'param_override',
            JSON.stringify(PARAM_OVERRIDE_LEGACY_TEMPLATE, null, 2),
          );
          return;
        }
        const currentLegacy =
          parsedCurrent &&
          typeof parsedCurrent === 'object' &&
          !Array.isArray(parsedCurrent) &&
          !Array.isArray(parsedCurrent.operations)
            ? parsedCurrent
            : {};
        const merged = {
          ...PARAM_OVERRIDE_LEGACY_TEMPLATE,
          ...currentLegacy,
        };
        handleInputChange('param_override', JSON.stringify(merged, null, 2));
        return;
      }

      if (applyMode === 'fill') {
        handleInputChange(
          'param_override',
          JSON.stringify(PARAM_OVERRIDE_OPERATIONS_TEMPLATE, null, 2),
        );
        return;
      }
      const currentOperations =
        parsedCurrent &&
        typeof parsedCurrent === 'object' &&
        !Array.isArray(parsedCurrent) &&
        Array.isArray(parsedCurrent.operations)
          ? parsedCurrent.operations
          : [];
      const merged = {
        operations: [
          ...currentOperations,
          ...PARAM_OVERRIDE_OPERATIONS_TEMPLATE.operations,
        ],
      };
      handleInputChange('param_override', JSON.stringify(merged, null, 2));
    } catch (error) {
      showError(error.message || t('模板应用失败'));
    }
  };

  const clearParamOverride = () => {
    handleInputChange('param_override', '');
  };

  const loadChannel = async () => {
    setLoading(true);
    let res = await API.get(`/api/channel/${channelId}`);
    if (res === undefined) {
      return;
    }
    const { success, message, data } = res.data;
    if (success) {
      if (data.models === '') {
        data.models = [];
      } else {
        data.models = data.models.split(',');
      }
      if (data.group === '') {
        data.groups = [];
      } else {
        data.groups = data.group.split(',');
      }
      if (data.model_mapping !== '') {
        data.model_mapping = JSON.stringify(
          JSON.parse(data.model_mapping),
          null,
          2,
        );
      }
      const chInfo = data.channel_info || {};
      const isMulti = chInfo.is_multi_key === true;
      setIsMultiKeyChannel(isMulti);
      if (isMulti) {
        setBatch(true);
        setMultiToSingle(true);
        const modeVal = chInfo.multi_key_mode || 'random';
        setMultiKeyMode(modeVal);
        data.multi_key_mode = modeVal;
      } else {
        setBatch(false);
        setMultiToSingle(false);
      }
      // 解析渠道额外设置并合并到data中
      if (data.setting) {
        try {
          const parsedSettings = JSON.parse(data.setting);
          data.force_format = parsedSettings.force_format || false;
          data.thinking_to_content =
            parsedSettings.thinking_to_content || false;
          data.proxy = parsedSettings.proxy || '';
          data.pass_through_body_enabled =
            parsedSettings.pass_through_body_enabled || false;
          data.system_prompt = parsedSettings.system_prompt || '';
          data.system_prompt_override =
            parsedSettings.system_prompt_override || false;
        } catch (error) {
          console.error('解析渠道设置失败:', error);
          data.force_format = false;
          data.thinking_to_content = false;
          data.proxy = '';
          data.pass_through_body_enabled = false;
          data.system_prompt = '';
          data.system_prompt_override = false;
        }
      } else {
        data.force_format = false;
        data.thinking_to_content = false;
        data.proxy = '';
        data.pass_through_body_enabled = false;
        data.system_prompt = '';
        data.system_prompt_override = false;
      }

      if (data.settings) {
        try {
          const parsedSettings = JSON.parse(data.settings);
          data.azure_responses_version =
            parsedSettings.azure_responses_version || '';
          // 读取 Vertex 密钥格式
          data.vertex_key_type = parsedSettings.vertex_key_type || 'json';
          // 读取 AWS 密钥格式和区域
          data.aws_key_type = parsedSettings.aws_key_type || 'ak_sk';
          // 读取企业账户设置
          data.is_enterprise_account =
            parsedSettings.openrouter_enterprise === true;
          // 读取字段透传控制设置
          data.allow_service_tier = parsedSettings.allow_service_tier || false;
          data.disable_store = parsedSettings.disable_store || false;
          data.allow_safety_identifier =
            parsedSettings.allow_safety_identifier || false;
          data.allow_include_obfuscation =
            parsedSettings.allow_include_obfuscation || false;
          data.allow_inference_geo =
            parsedSettings.allow_inference_geo || false;
          data.claude_beta_query = parsedSettings.claude_beta_query || false;
          data.upstream_model_update_check_enabled =
            parsedSettings.upstream_model_update_check_enabled === true;
          data.upstream_model_update_auto_sync_enabled =
            parsedSettings.upstream_model_update_auto_sync_enabled === true;
          data.upstream_model_update_last_check_time =
            Number(parsedSettings.upstream_model_update_last_check_time) || 0;
          data.upstream_model_update_last_detected_models = Array.isArray(
            parsedSettings.upstream_model_update_last_detected_models,
          )
            ? parsedSettings.upstream_model_update_last_detected_models
            : [];
          data.upstream_model_update_ignored_models = Array.isArray(
            parsedSettings.upstream_model_update_ignored_models,
          )
            ? parsedSettings.upstream_model_update_ignored_models.join(',')
            : '';
        } catch (error) {
          console.error('解析其他设置失败:', error);
          data.azure_responses_version = '';
          data.region = '';
          data.vertex_key_type = 'json';
          data.aws_key_type = 'ak_sk';
          data.is_enterprise_account = false;
          data.allow_service_tier = false;
          data.disable_store = false;
          data.allow_safety_identifier = false;
          data.allow_include_obfuscation = false;
          data.allow_inference_geo = false;
          data.claude_beta_query = false;
          data.upstream_model_update_check_enabled = false;
          data.upstream_model_update_auto_sync_enabled = false;
          data.upstream_model_update_last_check_time = 0;
          data.upstream_model_update_last_detected_models = [];
          data.upstream_model_update_ignored_models = '';
        }
      } else {
        // 兼容历史数据：老渠道没有 settings 时，默认按 json 展示
        data.vertex_key_type = 'json';
        data.aws_key_type = 'ak_sk';
        data.is_enterprise_account = false;
        data.allow_service_tier = false;
        data.disable_store = false;
        data.allow_safety_identifier = false;
        data.allow_include_obfuscation = false;
        data.allow_inference_geo = false;
        data.claude_beta_query = false;
        data.upstream_model_update_check_enabled = false;
        data.upstream_model_update_auto_sync_enabled = false;
        data.upstream_model_update_last_check_time = 0;
        data.upstream_model_update_last_detected_models = [];
        data.upstream_model_update_ignored_models = '';
      }

      if (
        data.type === 45 &&
        (!data.base_url ||
          (typeof data.base_url === 'string' && data.base_url.trim() === ''))
      ) {
        data.base_url = 'https://ark.cn-beijing.volces.com';
      }

      setInputs(data);
      if (formApiRef.current) {
        formApiRef.current.setValues(data);
      }
      if (data.auto_ban === 0) {
        setAutoBan(false);
      } else {
        setAutoBan(true);
      }
      // 同步企业账户状态
      setIsEnterpriseAccount(data.is_enterprise_account || false);
      setBasicModels(getChannelModels(data.type));
      // 同步更新channelSettings状态显示
      setChannelSettings({
        force_format: data.force_format,
        thinking_to_content: data.thinking_to_content,
        proxy: data.proxy,
        pass_through_body_enabled: data.pass_through_body_enabled,
        system_prompt: data.system_prompt,
        system_prompt_override: data.system_prompt_override || false,
      });
      initialModelsRef.current = (data.models || [])
        .map((model) => (model || '').trim())
        .filter(Boolean);
      initialModelMappingRef.current = data.model_mapping || '';
      initialStatusCodeMappingRef.current = data.status_code_mapping || '';

      let parsedIonet = null;
      if (data.other_info) {
        try {
          const maybeMeta = JSON.parse(data.other_info);
          if (
            maybeMeta &&
            typeof maybeMeta === 'object' &&
            maybeMeta.source === 'ionet'
          ) {
            parsedIonet = maybeMeta;
          }
        } catch (error) {
          // ignore parse error
        }
      }
      const managedByIonet = !!parsedIonet;
      setIsIonetChannel(managedByIonet);
      setIonetMetadata(parsedIonet);
      // console.log(data);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const fetchUpstreamModelList = async (name, options = {}) => {
    const silent = !!options.silent;
    // if (inputs['type'] !== 1) {
    //   showError(t('仅支持 OpenAI 接口格式'));
    //   return;
    // }
    setLoading(true);
    const models = [];
    let err = false;

    if (isEdit) {
      // 如果是编辑模式，使用已有的 channelId 获取模型列表
      const res = await API.get('/api/channel/fetch_models/' + channelId, {
        skipErrorHandler: true,
      });
      if (res && res.data && res.data.success) {
        models.push(...res.data.data);
      } else {
        err = true;
      }
    } else {
      // 如果是新建模式，通过后端代理获取模型列表
      if (!inputs?.['key']) {
        showError(t('请填写密钥'));
        err = true;
      } else {
        try {
          const res = await API.post(
            '/api/channel/fetch_models',
            {
              base_url: inputs['base_url'],
              type: inputs['type'],
              key: inputs['key'],
            },
            { skipErrorHandler: true },
          );

          if (res && res.data && res.data.success) {
            models.push(...res.data.data);
          } else {
            err = true;
          }
        } catch (error) {
          console.error('Error fetching models:', error);
          err = true;
        }
      }
    }

    if (!err) {
      const uniqueModels = Array.from(new Set(models));
      setFetchedModels(uniqueModels);
      if (!silent) {
        setModelModalVisible(true);
      }
      setLoading(false);
      return uniqueModels;
    } else {
      showError(t('获取模型列表失败'));
    }
    setLoading(false);
    return null;
  };

  const openModelMappingValueModal = async ({ pairKey, value }) => {
    const mappingKey = String(pairKey ?? '').trim();
    if (!mappingKey) return;

    if (!MODEL_FETCHABLE_CHANNEL_TYPES.has(inputs.type)) {
      return;
    }

    let modelsToUse = fetchedModels;
    if (!Array.isArray(modelsToUse) || modelsToUse.length === 0) {
      const fetched = await fetchUpstreamModelList('models', { silent: true });
      if (Array.isArray(fetched)) {
        modelsToUse = fetched;
      }
    }

    if (!Array.isArray(modelsToUse) || modelsToUse.length === 0) {
      showInfo(t('暂无模型'));
      return;
    }

    const normalizedModelsToUse = Array.from(
      new Set(
        modelsToUse.map((model) => String(model ?? '').trim()).filter(Boolean),
      ),
    );
    const currentValue = String(value ?? '').trim();

    setModelMappingValueModalModels(normalizedModelsToUse);
    setModelMappingValueKey(mappingKey);
    setModelMappingValueSelected(
      normalizedModelsToUse.includes(currentValue) ? currentValue : '',
    );
    setModelMappingValueModalVisible(true);
  };

  const fetchModels = async () => {
    try {
      let res = await API.get(`/api/channel/models`);
      const localModelOptions = res.data.data.map((model) => {
        const id = (model.id || '').trim();
        return {
          key: id,
          label: id,
          value: id,
        };
      });
      setOriginModelOptions(localModelOptions);
      setFullModels(res.data.data.map((model) => model.id));
      setBasicModels(
        res.data.data
          .filter((model) => {
            return model.id.startsWith('gpt-') || model.id.startsWith('text-');
          })
          .map((model) => model.id),
      );
    } catch (error) {
      showError(error.message);
    }
  };

  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/user/groups`);
      if (res === undefined) {
        return;
      }
      const groups = Object.keys(res?.data?.data || {});
      setGroupOptions(
        groups.map((group) => ({
          label: group,
          value: group,
        })),
      );
    } catch (error) {
      showError(error.message);
    }
  };

  const fetchModelGroups = async () => {
    try {
      const res = await API.get('/api/prefill_group?type=model');
      if (res?.data?.success) {
        setModelGroups(res.data.data || []);
      }
    } catch (error) {
      // ignore
    }
  };

  // 查看渠道密钥
  const handleShow2FAModal = async () => {
    try {
      const res = await API.get(`/api/channel/${channelId}/key`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('获取密钥失败'));
        return;
      }
      if (!data?.key) {
        showError(t('获取密钥失败'));
        return;
      }
      showSuccess(t('密钥获取成功'));
      setKeyDisplayState({
        showModal: true,
        keyData: data.key,
      });
    } catch (error) {
      console.error('Failed to view channel key:', error);
      showError(error.message || t('获取密钥失败'));
    }
  };

  const handleCodexOAuthGenerated = (key) => {
    handleInputChange('key', key);
    formatJsonField('key');
  };

  const handleRefreshCodexCredential = async () => {
    if (!isEdit) return;

    setCodexCredentialRefreshing(true);
    try {
      const res = await API.post(
        `/api/channel/${channelId}/codex/refresh`,
        {},
        { skipErrorHandler: true },
      );
      if (!res?.data?.success) {
        throw new Error(res?.data?.message || 'Failed to refresh credential');
      }
      showSuccess(t('凭证已刷新'));
    } catch (error) {
      showError(error.message || t('刷新失败'));
    } finally {
      setCodexCredentialRefreshing(false);
    }
  };

  useEffect(() => {
    if (inputs.type !== 45) {
      doubaoApiClickCountRef.current = 0;
      setDoubaoApiEditUnlocked(false);
    }
  }, [inputs.type]);

  useEffect(() => {
    const modelMap = new Map();

    originModelOptions.forEach((option) => {
      const v = (option.value || '').trim();
      if (!modelMap.has(v)) {
        modelMap.set(v, option);
      }
    });

    inputs.models.forEach((model) => {
      const v = (model || '').trim();
      if (!modelMap.has(v)) {
        modelMap.set(v, {
          key: v,
          label: v,
          value: v,
        });
      }
    });

    const categories = getModelCategories(t);
    const optionsWithIcon = Array.from(modelMap.values()).map((opt) => {
      const modelName = opt.value;
      let icon = null;
      for (const [key, category] of Object.entries(categories)) {
        if (key !== 'all' && category.filter({ model_name: modelName })) {
          icon = category.icon;
          break;
        }
      }
      return {
        ...opt,
        label: (
          <span className='flex items-center gap-1'>
            {icon}
            {modelName}
          </span>
        ),
      };
    });

    setModelOptions(optionsWithIcon);
  }, [originModelOptions, inputs.models, t]);

  useEffect(() => {
    fetchModels().then();
    fetchGroups().then();
    if (!isEdit) {
      setInputs(originInputs);
      if (formApiRef.current) {
        formApiRef.current.setValues(originInputs);
      }
      let localModels = getChannelModels(inputs.type);
      setBasicModels(localModels);
      setInputs((inputs) => ({ ...inputs, models: localModels }));
    }
  }, [props.editingChannel.id]);

  useEffect(() => {
    if (formApiRef.current) {
      formApiRef.current.setValues(inputs);
    }
  }, [inputs]);

  useEffect(() => {
    setModelSearchValue('');
    if (props.visible) {
      if (isEdit) {
        loadChannel();
      } else {
        formApiRef.current?.setValues(getInitValues());
      }
      fetchModelGroups();
      // 重置手动输入模式状态
      setUseManualInput(false);
      // 重置导航状态
      setCurrentSectionIndex(0);
    } else {
      // 统一的模态框关闭重置逻辑
      resetModalState();
    }
  }, [props.visible, channelId]);

  useEffect(() => {
    if (!isEdit) {
      initialModelsRef.current = [];
      initialModelMappingRef.current = '';
      initialStatusCodeMappingRef.current = '';
    }
  }, [isEdit, props.visible]);

  useEffect(() => {
    return () => {
      if (statusCodeRiskConfirmResolverRef.current) {
        statusCodeRiskConfirmResolverRef.current(false);
        statusCodeRiskConfirmResolverRef.current = null;
      }
    };
  }, []);

  // 统一的模态框重置函数
  const resetModalState = () => {
    resolveStatusCodeRiskConfirm(false);
    formApiRef.current?.reset();
    // 重置渠道设置状态
    setChannelSettings({
      force_format: false,
      thinking_to_content: false,
      proxy: '',
      pass_through_body_enabled: false,
      system_prompt: '',
      system_prompt_override: false,
    });
    // 重置密钥模式状态
    setKeyMode('append');
    // 重置企业账户状态
    setIsEnterpriseAccount(false);
    // 重置豆包隐藏入口状态
    setDoubaoApiEditUnlocked(false);
    doubaoApiClickCountRef.current = 0;
    setModelSearchValue('');
    // 清空表单中的key_mode字段
    if (formApiRef.current) {
      formApiRef.current.setValue('key_mode', undefined);
    }
    // 重置本地输入，避免下次打开残留上一次的 JSON 字段值
    setInputs(getInitValues());
    // 重置密钥显示状态
    resetKeyDisplayState();
  };

  const handleVertexUploadChange = ({ fileList }) => {
    vertexErroredNames.current.clear();
    (async () => {
      let validFiles = [];
      let keys = [];
      const errorNames = [];
      for (const item of fileList) {
        const fileObj = item.fileInstance;
        if (!fileObj) continue;
        try {
          const txt = await fileObj.text();
          keys.push(JSON.parse(txt));
          validFiles.push(item);
        } catch (err) {
          if (!vertexErroredNames.current.has(item.name)) {
            errorNames.push(item.name);
            vertexErroredNames.current.add(item.name);
          }
        }
      }

      // 非批量模式下只保留一个文件（最新选择的），避免重复叠加
      if (!batch && validFiles.length > 1) {
        validFiles = [validFiles[validFiles.length - 1]];
        keys = [keys[keys.length - 1]];
      }

      setVertexKeys(keys);
      setVertexFileList(validFiles);
      if (formApiRef.current) {
        formApiRef.current.setValue('vertex_files', validFiles);
      }
      setInputs((prev) => ({ ...prev, vertex_files: validFiles }));

      if (errorNames.length > 0) {
        showError(
          t('以下文件解析失败，已忽略：{{list}}', {
            list: errorNames.join(', '),
          }),
        );
      }
    })();
  };

  const confirmMissingModelMappings = (missingModels) =>
    new Promise((resolve) => {
      const shouldAdd = window.confirm(
        `${t(
          '模型重定向里的下列模型尚未添加到“模型”列表，调用时会因为缺少可用模型而失败：',
        )}\n${missingModels.join(', ')}\n\n${t(
          '选择“确定”将自动添加后提交，选择“取消”则返回修改。',
        )}`,
      );
      resolve(shouldAdd ? 'add' : 'cancel');
    });

  const resolveStatusCodeRiskConfirm = (confirmed) => {
    setStatusCodeRiskConfirmVisible(false);
    setStatusCodeRiskDetailItems([]);
    if (statusCodeRiskConfirmResolverRef.current) {
      statusCodeRiskConfirmResolverRef.current(confirmed);
      statusCodeRiskConfirmResolverRef.current = null;
    }
  };

  const confirmStatusCodeRisk = (detailItems) =>
    new Promise((resolve) => {
      statusCodeRiskConfirmResolverRef.current = resolve;
      setStatusCodeRiskDetailItems(detailItems);
      setStatusCodeRiskConfirmVisible(true);
    });

  const hasModelConfigChanged = (normalizedModels, modelMappingStr) => {
    if (!isEdit) return true;
    const initialModels = initialModelsRef.current;
    if (normalizedModels.length !== initialModels.length) {
      return true;
    }
    for (let i = 0; i < normalizedModels.length; i++) {
      if (normalizedModels[i] !== initialModels[i]) {
        return true;
      }
    }
    const normalizedMapping = (modelMappingStr || '').trim();
    const initialMapping = (initialModelMappingRef.current || '').trim();
    return normalizedMapping !== initialMapping;
  };

  const submit = async () => {
    const formValues = formApiRef.current ? formApiRef.current.getValues() : {};
    let localInputs = { ...formValues };
    localInputs.param_override = inputs.param_override;

    if (localInputs.type === 57) {
      if (batch) {
        showInfo(t('Codex 渠道不支持批量创建'));
        return;
      }

      const rawKey = (localInputs.key || '').trim();
      if (!isEdit && rawKey === '') {
        showInfo(t('请输入密钥！'));
        return;
      }

      if (rawKey !== '') {
        if (!verifyJSON(rawKey)) {
          showInfo(t('密钥必须是合法的 JSON 格式！'));
          return;
        }
        try {
          const parsed = JSON.parse(rawKey);
          if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
            showInfo(t('密钥必须是 JSON 对象'));
            return;
          }
          const accessToken = String(parsed.access_token || '').trim();
          const accountId = String(parsed.account_id || '').trim();
          if (!accessToken) {
            showInfo(t('密钥 JSON 必须包含 access_token'));
            return;
          }
          if (!accountId) {
            showInfo(t('密钥 JSON 必须包含 account_id'));
            return;
          }
          localInputs.key = JSON.stringify(parsed);
        } catch (error) {
          showInfo(t('密钥必须是合法的 JSON 格式！'));
          return;
        }
      }
    }

    if (localInputs.type === 41) {
      const keyType = localInputs.vertex_key_type || 'json';
      if (keyType === 'api_key') {
        // 直接作为普通字符串密钥处理
        if (!isEdit && (!localInputs.key || localInputs.key.trim() === '')) {
          showInfo(t('请输入密钥！'));
          return;
        }
      } else {
        // JSON 服务账号密钥
        if (useManualInput) {
          if (localInputs.key && localInputs.key.trim() !== '') {
            try {
              const parsedKey = JSON.parse(localInputs.key);
              localInputs.key = JSON.stringify(parsedKey);
            } catch (err) {
              showError(t('密钥格式无效，请输入有效的 JSON 格式密钥'));
              return;
            }
          } else if (!isEdit) {
            showInfo(t('请输入密钥！'));
            return;
          }
        } else {
          // 文件上传模式
          let keys = vertexKeys;
          if (keys.length === 0 && vertexFileList.length > 0) {
            try {
              const parsed = await Promise.all(
                vertexFileList.map(async (item) => {
                  const fileObj = item.fileInstance;
                  if (!fileObj) return null;
                  const txt = await fileObj.text();
                  return JSON.parse(txt);
                }),
              );
              keys = parsed.filter(Boolean);
            } catch (err) {
              showError(t('解析密钥文件失败: {{msg}}', { msg: err.message }));
              return;
            }
          }
          if (keys.length === 0) {
            if (!isEdit) {
              showInfo(t('请上传密钥文件！'));
              return;
            } else {
              delete localInputs.key;
            }
          } else {
            localInputs.key = batch
              ? JSON.stringify(keys)
              : JSON.stringify(keys[0]);
          }
        }
      }
    }

    // 如果是编辑模式且 key 为空字符串，避免提交空值覆盖旧密钥
    if (isEdit && (!localInputs.key || localInputs.key.trim() === '')) {
      delete localInputs.key;
    }
    delete localInputs.vertex_files;

    if (!isEdit && (!localInputs.name || !localInputs.key)) {
      showInfo(t('请填写渠道名称和渠道密钥！'));
      return;
    }
    if (!Array.isArray(localInputs.models) || localInputs.models.length === 0) {
      showInfo(t('请至少选择一个模型！'));
      return;
    }
    if (
      localInputs.type === 45 &&
      (!localInputs.base_url || localInputs.base_url.trim() === '')
    ) {
      showInfo(t('请输入API地址！'));
      return;
    }
    const hasModelMapping =
      typeof localInputs.model_mapping === 'string' &&
      localInputs.model_mapping.trim() !== '';
    let parsedModelMapping = null;
    if (hasModelMapping) {
      if (!verifyJSON(localInputs.model_mapping)) {
        showInfo(t('模型映射必须是合法的 JSON 格式！'));
        return;
      }
      try {
        parsedModelMapping = JSON.parse(localInputs.model_mapping);
      } catch (error) {
        showInfo(t('模型映射必须是合法的 JSON 格式！'));
        return;
      }
    }

    const normalizedModels = (localInputs.models || [])
      .map((model) => (model || '').trim())
      .filter(Boolean);
    localInputs.models = normalizedModels;

    if (
      parsedModelMapping &&
      typeof parsedModelMapping === 'object' &&
      !Array.isArray(parsedModelMapping)
    ) {
      const modelSet = new Set(normalizedModels);
      const missingModels = Object.keys(parsedModelMapping)
        .map((key) => (key || '').trim())
        .filter((key) => key && !modelSet.has(key));
      const shouldPromptMissing =
        missingModels.length > 0 &&
        hasModelConfigChanged(normalizedModels, localInputs.model_mapping);
      if (shouldPromptMissing) {
        const confirmAction = await confirmMissingModelMappings(missingModels);
        if (confirmAction === 'cancel') {
          return;
        }
        if (confirmAction === 'add') {
          const updatedModels = Array.from(
            new Set([...normalizedModels, ...missingModels]),
          );
          localInputs.models = updatedModels;
          handleInputChange('models', updatedModels);
        }
      }
    }

    const invalidStatusCodeEntries = collectInvalidStatusCodeEntries(
      localInputs.status_code_mapping,
    );
    if (invalidStatusCodeEntries.length > 0) {
      showError(
        `${t('状态码复写包含无效的状态码')}: ${invalidStatusCodeEntries.join(', ')}`,
      );
      return;
    }

    const riskyStatusCodeRedirects = collectNewDisallowedStatusCodeRedirects(
      initialStatusCodeMappingRef.current,
      localInputs.status_code_mapping,
    );
    if (riskyStatusCodeRedirects.length > 0) {
      const confirmed = await confirmStatusCodeRisk(riskyStatusCodeRedirects);
      if (!confirmed) {
        return;
      }
    }

    if (localInputs.base_url && localInputs.base_url.endsWith('/')) {
      localInputs.base_url = localInputs.base_url.slice(
        0,
        localInputs.base_url.length - 1,
      );
    }
    if (localInputs.type === 18 && localInputs.other === '') {
      localInputs.other = 'v2.1';
    }

    // 生成渠道额外设置JSON
    const channelExtraSettings = {
      force_format: localInputs.force_format || false,
      thinking_to_content: localInputs.thinking_to_content || false,
      proxy: localInputs.proxy || '',
      pass_through_body_enabled: localInputs.pass_through_body_enabled || false,
      system_prompt: localInputs.system_prompt || '',
      system_prompt_override: localInputs.system_prompt_override || false,
    };
    localInputs.setting = JSON.stringify(channelExtraSettings);

    // 处理 settings 字段（包括企业账户设置和字段透传控制）
    let settings = {};
    if (localInputs.settings) {
      try {
        settings = JSON.parse(localInputs.settings);
      } catch (error) {
        console.error('解析settings失败:', error);
      }
    }

    // type === 20: 设置企业账户标识，无论是true还是false都要传到后端
    if (localInputs.type === 20) {
      settings.openrouter_enterprise =
        localInputs.is_enterprise_account === true;
    }

    // type === 33 (AWS): 保存 aws_key_type 到 settings
    if (localInputs.type === 33) {
      settings.aws_key_type = localInputs.aws_key_type || 'ak_sk';
    }

    // type === 41 (Vertex): 始终保存 vertex_key_type 到 settings，避免编辑时被重置
    if (localInputs.type === 41) {
      settings.vertex_key_type = localInputs.vertex_key_type || 'json';
    } else if ('vertex_key_type' in settings) {
      delete settings.vertex_key_type;
    }

    // type === 1 (OpenAI) 或 type === 14 (Claude): 设置字段透传控制（显式保存布尔值）
    if (localInputs.type === 1 || localInputs.type === 14) {
      settings.allow_service_tier = localInputs.allow_service_tier === true;
      // 仅 OpenAI 渠道需要 store / safety_identifier / include_obfuscation
      if (localInputs.type === 1) {
        settings.disable_store = localInputs.disable_store === true;
        settings.allow_safety_identifier =
          localInputs.allow_safety_identifier === true;
        settings.allow_include_obfuscation =
          localInputs.allow_include_obfuscation === true;
      }
      if (localInputs.type === 14) {
        settings.allow_inference_geo = localInputs.allow_inference_geo === true;
        settings.claude_beta_query = localInputs.claude_beta_query === true;
      }
    }

    settings.upstream_model_update_check_enabled =
      localInputs.upstream_model_update_check_enabled === true;
    settings.upstream_model_update_auto_sync_enabled =
      settings.upstream_model_update_check_enabled &&
      localInputs.upstream_model_update_auto_sync_enabled === true;
    settings.upstream_model_update_ignored_models = Array.from(
      new Set(
        String(localInputs.upstream_model_update_ignored_models || '')
          .split(',')
          .map((model) => model.trim())
          .filter(Boolean),
      ),
    );
    if (
      !Array.isArray(settings.upstream_model_update_last_detected_models) ||
      !settings.upstream_model_update_check_enabled
    ) {
      settings.upstream_model_update_last_detected_models = [];
    }
    if (typeof settings.upstream_model_update_last_check_time !== 'number') {
      settings.upstream_model_update_last_check_time = 0;
    }

    localInputs.settings = JSON.stringify(settings);

    // 清理不需要发送到后端的字段
    delete localInputs.force_format;
    delete localInputs.thinking_to_content;
    delete localInputs.proxy;
    delete localInputs.pass_through_body_enabled;
    delete localInputs.system_prompt;
    delete localInputs.system_prompt_override;
    delete localInputs.is_enterprise_account;
    // 顶层的 vertex_key_type 不应发送给后端
    delete localInputs.vertex_key_type;
    // 顶层的 aws_key_type 不应发送给后端
    delete localInputs.aws_key_type;
    // 清理字段透传控制的临时字段
    delete localInputs.allow_service_tier;
    delete localInputs.disable_store;
    delete localInputs.allow_safety_identifier;
    delete localInputs.allow_include_obfuscation;
    delete localInputs.allow_inference_geo;
    delete localInputs.claude_beta_query;
    delete localInputs.upstream_model_update_check_enabled;
    delete localInputs.upstream_model_update_auto_sync_enabled;
    delete localInputs.upstream_model_update_last_check_time;
    delete localInputs.upstream_model_update_last_detected_models;
    delete localInputs.upstream_model_update_ignored_models;

    let res;
    localInputs.auto_ban = localInputs.auto_ban ? 1 : 0;
    localInputs.models = localInputs.models.join(',');
    localInputs.group = (localInputs.groups || []).join(',');

    let mode = 'single';
    if (batch) {
      mode = multiToSingle ? 'multi_to_single' : 'batch';
    }

    if (isEdit) {
      res = await API.put(`/api/channel/`, {
        ...localInputs,
        id: parseInt(channelId),
        key_mode: isMultiKeyChannel ? keyMode : undefined, // 只在多key模式下传递
      });
    } else {
      res = await API.post(`/api/channel/`, {
        mode: mode,
        multi_key_mode: mode === 'multi_to_single' ? multiKeyMode : undefined,
        channel: localInputs,
      });
    }
    const { success, message } = res.data;
    if (success) {
      if (isEdit) {
        showSuccess(t('渠道更新成功！'));
      } else {
        showSuccess(t('渠道创建成功！'));
        setInputs(originInputs);
      }
      props.refresh();
      props.handleClose();
    } else {
      showError(message);
    }
  };

  // 密钥去重函数
  const deduplicateKeys = () => {
    const currentKey = formApiRef.current?.getValue('key') || inputs.key || '';

    if (!currentKey.trim()) {
      showInfo(t('请先输入密钥'));
      return;
    }

    // 按行分割密钥
    const keyLines = currentKey.split('\n');
    const beforeCount = keyLines.length;

    // 使用哈希表去重，保持原有顺序
    const keySet = new Set();
    const deduplicatedKeys = [];

    keyLines.forEach((line) => {
      const trimmedLine = line.trim();
      if (trimmedLine && !keySet.has(trimmedLine)) {
        keySet.add(trimmedLine);
        deduplicatedKeys.push(trimmedLine);
      }
    });

    const afterCount = deduplicatedKeys.length;
    const deduplicatedKeyText = deduplicatedKeys.join('\n');

    // 更新表单和状态
    if (formApiRef.current) {
      formApiRef.current.setValue('key', deduplicatedKeyText);
    }
    handleInputChange('key', deduplicatedKeyText);

    // 显示去重结果
    const message = t(
      '去重完成：去重前 {{before}} 个密钥，去重后 {{after}} 个密钥',
      {
        before: beforeCount,
        after: afterCount,
      },
    );

    if (beforeCount === afterCount) {
      showInfo(t('未发现重复密钥'));
    } else {
      showSuccess(message);
    }
  };

  const addCustomModels = () => {
    if (customModel.trim() === '') return;
    const modelArray = customModel.split(',').map((model) => model.trim());

    let localModels = [...inputs.models];
    let localModelOptions = [...modelOptions];
    const addedModels = [];

    modelArray.forEach((model) => {
      if (model && !localModels.includes(model)) {
        localModels.push(model);
        localModelOptions.push({
          key: model,
          label: model,
          value: model,
        });
        addedModels.push(model);
      }
    });

    setModelOptions(localModelOptions);
    setCustomModel('');
    handleInputChange('models', localModels);

    if (addedModels.length > 0) {
      showSuccess(
        t('已新增 {{count}} 个模型：{{list}}', {
          count: addedModels.length,
          list: addedModels.join(', '),
        }),
      );
    } else {
      showInfo(t('未发现新增模型'));
    }
  };

  const batchAllowed = (!isEdit || isMultiKeyChannel) && inputs.type !== 57;
  const batchExtra = batchAllowed ? (
    <div className='flex flex-wrap items-center gap-3'>
      {!isEdit && (
        <label className='flex items-center gap-2 text-sm text-white/80'>
          <Checkbox
            disabled={isEdit}
            checked={batch}
            onCheckedChange={(checked) => {
              const nextChecked = !!checked;

              if (!nextChecked && vertexFileList.length > 1) {
                const confirmed = window.confirm(
                  t('将仅保留第一个密钥文件，其余文件将被移除，是否继续？'),
                );
                if (confirmed) {
                  const firstFile = vertexFileList[0];
                  const firstKey = vertexKeys[0] ? [vertexKeys[0]] : [];

                  setVertexFileList([firstFile]);
                  setVertexKeys(firstKey);

                  formApiRef.current?.setValue('vertex_files', [firstFile]);
                  setInputs((prev) => ({ ...prev, vertex_files: [firstFile] }));

                  setBatch(false);
                  setMultiToSingle(false);
                  setMultiKeyMode('random');
                } else {
                  setBatch(true);
                }
                return;
              }

              setBatch(nextChecked);
              if (!nextChecked) {
                setMultiToSingle(false);
                setMultiKeyMode('random');
              } else {
                setUseManualInput(false);
                if (inputs.type === 41) {
                  if (formApiRef.current) {
                    formApiRef.current.setValue('key', '');
                  }
                  handleInputChange('key', '');
                }
              }
            }}
          />
          {t('批量创建')}
        </label>
      )}
      {batch && (
        <>
          <label className='flex items-center gap-2 text-sm text-white/80'>
            <Checkbox
              disabled={isEdit}
              checked={multiToSingle}
              onCheckedChange={() => {
                setMultiToSingle((prev) => {
                  const nextValue = !prev;
                  setInputs((prevInputs) => {
                    const newInputs = { ...prevInputs };
                    if (nextValue) {
                      newInputs.multi_key_mode = multiKeyMode;
                    } else {
                      delete newInputs.multi_key_mode;
                    }
                    return newInputs;
                  });
                  return nextValue;
                });
              }}
            />
            {t('密钥聚合模式')}
          </label>

          {inputs.type !== 41 && (
            <Button
              size='sm'
              type='button'
              variant='outline'
              onClick={deduplicateKeys}
              className='border-white/10 bg-transparent text-white hover:bg-white/10'
            >
              {t('密钥去重')}
            </Button>
          )}
        </>
      )}
    </div>
  ) : null;

  const channelOptionList = useMemo(
    () =>
      CHANNEL_OPTIONS.map((opt) => ({
        ...opt,
        // 保持 label 为纯文本以支持搜索
        label: opt.label,
      })),
    [],
  );

  const renderChannelOption = (renderProps) => {
    const {
      disabled,
      selected,
      label,
      value,
      focused,
      className,
      style,
      onMouseEnter,
      onClick,
      ...rest
    } = renderProps;

    const searchWords = channelSearchValue ? [channelSearchValue] : [];

    // 构建样式类名
    const optionClassName = [
      'flex items-center gap-3 px-3 py-2 transition-all duration-200 rounded-lg mx-2 my-1',
      focused && 'bg-blue-50 shadow-sm',
      selected &&
        'bg-blue-100 text-blue-700 shadow-lg ring-2 ring-blue-200 ring-opacity-50',
      disabled && 'opacity-50 cursor-not-allowed',
      !disabled && 'hover:bg-gray-50 hover:shadow-md cursor-pointer',
      className,
    ]
      .filter(Boolean)
      .join(' ');

    return (
      <div
        style={style}
        className={optionClassName}
        onClick={() => !disabled && onClick()}
        onMouseEnter={(e) => onMouseEnter()}
      >
        <div className='flex items-center gap-3 w-full'>
          <div className='flex-shrink-0 w-5 h-5 flex items-center justify-center'>
            {getChannelIcon(value)}
          </div>
          <div className='flex-1 min-w-0'>
            <Highlight
              sourceString={label}
              searchWords={searchWords}
              className='text-sm font-medium truncate'
            />
          </div>
          {selected && (
            <div className='flex-shrink-0 text-blue-600'>
              <svg
                width='16'
                height='16'
                viewBox='0 0 16 16'
                fill='currentColor'
              >
                <path d='M13.78 4.22a.75.75 0 010 1.06l-7.25 7.25a.75.75 0 01-1.06 0L2.22 9.28a.75.75 0 011.06-1.06L6 10.94l6.72-6.72a.75.75 0 011.06 0z' />
              </svg>
            </div>
          )}
        </div>
      </div>
    );
  };

  return (
    <>
      <Dialog open={props.visible} onOpenChange={(open) => !open && handleCancel()}>
        <DialogContent
          className='max-w-[1100px] border-white/10 bg-black p-0 text-white sm:max-w-[1100px]'
          showCloseButton={false}
        >
          <DialogHeader className='border-b border-white/10 px-6 py-4'>
            <div className='flex items-center gap-3'>
              <Badge className='border-blue-500/20 bg-blue-500/15 text-blue-200'>
                {isEdit ? t('编辑') : t('新建')}
              </Badge>
              <DialogTitle className='text-xl text-white'>
                {isEdit ? t('更新渠道信息') : t('创建新的渠道')}
              </DialogTitle>
            </div>
          </DialogHeader>

          <ScrollArea className='max-h-[78vh] px-6 py-5'>
            <div className='space-y-3' ref={formContainerRef}>
                <div ref={(el) => (formSectionRefs.current.basicInfo = el)}>
                  <Card className='mb-6 rounded-2xl border-white/10 bg-white/5 py-0 shadow-sm'>
                    {/* Header: Basic Info */}
                    <div className='mb-2 flex items-center gap-2'>
                      <SectionIcon
                        icon={Server}
                        className='bg-blue-500/15 text-blue-200'
                      />
                      <div>
                        <div className='text-lg font-medium text-white'>
                          {t('基本信息')}
                        </div>
                        <div className='text-xs text-white/60'>
                          {t('渠道的基本配置信息')}
                        </div>
                      </div>
                    </div>

                    {isIonetChannel && (
                      <NoticeBanner
                        tone='info'
                        description={t(
                          '此渠道由 IO.NET 自动同步，类型、密钥和 API 地址已锁定。',
                        )}
                        actions={
                          ionetMetadata?.deployment_id ? (
                            <Button
                              size='sm'
                              variant='secondary'
                              onClick={handleOpenIonetDeployment}
                            >
                              <Globe className='mr-1 h-4 w-4' />
                              {t('查看关联部署')}
                            </Button>
                          ) : null
                        }
                      />
                    )}

                    <div className='space-y-2'>
                      <FieldLabel>{t('类型')}</FieldLabel>
                      <Select
                        value={String(inputs.type ?? '')}
                        onValueChange={(value) =>
                          handleInputChange('type', Number(value))
                        }
                        disabled={isIonetLocked}
                      >
                        <SelectTrigger className='w-full border-white/10 bg-white/5 text-white'>
                          <SelectValue placeholder={t('请选择渠道类型')} />
                        </SelectTrigger>
                        <SelectContent className='max-h-[320px] border-white/10 bg-[#050816] text-white'>
                          {channelOptionList.map((option) => (
                            <SelectItem
                              key={String(option.value)}
                              value={String(option.value)}
                            >
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    {inputs.type === 57 && (
                      <NoticeBanner
                        tone='warning'
                        description={t(
                          '免责声明：仅限个人使用，请勿分发或共享任何凭证。该渠道存在前置条件与使用门槛，请在充分了解流程与风险后使用，并遵守 OpenAI 的相关条款与政策。相关凭证与配置仅限接入 Codex CLI 使用，不适用于其他客户端、平台或渠道。',
                        )}
                      />
                    )}

                    {inputs.type === 20 && (
                      <div className='space-y-2'>
                        <FieldLabel>{t('是否为企业账户')}</FieldLabel>
                        <div className='flex items-center gap-3'>
                          <Switch
                            checked={!!inputs.is_enterprise_account}
                            onCheckedChange={(value) => {
                              const checked = !!value;
                              setIsEnterpriseAccount(checked);
                              handleInputChange(
                                'is_enterprise_account',
                                checked,
                              );
                            }}
                          />
                          <span className='text-sm text-white/70'>
                            {inputs.is_enterprise_account ? t('是') : t('否')}
                          </span>
                        </div>
                        <FieldHint>
                          {t(
                            '企业账户为特殊返回格式，需要特殊处理，如果非企业账户，请勿勾选',
                          )}
                        </FieldHint>
                      </div>
                    )}

                    <div className='space-y-2'>
                      <FieldLabel>{t('名称')}</FieldLabel>
                      <Input
                        value={inputs.name || ''}
                        placeholder={t('请为渠道命名')}
                        onChange={(event) =>
                          handleInputChange('name', event.target.value)
                        }
                        autoComplete='new-password'
                        className='border-white/10 bg-white/5 text-white'
                      />
                    </div>

                    {inputs.type === 33 && (
                      <div className='space-y-2'>
                        <FieldLabel>{t('密钥格式')}</FieldLabel>
                        <Select
                          value={inputs.aws_key_type || 'ak_sk'}
                          onValueChange={(value) =>
                            handleChannelOtherSettingsChange(
                              'aws_key_type',
                              value,
                            )
                          }
                        >
                          <SelectTrigger className='w-full border-white/10 bg-white/5 text-white'>
                            <SelectValue placeholder={t('请选择密钥格式')} />
                          </SelectTrigger>
                          <SelectContent className='border-white/10 bg-[#050816] text-white'>
                            <SelectItem value='ak_sk'>
                              AccessKey / SecretAccessKey
                            </SelectItem>
                            <SelectItem value='api_key'>API Key</SelectItem>
                          </SelectContent>
                        </Select>
                        <FieldHint>
                          {t(
                            'AK/SK 模式：使用 AccessKey 和 SecretAccessKey；API Key 模式：使用 API Key',
                          )}
                        </FieldHint>
                      </div>
                    )}

                    {inputs.type === 41 && (
                      <div className='space-y-2'>
                        <FieldLabel>{t('密钥格式')}</FieldLabel>
                        <Select
                          value={inputs.vertex_key_type || 'json'}
                          onValueChange={(value) => {
                            handleChannelOtherSettingsChange(
                              'vertex_key_type',
                              value,
                            );
                            if (value === 'api_key') {
                              setBatch(false);
                              setUseManualInput(false);
                              setVertexKeys([]);
                              setVertexFileList([]);
                              if (formApiRef.current) {
                                formApiRef.current.setValue('vertex_files', []);
                              }
                            }
                          }}
                        >
                          <SelectTrigger className='w-full border-white/10 bg-white/5 text-white'>
                            <SelectValue placeholder={t('请选择密钥格式')} />
                          </SelectTrigger>
                          <SelectContent className='border-white/10 bg-[#050816] text-white'>
                            <SelectItem value='json'>JSON</SelectItem>
                            <SelectItem value='api_key'>API Key</SelectItem>
                          </SelectContent>
                        </Select>
                        <FieldHint>
                          {inputs.vertex_key_type === 'api_key'
                            ? t('API Key 模式下不支持批量创建')
                            : t('JSON 模式支持手动输入或上传服务账号 JSON')}
                        </FieldHint>
                      </div>
                    )}
                    {batch ? (
                      inputs.type === 41 &&
                      (inputs.vertex_key_type || 'json') === 'json' ? (
                        <div className='space-y-2'>
                          <FieldLabel>{t('密钥文件 (.json)')}</FieldLabel>
                          <UploadPlaceholder
                            title={t('点击上传文件或拖拽文件到这里')}
                            subtitle={t('仅支持 JSON 文件，支持多文件')}
                          />
                          <Input
                            type='file'
                            accept='.json'
                            multiple
                            onChange={(event) => {
                              const files = Array.from(event.target.files || []);
                              const fileList = files.map((file) => ({
                                uid: `${file.name}-${file.lastModified}`,
                                name: file.name,
                                fileInstance: file,
                              }));
                              handleVertexUploadChange({ fileList });
                            }}
                            className='border-white/10 bg-white/5 text-white file:mr-3 file:rounded-md file:border-0 file:bg-white file:px-3 file:py-1.5 file:text-black'
                          />
                          {vertexFileList.length > 0 && (
                            <div className='space-y-1 rounded-lg border border-white/10 bg-black/20 p-3 text-xs text-white/70'>
                              {vertexFileList.map((fileItem, index) => (
                                <div key={fileItem.uid || `${fileItem.name}-${index}`}>
                                  {fileItem.name}
                                </div>
                              ))}
                            </div>
                          )}
                          {batchExtra}
                        </div>
                      ) : (
                        <div className='space-y-2'>
                          <FieldLabel>{t('密钥')}</FieldLabel>
                          <Textarea
                            value={inputs.key || ''}
                            placeholder={
                              inputs.type === 33
                                ? inputs.aws_key_type === 'api_key'
                                  ? t(
                                      '请输入 API Key，一行一个，格式：APIKey|Region',
                                    )
                                  : t(
                                      '请输入密钥，一行一个，格式：AccessKey|SecretAccessKey|Region',
                                    )
                                : t('请输入密钥，一行一个')
                            }
                            autoComplete='new-password'
                            onChange={(event) =>
                              handleInputChange('key', event.target.value)
                            }
                            disabled={isIonetLocked}
                            className='min-h-[140px] border-white/10 bg-white/5 text-white'
                          />
                          <div className='flex flex-wrap items-center gap-2'>
                            {isEdit &&
                              isMultiKeyChannel &&
                              keyMode === 'append' && (
                                <FieldHint tone='warning'>
                                  {t(
                                    '追加模式：新密钥将添加到现有密钥列表的末尾',
                                  )}
                                </FieldHint>
                              )}
                            {isEdit && (
                              <Button
                                size='sm'
                                type='button'
                                variant='outline'
                                onClick={handleShow2FAModal}
                              >
                                {t('查看密钥')}
                              </Button>
                            )}
                            {batchExtra}
                          </div>
                        </div>
                      )
                    ) : (
                      <>
                        {inputs.type === 57 ? (
                          <>
                            <div className='space-y-2'>
                              <FieldLabel>
                                {isEdit
                                  ? t('密钥（编辑模式下，保存的密钥不会显示）')
                                  : t('密钥')}
                              </FieldLabel>
                              <Textarea
                                value={inputs.key || ''}
                                placeholder={t(
                                  '请输入 JSON 格式的 OAuth 凭据，例如：\n{\n  "access_token": "...",\n  "account_id": "..." \n}',
                                )}
                                autoComplete='new-password'
                                onChange={(event) =>
                                  handleInputChange('key', event.target.value)
                                }
                                disabled={isIonetLocked}
                                className='min-h-[180px] border-white/10 bg-white/5 text-white'
                              />
                              <div className='flex flex-col gap-2'>
                                <FieldHint>
                                  {t(
                                    '仅支持 JSON 对象，必须包含 access_token 与 account_id',
                                  )}
                                </FieldHint>

                                <div className='flex flex-wrap gap-2'>
                                  <Button
                                    size='sm'
                                    type='button'
                                    variant='outline'
                                    onClick={() =>
                                      setCodexOAuthModalVisible(true)
                                    }
                                    disabled={isIonetLocked}
                                  >
                                    {t('Codex 授权')}
                                  </Button>
                                  {isEdit && (
                                    <Button
                                      size='sm'
                                      type='button'
                                      variant='outline'
                                      onClick={handleRefreshCodexCredential}
                                      disabled={isIonetLocked}
                                    >
                                      {t('刷新凭证')}
                                    </Button>
                                  )}
                                  <Button
                                    size='sm'
                                    type='button'
                                    variant='outline'
                                    onClick={() => formatJsonField('key')}
                                    disabled={isIonetLocked}
                                  >
                                    {t('格式化')}
                                  </Button>
                                  {isEdit && (
                                    <Button
                                      size='sm'
                                      type='button'
                                      variant='outline'
                                      onClick={handleShow2FAModal}
                                      disabled={isIonetLocked}
                                    >
                                      {t('查看密钥')}
                                    </Button>
                                  )}
                                  {batchExtra}
                                </div>
                              </div>
                            </div>

                            <CodexOAuthModal
                              visible={codexOAuthModalVisible}
                              onCancel={() => setCodexOAuthModalVisible(false)}
                              onSuccess={handleCodexOAuthGenerated}
                            />
                          </>
                        ) : inputs.type === 41 &&
                          (inputs.vertex_key_type || 'json') === 'json' ? (
                          <>
                            {!batch && (
                              <div className='mb-3 flex items-center justify-between gap-3'>
                                <div className='text-sm font-medium text-white'>
                                  {t('密钥输入方式')}
                                </div>
                                <div className='flex gap-2'>
                                  <Button
                                    size='sm'
                                    type='button'
                                    variant={
                                      !useManualInput ? 'default' : 'outline'
                                    }
                                    onClick={() => {
                                      setUseManualInput(false);
                                      if (formApiRef.current) {
                                        formApiRef.current.setValue('key', '');
                                      }
                                      handleInputChange('key', '');
                                    }}
                                  >
                                    {t('文件上传')}
                                  </Button>
                                  <Button
                                    size='sm'
                                    type='button'
                                    variant={
                                      useManualInput ? 'default' : 'outline'
                                    }
                                    onClick={() => {
                                      setUseManualInput(true);
                                      setVertexKeys([]);
                                      setVertexFileList([]);
                                      if (formApiRef.current) {
                                        formApiRef.current.setValue(
                                          'vertex_files',
                                          [],
                                        );
                                      }
                                      setInputs((prev) => ({
                                        ...prev,
                                        vertex_files: [],
                                      }));
                                    }}
                                  >
                                    {t('手动输入')}
                                  </Button>
                                </div>
                              </div>
                            )}

                            {batch && (
                              <NoticeBanner
                                tone='info'
                                description={t(
                                  '批量创建模式下仅支持文件上传，不支持手动输入',
                                )}
                              />
                            )}

                            {useManualInput && !batch ? (
                              <div className='space-y-2'>
                                <FieldLabel>
                                  {isEdit
                                    ? t(
                                        '密钥（编辑模式下，保存的密钥不会显示）',
                                      )
                                    : t('密钥')}
                                </FieldLabel>
                                <Textarea
                                  value={inputs.key || ''}
                                  placeholder={t(
                                    '请输入 JSON 格式的密钥内容，例如：\n{\n  "type": "service_account",\n  "project_id": "your-project-id",\n  "private_key_id": "...",\n  "private_key": "...",\n  "client_email": "...",\n  "client_id": "...",\n  "auth_uri": "...",\n  "token_uri": "...",\n  "auth_provider_x509_cert_url": "...",\n  "client_x509_cert_url": "..."\n}',
                                  )}
                                  autoComplete='new-password'
                                  onChange={(event) =>
                                    handleInputChange('key', event.target.value)
                                  }
                                  className='min-h-[220px] border-white/10 bg-white/5 text-white'
                                />
                                <div className='flex flex-wrap items-center gap-2'>
                                  <FieldHint>
                                    {t('请输入完整的 JSON 格式密钥内容')}
                                  </FieldHint>
                                  {isEdit &&
                                    isMultiKeyChannel &&
                                    keyMode === 'append' && (
                                      <FieldHint tone='warning'>
                                        {t(
                                          '追加模式：新密钥将添加到现有密钥列表的末尾',
                                        )}
                                      </FieldHint>
                                    )}
                                  {isEdit && (
                                    <Button
                                      size='sm'
                                      type='button'
                                      variant='outline'
                                      onClick={handleShow2FAModal}
                                    >
                                      {t('查看密钥')}
                                    </Button>
                                  )}
                                  {batchExtra}
                                </div>
                              </div>
                            ) : (
                              <div className='space-y-2'>
                                <FieldLabel>{t('密钥文件 (.json)')}</FieldLabel>
                                <UploadPlaceholder
                                  title={t('点击上传文件或拖拽文件到这里')}
                                  subtitle={t('仅支持 JSON 文件')}
                                />
                                <Input
                                  type='file'
                                  accept='.json'
                                  onChange={(event) => {
                                    const files = Array.from(
                                      event.target.files || [],
                                    );
                                    const fileList = files.map((file) => ({
                                      uid: `${file.name}-${file.lastModified}`,
                                      name: file.name,
                                      fileInstance: file,
                                    }));
                                    handleVertexUploadChange({ fileList });
                                  }}
                                  className='border-white/10 bg-white/5 text-white file:mr-3 file:rounded-md file:border-0 file:bg-white file:px-3 file:py-1.5 file:text-black'
                                />
                                {vertexFileList.length > 0 && (
                                  <div className='space-y-1 rounded-lg border border-white/10 bg-black/20 p-3 text-xs text-white/70'>
                                    {vertexFileList.map((fileItem, index) => (
                                      <div
                                        key={
                                          fileItem.uid ||
                                          `${fileItem.name}-${index}`
                                        }
                                      >
                                        {fileItem.name}
                                      </div>
                                    ))}
                                  </div>
                                )}
                                {batchExtra}
                              </div>
                            )}
                          </>
                        ) : (
                          <div className='space-y-2'>
                            <FieldLabel>
                              {isEdit
                                ? t('密钥（编辑模式下，保存的密钥不会显示）')
                                : t('密钥')}
                            </FieldLabel>
                            <Input
                              value={inputs.key || ''}
                              placeholder={
                                inputs.type === 33
                                  ? inputs.aws_key_type === 'api_key'
                                    ? t('请输入 API Key，格式：APIKey|Region')
                                    : t(
                                        '按照如下格式输入：AccessKey|SecretAccessKey|Region',
                                      )
                                  : t(type2secretPrompt(inputs.type))
                              }
                              autoComplete='new-password'
                              onChange={(event) =>
                                handleInputChange('key', event.target.value)
                              }
                              className='border-white/10 bg-white/5 text-white'
                            />
                            <div className='flex flex-wrap items-center gap-2'>
                              {isEdit &&
                                isMultiKeyChannel &&
                                keyMode === 'append' && (
                                  <FieldHint tone='warning'>
                                    {t(
                                      '追加模式：新密钥将添加到现有密钥列表的末尾',
                                    )}
                                  </FieldHint>
                                )}
                              {isEdit && (
                                <Button
                                  size='sm'
                                  type='button'
                                  variant='outline'
                                  onClick={handleShow2FAModal}
                                >
                                  {t('查看密钥')}
                                </Button>
                              )}
                              {batchExtra}
                            </div>
                          </div>
                        )}
                      </>
                    )}

                    {isEdit && isMultiKeyChannel && (
                      <div className='space-y-2'>
                        <FieldLabel>{t('密钥更新模式')}</FieldLabel>
                        <Select value={keyMode} onValueChange={setKeyMode}>
                          <SelectTrigger className='w-full border-white/10 bg-white/5 text-white'>
                            <SelectValue placeholder={t('请选择密钥更新模式')} />
                          </SelectTrigger>
                          <SelectContent className='border-white/10 bg-[#050816] text-white'>
                            <SelectItem value='append'>
                              {t('追加到现有密钥')}
                            </SelectItem>
                            <SelectItem value='replace'>
                              {t('覆盖现有密钥')}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                        <FieldHint>
                          {keyMode === 'replace'
                            ? t('覆盖模式：将完全替换现有的所有密钥')
                            : t('追加模式：将新密钥添加到现有密钥列表末尾')}
                        </FieldHint>
                      </div>
                    )}
                    {batch && multiToSingle && (
                      <>
                        <div className='space-y-2'>
                          <FieldLabel>{t('密钥聚合模式')}</FieldLabel>
                          <Select
                            value={inputs.multi_key_mode || 'random'}
                            onValueChange={(value) => {
                              setMultiKeyMode(value);
                              handleInputChange('multi_key_mode', value);
                            }}
                          >
                            <SelectTrigger className='w-full border-white/10 bg-white/5 text-white'>
                              <SelectValue
                                placeholder={t('请选择多密钥使用策略')}
                              />
                            </SelectTrigger>
                            <SelectContent className='border-white/10 bg-[#050816] text-white'>
                              <SelectItem value='random'>{t('随机')}</SelectItem>
                              <SelectItem value='polling'>{t('轮询')}</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        {inputs.multi_key_mode === 'polling' && (
                          <NoticeBanner
                            tone='warning'
                            description={t(
                              '轮询模式必须搭配Redis和内存缓存功能使用，否则性能将大幅降低，并且无法实现轮询功能',
                            )}
                            className='mt-2'
                          />
                        )}
                      </>
                    )}

                    {inputs.type === 18 && (
                      <div className='space-y-2'>
                        <FieldLabel>{t('模型版本')}</FieldLabel>
                        <Input
                          value={inputs.other || ''}
                          placeholder='请输入星火大模型版本，注意是接口地址中的版本号，例如：v2.1'
                          onChange={(event) =>
                            handleInputChange('other', event.target.value)
                          }
                          className='border-white/10 bg-white/5 text-white'
                        />
                      </div>
                    )}

                    {inputs.type === 41 && (
                      <JSONEditor
                        key={`region-${isEdit ? channelId : 'new'}`}
                        field='other'
                        label={t('部署地区')}
                        placeholder={t(
                          '请输入部署地区，例如：us-central1\n支持使用模型映射格式\n{\n    "default": "us-central1",\n    "claude-3-5-sonnet-20240620": "europe-west1"\n}',
                        )}
                        value={inputs.other || ''}
                        onChange={(value) => handleInputChange('other', value)}
                        rules={[
                          { required: true, message: t('请填写部署地区') },
                        ]}
                        template={REGION_EXAMPLE}
                        templateLabel={t('填入模板')}
                        editorType='region'
                        formApi={formApiRef.current}
                        extraText={t('设置默认地区和特定模型的专用地区')}
                      />
                    )}

                    {inputs.type === 21 && (
                      <div className='space-y-2'>
                        <FieldLabel>{t('知识库 ID')}</FieldLabel>
                        <Input
                          value={inputs.other || ''}
                          placeholder='请输入知识库 ID，例如：123456'
                          onChange={(event) =>
                            handleInputChange('other', event.target.value)
                          }
                          className='border-white/10 bg-white/5 text-white'
                        />
                      </div>
                    )}

                    {inputs.type === 39 && (
                      <div className='space-y-2'>
                        <FieldLabel>Account ID</FieldLabel>
                        <Input
                          value={inputs.other || ''}
                          placeholder='请输入Account ID，例如：d6b5da8hk1awo8nap34ube6gh'
                          onChange={(event) =>
                            handleInputChange('other', event.target.value)
                          }
                          className='border-white/10 bg-white/5 text-white'
                        />
                      </div>
                    )}

                    {inputs.type === 49 && (
                      <div className='space-y-2'>
                        <FieldLabel>{t('智能体ID')}</FieldLabel>
                        <Input
                          value={inputs.other || ''}
                          placeholder='请输入智能体ID，例如：7342866812345'
                          onChange={(event) =>
                            handleInputChange('other', event.target.value)
                          }
                          className='border-white/10 bg-white/5 text-white'
                        />
                      </div>
                    )}

                    {inputs.type === 1 && (
                      <div className='space-y-2'>
                        <FieldLabel>{t('组织')}</FieldLabel>
                        <Input
                          value={inputs.openai_organization || ''}
                          placeholder={t('请输入组织org-xxx')}
                          onChange={(event) =>
                            handleInputChange(
                              'openai_organization',
                              event.target.value,
                            )
                          }
                          className='border-white/10 bg-white/5 text-white'
                        />
                        <FieldHint>{t('组织，不填则为默认组织')}</FieldHint>
                      </div>
                    )}
                  </Card>
                </div>

                {/* API Configuration Card */}
                {showApiConfigCard && (
                  <div ref={(el) => (formSectionRefs.current.apiConfig = el)}>
                    <Card className='mb-6 rounded-2xl border-white/10 bg-white/5 py-0 shadow-sm'>
                      {/* Header: API Config */}
                      <div
                        className='mb-2 flex items-center gap-2'
                        onClick={handleApiConfigSecretClick}
                      >
                        <SectionIcon
                          icon={Globe}
                          className='bg-emerald-500/15 text-emerald-200'
                        />
                        <div>
                          <div className='text-lg font-medium text-white'>
                            {t('API 配置')}
                          </div>
                          <div className='text-xs text-white/60'>
                            {t('API 地址和相关配置')}
                          </div>
                        </div>
                      </div>

                      {inputs.type === 40 && (
                        <NoticeBanner
                          tone='info'
                          description={
                            <div className='text-sm leading-6'>
                              <span className='font-semibold'>
                                {t('邀请链接')}:
                              </span>
                              <button
                                type='button'
                                className='ml-2 underline underline-offset-4'
                                onClick={() =>
                                  window.open(
                                    'https://cloud.siliconflow.cn/i/hij0YNTZ',
                                  )
                                }
                              >
                                https://cloud.siliconflow.cn/i/hij0YNTZ
                              </button>
                            </div>
                          }
                        />
                      )}

                      {inputs.type === 3 && (
                        <>
                          <NoticeBanner
                            tone='warning'
                            description={t(
                              '2025年5月10日后添加的渠道，不需要再在部署的时候移除模型名称中的"."',
                            )}
                          />
                          <div className='space-y-2'>
                            <FieldLabel>AZURE_OPENAI_ENDPOINT</FieldLabel>
                            <Input
                              value={inputs.base_url || ''}
                              placeholder={t(
                                '请输入 AZURE_OPENAI_ENDPOINT，例如：https://docs-test-001.openai.azure.com',
                              )}
                              onChange={(event) =>
                                handleInputChange(
                                  'base_url',
                                  event.target.value,
                                )
                              }
                              disabled={isIonetLocked}
                              className='border-white/10 bg-white/5 text-white'
                            />
                          </div>
                          <div className='space-y-2'>
                            <FieldLabel>{t('默认 API 版本')}</FieldLabel>
                            <Input
                              value={inputs.other || ''}
                              placeholder={t(
                                '请输入默认 API 版本，例如：2025-04-01-preview',
                              )}
                              onChange={(event) =>
                                handleInputChange('other', event.target.value)
                              }
                              className='border-white/10 bg-white/5 text-white'
                            />
                          </div>
                          <div className='space-y-2'>
                            <FieldLabel>
                              {t('默认 Responses API 版本，为空则使用上方版本')}
                            </FieldLabel>
                            <Input
                              value={readJsonSetting(
                                inputs.settings,
                                'azure_responses_version',
                                '',
                              )}
                              placeholder={t('例如：preview')}
                              onChange={(event) =>
                                handleChannelOtherSettingsChange(
                                  'azure_responses_version',
                                  event.target.value,
                                )
                              }
                              className='border-white/10 bg-white/5 text-white'
                            />
                          </div>
                        </>
                      )}

                      {inputs.type === 8 && (
                        <>
                          <NoticeBanner
                            tone='warning'
                            description={t(
                              '如果你对接的是上游 One API、OpenCrab 或其他兼容转发项目，请使用 OpenAI 类型，不要使用此类型，除非你知道你在做什么。',
                            )}
                          />
                          <div className='space-y-2'>
                            <FieldLabel>
                              {t('完整的 Base URL，支持变量{model}')}
                            </FieldLabel>
                            <Input
                              value={inputs.base_url || ''}
                              placeholder={t(
                                '请输入完整的URL，例如：https://api.openai.com/v1/chat/completions',
                              )}
                              onChange={(event) =>
                                handleInputChange(
                                  'base_url',
                                  event.target.value,
                                )
                              }
                              disabled={isIonetLocked}
                              className='border-white/10 bg-white/5 text-white'
                            />
                          </div>
                        </>
                      )}

                      {inputs.type === 37 && (
                        <NoticeBanner
                          tone='warning'
                          description={t(
                            'Dify渠道只适配chatflow和agent，并且agent不支持图片！',
                          )}
                        />
                      )}

                      {inputs.type !== 3 &&
                        inputs.type !== 8 &&
                        inputs.type !== 22 &&
                        inputs.type !== 36 &&
                        (inputs.type !== 45 || doubaoApiEditUnlocked) && (
                          <div className='space-y-2'>
                            <FieldLabel>{t('API地址')}</FieldLabel>
                            <Input
                              value={inputs.base_url || ''}
                              placeholder={t(
                                '此项可选，用于通过自定义API地址来进行 API 调用，末尾不要带/v1和/',
                              )}
                              onChange={(event) =>
                                handleInputChange(
                                  'base_url',
                                  event.target.value,
                                )
                              }
                              disabled={isIonetLocked}
                              className='border-white/10 bg-white/5 text-white'
                            />
                            <FieldHint>
                              {t(
                                '对于官方渠道，opencrab已经内置地址，除非是第三方代理站点或者Azure的特殊接入地址，否则不需要填写',
                              )}
                            </FieldHint>
                          </div>
                        )}

                      {inputs.type === 22 && (
                        <div className='space-y-2'>
                          <FieldLabel>{t('私有部署地址')}</FieldLabel>
                          <Input
                            value={inputs.base_url || ''}
                            placeholder={t(
                              '请输入私有部署地址，格式为：https://fastgpt.run/api/openapi',
                            )}
                            onChange={(event) =>
                              handleInputChange('base_url', event.target.value)
                            }
                            disabled={isIonetLocked}
                            className='border-white/10 bg-white/5 text-white'
                          />
                        </div>
                      )}

                      {inputs.type === 36 && (
                        <div className='space-y-2'>
                          <FieldLabel>
                            {t(
                              '注意非Chat API，请务必填写正确的API地址，否则可能导致无法使用',
                            )}
                          </FieldLabel>
                          <Input
                            value={inputs.base_url || ''}
                            placeholder={t(
                              '请输入到 /suno 前的路径，通常就是域名，例如：https://api.example.com',
                            )}
                            onChange={(event) =>
                              handleInputChange('base_url', event.target.value)
                            }
                            disabled={isIonetLocked}
                            className='border-white/10 bg-white/5 text-white'
                          />
                        </div>
                      )}

                      {inputs.type === 45 && !doubaoApiEditUnlocked && (
                        <div className='space-y-2'>
                          <FieldLabel>{t('API地址')}</FieldLabel>
                          <Select
                            value={
                              inputs.base_url ||
                              'https://ark.cn-beijing.volces.com'
                            }
                            onValueChange={(value) =>
                              handleInputChange('base_url', value)
                            }
                            disabled={isIonetLocked}
                          >
                            <SelectTrigger className='w-full border-white/10 bg-white/5 text-white'>
                              <SelectValue placeholder={t('请选择API地址')} />
                            </SelectTrigger>
                            <SelectContent className='border-white/10 bg-[#050816] text-white'>
                              <SelectItem value='https://ark.cn-beijing.volces.com'>
                                https://ark.cn-beijing.volces.com
                              </SelectItem>
                              <SelectItem value='https://ark.ap-southeast.bytepluses.com'>
                                https://ark.ap-southeast.bytepluses.com
                              </SelectItem>
                              <SelectItem value='doubao-coding-plan'>
                                Doubao Coding Plan
                              </SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      )}
                    </Card>
                  </div>
                )}

                {/* Model Configuration Card */}
                <div ref={(el) => (formSectionRefs.current.modelConfig = el)}>
                  <Card className='mb-6 rounded-2xl border-white/10 bg-white/5 py-0 shadow-sm'>
                    {/* Header: Model Config */}
                    <div className='mb-2 flex items-center gap-2'>
                      <SectionIcon
                        icon={Code2}
                        className='bg-violet-500/15 text-violet-200'
                      />
                      <div>
                        <div className='text-lg font-medium text-white'>
                          {t('模型配置')}
                        </div>
                        <div className='text-xs text-white/60'>
                          {t('模型选择和映射设置')}
                        </div>
                      </div>
                    </div>

                    <Form.Select
                      field='models'
                      label={t('模型')}
                      placeholder={t('请选择该渠道所支持的模型')}
                      rules={[{ required: true, message: t('请选择模型') }]}
                      multiple
                      filter={selectFilter}
                      allowCreate
                      autoClearSearchValue={false}
                      searchPosition='dropdown'
                      optionList={modelOptions}
                      onSearch={(value) => setModelSearchValue(value)}
                      innerBottomSlot={
                        modelSearchHintText ? (
                          <Text className='px-3 py-2 block text-xs !text-semi-color-text-2'>
                            {modelSearchHintText}
                          </Text>
                        ) : null
                      }
                      style={{ width: '100%' }}
                      onChange={(value) => handleInputChange('models', value)}
                      renderSelectedItem={(optionNode) => {
                        const modelName = String(optionNode?.value ?? '');
                        return {
                          isRenderInTag: true,
                          content: (
                            <span
                              className='cursor-pointer select-none'
                              role='button'
                              tabIndex={0}
                              title={t('点击复制模型名称')}
                              onClick={async (e) => {
                                e.stopPropagation();
                                const ok = await copy(modelName);
                                if (ok) {
                                  showSuccess(
                                    t('已复制：{{name}}', { name: modelName }),
                                  );
                                } else {
                                  showError(t('复制失败'));
                                }
                              }}
                            >
                              {optionNode.label || modelName}
                            </span>
                          ),
                        };
                      }}
                      extraText={
                        <div className='flex flex-wrap gap-2'>
                          <Button
                            size='sm'
                            type='button'
                            onClick={() =>
                              handleInputChange('models', basicModels)
                            }
                          >
                            {t('填入相关模型')}
                          </Button>
                          <Button
                            size='sm'
                            type='button'
                            variant='secondary'
                            onClick={() =>
                              handleInputChange('models', fullModels)
                            }
                          >
                            {t('填入所有模型')}
                          </Button>
                          {MODEL_FETCHABLE_CHANNEL_TYPES.has(inputs.type) && (
                            <Button
                              size='sm'
                              type='button'
                              variant='outline'
                              onClick={() => fetchUpstreamModelList('models')}
                            >
                              {t('获取模型列表')}
                            </Button>
                          )}
                          {inputs.type === 4 && isEdit && (
                            <Button
                              size='sm'
                              type='button'
                              variant='secondary'
                              onClick={() => setOllamaModalVisible(true)}
                            >
                              {t('Ollama 模型管理')}
                            </Button>
                          )}
                          <Button
                            size='sm'
                            type='button'
                            variant='destructive'
                            onClick={() => handleInputChange('models', [])}
                          >
                            {t('清除所有模型')}
                          </Button>
                          <Button
                            size='sm'
                            type='button'
                            variant='outline'
                            onClick={() => {
                              if (inputs.models.length === 0) {
                                showInfo(t('没有模型可以复制'));
                                return;
                              }
                              try {
                                copy(inputs.models.join(','));
                                showSuccess(t('模型列表已复制到剪贴板'));
                              } catch (error) {
                                showError(t('复制失败'));
                              }
                            }}
                          >
                            {t('复制所有模型')}
                          </Button>
                          {modelGroups &&
                            modelGroups.length > 0 &&
                            modelGroups.map((group) => (
                              <Button
                                key={group.id}
                                size='sm'
                                type='button'
                                onClick={() => {
                                  let items = [];
                                  try {
                                    if (Array.isArray(group.items)) {
                                      items = group.items;
                                    } else if (
                                      typeof group.items === 'string'
                                    ) {
                                      const parsed = JSON.parse(
                                        group.items || '[]',
                                      );
                                      if (Array.isArray(parsed)) items = parsed;
                                    }
                                  } catch {}
                                  const current =
                                    formApiRef.current?.getValue('models') ||
                                    inputs.models ||
                                    [];
                                  const merged = Array.from(
                                    new Set(
                                      [...current, ...items]
                                        .map((m) => (m || '').trim())
                                        .filter(Boolean),
                                    ),
                                  );
                                  handleInputChange('models', merged);
                                }}
                              >
                                {group.name}
                              </Button>
                            ))}
                        </div>
                      }
                    />

                    <div className='space-y-2'>
                      <FieldLabel>{t('自定义模型名称')}</FieldLabel>
                      <div className='flex gap-2'>
                        <Input
                          value={customModel}
                          placeholder={t('输入自定义模型名称')}
                          onChange={(event) =>
                            setCustomModel(event.target.value.trim())
                          }
                          className='border-white/10 bg-white/5 text-white'
                        />
                        <Button size='sm' type='button' onClick={addCustomModels}>
                          {t('填入')}
                        </Button>
                      </div>
                    </div>

                    {MODEL_FETCHABLE_CHANNEL_TYPES.has(inputs.type) && (
                      <>
                        <Form.Switch
                          field='upstream_model_update_check_enabled'
                          label={t('是否检测上游模型更新')}
                          checkedText={t('开')}
                          uncheckedText={t('关')}
                          onChange={(value) =>
                            handleChannelOtherSettingsChange(
                              'upstream_model_update_check_enabled',
                              value,
                            )
                          }
                          extraText={t(
                            '开启后由后端定时任务检测该渠道上游模型变化',
                          )}
                        />
                        <div className='text-xs text-gray-500 mb-2'>
                          {t('上次检测时间')}:&nbsp;
                          {formatUnixTime(
                            inputs.upstream_model_update_last_check_time,
                          )}
                        </div>
                        <Form.Input
                          field='upstream_model_update_ignored_models'
                          label={t('已忽略模型')}
                          placeholder={t('例如：gpt-4.1-nano,gpt-4o-mini')}
                          onChange={(value) =>
                            handleInputChange(
                              'upstream_model_update_ignored_models',
                              value,
                            )
                          }
                          showClear
                        />
                      </>
                    )}

                    <Form.Input
                      field='test_model'
                      label={t('默认测试模型')}
                      placeholder={t('不填则为模型列表第一个')}
                      onChange={(value) =>
                        handleInputChange('test_model', value)
                      }
                      showClear
                    />

                    <JSONEditor
                      key={`model_mapping-${isEdit ? channelId : 'new'}`}
                      field='model_mapping'
                      label={t('模型重定向')}
                      placeholder={
                        t(
                          '此项可选，用于修改请求体中的模型名称，为一个 JSON 字符串，键为请求中模型名称，值为要替换的模型名称，例如：',
                        ) +
                        `\n${JSON.stringify(MODEL_MAPPING_EXAMPLE, null, 2)}`
                      }
                      value={inputs.model_mapping || ''}
                      onChange={(value) =>
                        handleInputChange('model_mapping', value)
                      }
                      template={MODEL_MAPPING_EXAMPLE}
                      templateLabel={t('填入模板')}
                      editorType='keyValue'
                      formApi={formApiRef.current}
                      renderStringValueSuffix={({ pairKey, value }) => {
                        if (!MODEL_FETCHABLE_CHANNEL_TYPES.has(inputs.type)) {
                          return null;
                        }
                        const disabled = !String(pairKey ?? '').trim();
                        return (
                          <SemiTooltip content={t('选择模型')}>
                            <span>
                              <Button
                                type='button'
                                variant='ghost'
                                size='icon'
                                disabled={disabled}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  openModelMappingValueModal({ pairKey, value });
                                }}
                              >
                                <IconSearch size={14} />
                              </Button>
                            </span>
                          </SemiTooltip>
                        );
                      }}
                      extraText={t(
                        '键为请求中的模型名称，值为要替换的模型名称',
                      )}
                    />
                  </Card>
                </div>

                {/* Advanced Settings Card */}
                <div
                  ref={(el) => (formSectionRefs.current.advancedSettings = el)}
                >
                  <Card className='mb-6 rounded-2xl border-white/10 bg-white/5 py-0 shadow-sm'>
                    {/* Header: Advanced Settings */}
                    <div className='mb-2 flex items-center gap-2'>
                      <SectionIcon
                        icon={Settings}
                        className='bg-amber-500/15 text-amber-200'
                      />
                      <div>
                        <div className='text-lg font-medium text-white'>
                          {t('高级设置')}
                        </div>
                        <div className='text-xs text-white/60'>
                          {t('渠道的高级配置选项')}
                        </div>
                      </div>
                    </div>

                    <div className='space-y-2'>
                      <FieldLabel>{t('分组')}</FieldLabel>
                      <Textarea
                        value={Array.isArray(inputs.groups) ? inputs.groups.join(',') : ''}
                        placeholder={t('请选择可以使用该渠道的分组')}
                        onChange={(event) =>
                          handleInputChange(
                            'groups',
                            event.target.value
                              .split(',')
                              .map((item) => item.trim())
                              .filter(Boolean),
                          )
                        }
                        className='min-h-[88px] border-white/10 bg-white/5 text-white'
                      />
                      <FieldHint>
                        {t('请在系统设置页面编辑分组倍率以添加新的分组：')}
                      </FieldHint>
                    </div>

                    <div className='space-y-2'>
                      <FieldLabel>{t('渠道标签')}</FieldLabel>
                      <Input
                        value={inputs.tag || ''}
                        placeholder={t('渠道标签')}
                        onChange={(event) =>
                          handleInputChange('tag', event.target.value)
                        }
                        className='border-white/10 bg-white/5 text-white'
                      />
                    </div>
                    <div className='space-y-2'>
                      <FieldLabel>{t('备注')}</FieldLabel>
                      <Textarea
                        value={inputs.remark || ''}
                        placeholder={t('请输入备注（仅管理员可见）')}
                        maxLength={255}
                        onChange={(event) =>
                          handleInputChange('remark', event.target.value)
                        }
                        className='min-h-[100px] border-white/10 bg-white/5 text-white'
                      />
                    </div>

                    <div className='grid gap-4 md:grid-cols-2'>
                      <div className='space-y-2'>
                        <FieldLabel>{t('渠道优先级')}</FieldLabel>
                        <Input
                          type='number'
                          min={0}
                          value={inputs.priority ?? 0}
                          placeholder={t('渠道优先级')}
                          onChange={(event) =>
                            handleInputChange(
                              'priority',
                              Number(event.target.value || 0),
                            )
                          }
                          className='border-white/10 bg-white/5 text-white'
                        />
                      </div>
                      <div className='space-y-2'>
                        <FieldLabel>{t('渠道权重')}</FieldLabel>
                        <Input
                          type='number'
                          min={0}
                          value={inputs.weight ?? 0}
                          placeholder={t('渠道权重')}
                          onChange={(event) =>
                            handleInputChange(
                              'weight',
                              Number(event.target.value || 0),
                            )
                          }
                          className='border-white/10 bg-white/5 text-white'
                        />
                      </div>
                    </div>

                    <div className='space-y-2'>
                      <FieldLabel>{t('是否自动禁用')}</FieldLabel>
                      <div className='flex items-center gap-3'>
                        <Switch
                          checked={!!autoBan}
                          onCheckedChange={(value) => setAutoBan(!!value)}
                        />
                        <span className='text-sm text-white/70'>
                          {autoBan ? t('开') : t('关')}
                        </span>
                      </div>
                      <FieldHint>
                        {t('仅当自动禁用开启时有效，关闭后不会自动禁用该渠道')}
                      </FieldHint>
                    </div>

                    <div className='space-y-2'>
                      <FieldLabel>{t('是否自动同步上游模型更新')}</FieldLabel>
                      <div className='flex items-center gap-3'>
                        <Switch
                          checked={!!inputs.upstream_model_update_auto_sync_enabled}
                          disabled={!inputs.upstream_model_update_check_enabled}
                          onCheckedChange={(value) =>
                            handleChannelOtherSettingsChange(
                              'upstream_model_update_auto_sync_enabled',
                              !!value,
                            )
                          }
                        />
                        <span className='text-sm text-white/70'>
                          {inputs.upstream_model_update_auto_sync_enabled
                            ? t('开')
                            : t('关')}
                        </span>
                      </div>
                      <FieldHint>
                        {t('开启后检测到新增模型会自动加入当前渠道模型列表')}
                      </FieldHint>
                    </div>

                    <div className='mb-3 text-xs text-white/60'>
                      {t('上次检测到可加入模型')}:&nbsp;
                      {upstreamDetectedModels.length === 0 ? (
                        t('暂无')
                      ) : (
                        <>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className='cursor-help break-all'>
                                {upstreamDetectedModelsPreview.join(', ')}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent className='max-w-[640px] border-white/10 bg-black/90 text-xs leading-5 text-white'>
                              {upstreamDetectedModels.join(', ')}
                            </TooltipContent>
                          </Tooltip>
                          <span className='ml-1 text-white/40'>
                            {upstreamDetectedModelsOmittedCount > 0
                              ? t('（共 {{total}} 个，省略 {{omit}} 个）', {
                                  total: upstreamDetectedModels.length,
                                  omit: upstreamDetectedModelsOmittedCount,
                                })
                              : t('（共 {{total}} 个）', {
                                  total: upstreamDetectedModels.length,
                                })}
                          </span>
                        </>
                      )}
                    </div>

                    <div className='mb-4'>
                      <div className='flex items-center justify-between gap-2 mb-1'>
                        <div className='text-sm font-medium text-white'>
                          {t('参数覆盖')}
                        </div>
                        <div className='flex flex-wrap gap-2'>
                          <Button
                            size='sm'
                            type='button'
                            onClick={() => setParamOverrideEditorVisible(true)}
                          >
                            <Code2 className='mr-1 h-4 w-4' />
                            {t('可视化编辑')}
                          </Button>
                          <Button
                            size='sm'
                            type='button'
                            variant='secondary'
                            onClick={() =>
                              applyParamOverrideTemplate('operations', 'fill')
                            }
                          >
                            {t('填充新模板')}
                          </Button>
                          <Button
                            size='sm'
                            type='button'
                            variant='secondary'
                            onClick={() =>
                              applyParamOverrideTemplate('legacy', 'fill')
                            }
                          >
                            {t('填充旧模板')}
                          </Button>
                          <Button
                            size='sm'
                            type='button'
                            variant='outline'
                            onClick={clearParamOverride}
                          >
                            {t('清空')}
                          </Button>
                        </div>
                      </div>
                      <FieldHint>
                        {t(
                          '此项可选，用于覆盖请求参数。不支持覆盖 stream 参数',
                        )}
                      </FieldHint>
                      <div
                        className='mt-2 rounded-xl border border-white/10 bg-white/5 p-3'
                      >
                        <div className='flex items-center justify-between mb-2'>
                          <Badge
                            className={
                              paramOverrideMeta.tagColor === 'green'
                                ? 'bg-emerald-500/15 text-emerald-200'
                                : paramOverrideMeta.tagColor === 'orange'
                                  ? 'bg-amber-500/15 text-amber-200'
                                  : 'bg-zinc-500/15 text-zinc-200'
                            }
                          >
                            {paramOverrideMeta.tagLabel}
                          </Badge>
                          <div className='flex gap-2'>
                            <Button
                              size='sm'
                              type='button'
                              variant='outline'
                              onClick={copyParamOverrideJson}
                            >
                              <Copy className='mr-1 h-4 w-4' />
                              {t('复制')}
                            </Button>
                            <Button
                              size='sm'
                              type='button'
                              variant='outline'
                              onClick={() =>
                                setParamOverrideEditorVisible(true)
                              }
                            >
                              {t('编辑')}
                            </Button>
                          </div>
                        </div>
                        <pre className='mb-0 text-xs leading-5 whitespace-pre-wrap break-all max-h-56 overflow-auto'>
                          {paramOverrideMeta.preview}
                        </pre>
                      </div>
                    </div>

                    <div className='space-y-2'>
                      <FieldLabel>{t('请求头覆盖')}</FieldLabel>
                      <Textarea
                        value={inputs.header_override || ''}
                        placeholder={
                          t('此项可选，用于覆盖请求头参数') +
                          '\n' +
                          t('格式示例：') +
                          '\n{\n  "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0",\n  "Authorization": "Bearer {api_key}"\n}'
                        }
                        onChange={(event) =>
                          handleInputChange('header_override', event.target.value)
                        }
                        className='min-h-[180px] border-white/10 bg-white/5 text-white'
                      />
                      <div className='flex flex-col gap-2'>
                        <div className='flex flex-wrap items-center gap-3 text-sm'>
                          <button
                            type='button'
                            className='text-blue-300 transition hover:text-blue-200'
                            onClick={() =>
                              handleInputChange(
                                'header_override',
                                JSON.stringify(
                                  {
                                    '*': true,
                                    're:^X-Trace-.*$': true,
                                    'X-Foo': '{client_header:X-Foo}',
                                    Authorization: 'Bearer {api_key}',
                                    'User-Agent':
                                      'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0',
                                  },
                                  null,
                                  2,
                                ),
                              )
                            }
                          >
                            {t('填入模板')}
                          </button>
                          <button
                            type='button'
                            className='text-blue-300 transition hover:text-blue-200'
                            onClick={() =>
                              handleInputChange(
                                'header_override',
                                JSON.stringify(
                                  {
                                    '*': true,
                                  },
                                  null,
                                  2,
                                ),
                              )
                            }
                          >
                            {t('填入透传模版')}
                          </button>
                          <button
                            type='button'
                            className='text-blue-300 transition hover:text-blue-200'
                            onClick={() => formatJsonField('header_override')}
                          >
                            {t('格式化')}
                          </button>
                        </div>
                        <div>
                          <FieldHint>{t('支持变量：')}</FieldHint>
                          <div className='ml-2 text-xs text-white/60'>
                            <div>
                              {t('渠道密钥')}: {'{api_key}'}
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                    <JSONEditor
                      key={`status_code_mapping-${isEdit ? channelId : 'new'}`}
                      field='status_code_mapping'
                      label={t('状态码复写')}
                      placeholder={
                        t(
                          '此项可选，用于复写返回的状态码，仅影响本地判断，不修改返回到上游的状态码，比如将claude渠道的400错误复写为500（用于重试），请勿滥用该功能，例如：',
                        ) +
                        '\n' +
                        JSON.stringify(STATUS_CODE_MAPPING_EXAMPLE, null, 2)
                      }
                      value={inputs.status_code_mapping || ''}
                      onChange={(value) =>
                        handleInputChange('status_code_mapping', value)
                      }
                      template={STATUS_CODE_MAPPING_EXAMPLE}
                      templateLabel={t('填入模板')}
                      editorType='keyValue'
                      formApi={formApiRef.current}
                      extraText={t(
                        '键为原状态码，值为要复写的状态码，仅影响本地判断',
                      )}
                    />

                    {/* 字段透传控制 - OpenAI 渠道 */}
                    {inputs.type === 1 && (
                      <>
                        <div className='mb-2 mt-4 text-sm font-medium text-white'>
                          {t('字段透传控制')}
                        </div>

                        <ToggleField
                          label={t('允许 service_tier 透传')}
                          checked={inputs.allow_service_tier}
                          onChange={(value) =>
                            handleChannelOtherSettingsChange(
                              'allow_service_tier',
                              !!value,
                            )
                          }
                          hint={t(
                            'service_tier 字段用于指定服务层级，允许透传可能导致实际计费高于预期。默认关闭以避免额外费用',
                          )}
                        />

                        <ToggleField
                          label={t('禁用 store 透传')}
                          checked={inputs.disable_store}
                          onChange={(value) =>
                            handleChannelOtherSettingsChange(
                              'disable_store',
                              !!value,
                            )
                          }
                          hint={t(
                            'store 字段用于授权 OpenAI 存储请求数据以评估和优化产品。默认关闭，开启后可能导致 Codex 无法正常使用',
                          )}
                        />

                        <ToggleField
                          label={t('允许 safety_identifier 透传')}
                          checked={inputs.allow_safety_identifier}
                          onChange={(value) =>
                            handleChannelOtherSettingsChange(
                              'allow_safety_identifier',
                              !!value,
                            )
                          }
                          hint={t(
                            'safety_identifier 字段用于帮助 OpenAI 识别可能违反使用政策的应用程序用户。默认关闭以保护用户隐私',
                          )}
                        />

                        <ToggleField
                          label={t(
                            '允许 stream_options.include_obfuscation 透传',
                          )}
                          checked={inputs.allow_include_obfuscation}
                          onChange={(value) =>
                            handleChannelOtherSettingsChange(
                              'allow_include_obfuscation',
                              !!value,
                            )
                          }
                          hint={t(
                            'include_obfuscation 用于控制 Responses 流混淆字段。默认关闭以避免客户端关闭该安全保护',
                          )}
                        />
                      </>
                    )}

                    {/* 字段透传控制 - Claude 渠道 */}
                    {inputs.type === 14 && (
                      <>
                        <div className='mb-2 mt-4 text-sm font-medium text-white'>
                          {t('字段透传控制')}
                        </div>

                        <ToggleField
                          label={t('允许 service_tier 透传')}
                          checked={inputs.allow_service_tier}
                          onChange={(value) =>
                            handleChannelOtherSettingsChange(
                              'allow_service_tier',
                              !!value,
                            )
                          }
                          hint={t(
                            'service_tier 字段用于指定服务层级，允许透传可能导致实际计费高于预期。默认关闭以避免额外费用',
                          )}
                        />

                        <ToggleField
                          label={t('允许 inference_geo 透传')}
                          checked={inputs.allow_inference_geo}
                          onChange={(value) =>
                            handleChannelOtherSettingsChange(
                              'allow_inference_geo',
                              !!value,
                            )
                          }
                          hint={t(
                            'inference_geo 字段用于控制 Claude 数据驻留推理区域。默认关闭以避免未经授权透传地域信息',
                          )}
                        />
                      </>
                    )}
                  </Card>
                </div>

                {/* Channel Extra Settings Card */}
                <div
                  ref={(el) =>
                    (formSectionRefs.current.channelExtraSettings = el)
                  }
                >
                  <Card className='mb-6 rounded-2xl border-white/10 bg-white/5 py-0 shadow-sm'>
                    {/* Header: Channel Extra Settings */}
                    <div className='mb-2 flex items-center gap-2'>
                      <SectionIcon
                        icon={Zap}
                        className='bg-fuchsia-500/15 text-fuchsia-200'
                      />
                      <div>
                        <div className='text-lg font-medium text-white'>
                          {t('渠道额外设置')}
                        </div>
                      </div>
                    </div>

                    {inputs.type === 14 && (
                      <ToggleField
                        label={t('Claude 强制 beta=true')}
                        checked={inputs.claude_beta_query}
                        onChange={(value) =>
                          handleChannelOtherSettingsChange(
                            'claude_beta_query',
                            !!value,
                          )
                        }
                        hint={t(
                          '开启后，该渠道请求 Claude 时将强制追加 ?beta=true（无需客户端手动传参）',
                        )}
                      />
                    )}

                    {inputs.type === 1 && (
                      <ToggleField
                        label={t('强制格式化')}
                        checked={inputs.force_format}
                        onChange={(value) =>
                          handleChannelSettingsChange('force_format', !!value)
                        }
                        hint={t(
                          '强制将响应格式化为 OpenAI 标准格式（只适用于OpenAI渠道类型）',
                        )}
                      />
                    )}

                    <ToggleField
                      label={t('思考内容转换')}
                      checked={inputs.thinking_to_content}
                      onChange={(value) =>
                        handleChannelSettingsChange(
                          'thinking_to_content',
                          !!value,
                        )
                      }
                      hint={t(
                        '将 reasoning_content 转换为 <think> 标签拼接到内容中',
                      )}
                    />

                    <ToggleField
                      label={t('透传请求体')}
                      checked={inputs.pass_through_body_enabled}
                      onChange={(value) =>
                        handleChannelSettingsChange(
                          'pass_through_body_enabled',
                          !!value,
                        )
                      }
                      hint={t('启用请求体透传功能')}
                    />

                    <div className='space-y-2'>
                      <FieldLabel>{t('代理地址')}</FieldLabel>
                      <Input
                        value={inputs.proxy || ''}
                        placeholder={t('例如: socks5://user:pass@host:port')}
                        onChange={(event) =>
                          handleChannelSettingsChange('proxy', event.target.value)
                        }
                        className='border-white/10 bg-white/5 text-white'
                      />
                      <FieldHint>{t('用于配置网络代理，支持 socks5 协议')}</FieldHint>
                    </div>

                    <div className='space-y-2'>
                      <FieldLabel>{t('系统提示词')}</FieldLabel>
                      <Textarea
                        value={inputs.system_prompt || ''}
                        placeholder={t(
                          '输入系统提示词，用户的系统提示词将优先于此设置',
                        )}
                        onChange={(event) =>
                          handleChannelSettingsChange(
                            'system_prompt',
                            event.target.value,
                          )
                        }
                        className='min-h-[120px] border-white/10 bg-white/5 text-white'
                      />
                      <FieldHint>
                        {t(
                          '用户优先：如果用户在请求中指定了系统提示词，将优先使用用户的设置',
                        )}
                      </FieldHint>
                    </div>
                    <ToggleField
                      label={t('系统提示词拼接')}
                      checked={inputs.system_prompt_override}
                      onChange={(value) =>
                        handleChannelSettingsChange(
                          'system_prompt_override',
                          !!value,
                        )
                      }
                      hint={t(
                        '如果用户请求中包含系统提示词，则使用此设置拼接到用户的系统提示词前面',
                      )}
                    />
                  </Card>
                </div>
            </div>
          </ScrollArea>
          <DialogFooter className='border-white/10 bg-black/80'>
            <div className='mr-auto flex gap-2'>
              <Button
                size='icon-sm'
                type='button'
                variant='secondary'
                onClick={() => navigateToSection('up')}
                title={t('上一个表单块')}
              >
                <ChevronUp className='h-4 w-4' />
              </Button>
              <Button
                size='icon-sm'
                type='button'
                variant='secondary'
                onClick={() => navigateToSection('down')}
                title={t('下一个表单块')}
              >
                <ChevronDown className='h-4 w-4' />
              </Button>
            </div>
            <Button type='button' variant='secondary' onClick={handleCancel}>
              <X className='mr-1 h-4 w-4' />
              {t('取消')}
            </Button>
            <Button type='button' onClick={submit}>
              <Save className='mr-1 h-4 w-4' />
              {t('提交')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <StatusCodeRiskGuardModal
        visible={statusCodeRiskConfirmVisible}
        detailItems={statusCodeRiskDetailItems}
        onCancel={() => resolveStatusCodeRiskConfirm(false)}
        onConfirm={() => resolveStatusCodeRiskConfirm(true)}
      />

      <Dialog
        open={keyDisplayState.showModal}
        onOpenChange={(open) => !open && resetKeyDisplayState()}
      >
        <DialogContent className='max-w-[700px] border-white/10 bg-black text-white'>
          <DialogHeader>
            <div className='flex items-center'>
              <div className='mr-3 flex h-8 w-8 items-center justify-center rounded-full bg-green-100 dark:bg-green-900'>
                <svg
                  className='h-4 w-4 text-green-600 dark:text-green-400'
                  fill='currentColor'
                  viewBox='0 0 20 20'
                >
                  <path
                    fillRule='evenodd'
                    d='M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z'
                    clipRule='evenodd'
                  />
                </svg>
              </div>
              <DialogTitle>{t('渠道密钥信息')}</DialogTitle>
            </div>
          </DialogHeader>
          <ChannelKeyDisplay
            keyData={keyDisplayState.keyData}
            showSuccessIcon={true}
            successText={t('密钥获取成功')}
            showWarning={true}
            warningText={t(
              '请妥善保管密钥信息，不要泄露给他人。如有安全疑虑，请及时更换密钥。',
            )}
          />
          <DialogFooter className='border-white/10 bg-black/80'>
            <Button type='button' onClick={resetKeyDisplayState}>
              {t('完成')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ParamOverrideEditorModal
        visible={paramOverrideEditorVisible}
        value={inputs.param_override || ''}
        onCancel={() => setParamOverrideEditorVisible(false)}
        onSave={(nextValue) => {
          handleInputChange('param_override', nextValue);
          setParamOverrideEditorVisible(false);
        }}
      />

      <ModelSelectModal
        visible={modelModalVisible}
        models={fetchedModels}
        selected={inputs.models}
        redirectModels={redirectModelList}
        onConfirm={(selectedModels) => {
          handleInputChange('models', selectedModels);
          showSuccess(t('模型列表已更新'));
          setModelModalVisible(false);
        }}
        onCancel={() => setModelModalVisible(false)}
      />

      <SingleModelSelectModal
        visible={modelMappingValueModalVisible}
        models={modelMappingValueModalModels}
        selected={modelMappingValueSelected}
        onConfirm={(selectedModel) => {
          const modelName = String(selectedModel ?? '').trim();
          if (!modelName) {
            showError(t('请先选择模型！'));
            return;
          }

          const mappingKey = String(modelMappingValueKey ?? '').trim();
          if (!mappingKey) {
            setModelMappingValueModalVisible(false);
            return;
          }

          let parsed = {};
          const currentMapping = inputs.model_mapping;
          if (typeof currentMapping === 'string' && currentMapping.trim()) {
            try {
              parsed = JSON.parse(currentMapping);
            } catch (error) {
              parsed = {};
            }
          } else if (
            currentMapping &&
            typeof currentMapping === 'object' &&
            !Array.isArray(currentMapping)
          ) {
            parsed = currentMapping;
          }
          if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
            parsed = {};
          }

          parsed[mappingKey] = modelName;
          const nextMapping = JSON.stringify(parsed, null, 2);
          handleInputChange('model_mapping', nextMapping);
          if (formApiRef.current) {
            formApiRef.current.setValue('model_mapping', nextMapping);
          }
          setModelMappingValueModalVisible(false);
        }}
        onCancel={() => setModelMappingValueModalVisible(false)}
      />

      <OllamaModelModal
        visible={ollamaModalVisible}
        onCancel={() => setOllamaModalVisible(false)}
        channelId={channelId}
        channelInfo={inputs}
        onModelsUpdate={(options = {}) => {
          // 当模型更新后，重新获取模型列表以更新表单
          fetchUpstreamModelList('models', { silent: !!options.silent });
        }}
        onApplyModels={({ mode, modelIds } = {}) => {
          if (!Array.isArray(modelIds) || modelIds.length === 0) {
            return;
          }
          const existingModels = Array.isArray(inputs.models)
            ? inputs.models.map(String)
            : [];
          const incoming = modelIds.map(String);
          const nextModels = Array.from(
            new Set([...existingModels, ...incoming]),
          );

          handleInputChange('models', nextModels);
          if (formApiRef.current) {
            formApiRef.current.setValue('models', nextModels);
          }
          showSuccess(t('模型列表已追加更新'));
        }}
      />
    </>
  );
};

export default EditChannelModal;

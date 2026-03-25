
import React, { useState, useEffect, useMemo } from 'react';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  showWarning,
  verifyJSON,
  selectFilter,
} from '../../../../helpers';
import {
  Save,
  X,
  Bookmark,
  User,
  Code2,
  Settings,
  Plus,
} from 'lucide-react';
import { getChannelModels } from '../../../../helpers';
import { useTranslation } from 'react-i18next';
import {
  Dialog,
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
import { ScrollArea } from '@/components/ui/scroll-area';
import { Checkbox } from '@/components/ui/checkbox';

const MODEL_MAPPING_EXAMPLE = {
  'gpt-3.5-turbo': 'gpt-3.5-turbo-0125',
};

const createInitialInputs = (tag = '') => ({
  tag,
  new_tag: tag || null,
  model_mapping: null,
  groups: [],
  models: [],
  param_override: null,
  header_override: null,
});

const SectionHeader = ({ icon: Icon, title, description, iconClassName }) => (
  <div className='mb-3 flex items-center gap-3'>
    <div
      className={`flex h-9 w-9 items-center justify-center rounded-full ${iconClassName}`}
    >
      <Icon className='h-4 w-4' />
    </div>
    <div>
      <div className='text-lg font-medium text-white'>{title}</div>
      <div className='text-xs text-white/60'>{description}</div>
    </div>
  </div>
);

const HelperLink = ({ children, onClick }) => (
  <button
    type='button'
    className='text-sm text-blue-300 transition hover:text-blue-200'
    onClick={onClick}
  >
    {children}
  </button>
);

const EditTagModal = (props) => {
  const { t } = useTranslation();
  const { visible, tag, handleClose, refresh } = props;
  const [loading, setLoading] = useState(false);
  const [originModelOptions, setOriginModelOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);
  const [groupOptions, setGroupOptions] = useState([]);
  const [customModel, setCustomModel] = useState('');
  const [modelSearchValue, setModelSearchValue] = useState('');
  const [inputs, setInputs] = useState(() => createInitialInputs());
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
  const handleInputChange = (name, value) => {
    setInputs((currentInputs) => ({ ...currentInputs, [name]: value }));
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
        case 53:
          localModels = [
            'NousResearch/Hermes-4-405B-FP8',
            'Qwen/Qwen3-235B-A22B-Thinking-2507',
            'Qwen/Qwen3-Coder-480B-A35B-Instruct-FP8',
            'Qwen/Qwen3-235B-A22B-Instruct-2507',
            'zai-org/GLM-4.5-FP8',
            'openai/gpt-oss-120b',
            'deepseek-ai/DeepSeek-R1-0528',
            'deepseek-ai/DeepSeek-R1',
            'deepseek-ai/DeepSeek-V3-0324',
            'deepseek-ai/DeepSeek-V3.1',
          ];
          break;
        default:
          localModels = getChannelModels(value);
          break;
      }
      if (inputs.models.length === 0) {
        setInputs((inputs) => ({ ...inputs, models: localModels }));
      }
    }
  };

  const fetchModels = async () => {
    try {
      let res = await API.get(`/api/channel/models`);
      let localModelOptions = res.data.data.map((model) => ({
        label: model.id,
        value: model.id,
      }));
      setOriginModelOptions(localModelOptions);
    } catch (error) {
      showError(error.message);
    }
  };

  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/group/`);
      if (res === undefined) {
        return;
      }
      setGroupOptions(
        res.data.data.map((group) => ({
          label: group,
          value: group,
        })),
      );
    } catch (error) {
      showError(error.message);
    }
  };

  const handleSave = async () => {
    setLoading(true);
    const formVals = inputs || {};
    let data = { tag };
    if (formVals.model_mapping) {
      if (!verifyJSON(formVals.model_mapping)) {
        showInfo('模型映射必须是合法的 JSON 格式！');
        setLoading(false);
        return;
      }
      data.model_mapping = formVals.model_mapping;
    }
    if (formVals.groups && formVals.groups.length > 0) {
      data.groups = formVals.groups.join(',');
    }
    if (formVals.models && formVals.models.length > 0) {
      data.models = formVals.models.join(',');
    }
    if (
      formVals.param_override !== undefined &&
      formVals.param_override !== null
    ) {
      if (typeof formVals.param_override !== 'string') {
        showInfo('参数覆盖必须是合法的 JSON 格式！');
        setLoading(false);
        return;
      }
      const trimmedParamOverride = formVals.param_override.trim();
      if (trimmedParamOverride !== '' && !verifyJSON(trimmedParamOverride)) {
        showInfo('参数覆盖必须是合法的 JSON 格式！');
        setLoading(false);
        return;
      }
      data.param_override = trimmedParamOverride;
    }
    if (
      formVals.header_override !== undefined &&
      formVals.header_override !== null
    ) {
      if (typeof formVals.header_override !== 'string') {
        showInfo('请求头覆盖必须是合法的 JSON 格式！');
        setLoading(false);
        return;
      }
      const trimmedHeaderOverride = formVals.header_override.trim();
      if (trimmedHeaderOverride !== '' && !verifyJSON(trimmedHeaderOverride)) {
        showInfo('请求头覆盖必须是合法的 JSON 格式！');
        setLoading(false);
        return;
      }
      data.header_override = trimmedHeaderOverride;
    }
    data.new_tag = formVals.new_tag;
    if (
      data.model_mapping === undefined &&
      data.groups === undefined &&
      data.models === undefined &&
      data.new_tag === undefined &&
      data.param_override === undefined &&
      data.header_override === undefined
    ) {
      showWarning('没有任何修改！');
      setLoading(false);
      return;
    }
    await submit(data);
    setLoading(false);
  };

  const submit = async (data) => {
    try {
      const res = await API.put('/api/channel/tag', data);
      if (res?.data?.success) {
        showSuccess('标签更新成功！');
        refresh();
        handleClose();
      }
    } catch (error) {
      showError(error);
    }
  };

  useEffect(() => {
    let localModelOptions = [...originModelOptions];
    inputs.models.forEach((model) => {
      if (!localModelOptions.find((option) => option.label === model)) {
        localModelOptions.push({
          label: model,
          value: model,
        });
      }
    });
    setModelOptions(localModelOptions);
  }, [originModelOptions, inputs.models]);

  useEffect(() => {
    const fetchTagModels = async () => {
      if (!tag) return;
      setLoading(true);
      try {
        const res = await API.get(`/api/channel/tag/models?tag=${tag}`);
        if (res?.data?.success) {
          const models = res.data.data ? res.data.data.split(',') : [];
          handleInputChange('models', models);
        } else {
          showError(res.data.message);
        }
      } catch (error) {
        showError(error.message);
      } finally {
        setLoading(false);
      }
    };

    fetchModels().then();
    fetchGroups().then();
    fetchTagModels().then();
    setModelSearchValue('');
    setInputs(createInitialInputs(tag));
  }, [visible, tag]);

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
          text: model,
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

  const filteredModelOptions = useMemo(() => {
    const keyword = modelSearchValue.trim();
    if (!keyword) return modelOptions;
    return modelOptions.filter((option) => selectFilter(keyword, option));
  }, [modelOptions, modelSearchValue]);

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className='max-w-[960px] border-white/10 bg-black p-0 text-white sm:max-w-[960px]'>
        <DialogHeader className='border-b border-white/10 px-6 py-4'>
          <div className='flex items-center gap-3'>
            <Badge className='border-blue-500/20 bg-blue-500/15 text-blue-200'>
              {t('编辑')}
            </Badge>
            <DialogTitle className='text-xl text-white'>
              {t('编辑标签')}
            </DialogTitle>
          </div>
        </DialogHeader>

        <ScrollArea className='max-h-[75vh] px-6 py-5'>
          <div className='space-y-6'>
            <Card className='border-white/10 bg-white/5 py-0'>
              <CardContent className='p-5'>
                <SectionHeader
                  icon={Bookmark}
                  title={t('标签信息')}
                  description={t('标签的基本配置')}
                  iconClassName='bg-blue-500/15 text-blue-200'
                />

                <div className='mb-4 rounded-lg border border-yellow-500/20 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-100'>
                  {t('所有编辑均为覆盖操作，留空则不更改')}
                </div>

                <div className='space-y-2'>
                  <label className='text-sm text-white/70'>{t('标签名称')}</label>
                  <Input
                    value={inputs.new_tag ?? ''}
                    placeholder={t('请输入新标签，留空则解散标签')}
                    onChange={(e) =>
                      handleInputChange('new_tag', e.target.value)
                    }
                    className='border-white/10 bg-white/5 text-white'
                  />
                </div>
              </CardContent>
            </Card>

            <Card className='border-white/10 bg-white/5 py-0'>
              <CardContent className='p-5'>
                <SectionHeader
                  icon={Code2}
                  title={t('模型配置')}
                  description={t('模型选择和映射设置')}
                  iconClassName='bg-purple-500/15 text-purple-200'
                />

                <div className='space-y-4'>
                  <div className='rounded-lg border border-blue-500/20 bg-blue-500/10 px-3 py-2 text-sm text-blue-100'>
                    {t(
                      '当前模型列表为该标签下所有渠道模型列表最长的一个，并非所有渠道的并集，请注意可能导致某些渠道模型丢失。',
                    )}
                  </div>

                  <div className='space-y-2'>
                    <label className='text-sm text-white/70'>{t('模型')}</label>
                    <Input
                      value={modelSearchValue}
                      placeholder={t('搜索模型')}
                      onChange={(e) => setModelSearchValue(e.target.value || '')}
                      className='border-white/10 bg-white/5 text-white'
                    />
                    <div className='rounded-lg border border-white/10 bg-black/20 p-3'>
                      <div className='mb-2 flex items-center justify-between text-xs text-white/50'>
                        <span>
                          {t('已选 {{count}} 项', {
                            count: inputs.models?.length || 0,
                          })}
                        </span>
                        <span>
                          {t('匹配 {{count}} 项', {
                            count: filteredModelOptions.length,
                          })}
                        </span>
                      </div>
                      <ScrollArea className='h-56 pr-3'>
                        <div className='space-y-2'>
                          {filteredModelOptions.map((option) => {
                            const checked = inputs.models.includes(option.value);
                            return (
                              <label
                                key={option.value}
                                className='flex items-center gap-2 rounded-md border border-white/10 px-3 py-2 text-sm hover:bg-white/5'
                              >
                                <Checkbox
                                  checked={checked}
                                  onCheckedChange={(nextChecked) => {
                                    const nextModels = nextChecked
                                      ? [...inputs.models, option.value]
                                      : inputs.models.filter(
                                          (model) => model !== option.value,
                                        );
                                    handleInputChange('models', nextModels);
                                  }}
                                />
                                <span>{option.label}</span>
                              </label>
                            );
                          })}
                        </div>
                      </ScrollArea>
                      {modelSearchHintText ? (
                        <p className='mt-2 text-xs text-white/50'>
                          {modelSearchHintText}
                        </p>
                      ) : null}
                    </div>
                  </div>

                  <div className='space-y-2'>
                    <label className='text-sm text-white/70'>
                      {t('自定义模型名称')}
                    </label>
                    <div className='flex gap-2'>
                      <Input
                        value={customModel}
                        placeholder={t('输入自定义模型名称')}
                        onChange={(e) => setCustomModel(e.target.value.trim())}
                        className='border-white/10 bg-white/5 text-white'
                      />
                      <Button type='button' onClick={addCustomModels}>
                        <Plus className='mr-1 h-4 w-4' />
                        {t('填入')}
                      </Button>
                    </div>
                  </div>

                  <div className='space-y-2'>
                    <label className='text-sm text-white/70'>
                      {t('模型重定向')}
                    </label>
                    <Textarea
                      value={inputs.model_mapping ?? ''}
                      placeholder={t(
                        '此项可选，用于修改请求体中的模型名称，为一个 JSON 字符串，键为请求中模型名称，值为要替换的模型名称，留空则不更改',
                      )}
                      onChange={(e) =>
                        handleInputChange('model_mapping', e.target.value)
                      }
                      className='min-h-[140px] border-white/10 bg-white/5 font-mono text-sm text-white'
                    />
                    <div className='flex flex-wrap gap-3'>
                      <HelperLink
                        onClick={() =>
                          handleInputChange(
                            'model_mapping',
                            JSON.stringify(MODEL_MAPPING_EXAMPLE, null, 2),
                          )
                        }
                      >
                        {t('填入模板')}
                      </HelperLink>
                      <HelperLink
                        onClick={() =>
                          handleInputChange(
                            'model_mapping',
                            JSON.stringify({}, null, 2),
                          )
                        }
                      >
                        {t('清空重定向')}
                      </HelperLink>
                      <HelperLink
                        onClick={() => handleInputChange('model_mapping', '')}
                      >
                        {t('不更改')}
                      </HelperLink>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className='border-white/10 bg-white/5 py-0'>
              <CardContent className='p-5'>
                <SectionHeader
                  icon={Settings}
                  title={t('高级设置')}
                  description={t('渠道的高级配置选项')}
                  iconClassName='bg-orange-500/15 text-orange-200'
                />

                <div className='space-y-4'>
                  <div className='space-y-2'>
                    <label className='text-sm text-white/70'>
                      {t('参数覆盖')}
                    </label>
                    <Textarea
                      value={inputs.param_override ?? ''}
                      placeholder={
                        t('此项可选，用于覆盖请求参数。不支持覆盖 stream 参数') +
                        '\n' +
                        t('旧格式（直接覆盖）：') +
                        '\n{\n  "temperature": 0,\n  "max_tokens": 1000\n}' +
                        '\n\n' +
                        t('新格式（支持条件判断与json自定义）：') +
                        '\n{\n  "operations": [\n    {\n      "path": "temperature",\n      "mode": "set",\n      "value": 0.7,\n      "conditions": [\n        {\n          "path": "model",\n          "mode": "prefix",\n          "value": "gpt"\n        }\n      ]\n    }\n  ]\n}'
                      }
                      onChange={(e) =>
                        handleInputChange('param_override', e.target.value)
                      }
                      className='min-h-[220px] border-white/10 bg-white/5 font-mono text-sm text-white'
                    />
                    <div className='flex flex-wrap gap-3'>
                      <HelperLink
                        onClick={() =>
                          handleInputChange(
                            'param_override',
                            JSON.stringify({ temperature: 0 }, null, 2),
                          )
                        }
                      >
                        {t('旧格式模板')}
                      </HelperLink>
                      <HelperLink
                        onClick={() =>
                          handleInputChange(
                            'param_override',
                            JSON.stringify(
                              {
                                operations: [
                                  {
                                    path: 'temperature',
                                    mode: 'set',
                                    value: 0.7,
                                    conditions: [
                                      {
                                        path: 'model',
                                        mode: 'prefix',
                                        value: 'gpt',
                                      },
                                    ],
                                    logic: 'AND',
                                  },
                                ],
                              },
                              null,
                              2,
                            ),
                          )
                        }
                      >
                        {t('新格式模板')}
                      </HelperLink>
                      <HelperLink
                        onClick={() => handleInputChange('param_override', null)}
                      >
                        {t('不更改')}
                      </HelperLink>
                    </div>
                  </div>

                  <div className='space-y-2'>
                    <label className='text-sm text-white/70'>
                      {t('请求头覆盖')}
                    </label>
                    <Textarea
                      value={inputs.header_override ?? ''}
                      placeholder={
                        t('此项可选，用于覆盖请求头参数') +
                        '\n' +
                        t('格式示例：') +
                        '\n{\n  "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0",\n  "Authorization": "Bearer {api_key}"\n}'
                      }
                      onChange={(e) =>
                        handleInputChange('header_override', e.target.value)
                      }
                      className='min-h-[180px] border-white/10 bg-white/5 font-mono text-sm text-white'
                    />
                    <div className='flex flex-col gap-2'>
                      <div className='flex flex-wrap items-center gap-3'>
                        <HelperLink
                          onClick={() =>
                            handleInputChange(
                              'header_override',
                              JSON.stringify(
                                {
                                  'User-Agent':
                                    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0',
                                  Authorization: 'Bearer {api_key}',
                                },
                                null,
                                2,
                              ),
                            )
                          }
                        >
                          {t('填入模板')}
                        </HelperLink>
                        <HelperLink
                          onClick={() =>
                            handleInputChange('header_override', null)
                          }
                        >
                          {t('不更改')}
                        </HelperLink>
                      </div>
                      <p className='text-xs text-white/50'>
                        {t('支持变量：')} {t('渠道密钥')}: {'{api_key}'}
                      </p>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className='border-white/10 bg-white/5 py-0'>
              <CardContent className='p-5'>
                <SectionHeader
                  icon={User}
                  title={t('分组设置')}
                  description={t('用户分组配置')}
                  iconClassName='bg-green-500/15 text-green-200'
                />

                <div className='space-y-2'>
                  <label className='text-sm text-white/70'>{t('分组')}</label>
                  <div className='grid gap-2 sm:grid-cols-2'>
                    {groupOptions.map((group) => {
                      const checked = inputs.groups.includes(group.value);
                      return (
                        <label
                          key={group.value}
                          className='flex items-center gap-2 rounded-md border border-white/10 px-3 py-2 text-sm hover:bg-white/5'
                        >
                          <Checkbox
                            checked={checked}
                            onCheckedChange={(nextChecked) => {
                              const nextGroups = nextChecked
                                ? [...inputs.groups, group.value]
                                : inputs.groups.filter(
                                    (item) => item !== group.value,
                                  );
                              handleInputChange('groups', nextGroups);
                            }}
                          />
                          <span>{group.label}</span>
                        </label>
                      );
                    })}
                  </div>
                  <p className='text-xs text-white/50'>
                    {t('请在系统设置页面编辑分组倍率以添加新的分组：')}
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        </ScrollArea>

        <DialogFooter className='border-white/10 bg-black/80'>
          <Button
            type='button'
            variant='secondary'
            onClick={handleClose}
            disabled={loading}
          >
            <X className='mr-1 h-4 w-4' />
            {t('取消')}
          </Button>
          <Button type='button' onClick={handleSave} disabled={loading}>
            <Save className='mr-1 h-4 w-4' />
            {loading ? t('保存中...') : t('保存')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default EditTagModal;

import React, {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { toast } from 'sonner';
import dayjs from 'dayjs';
import {
  Calendar as AntCalendar,
  Checkbox as AntCheckbox,
  Collapse as AntCollapse,
  DatePicker as AntDatePicker,
  Descriptions as AntDescriptions,
  Input as AntInput,
  InputNumber as AntInputNumber,
  Pagination as AntPagination,
  Popconfirm as AntPopconfirm,
  Progress as AntProgress,
  Select as AntSelect,
  Switch as AntSwitch,
  Table as AntTable,
  Timeline as AntTimeline,
} from 'antd';
import { Loader2 } from 'lucide-react';
import { Card as UiCard, CardContent } from '@/components/ui/card';
import { Button as UiButton } from '@/components/ui/button';
import { Badge as UiBadge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Skeleton as UiSkeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

const FormContext = createContext(null);

function getTextContent(content) {
  if (typeof content === 'string' || typeof content === 'number') {
    return String(content);
  }
  if (Array.isArray(content)) {
    return content.map(getTextContent).join(' ');
  }
  if (React.isValidElement(content)) {
    return getTextContent(content.props?.children);
  }
  return '';
}

function mapButtonVariant(type, theme) {
  if (type === 'danger') return 'destructive';
  if (type === 'warning') return 'outline';
  if (theme === 'borderless' || type === 'tertiary') return 'ghost';
  if (theme === 'outline') return 'outline';
  if (theme === 'light') return 'secondary';
  return 'default';
}

function mapButtonSize(size) {
  if (size === 'small') return 'sm';
  if (size === 'large') return 'lg';
  return 'default';
}

function Button({
  children,
  icon,
  theme,
  type,
  size,
  block,
  className,
  htmlType,
  ...props
}) {
  const buttonType = ['button', 'submit', 'reset'].includes(type)
    ? type
    : htmlType || 'button';

  return (
    <UiButton
      type={buttonType}
      variant={mapButtonVariant(type, theme)}
      size={mapButtonSize(size)}
      className={cn(block && 'w-full', className)}
      {...props}
    >
      {icon}
      {children}
    </UiButton>
  );
}

function Card({ children, title, footer, className, bodyStyle, style, ...props }) {
  return (
    <UiCard className={className} style={style} {...props}>
      {title ? <div className='px-6 pt-6'>{title}</div> : null}
      <CardContent className={cn('p-6', !title && 'pt-6')} style={bodyStyle}>
        {children}
      </CardContent>
      {footer ? <div className='border-t border-white/10 px-6 py-4'>{footer}</div> : null}
    </UiCard>
  );
}

function Divider({ margin, className, style }) {
  return <Separator className={className} style={{ margin, ...style }} />;
}

function Text({ children, className, type, strong, style, ellipsis, ...props }) {
  const colorClass =
    type === 'tertiary'
      ? 'text-white/45'
      : type === 'secondary'
        ? 'text-white/60'
        : type === 'danger'
          ? 'text-red-400'
          : type === 'warning'
            ? 'text-amber-400'
            : 'text-inherit';

  return (
    <span
      className={cn(colorClass, strong && 'font-semibold', ellipsis && 'truncate', className)}
      style={style}
      title={ellipsis?.showTooltip ? getTextContent(children) : undefined}
      {...props}
    >
      {children}
    </span>
  );
}

function Title({ children, heading = 5, className, style, ...props }) {
  const Tag = `h${Math.min(Math.max(Number(heading) || 5, 1), 6)}`;
  return (
    <Tag className={cn('font-semibold tracking-tight', className)} style={style} {...props}>
      {children}
    </Tag>
  );
}

const Typography = { Text, Title };

function Space({ children, spacing = 2, wrap, className, style }) {
  return (
    <div
      className={cn('flex items-center', wrap && 'flex-wrap', className)}
      style={{ gap: spacing * 4, ...style }}
    >
      {children}
    </div>
  );
}

function mapTagClasses(color, type, shape) {
  const colorMap = {
    red: 'border-red-400/20 bg-red-500/10 text-red-200',
    green: 'border-emerald-400/20 bg-emerald-500/10 text-emerald-200',
    blue: 'border-sky-400/20 bg-sky-500/10 text-sky-200',
    orange: 'border-amber-400/20 bg-amber-500/10 text-amber-200',
    cyan: 'border-cyan-400/20 bg-cyan-500/10 text-cyan-200',
    grey: 'border-white/10 bg-white/10 text-white/75',
  };

  return cn(
    'inline-flex items-center border text-xs font-medium',
    shape === 'circle' ? 'rounded-full px-2.5 py-0.5' : 'rounded-md px-2 py-0.5',
    type === 'solid' ? 'opacity-100' : 'opacity-95',
    colorMap[color] || colorMap.grey,
  );
}

function Tag({ children, color = 'grey', type = 'light', shape, className, ...props }) {
  return (
    <span className={cn(mapTagClasses(color, type, shape), className)} {...props}>
      {children}
    </span>
  );
}

function Spin({ children, spinning }) {
  return (
    <div className='relative'>
      {spinning ? (
        <div className='absolute inset-0 z-10 flex items-center justify-center rounded-xl bg-black/30 backdrop-blur-[1px]'>
          <Loader2 className='h-5 w-5 animate-spin text-white' />
        </div>
      ) : null}
      <div className={cn(spinning && 'pointer-events-none opacity-70')}>{children}</div>
    </div>
  );
}

function Empty({ image, title, description, className, style }) {
  return (
    <div className={cn('flex flex-col items-center justify-center gap-3 py-8 text-center', className)} style={style}>
      {image || <div className='text-4xl text-white/20'>-</div>}
      {title ? <div className='text-sm font-medium text-white'>{title}</div> : null}
      {description ? <div className='max-w-sm text-xs text-white/50'>{description}</div> : null}
    </div>
  );
}

function Banner({ type = 'info', icon, description, className }) {
  const tone =
    type === 'warning'
      ? 'border-amber-400/20 bg-amber-500/10 text-amber-50'
      : type === 'danger'
        ? 'border-red-400/20 bg-red-500/10 text-red-50'
        : type === 'success'
          ? 'border-emerald-400/20 bg-emerald-500/10 text-emerald-50'
          : 'border-sky-400/20 bg-sky-500/10 text-sky-50';

  return (
    <div className={cn('flex items-start gap-3 rounded-2xl border px-4 py-3', tone, className)}>
      {icon ? <div className='mt-0.5 shrink-0'>{icon}</div> : null}
      <div className='text-sm leading-6'>{description}</div>
    </div>
  );
}

function Input({
  prefix,
  suffix,
  mode,
  onChange,
  value,
  autosize,
  showClear,
  type,
  ...props
}) {
  const sharedProps = {
    value,
    allowClear: showClear,
    prefix,
    suffix,
    onChange: (event) => onChange?.(event?.target?.value ?? ''),
    ...props,
  };

  if (autosize || props.rows) {
    return <AntInput.TextArea autoSize={autosize} {...sharedProps} />;
  }

  if (mode === 'password') {
    return <AntInput.Password {...sharedProps} />;
  }

  return <AntInput type={type} {...sharedProps} />;
}

const TextArea = AntInput.TextArea;
const InputNumber = AntInputNumber;
const DatePicker = AntDatePicker;
const Switch = AntSwitch;
const Checkbox = AntCheckbox;
const Progress = AntProgress;
const Pagination = AntPagination;
const Descriptions = AntDescriptions;
const Timeline = AntTimeline;
const Popconfirm = AntPopconfirm;
const Collapse = AntCollapse;
const Collapsible = AntCollapse;

function Table(props) {
  return <AntTable pagination={false} {...props} />;
}

function Calendar(props) {
  return <AntCalendar fullscreen={false} {...props} />;
}

function Image(props) {
  return <img alt='' {...props} />;
}

function Avatar({ src, alt, children, color, className, style, size }) {
  const pixelSize = size === 'small' ? 28 : size === 'large' ? 48 : 40;
  return (
    <div
      className={cn('inline-flex items-center justify-center overflow-hidden rounded-full bg-white/10 text-white', className)}
      style={{ width: pixelSize, height: pixelSize, color, ...style }}
    >
      {src ? <img src={src} alt={alt || ''} className='h-full w-full object-cover' /> : children}
    </div>
  );
}

function Badge({ children, count, overflowCount = 99, type }) {
  const display = count > overflowCount ? `${overflowCount}+` : count;
  const tone = type === 'danger' ? 'bg-red-500 text-white' : 'bg-white text-black';

  return (
    <div className='relative inline-flex'>
      {children}
      {count > 0 ? (
        <span className={cn('absolute -right-1 -top-1 min-w-5 rounded-full px-1 text-center text-[10px] font-semibold leading-5', tone)}>
          {display}
        </span>
      ) : null}
    </div>
  );
}

function PopoverCompat({ content, children }) {
  return (
    <Popover>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className='border-white/10 bg-[#0b1220] text-white'>
        {content}
      </PopoverContent>
    </Popover>
  );
}

function TooltipCompat({ content, children }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>{children}</TooltipTrigger>
      <TooltipContent className='border-white/10 bg-black text-white'>
        {content}
      </TooltipContent>
    </Tooltip>
  );
}

function NotificationView({ title, content }) {
  return (
    <div className='space-y-2 text-sm'>
      {title ? <div className='font-medium text-white'>{title}</div> : null}
      <div>{content}</div>
    </div>
  );
}

const Notification = {
  info({ id, title, content, duration }) {
    return toast.custom(() => <NotificationView title={title} content={content} />, {
      id,
      duration: duration === 0 ? Infinity : duration ? duration * 1000 : 4000,
    });
  },
  close(id) {
    toast.dismiss(id);
  },
};

const Toast = {
  success(content) {
    toast.success(content);
  },
  error(content) {
    toast.error(content);
  },
  warning(content) {
    toast.warning(content);
  },
  info(content) {
    toast.info(content);
  },
};

function Select({ optionList = [], onChange, value, placeholder, showClear, searchable, style, ...props }) {
  return (
    <AntSelect
      style={style}
      value={value}
      allowClear={showClear}
      showSearch={searchable}
      placeholder={placeholder}
      options={optionList}
      onChange={onChange}
      {...props}
    />
  );
}

function TabPane() {
  return null;
}

function Tabs({ activeKey, onChange, tabBarExtraContent, children, className }) {
  const panes = React.Children.toArray(children).filter(Boolean);
  const currentPane = panes.find((pane) => pane.props.itemKey === activeKey) || panes[0];

  return (
    <div className={cn('space-y-4', className)}>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='flex flex-wrap gap-2'>
          {panes.map((pane) => {
            const selected = pane.props.itemKey === (currentPane?.props.itemKey || activeKey);
            return (
              <button
                key={pane.props.itemKey}
                type='button'
                onClick={() => onChange?.(pane.props.itemKey)}
                className={cn(
                  'rounded-2xl border px-3 py-2 text-sm transition-colors',
                  selected
                    ? 'border-white/20 bg-white/12 text-white'
                    : 'border-white/10 bg-white/5 text-white/65 hover:bg-white/10',
                )}
              >
                {pane.props.tab}
              </button>
            );
          })}
        </div>
        {tabBarExtraContent}
      </div>
      {currentPane?.props.children ? <div>{currentPane.props.children}</div> : null}
    </div>
  );
}

function DropdownMenu({ children, className }) {
  return <div className={cn('grid gap-1', className)}>{children}</div>;
}

function DropdownItem({ children, icon, type, onClick, className }) {
  return (
    <button
      type='button'
      onClick={onClick}
      className={cn(
        'flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-sm text-white/85 hover:bg-white/10',
        type === 'danger' && 'text-red-200 hover:bg-red-500/10',
        className,
      )}
    >
      {icon}
      <span>{children}</span>
    </button>
  );
}

function Dropdown({ children, render }) {
  return (
    <Popover>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className='border-white/10 bg-[#0b1220] p-2 text-white'>
        {render}
      </PopoverContent>
    </Popover>
  );
}

Dropdown.Menu = DropdownMenu;
Dropdown.Item = DropdownItem;

function ModalComponent({
  visible,
  title,
  onCancel,
  onOk,
  children,
  footer,
  okText = '确定',
  cancelText = '取消',
  width,
  className,
}) {
  return (
    <Dialog open={!!visible} onOpenChange={(open) => !open && onCancel?.()}>
      <DialogContent className={cn('border-white/10 bg-black text-white sm:max-w-lg', className)} style={{ width }}>
        {title ? (
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
          </DialogHeader>
        ) : null}
        <div>{children}</div>
        {footer === null ? null : footer ? (
          footer
        ) : (
          <DialogFooter className='border-white/10 bg-transparent'>
            <UiButton type='button' variant='outline' onClick={onCancel}>
              {cancelText}
            </UiButton>
            <UiButton type='button' onClick={onOk}>
              {okText}
            </UiButton>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}

const Modal = Object.assign(ModalComponent, {
  confirm({ title, content, onOk }) {
    if (window.confirm([getTextContent(title), getTextContent(content)].filter(Boolean).join('\n\n'))) {
      return onOk?.();
    }
    return undefined;
  },
  info({ title, content }) {
    toast.custom(() => <NotificationView title={title} content={content} />);
  },
  error({ title, content }) {
    toast.error(getTextContent(title || content) || 'Error');
  },
});

function Highlight({ sourceString = '', searchWords = [], className }) {
  const words = (searchWords || []).filter(Boolean);
  if (!words.length) {
    return <span className={className}>{sourceString}</span>;
  }

  const pattern = new RegExp(`(${words.map((word) => word.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})`, 'ig');
  const parts = String(sourceString).split(pattern);
  const lowerWords = words.map((word) => String(word).toLowerCase());

  return (
    <span className={className}>
      {parts.map((part, index) => (
        lowerWords.includes(String(part).toLowerCase()) ? (
          <mark key={index} className='rounded bg-amber-300/20 px-0.5 text-amber-100'>
            {part}
          </mark>
        ) : (
          <React.Fragment key={index}>{part}</React.Fragment>
        )
      ))}
    </span>
  );
}

function Icon({ svg, ...props }) {
  return <span {...props}>{svg}</span>;
}

function Row({ children, gutter = 0, className, style }) {
  const gap = Array.isArray(gutter) ? gutter[0] : gutter;
  return (
    <div className={cn('flex flex-wrap', className)} style={{ gap, ...style }}>
      {children}
    </div>
  );
}

function Col({ children, xs, sm, md, lg, xl, className, style }) {
  const span = xl || lg || md || sm || xs || 24;
  const width = `${(span / 24) * 100}%`;
  return (
    <div className={className} style={{ width, ...style }}>
      {children}
    </div>
  );
}

function SplitButtonGroup({ children, className }) {
  return <div className={cn('inline-flex items-center gap-px', className)}>{children}</div>;
}

function Skeleton({ loading, placeholder, children, className }) {
  if (!loading) return children || null;
  if (placeholder) return placeholder;
  return <UiSkeleton className={cn('h-4 w-full bg-white/10', className)} />;
}

Skeleton.Title = function SkeletonTitle({ style, className }) {
  return <UiSkeleton className={cn('h-4 bg-white/10', className)} style={style} />;
};

Skeleton.Avatar = function SkeletonAvatar({ style, className, shape }) {
  return (
    <UiSkeleton
      className={cn('size-8 bg-white/10', shape === 'square' ? 'rounded-lg' : 'rounded-full', className)}
      style={style}
    />
  );
};

Skeleton.Image = function SkeletonImage({ style, className }) {
  return <UiSkeleton className={cn('h-24 w-full rounded-xl bg-white/10', className)} style={style} />;
};

function useFormField(field, fallback) {
  const context = useContext(FormContext);
  const value = field && context ? context.values?.[field] : fallback;
  const setValue = (nextValue) => {
    if (field && context) {
      context.setValue(field, nextValue);
    }
  };
  return [value, setValue, context];
}

function Form({ values = {}, getFormApi, children, className, style }) {
  const [formValues, setFormValues] = useState(values || {});

  useEffect(() => {
    setFormValues(values || {});
  }, [values]);

  const api = useMemo(() => ({
    getValues: () => formValues,
    setValue: (field, value) => {
      setFormValues((prev) => ({ ...prev, [field]: value }));
    },
    setValues: (nextValues) => {
      setFormValues((prev) => ({ ...prev, ...(nextValues || {}) }));
    },
  }), [formValues]);

  useEffect(() => {
    getFormApi?.(api);
  }, [api, getFormApi]);

  return (
    <FormContext.Provider value={{ values: formValues, setValue: api.setValue, setValues: api.setValues }}>
      <div className={cn('space-y-4', className)} style={style}>
        {children}
      </div>
    </FormContext.Provider>
  );
}

Form.Section = function FormSection({ text, children }) {
  return (
    <section className='space-y-4'>
      {text ? <div className='text-sm font-semibold text-white'>{text}</div> : null}
      {children}
    </section>
  );
};

function FieldShell({ label, extraText, noLabel, children }) {
  return (
    <div className='space-y-2'>
      {!noLabel && label ? <div className='text-sm font-medium text-white'>{label}</div> : null}
      {children}
      {extraText ? <div className='text-xs text-white/45'>{extraText}</div> : null}
    </div>
  );
}

Form.Slot = function FormSlot({ label, children }) {
  return <FieldShell label={label}>{children}</FieldShell>;
};

Form.Input = function FormInput({ field, label, onChange, value, extraText, noLabel, mode, prefix, suffix, ...props }) {
  const [currentValue, setCurrentValue] = useFormField(field, value ?? '');
  const resolvedValue = currentValue ?? '';
  return (
    <FieldShell label={label} extraText={extraText} noLabel={noLabel}>
      <Input
        {...props}
        mode={mode}
        prefix={prefix}
        suffix={suffix}
        value={resolvedValue}
        onChange={(nextValue) => {
          setCurrentValue(nextValue);
          onChange?.(nextValue);
        }}
      />
    </FieldShell>
  );
};

Form.TextArea = function FormTextArea({ field, label, onChange, value, extraText, autosize, ...props }) {
  const [currentValue, setCurrentValue] = useFormField(field, value ?? '');
  return (
    <FieldShell label={label} extraText={extraText}>
      <Input
        {...props}
        value={currentValue ?? ''}
        autosize={autosize}
        onChange={(nextValue) => {
          setCurrentValue(nextValue);
          onChange?.(nextValue);
        }}
      />
    </FieldShell>
  );
};

Form.Switch = function FormSwitch({ field, label, onChange, extraText, value, checkedText, uncheckedText, ...props }) {
  const [currentValue, setCurrentValue] = useFormField(field, !!value);
  return (
    <FieldShell label={label} extraText={extraText}>
      <div className='flex items-center gap-3'>
        <AntSwitch
          {...props}
          checked={!!currentValue}
          checkedChildren={checkedText}
          unCheckedChildren={uncheckedText}
          onChange={(nextValue) => {
            setCurrentValue(nextValue);
            onChange?.(nextValue);
          }}
        />
      </div>
    </FieldShell>
  );
};

Form.InputNumber = function FormInputNumber({ field, label, onChange, value, extraText, ...props }) {
  const [currentValue, setCurrentValue] = useFormField(field, value);
  return (
    <FieldShell label={label} extraText={extraText}>
      <AntInputNumber
        {...props}
        value={currentValue}
        className='w-full'
        onChange={(nextValue) => {
          setCurrentValue(nextValue);
          onChange?.(nextValue);
        }}
      />
    </FieldShell>
  );
};

Form.DatePicker = function FormDatePicker({ field, label, onChange, value, extraText, ...props }) {
  const [currentValue, setCurrentValue] = useFormField(field, value);
  const resolvedValue = currentValue ? dayjs(currentValue) : null;
  return (
    <FieldShell label={label} extraText={extraText}>
      <AntDatePicker
        {...props}
        value={resolvedValue}
        className='w-full'
        onChange={(_, dateString) => {
          setCurrentValue(dateString);
          onChange?.(dateString);
        }}
      />
    </FieldShell>
  );
};

Form.Select = function FormSelect({
  field,
  label,
  onChange,
  value,
  extraText,
  optionList = [],
  multiple,
  allowCreate,
  onSearch,
  renderSelectedItem,
  innerBottomSlot,
  style,
  ...props
}) {
  const [currentValue, setCurrentValue] = useFormField(
    field,
    value ?? (multiple ? [] : undefined),
  );

  return (
    <FieldShell label={label} extraText={extraText}>
      <AntSelect
        {...props}
        mode={multiple || allowCreate ? 'tags' : undefined}
        value={currentValue}
        style={{ width: '100%', ...style }}
        options={optionList}
        showSearch
        onSearch={onSearch}
        tagRender={
          renderSelectedItem
            ? (tagProps) => {
                const rendered = renderSelectedItem({ label: tagProps.label, value: tagProps.value });
                if (rendered?.content) {
                  return rendered.content;
                }
                return <Tag>{tagProps.label}</Tag>;
              }
            : undefined
        }
        dropdownRender={(menu) => (
          <div>
            {menu}
            {innerBottomSlot}
          </div>
        )}
        onChange={(nextValue) => {
          setCurrentValue(nextValue);
          onChange?.(nextValue);
        }}
      />
    </FieldShell>
  );
};

export {
  Avatar,
  Badge,
  Banner,
  Button,
  Calendar,
  Card,
  Checkbox,
  Col,
  Collapse,
  Collapsible,
  DatePicker,
  Descriptions,
  Divider,
  Dropdown,
  Empty,
  Form,
  Highlight,
  Icon,
  Image,
  Input,
  InputNumber,
  Modal,
  Notification,
  Pagination,
  Popconfirm,
  PopoverCompat as Popover,
  Progress,
  Row,
  Select,
  Skeleton,
  Space,
  Spin,
  SplitButtonGroup,
  TabPane,
  Table,
  Tabs,
  Tag,
  Text,
  TextArea,
  Title,
  Timeline,
  Toast,
  TooltipCompat as Tooltip,
  Typography,
  Switch,
};

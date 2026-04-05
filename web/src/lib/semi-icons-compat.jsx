import React from 'react';
import {
  AlertTriangle,
  ChevronDown,
  ChevronUp,
  Code2,
  Copy,
  Eye,
  EyeOff,
  KeyRound,
  Lock,
  Mail,
  Menu,
  MoreHorizontal,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Settings,
  SquareChevronRight,
  Trash2,
  User,
  X,
} from 'lucide-react';

function withDefaultSize(Component) {
  return function IconComponent(props) {
    return <Component size={16} {...props} />;
  };
}

export const IconSearch = withDefaultSize(Search);
export const IconKey = withDefaultSize(KeyRound);
export const IconEdit = withDefaultSize(Pencil);
export const IconDelete = withDefaultSize(Trash2);
export const IconAlertTriangle = withDefaultSize(AlertTriangle);
export const IconClose = withDefaultSize(X);
export const IconCode = withDefaultSize(Code2);
export const IconChevronDown = withDefaultSize(ChevronDown);
export const IconChevronUp = withDefaultSize(ChevronUp);
export const IconEyeOpened = withDefaultSize(Eye);
export const IconEyeClosed = withDefaultSize(EyeOff);
export const IconPlus = withDefaultSize(Plus);
export const IconMore = withDefaultSize(MoreHorizontal);
export const IconRefresh = withDefaultSize(RefreshCw);
export const IconCopy = withDefaultSize(Copy);
export const IconExit = withDefaultSize(SquareChevronRight);
export const IconUserSetting = withDefaultSize(Settings);
export const IconMail = withDefaultSize(Mail);
export const IconLock = withDefaultSize(Lock);
export const IconUser = withDefaultSize(User);
export const IconTreeTriangleDown = withDefaultSize(ChevronDown);
export const IconMenu = withDefaultSize(Menu);

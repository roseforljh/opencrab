
import React from 'react';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

export default function HttpStatusCodeRulesInput(props) {
  const {
    label,
    field,
    placeholder,
    extraText,
    onChange,
    parsed,
    invalidText,
    value,
  } = props;

  return (
    <div className="space-y-2">
      <Label htmlFor={field} className="text-white/80">{label}</Label>
      <Input
        id={field}
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
        className='!bg-black/40 border-white/10 text-white placeholder:text-white/30 focus-visible:ring-white/20'
      />
      {extraText && <div className="text-xs text-white/50">{extraText}</div>}
      
      {parsed?.ok && parsed.tokens?.length > 0 && (
        <div className="flex flex-wrap gap-2 mt-2">
          {parsed.tokens.map((token) => (
            <Badge key={token} variant="secondary" className="bg-white/10 text-white/80 hover:bg-white/20">
              {token}
            </Badge>
          ))}
        </div>
      )}
      {!parsed?.ok && (
        <div className="text-sm text-red-500 mt-2 block">
          {invalidText}
          {parsed?.invalidTokens && parsed.invalidTokens.length > 0
            ? `: ${parsed.invalidTokens.join(', ')}`
            : ''}
        </div>
      )}
    </div>
  );
}

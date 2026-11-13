import { useAppState } from '@/state/useAppState';
import { useState, FormEvent, ChangeEvent } from 'react';
import { Input } from './Input';
import { Textarea } from './Textarea';
import { Button } from './Button';

export interface RequestFormProps {
  form: {
    type: 'absence' | 'access';
    title: string;
    fields: Array<{
      name: string;
      label: string;
      type: 'text' | 'textarea' | 'date';
      value: string;
      required?: boolean;
      readonly?: boolean;
      min?: string;
      maxLength?: number;
    }>;
  };
  onSubmit: (data: Record<string, string>) => void;
  onCancel: () => void;
}

export function RequestForm({ form, onSubmit, onCancel }: RequestFormProps) {
  const { serviceOffline, setServiceOffline } = useAppState();
  const [expires, setExpires] = useState(0);
  const [completed, setCompleted] = useState(false);
  const [values, setValues] = useState<Record<string, string>>(
    form.fields.reduce((acc, f) => ({ ...acc, [f.name]: f.value }), {})
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [showConfirmation, setShowConfirmation] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const handleChange = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setValues((prev) => ({ ...prev, [name]: value }));
    if (errors[name]) {
      setErrors((prev) => ({ ...prev, [name]: '' }));
    }

    // Handle date validation for absence form
    if (form.type === 'absence' && name === 'from') {
      const toValue = values.to;
      if (toValue && value > toValue) {
        setValues((prev) => ({ ...prev, to: value }));
      }
    }
  };

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (form.type === 'absence') {
      if (!values.from) newErrors.from = 'Укажи первый день';
      if (!values.to) newErrors.to = 'Укажи последний день';
      if (values.from && values.to && values.to < values.from) {
        newErrors.to = 'Последний день не может быть раньше первого';
      }
    } else if (form.type === 'access') {
      if (!values.reason?.trim()) newErrors.reason = 'Укажи цель доступа';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    setExpires(Date.now() + 600000);
    setShowConfirmation(true);
  };

  const handleConfirm = () => {
    if (submitting || completed) return;
    if (Date.now() > expires) { setShowConfirmation(false); return; }
    setSubmitting(true);
    // Simulate API call
    setTimeout(() => {
      setSubmitting(false);
      setCompleted(true);
      onSubmit(values);
      setShowConfirmation(false);
    }, 700);
  };

  const renderFields = () => {
    return form.fields.map((field) => {
      const error = errors[field.name];
      const commonProps = {
        value: values[field.name],
        onChange: handleChange,
        error,
        required: field.required,
        readOnly: field.readonly,
        min: field.min,
        maxLength: field.maxLength,
      };

      if (field.type === 'textarea') {
        return <Textarea key={field.name} label={field.label} name={field.name} {...commonProps} rows={3} />;
      }
      return <Input key={field.name} label={field.label} name={field.name} type={field.type} {...commonProps} />;
    });
  };

  if (completed) return <p role="status">Заявка передана на обработку. Результат — в разделе «Мои заявки».</p>;

  if (showConfirmation) {
    return (
      <div className="confirmation-screen" style={{ marginTop: 'var(--space-4)', padding: 'var(--space-4)', border: '1px solid var(--border)', borderRadius: 'var(--radius-md)', background: 'var(--surface)' }}>
        <strong style={{ display: 'block', marginBottom: 'var(--space-3)', fontSize: 'var(--text-sm)' }}>Подтверди действие</strong>
        <dl className="summary-grid" style={{ display: 'grid', gridTemplateColumns: 'minmax(100px, 1fr) 2fr', gap: 'var(--space-2) var(--space-4)', fontSize: 'var(--text-sm)', marginBottom: 'var(--space-4)' }}>
          <dt style={{ color: 'var(--muted)' }}>Сотрудник</dt>
          <dd style={{ margin: 0, overflowWrap: 'anywhere' }}>Алексей Смирнов (демо)</dd>
          {form.type === 'access' ? (
            <>
              <dt style={{ color: 'var(--muted)' }}>Ресурс</dt>
              <dd style={{ margin: 0, overflowWrap: 'anywhere' }}>{values.resource}</dd>
              <dt style={{ color: 'var(--muted)' }}>Права</dt>
              <dd style={{ margin: 0, overflowWrap: 'anywhere' }}>Чтение</dd>
              <dt style={{ color: 'var(--muted)' }}>Обоснование</dt>
              <dd style={{ margin: 0, overflowWrap: 'anywhere' }}>{values.reason}</dd>
            </>
          ) : (
            <>
              <dt style={{ color: 'var(--muted)' }}>Тип</dt>
              <dd style={{ margin: 0, overflowWrap: 'anywhere' }}>Отгул</dd>
              <dt style={{ color: 'var(--muted)' }}>Даты</dt>
              <dd style={{ margin: 0, overflowWrap: 'anywhere' }}>{values.from} — {values.to}</dd>
              <dt style={{ color: 'var(--muted)' }}>Комментарий</dt>
              <dd style={{ margin: 0, overflowWrap: 'anywhere' }}>{values.comment || 'Не указан'}</dd>
            </>
          )}
        </dl>
        <p style={{ margin: '0 0 var(--space-4)', fontSize: 'var(--text-sm)', color: 'var(--fg-2)', lineHeight: 'var(--leading-body)' }}>
          Действие создаст только твою заявку в локальной имитации сервиса. Подтверждение действительно 10 минут. Реальная отправка не произойдёт.
        </p>
        <label className="demo-switch" style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', fontSize: 'var(--text-sm)', color: 'var(--muted)', marginBottom: 'var(--space-3)' }}>
          <input
            type="checkbox"
            checked={serviceOffline}
            onChange={(e) => setServiceOffline(e.target.checked)}
            style={{ width: '16px', height: '16px', accentColor: 'var(--accent)' }}
          />
          Демо: сервис недоступен
        </label>
        <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
          <Button variant="primary" onClick={handleConfirm} disabled={submitting} style={{ flex: 1 }}>
            {submitting ? 'Отправка…' : 'Подтвердить заявку'}
          </Button>
          <Button variant="text" onClick={() => setShowConfirmation(false)} style={{ flex: 1 }}>
            Изменить
          </Button>
        </div>
      </div>
    );
  }

  return (
    <form className="request-form" onSubmit={handleSubmit} style={{ marginTop: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
      <strong style={{ fontSize: 'var(--text-sm)', color: 'var(--fg)' }}>
        {form.type === 'access' ? '01 / Данные запроса доступа' : '01 / Данные отгула'}
      </strong>
      {renderFields()}
      <p style={{ margin: 0, fontSize: 'var(--text-xs)', color: 'var(--muted)' }}>
        Только для Алексея Смирнова · Демо. Не указывай медицинские причины, сведения об оплате и другие чувствительные данные.
      </p>
      <div style={{ display: 'flex', gap: 'var(--space-2)', marginTop: 'var(--space-2)' }}>
        <Button type="submit" variant="primary" style={{ flex: 1 }}>
          Проверить заявку
        </Button>
        <Button type="button" variant="text" onClick={onCancel} style={{ flex: 1 }}>
          Отмена
        </Button>
      </div>
    </form>
  );
}
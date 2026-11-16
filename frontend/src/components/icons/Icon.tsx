import { SVGAttributes } from 'react';
import spriteUrl from '../../assets/icons/sprite.svg?url';

export interface IconProps extends SVGAttributes<SVGSVGElement> {
  name: string;
  size?: number | string;
}

export function Icon({
  name,
  size = 20,
  className,
  'aria-hidden': ariaHidden = 'true',
  ...props
}: IconProps) {
  return (
    <svg className={className} width={size} height={size} aria-hidden={ariaHidden} {...props}>
      <use href={`${spriteUrl}#${name}`} />
    </svg>
  );
}

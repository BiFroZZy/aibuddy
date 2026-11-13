import { SVGAttributes } from 'react';

export interface IconProps extends SVGAttributes<SVGSVGElement> {
  name: string;
  size?: number | string;
}

export function Icon({ name, size = 20, className, 'aria-hidden': ariaHidden = 'true', ...props }: IconProps) {
  return (
    <svg
      className={className}
      width={size}
      height={size}
      aria-hidden={ariaHidden}
      {...props}
    >
      <use href={`/src/assets/icons/sprite.svg#${name}`} />
    </svg>
  );
}
import logo from '../assets/virtforge-logo.png';

type VirtForgeLogoProps = {
  size?: number;
  className?: string;
};

export function VirtForgeLogo({ size = 40, className = '' }: VirtForgeLogoProps) {
  return (
    <img
      src={logo}
      alt="VirtForge"
      width={size}
      height={size}
      className={`shrink-0 rounded-xl object-cover ${className}`}
      draggable={false}
    />
  );
}

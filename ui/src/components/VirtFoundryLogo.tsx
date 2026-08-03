import logo from '../assets/virtfoundry-logo.png';

type VirtFoundryLogoProps = {
  size?: number;
  className?: string;
};

export function VirtFoundryLogo({ size = 40, className = '' }: VirtFoundryLogoProps) {
  return (
    <img
      src={logo}
      alt="VirtFoundry"
      width={size}
      height={size}
      className={`shrink-0 rounded-xl object-cover ${className}`}
      draggable={false}
    />
  );
}

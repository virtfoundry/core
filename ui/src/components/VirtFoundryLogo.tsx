import clsx from 'clsx';
import { useTheme } from '../lib/theme';

const logoLight = '/virtfounfry-light.png';
const logoDark = '/virtfounfry-dark.png';
const iconLight = '/virtfounfry-icon-light.png';
const iconDark = '/virtfounfry-icon-dark.png';

const LOGO_ASSETS = [logoLight, logoDark, iconLight, iconDark];

let logosPreloaded = false;
function preloadLogos() {
  if (logosPreloaded || typeof document === 'undefined') return;
  logosPreloaded = true;
  for (const src of LOGO_ASSETS) {
    const img = new Image();
    img.src = src;
  }
}
preloadLogos();

type VirtFoundryLogoProps = {
  /** Logo height in px (ignored when fullWidth). */
  height?: number;
  className?: string;
  /** Force dark-style logo (e.g. login hero on blue). */
  variant?: 'light' | 'dark';
  /** Sidebar collapsed: show only the cube mark. */
  iconOnly?: boolean;
  /** Fill the container width (sidebar header). */
  fullWidth?: boolean;
};

function LogoStack({
  lightSrc,
  darkSrc,
  isDark,
  alt,
  className,
  imgClass,
  style,
}: {
  lightSrc: string;
  darkSrc: string;
  isDark: boolean;
  alt: string;
  className?: string;
  imgClass: string;
  style?: React.CSSProperties;
}) {
  return (
    <span className={clsx('grid shrink-0 [&>img]:col-start-1 [&>img]:row-start-1', className)} style={style}>
      <img
        src={lightSrc}
        alt={isDark ? '' : alt}
        aria-hidden={isDark}
        className={clsx(imgClass, isDark && 'invisible')}
        decoding="sync"
        draggable={false}
      />
      <img
        src={darkSrc}
        alt={isDark ? alt : ''}
        aria-hidden={!isDark}
        className={clsx(imgClass, !isDark && 'invisible')}
        decoding="sync"
        draggable={false}
      />
    </span>
  );
}

export function VirtFoundryLogo({
  height = 32,
  className = '',
  variant,
  iconOnly = false,
  fullWidth = false,
}: VirtFoundryLogoProps) {
  const { theme } = useTheme();
  const isDark = variant ? variant === 'dark' : theme === 'dark';

  const imgClass = clsx(
    'select-none',
    iconOnly ? 'object-contain' : 'object-contain object-left',
    fullWidth && !iconOnly && 'h-auto w-full',
  );

  const sizeStyle = iconOnly
    ? { width: height, height, minWidth: height }
    : fullWidth
      ? { width: '100%' }
      : { height, width: 'auto' as const };

  if (iconOnly) {
    return (
      <LogoStack
        lightSrc={iconLight}
        darkSrc={iconDark}
        isDark={isDark}
        alt="VirtFoundry"
        className={className}
        imgClass={imgClass}
        style={sizeStyle}
      />
    );
  }

  return (
    <LogoStack
      lightSrc={logoLight}
      darkSrc={logoDark}
      isDark={isDark}
      alt="VirtFoundry Cloud Orchestrator"
      className={clsx(fullWidth && 'w-full', className)}
      imgClass={imgClass}
      style={sizeStyle}
    />
  );
}

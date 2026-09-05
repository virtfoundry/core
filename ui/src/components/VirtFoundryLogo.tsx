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
  imgStyle,
  containerStyle,
  layout = 'inline',
}: {
  lightSrc: string;
  darkSrc: string;
  isDark: boolean;
  alt: string;
  className?: string;
  imgClass: string;
  imgStyle?: React.CSSProperties;
  containerStyle?: React.CSSProperties;
  layout?: 'inline' | 'block';
}) {
  return (
    <span
      className={clsx(
        layout === 'block' ? 'block w-full' : 'inline-flex shrink-0 items-center overflow-hidden',
        className,
      )}
      style={containerStyle}
    >
      <img
        src={lightSrc}
        alt={isDark ? '' : alt}
        aria-hidden={isDark}
        className={clsx(imgClass, isDark && 'hidden')}
        style={imgStyle}
        decoding="sync"
        draggable={false}
      />
      <img
        src={darkSrc}
        alt={isDark ? alt : ''}
        aria-hidden={!isDark}
        className={clsx(imgClass, !isDark && 'hidden')}
        style={imgStyle}
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

  const effectiveFullWidth = fullWidth && !iconOnly;

  const imgClass = clsx(
    'select-none block object-left',
    iconOnly
      ? 'object-cover h-full w-full'
      : effectiveFullWidth
        ? 'object-contain h-auto w-full'
        : 'object-contain h-auto w-auto max-w-full',
  );

  const imgStyle: React.CSSProperties | undefined = iconOnly
    ? undefined
    : effectiveFullWidth
      ? undefined
      : { height, width: 'auto' };

  if (iconOnly) {
    return (
      <LogoStack
        lightSrc={logoLight}
        darkSrc={logoDark}
        isDark={isDark}
        alt="VirtFoundry"
        className={clsx('overflow-hidden', className)}
        containerStyle={{ width: height, height, minWidth: height, minHeight: height }}
        imgClass={imgClass}
        imgStyle={imgStyle}
      />
    );
  }

  return (
    <LogoStack
      lightSrc={logoLight}
      darkSrc={logoDark}
      isDark={isDark}
      alt="VirtFoundry Cloud Orchestrator"
      layout={effectiveFullWidth ? 'block' : 'inline'}
      className={className}
      imgClass={imgClass}
      imgStyle={imgStyle}
    />
  );
}

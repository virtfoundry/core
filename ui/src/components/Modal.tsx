import { ReactNode } from 'react';
import { X } from 'lucide-react';
import clsx from 'clsx';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}

const sizeStyles = {
  sm: 'max-w-md',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
};

export function Modal({ isOpen, onClose, title, children, size = 'md' }: ModalProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex min-h-full items-center justify-center p-4">
        <div className="fixed inset-0 bg-black/50 backdrop-blur-[1px]" onClick={onClose} />
        <div className={clsx('relative vf-card shadow-xl w-full', sizeStyles[size])}>
          <div className="flex items-center justify-between px-6 py-4 border-b border-card-border">
            <h3 className="font-headline text-headline-md font-semibold text-on-surface">{title}</h3>
            <button
              type="button"
              onClick={onClose}
              className="btn-icon-neutral p-2 text-on-surface-variant"
            >
              <X size={20} />
            </button>
          </div>
          <div className="px-6 py-4">{children}</div>
        </div>
      </div>
    </div>
  );
}

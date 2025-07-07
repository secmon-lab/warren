import * as React from "react";

type ToastVariant = "default" | "destructive" | "success" | "warning" | "info";

interface ToastProps {
  id: string;
  title?: string;
  description?: string;
  variant?: ToastVariant;
  duration?: number;
}

interface ToastContextValue {
  toasts: ToastProps[];
  addToast: (props: Omit<ToastProps, "id">) => void;
  removeToast: (id: string) => void;
}

const TOAST_LIMIT = 5;
const TOAST_REMOVE_DELAY = 5000;

const ToastContext = React.createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = React.useState<ToastProps[]>([]);

  const removeToast = React.useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  const addToast = React.useCallback(
    (props: Omit<ToastProps, "id">) => {
      const id = Math.random().toString(36).substr(2, 9);
      const toast: ToastProps = {
        id,
        duration: TOAST_REMOVE_DELAY,
        ...props,
      };

      setToasts((prev) => {
        const newToasts = [...prev, toast];
        return newToasts.slice(-TOAST_LIMIT);
      });

      setTimeout(() => {
        removeToast(id);
      }, toast.duration);
    },
    [removeToast]
  );

  const contextValue = React.useMemo(
    () => ({
      toasts,
      addToast,
      removeToast,
    }),
    [toasts, addToast, removeToast]
  );

  return (
    <ToastContext.Provider value={contextValue}>
      {children}
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = React.useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within a ToastProvider");
  }

  return {
    toast: context.addToast,
    toasts: context.toasts,
    dismiss: context.removeToast,
  };
}

export function useErrorToast() {
  const { toast } = useToast();

  return React.useCallback(
    (message: string, title?: string) => {
      toast({
        title: title || "Error",
        description: message,
        variant: "destructive",
      });
    },
    [toast]
  );
}

export function useSuccessToast() {
  const { toast } = useToast();

  return React.useCallback(
    (message: string, title?: string) => {
      toast({
        title: title || "Success",
        description: message,
        variant: "success",
      });
    },
    [toast]
  );
}

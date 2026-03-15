
import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react';



type ToastType = 'success' | 'error' | 'info';



interface Toast {

  id: number;

  message: string;

  type: ToastType;

}



interface ToastContextType {

  showToast: (message: string, type?: ToastType) => void;

}



const ToastContext = createContext<ToastContextType | undefined>(undefined);



let nextId = 0;



export const ToastProvider: React.FC<{ children: ReactNode }> = ({ children }) => {

  const [toasts, setToasts] = useState<Toast[]>([]);



  const showToast = useCallback((message: string, type: ToastType = 'info') => {

    const id = ++nextId;

    setToasts((prev) => [...prev, { id, message, type }]);

    setTimeout(() => {

      setToasts((prev) => prev.filter((t) => t.id !== id));

    }, 4000);

  }, []);



  const dismiss = (id: number) => setToasts((prev) => prev.filter((t) => t.id !== id));



  const BG: Record<ToastType, string> = {

    success: '#16a34a',

    error: '#dc2626',

    info: '#2563eb',

  };



  return (

    <ToastContext.Provider value={{ showToast }}>

      {children}

      {/* Toast container */}

      <div

        style={{

          position: 'fixed',

          bottom: '1.5rem',

          right: '1.5rem',

          zIndex: 9999,

          display: 'flex',

          flexDirection: 'column',

          gap: '0.5rem',

          maxWidth: '360px',

        }}

        aria-live="polite"

        aria-atomic="false"

      >

        {toasts.map((t) => (

          <div

            key={t.id}

            role="alert"

            style={{

              backgroundColor: BG[t.type],

              borderRadius: '6px',

              boxShadow: '0 4px 12px rgba(0,0,0,0.2)',

              color: '#fff',

              display: 'flex',

              alignItems: 'center',

              justifyContent: 'space-between',

              fontSize: '0.875rem',

              fontWeight: 500,

              gap: '0.75rem',

              padding: '0.75rem 1rem',

              animation: 'slideIn 0.2s ease',

            }}

          >

            <span>{t.message}</span>

            <button

              onClick={() => dismiss(t.id)}

              aria-label="Dismiss"

              style={{

                background: 'none',

                border: 'none',

                color: 'rgba(255,255,255,0.8)',

                cursor: 'pointer',

                fontSize: '1rem',

                lineHeight: 1,

                padding: 0,

                flexShrink: 0,

              }}

            >

              ✕

            </button>

          </div>

        ))}

      </div>

    </ToastContext.Provider>

  );

};



export const useToast = (): ToastContextType => {

  const ctx = useContext(ToastContext);

  if (!ctx) throw new Error('useToast must be used within ToastProvider');

  return ctx;

};

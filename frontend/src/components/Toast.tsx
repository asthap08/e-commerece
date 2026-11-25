"use client";
import { useEffect } from "react";

type ToastProps = {
  message: string;
  type?: "success" | "error" | "info";
  onClose: () => void;
  duration?: number;
};

export default function Toast({ message, type = "info", onClose, duration = 3000 }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(onClose, duration);
    return () => clearTimeout(timer);
  }, [onClose, duration]);

  const colors = {
    success: { bg: "#d1fae5", border: "#a7f3d0", text: "#065f46" },
    error: { bg: "#fee2e2", border: "#fecaca", text: "#991b1b" },
    info: { bg: "#dbeafe", border: "#bfdbfe", text: "#1e40af" },
  };
  const c = colors[type];

  return (
    <div
      style={{
        position: "fixed",
        bottom: "2rem",
        right: "2rem",
        zIndex: 9999,
        background: c.bg,
        border: `1px solid ${c.border}`,
        color: c.text,
        borderRadius: "0.5rem",
        padding: "1rem 1.5rem",
        boxShadow: "0 4px 16px rgba(0,0,0,0.1)",
        maxWidth: "360px",
        animation: "slideInRight 0.3s ease",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "1rem" }}>
        <span style={{ fontSize: "0.875rem", fontWeight: 500 }}>{message}</span>
        <button
          onClick={onClose}
          style={{ background: "none", border: "none", cursor: "pointer", color: c.text, fontSize: "1.25rem", lineHeight: 1 }}
        >
          ×
        </button>
      </div>
      <style>{`@keyframes slideInRight { from { transform: translateX(110%); opacity: 0; } to { transform: translateX(0); opacity: 1; } }`}</style>
    </div>
  );
}

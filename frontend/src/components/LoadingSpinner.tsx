"use client";

export default function LoadingSpinner({ size = 24 }: { size?: number }) {
  return (
    <div style={{ display: "flex", alignItems: "center", justifyContent: "center", padding: "2rem" }}>
      <div
        style={{
          width: size,
          height: size,
          border: "2px solid #e2e8f0",
          borderTopColor: "#064e3b",
          borderRadius: "50%",
          animation: "spin 0.8s linear infinite",
        }}
      />
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}

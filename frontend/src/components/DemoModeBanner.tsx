import { DEMO_MODE } from "@/lib/demo";

export default function DemoModeBanner() {
  if (!DEMO_MODE) return null;

  return (
    <div
      role="status"
      style={{
        width: "100%",
        padding: "0.4rem 1rem",
        background: "#fef9c3",
        color: "#422006",
        fontSize: "0.75rem",
        fontWeight: 700,
        lineHeight: 1.4,
        textAlign: "center",
      }}
    >
      Demo store · Orders and payments are simulated
    </div>
  );
}

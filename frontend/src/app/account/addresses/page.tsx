"use client";
import { useEffect, useState } from "react";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { MapPin, Plus, Trash2, Check } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useCustomerAuth } from "@/lib/use-customer-auth";

type Address = {
  id: string;
  label: string;
  line1: string;
  line2?: string;
  city: string;
  state: string;
  pincode: string;
  country: string;
  is_default: boolean;
};

export default function AddressesPage() {
  const authorized = useCustomerAuth();
  const [addresses, setAddresses] = useState<Address[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ label: "Home", line1: "", line2: "", city: "", state: "", pincode: "", country: "IN" });

  useEffect(() => {
    if (!authorized) return;
    fetchAddresses();
  }, [authorized]);

  async function fetchAddresses() {
    setLoading(true);
    try {
      const res = await apiFetch("/api/v1/addresses");
      if (res.ok) {
        const data = await res.json();
        setAddresses(data.data || []);
      }
    } catch { /* ignore */ }
    setLoading(false);
  }

  async function addAddress(e: React.FormEvent) {
    e.preventDefault();
    try {
      const res = await apiFetch("/api/v1/addresses", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      });
      if (res.ok) {
        setShowForm(false);
        setForm({ label: "Home", line1: "", line2: "", city: "", state: "", pincode: "", country: "IN" });
        fetchAddresses();
      }
    } catch { /* ignore */ }
  }

  async function deleteAddress(id: string) {
    try {
      await apiFetch(`/api/v1/addresses/${id}`, { method: "DELETE" });
      fetchAddresses();
    } catch { /* ignore */ }
  }

  async function setDefault(id: string) {
    try {
      await apiFetch(`/api/v1/addresses/${id}/default`, { method: "PUT" });
      fetchAddresses();
    } catch { /* ignore */ }
  }

  if (!authorized) return null;

  return (
    <>
      <Navbar />
      <main style={{ paddingTop: "90px", minHeight: "100vh" }} className="container-max">
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "2rem" }}>
          <h1 style={{ margin: 0, fontSize: "1.75rem", fontWeight: 800 }}>Your Addresses</h1>
          <button onClick={() => setShowForm(!showForm)} style={{ display: "flex", alignItems: "center", gap: "0.5rem", padding: "0.625rem 1rem", background: "#064e3b", color: "white", border: "none", borderRadius: "999px", fontWeight: 600, cursor: "pointer" }}>
            <Plus size={16} /> Add Address
          </button>
        </div>

        {showForm && (
          <form onSubmit={addAddress} style={{ border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-lg)", padding: "2rem", background: "var(--bg-card)", marginBottom: "2rem" }}>
            <h2 style={{ margin: "0 0 1rem", fontSize: "1.25rem", fontWeight: 700 }}>Add New Address</h2>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1rem" }}>
              <div>
                <label className="form-label">Label</label>
                <select className="input-field" value={form.label} onChange={(e) => setForm({ ...form, label: e.target.value })}>
                  <option>Home</option>
                  <option>Work</option>
                  <option>Other</option>
                </select>
              </div>
              <div>
                <label className="form-label">PIN Code</label>
                <input className="input-field" value={form.pincode} onChange={(e) => setForm({ ...form, pincode: e.target.value })} required />
              </div>
            </div>
            <div style={{ marginTop: "1rem" }}>
              <label className="form-label">Street Address</label>
              <input className="input-field" value={form.line1} onChange={(e) => setForm({ ...form, line1: e.target.value })} required />
            </div>
            <div style={{ marginTop: "1rem" }}>
              <label className="form-label">Apartment/Suite (optional)</label>
              <input className="input-field" value={form.line2} onChange={(e) => setForm({ ...form, line2: e.target.value })} />
            </div>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1rem", marginTop: "1rem" }}>
              <div>
                <label className="form-label">City</label>
                <input className="input-field" value={form.city} onChange={(e) => setForm({ ...form, city: e.target.value })} required />
              </div>
              <div>
                <label className="form-label">State</label>
                <input className="input-field" value={form.state} onChange={(e) => setForm({ ...form, state: e.target.value })} required />
              </div>
            </div>
            <div style={{ display: "flex", gap: "0.75rem", marginTop: "1.5rem" }}>
              <button type="button" onClick={() => setShowForm(false)} style={{ flex: 1, padding: "0.75rem", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-md)", background: "transparent", cursor: "pointer", fontWeight: 600, color: "#6b7280" }}>Cancel</button>
              <button type="submit" style={{ flex: 1, padding: "0.75rem", border: "none", borderRadius: "var(--radius-md)", background: "#064e3b", color: "white", cursor: "pointer", fontWeight: 600 }}>Save Address</button>
            </div>
          </form>
        )}

        {loading ? (
          <div style={{ textAlign: "center", padding: "3rem" }}>Loading...</div>
        ) : addresses.length === 0 ? (
          <div style={{ textAlign: "center", padding: "3rem", color: "#6b7280" }}>
            <MapPin size={48} style={{ margin: "0 auto 1rem", opacity: 0.3 }} />
            <p>No addresses saved yet.</p>
          </div>
        ) : (
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(300px, 1fr))", gap: "1.5rem" }}>
            {addresses.map((addr) => (
              <div key={addr.id} style={{ border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-lg)", padding: "1.5rem", background: "var(--bg-card)", position: "relative" }}>
                {addr.is_default && (
                  <span style={{ position: "absolute", top: "0.75rem", right: "0.75rem", fontSize: "0.7rem", padding: "0.2rem 0.5rem", background: "#d1fae5", color: "#065f46", borderRadius: "4px", fontWeight: 600 }}>Default</span>
                )}
                <div style={{ fontWeight: 700, fontSize: "0.95rem", marginBottom: "0.5rem" }}>{addr.label}</div>
                <div style={{ fontSize: "0.85rem", color: "#6b7280", lineHeight: 1.6 }}>
                  {addr.line1}{addr.line2 ? `, ${addr.line2}` : ""}<br />
                  {addr.city}, {addr.state} {addr.pincode}<br />
                  {addr.country}
                </div>
                <div style={{ display: "flex", gap: "0.5rem", marginTop: "1rem" }}>
                  {!addr.is_default && (
                    <button onClick={() => setDefault(addr.id)} style={{ padding: "0.4rem 0.75rem", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-md)", background: "transparent", cursor: "pointer", fontSize: "0.8rem", color: "#6b7280" }}>
                      <Check size={14} style={{ marginRight: "0.25rem" }} /> Set Default
                    </button>
                  )}
                  <button onClick={() => deleteAddress(addr.id)} style={{ padding: "0.4rem 0.75rem", border: "1px solid #fecaca", borderRadius: "var(--radius-md)", background: "transparent", cursor: "pointer", fontSize: "0.8rem", color: "#dc2626" }}>
                    <Trash2 size={14} style={{ marginRight: "0.25rem" }} /> Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
      <Footer />
    </>
  );
}

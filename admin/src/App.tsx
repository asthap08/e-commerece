import { useCallback, useEffect, useState } from 'react'
import './App.css'
import { api, ApiError, fmtDate, money, toMinor } from './api'

type Me = { id: string; email: string; role: string }
type Tab = 'dashboard' | 'orders' | 'products' | 'inventory' | 'coupons' | 'returns'
const TABS: { id: Tab; label: string }[] = [
  { id: 'dashboard', label: 'Dashboard' }, { id: 'orders', label: 'Orders' }, { id: 'products', label: 'Products' },
  { id: 'inventory', label: 'Inventory' }, { id: 'coupons', label: 'Coupons' }, { id: 'returns', label: 'Returns' },
]

function useLoad<T>(loader: () => Promise<T>, deps: unknown[]) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [tick, setTick] = useState(0)
  useEffect(() => {
    let live = true
    // eslint-disable-next-line react-hooks/set-state-in-effect -- loading flag mirrors the request lifecycle
    setLoading(true)
    loader().then((d) => live && (setData(d), setError(''))).catch((e: unknown) => live && setError(e instanceof Error ? e.message : 'Failed to load')).finally(() => live && setLoading(false))
    return () => { live = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick])
  return { data, error, loading, reload: () => setTick((t) => t + 1) }
}

function Notice({ kind, children }: { kind: 'ok' | 'err'; children: React.ReactNode }) {
  return <p className={`notice ${kind}`} role={kind === 'err' ? 'alert' : 'status'}>{children}</p>
}

function Pager({ total, offset, limit, onChange }: { total: number; offset: number; limit: number; onChange: (o: number) => void }) {
  if (total <= limit) return null
  return (
    <div className="pager">
      <button disabled={offset === 0} onClick={() => onChange(Math.max(0, offset - limit))}>← Prev</button>
      <span>{offset + 1}–{Math.min(offset + limit, total)} of {total}</span>
      <button disabled={offset + limit >= total} onClick={() => onChange(offset + limit)}>Next →</button>
    </div>
  )
}

/* ---------------------------------------------------------------- login */
function Login({ onDone }: { onDone: (m: Me) => void }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  const submit = async (e: React.FormEvent) => {
    e.preventDefault(); setBusy(true); setErr('')
    try {
      const r = await api<{ user: Me }>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) })
      if (r.user.role !== 'admin') { await api('/api/v1/auth/logout', { method: 'POST' }).catch(() => undefined); throw new Error('This account does not have admin access.') }
      onDone(r.user)
    } catch (e2) { setErr(e2 instanceof Error ? e2.message : 'Sign-in failed') } finally { setBusy(false) }
  }
  return (
    <main className="login">
      <form onSubmit={submit} className="card">
        <h1>OmniRetail Admin</h1>
        <label>Email<input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="username" required /></label>
        <label>Password<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required /></label>
        {err && <Notice kind="err">{err}</Notice>}
        <button className="primary" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
      </form>
    </main>
  )
}

/* ------------------------------------------------------------ dashboard */
type Dash = { orders: number; open_orders: number; revenue_minor: number; products: number; active_products: number; low_stock: number; out_of_stock: number; customers: number; pending_returns: number; orders_today: number; revenue_today_minor: number }
function Dashboard({ go }: { go: (t: Tab) => void }) {
  const { data, error, loading } = useLoad(() => api<Dash>('/api/v1/admin/dashboard'), [])
  if (loading) return <p>Loading…</p>
  if (error || !data) return <Notice kind="err">{error}</Notice>
  const cards: [string, string, Tab?][] = [
    ['Revenue (paid + fulfilled)', money(data.revenue_minor)], ['Revenue today', money(data.revenue_today_minor)],
    ['Orders', String(data.orders), 'orders'], ['Open orders', String(data.open_orders), 'orders'], ['Orders today', String(data.orders_today), 'orders'],
    ['Active products', `${data.active_products} / ${data.products}`, 'products'], ['Low stock', String(data.low_stock), 'inventory'], ['Out of stock', String(data.out_of_stock), 'inventory'],
    ['Customers', String(data.customers)], ['Return requests to review', String(data.pending_returns), 'returns'],
  ]
  return <div className="cards">{cards.map(([l, v, t]) => <button key={l} className="stat" onClick={() => t && go(t)} disabled={!t}><span>{l}</span><strong>{v}</strong></button>)}</div>
}

/* --------------------------------------------------------------- orders */
type OrderRow = { id: string; status: string; total_minor: number; created_at: string; customer_name: string; customer_email: string; payment_method: string; payment_status?: string; item_count?: number; tracking_number: string }
type OrderFull = OrderRow & { items: { id: string; name: string; sku: string; quantity: number; price_minor: number }[]; shipping_address: Record<string, string> | null; shipping_method: string; shipping_cost_minor: number; discount_minor: number; coupon_code?: string; customer_phone: string; tracking_carrier: string }
const NEXT: Record<string, string[]> = { pending: ['processing', 'cancelled'], paid: ['processing', 'cancelled'], processing: ['shipped', 'cancelled'], shipped: ['delivered'], payment_failed: ['cancelled'] }
const STATUSES = ['pending', 'paid', 'processing', 'shipped', 'delivered', 'cancelled', 'payment_failed']

function OrderDetail({ id, onClose, onChanged }: { id: string; onClose: () => void; onChanged: () => void }) {
  const { data: o, error, reload } = useLoad(() => api<OrderFull>(`/api/v1/orders/${id}`), [id])
  const [msg, setMsg] = useState<{ k: 'ok' | 'err'; t: string } | null>(null)
  const [num, setNum] = useState(''); const [carrier, setCarrier] = useState('')
  const act = async (fn: () => Promise<unknown>, ok: string) => {
    setMsg(null)
    try { await fn(); setMsg({ k: 'ok', t: ok }); reload(); onChanged() } catch (e) { setMsg({ k: 'err', t: e instanceof Error ? e.message : 'Failed' }) }
  }
  return (
    <aside className="drawer" aria-label="Order detail">
      <button className="close" onClick={onClose} aria-label="Close">×</button>
      {error && <Notice kind="err">{error}</Notice>}
      {o && (<>
        <h2>Order #{o.id.slice(0, 8)} <span className={`pill ${o.status}`}>{o.status}</span></h2>
        <p className="muted">{fmtDate(o.created_at)} · {o.payment_method.toUpperCase()}{o.payment_status ? ` (${o.payment_status})` : ''}</p>
        <h3>Customer</h3>
        <p>{o.customer_name}<br />{o.customer_email} · {o.customer_phone}</p>
        {o.shipping_address && <p className="muted">{[o.shipping_address.address, o.shipping_address.city, o.shipping_address.state, o.shipping_address.pincode].filter(Boolean).join(', ')}</p>}
        <h3>Items</h3>
        <table><tbody>{o.items.map((i) => <tr key={i.id}><td>{i.name}<br /><small className="muted">{i.sku}</small></td><td>× {i.quantity}</td><td className="r">{money(i.price_minor * i.quantity)}</td></tr>)}</tbody></table>
        <p className="r">Shipping ({o.shipping_method}) {money(o.shipping_cost_minor)}{o.discount_minor > 0 && <><br />Discount {o.coupon_code} − {money(o.discount_minor)}</>}<br /><strong>Total {money(o.total_minor)}</strong></p>
        {NEXT[o.status] && (<><h3>Update status</h3><div className="row">{NEXT[o.status].map((s) => (
          <button key={s} className={s === 'cancelled' ? 'danger' : 'primary'} onClick={() => { if (s !== 'cancelled' || confirm('Cancel this order? Stock will be returned.')) void act(() => api(`/api/v1/orders/${o.id}/status`, { method: 'PATCH', body: JSON.stringify({ status: s }) }), `Marked ${s}.`) }}>{s}</button>))}</div></>)}
        <h3>Tracking</h3>
        {o.tracking_number && <p>Current: <strong>{o.tracking_number}</strong> {o.tracking_carrier && `(${o.tracking_carrier})`}</p>}
        <form className="row" onSubmit={(e) => { e.preventDefault(); void act(() => api(`/api/v1/orders/${o.id}/tracking`, { method: 'PUT', body: JSON.stringify({ tracking_number: num, tracking_carrier: carrier }) }), 'Tracking saved.') }}>
          <input placeholder="Tracking number" value={num} onChange={(e) => setNum(e.target.value)} required maxLength={64} />
          <input placeholder="Carrier" value={carrier} onChange={(e) => setCarrier(e.target.value)} maxLength={64} />
          <button>Save</button>
        </form>
        {msg && <Notice kind={msg.k}>{msg.t}</Notice>}
      </>)}
    </aside>
  )
}

function Orders() {
  const [q, setQ] = useState(''); const [status, setStatus] = useState(''); const [offset, setOffset] = useState(0); const [open, setOpen] = useState('')
  const limit = 25
  const { data, error, loading, reload } = useLoad(() => api<{ data: OrderRow[]; meta: { total: number } }>(`/api/v1/admin/orders?limit=${limit}&offset=${offset}&status=${status}&q=${encodeURIComponent(q)}`), [q, status, offset])
  return (<>
    <div className="toolbar">
      <input placeholder="Search id, email, name, tracking" value={q} onChange={(e) => { setQ(e.target.value); setOffset(0) }} />
      <select value={status} onChange={(e) => { setStatus(e.target.value); setOffset(0) }}><option value="">All statuses</option>{STATUSES.map((s) => <option key={s}>{s}</option>)}</select>
    </div>
    {error && <Notice kind="err">{error}</Notice>}
    <table><thead><tr><th>Order</th><th>Customer</th><th>Placed</th><th>Items</th><th>Status</th><th className="r">Total</th></tr></thead>
      <tbody>{data?.data.map((o) => <tr key={o.id} className="click" onClick={() => setOpen(o.id)}><td><code>{o.id.slice(0, 8)}</code></td><td>{o.customer_name || '—'}<br /><small className="muted">{o.customer_email}</small></td><td>{fmtDate(o.created_at)}</td><td>{o.item_count}</td><td><span className={`pill ${o.status}`}>{o.status}</span></td><td className="r">{money(o.total_minor)}</td></tr>)}
        {!loading && data?.data.length === 0 && <tr><td colSpan={6} className="muted">No orders match.</td></tr>}</tbody></table>
    {data && <Pager total={data.meta.total} offset={offset} limit={limit} onChange={setOffset} />}
    {open && <OrderDetail id={open} onClose={() => setOpen('')} onChanged={reload} />}
  </>)
}

/* ------------------------------------------------------------- products */
type ProdRow = { id: string; name: string; slug: string; status: string; category: string; image_url: string; variant_count: number; min_price_minor: number; stock: number }
type Variant = { id: string; sku: string; name: string; price_minor: number; compare_at_minor?: number; available: number }
type ProdFull = { id: string; name: string; description: string; category: string; image_url?: string; status: string; variants: Variant[] }

function ProductEditor({ id, onClose, onChanged }: { id: string; onClose: () => void; onChanged: () => void }) {
  const { data: p, error, reload } = useLoad(() => api<ProdFull>(`/api/v1/products/${id}/admin`), [id])
  const [f, setF] = useState<{ name: string; description: string; category: string; image: string } | null>(null)
  const [v, setV] = useState({ sku: '', name: '', price: '', compare: '', stock: '' })
  const [msg, setMsg] = useState<{ k: 'ok' | 'err'; t: string } | null>(null)
  useEffect(() => { if (p) setF({ name: p.name, description: p.description, category: p.category, image: p.image_url ?? '' }) }, [p])
  const act = async (fn: () => Promise<unknown>, ok: string) => { setMsg(null); try { await fn(); setMsg({ k: 'ok', t: ok }); reload(); onChanged() } catch (e) { setMsg({ k: 'err', t: e instanceof Error ? e.message : 'Failed' }) } }
  return (
    <aside className="drawer" aria-label="Product editor">
      <button className="close" onClick={onClose} aria-label="Close">×</button>
      {error && <Notice kind="err">{error}</Notice>}
      {p && f && (<>
        <h2>{p.name} <span className={`pill ${p.status}`}>{p.status}</span></h2>
        <form onSubmit={(e) => { e.preventDefault(); void act(() => api(`/api/v1/products/${p.id}`, { method: 'PATCH', body: JSON.stringify({ name: f.name, description: f.description, category: f.category, image_url: f.image }) }), 'Product saved.') }}>
          <label>Name<input value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} required maxLength={200} /></label>
          <label>Category<input value={f.category} onChange={(e) => setF({ ...f, category: e.target.value })} /></label>
          <label>Image URL<input value={f.image} onChange={(e) => setF({ ...f, image: e.target.value })} placeholder="https://…" /></label>
          <label>Description<textarea rows={4} value={f.description} onChange={(e) => setF({ ...f, description: e.target.value })} /></label>
          <button className="primary">Save details</button>
        </form>
        <div className="row">
          {p.status !== 'active' && <button className="primary" onClick={() => void act(() => api(`/api/v1/products/${p.id}/status`, { method: 'PATCH', body: JSON.stringify({ status: 'active' }) }), 'Published.')}>Publish</button>}
          {p.status === 'active' && <button onClick={() => void act(() => api(`/api/v1/products/${p.id}/status`, { method: 'PATCH', body: JSON.stringify({ status: 'draft' }) }), 'Moved to draft.')}>Unpublish</button>}
          {p.status !== 'archived' && <button className="danger" onClick={() => void act(() => api(`/api/v1/products/${p.id}/status`, { method: 'PATCH', body: JSON.stringify({ status: 'archived' }) }), 'Archived.')}>Archive</button>}
        </div>
        <h3>Variants</h3>
        <table><thead><tr><th>SKU</th><th>Name</th><th className="r">Price</th><th className="r">Stock</th></tr></thead><tbody>
          {p.variants.map((x) => <tr key={x.id}><td><code>{x.sku}</code></td><td>{x.name || '—'}</td><td className="r">{money(x.price_minor)}{x.compare_at_minor ? <small className="muted"> <s>{money(x.compare_at_minor)}</s></small> : null}</td><td className="r">{x.available}</td></tr>)}
          {p.variants.length === 0 && <tr><td colSpan={4} className="muted">No variants yet — add one to publish.</td></tr>}</tbody></table>
        <form className="grid2" onSubmit={(e) => { e.preventDefault(); void act(async () => {
          const made = await api<{ id: string }>(`/api/v1/products/${p.id}/variants`, { method: 'POST', body: JSON.stringify({ sku: v.sku, name: v.name, price_minor: toMinor(v.price), compare_at_minor: v.compare ? toMinor(v.compare) : undefined }) })
          if (parseInt(v.stock || '0', 10) > 0) await api('/api/v1/inventory/adjust', { method: 'POST', body: JSON.stringify({ variant_id: made.id, delta: parseInt(v.stock, 10), reason: 'initial stock' }) })
          setV({ sku: '', name: '', price: '', compare: '', stock: '' })
        }, 'Variant added.') }}>
          <input placeholder="SKU" value={v.sku} onChange={(e) => setV({ ...v, sku: e.target.value })} required />
          <input placeholder="Option name (e.g. Blue / M)" value={v.name} onChange={(e) => setV({ ...v, name: e.target.value })} />
          <input placeholder="Price ₹" type="number" min="1" step="0.01" value={v.price} onChange={(e) => setV({ ...v, price: e.target.value })} required />
          <input placeholder="Compare-at ₹ (optional)" type="number" min="1" step="0.01" value={v.compare} onChange={(e) => setV({ ...v, compare: e.target.value })} />
          <input placeholder="Initial stock" type="number" min="0" value={v.stock} onChange={(e) => setV({ ...v, stock: e.target.value })} />
          <button>Add variant</button>
        </form>
        {msg && <Notice kind={msg.k}>{msg.t}</Notice>}
      </>)}
    </aside>
  )
}

function Products() {
  const [q, setQ] = useState(''); const [status, setStatus] = useState(''); const [offset, setOffset] = useState(0); const [open, setOpen] = useState('')
  const [creating, setCreating] = useState(false); const [nf, setNf] = useState({ name: '', category: '', image: '', description: '' }); const [err, setErr] = useState('')
  const limit = 25
  const { data, error, loading, reload } = useLoad(() => api<{ data: ProdRow[]; meta: { total: number } }>(`/api/v1/admin/products?limit=${limit}&offset=${offset}&status=${status}&q=${encodeURIComponent(q)}`), [q, status, offset])
  const create = async (e: React.FormEvent) => {
    e.preventDefault(); setErr('')
    try { const p = await api<{ id: string }>('/api/v1/products', { method: 'POST', body: JSON.stringify({ name: nf.name, category: nf.category, image_url: nf.image, description: nf.description }) }); setCreating(false); setNf({ name: '', category: '', image: '', description: '' }); reload(); setOpen(p.id) }
    catch (e2) { setErr(e2 instanceof Error ? e2.message : 'Failed') }
  }
  return (<>
    <div className="toolbar">
      <input placeholder="Search name or slug" value={q} onChange={(e) => { setQ(e.target.value); setOffset(0) }} />
      <select value={status} onChange={(e) => { setStatus(e.target.value); setOffset(0) }}><option value="">All</option><option>active</option><option>draft</option><option>archived</option></select>
      <button className="primary" onClick={() => setCreating(!creating)}>{creating ? 'Cancel' : '+ New product'}</button>
    </div>
    {creating && (
      <form onSubmit={create} className="card grid2">
        <input placeholder="Product name" value={nf.name} onChange={(e) => setNf({ ...nf, name: e.target.value })} required />
        <input placeholder="Category (e.g. electronics)" value={nf.category} onChange={(e) => setNf({ ...nf, category: e.target.value })} />
        <input placeholder="Image URL" value={nf.image} onChange={(e) => setNf({ ...nf, image: e.target.value })} />
        <input placeholder="Short description" value={nf.description} onChange={(e) => setNf({ ...nf, description: e.target.value })} />
        <button className="primary">Create draft</button>{err && <Notice kind="err">{err}</Notice>}
      </form>)}
    {error && <Notice kind="err">{error}</Notice>}
    <table><thead><tr><th></th><th>Product</th><th>Category</th><th>Status</th><th>Variants</th><th className="r">From</th><th className="r">Stock</th></tr></thead>
      <tbody>{data?.data.map((p) => <tr key={p.id} className="click" onClick={() => setOpen(p.id)}><td>{p.image_url ? <img className="thumb" src={p.image_url} alt="" /> : <span className="thumb" />}</td><td>{p.name}<br /><small className="muted">{p.slug}</small></td><td>{p.category}</td><td><span className={`pill ${p.status}`}>{p.status}</span></td><td>{p.variant_count}</td><td className="r">{p.variant_count ? money(p.min_price_minor) : '—'}</td><td className="r">{p.stock}</td></tr>)}
        {!loading && data?.data.length === 0 && <tr><td colSpan={7} className="muted">No products yet.</td></tr>}</tbody></table>
    {data && <Pager total={data.meta.total} offset={offset} limit={limit} onChange={setOffset} />}
    {open && <ProductEditor id={open} onClose={() => setOpen('')} onChanged={reload} />}
  </>)
}

/* ------------------------------------------------------------ inventory */
type InvRow = { variant_id: string; sku: string; product_name: string; quantity: number; reserved: number; available: number }
function Inventory() {
  const [low, setLow] = useState(false)
  const { data, error, reload } = useLoad(() => api<{ data: InvRow[]; meta: { low_stock_threshold: number } }>(`/api/v1/admin/inventory?limit=200${low ? '&low=1' : ''}`), [low])
  const [delta, setDelta] = useState<Record<string, string>>({}); const [msg, setMsg] = useState<{ k: 'ok' | 'err'; t: string } | null>(null)
  const adjust = async (id: string) => {
    const d = parseInt(delta[id] || '0', 10)
    if (!d) return
    try { await api('/api/v1/inventory/adjust', { method: 'POST', body: JSON.stringify({ variant_id: id, delta: d, reason: d > 0 ? 'restock (admin)' : 'correction (admin)' }) }); setDelta({ ...delta, [id]: '' }); setMsg({ k: 'ok', t: 'Stock updated.' }); reload() }
    catch (e) { setMsg({ k: 'err', t: e instanceof Error ? e.message : 'Failed' }) }
  }
  return (<>
    <div className="toolbar"><label className="check"><input type="checkbox" checked={low} onChange={(e) => setLow(e.target.checked)} /> Low / out of stock only (≤ {data?.meta.low_stock_threshold ?? 5})</label></div>
    {error && <Notice kind="err">{error}</Notice>}{msg && <Notice kind={msg.k}>{msg.t}</Notice>}
    <table><thead><tr><th>Product</th><th>SKU</th><th className="r">On hand</th><th className="r">Reserved</th><th className="r">Available</th><th>Adjust (+/−)</th></tr></thead>
      <tbody>{data?.data.map((r) => <tr key={r.variant_id}><td>{r.product_name}</td><td><code>{r.sku}</code></td><td className="r">{r.quantity}</td><td className="r">{r.reserved}</td><td className={`r ${r.available === 0 ? 'bad' : r.available <= 5 ? 'warn' : ''}`}>{r.available}</td>
        <td><div className="row"><input type="number" style={{ width: 90 }} value={delta[r.variant_id] ?? ''} onChange={(e) => setDelta({ ...delta, [r.variant_id]: e.target.value })} /><button onClick={() => void adjust(r.variant_id)}>Apply</button></div></td></tr>)}</tbody></table>
  </>)
}

/* --------------------------------------------------------------- coupons */
type Coupon = { id: string; code: string; discount_type: string; value: number; active: boolean; min_order_minor: number; max_uses?: number; used_count: number; expires_at?: string }
function Coupons() {
  const { data, error, reload } = useLoad(() => api<{ data: Coupon[] }>('/api/v1/coupons'), [])
  const [f, setF] = useState({ code: '', type: 'percentage', value: '', min: '', max: '', exp: '' }); const [msg, setMsg] = useState<{ k: 'ok' | 'err'; t: string } | null>(null)
  const create = async (e: React.FormEvent) => {
    e.preventDefault(); setMsg(null)
    try {
      await api('/api/v1/coupons', { method: 'POST', body: JSON.stringify({ code: f.code, discount_type: f.type, value: f.type === 'percentage' ? parseInt(f.value, 10) : toMinor(f.value), min_order_minor: f.min ? toMinor(f.min) : 0, max_uses: f.max ? parseInt(f.max, 10) : undefined, expires_at: f.exp ? new Date(f.exp).toISOString() : undefined }) })
      setF({ code: '', type: 'percentage', value: '', min: '', max: '', exp: '' }); setMsg({ k: 'ok', t: 'Coupon created.' }); reload()
    } catch (e2) { setMsg({ k: 'err', t: e2 instanceof Error ? e2.message : 'Failed' }) }
  }
  const toggle = async (c: Coupon) => { try { await api(`/api/v1/coupons/${c.id}`, { method: 'PATCH', body: JSON.stringify({ active: !c.active }) }); reload() } catch (e) { setMsg({ k: 'err', t: e instanceof Error ? e.message : 'Failed' }) } }
  return (<>
    <form onSubmit={create} className="card grid2">
      <input placeholder="CODE" value={f.code} onChange={(e) => setF({ ...f, code: e.target.value.toUpperCase() })} required />
      <select value={f.type} onChange={(e) => setF({ ...f, type: e.target.value })}><option value="percentage">Percentage off</option><option value="fixed">Fixed ₹ off</option></select>
      <input placeholder={f.type === 'percentage' ? 'Percent (1-100)' : 'Amount ₹'} type="number" min="1" step={f.type === 'percentage' ? 1 : 0.01} value={f.value} onChange={(e) => setF({ ...f, value: e.target.value })} required />
      <input placeholder="Min order ₹ (optional)" type="number" min="0" value={f.min} onChange={(e) => setF({ ...f, min: e.target.value })} />
      <input placeholder="Max uses (optional)" type="number" min="1" value={f.max} onChange={(e) => setF({ ...f, max: e.target.value })} />
      <input type="datetime-local" aria-label="Expires" value={f.exp} onChange={(e) => setF({ ...f, exp: e.target.value })} />
      <button className="primary">Create coupon</button>
    </form>
    {msg && <Notice kind={msg.k}>{msg.t}</Notice>}{error && <Notice kind="err">{error}</Notice>}
    <table><thead><tr><th>Code</th><th>Discount</th><th>Min order</th><th>Used</th><th>Expires</th><th>Status</th><th></th></tr></thead>
      <tbody>{data?.data.map((c) => <tr key={c.id}><td><code>{c.code}</code></td><td>{c.discount_type === 'percentage' ? `${c.value}%` : money(c.value)}</td><td>{c.min_order_minor ? money(c.min_order_minor) : '—'}</td><td>{c.used_count}{c.max_uses ? ` / ${c.max_uses}` : ''}</td><td>{c.expires_at ? fmtDate(c.expires_at) : 'Never'}</td><td><span className={`pill ${c.active ? 'active' : 'archived'}`}>{c.active ? 'active' : 'off'}</span></td><td><button onClick={() => void toggle(c)}>{c.active ? 'Disable' : 'Enable'}</button></td></tr>)}</tbody></table>
  </>)
}

/* --------------------------------------------------------------- returns */
type Ret = { id: string; order_id: string; status: string; reason: string; refund_amount_minor: number; created_at: string; customer_email: string; admin_note: string; return_items: { name: string; quantity: number }[] }
const RNEXT: Record<string, string[]> = { requested: ['approved', 'rejected'], approved: ['received', 'rejected'], received: ['refunded'] }
function Returns() {
  const [status, setStatus] = useState('requested')
  const { data, error, reload } = useLoad(() => api<{ data: Ret[] }>(`/api/v1/returns?status=${status}`), [status])
  const [notes, setNotes] = useState<Record<string, string>>({}); const [msg, setMsg] = useState<{ k: 'ok' | 'err'; t: string } | null>(null)
  const move = async (r: Ret, to: string) => {
    try { await api(`/api/v1/returns/${r.id}/status`, { method: 'PATCH', body: JSON.stringify({ status: to, note: notes[r.id] ?? '' }) }); setMsg({ k: 'ok', t: `Return ${to}.` }); reload() } catch (e) { setMsg({ k: 'err', t: e instanceof Error ? e.message : 'Failed' }) }
  }
  return (<>
    <div className="toolbar"><select value={status} onChange={(e) => setStatus(e.target.value)}>{['requested', 'approved', 'received', 'refunded', 'rejected', ''].map((s) => <option key={s} value={s}>{s || 'All'}</option>)}</select></div>
    {error && <Notice kind="err">{error}</Notice>}{msg && <Notice kind={msg.k}>{msg.t}</Notice>}
    {data?.data.length === 0 && <p className="muted">No returns in this state.</p>}
    {data?.data.map((r) => (
      <div key={r.id} className="card">
        <div className="row spread"><strong>Order #{r.order_id.slice(0, 8)}</strong><span className={`pill ${r.status}`}>{r.status}</span></div>
        <p className="muted">{r.customer_email} · {fmtDate(r.created_at)} · Refund {money(r.refund_amount_minor)}</p>
        <p>{r.return_items.map((i) => `${i.name} × ${i.quantity}`).join(', ')}</p><p>“{r.reason}”</p>
        {RNEXT[r.status] && <div className="row"><input placeholder="Note to customer (optional)" value={notes[r.id] ?? ''} onChange={(e) => setNotes({ ...notes, [r.id]: e.target.value })} />{RNEXT[r.status].map((s) => <button key={s} className={s === 'rejected' ? 'danger' : 'primary'} onClick={() => void move(r, s)}>{s}</button>)}</div>}
        {r.status === 'received' && <p className="muted">Issue the refund in your payment provider, then mark it refunded.</p>}
      </div>))}
  </>)
}

/* ------------------------------------------------------------------ app */
export default function App() {
  const [me, setMe] = useState<Me | null | undefined>(undefined)
  const [tab, setTab] = useState<Tab>('dashboard')
  useEffect(() => { api<Me>('/api/v1/auth/me').then((u) => setMe(u.role === 'admin' ? u : null)).catch((e: unknown) => { if (!(e instanceof ApiError) || e.status !== 0) setMe(null); else setMe(null) }) }, [])
  const logout = useCallback(async () => { await api('/api/v1/auth/logout', { method: 'POST' }).catch(() => undefined); setMe(null) }, [])
  if (me === undefined) return <p className="center">Loading…</p>
  if (!me) return <Login onDone={setMe} />
  return (
    <div className="shell">
      <header><strong>OmniRetail Admin</strong><nav>{TABS.map((t) => <button key={t.id} className={tab === t.id ? 'on' : ''} onClick={() => setTab(t.id)}>{t.label}</button>)}</nav><span className="muted">{me.email}</span><button onClick={() => void logout()}>Sign out</button></header>
      <main>
        <h1>{TABS.find((t) => t.id === tab)?.label}</h1>
        {tab === 'dashboard' && <Dashboard go={setTab} />}{tab === 'orders' && <Orders />}{tab === 'products' && <Products />}
        {tab === 'inventory' && <Inventory />}{tab === 'coupons' && <Coupons />}{tab === 'returns' && <Returns />}
      </main>
    </div>
  )
}

import http.client, json, threading, uuid, subprocess, time
O="http://127.0.0.1:3000"; P=18081
def call(ck,m,p,b=None,h=None):
    c=http.client.HTTPConnection("127.0.0.1",P,timeout=30); hh={"Content-Type":"application/json","Origin":O}; hh.update(h or {})
    if ck[0]: hh["Cookie"]=ck[0]
    c.request(m,p,json.dumps(b) if b is not None else None,hh); r=c.getresponse(); d=r.read().decode(); sc=r.getheader("Set-Cookie")
    if sc and "omniretail_session=" in sc: ck[0]=sc.split(";")[0]
    return r.status,(json.loads(d) if d.startswith(("{","[")) else d)
sql=lambda q: subprocess.run(["psql","-h","/tmp","-p","55432","-U","app","-d","commerce_e2e","-tAc",q],capture_output=True,text=True).stdout.strip()
vid=sql("select v.id from product_variants v join products p on p.id=v.product_id where p.name='Smart Fitness Band'")
sql(f"update inventory set quantity=3,reserved=0 where variant_id='{vid}'")
addr=json.dumps({"address":"1 St","city":"Pune","state":"MH","pincode":"411001"})
users=[]
for i in range(8):
    ck=[None]; e=f"race{uuid.uuid4().hex[:6]}@x.test"; call(ck,"POST","/api/v1/auth/register",{"email":e,"password":"Racer!pass99"})
    s,_=call(ck,"POST","/api/v1/cart/items",{"variant_id":vid,"quantity":1}); users.append((ck,e,s))
print("carts filled:",[u[2] for u in users])
out=[]
def go(u):
    ck,e,_=u
    s,r=call(ck,"POST","/api/v1/checkout",{"payment_provider":"manual","payment_method":"cod","currency":"INR","shipping_method":"standard","shipping_address":addr,"customer_name":"R","customer_email":e,"customer_phone":"9999999999"},{"Idempotency-Key":uuid.uuid4().hex+"race"})
    out.append(s)
ts=[threading.Thread(target=go,args=(u,)) for u in users]; [t.start() for t in ts]; [t.join() for t in ts]
print("checkout statuses:",sorted(out))
q=sql(f"select quantity from inventory where variant_id='{vid}'")
ok=out.count(200)==3 and q=="0" and out.count(409)==5
print(("PASS" if ok else "FAIL"),"no oversell: 3 succeeded, 5 rejected, stock",q)
# same idempotency key, concurrent
sql(f"update inventory set quantity=10 where variant_id='{vid}'")
ck=[None]; e=f"idem{uuid.uuid4().hex[:6]}@x.test"; call(ck,"POST","/api/v1/auth/register",{"email":e,"password":"Racer!pass99"}); call(ck,"POST","/api/v1/cart/items",{"variant_id":vid,"quantity":2})
key=uuid.uuid4().hex+"same"; res=[]
def dup():
    s,r=call(ck,"POST","/api/v1/checkout",{"payment_provider":"manual","payment_method":"cod","currency":"INR","shipping_method":"standard","shipping_address":addr,"customer_name":"R","customer_email":e,"customer_phone":"9999999999"},{"Idempotency-Key":key}); res.append((s,(r or {}).get("order_id") if isinstance(r,dict) else None))
ts=[threading.Thread(target=dup) for _ in range(5)]; [t.start() for t in ts]; [t.join() for t in ts]
orders=sql(f"select count(*) from orders o join users u on u.id=o.user_id where u.email='{e}'")
print(("PASS" if orders=="1" and len({r[1] for r in res if r[1]})==1 else "FAIL"),"5 concurrent identical requests created exactly",orders,"order; statuses",sorted(r[0] for r in res), "stock", sql(f"select quantity from inventory where variant_id='{vid}'"))
# latency sanity on catalogue
lat=[]
def hit():
    for _ in range(25):
        t=time.time(); call([None],"GET","/api/v1/products?limit=24"); lat.append(time.time()-t)
ts=[threading.Thread(target=hit) for _ in range(8)]; [t.start() for t in ts]; [t.join() for t in ts]
lat.sort(); print(f"catalogue 200 reqs x8 threads: p50={lat[len(lat)//2]*1000:.1f}ms p95={lat[int(len(lat)*.95)]*1000:.1f}ms max={lat[-1]*1000:.1f}ms")

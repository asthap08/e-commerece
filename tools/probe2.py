import http.client, json, re, subprocess, time, uuid, os
O="http://localhost:3000"; P=18083
def call(ck,m,p,b=None,h=None):
    c=http.client.HTTPConnection("127.0.0.1",P); hh={"Content-Type":"application/json","Origin":O}; hh.update(h or {})
    if ck[0]: hh["Cookie"]=ck[0]
    c.request(m,p,json.dumps(b) if b is not None else None,hh); r=c.getresponse(); d=r.read().decode(); sc=r.getheader("Set-Cookie")
    if sc and "omniretail_session=" in sc: v=sc.split(";")[0]; ck[0]=None if v.endswith("=") else v
    return r.status,(json.loads(d) if d.startswith(("{","[")) else d)
def sql(q): return subprocess.run(["psql","-h","/tmp","-p","55432","-U","app","-d","commerce_p2","-tAc",q],capture_output=True,text=True).stdout.strip()
ok=[]
def check(l,c,d=""): ok.append(c); print(("PASS " if c else "FAIL ")+l+(f" -> {d}" if d else ""))
u=ck=[None]; e=f"reset{uuid.uuid4().hex[:5]}@x.test"; pw="Oldpass1!xx"
call(u,"POST","/api/v1/auth/register",{"email":e,"password":pw})
s,_=call([None],"POST","/api/v1/auth/forgot",{"email":"nobody@x.test"}); check("forgot for unknown email still 200 (no enumeration)",s==200)
s,_=call([None],"POST","/api/v1/auth/forgot",{"email":e.upper()}); check("forgot for real email 200",s==200); time.sleep(1.5)
mails=open("/tmp/mails.txt").read() if os.path.exists("/tmp/mails.txt") else ""
m=re.search(r"reset-password\?token=([A-Za-z0-9_-]+)",mails); check("reset e-mail delivered over SMTP with link to PUBLIC_URL","https://shop.example.com/reset-password" in mails and bool(m))
tok=m.group(1) if m else ""
check("only one e-mail sent (none for unknown address)", mails.count("Subject: Reset your password")==1)
s,r=call([None],"POST","/api/v1/auth/reset",{"token":tok,"new_password":"weak"}); check("weak new password rejected",s==400,str(r)[:60])
s,r=call([None],"POST","/api/v1/auth/reset",{"token":tok,"new_password":"Brandnew1!xx"}); check("reset with valid token works",s==200,str(r)[:60])
s,r=call([None],"POST","/api/v1/auth/reset",{"token":tok,"new_password":"Another1!xx"}); check("token is single-use",s==400)
s,_=call([None],"POST","/api/v1/auth/login",{"email":e,"password":pw}); check("old password no longer works",s==401)
s,_=call([None],"POST","/api/v1/auth/login",{"email":e,"password":"Brandnew1!xx"}); check("new password works",s==200)
s,_=call(u,"GET","/api/v1/auth/me"); check("pre-reset session was revoked",s==401)
s,_=call([None],"POST","/api/v1/auth/reset",{"token":"x"*40,"new_password":"Brandnew1!xx"}); check("random token rejected",s==400)
# order confirmation email
adm=[None]; call(adm,"POST","/api/v1/auth/register",{"email":"a@x.test","password":"Adm1n!pass99"}); sql("UPDATE users SET role='admin' WHERE email='a@x.test'"); call(adm,"POST","/api/v1/auth/login",{"email":"a@x.test","password":"Adm1n!pass99"})
s,p=call(adm,"POST","/api/v1/products",{"name":"Mail Widget"}); s,v=call(adm,"POST",f"/api/v1/products/{p['id']}/variants",{"sku":"MW1","price_minor":10000}); call(adm,"POST","/api/v1/inventory/adjust",{"variant_id":v["id"],"delta":50,"reason":"t"}); call(adm,"PATCH",f"/api/v1/products/{p['id']}/status",{"status":"active"})
c=[None]; call(c,"POST","/api/v1/auth/login",{"email":e,"password":"Brandnew1!xx"}); call(c,"POST","/api/v1/cart/items",{"variant_id":v["id"],"quantity":1})
addr=json.dumps({"address":"1 St","city":"Pune","state":"MH","pincode":"411001"})
s,co=call(c,"POST","/api/v1/checkout",{"payment_provider":"manual","payment_method":"cod","currency":"INR","shipping_method":"standard","shipping_address":addr,"customer_name":"Asha","customer_email":e,"customer_phone":"9999999999"},{"Idempotency-Key":uuid.uuid4().hex+"zz"})
time.sleep(1.5); mails=open("/tmp/mails.txt").read()
check("order confirmation e-mail sent with tracking link",s==200 and co["order_id"] in mails and "Your order is confirmed" in mails)
# worker: aged unpaid card order is cancelled + restocked, coupon use restored
sql(f"INSERT INTO coupons(code,discount_type,value,active,used_count) VALUES('WK1','percentage',10,true,1)")
sql(f"INSERT INTO orders(id,user_id,status,total_minor,currency,created_at,coupon_code,payment_method,shipping_method) SELECT gen_random_uuid(),id,'pending',10000,'INR',now()-interval '3 hours','WK1','card','standard' FROM users WHERE email='{e}'")
oid=sql("SELECT id FROM orders WHERE coupon_code='WK1'"); sql(f"INSERT INTO order_items(order_id,variant_id,sku,name,price_minor,quantity) VALUES('{oid}','{v['id']}','MW1','Mail Widget',10000,3)"); sql(f"INSERT INTO payments(order_id,provider,status,amount_minor,method) VALUES('{oid}','stripe','pending',10000,'card')")
before=int(sql(f"select quantity from inventory where variant_id='{v['id']}'"))
env=dict(os.environ,APP_ENV="development",CORS_ORIGINS="http://localhost:3000",DATABASE_URL="postgres://app@/commerce_p2?host=/tmp&port=55432")
try: subprocess.run(["/home/claude/tools/worker_new"],env=env,timeout=6,capture_output=True)
except subprocess.TimeoutExpired: pass
check("worker cancels stale unpaid card order",sql(f"select status from orders where id='{oid}'")=="cancelled")
check("worker restocks its items",int(sql(f"select quantity from inventory where variant_id='{v['id']}'"))==before+3)
check("worker restores coupon use",sql("select used_count from coupons where code='WK1'")=="0")
check("COD pending order untouched by worker",sql(f"select status from orders where id='{co['order_id']}'")=="pending")
print(sum(ok),"/",len(ok))

import time, sys, json, re, uuid
sys.path.insert(0,'/home/claude/tools')
from selenium import webdriver
from selenium.webdriver.firefox.options import Options
from selenium.webdriver.firefox.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
B="http://127.0.0.1:3000"
o=Options(); o.add_argument("-headless"); o.binary_location="/home/claude/tools/mamba/envs/dev/bin/firefox"
d=webdriver.Firefox(options=o, service=Service("/home/claude/tools/mamba/envs/dev/bin/geckodriver")); d.set_window_size(1280,900)
res=[]
def check(l,ok,det=""):
    res.append(ok); print(("PASS " if ok else "FAIL ")+l+(f" -> {det}" if det else ""))
def shot(n): d.save_screenshot(f"/home/claude/shots/{n}.png")
def text(): return d.find_element(By.TAG_NAME,"body").text
def go(p): d.get(B+p); time.sleep(1.5)
def click_text(t, tag="button"):
    for e in d.find_elements(By.XPATH,f"//{tag}[contains(normalize-space(.),'{t}')]"):
        if e.is_displayed() and e.is_enabled(): e.click(); time.sleep(0.8); return True
    return False
def visible_inputs(): return [i for i in d.find_elements(By.CSS_SELECTOR,"input:not([type=radio]):not([type=checkbox]):not([type=hidden]),select") if i.is_displayed()]
email=f"buyer{uuid.uuid4().hex[:6]}@shop.test"; PW="Buyer!pass99"
try:
    go("/signup")
    ins=visible_inputs()
    ins[0].send_keys("Asha Verma"); ins[1].send_keys(email); ins[2].send_keys(PW); ins[3].send_keys(PW)
    click_text("Create"); time.sleep(2.5)
    check("signup redirects to account", "/account" in d.current_url, d.current_url)
    check("navbar greets signed-in user and shows Sign out", "Sign out" in text() and "Hello, Sign in" not in text(), text()[:0])
    go("/products"); d.find_element(By.XPATH,"//h3//a[contains(.,'Wireless Earbuds')]").click(); time.sleep(1.5)
    click_text("Add to cart"); time.sleep(1.5)
    check("cart drawer opens after add", "Your cart" in text() and "Wireless Earbuds Pro" in text()); shot("04_cart")
    # coupon
    ci=[i for i in d.find_elements(By.CSS_SELECTOR,"input[aria-label='Coupon code']")][0]; ci.send_keys("welcome10"); click_text("Apply"); time.sleep(1.2)
    check("coupon applies (10% of 1999 = 199.90)", "WELCOME10 applied" in text() and "199.9" in text(), "")
    shot("05_coupon")
    bad=d.find_elements(By.CSS_SELECTOR,"input[aria-label='Coupon code']")
    # qty + stock
    click_text("Proceed to checkout","a"); time.sleep(2)
    check("checkout page loads with prefilled email", email in " ".join(i.get_attribute("value") or "" for i in visible_inputs()), d.current_url)
    ins=visible_inputs(); print("checkout step1 inputs:",[ (i.get_attribute('placeholder') or i.get_attribute('name')) for i in ins])
    shot("06_checkout1")
    def fill_ph(ph,val):
        for i in visible_inputs():
            if (i.get_attribute('placeholder') or '').startswith(ph):
                i.clear(); i.send_keys(val); return True
        return False
    fill_ph("+91","9876543210")
    click_text("Continue"); time.sleep(1)
    print("step2 placeholders:",[ (i.get_attribute('placeholder') or i.get_attribute('name')) for i in visible_inputs()]); shot("06b_checkout2")
    check("step 2 (shipping) reached", "Shipping" in text() and "Back" in text())
    click_text("Back"); time.sleep(0.6)
    check("Back returns to details", "FIRST NAME" in text().upper())
    click_text("Continue"); time.sleep(0.8)
    for ph,val in [("123","12 MG Road"),("Mumbai","Pune"),("Maharashtra","Maharashtra"),("400001","411001")]: fill_ph(ph,val)
    shot("06c_checkout2b")
    click_text("Continue"); time.sleep(0.8)
    shot("07_checkout3"); print("step3:",text()[text().find("Payment"):][:200].replace("\n"," | "))
    click_text("Place"); time.sleep(3)
    shot("08_confirmed")
    t=text()
    m=re.search(r"#([0-9a-f-]{36})",t)
    check("order placed and id shown", bool(m), t[:0])
    check("confirmation total is real (₹1,799.10 + shipping), not ₹0", "1,799" in t or "1799" in t, re.findall(r"Order Total[^\n]*\n?[^\n]*",t)[:1])
    oid=m.group(1) if m else ""
    go("/account/orders"); check("order appears in My Orders", oid[:8] in text(), ""); shot("09_orders")
    go(f"/track-order?id={oid}"); 
    ins=visible_inputs(); 
    for i in ins:
        if i.get_attribute("type")=="email": i.send_keys(email)
    click_text("Track order"); time.sleep(1.5); shot("10_track")
    check("guest-style tracking shows order status", "Order placed" in text() and "Total" in text())
    go("/account/profile"); time.sleep(1)
    check("profile page loads with email", email in " ".join(i.get_attribute("value") or "" for i in d.find_elements(By.CSS_SELECTOR,"input")))
    go("/account/addresses"); shot("11_addresses"); check("addresses page loads (no crash)", "Application error" not in text())
    go("/returns"); check("returns page loads", "Start a return" in text() and "no delivered orders" in text().lower() or "Start a return" in text())
    # cancel own pending order
    go("/account/orders"); 
    ok=click_text("Cancel Order"); 
    if ok:
        try: d.switch_to.alert.accept()
        except Exception: pass
        time.sleep(2)
    check("customer can cancel own pending order from UI", ok and "cancelled" in text().lower(), "")
    # sign out
    click_text("Sign out"); time.sleep(2); go("/account/orders"); time.sleep(1.5)
    check("after sign out, protected page redirects to login", "/login" in d.current_url, d.current_url)
finally:
    print("RESULT",sum(res),"/",len(res)); d.quit()

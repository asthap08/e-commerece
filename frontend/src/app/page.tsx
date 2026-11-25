"use client";
import { fmtNum } from "@/lib/api";
import { useState } from "react";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useStore } from "@/lib/store";
import { useCatalogProducts } from "@/lib/catalog";
import { DEMO_MODE } from "@/lib/demo";

export default function HomePage() {
  const router = useRouter();
  const { addToCart, setCartOpen } = useStore();
  const { products } = useCatalogProducts();
  const [purchasePending, setPurchasePending] = useState(false);
  const [purchaseError, setPurchaseError] = useState("");
  const featuredProduct = products[0];
  const canPurchaseFeatured = featuredProduct?.inStock &&
    (Boolean(featuredProduct.variantId) || DEMO_MODE);

  async function addFeaturedToCart(andCheckout: boolean) {
    if (!featuredProduct || !canPurchaseFeatured || purchasePending) return;
    setPurchasePending(true);
    setPurchaseError("");
    try {
      await addToCart({
        id: featuredProduct.id,
        variantId: featuredProduct.variantId,
        name: featuredProduct.name,
        price: featuredProduct.price,
        originalPrice: featuredProduct.originalPrice,
        image: featuredProduct.image,
        category: featuredProduct.category,
      });
      if (andCheckout) router.push("/checkout");
      else setCartOpen(true);
    } catch (error) {
      setPurchaseError(error instanceof Error ? error.message : "The product could not be added to your cart.");
    } finally {
      setPurchasePending(false);
    }
  }

  return (
    <>
      <Navbar />
      <main className="flex-1 w-full max-w-[1536px] mx-auto px-4 sm:px-6 lg:px-8 pb-16">
        {/* Hero Section */}
        <div className="relative w-full mb-8">
          <div className="relative h-[360px] md:h-[440px] rounded-b-xl overflow-hidden bg-gradient-to-r from-primary-container via-tertiary-container to-primary-fixed-variant flex items-center justify-between px-8 md:px-16 text-on-primary shadow-inner">
            <div className="max-w-xl z-10 py-6">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded bg-secondary-container text-primary-container font-label-md text-label-md font-bold mb-3 shadow-sm">
                🔥 SEASONAL SAVINGS EVENT
              </div>
              <h1 className="font-headline-xl text-headline-xl font-bold tracking-tight text-on-primary mb-2">
                Spring Refresh Sale — Up to 40% Off Home &amp; Tech
              </h1>
              <p className="font-body-lg text-body-lg text-surface-variant mb-6">
                Equip your spaces with smart climate devices, organic home textiles, and premium noise-cancelling gear.
              </p>
              <div className="flex items-center gap-4">
                <Link href="/products" className="px-6 py-2.5 rounded-full bg-secondary-container hover:bg-secondary text-primary-container font-label-md text-label-md font-bold transition-all shadow-md active:scale-95">
                  Explore Deals Now
                </Link>
                <span className="font-body-xs text-body-xs text-on-primary-container flex items-center gap-1">
                  ✅ Guaranteed Authenticity
                </span>
              </div>
            </div>
            <div className="hidden lg:block relative w-96 h-80 opacity-90">
              <Image
                fill
                sizes="(max-width: 1024px) 0px, 384px"
                className="object-cover rounded-xl shadow-2xl border border-outline-variant/30"
                src="https://lh3.googleusercontent.com/aida-public/AB6AXuDBq-BqA1yEIAjWKctdJz4dHa8fPmZQnfaBDPh7j-yHgESqDyPh4XiWP5lMdtC2Wzt0cUjoCWN4Te8NntGEeOb19yFsAd54577-H6gzPmbuy8aGlclYz6srMyMOsgr4NPsXf0i2C6Eid3Krr893NDZhEHXCWK3-e_DXWcGelkqnPn0NiuAAqFndIDZh_MnWymXQFsdWAtbN29fmlGArbpzTZICOTD-U6Yw2QmiRre3KTjLzoPvdUoxa"
                alt="Modern living room with smart home decor"
              />
            </div>
            <div className="absolute bottom-0 left-0 right-0 h-40 bg-gradient-to-t from-background to-transparent pointer-events-none" />
          </div>

          {/* Bento Tiles */}
          <div className="mt-8 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
            {/* Card 1: Home Refresh */}
            <div className="bg-surface-container-lowest border border-outline-variant rounded-lg p-4 shadow-sm flex flex-col justify-between hover:shadow-md transition-shadow">
              <div>
                <h2 className="font-headline-sm text-headline-sm font-bold text-on-surface mb-3">Refresh your space</h2>
                <div className="grid grid-cols-2 gap-3 mb-4">
                  {[
                    { label: "Living room", img: "https://lh3.googleusercontent.com/aida-public/AB6AXuBdsDMCZAncyFuiEn3pGD8wiO3hZdUy7KffBYjqz7TIDnFyhznIK9bUBdd-mc5u26QPGeFxo0Jfy3Ox83rgyG6qNA_ZaDnrR4L8dBJinCXYtAYmlj-rcMpE_CPlBROoVcKD72cIm_2jwMGZVdamxhRbpe1jOFZ0yA_F7f74nhiONM6KMx1RzsY__cmRmwHBGPcffOxqHf18MBW93DJq2B9JKhcGDs0fcwa2dvyZZHMCEWwwK__uUFHo" },
                    { label: "Kitchen essentials", img: "https://lh3.googleusercontent.com/aida-public/AB6AXuDK_iws-UMPYXXzC4Bbw6MYlH3-BJ88QJgk0Ht0hmdtpWhAzauxIhT7RBcNYOa-W-Kfi0pZsUhTd0Ib3F5vgcr9M_seOuxKrp4CnzNL8q0VEh5pJFZFaCoCNYyuWNMe6IoaXnpSCAbIsPJ6GhJp8TrNXwNgBESjooib6DuJdM878XYZ400xcKDPpPlPSLW53hKXNjw5yHKJ7A760HP1iTZ9OLtcvbBsVLN0QvsoJV6PlOTXaF3DINA3" },
                    { label: "Bedding", img: "https://lh3.googleusercontent.com/aida-public/AB6AXuDmSGlwxy3o12T-nn6PQq9dslX5vWBeuYt1rIgwBXxFEKTmISkTPcEA5UQ1fZmOzO2ZPC0qubmHYsGpBYmW-hW-a3i4XfWndyxqqcqmgeIzDJv3zJthwW28aSj2Z9jdAqtjQHQccG8k0mLPytyCk3ZQut9XbBiKd48UlE25No2mKjQTvrPk7o_93JnMUagxuwFWgrKaXaWXRovMAvOc29Mgi-XMp9y5wbWRo9z5d7fXIDGiH1R1ptrJ" },
                    { label: "Storage bins", img: "https://lh3.googleusercontent.com/aida-public/AB6AXuDlgbOfNjUzXm5Km6QvyZQ1fHSrIbkQjts6qWSMsbvBtn_wmjt04pLSrVwzZKXToAnT9-IyHp9-AIFm5xgPTbu8VonWNnDAqHmCI4htxUWlYlfOC1qxppU3NvAg7pOBFtM_WHG8o2bKypj8oOLXjLtmaoPG3mYaKtFgSpnNX_NJlNkRk27Oxwjo9AUgNlMDKCDbwrKNHQ0SiwUslAtFhZ7cpn-OlfVaVCTnOMTYTNo_DwnUU0UzK1qW" },
                  ].map((item) => (
                    <Link key={item.label} href="/products" className="group block">
                      <div className="aspect-square bg-surface-container-low rounded overflow-hidden mb-1 relative">
                        <Image fill sizes="(max-width: 640px) 45vw, 150px" className="object-cover group-hover:scale-105 transition-transform duration-200" src={item.img} alt={item.label} />
                      </div>
                      <span className="font-body-xs text-body-xs text-on-surface-variant group-hover:text-primary-container font-medium">{item.label}</span>
                    </Link>
                  ))}
                </div>
              </div>
              <Link href="/products" className="font-label-md text-label-md font-semibold text-primary-container hover:underline inline-flex items-center gap-1">
                Explore all home →
              </Link>
            </div>

            {/* Card 2: Tech Picks */}
            <div className="bg-surface-container-lowest border border-outline-variant rounded-lg p-4 shadow-sm flex flex-col justify-between hover:shadow-md transition-shadow">
              <div>
                <h2 className="font-headline-sm text-headline-sm font-bold text-on-surface mb-3">Top picks in Electronics</h2>
                <div className="grid grid-cols-2 gap-3 mb-4">
                  {[
                    { label: "Noise cancelling", img: "https://lh3.googleusercontent.com/aida-public/AB6AXuD1lgj8TLc5-fWqfT0qqf8dfv0_h0BGevFKHy6QZrda3EUHSJPjSSNEU8yUhlSIW02c7K8NvsY8Y9of28MYZji0ypbC9XsgDZFyAnL1MzC_UHE-xOr0-e_KL-ffEWgejSg6gYyOoV-e1chF0ETnFEep86-WtoT6l63nnNCcvEVS9lORiG1-aGQyQZuGwTTlYrbaWc5TiiywAZGzQKYO8wmWBvET6Aekhw-WMngkCP4slOzeFEzHZWmi" },
                    { label: "Smart tablets", img: "https://lh3.googleusercontent.com/aida-public/AB6AXuD45ws-5KNX1PZdKUzIROugvXmu2HkTdYkpqmP7YCz_ti59XypvaSXQI3iEQ-8ulznoq6zySexLiCctr7ni1u66czrntv02IzGC9uFEBenG757MC0utBvYY87qT6o3ET5rafu5kjvSaWZ8GPV1bN3rWAwL_7vi4lr6QvPGMlwbJfRJ3DOgfX0GYEW2hsLvXMFoIwbtOLBWnOR7rN-_esVVPo5UX3XzYwJL-_1tTWdgmw3fUTr6P_r8Z" },
                    { label: "4K monitors", img: "https://lh3.googleusercontent.com/aida-public/AB6AXuDsGJ617PXlEOaIFWAvSIE_eLbSBJeOZl9kAe3o4Af1ff6akKlh-FpnfetFWIEUY1YedB5F8zUt7ITvqDzkJrOLJbEJP51-lrMxvhhCCD5MXPFFbJuEBP3cO63zlldQ5ermVTJ4ZdS1QcMnPCfskYUDezo8bGxZCnT_NeaNdT1cNnlukSsWa1NkWbKGha5_Y7oMKDuI5NWxeRfsJ_fQppjdK5Gzr0ySuRwUDpuayjXaLLy6ro5oD5fE" },
                    { label: "Wireless chargers", img: "https://lh3.googleusercontent.com/aida-public/AB6AXuDfZTrK-SHvlZnv1p7nDAofaA8ojIuf1m5hdEJukH_qw5Rng9nMduFUQjsHnXpGFLASmaJvNpEQ_ovZ96n9_e2BCDGFXS3EOSN17BLTLW5YDSBoVbiLbI09wMr_UwHdxacTA90_3KExDiwFE-dkB77MtEJzqMlzI38d_moB5P8MWNh9cOzv4428myDoSnhLrF29Wfoi134LKkWdrubhDHVm8RpgDRMiFkXQNyJk-QdWvMeLVLLpktIR" },
                  ].map((item) => (
                    <Link key={item.label} href="/products" className="group block">
                      <div className="aspect-square bg-surface-container-low rounded overflow-hidden mb-1 relative">
                        <Image fill sizes="(max-width: 640px) 45vw, 150px" className="object-cover group-hover:scale-105 transition-transform duration-200" src={item.img} alt={item.label} />
                      </div>
                      <span className="font-body-xs text-body-xs text-on-surface-variant group-hover:text-primary-container font-medium">{item.label}</span>
                    </Link>
                  ))}
                </div>
              </div>
              <Link href="/products" className="font-label-md text-label-md font-semibold text-primary-container hover:underline inline-flex items-center gap-1">
                Shop tech deals →
              </Link>
            </div>

            {/* Card 3: Deal of the Day */}
            {featuredProduct && (
            <div className="self-start w-full bg-surface-container-lowest border border-outline-variant rounded-lg p-4 shadow-sm flex flex-col gap-3 hover:shadow-md transition-shadow relative">
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="px-2 py-0.5 rounded bg-tertiary-container text-tertiary-fixed font-label-sm text-label-sm font-bold uppercase tracking-wider">
                    {DEMO_MODE ? "Deal of the Day" : "Featured Product"}
                  </span>
                  {DEMO_MODE && <span className="font-body-xs text-body-xs text-error font-semibold flex items-center gap-0.5">⏱ Ends 06:42:19</span>}
                </div>
                <div className="h-36 sm:h-40 bg-surface-container-low rounded overflow-hidden mb-3 relative flex items-center justify-center p-3">
                  {featuredProduct.image.startsWith("http") || featuredProduct.image.startsWith("/")
                    ? <span aria-label={featuredProduct.name} style={{ position: "absolute", inset: 0, backgroundImage: `url("${featuredProduct.image}")`, backgroundPosition: "center", backgroundSize: "contain", backgroundRepeat: "no-repeat" }} />
                    : <span style={{ fontSize: "4rem" }}>{featuredProduct.image}</span>}
                  {featuredProduct.discount > 0 && <span className="absolute top-2 left-2 px-2 py-0.5 bg-error text-on-error font-label-sm text-label-sm font-bold rounded">-{featuredProduct.discount}%</span>}
                </div>
                <h3 className="font-headline-sm text-headline-sm font-bold text-on-surface line-clamp-1 mb-1">
                  {featuredProduct.name}
                </h3>
                {DEMO_MODE && <div className="flex items-center gap-1.5 mb-2">
                  <div className="flex text-secondary-container text-sm">⭐⭐⭐⭐½</div>
                  <span className="font-body-xs text-body-xs text-outline font-medium">(14,290)</span>
                </div>}
                <div className="flex items-baseline gap-2 mb-3">
                  <span className="font-price-card text-price-card font-bold" style={{ color: "var(--accent-primary)" }}>₹{fmtNum(featuredProduct.price)}</span>
                  {featuredProduct.originalPrice > featuredProduct.price && <span className="font-body-xs text-body-xs text-outline line-through">₹{fmtNum(featuredProduct.originalPrice)}</span>}
                  {featuredProduct.discount > 0 && <span className="font-body-xs text-body-xs font-semibold" style={{ color: "var(--accent-emerald)" }}>{featuredProduct.discount}% off</span>}
                </div>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <button
                  type="button"
                  disabled={!canPurchaseFeatured || purchasePending}
                  onClick={() => void addFeaturedToCart(false)}
                  className="w-full min-h-10 px-2 bg-secondary-container hover:bg-secondary text-primary-container font-label-md text-label-md font-bold rounded-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {purchasePending ? "Please wait…" : "Add to Cart"}
                </button>
                <button
                  type="button"
                  disabled={!canPurchaseFeatured || purchasePending}
                  onClick={() => void addFeaturedToCart(true)}
                  className="w-full min-h-10 px-2 bg-primary-container hover:bg-primary text-on-primary font-label-md text-label-md font-bold rounded-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Buy Now
                </button>
              </div>
              {!canPurchaseFeatured && <p className="font-body-xs text-body-xs text-error">This product is currently unavailable.</p>}
              {purchaseError && <p role="alert" className="font-body-xs text-body-xs text-error">{purchaseError}</p>}
            </div>
            )}

            {/* Card 4: Account Personalization & Fashion */}
            <div className="bg-surface-container-lowest border border-outline-variant rounded-lg p-4 shadow-sm flex flex-col justify-between hover:shadow-md transition-shadow">
              <div>
                <div className="bg-surface-container-low p-3.5 rounded-lg border border-outline-variant/60 mb-4">
                  <h3 className="font-headline-sm text-headline-sm font-bold text-on-surface mb-1">Sign in for your best experience</h3>
                  <p className="font-body-xs text-body-xs text-on-surface-variant mb-3">Track packages, view saved lists, and claim exclusive member discounts.</p>
                  <Link href="/login" className="block w-full text-center py-2 bg-secondary-container hover:bg-secondary text-primary-container font-label-md text-label-md font-bold rounded-full transition-all shadow-sm">
                    Sign in securely
                  </Link>
                </div>
                <h4 className="font-label-md text-label-md font-bold text-on-surface mb-2">Trending in Fashion</h4>
                <div className="aspect-[16/10] bg-surface-container-low rounded overflow-hidden relative group">
                  <Image
                    fill
                    sizes="(max-width: 640px) 90vw, 360px"
                    className="object-cover group-hover:scale-105 transition-transform duration-200"
                    src="https://lh3.googleusercontent.com/aida-public/AB6AXuC3qM9bTcGFUEJUhi-amlbeLTQJ4Nk_5dy6Dc3GrRpAB4tf3D7eB4AMJo7wcFQT4lb8s76_MCE-5ybFX1tP9a68aHBzR6LYh509nRbOXf37mrzCbL3DO3RXpsTBOImb0XZq-Rwsz-_yOTEeskPl9FWVHvOafS_qXRb1F5ruh0-XGDYQLc8NHlrdnb917laAwuDtANsNclDDlwD6hAsPwF87Wen3RALJg56v_MZmE0y_I_sOcm5zyNSw"
                    alt="Fashion outerwear"
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-primary-container/80 to-transparent flex items-end p-2.5">
                    <span className="text-on-primary font-label-sm text-label-sm font-semibold">Outerwear Essentials up to 30% off</span>
                  </div>
                </div>
              </div>
              <Link href="/products" className="font-label-md text-label-md font-semibold text-primary-container hover:underline inline-flex items-center gap-1 mt-3">
                Explore wardrobe arrivals →
              </Link>
            </div>
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}

"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Trash2, CreditCard } from "lucide-react";

type CartItem = {
    offer_id: string;
    vendor_id: string;
    price: number;
    title: string;
    quantity: number;
};

type CartData = {
    cart: { user_id: string; items: CartItem[] };
    total: number;
    user: string;
};

export default function CartPage() {
    const [data, setData] = useState<CartData | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    useEffect(() => {
        const BFF_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3000";

        // In actual implementation, cookies are sent automatically to BFF
        fetch(`${BFF_URL}/cart`)
            .then((res) => {
                if (!res.ok) {
                    if (res.status === 401) window.location.href = "/login";
                    throw new Error("Failed to load cart");
                }
                return res.json();
            })
            .then((json) => setData(json))
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    }, []);

    if (loading) return <div className="container py-16 text-center animate-pulse">Loading Cart...</div>;
    if (error) return <div className="container py-16 text-center text-red-500">{error}</div>;

    const items = data?.cart?.items || [];

    return (
        <div className="container py-12 px-4 md:px-6 max-w-5xl mx-auto">
            <h1 className="text-3xl font-bold tracking-tight text-gray-900 mb-8">Shopping Cart</h1>

            {items.length === 0 ? (
                <div className="text-center py-16 bg-white rounded-xl border border-dashed shadow-sm">
                    <ShoppingCartIcon className="mx-auto h-12 w-12 text-gray-300 mb-4" />
                    <h3 className="text-lg font-medium text-gray-900 mb-2">Your cart is empty</h3>
                    <p className="text-gray-500 mb-6">Looks like you haven't added anything to your cart yet.</p>
                    <Link
                        href="/"
                        className="inline-flex items-center justify-center rounded-md bg-blue-600 px-6 py-3 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 transition-colors"
                    >
                        Start Shopping
                    </Link>
                </div>
            ) : (
                <div className="lg:grid lg:grid-cols-12 lg:items-start lg:gap-x-12 xl:gap-x-16">
                    <section aria-labelledby="cart-heading" className="lg:col-span-8">
                        <h2 id="cart-heading" className="sr-only">Items in your shopping cart</h2>

                        <ul role="list" className="divide-y divide-gray-200 border-b border-t border-gray-200">
                            {items.map((item, itemIdx) => (
                                <li key={itemIdx} className="flex py-6 sm:py-10">
                                    <div className="flex-shrink-0">
                                        <div className="h-24 w-24 rounded-md bg-gray-100 flex items-center justify-center border border-gray-200">
                                            <span className="text-gray-400 text-xs">No Image</span>
                                        </div>
                                    </div>

                                    <div className="ml-4 flex flex-1 flex-col justify-between sm:ml-6">
                                        <div className="relative pr-9 sm:grid sm:grid-cols-2 sm:gap-x-6 sm:pr-0">
                                            <div>
                                                <div className="flex justify-between">
                                                    <h3 className="text-sm font-medium text-gray-700 hover:text-gray-800">
                                                        {item.title}
                                                    </h3>
                                                </div>
                                                <p className="mt-1 text-sm font-medium text-gray-900">${item.price.toFixed(2)}</p>
                                            </div>

                                            <div className="mt-4 sm:mt-0 sm:pr-9">
                                                <label htmlFor={`quantity-${itemIdx}`} className="sr-only">
                                                    Quantity, {item.title}
                                                </label>
                                                <select
                                                    id={`quantity-${itemIdx}`}
                                                    name={`quantity-${itemIdx}`}
                                                    className="max-w-full rounded-md border border-gray-300 py-1.5 text-left text-base font-medium leading-5 text-gray-700 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 sm:text-sm"
                                                    defaultValue={item.quantity}
                                                >
                                                    <option value={1}>1</option>
                                                    <option value={2}>2</option>
                                                    <option value={3}>3</option>
                                                </select>

                                                <div className="absolute right-0 top-0">
                                                    <button type="button" className="-m-2 inline-flex p-2 text-gray-400 hover:text-red-500">
                                                        <span className="sr-only">Remove</span>
                                                        <Trash2 className="h-5 w-5" aria-hidden="true" />
                                                    </button>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    </section>

                    {/* Order summary */}
                    <section
                        aria-labelledby="summary-heading"
                        className="mt-16 rounded-lg bg-gray-50 px-4 py-6 sm:p-6 lg:col-span-4 lg:mt-0 lg:p-8 border"
                    >
                        <h2 id="summary-heading" className="text-lg font-medium text-gray-900">
                            Order summary
                        </h2>

                        <dl className="mt-6 space-y-4 text-sm text-gray-600">
                            <div className="flex items-center justify-between border-t border-gray-200 pt-4">
                                <dt className="text-base font-medium text-gray-900">Order total</dt>
                                <dd className="text-base font-bold text-gray-900">${(data?.total || 0).toFixed(2)}</dd>
                            </div>
                        </dl>

                        <div className="mt-6">
                            <Link
                                href="/payment"
                                className="w-full rounded-md border border-transparent bg-blue-600 px-4 py-3 text-base font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-gray-50 flex items-center justify-center gap-2 transition-colors"
                            >
                                <CreditCard className="w-5 h-5" />
                                Checkout
                            </Link>
                        </div>
                    </section>
                </div>
            )}
        </div>
    );
}

function ShoppingCartIcon(props: any) {
    return (
        <svg
            {...props}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
        >
            <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z"
            />
        </svg>
    );
}

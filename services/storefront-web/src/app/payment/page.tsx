"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ShieldCheck } from "lucide-react";

type CheckoutData = {
    total: number;
    user: string;
};

export default function PaymentGateway() {
    const router = useRouter();
    const [data, setData] = useState<CheckoutData | null>(null);
    const [loading, setLoading] = useState(true);
    const [processing, setProcessing] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        const BFF_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3000";

        fetch(`${BFF_URL}/checkout`)
            .then((res) => {
                if (!res.ok) {
                    if (res.status === 401) window.location.href = "/login";
                    throw new Error("Failed to load checkout data");
                }
                return res.json();
            })
            .then((json) => setData(json))
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    }, []);

    const handlePayment = async (e: React.FormEvent) => {
        e.preventDefault();
        setProcessing(true);

        try {
            const BFF_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3000";
            const res = await fetch(`${BFF_URL}/payment/process`, {
                method: "POST",
            });

            if (!res.ok) throw new Error("Payment failed to process");

            const result = await res.json();
            // Redirect to success page with query params
            router.push(`/checkout?days=${result.deliveryDays}`);
        } catch (err: any) {
            setError(err.message);
            setProcessing(false);
        }
    };

    if (loading) return <div className="container py-16 text-center animate-pulse">Initializing Secure Gateway...</div>;

    return (
        <div className="bg-gray-50 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8 mt-12 mb-12">
            <div className="w-full max-w-md bg-white p-8 rounded-xl shadow-lg border">

                <div className="text-center mb-8">
                    <ShieldCheck className="mx-auto h-12 w-12 text-emerald-500 mb-2" />
                    <h2 className="text-2xl font-bold text-gray-900">Secure Payment</h2>
                    <p className="text-gray-500 text-sm mt-1">Stripe Mock Gateway</p>
                </div>

                <div className="bg-gray-50 border rounded-lg p-4 mb-8">
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-gray-600">Total Amount:</span>
                        <span className="text-2xl font-bold text-gray-900">${data?.total.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between items-center text-sm">
                        <span className="text-gray-500">Account:</span>
                        <span className="font-medium">{data?.user}</span>
                    </div>
                </div>

                {error && (
                    <div className="mb-4 rounded-md bg-red-50 p-4 border border-red-200 text-sm text-red-700">
                        {error}
                    </div>
                )}

                <form onSubmit={handlePayment} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">Card Number (Mock)</label>
                        <input
                            type="text"
                            defaultValue="4242 4242 4242 4242"
                            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm bg-gray-50"
                            disabled
                        />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">Expiry</label>
                            <input type="text" defaultValue="12/25" className="block w-full rounded-md border-gray-300 shadow-sm sm:text-sm bg-gray-50" disabled />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">CVC</label>
                            <input type="text" defaultValue="123" className="block w-full rounded-md border-gray-300 shadow-sm sm:text-sm bg-gray-50" disabled />
                        </div>
                    </div>

                    <button
                        type="submit"
                        disabled={processing || data?.total === 0}
                        className="w-full mt-6 bg-emerald-600 border border-transparent rounded-md shadow-sm py-3 px-4 text-base font-semibold text-white hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500 transition-all disabled:opacity-75 disabled:cursor-not-allowed"
                    >
                        {processing ? "Processing Transaction..." : `Pay $${data?.total.toFixed(2)}`}
                    </button>
                </form>

                <p className="mt-4 text-center text-xs text-gray-400">
                    This is a mock transaction for the enterprise DevOps demonstration environment. No real funds are captured.
                </p>

            </div>
        </div>
    );
}

"use client";

import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { CheckCircle } from "lucide-react";

export default function CheckoutComplete() {
    const searchParams = useSearchParams();
    const days = searchParams.get("days") || "3-5";

    return (
        <div className="flex min-h-[60vh] items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
            <div className="w-full max-w-md bg-white p-10 rounded-2xl shadow-xl border border-gray-100 text-center">

                <div className="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-emerald-100 mb-6">
                    <CheckCircle className="h-12 w-12 text-emerald-600" />
                </div>

                <h2 className="text-3xl font-extrabold text-gray-900 mb-2">Order Confirmed!</h2>
                <p className="text-lg text-gray-600 mb-8">
                    Your payment was completely successful.
                </p>

                <div className="bg-gray-50 rounded-lg p-6 mb-8 text-left border">
                    <h3 className="font-semibold text-gray-900 mb-2 border-b pb-2">Delivery Estimate</h3>
                    <p className="text-gray-700">
                        Expect your items to arrive in <span className="font-bold text-blue-600">{days}</span> business days.
                    </p>
                    <p className="text-sm text-gray-500 mt-2">
                        A confirmation receipt has been generated and dispatched via the microservice event bus.
                    </p>
                </div>

                <Link
                    href="/"
                    className="inline-flex w-full items-center justify-center rounded-md bg-blue-600 px-6 py-3 text-base font-semibold text-white shadow-sm hover:bg-blue-500 transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
                >
                    Continue Shopping
                </Link>
            </div>
        </div>
    );
}

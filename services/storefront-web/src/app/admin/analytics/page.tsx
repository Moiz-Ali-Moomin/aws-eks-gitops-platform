"use client";

import { useEffect, useState } from "react";
import { BarChart3, TrendingUp, Users, AlertCircle } from "lucide-react";

export default function AdminAnalyticsDashboard() {
    const [iframeUrl, setIframeUrl] = useState("");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        // Note: The admin backoffice API would typically be on a different path/port than the storefront BFF
        // In production, the API Gateway routes `/admin/api/*` to admin-backoffice-service
        const ADMIN_API_URL = process.env.NEXT_PUBLIC_ADMIN_API_URL || "http://localhost:8080";

        const fetchMetabaseToken = async () => {
            try {
                const res = await fetch(`${ADMIN_API_URL}/api/analytics/token`);
                if (!res.ok) throw new Error("Failed to fetch analytics secure token");

                const data = await res.json();

                // Construct the frontend iframe URL
                const metabaseSiteUrl = "http://localhost:3000"; // Assuming local metabase
                const finalUrl = `${metabaseSiteUrl}/embed/dashboard/${data.token}#bordered=false&titled=false`;

                setIframeUrl(finalUrl);
            } catch (err: any) {
                setError(err.message);
            } finally {
                setLoading(false);
            }
        };

        fetchMetabaseToken();
    }, []);

    return (
        <div className="min-h-screen bg-gray-50 flex flex-col">
            <header className="bg-slate-900 text-white shadow">
                <div className="container mx-auto px-4 py-4 flex justify-between items-center">
                    <div className="flex items-center gap-3">
                        <BarChart3 className="h-6 w-6 text-blue-400" />
                        <h1 className="text-xl font-bold">Admin Backoffice</h1>
                    </div>
                    <div className="text-sm font-medium text-slate-300">
                        System Status: <span className="text-emerald-400">Operational</span>
                    </div>
                </div>
            </header>

            <main className="flex-1 container mx-auto px-4 py-8">
                <div className="mb-8">
                    <h2 className="text-2xl font-bold text-gray-900">Analytics Overview</h2>
                    <p className="text-gray-500 mt-1">Real-time performance metrics via Metabase</p>
                </div>

                {/* Quick Stats Row */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                    <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-100 flex items-center gap-4">
                        <div className="p-3 bg-blue-50 text-blue-600 rounded-lg">
                            <TrendingUp className="h-6 w-6" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-gray-500">Total Revenue</p>
                            <h3 className="text-2xl font-bold text-gray-900">$124,500</h3>
                        </div>
                    </div>
                    <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-100 flex items-center gap-4">
                        <div className="p-3 bg-emerald-50 text-emerald-600 rounded-lg">
                            <Users className="h-6 w-6" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-gray-500">Active Users</p>
                            <h3 className="text-2xl font-bold text-gray-900">8,230</h3>
                        </div>
                    </div>
                    <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-100 flex items-center gap-4">
                        <div className="p-3 bg-purple-50 text-purple-600 rounded-lg">
                            <BarChart3 className="h-6 w-6" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-gray-500">Conversion Rate</p>
                            <h3 className="text-2xl font-bold text-gray-900">4.2%</h3>
                        </div>
                    </div>
                </div>

                {/* Metabase Embed Container */}
                <div className="bg-white rounded-xl shadow-md border border-gray-200 overflow-hidden min-h-[600px] flex flex-col">
                    <div className="px-6 py-4 border-b border-gray-100 bg-gray-50 flex justify-between items-center">
                        <h3 className="font-semibold text-gray-700">Executive Dashboard</h3>
                    </div>

                    <div className="flex-1 relative bg-slate-50">
                        {loading ? (
                            <div className="absolute inset-0 flex items-center justify-center">
                                <div className="animate-pulse flex flex-col items-center">
                                    <div className="h-8 w-8 bg-blue-200 rounded-full mb-4"></div>
                                    <div className="text-gray-500 font-medium">Authenticating with Metabase...</div>
                                </div>
                            </div>
                        ) : error ? (
                            <div className="absolute inset-0 flex items-center justify-center p-6">
                                <div className="text-center">
                                    <AlertCircle className="mx-auto h-12 w-12 text-red-500 mb-4" />
                                    <h3 className="text-lg font-medium text-gray-900 mb-2">Analytics Currently Unavailable</h3>
                                    <p className="text-red-500 bg-red-50 px-4 py-2 rounded-md">{error}</p>
                                </div>
                            </div>
                        ) : iframeUrl ? (
                            <iframe
                                src={iframeUrl}
                                frameBorder="0"
                                width="100%"
                                height="100%"
                                className="absolute inset-0 w-full h-full"
                                allowTransparency
                            />
                        ) : null}
                    </div>
                </div>
            </main>
        </div>
    );
}

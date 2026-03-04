import Link from "next/link";
import { ShoppingCart, User } from "lucide-react";

export default function Navbar() {
    // In a real app, you'd fetch user state from Zustand or React Query.
    // We'll mock it temporarily during the page migration.
    const user = null;

    return (
        <header className="sticky top-0 z-50 w-full border-b bg-white/95 backdrop-blur supports-[backdrop-filter]:bg-white/60">
            <div className="container flex h-16 items-center justify-between px-4 md:px-6">
                <Link href="/" className="flex items-center gap-2">
                    <span className="text-xl font-bold tracking-tight">GO commerce™</span>
                </Link>

                <nav className="flex items-center gap-6">
                    <Link href="/" className="text-sm font-medium hover:text-blue-600 transition-colors">
                        Home
                    </Link>
                    <Link href="/cart" className="relative text-sm font-medium hover:text-blue-600 transition-colors flex items-center gap-1">
                        <ShoppingCart className="h-4 w-4" />
                        Cart
                    </Link>

                    {user ? (
                        <div className="flex items-center gap-4 ml-2 pl-4 border-l">
                            <span className="text-sm text-gray-600 flex items-center gap-1">
                                <User className="h-4 w-4" /> Hi, {user}
                            </span>
                            <Link href="/logout" className="text-sm font-medium text-red-600 hover:text-red-700">
                                Logout
                            </Link>
                        </div>
                    ) : (
                        <div className="ml-2 pl-4 border-l">
                            <Link
                                href="/login"
                                className="inline-flex h-9 items-center justify-center rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow transition-colors hover:bg-blue-700 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-700 disabled:pointer-events-none disabled:opacity-50"
                            >
                                Login / Sign Up
                            </Link>
                        </div>
                    )}
                </nav>
            </div>
        </header>
    );
}

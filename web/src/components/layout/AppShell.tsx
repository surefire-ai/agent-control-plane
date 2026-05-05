import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { Header } from "./Header";

export function AppShell() {
  return (
    <div className="console-shell flex h-screen overflow-hidden text-zinc-900">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-auto">
          {/* Subtle radial glow behind content area */}
          <div className="relative min-h-full">
            <div
              className="pointer-events-none absolute inset-0"
              style={{
                background:
                  "radial-gradient(ellipse 60% 40% at 50% 0%, rgba(20, 184, 166, 0.04), transparent 70%)",
              }}
              aria-hidden="true"
            />
            <div className="relative mx-auto w-full max-w-7xl px-5 py-6 sm:px-8 lg:px-10">
              <Outlet />
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}

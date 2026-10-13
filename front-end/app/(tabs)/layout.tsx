import { AuthGate } from "@/components/AuthGate";
import { BottomNav } from "@/components/BottomNav";
import { ServiceWorkerRegistrar } from "@/components/ServiceWorkerRegistrar";

export default function TabsLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGate>
      <ServiceWorkerRegistrar />
      <div className="bq-screen">
        <main className="bq-body">{children}</main>
        <BottomNav />
      </div>
    </AuthGate>
  );
}

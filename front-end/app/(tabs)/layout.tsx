import { BottomNav } from "@/components/BottomNav";

export default function TabsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="bq-screen">
      <main className="bq-body">{children}</main>
      <BottomNav />
    </div>
  );
}

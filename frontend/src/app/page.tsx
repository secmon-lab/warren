import { MainLayout } from '@/components/layout/main-layout';

export default function Dashboard() {
  return (
    <MainLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Warren</h1>
          <p className="text-muted-foreground">
            Security monitoring and ticket management system
          </p>
        </div>
      </div>
    </MainLayout>
  );
}

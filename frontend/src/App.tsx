import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { ApolloProvider } from "@apollo/client";
import { apolloClient } from "@/lib/apollo-client";
import { AuthProvider } from "@/contexts/auth-context";
import { AuthGuard } from "@/components/auth/auth-guard";
import { MainLayout } from "@/components/layout/main-layout";
import { ToastProvider } from "@/hooks/use-toast";
import { ConfirmProvider } from "@/hooks/use-confirm";
import { Toaster } from "@/components/ui/toaster";
import Dashboard from "@/pages/Dashboard";
import TicketsPage from "@/pages/TicketsPage";
import TicketDetailPage from "@/pages/TicketDetailPage";
import AlertsPage from "@/pages/AlertsPage";
import AlertDetailPage from "@/pages/AlertDetailPage";
import BoardPage from "@/pages/BoardPage";
import NotFoundPage from "@/pages/NotFoundPage";

function AuthenticatedApp() {
  return (
    <MainLayout>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/tickets" element={<TicketsPage />} />
        <Route path="/tickets/:id" element={<TicketDetailPage />} />
        <Route path="/alerts" element={<AlertsPage />} />
        <Route path="/alerts/:id" element={<AlertDetailPage />} />
        <Route path="/board" element={<BoardPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </MainLayout>
  );
}

function App() {
  return (
    <ApolloProvider client={apolloClient}>
      <AuthProvider>
        <ToastProvider>
          <ConfirmProvider>
            <Router>
              <AuthGuard>
                <AuthenticatedApp />
              </AuthGuard>
              <Toaster />
            </Router>
          </ConfirmProvider>
        </ToastProvider>
      </AuthProvider>
    </ApolloProvider>
  );
}

export default App;

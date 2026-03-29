import { BrowserRouter as Router, Routes, Route, Navigate } from "react-router-dom";
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
import KnowledgePage from "@/pages/KnowledgePage";
import KnowledgeDetailPage from "@/pages/KnowledgeDetailPage";
import KnowledgeEditPage from "@/pages/KnowledgeEditPage";
import KnowledgeTagsPage from "@/pages/KnowledgeTagsPage";
import SessionDetailPage from "@/pages/SessionDetailPage";
import SettingsPage from "@/pages/SettingsPage";
import DiagnosisPage from "@/pages/DiagnosisPage";
import DiagnosisDetailPage from "@/pages/DiagnosisDetailPage";
import QueuedAlertsPage from "@/pages/QueuedAlertsPage";
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
        <Route path="/knowledge" element={<Navigate to="/knowledge/fact" replace />} />
        <Route path="/knowledge/new" element={<KnowledgeEditPage />} />
        <Route path="/knowledge/tags" element={<KnowledgeTagsPage />} />
        <Route path="/knowledge/fact" element={<KnowledgePage />} />
        <Route path="/knowledge/technique" element={<KnowledgePage />} />
        <Route path="/knowledge/:id" element={<KnowledgeDetailPage />} />
        <Route path="/knowledge/:id/edit" element={<KnowledgeEditPage />} />
        <Route path="/sessions/:id" element={<SessionDetailPage />} />
        <Route path="/diagnosis" element={<DiagnosisPage />} />
        <Route path="/diagnosis/:id" element={<DiagnosisDetailPage />} />
        <Route path="/queue" element={<QueuedAlertsPage />} />
        <Route path="/settings" element={<SettingsPage />} />
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

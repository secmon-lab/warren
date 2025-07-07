import { useAuth } from "@/contexts/auth-context";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export function LoginPage() {
  const { login } = useAuth();

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-1">
          <CardTitle className="text-2xl text-center">Warren</CardTitle>
          <CardDescription className="text-center">
            Security incident management system
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="text-center text-sm text-gray-600 mb-4">
            Please sign in with your Slack account to continue.
          </div>
          <Button onClick={login} className="w-full" size="lg">
            Sign in with Slack
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

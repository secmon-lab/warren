import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { User, Bot, ShieldCheck, AlertTriangle, Info, Star, Pencil } from 'lucide-react';
import { Ticket, AlertConclusion, ALERT_CONCLUSION_LABELS } from '@/lib/types';

interface ResolveInfoProps {
  ticket: Ticket;
  onEditConclusion?: () => void;
}

export function ResolveInfo({ ticket, onEditConclusion }: ResolveInfoProps) {
  const isResolved = ticket.status === 'resolved';
  const hasConclusion = ticket.conclusion || ticket.reason;
  const hasFinding = ticket.finding;

  // Don't display anything if not resolved or has neither conclusion nor finding
  if (!isResolved && !hasFinding) {
    return null;
  }

  const getSeverityIcon = (severity: string) => {
    switch (severity.toLowerCase()) {
      case 'critical':
        return <AlertTriangle className="h-4 w-4 text-red-500" />;
      case 'high':
        return <ShieldCheck className="h-4 w-4 text-orange-500" />;
      case 'medium':
        return <Info className="h-4 w-4 text-yellow-500" />;
      case 'low':
        return <Star className="h-4 w-4 text-blue-500" />;
      default:
        return <Info className="h-4 w-4 text-gray-500" />;
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity.toLowerCase()) {
      case 'critical':
        return 'bg-red-100 text-red-800 border-red-200';
      case 'high':
        return 'bg-orange-100 text-orange-800 border-orange-200';
      case 'medium':
        return 'bg-yellow-100 text-yellow-800 border-yellow-200';
      case 'low':
        return 'bg-blue-100 text-blue-800 border-blue-200';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  };

  const getConclusionBadgeColor = (conclusion: string) => {
    switch (conclusion) {
      case 'true_positive':
        return 'bg-red-100 text-red-800 border-red-200 hover:bg-red-200';
      case 'false_positive':
        return 'bg-gray-100 text-gray-800 border-gray-200 hover:bg-gray-200';
      case 'unaffected':
        return 'bg-blue-100 text-blue-800 border-blue-200 hover:bg-blue-200';
      case 'intended':
        return 'bg-green-100 text-green-800 border-green-200 hover:bg-green-200';
      case 'escalated':
        return 'bg-orange-100 text-orange-800 border-orange-200 hover:bg-orange-200';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200 hover:bg-gray-200';
    }
  };

  return (
    <div className="space-y-4">
      {/* Human Conclusion Section */}
      {isResolved && (
        <Card className="border-green-200 bg-green-50/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-green-800">
              <User className="h-5 w-5" />
              Human Review
              
              {/* Conclusion Badge in Header */}
              {ticket.conclusion && (
                <Badge 
                  variant="outline" 
                  className={`text-sm font-medium px-3 py-1 ${getConclusionBadgeColor(ticket.conclusion)}`}
                >
                  {ALERT_CONCLUSION_LABELS[ticket.conclusion as AlertConclusion]}
                </Badge>
              )}

              <div className="ml-auto flex items-center gap-2">
                {onEditConclusion && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onEditConclusion}
                    className="h-7 px-2 bg-white hover:bg-green-100 border-green-300"
                  >
                    <Pencil className="h-3 w-3 mr-1" />
                    Edit
                  </Button>
                )}
              </div>
            </CardTitle>
          </CardHeader>
          
          {hasConclusion ? (
            <CardContent>
              {ticket.reason && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-green-800">Reason</label>
                  <div className="bg-white/70 border border-green-200 rounded-lg p-3">
                    <p className="text-sm text-green-700 leading-relaxed whitespace-pre-wrap">
                      {ticket.reason}
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
          ) : (
            <CardContent>
              <div className="text-center py-4">
                <div className="w-10 h-10 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-2">
                  <User className="h-5 w-5 text-green-600" />
                </div>
                <p className="text-sm text-green-600 mb-2 font-medium">
                  No conclusion set
                </p>
                <p className="text-xs text-green-500 mb-3">
                  Add a conclusion to document the resolution.
                </p>
                {onEditConclusion && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onEditConclusion}
                    className="bg-white hover:bg-green-100 border-green-300"
                  >
                    <Pencil className="h-3 w-3 mr-1" />
                    Add Conclusion
                  </Button>
                )}
              </div>
            </CardContent>
          )}
        </Card>
      )}

      {/* AI Finding Section */}
      {hasFinding && ticket.finding && (
        <Card className="border-blue-200 bg-blue-50/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-blue-800">
              <Bot className="h-5 w-5" />
              AI Analysis
              
              {/* Severity Badge in Header */}
              <div className="flex items-center gap-2">
                {getSeverityIcon(ticket.finding.severity)}
                <Badge 
                  variant="outline" 
                  className={`text-xs ${getSeverityColor(ticket.finding.severity)}`}
                >
                  {ticket.finding.severity.toUpperCase()}
                </Badge>
              </div>

              <Badge variant="outline" className="ml-auto text-xs bg-blue-100 text-blue-700 border-blue-300">
                AI Generated
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Summary */}
            {ticket.finding.summary && (
              <div>
                <label className="text-sm font-medium text-blue-800">Summary</label>
                <p className="text-sm text-blue-700 mt-1 leading-relaxed">
                  {ticket.finding.summary}
                </p>
              </div>
            )}

            {/* Reason */}
            {ticket.finding.reason && (
              <div>
                <label className="text-sm font-medium text-blue-800">Analysis</label>
                <p className="text-sm text-blue-700 mt-1 leading-relaxed whitespace-pre-wrap">
                  {ticket.finding.reason}
                </p>
              </div>
            )}

            {/* Recommendation */}
            {ticket.finding.recommendation && (
              <div>
                <label className="text-sm font-medium text-blue-800">Recommendation</label>
                <p className="text-sm text-blue-700 mt-1 leading-relaxed whitespace-pre-wrap">
                  {ticket.finding.recommendation}
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
} 
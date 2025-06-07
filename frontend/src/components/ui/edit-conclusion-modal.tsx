import { useState } from "react";
import { useMutation } from "@apollo/client";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useErrorToast, useSuccessToast } from "@/hooks/use-toast";
import { UPDATE_TICKET_CONCLUSION, GET_TICKET } from "@/lib/graphql/queries";
import { 
  AlertConclusion, 
  ALERT_CONCLUSION_LABELS,
  ALERT_CONCLUSION_DESCRIPTIONS
} from "@/lib/types";

interface EditConclusionModalProps {
  isOpen: boolean;
  onClose: () => void;
  ticketId: string;
  currentConclusion?: string;
  currentReason?: string;
}

export function EditConclusionModal({
  isOpen,
  onClose,
  ticketId,
  currentConclusion,
  currentReason,
}: EditConclusionModalProps) {
  const [conclusion, setConclusion] = useState<AlertConclusion | "">(
    (currentConclusion as AlertConclusion) || ""
  );
  const [reason, setReason] = useState(currentReason || "");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const errorToast = useErrorToast();
  const successToast = useSuccessToast();

  const [updateTicketConclusion] = useMutation(UPDATE_TICKET_CONCLUSION, {
    refetchQueries: [{ query: GET_TICKET, variables: { id: ticketId } }],
  });

  const handleSubmit = async () => {
    if (!conclusion) {
      errorToast("Please select a conclusion");
      return;
    }

    setIsSubmitting(true);
    try {
      await updateTicketConclusion({
        variables: {
          id: ticketId,
          conclusion: conclusion,
          reason: reason,
        },
      });
      successToast("Ticket conclusion updated successfully");
      onClose();
    } catch (error) {
      console.error("Failed to update ticket conclusion:", error);
      errorToast("Failed to update ticket conclusion");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = () => {
    if (!isSubmitting) {
      onClose();
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit Ticket Conclusion</DialogTitle>
        </DialogHeader>
        
        <div className="space-y-6">
          {/* Conclusion Selection */}
          <div className="space-y-3">
            <Label htmlFor="conclusion" className="text-sm font-medium">
              Conclusion
            </Label>
            <Select
              value={conclusion}
              onValueChange={(value) => setConclusion(value as AlertConclusion)}
              disabled={isSubmitting}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Select a conclusion..." />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(ALERT_CONCLUSION_LABELS).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            
            {/* Selected Conclusion Description */}
            {conclusion && (
              <div className="bg-muted/50 rounded-lg p-3 border">
                <p className="text-sm text-muted-foreground">
                  {ALERT_CONCLUSION_DESCRIPTIONS[conclusion]}
                </p>
              </div>
            )}
          </div>

          {/* Reason Input */}
          <div className="space-y-3">
            <Label htmlFor="reason" className="text-sm font-medium">
              Reason
            </Label>
            <Textarea
              id="reason"
              placeholder="Add detailed reasoning, context, or additional information..."
              value={reason}
              onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setReason(e.target.value)}
              disabled={isSubmitting}
              rows={4}
              className="resize-none"
            />
            <p className="text-xs text-muted-foreground">
              Provide context for this conclusion to help future analysis.
            </p>
          </div>
        </div>

        <DialogFooter className="gap-2">
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isSubmitting || !conclusion}
            className="min-w-[80px]"
          >
            {isSubmitting ? "Updating..." : "Update"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
} 
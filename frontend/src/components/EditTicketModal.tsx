import { useState, useEffect } from "react";
import { useMutation } from "@apollo/client";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { UPDATE_TICKET, GET_TICKET } from "@/lib/graphql/queries";
import { useToast } from "@/hooks/use-toast";
import { Ticket } from "@/lib/types";

interface EditTicketModalProps {
  isOpen: boolean;
  onClose: () => void;
  ticket: Ticket | null;
}

export function EditTicketModal({
  isOpen,
  onClose,
  ticket,
}: EditTicketModalProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const { toast } = useToast();

  // Update local state when ticket prop changes
  useEffect(() => {
    if (ticket) {
      setTitle(ticket.title || "");
      setDescription(ticket.description || "");
    }
  }, [ticket]);

  const [updateTicket, { loading }] = useMutation(UPDATE_TICKET, {
    refetchQueries: [{ query: GET_TICKET, variables: { id: ticket?.id } }],
    onCompleted: () => {
      toast({
        title: "Ticket updated",
        description: "Your ticket has been successfully updated.",
      });
      onClose();
    },
    onError: (error) => {
      toast({
        title: "Error updating ticket",
        description: error.message,
        variant: "destructive",
      });
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      toast({
        title: "Missing information",
        description: "Please provide a title.",
        variant: "destructive",
      });
      return;
    }

    if (!ticket) {
      toast({
        title: "Error",
        description: "No ticket selected for editing.",
        variant: "destructive",
      });
      return;
    }

    updateTicket({
      variables: {
        id: ticket.id,
        title: title.trim(),
        description: description.trim() || null,
      },
    });
  };

  const handleClose = () => {
    // Reset form when closing
    if (ticket) {
      setTitle(ticket.title || "");
      setDescription(ticket.description || "");
    }
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Edit Ticket</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-title">Title</Label>
              <Input
                id="edit-title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Enter ticket title"
                disabled={loading}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-description">Description</Label>
              <Textarea
                id="edit-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Enter ticket description"
                rows={4}
                disabled={loading}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? "Updating..." : "Update Ticket"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

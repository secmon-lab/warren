import { useState } from "react";
import { useMutation } from "@apollo/client";
import { useNavigate } from "react-router-dom";
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
import { CREATE_TICKET, GET_TICKETS } from "@/lib/graphql/queries";
import { useToast } from "@/hooks/use-toast";

interface CreateTicketModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function CreateTicketModal({ isOpen, onClose }: CreateTicketModalProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [isTest, setIsTest] = useState(false);
  const navigate = useNavigate();
  const { toast } = useToast();

  const [createTicket, { loading }] = useMutation(CREATE_TICKET, {
    refetchQueries: [{ query: GET_TICKETS }],
    onCompleted: (data) => {
      const ticketType = data.createTicket.isTest ? "test ticket" : "ticket";
      toast({
        title: `${
          ticketType.charAt(0).toUpperCase() + ticketType.slice(1)
        } created`,
        description: `Your ${ticketType} has been successfully created.`,
      });
      onClose();
      setTitle("");
      setDescription("");
      setIsTest(false);
      // Navigate to the created ticket
      navigate(`/tickets/${data.createTicket.id}`);
    },
    onError: (error) => {
      toast({
        title: "Error creating ticket",
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

    createTicket({
      variables: {
        title: title.trim(),
        description: description.trim(),
        isTest: isTest,
      },
    });
  };

  const handleClose = () => {
    setTitle("");
    setDescription("");
    setIsTest(false);
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Create New Ticket</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="title">Title</Label>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Enter ticket title"
                disabled={loading}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Enter ticket description"
                rows={4}
                disabled={loading}
              />
            </div>
            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="isTest"
                checked={isTest}
                onChange={(e) => setIsTest(e.target.checked)}
                disabled={loading}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary disabled:cursor-not-allowed disabled:opacity-50"
              />
              <Label
                htmlFor="isTest"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                This is a test ticket 🧪
              </Label>
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
              {loading ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

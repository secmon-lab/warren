import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation } from "@apollo/client";
import {
  GET_KNOWLEDGE,
  GET_KNOWLEDGE_LOGS,
  DELETE_KNOWLEDGE,
} from "@/lib/graphql/queries";

export default function KnowledgeDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { data, loading } = useQuery(GET_KNOWLEDGE, {
    variables: { id },
    skip: !id,
  });
  const knowledge = data?.knowledge;

  const { data: logsData } = useQuery(GET_KNOWLEDGE_LOGS, {
    variables: { knowledgeID: id },
    skip: !id,
  });
  const logs = logsData?.knowledgeLogs || [];

  const [deleteKnowledge] = useMutation(DELETE_KNOWLEDGE);

  const handleDelete = async () => {
    const reason = prompt("Reason for deletion:");
    if (!reason || !id) return;
    await deleteKnowledge({ variables: { id, reason } });
    navigate("/knowledge");
  };

  if (loading) return <p className="text-sm text-gray-500">Loading...</p>;
  if (!knowledge) return <p className="text-sm text-gray-500">Knowledge not found.</p>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{knowledge.title}</h1>
          <p className="text-sm text-gray-500">
            Category: {knowledge.category} | Author: {knowledge.authorID}
          </p>
          <p className="text-sm text-gray-500">
            Created: {new Date(knowledge.createdAt).toISOString().split("T")[0].replace(/-/g, "/")} |
            Updated: {new Date(knowledge.updatedAt).toISOString().split("T")[0].replace(/-/g, "/")}
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => navigate(`/knowledge/${id}/edit`)}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            Edit
          </button>
          <button
            onClick={handleDelete}
            className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
          >
            Delete
          </button>
        </div>
      </div>

      {/* Tags */}
      <div className="flex gap-1">
        {knowledge.tags.map((tag: { id: string; name: string; description: string }) => (
          <span
            key={tag.id}
            className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600"
            title={tag.description}
          >
            {tag.name}
          </span>
        ))}
      </div>

      {/* Claim content */}
      <div className="rounded-lg border p-4">
        <pre className="whitespace-pre-wrap text-sm">{knowledge.claim}</pre>
      </div>

      {/* Change history */}
      <div>
        <h2 className="text-lg font-semibold">Change History</h2>
        {logs.length === 0 && <p className="text-sm text-gray-500">No history.</p>}
        <div className="mt-2 space-y-2">
          {logs.map((log: { id: string; message: string; authorID: string; ticketID: string | null; createdAt: string }) => (
            <div key={log.id} className="rounded border p-3 text-sm">
              <p className="font-medium">{log.message}</p>
              <p className="text-xs text-gray-500">
                By: {log.authorID}
                {log.ticketID && ` | Ticket: ${log.ticketID}`}
                {" | "}
                {new Date(log.createdAt).toISOString().split("T")[0].replace(/-/g, "/")}
              </p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

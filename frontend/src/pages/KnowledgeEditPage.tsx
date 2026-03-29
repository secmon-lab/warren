import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation } from "@apollo/client";
import {
  GET_KNOWLEDGE,
  GET_KNOWLEDGE_TAGS,
  CREATE_KNOWLEDGE,
  UPDATE_KNOWLEDGE,
  CREATE_KNOWLEDGE_TAG,
} from "@/lib/graphql/queries";

export default function KnowledgeEditPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNew = !id || id === "new";

  const [title, setTitle] = useState("");
  const [claim, setClaim] = useState("");
  const [category, setCategory] = useState("fact");
  const [selectedTagIds, setSelectedTagIds] = useState<string[]>([]);
  const [message, setMessage] = useState("");
  const [newTagName, setNewTagName] = useState("");

  const { data: tagsData, refetch: refetchTags } = useQuery(GET_KNOWLEDGE_TAGS);
  const tags = tagsData?.knowledgeTags || [];

  const { data: existingData } = useQuery(GET_KNOWLEDGE, {
    variables: { id },
    skip: isNew,
  });

  useEffect(() => {
    if (existingData?.knowledge) {
      const k = existingData.knowledge;
      setTitle(k.title);
      setClaim(k.claim);
      setCategory(k.category);
      setSelectedTagIds(k.tags.map((t: { id: string }) => t.id));
    }
  }, [existingData]);

  const [createKnowledge] = useMutation(CREATE_KNOWLEDGE);
  const [updateKnowledge] = useMutation(UPDATE_KNOWLEDGE);
  const [createTag] = useMutation(CREATE_KNOWLEDGE_TAG);

  const handleSave = async () => {
    if (!title || !claim || !message || selectedTagIds.length === 0) {
      alert("Title, claim, message, and at least one tag are required.");
      return;
    }

    if (isNew) {
      const { data } = await createKnowledge({
        variables: {
          input: { category, title, claim, tags: selectedTagIds, message },
        },
      });
      navigate(`/knowledge/${data.createKnowledge.id}`);
    } else {
      await updateKnowledge({
        variables: {
          input: { id, title, claim, tags: selectedTagIds, message },
        },
      });
      navigate(`/knowledge/${id}`);
    }
  };

  const handleCreateTag = async () => {
    if (!newTagName) return;
    await createTag({
      variables: { input: { name: newTagName.toLowerCase().trim() } },
    });
    setNewTagName("");
    refetchTags();
  };

  const toggleTag = (tagId: string) => {
    setSelectedTagIds((prev) =>
      prev.includes(tagId)
        ? prev.filter((id) => id !== tagId)
        : [...prev, tagId]
    );
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">
        {isNew ? "New Knowledge" : "Edit Knowledge"}
      </h1>

      {/* Category (only for new) */}
      {isNew && (
        <div>
          <label className="block text-sm font-medium text-gray-700">Category</label>
          <div className="mt-1 flex gap-2">
            {["fact", "technique"].map((cat) => (
              <button
                key={cat}
                onClick={() => setCategory(cat)}
                className={`rounded-md px-3 py-1 text-sm font-medium ${
                  category === cat
                    ? "bg-blue-600 text-white"
                    : "bg-gray-100 text-gray-700 hover:bg-gray-200"
                }`}
              >
                {cat}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Title */}
      <div>
        <label className="block text-sm font-medium text-gray-700">Title</label>
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
          placeholder="e.g., svchost.exe, BigQuery CloudTrail search"
        />
      </div>

      {/* Claim */}
      <div>
        <label className="block text-sm font-medium text-gray-700">Claim (Markdown)</label>
        <textarea
          value={claim}
          onChange={(e) => setClaim(e.target.value)}
          rows={12}
          className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono"
          placeholder="Write facts or techniques in Markdown format..."
        />
      </div>

      {/* Tags */}
      <div>
        <label className="block text-sm font-medium text-gray-700">Tags</label>
        <div className="mt-1 flex flex-wrap gap-2">
          {tags.map((tag: { id: string; name: string }) => (
            <button
              key={tag.id}
              onClick={() => toggleTag(tag.id)}
              className={`rounded-full px-3 py-1 text-xs font-medium ${
                selectedTagIds.includes(tag.id)
                  ? "bg-blue-100 text-blue-800 ring-1 ring-blue-600"
                  : "bg-gray-100 text-gray-600 hover:bg-gray-200"
              }`}
            >
              {tag.name}
            </button>
          ))}
        </div>
        <div className="mt-2 flex gap-2">
          <input
            type="text"
            value={newTagName}
            onChange={(e) => setNewTagName(e.target.value)}
            className="rounded-md border border-gray-300 px-3 py-1 text-sm"
            placeholder="New tag name"
          />
          <button
            onClick={handleCreateTag}
            className="rounded-md bg-gray-200 px-3 py-1 text-sm hover:bg-gray-300"
          >
            Create Tag
          </button>
        </div>
      </div>

      {/* Message */}
      <div>
        <label className="block text-sm font-medium text-gray-700">Change message</label>
        <input
          type="text"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
          placeholder="Reason for this change..."
        />
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <button
          onClick={handleSave}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          Save
        </button>
        <button
          onClick={() => navigate(-1)}
          className="rounded-md bg-gray-200 px-4 py-2 text-sm font-medium hover:bg-gray-300"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

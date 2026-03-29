import { useState } from "react";
import { useQuery } from "@apollo/client";
import {
  GET_KNOWLEDGES,
  GET_KNOWLEDGE_TAGS,
} from "@/lib/graphql/queries";
import { Link, useLocation, useNavigate } from "react-router-dom";

export default function KnowledgePage() {
  const location = useLocation();
  const navigate = useNavigate();
  const selectedCategory = location.pathname.endsWith("/technique") ? "technique" : "fact";
  const [selectedTagIds, setSelectedTagIds] = useState<string[]>([]);
  const [keyword, setKeyword] = useState("");

  const { data: tagsData } = useQuery(GET_KNOWLEDGE_TAGS);
  const tags = tagsData?.knowledgeTags || [];

  const { data, loading } = useQuery(GET_KNOWLEDGES, {
    variables: {
      category: selectedCategory,
      tags: selectedTagIds.length > 0 ? selectedTagIds : undefined,
      keyword: keyword || undefined,
    },
  });
  const knowledges = data?.knowledges || [];

  const toggleTag = (tagId: string) => {
    setSelectedTagIds((prev) =>
      prev.includes(tagId)
        ? prev.filter((id) => id !== tagId)
        : [...prev, tagId]
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Knowledge Base</h1>
        <Link
          to="/knowledge/new"
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          New Knowledge
        </Link>
      </div>

      {/* Category filter */}
      <div className="flex gap-2">
        {["fact", "technique"].map((cat) => (
          <button
            key={cat}
            onClick={() => navigate(`/knowledge/${cat}`)}
            className={`rounded-md px-3 py-1 text-sm font-medium ${
              selectedCategory === cat
                ? "bg-blue-600 text-white"
                : "bg-gray-100 text-gray-700 hover:bg-gray-200"
            }`}
          >
            {cat}
          </button>
        ))}
      </div>

      {/* Tag filter */}
      <div className="flex flex-wrap gap-2">
        {tags.map((tag: { id: string; name: string }) => (
          <button
            key={tag.id}
            data-testid="knowledge-tag-button"
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
        {tags.length === 0 && (
          <p className="text-sm text-gray-500">No tags yet. Create tags to organize knowledge.</p>
        )}
      </div>

      {/* Keyword search */}
      <input
        type="text"
        placeholder="Search by keyword..."
        value={keyword}
        onChange={(e) => setKeyword(e.target.value)}
        className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
      />

      {/* Results */}
      {loading && <p className="text-sm text-gray-500">Loading...</p>}
      {!loading && knowledges.length === 0 && (
        <p className="text-sm text-gray-500">No knowledge found.</p>
      )}

      <div className="border rounded-md divide-y">
        {knowledges.map((k: { id: string; title: string; claim: string; category: string; tags: { id: string; name: string }[]; updatedAt: string }) => (
          <Link
            key={k.id}
            to={`/knowledge/${k.id}`}
            data-testid="knowledge-item"
            className="flex items-center gap-3 px-4 py-3 hover:bg-muted/40 cursor-pointer transition-colors"
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="font-semibold text-[0.9rem] leading-snug truncate">
                  {k.title}
                </span>
                {k.tags.map((tag) => (
                  <span
                    key={tag.id}
                    className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 shrink-0"
                  >
                    {tag.name}
                  </span>
                ))}
              </div>
              <div className="text-xs text-muted-foreground mt-1">
                {new Date(k.updatedAt).toISOString().split("T")[0].replace(/-/g, "/")}
              </div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}

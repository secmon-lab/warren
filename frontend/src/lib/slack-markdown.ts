// Slack markdown to HTML converter utility

export interface SlackMarkdownOptions {
  allowHtml?: boolean;
  className?: {
    link?: string;
    bold?: string;
    italic?: string;
    code?: string;
    codeBlock?: string;
    strikethrough?: string;
  };
}

const defaultOptions: SlackMarkdownOptions = {
  allowHtml: false,
  className: {
    link: "text-blue-600 hover:text-blue-800 underline",
    bold: "font-semibold",
    italic: "italic",
    code: "bg-gray-100 px-1 py-0.5 rounded text-sm font-mono",
    codeBlock: "bg-gray-100 p-3 rounded-md overflow-x-auto",
    strikethrough: "line-through",
  },
};

/**
 * Convert Slack markdown to HTML
 */
export function slackToHtml(
  text: string,
  options: SlackMarkdownOptions = {}
): string {
  if (!text) return "";

  const opts = { ...defaultOptions, ...options };

  // Escape HTML to prevent XSS unless explicitly allowed
  let result = opts.allowHtml ? text : escapeHtml(text);

  // Convert Slack-specific markdown to HTML
  result = convertSlackLinks(result, opts.className?.link || "");
  result = convertUserMentions(result);
  result = convertChannelMentions(result);
  result = convertBold(result, opts.className?.bold || "");
  result = convertItalic(result, opts.className?.italic || "");
  result = convertCode(result, opts.className?.code || "");
  result = convertCodeBlocks(result, opts.className?.codeBlock || "");
  result = convertStrikethrough(result, opts.className?.strikethrough || "");
  result = convertLineBreaks(result);

  return result;
}

// Helper functions

function escapeHtml(text: string): string {
  if (typeof document !== "undefined") {
    const div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
  }
  // Fallback for server-side rendering
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

// Convert Slack link format <url|text> to HTML <a href="url">text</a>
function convertSlackLinks(text: string, className: string): string {
  // Pattern for <url|text> format
  const linkPattern = /<([^|>]+)\|([^>]+)>/g;
  text = text.replace(
    linkPattern,
    `<a href="$1" target="_blank" rel="noopener noreferrer" class="${className}">$2</a>`
  );

  // Pattern for plain URLs <url>
  const plainUrlPattern = /<(https?:\/\/[^>]+)>/g;
  text = text.replace(
    plainUrlPattern,
    `<a href="$1" target="_blank" rel="noopener noreferrer" class="${className}">$1</a>`
  );

  return text;
}

// Convert user mentions <@U123456> to styled mentions
function convertUserMentions(text: string): string {
  const userPattern = /<@([UW][A-Z0-9]+)(\|([^>]+))?>/g;
  return text.replace(userPattern, (match, userId, _, displayName) => {
    const name = displayName || userId;
    return `<span class="bg-blue-100 text-blue-800 px-1 py-0.5 rounded text-sm font-medium">@${name}</span>`;
  });
}

// Convert channel mentions <#C123456|general> to styled mentions
function convertChannelMentions(text: string): string {
  const channelPattern = /<#([C][A-Z0-9]+)(\|([^>]+))?>/g;
  return text.replace(channelPattern, (match, channelId, _, channelName) => {
    const name = channelName || channelId;
    return `<span class="bg-green-100 text-green-800 px-1 py-0.5 rounded text-sm font-medium">#${name}</span>`;
  });
}

// Convert *text* to <strong>text</strong>
function convertBold(text: string, className: string): string {
  const boldPattern = /\*([^*]+)\*/g;
  return text.replace(boldPattern, `<strong class="${className}">$1</strong>`);
}

// Convert _text_ to <em>text</em>
function convertItalic(text: string, className: string): string {
  const italicPattern = /_([^_]+)_/g;
  return text.replace(italicPattern, `<em class="${className}">$1</em>`);
}

// Convert `text` to <code>text</code>
function convertCode(text: string, className: string): string {
  const codePattern = /`([^`]+)`/g;
  return text.replace(codePattern, `<code class="${className}">$1</code>`);
}

// Convert ```text``` to <pre><code>text</code></pre>
function convertCodeBlocks(text: string, className: string): string {
  const codeBlockPattern = /```([^`]+)```/g;
  return text.replace(
    codeBlockPattern,
    `<pre class="${className}"><code class="text-sm font-mono">$1</code></pre>`
  );
}

// Convert ~text~ to <del>text</del>
function convertStrikethrough(text: string, className: string): string {
  const strikePattern = /~([^~]+)~/g;
  return text.replace(strikePattern, `<del class="${className}">$1</del>`);
}

// Convert \n to <br>
function convertLineBreaks(text: string): string {
  return text.replace(/\n/g, "<br>");
}

/**
 * Plain text converter that strips Slack markdown but preserves readability
 */
export function slackToPlainText(text: string): string {
  if (!text) return "";

  let result = text;

  // Remove Slack link formatting but keep the text
  result = result.replace(/<([^|>]+)\|([^>]+)>/g, "$2"); // <url|text> -> text
  result = result.replace(/<(https?:\/\/[^>]+)>/g, "$1"); // <url> -> url

  // Remove user/channel mentions formatting but keep names
  result = result.replace(
    /<@([UW][A-Z0-9]+)(\|([^>]+))?>/g,
    (match, userId, _, displayName) => {
      return `@${displayName || userId}`;
    }
  );
  result = result.replace(
    /<#([C][A-Z0-9]+)(\|([^>]+))?>/g,
    (match, channelId, _, channelName) => {
      return `#${channelName || channelId}`;
    }
  );

  // Remove markdown formatting
  result = result.replace(/\*([^*]+)\*/g, "$1"); // *bold* -> bold
  result = result.replace(/_([^_]+)_/g, "$1"); // _italic_ -> italic
  result = result.replace(/`([^`]+)`/g, "$1"); // `code` -> code
  result = result.replace(/```([^`]+)```/g, "$1"); // ```code``` -> code
  result = result.replace(/~([^~]+)~/g, "$1"); // ~strike~ -> strike

  return result;
}

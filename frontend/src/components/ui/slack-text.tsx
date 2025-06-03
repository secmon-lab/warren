import React from "react";
import { slackToHtml, SlackMarkdownOptions } from "@/lib/slack-markdown";
import { cn } from "@/lib/utils";

interface SlackTextProps {
  children: string;
  className?: string;
  options?: SlackMarkdownOptions;
}

/**
 * Component that renders Slack markdown as HTML
 */
export function SlackText({ children, className, options }: SlackTextProps) {
  if (!children) return null;

  const html = slackToHtml(children, options);

  return (
    <span
      className={cn("slack-text", className)}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

interface SlackTextBlockProps {
  children: string;
  className?: string;
  options?: SlackMarkdownOptions;
}

/**
 * Component that renders Slack markdown as a block element (div)
 */
export function SlackTextBlock({
  children,
  className,
  options,
}: SlackTextBlockProps) {
  if (!children) return null;

  const html = slackToHtml(children, options);

  return (
    <div
      className={cn("slack-text-block", className)}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

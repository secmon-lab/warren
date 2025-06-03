import React from "react";
import { render } from "@testing-library/react";
import { SlackText, SlackTextBlock } from "../slack-text";
import { SlackMarkdownOptions } from "@/lib/slack-markdown";

describe("SlackText", () => {
  it("should render nothing for empty content", () => {
    const { container } = render(<SlackText>{""}</SlackText>);
    expect(container.firstChild).toBeNull();
  });

  it("should render converted Slack markdown as HTML", () => {
    const { container } = render(
      <SlackText>{"This is *bold* text"}</SlackText>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain(
      '<strong class="font-semibold">bold</strong>'
    );
  });

  it("should apply custom className", () => {
    const { container } = render(
      <SlackText className="custom-class">{"test"}</SlackText>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.className).toContain("slack-text");
    expect(element.className).toContain("custom-class");
  });

  it("should use custom options", () => {
    const options: SlackMarkdownOptions = {
      className: { bold: "custom-bold" },
    };
    const { container } = render(
      <SlackText options={options}>{"*bold*"}</SlackText>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain(
      '<strong class="custom-bold">bold</strong>'
    );
  });

  it("should use children as key prop", () => {
    const { container, rerender } = render(
      <SlackText>{"original text"}</SlackText>
    );
    const firstElement = container.firstChild as HTMLElement;
    expect(firstElement).toBeTruthy();

    rerender(<SlackText>{"new text"}</SlackText>);
    const secondElement = container.firstChild as HTMLElement;
    expect(secondElement).toBeTruthy();

    // Different content should result in different elements being re-rendered
    expect(firstElement.innerHTML).not.toBe(secondElement.innerHTML);
  });

  it("should handle user mentions correctly", () => {
    const { container } = render(
      <SlackText>{"Hello <@U123456|john.doe>!"}</SlackText>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain("@john.doe");
    expect(element.innerHTML).toContain("bg-blue-100");
  });

  it("should handle links correctly", () => {
    const { container } = render(
      <SlackText>{"Visit <https://example.com|our site>"}</SlackText>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain('<a href="https://example.com"');
    expect(element.innerHTML).toContain("our site</a>");
    expect(element.innerHTML).toContain('target="_blank"');
    expect(element.innerHTML).toContain('rel="noopener noreferrer"');
  });

  it("should escape HTML by default", () => {
    const { container } = render(
      <SlackText>{"<script>alert('xss')</script>"}</SlackText>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain("&lt;script&gt;");
    expect(element.innerHTML).not.toContain("<script>");
  });
});

describe("SlackTextBlock", () => {
  it("should render nothing for empty content", () => {
    const { container } = render(<SlackTextBlock>{""}</SlackTextBlock>);
    expect(container.firstChild).toBeNull();
  });

  it("should render as div element", () => {
    const { container } = render(
      <SlackTextBlock>{"test content"}</SlackTextBlock>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.tagName).toBe("DIV");
  });

  it("should render converted Slack markdown as HTML", () => {
    const { container } = render(
      <SlackTextBlock>{"This is *bold* text"}</SlackTextBlock>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain(
      '<strong class="font-semibold">bold</strong>'
    );
  });

  it("should apply custom className", () => {
    const { container } = render(
      <SlackTextBlock className="custom-block-class">{"test"}</SlackTextBlock>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.className).toContain("slack-text-block");
    expect(element.className).toContain("custom-block-class");
  });

  it("should use custom options", () => {
    const options: SlackMarkdownOptions = {
      className: { italic: "custom-italic" },
    };
    const { container } = render(
      <SlackTextBlock options={options}>{"_italic_"}</SlackTextBlock>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain(
      '<em class="custom-italic">italic</em>'
    );
  });

  it("should use children as key prop", () => {
    const { container, rerender } = render(
      <SlackTextBlock>{"original content"}</SlackTextBlock>
    );
    const firstElement = container.firstChild as HTMLElement;
    expect(firstElement).toBeTruthy();

    rerender(<SlackTextBlock>{"new content"}</SlackTextBlock>);
    const secondElement = container.firstChild as HTMLElement;
    expect(secondElement).toBeTruthy();

    // Different content should result in different elements being re-rendered
    expect(firstElement.innerHTML).not.toBe(secondElement.innerHTML);
  });

  it("should handle line breaks correctly", () => {
    const { container } = render(
      <SlackTextBlock>{"Line 1\nLine 2\nLine 3"}</SlackTextBlock>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain("Line 1<br>Line 2<br>Line 3");
  });

  it("should handle code blocks correctly", () => {
    const { container } = render(
      <SlackTextBlock>
        {"```function test() { return true; }```"}
      </SlackTextBlock>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain(
      '<pre class="bg-gray-100 p-3 rounded-md overflow-x-auto">'
    );
    expect(element.innerHTML).toContain('<code class="text-sm font-mono">');
    expect(element.innerHTML).toContain("function test()");
  });

  it("should handle complex mixed content", () => {
    const complexContent = `*Alert:* Suspicious activity detected
_Source:_ <@U123456|security-bot>
Check <https://dashboard.example.com|Dashboard> for details`;

    const { container } = render(
      <SlackTextBlock>{complexContent}</SlackTextBlock>
    );
    const element = container.firstChild as HTMLElement;

    expect(element.innerHTML).toContain(
      '<strong class="font-semibold">Alert:</strong>'
    );
    expect(element.innerHTML).toContain('<em class="italic">Source:</em>');
    expect(element.innerHTML).toContain("@security-bot");
    expect(element.innerHTML).toContain(
      '<a href="https://dashboard.example.com"'
    );
    expect(element.innerHTML).toContain("Dashboard</a>");
    expect(element.innerHTML).toContain("<br>");
  });

  it("should handle channel mentions correctly", () => {
    const { container } = render(
      <SlackTextBlock>{"Posted in <#C123456|general>"}</SlackTextBlock>
    );
    const element = container.firstChild as HTMLElement;
    expect(element.innerHTML).toContain("#general");
    expect(element.innerHTML).toContain("bg-green-100");
  });

  it("should handle security concerns with XSS protection", () => {
    const maliciousContent = `*Safe* content with <script>alert('xss')</script> attempt`;
    const { container } = render(
      <SlackTextBlock>{maliciousContent}</SlackTextBlock>
    );
    const element = container.firstChild as HTMLElement;

    expect(element.innerHTML).toContain(
      '<strong class="font-semibold">Safe</strong>'
    );
    expect(element.innerHTML).toContain("&lt;script&gt;");
    expect(element.innerHTML).not.toContain("<script>alert");
  });
});

import {
  slackToHtml,
  slackToPlainText,
  SlackMarkdownOptions,
} from "../slack-markdown";

describe("slack-markdown", () => {
  describe("slackToHtml", () => {
    describe("basic functionality", () => {
      it("should handle empty and null/undefined input", () => {
        expect(slackToHtml("")).toBe("");
        // @ts-expect-error Testing null input
        expect(slackToHtml(null)).toBe("");
        // @ts-expect-error Testing undefined input
        expect(slackToHtml(undefined)).toBe("");
      });

      it("should escape HTML by default", () => {
        const result = slackToHtml('<script>alert("xss")</script>');
        expect(result).toContain("&lt;script&gt;");
        expect(result).not.toContain("<script>");
      });

      it("should allow HTML when explicitly enabled", () => {
        const options: SlackMarkdownOptions = { allowHtml: true };
        const result = slackToHtml("<span>test</span>", options);
        expect(result).toContain("<span>test</span>");
      });
    });

    describe("bold text conversion", () => {
      it("should convert *text* to bold", () => {
        const result = slackToHtml("This is *bold* text");
        expect(result).toContain('<strong class="font-semibold">bold</strong>');
      });

      it("should handle multiple bold sections", () => {
        const result = slackToHtml("*First* and *second* bold");
        expect(result).toContain(
          '<strong class="font-semibold">First</strong>'
        );
        expect(result).toContain(
          '<strong class="font-semibold">second</strong>'
        );
      });

      it("should not convert single asterisk", () => {
        const result = slackToHtml("Just * one asterisk");
        expect(result).not.toContain("<strong>");
        expect(result).toContain("Just * one asterisk");
      });

      it("should use custom bold class when provided", () => {
        const options: SlackMarkdownOptions = {
          className: { bold: "custom-bold" },
        };
        const result = slackToHtml("*bold*", options);
        expect(result).toContain('<strong class="custom-bold">bold</strong>');
      });
    });

    describe("italic text conversion", () => {
      it("should convert _text_ to italic", () => {
        const result = slackToHtml("This is _italic_ text");
        expect(result).toContain('<em class="italic">italic</em>');
      });

      it("should handle multiple italic sections", () => {
        const result = slackToHtml("_First_ and _second_ italic");
        expect(result).toContain('<em class="italic">First</em>');
        expect(result).toContain('<em class="italic">second</em>');
      });

      it("should not convert single underscore", () => {
        const result = slackToHtml("Just _ one underscore");
        expect(result).not.toContain("<em>");
        expect(result).toContain("Just _ one underscore");
      });

      it("should use custom italic class when provided", () => {
        const options: SlackMarkdownOptions = {
          className: { italic: "custom-italic" },
        };
        const result = slackToHtml("_italic_", options);
        expect(result).toContain('<em class="custom-italic">italic</em>');
      });
    });

    describe("code conversion", () => {
      it("should convert `code` to inline code", () => {
        const result = slackToHtml("This is `code` text");
        expect(result).toContain(
          '<code class="bg-gray-100 px-1 py-0.5 rounded text-sm font-mono">code</code>'
        );
      });

      it("should handle multiple code sections", () => {
        const result = slackToHtml("`first` and `second` code");
        expect(result).toContain(
          '<code class="bg-gray-100 px-1 py-0.5 rounded text-sm font-mono">first</code>'
        );
        expect(result).toContain(
          '<code class="bg-gray-100 px-1 py-0.5 rounded text-sm font-mono">second</code>'
        );
      });

      it("should use custom code class when provided", () => {
        const options: SlackMarkdownOptions = {
          className: { code: "custom-code" },
        };
        const result = slackToHtml("`code`", options);
        expect(result).toContain('<code class="custom-code">code</code>');
      });
    });

    describe("code block conversion", () => {
      it("should convert ```code``` to code blocks", () => {
        const result = slackToHtml(
          "```\nfunction test() {\n  return true;\n}\n```"
        );
        expect(result).toContain(
          '<pre class="bg-gray-100 p-3 rounded-md overflow-x-auto"><code class="text-sm font-mono">'
        );
        expect(result).toContain("function test()");
      });

      it("should use custom code block class when provided", () => {
        const options: SlackMarkdownOptions = {
          className: { codeBlock: "custom-code-block" },
        };
        const result = slackToHtml("```code```", options);
        expect(result).toContain('<pre class="custom-code-block">');
      });
    });

    describe("strikethrough conversion", () => {
      it("should convert ~text~ to strikethrough", () => {
        const result = slackToHtml("This is ~deleted~ text");
        expect(result).toContain('<del class="line-through">deleted</del>');
      });

      it("should use custom strikethrough class when provided", () => {
        const options: SlackMarkdownOptions = {
          className: { strikethrough: "custom-strike" },
        };
        const result = slackToHtml("~strike~", options);
        expect(result).toContain('<del class="custom-strike">strike</del>');
      });
    });

    describe("Slack link conversion", () => {
      it("should convert <url|text> format to HTML links", () => {
        const result = slackToHtml("<https://example.com|Example Site>");
        expect(result).toContain(
          '<a href="https://example.com" target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:text-blue-800 underline">Example Site</a>'
        );
      });

      it("should convert plain <url> format to HTML links", () => {
        const result = slackToHtml("<https://example.com>");
        expect(result).toContain(
          '<a href="https://example.com" target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:text-blue-800 underline">https://example.com</a>'
        );
      });

      it("should use custom link class when provided", () => {
        const options: SlackMarkdownOptions = {
          className: { link: "custom-link" },
        };
        const result = slackToHtml("<https://example.com|Link>", options);
        expect(result).toContain(
          '<a href="https://example.com" target="_blank" rel="noopener noreferrer" class="custom-link">Link</a>'
        );
      });
    });

    describe("user mention conversion", () => {
      it("should convert <@U123456> to styled mentions", () => {
        const result = slackToHtml("<@U123456>");
        expect(result).toContain(
          '<span class="bg-blue-100 text-blue-800 px-1 py-0.5 rounded text-sm font-medium">@U123456</span>'
        );
      });

      it("should convert <@U123456|username> to styled mentions with display name", () => {
        const result = slackToHtml("<@U123456|john.doe>");
        expect(result).toContain(
          '<span class="bg-blue-100 text-blue-800 px-1 py-0.5 rounded text-sm font-medium">@john.doe</span>'
        );
      });

      it("should handle workflow mentions <@W123456>", () => {
        const result = slackToHtml("<@W123456>");
        expect(result).toContain(
          '<span class="bg-blue-100 text-blue-800 px-1 py-0.5 rounded text-sm font-medium">@W123456</span>'
        );
      });
    });

    describe("channel mention conversion", () => {
      it("should convert <#C123456> to styled mentions", () => {
        const result = slackToHtml("<#C123456>");
        expect(result).toContain(
          '<span class="bg-green-100 text-green-800 px-1 py-0.5 rounded text-sm font-medium">#C123456</span>'
        );
      });

      it("should convert <#C123456|general> to styled mentions with channel name", () => {
        const result = slackToHtml("<#C123456|general>");
        expect(result).toContain(
          '<span class="bg-green-100 text-green-800 px-1 py-0.5 rounded text-sm font-medium">#general</span>'
        );
      });
    });

    describe("line break conversion", () => {
      it("should convert \\n to <br> tags", () => {
        const result = slackToHtml("Line 1\nLine 2\nLine 3");
        expect(result).toBe("Line 1<br>Line 2<br>Line 3");
      });

      it("should handle multiple consecutive line breaks", () => {
        const result = slackToHtml("Line 1\n\n\nLine 2");
        expect(result).toBe("Line 1<br><br><br>Line 2");
      });
    });

    describe("mixed markdown", () => {
      it("should handle combination of different markdown types", () => {
        const text =
          "*Bold* and _italic_ with `code` and <https://example.com|link>";
        const result = slackToHtml(text);

        expect(result).toContain('<strong class="font-semibold">Bold</strong>');
        expect(result).toContain('<em class="italic">italic</em>');
        expect(result).toContain(
          '<code class="bg-gray-100 px-1 py-0.5 rounded text-sm font-mono">code</code>'
        );
        expect(result).toContain(
          '<a href="https://example.com" target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:text-blue-800 underline">link</a>'
        );
      });

      it("should handle nested markdown correctly", () => {
        const text = "*Bold with _italic_ inside*";
        const result = slackToHtml(text);
        // Should convert both bold and italic
        expect(result).toContain(
          '<strong class="font-semibold">Bold with <em class="italic">italic</em> inside</strong>'
        );
      });

      it("should handle complex real-world example", () => {
        const text = `*Alert:* Suspicious activity detected
_Source:_ <@U123456|security-bot>
_Details:_ Multiple failed login attempts from \`192.168.1.100\`
_Action Required:_ Please review <https://dashboard.example.com/alerts/123|Alert Dashboard>`;

        const result = slackToHtml(text);

        expect(result).toContain(
          '<strong class="font-semibold">Alert:</strong>'
        );
        expect(result).toContain('<em class="italic">Source:</em>');
        expect(result).toContain("@security-bot");
        expect(result).toContain(
          '<code class="bg-gray-100 px-1 py-0.5 rounded text-sm font-mono">192.168.1.100</code>'
        );
        expect(result).toContain(
          '<a href="https://dashboard.example.com/alerts/123"'
        );
      });
    });

    describe("edge cases", () => {
      it("should handle malformed markdown gracefully", () => {
        const cases = [
          "*unclosed bold",
          "_unclosed italic",
          "`unclosed code",
          "```unclosed code block",
          "~unclosed strike",
          "<malformed|link",
          "<@malformed user",
          "<#malformed channel",
        ];

        cases.forEach((testCase) => {
          expect(() => slackToHtml(testCase)).not.toThrow();
        });
      });

      it("should handle special characters", () => {
        const result = slackToHtml("Test with & < > \" ' characters");
        expect(result).toContain("&amp;");
        expect(result).toContain("&lt;");
        expect(result).toContain("&gt;");
        // In browser environment, quotes might not be escaped the same way
        expect(result).toMatch(/[\"']|&quot;|&#39;/);
      });

      it("should handle very long text", () => {
        const longText = "a".repeat(10000);
        const result = slackToHtml(longText);
        expect(result).toBe(longText);
      });

      it("should handle empty markdown patterns", () => {
        const cases = ["**", "__", "``", "~~~"];

        cases.forEach((testCase) => {
          const result = slackToHtml(testCase);
          expect(result).toBe(testCase);
        });

        // These patterns involve angle brackets so will be escaped
        const escapedCases = ["<>", "<@>", "<#>"];
        escapedCases.forEach((testCase) => {
          const result = slackToHtml(testCase);
          expect(result).toContain("&lt;");
          expect(result).toContain("&gt;");
        });
      });
    });
  });

  describe("slackToPlainText", () => {
    it("should handle empty and null/undefined input", () => {
      expect(slackToPlainText("")).toBe("");
      // @ts-expect-error Testing null input
      expect(slackToPlainText(null)).toBe("");
      // @ts-expect-error Testing undefined input
      expect(slackToPlainText(undefined)).toBe("");
    });

    it("should remove bold formatting but keep text", () => {
      const result = slackToPlainText("This is *bold* text");
      expect(result).toBe("This is bold text");
    });

    it("should remove italic formatting but keep text", () => {
      const result = slackToPlainText("This is _italic_ text");
      expect(result).toBe("This is italic text");
    });

    it("should remove code formatting but keep text", () => {
      const result = slackToPlainText("This is `code` text");
      expect(result).toBe("This is code text");
    });

    it("should remove code block formatting but keep text", () => {
      const result = slackToPlainText("```function test() { return true; }```");
      expect(result).toBe("function test() { return true; }");
    });

    it("should remove strikethrough formatting but keep text", () => {
      const result = slackToPlainText("This is ~deleted~ text");
      expect(result).toBe("This is deleted text");
    });

    it("should convert Slack links to plain text", () => {
      const result = slackToPlainText("<https://example.com|Example Site>");
      expect(result).toBe("Example Site");
    });

    it("should convert plain URLs to just the URL", () => {
      const result = slackToPlainText("<https://example.com>");
      expect(result).toBe("https://example.com");
    });

    it("should convert user mentions to readable format", () => {
      const result = slackToPlainText("<@U123456|john.doe>");
      expect(result).toBe("@john.doe");

      const result2 = slackToPlainText("<@U123456>");
      expect(result2).toBe("@U123456");
    });

    it("should convert channel mentions to readable format", () => {
      const result = slackToPlainText("<#C123456|general>");
      expect(result).toBe("#general");

      const result2 = slackToPlainText("<#C123456>");
      expect(result2).toBe("#C123456");
    });

    it("should handle complex mixed markdown", () => {
      const text = `*Alert:* Suspicious activity detected
_Source:_ <@U123456|security-bot>
_Details:_ Multiple failed login attempts from \`192.168.1.100\`
_Action Required:_ Please review <https://dashboard.example.com/alerts/123|Alert Dashboard>`;

      const result = slackToPlainText(text);

      expect(result).toContain("Alert: Suspicious activity detected");
      expect(result).toContain("Source: @security-bot");
      expect(result).toContain(
        "Details: Multiple failed login attempts from 192.168.1.100"
      );
      expect(result).toContain(
        "Action Required: Please review Alert Dashboard"
      );
      expect(result).not.toContain("*");
      expect(result).not.toContain("_");
      expect(result).not.toContain("`");
      expect(result).not.toContain("<");
      expect(result).not.toContain(">");
    });

    it("should preserve line breaks and spacing", () => {
      const text = `Line 1\nLine 2\n\nLine 3`;
      const result = slackToPlainText(text);
      expect(result).toBe("Line 1\nLine 2\n\nLine 3");
    });

    it("should handle edge cases gracefully", () => {
      const cases = [
        "*unclosed bold",
        "_unclosed italic",
        "`unclosed code",
        "```unclosed code block",
        "~unclosed strike",
        "<malformed|link",
        "<@malformed user",
        "<#malformed channel",
      ];

      cases.forEach((testCase) => {
        expect(() => slackToPlainText(testCase)).not.toThrow();
      });
    });
  });
});

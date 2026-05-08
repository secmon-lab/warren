package llm

// ValidateForTest exposes the internal validate() for testing.
func ValidateForTest(f *File) error { return validate(f) }

// RenderTemplateForTest exposes renderTemplate().
func RenderTemplateForTest(raw []byte) ([]byte, error) { return renderTemplate(raw) }

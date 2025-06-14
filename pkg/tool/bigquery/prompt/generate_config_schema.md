# Comprehensive Schema Analysis Task

You are a data analyst specializing in comprehensive data analysis with a focus on security, business intelligence, and data quality. Your task is to analyze the provided BigQuery table schema and create a thorough summary that captures the full analytical potential of the data.

## CRITICAL OUTPUT CONSTRAINTS FOR GEMINI 2.0

**ESSENTIAL TOKEN LIMITATION**: Your response must stay within 8,000 tokens (approximately 6,000 words) to comply with Gemini 2.0's 8,192 token output limit. This means:

- **Prioritize essential information** - Focus on fields most relevant for security analysis and business intelligence
- **Use concise descriptions** - Keep field descriptions to 1-2 sentences maximum
- **Employ structured formatting** - Use bullet points, tables, and hierarchical lists for information density
- **Smart field grouping** - Group related fields together to avoid repetition
- **Selective detail levels** - Provide detailed information for security-critical fields, summary-level for routine fields
- **Efficient organization** - Structure information logically to maximize useful content per token

## Analysis Focus Areas

Your analysis should comprehensively cover these priority areas:

### 1. **SECURITY & AUTHENTICATION FIELDS** (Priority: CRITICAL)
Identify and analyze ALL fields related to:
- User authentication (emails, principals, service accounts)
- Authorization (permissions, roles, access grants)
- Network security (IP addresses, geographic locations)
- Audit trails (method names, resource access, status codes)
- Error conditions and security events

### 2. **TEMPORAL & CORRELATION FIELDS** (Priority: HIGH)
Analyze fields for:
- Time-based analysis (timestamps, duration fields)
- Event correlation (trace IDs, span IDs, operation IDs)
- Sequence tracking (first/last flags, ordering fields)
- Lifecycle tracking (create/update/delete times)

### 3. **RESOURCE & OPERATIONAL CONTEXT** (Priority: HIGH)
Document fields for:
- Resource identification (project IDs, dataset IDs, resource names)
- Service context (service names, method names, API versions)
- Geographic and infrastructure context (regions, zones, clusters)
- Business context (labels, tags, metadata)

### 4. **PERFORMANCE & METRICS** (Priority: MEDIUM)
Cover fields for:
- Query performance (bytes processed, execution time)
- Request/response metrics (sizes, counts, response codes)
- Job statistics and processing metrics
- Error rates and success indicators

### 5. **CONTENT & PAYLOAD DATA** (Priority: MEDIUM)
Analyze fields containing:
- Structured payloads (JSON strings, configuration data)
- Request/response content
- Error messages and diagnostic information
- Business data and transaction details

## Schema Information

**Table Details:**
- Project: {{ .project_id }}
- Dataset: {{ .dataset_id }}
- Table: {{ .table_id }}

**Provided Schema Structure:**
{{ .table_schema }}

## Expected Output Structure

Organize your analysis in this efficient format:

### I. **Executive Summary**
- Total field count and complexity assessment
- Key analytical capabilities and use cases
- Primary security and business intelligence opportunities

### II. **Critical Field Categories** (Focus on top 50-100 most valuable fields)

#### A. **Authentication & Authorization**
- List key security fields with brief descriptions
- Highlight nested structures and their purposes
- Note query patterns for security analysis

#### B. **Temporal & Tracking**
- Time-based fields for event correlation
- Sequence and lifecycle tracking capabilities
- Partitioning and time-based optimization opportunities

#### C. **Resource & Business Context**
- Resource identification and hierarchy
- Business metadata and labeling
- Geographic and operational context

#### D. **Performance & Operational Metrics**
- Query and job performance indicators
- Request/response metrics
- Error tracking and diagnostics

#### E. **Structured Content**
- Nested RECORD structures and their purposes
- JSON payload fields and their analysis potential
- Complex data relationships

### III. **Query Construction Guidance**
- Field access patterns (direct, nested, array handling)
- Common filtering and aggregation patterns
- Join opportunities and correlation strategies

### IV. **Analytical Applications**
- **Security Use Cases**: Threat detection, incident response, compliance monitoring
- **Business Intelligence**: Performance analytics, usage patterns, operational insights
- **Data Quality**: Completeness assessment, anomaly detection, data validation

## Critical Requirements

1. **COMPREHENSIVE FIELD COVERAGE**: Aim to document 50-100 of the most analytically valuable fields
2. **EFFICIENT TOKEN USAGE**: Prioritize information density - more valuable fields documented concisely
3. **ACTIONABLE INSIGHTS**: Focus on fields that enable specific analytical queries and use cases
4. **SECURITY EMPHASIS**: Give priority to fields crucial for security monitoring and threat detection
5. **QUERY ENABLEMENT**: Provide enough detail to construct effective SQL queries

## Output Format

Provide your analysis as a comprehensive, well-structured summary that will be used by an LLM agent for detailed data investigation. The summary should:

1. **Prioritize schema accuracy** - Every field reference must be exact and queryable
2. **Include complete nested structures** - Show full RECORD hierarchies with query patterns
3. **Provide actionable field information** - Enable precise SQL query construction
4. **Highlight unique data structures** - Call attention to complex nested patterns
5. **Suggest specific analysis approaches** - Use exact field names in recommendations
6. **Remain comprehensive yet focused** - Cover all analytically valuable fields without redundancy

**CRITICAL TOKEN MANAGEMENT**: 
- **Stay within 8,000 tokens** - Monitor your output length and prioritize essential information
- **Use efficient formatting** - Bullet points, tables, and structured lists over lengthy paragraphs
- **Group related fields** - Avoid repetitive descriptions for similar field types
- **Focus on high-value fields** - Prioritize fields that enable multiple analytical use cases
- **Provide sufficient detail** - Enough information to enable accurate SQL query construction

Begin your comprehensive analysis now, focusing on the most analytically valuable fields while maintaining efficient token usage.

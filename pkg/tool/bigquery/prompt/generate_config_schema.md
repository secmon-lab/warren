# Comprehensive Schema Analysis Task

You are a data analyst specializing in comprehensive data analysis with a focus on security, business intelligence, and data quality. Your task is to analyze the provided BigQuery table schema and create a thorough summary that captures the full analytical potential of the data.

## Table Information

- ProjectID: {{ .project_id }}
- DatasetID: {{ .dataset_id }}
- TableID: {{ .table_id }}

{{ .table_description }}

## Schema Data

The following is the flattened schema of the BigQuery table:

{{ .table_schema }}

## Comprehensive Analysis Requirements

Please analyze the schema systematically and provide a detailed summary that includes:

### 1. **Complete Field Inventory**
Categorize ALL fields into the following comprehensive categories:

#### **Identity & Authentication Fields**
- User identifiers (IDs, usernames, email addresses, employee IDs)
- Device/endpoint identifiers (device IDs, MAC addresses, hardware IDs)
- Account identifiers (account names, tenant IDs, organization IDs)
- Authentication data (login methods, tokens, certificates, session IDs)
- Authorization data (roles, permissions, access levels, group memberships)

#### **Network & Communication Fields**
- IP addresses (source, destination, internal, external)
- Hostnames, domains, subdomains, FQDNs
- Network infrastructure (ports, protocols, VLANs, subnets)
- Communication metadata (bytes transferred, packet counts, connection states)
- DNS data (queries, responses, resolution times)
- URL components (paths, parameters, fragments, referrers)

#### **Temporal & Time-based Fields**
- Primary timestamps (creation, modification, access times)
- Duration fields (session length, response times, processing times)
- Date components (year, month, day, hour, timezone)
- Sequence numbers, counters, version numbers
- Time intervals, ranges, and periods

#### **Event & Activity Fields**
- Event types, categories, classifications
- Action codes, operation types, method names
- Status indicators (success/failure, error codes, response codes)
- Severity levels, priority rankings, alert levels
- Event sources, origins, generators

#### **Content & Data Fields**
- Text content (messages, descriptions, comments, logs)
- File information (names, paths, extensions, sizes, hashes)
- Data payloads (request/response bodies, JSON blobs, binary data)
- Configuration values, settings, parameters
- Error messages, exceptions, stack traces

#### **Geographic & Location Fields**
- Country/region codes, city names, postal codes
- Coordinates (latitude, longitude, elevation)
- Location hierarchies (continent, country, state, city)
- Timezone information, locale settings
- Physical location identifiers (building, floor, room)

#### **Business & Operational Fields**
- Transaction data (amounts, currencies, payment methods)
- Product/service identifiers (SKUs, product names, categories)
- Customer data (customer IDs, demographics, preferences)
- Organizational data (departments, teams, cost centers)
- Workflow states (status, stage, phase, approval levels)

#### **Technical & System Fields**
- System identifiers (hostnames, instance IDs, container IDs)
- Software information (versions, build numbers, components)
- Performance metrics (CPU, memory, disk usage, latencies)
- Configuration data (settings, flags, environment variables)
- Debug information (trace IDs, correlation IDs, debug flags)

#### **Data Quality & Metadata Fields**
- Source system identifiers, data lineage information
- Data validation flags, quality scores, confidence levels
- Processing timestamps, ingestion metadata
- Data classification labels, sensitivity markers
- Audit trail information, change tracking data

### 2. **Data Type Analysis**
For each data type present in the schema:
- **String fields**: Identify potential patterns (IDs, codes, free text, structured data)
- **Numeric fields**: Distinguish between IDs, counters, measurements, monetary values
- **Boolean fields**: Identify flags, status indicators, configuration switches
- **Timestamp fields**: Analyze temporal patterns and relationships
- **Array fields**: Understand list structures and cardinality
- **Record fields**: Map nested structures and hierarchical relationships

### 3. **Structural Complexity Assessment**
- **Total field count**: Count all fields including nested ones
- **Nesting depth**: Identify maximum levels of nested structures
- **Array relationships**: Document repeated/array fields and their significance
- **Complex types**: Highlight RECORD, STRUCT, and nested array types
- **Normalization level**: Assess data normalization and potential denormalization
- **Key relationships**: Identify potential primary keys, foreign keys, and composite keys

### 4. **Analytical Potential Assessment**

#### **Security Analysis Capabilities**
- Threat detection scenarios this data enables
- Incident response investigation support
- Compliance monitoring possibilities
- Risk assessment data points
- Behavioral analysis opportunities

#### **Business Intelligence Opportunities**
- KPI calculation possibilities
- Trend analysis potential
- Customer behavior insights
- Operational efficiency metrics
- Revenue/cost analysis capabilities

#### **Data Science Applications**
- Machine learning feature candidates
- Anomaly detection possibilities
- Predictive modeling opportunities
- Classification/clustering potential
- Time series analysis capabilities

#### **Correlation & Join Opportunities**
- Internal correlation possibilities (within table)
- External join candidates (with other tables)
- Time-based correlation patterns
- Geographic correlation possibilities
- Hierarchical relationship mapping

### 5. **Data Quality Considerations**
- Fields likely to have missing/null values
- Potential data consistency issues
- Fields requiring validation or cleansing
- Duplicate detection opportunities
- Data freshness indicators

### 6. **Query Optimization Insights**
- High-cardinality fields suitable for filtering
- Low-cardinality fields suitable for grouping
- Time-partitioning candidates
- Clustering column suggestions
- Index optimization opportunities

## Output Format

Provide your analysis as a comprehensive, well-structured summary that will be used by an LLM agent for detailed data investigation. The summary should:

1. Be thorough enough to capture the full analytical potential of the table
2. Prioritize fields and categories by their analytical value
3. Highlight unique or interesting data structures
4. Suggest specific analysis approaches based on the available data
5. Remain concise enough for efficient processing in subsequent analysis phases

Focus on actionable insights that will enable comprehensive data exploration, analysis, and investigation across security, business, and operational use cases.

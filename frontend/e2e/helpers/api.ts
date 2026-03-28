import { type Page } from "@playwright/test";

interface GraphQLResponse<T = unknown> {
  data?: T;
  errors?: Array<{ message: string }>;
}

export async function executeGraphQL<T = unknown>(
  page: Page,
  query: string,
  variables?: Record<string, unknown>
): Promise<GraphQLResponse<T>> {
  const baseURL = page.url().startsWith("http")
    ? new URL(page.url()).origin
    : "http://localhost:8080";

  const response = await page.request.post(`${baseURL}/graphql`, {
    data: { query, variables },
    headers: { "Content-Type": "application/json" },
  });

  return response.json();
}

export async function createTicketViaAPI(
  page: Page,
  title: string,
  description: string,
  isTest: boolean = true
): Promise<{ id: string; title: string; status: string }> {
  const result = await executeGraphQL<{
    createTicket: { id: string; title: string; status: string };
  }>(
    page,
    `mutation CreateTicket($title: String!, $description: String!, $isTest: Boolean) {
      createTicket(title: $title, description: $description, isTest: $isTest) {
        id
        status
        title
      }
    }`,
    { title, description, isTest }
  );

  if (result.errors) {
    throw new Error(
      `GraphQL error creating ticket: ${result.errors[0].message}`
    );
  }

  return result.data!.createTicket;
}

export async function resolveTicketViaAPI(
  page: Page,
  ticketId: string,
  conclusion: string = "false_positive",
  reason: string = "E2E test resolution"
): Promise<void> {
  const result = await executeGraphQL(
    page,
    `mutation ResolveTicket($id: ID!, $conclusion: String!, $reason: String!) {
      resolveTicket(id: $id, conclusion: $conclusion, reason: $reason) {
        id
        status
      }
    }`,
    { id: ticketId, conclusion, reason }
  );

  if (result.errors) {
    throw new Error(
      `GraphQL error resolving ticket: ${result.errors[0].message}`
    );
  }
}

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
  let baseURL: string;
  if (page.url().startsWith("http")) {
    baseURL = new URL(page.url()).origin;
  } else if (process.env.BASE_URL) {
    baseURL = process.env.BASE_URL;
  } else {
    throw new Error(
      "executeGraphQL: cannot resolve baseURL — page has no http origin yet " +
        "and BASE_URL is not set. Make sure the test navigated first, or run " +
        "via ./frontend/scripts/e2e.sh which exports BASE_URL."
    );
  }

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

  if (result.errors && result.errors.length > 0) {
    const errorMessages = result.errors.map((e) => e.message).join(", ");
    throw new Error(`GraphQL error creating ticket: ${errorMessages}`);
  }
  if (!result.data?.createTicket) {
    throw new Error(
      "GraphQL error creating ticket: response did not include ticket data"
    );
  }

  return result.data.createTicket;
}

export async function archiveTicketViaAPI(
  page: Page,
  ticketId: string
): Promise<void> {
  const result = await executeGraphQL(
    page,
    `mutation ArchiveTicket($id: ID!) {
      archiveTicket(id: $id) {
        id
      }
    }`,
    { id: ticketId }
  );
  if (result.errors && result.errors.length > 0) {
    const errorMessages = result.errors.map((e) => e.message).join(", ");
    throw new Error(`GraphQL error archiving ticket: ${errorMessages}`);
  }
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

  if (result.errors && result.errors.length > 0) {
    const errorMessages = result.errors.map((e) => e.message).join(", ");
    throw new Error(`GraphQL error resolving ticket: ${errorMessages}`);
  }
}

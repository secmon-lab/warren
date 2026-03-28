export interface TestTicket {
  title: string;
  description: string;
  isTest?: boolean;
}

export const testTickets: TestTicket[] = [
  {
    title: "E2E Test Ticket 1",
    description: "First test ticket for e2e testing",
    isTest: true,
  },
  {
    title: "E2E Test Ticket 2",
    description: "Second test ticket for e2e testing",
    isTest: true,
  },
  {
    title: "E2E Test Ticket 3",
    description: "Third test ticket for e2e testing",
    isTest: true,
  },
];

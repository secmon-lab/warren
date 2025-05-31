import { gql } from '@apollo/client';

export const GET_TICKETS = gql`
  query GetTickets($statuses: [String!], $offset: Int, $limit: Int) {
    tickets(statuses: $statuses, offset: $offset, limit: $limit) {
      id
      status
      createdAt
      updatedAt
      alerts {
        id
        title
        createdAt
      }
      comments {
        id
        content
        createdAt
      }
    }
  }
`;

export const GET_TICKET = gql`
  query GetTicket($id: ID!) {
    ticket(id: $id) {
      id
      status
      createdAt
      updatedAt
      alerts {
        id
        title
        description
        createdAt
      }
      comments {
        id
        content
        createdAt
        updatedAt
      }
    }
  }
`;

export const GET_ALERT = gql`
  query GetAlert($id: ID!) {
    alert(id: $id) {
      id
      title
      description
      createdAt
      ticket {
        id
        status
      }
    }
  }
`;

export const GET_ALERTS = gql`
  query GetAlerts {
    alerts {
      id
      title
      description
      createdAt
      ticket {
        id
        status
      }
    }
  }
`;

export const UPDATE_TICKET_STATUS = gql`
  mutation UpdateTicketStatus($id: ID!, $status: String!) {
    updateTicketStatus(id: $id, status: $status) {
      id
      status
      updatedAt
    }
  }
`; 
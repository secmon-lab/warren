import { gql } from "@apollo/client";

export const GET_TICKETS = gql`
  query GetTickets($statuses: [String!], $offset: Int, $limit: Int) {
    tickets(statuses: $statuses, offset: $offset, limit: $limit) {
      id
      status
      title
      description
      conclusion
      assignee {
        id
        name
      }
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
      title
      description
      summary
      conclusion
      reason
      finding {
        severity
        summary
        reason
        recommendation
      }
      assignee {
        id
        name
      }
      createdAt
      updatedAt
      alerts {
        id
        title
        description
        schema
        data
        attributes {
          key
          value
          link
          auto
        }
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
      schema
      data
      attributes {
        key
        value
        link
        auto
      }
      createdAt
      ticket {
        id
        status
        title
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
      schema
      data
      attributes {
        key
        value
        link
        auto
      }
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

export const UPDATE_MULTIPLE_TICKETS_STATUS = gql`
  mutation UpdateMultipleTicketsStatus($ids: [ID!]!, $status: String!) {
    updateMultipleTicketsStatus(ids: $ids, status: $status) {
      id
      status
      updatedAt
    }
  }
`;

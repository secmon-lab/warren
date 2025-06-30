import { gql } from "@apollo/client";

export const GET_TICKETS = gql`
  query GetTickets($statuses: [String!], $offset: Int, $limit: Int) {
    tickets(statuses: $statuses, offset: $offset, limit: $limit) {
      tickets {
        id
        status
        title
        description
        conclusion
        reason
        isTest
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
          user {
            id
            name
          }
          createdAt
        }
      }
      totalCount
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
      isTest
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
      slackLink
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
        user {
          id
          name
        }
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
        description
      }
    }
  }
`;

export const GET_ALERTS = gql`
  query GetAlerts($offset: Int, $limit: Int) {
    alerts(offset: $offset, limit: $limit) {
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
          title
          description
        }
      }
      totalCount
    }
  }
`;

export const GET_DASHBOARD = gql`
  query GetDashboard {
    dashboard {
      openTicketsCount
      unboundAlertsCount
      openTickets {
        id
        status
        title
        description
        isTest
        assignee {
          id
          name
        }
        createdAt
        updatedAt
      }
      unboundAlerts {
        id
        title
        description
        createdAt
      }
    }
  }
`;

export const GET_ACTIVITIES = gql`
  query GetActivities($offset: Int, $limit: Int) {
    activities(offset: $offset, limit: $limit) {
      activities {
        id
        type
        userID
        alertID
        ticketID
        commentID
        createdAt
        metadata
        user {
          id
          name
        }
        alert {
          id
          title
          description
        }
        ticket {
          id
          title
          description
        }
      }
      totalCount
    }
  }
`;

export const GET_TICKET_COMMENTS = gql`
  query GetTicketComments($ticketId: ID!, $offset: Int, $limit: Int) {
    ticketComments(ticketId: $ticketId, offset: $offset, limit: $limit) {
      comments {
        id
        content
        user {
          id
          name
        }
        createdAt
        updatedAt
      }
      totalCount
    }
  }
`;

export const UPDATE_TICKET_STATUS = gql`
  mutation UpdateTicketStatus($id: ID!, $status: String!) {
    updateTicketStatus(id: $id, status: $status) {
      id
      status
      title
      description
      isTest
      assignee {
        id
        name
      }
      createdAt
      updatedAt
    }
  }
`;

export const UPDATE_MULTIPLE_TICKETS_STATUS = gql`
  mutation UpdateMultipleTicketsStatus($ids: [ID!]!, $status: String!) {
    updateMultipleTicketsStatus(ids: $ids, status: $status) {
      id
      status
      title
      description
      isTest
      assignee {
        id
        name
      }
      createdAt
      updatedAt
    }
  }
`;

export const UPDATE_TICKET_CONCLUSION = gql`
  mutation UpdateTicketConclusion(
    $id: ID!
    $conclusion: String!
    $reason: String!
  ) {
    updateTicketConclusion(id: $id, conclusion: $conclusion, reason: $reason) {
      id
      status
      title
      description
      conclusion
      reason
      isTest
      assignee {
        id
        name
      }
      createdAt
      updatedAt
    }
  }
`;

export const UPDATE_TICKET = gql`
  mutation UpdateTicket($id: ID!, $title: String!, $description: String) {
    updateTicket(id: $id, title: $title, description: $description) {
      id
      status
      title
      description
      summary
      conclusion
      reason
      isTest
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
      slackLink
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
        user {
          id
          name
        }
        createdAt
        updatedAt
      }
    }
  }
`;

export const CREATE_TICKET = gql`
  mutation CreateTicket(
    $title: String!
    $description: String!
    $isTest: Boolean
  ) {
    createTicket(title: $title, description: $description, isTest: $isTest) {
      id
      status
      title
      description
      isTest
      assignee {
        id
        name
      }
      createdAt
      updatedAt
    }
  }
`;

export const GET_SIMILAR_TICKETS = gql`
  query GetSimilarTickets($ticketId: ID!, $threshold: Float!, $offset: Int, $limit: Int) {
    similarTickets(ticketId: $ticketId, threshold: $threshold, offset: $offset, limit: $limit) {
      tickets {
        id
        status
        title
        description
        isTest
        assignee {
          id
          name
        }
        createdAt
        updatedAt
      }
      totalCount
    }
  }
`;

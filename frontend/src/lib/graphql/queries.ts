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
        alertsCount
        commentsCount
        tags
        tagObjects {
          id
          name
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
      alertsCount
      tags
      tagObjects {
        id
        name
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

export const GET_TICKET_ALERTS = gql`
  query GetTicketAlerts($id: ID!, $offset: Int, $limit: Int) {
    ticket(id: $id) {
      id
      alertsPaginated(offset: $offset, limit: $limit) {
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
          tags
          tagObjects {
            id
            name
          }
        }
        totalCount
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
      tags
      tagObjects {
        id
        name
      }
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
        tags
        tagObjects {
          id
          name
        }
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
        schema
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
      tags
      tagObjects {
        id
        name
      }
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
        tags
        tagObjects {
          id
          name
        }
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

export const GET_SIMILAR_TICKETS_FOR_ALERT = gql`
  query GetSimilarTicketsForAlert($alertId: ID!, $threshold: Float!, $offset: Int, $limit: Int) {
    similarTicketsForAlert(alertId: $alertId, threshold: $threshold, offset: $offset, limit: $limit) {
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

export const GET_UNBOUND_ALERTS = gql`
  query GetNewAlerts($threshold: Float, $keyword: String, $ticketId: ID, $offset: Int, $limit: Int) {
    unboundAlerts(threshold: $threshold, keyword: $keyword, ticketId: $ticketId, offset: $offset, limit: $limit) {
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
      totalCount
    }
  }
`;

export const BIND_ALERTS_TO_TICKET = gql`
  mutation BindAlertsToTicket($ticketId: ID!, $alertIds: [ID!]!) {
    bindAlertsToTicket(ticketId: $ticketId, alertIds: $alertIds) {
      id
      title
      alertsCount
      alerts {
        id
        title
        createdAt
      }
    }
  }
`;

export const CREATE_TICKET_FROM_ALERTS = gql`
  mutation CreateTicketFromAlerts($alertIds: [ID!]!, $title: String, $description: String) {
    createTicketFromAlerts(alertIds: $alertIds, title: $title, description: $description) {
      id
      status
      title
      description
      summary
      isTest
      assignee {
        id
        name
      }
      createdAt
      updatedAt
      alertsCount
      alerts {
        id
        title
        description
      }
    }
  }
`;

export const GET_ALERT_CLUSTERS = gql`
  query AlertClusters($limit: Int, $offset: Int, $minClusterSize: Int, $eps: Float, $minSamples: Int, $keyword: String) {
    alertClusters(limit: $limit, offset: $offset, minClusterSize: $minClusterSize, eps: $eps, minSamples: $minSamples, keyword: $keyword) {
      clusters {
        id
        size
        keywords
        createdAt
        centerAlert {
          id
          title
          description
          schema
          data
          createdAt
        }
      }
      noiseAlerts {
        id
        title
        description
        schema
        createdAt
      }
      parameters {
        eps
        minSamples
      }
      computedAt
      totalCount
    }
  }
`;

export const GET_CLUSTER_ALERTS = gql`
  query ClusterAlerts($clusterID: ID!, $keyword: String, $limit: Int, $offset: Int) {
    clusterAlerts(clusterID: $clusterID, keyword: $keyword, limit: $limit, offset: $offset) {
      alerts {
        id
        title
        description
        schema
        data
        createdAt
        ticket {
          id
          title
          status
        }
      }
      totalCount
    }
  }
`;

export const GET_TAGS = gql`
  query GetTags {
    tags {
      id
      name
      description
      color
      createdAt
      updatedAt
    }
  }
`;

export const CREATE_TAG = gql`
  mutation CreateTag($name: String!) {
    createTag(name: $name) {
      id
      name
      description
      color
      createdAt
      updatedAt
    }
  }
`;

export const DELETE_TAG = gql`
  mutation DeleteTag($name: String!) {
    deleteTag(name: $name)
  }
`;

export const UPDATE_TAG = gql`
  mutation UpdateTag($input: UpdateTagInput!) {
    updateTag(input: $input) {
      id
      name
      description
      color
      createdAt
      updatedAt
    }
  }
`;

export const GET_AVAILABLE_TAG_COLORS = gql`
  query GetAvailableTagColors {
    availableTagColors
  }
`;

export const GET_AVAILABLE_TAG_COLOR_NAMES = gql`
  query GetAvailableTagColorNames {
    availableTagColorNames
  }
`;

export const UPDATE_ALERT_TAGS = gql`
  mutation UpdateAlertTags($alertId: ID!, $tagIds: [ID!]!) {
    updateAlertTags(alertId: $alertId, tagIds: $tagIds) {
      id
      title
      tags
      tagObjects {
        id
        name
      }
    }
  }
`;

export const UPDATE_TICKET_TAGS = gql`
  mutation UpdateTicketTags($ticketId: ID!, $tagIds: [ID!]!) {
    updateTicketTags(ticketId: $ticketId, tagIds: $tagIds) {
      id
      title
      tags
      tagObjects {
        id
        name
      }
    }
  }
`;
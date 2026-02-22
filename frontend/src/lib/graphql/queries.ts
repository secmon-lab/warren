import { gql } from "@apollo/client";

export const GET_TICKETS = gql`
  query GetTickets($statuses: [String!], $keyword: String, $assigneeID: String, $offset: Int, $limit: Int) {
    tickets(statuses: $statuses, keyword: $keyword, assigneeID: $assigneeID, offset: $offset, limit: $limit) {
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
  query GetAlerts($offset: Int, $limit: Int, $status: AlertStatus) {
    alerts(offset: $offset, limit: $limit, status: $status) {
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
        status
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
      declinedAlertsCount
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

export const RESOLVE_TICKET = gql`
  mutation ResolveTicket($id: ID!, $conclusion: String!, $reason: String!) {
    resolveTicket(id: $id, conclusion: $conclusion, reason: $reason) {
      id
      status
      title
      description
      conclusion
      reason
      resolvedAt
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

export const REOPEN_TICKET = gql`
  mutation ReopenTicket($id: ID!) {
    reopenTicket(id: $id) {
      id
      status
      title
      description
      conclusion
      reason
      resolvedAt
      archivedAt
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

export const ARCHIVE_TICKET = gql`
  mutation ArchiveTicket($id: ID!) {
    archiveTicket(id: $id) {
      id
      status
      title
      description
      resolvedAt
      archivedAt
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

export const ARCHIVE_TICKETS = gql`
  mutation ArchiveTickets($ids: [ID!]!) {
    archiveTickets(ids: $ids) {
      id
      status
      title
      description
      resolvedAt
      archivedAt
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

export const UNARCHIVE_TICKET = gql`
  mutation UnarchiveTicket($id: ID!) {
    unarchiveTicket(id: $id) {
      id
      status
      title
      description
      resolvedAt
      archivedAt
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
  query GetSimilarTickets(
    $ticketId: ID!
    $threshold: Float!
    $offset: Int
    $limit: Int
  ) {
    similarTickets(
      ticketId: $ticketId
      threshold: $threshold
      offset: $offset
      limit: $limit
    ) {
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
  query GetSimilarTicketsForAlert(
    $alertId: ID!
    $threshold: Float!
    $offset: Int
    $limit: Int
  ) {
    similarTicketsForAlert(
      alertId: $alertId
      threshold: $threshold
      offset: $offset
      limit: $limit
    ) {
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
  query GetNewAlerts(
    $threshold: Float
    $keyword: String
    $ticketId: ID
    $offset: Int
    $limit: Int
  ) {
    unboundAlerts(
      threshold: $threshold
      keyword: $keyword
      ticketId: $ticketId
      offset: $offset
      limit: $limit
    ) {
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
  mutation CreateTicketFromAlerts(
    $alertIds: [ID!]!
    $title: String
    $description: String
  ) {
    createTicketFromAlerts(
      alertIds: $alertIds
      title: $title
      description: $description
    ) {
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
  mutation DeleteTag($id: ID!) {
    deleteTag(id: $id)
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

export const GET_KNOWLEDGE_TOPICS = gql`
  query GetKnowledgeTopics {
    knowledgeTopics {
      topic
      count
    }
  }
`;

export const GET_KNOWLEDGES_BY_TOPIC = gql`
  query GetKnowledgesByTopic($topic: String!) {
    knowledgesByTopic(topic: $topic) {
      slug
      name
      topic
      content
      commitID
      authorID
      author {
        id
        name
        icon
      }
      createdAt
      updatedAt
      state
    }
  }
`;

export const CREATE_KNOWLEDGE = gql`
  mutation CreateKnowledge($input: CreateKnowledgeInput!) {
    createKnowledge(input: $input) {
      slug
      name
      topic
      content
      commitID
      authorID
      author {
        id
        name
        icon
      }
      createdAt
      updatedAt
      state
    }
  }
`;

export const UPDATE_KNOWLEDGE = gql`
  mutation UpdateKnowledge($input: UpdateKnowledgeInput!) {
    updateKnowledge(input: $input) {
      slug
      name
      topic
      content
      commitID
      authorID
      author {
        id
        name
        icon
      }
      createdAt
      updatedAt
      state
    }
  }
`;

export const ARCHIVE_KNOWLEDGE = gql`
  mutation ArchiveKnowledge($topic: String!, $slug: String!) {
    archiveKnowledge(topic: $topic, slug: $slug)
  }
`;

export const GET_TICKET_SESSIONS = gql`
  query GetTicketSessions($ticketId: ID!) {
    ticketSessions(ticketId: $ticketId) {
      id
      ticketID
      status
      userID
      user {
        id
        name
        icon
      }
      query
      slackURL
      intent
      createdAt
      updatedAt
    }
  }
`;

export const GET_SESSION = gql`
  query GetSession($id: ID!) {
    session(id: $id) {
      id
      ticketID
      status
      userID
      user {
        id
        name
        icon
      }
      query
      slackURL
      intent
      createdAt
      updatedAt
    }
  }
`;

export const GET_SESSION_MESSAGES = gql`
  query GetSessionMessages($sessionId: ID!) {
    sessionMessages(sessionId: $sessionId) {
      id
      sessionID
      type
      content
      createdAt
      updatedAt
    }
  }
`;

export const LIST_AGENT_SUMMARIES = gql`
  query ListAgentSummaries($offset: Int, $limit: Int, $keyword: String) {
    listAgentSummaries(offset: $offset, limit: $limit, keyword: $keyword) {
      agents {
        agentID
        memoriesCount
        latestMemoryAt
      }
      totalCount
    }
  }
`;

export const LIST_AGENT_MEMORIES = gql`
  query ListAgentMemories(
    $agentID: String!
    $offset: Int
    $limit: Int
    $sortBy: MemorySortField
    $sortOrder: SortOrder
    $keyword: String
    $minScore: Float
    $maxScore: Float
  ) {
    listAgentMemories(
      agentID: $agentID
      offset: $offset
      limit: $limit
      sortBy: $sortBy
      sortOrder: $sortOrder
      keyword: $keyword
      minScore: $minScore
      maxScore: $maxScore
    ) {
      memories {
        id
        agentID
        query
        claim
        score
        createdAt
        lastUsedAt
      }
      totalCount
    }
  }
`;

export const GET_AGENT_MEMORY = gql`
  query GetAgentMemory($agentID: String!, $memoryID: ID!) {
    getAgentMemory(agentID: $agentID, memoryID: $memoryID) {
      id
      agentID
      query
      claim
      score
      createdAt
      lastUsedAt
    }
  }
`;

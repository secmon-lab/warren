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
      queuedAlertsCount
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

// chat-session-redesign Phase 6: timeline of SessionMessage rows for a
// ticket. source / type are optional filters matching the
// `warren_get_ticket_session_messages` base tool surface.
export const GET_TICKET_SESSION_MESSAGES = gql`
  query GetTicketSessionMessages(
    $ticketId: ID!
    $source: String
    $type: String
    $limit: Int
    $offset: Int
  ) {
    ticketSessionMessages(
      ticketId: $ticketId
      source: $source
      type: $type
      limit: $limit
      offset: $offset
    ) {
      id
      sessionID
      turnID
      ticketID
      type
      content
      createdAt
      updatedAt
      author {
        userID
        displayName
        slackUserID
      }
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

export const ARCHIVE_ALL_RESOLVED_TICKETS = gql`
  mutation ArchiveAllResolvedTickets {
    archiveAllResolvedTickets {
      archivedCount
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

export const DECLINE_ALERTS = gql`
  mutation DeclineAlerts($ids: [ID!]!) {
    declineAlerts(ids: $ids) {
      id
      status
      title
    }
  }
`;

// Knowledge queries
export const GET_KNOWLEDGES = gql`
  query GetKnowledges($category: String, $tags: [ID!], $keyword: String) {
    knowledges(category: $category, tags: $tags, keyword: $keyword) {
      id
      category
      title
      claim
      tags {
        id
        name
        description
      }
      authorID
      author {
        id
        name
        icon
      }
      createdAt
      updatedAt
    }
  }
`;

export const GET_KNOWLEDGE = gql`
  query GetKnowledge($id: ID!) {
    knowledge(id: $id) {
      id
      category
      title
      claim
      tags {
        id
        name
        description
      }
      authorID
      author {
        id
        name
        icon
      }
      createdAt
      updatedAt
    }
  }
`;

export const GET_KNOWLEDGE_LOGS = gql`
  query GetKnowledgeLogs($knowledgeID: ID!) {
    knowledgeLogs(knowledgeID: $knowledgeID) {
      id
      knowledgeID
      title
      claim
      authorID
      author {
        id
        name
        icon
      }
      ticketID
      message
      createdAt
    }
  }
`;

export const GET_KNOWLEDGE_TAGS = gql`
  query GetKnowledgeTags {
    knowledgeTags {
      id
      name
      description
      createdAt
      updatedAt
    }
  }
`;

export const CREATE_KNOWLEDGE = gql`
  mutation CreateKnowledge($input: CreateKnowledgeInput!) {
    createKnowledge(input: $input) {
      id
      category
      title
      claim
      tags {
        id
        name
      }
      createdAt
      updatedAt
    }
  }
`;

export const UPDATE_KNOWLEDGE = gql`
  mutation UpdateKnowledge($input: UpdateKnowledgeInput!) {
    updateKnowledge(input: $input) {
      id
      category
      title
      claim
      tags {
        id
        name
      }
      createdAt
      updatedAt
    }
  }
`;

export const DELETE_KNOWLEDGE = gql`
  mutation DeleteKnowledge($id: ID!, $reason: String!) {
    deleteKnowledge(id: $id, reason: $reason)
  }
`;

export const CREATE_KNOWLEDGE_TAG = gql`
  mutation CreateKnowledgeTag($input: CreateKnowledgeTagInput!) {
    createKnowledgeTag(input: $input) {
      id
      name
      description
    }
  }
`;

export const UPDATE_KNOWLEDGE_TAG = gql`
  mutation UpdateKnowledgeTag($input: UpdateKnowledgeTagInput!) {
    updateKnowledgeTag(input: $input) {
      id
      name
      description
    }
  }
`;

export const DELETE_KNOWLEDGE_TAG = gql`
  mutation DeleteKnowledgeTag($id: ID!) {
    deleteKnowledgeTag(id: $id)
  }
`;

export const MERGE_KNOWLEDGE_TAGS = gql`
  mutation MergeKnowledgeTags($oldID: ID!, $newID: ID!) {
    mergeKnowledgeTags(oldID: $oldID, newID: $newID)
  }
`;

export const GET_TICKET_SESSIONS = gql`
  query GetTicketSessions($ticketId: ID!) {
    ticketSessions(ticketId: $ticketId) {
      id
      ticketID
      status
      source
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


export const GET_DIAGNOSES = gql`
  query GetDiagnoses($offset: Int, $limit: Int) {
    diagnoses(offset: $offset, limit: $limit) {
      diagnoses {
        id
        status
        totalCount
        pendingCount
        fixedCount
        failedCount
        createdAt
        updatedAt
      }
      totalCount
    }
  }
`;

export const GET_DIAGNOSIS = gql`
  query GetDiagnosis($id: ID!) {
    diagnosis(id: $id) {
      id
      status
      totalCount
      pendingCount
      fixedCount
      failedCount
      createdAt
      updatedAt
    }
  }
`;

export const GET_DIAGNOSIS_ISSUES = gql`
  query GetDiagnosisIssues($diagnosisID: ID!, $offset: Int, $limit: Int, $status: String, $ruleID: String) {
    diagnosisIssues(diagnosisID: $diagnosisID, offset: $offset, limit: $limit, status: $status, ruleID: $ruleID) {
      issues {
        id
        diagnosisID
        ruleID
        targetID
        description
        status
        fixedAt
        failReason
        createdAt
      }
      totalCount
    }
  }
`;

export const RUN_DIAGNOSIS = gql`
  mutation RunDiagnosis {
    runDiagnosis {
      id
      status
      createdAt
    }
  }
`;

export const FIX_DIAGNOSIS = gql`
  mutation FixDiagnosis($id: ID!) {
    fixDiagnosis(id: $id) {
      id
      status
      totalCount
      pendingCount
      fixedCount
      failedCount
      updatedAt
    }
  }
`;

export const GET_QUEUED_ALERTS = gql`
  query GetQueuedAlerts($keyword: String, $offset: Int, $limit: Int) {
    queuedAlerts(keyword: $keyword, offset: $offset, limit: $limit) {
      alerts {
        id
        schema
        title
        data
        createdAt
      }
      totalCount
    }
  }
`;

export const GET_REPROCESS_JOB = gql`
  query GetReprocessJob($id: ID!) {
    reprocessJob(id: $id) {
      id
      queuedAlertID
      status
      error
      createdAt
      updatedAt
    }
  }
`;

export const REPROCESS_QUEUED_ALERT = gql`
  mutation ReprocessQueuedAlert($id: ID!) {
    reprocessQueuedAlert(id: $id) {
      id
      queuedAlertID
      status
      createdAt
      updatedAt
    }
  }
`;

export const DISCARD_QUEUED_ALERTS = gql`
  mutation DiscardQueuedAlerts($ids: [ID!]!) {
    discardQueuedAlerts(ids: $ids)
  }
`;

export const DISCARD_QUEUED_ALERTS_BY_FILTER = gql`
  mutation DiscardQueuedAlertsByFilter($keyword: String) {
    discardQueuedAlertsByFilter(keyword: $keyword)
  }
`;

export const REPROCESS_QUEUED_ALERTS_BY_FILTER = gql`
  mutation ReprocessQueuedAlertsByFilter($keyword: String) {
    reprocessQueuedAlertsByFilter(keyword: $keyword) {
      id
      status
      totalCount
      completedCount
      failedCount
      createdAt
      updatedAt
    }
  }
`;

export const GET_REPROCESS_BATCH_JOB = gql`
  query GetReprocessBatchJob($id: ID!) {
    reprocessBatchJob(id: $id) {
      id
      status
      totalCount
      completedCount
      failedCount
      createdAt
      updatedAt
    }
  }
`;

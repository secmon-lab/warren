import { DocumentNode } from 'graphql';
import * as Apollo from '@apollo/client';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
const defaultOptions = {} as const;
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
};

export type ActivitiesResponse = {
  __typename?: 'ActivitiesResponse';
  activities: Array<Activity>;
  totalCount: Scalars['Int']['output'];
};

export type Activity = {
  __typename?: 'Activity';
  alert?: Maybe<Alert>;
  alertID?: Maybe<Scalars['String']['output']>;
  commentID?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  metadata?: Maybe<Scalars['String']['output']>;
  ticket?: Maybe<Ticket>;
  ticketID?: Maybe<Scalars['String']['output']>;
  type: Scalars['String']['output'];
  user?: Maybe<User>;
  userID?: Maybe<Scalars['String']['output']>;
};

export type AgentMemoriesResponse = {
  __typename?: 'AgentMemoriesResponse';
  memories: Array<AgentMemory>;
  totalCount: Scalars['Int']['output'];
};

export type AgentMemory = {
  __typename?: 'AgentMemory';
  agentID: Scalars['String']['output'];
  claim: Scalars['String']['output'];
  createdAt: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  lastUsedAt?: Maybe<Scalars['String']['output']>;
  query: Scalars['String']['output'];
  score: Scalars['Float']['output'];
};

export type AgentSummariesResponse = {
  __typename?: 'AgentSummariesResponse';
  agents: Array<AgentSummary>;
  totalCount: Scalars['Int']['output'];
};

export type AgentSummary = {
  __typename?: 'AgentSummary';
  agentID: Scalars['String']['output'];
  latestMemoryAt?: Maybe<Scalars['String']['output']>;
  memoriesCount: Scalars['Int']['output'];
};

export type Alert = {
  __typename?: 'Alert';
  attributes: Array<AlertAttribute>;
  createdAt: Scalars['String']['output'];
  data: Scalars['String']['output'];
  description?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  schema: Scalars['String']['output'];
  status: AlertStatus;
  tagObjects: Array<TagObject>;
  tags: Array<Scalars['String']['output']>;
  ticket?: Maybe<Ticket>;
  title: Scalars['String']['output'];
};

export type AlertAttribute = {
  __typename?: 'AlertAttribute';
  auto: Scalars['Boolean']['output'];
  key: Scalars['String']['output'];
  link?: Maybe<Scalars['String']['output']>;
  value: Scalars['String']['output'];
};

export type AlertStatus =
  | 'ACTIVE'
  | 'DECLINED';

export type AlertsResponse = {
  __typename?: 'AlertsResponse';
  alerts: Array<Alert>;
  totalCount: Scalars['Int']['output'];
};

export type Comment = {
  __typename?: 'Comment';
  content: Scalars['String']['output'];
  createdAt: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  updatedAt: Scalars['String']['output'];
  user?: Maybe<User>;
};

export type CommentsResponse = {
  __typename?: 'CommentsResponse';
  comments: Array<Comment>;
  totalCount: Scalars['Int']['output'];
};

export type CreateKnowledgeInput = {
  content: Scalars['String']['input'];
  name: Scalars['String']['input'];
  slug: Scalars['String']['input'];
  topic: Scalars['String']['input'];
};

export type DashboardStats = {
  __typename?: 'DashboardStats';
  declinedAlertsCount: Scalars['Int']['output'];
  openTickets: Array<Ticket>;
  openTicketsCount: Scalars['Int']['output'];
  unboundAlerts: Array<Alert>;
  unboundAlertsCount: Scalars['Int']['output'];
};

export type Finding = {
  __typename?: 'Finding';
  reason: Scalars['String']['output'];
  recommendation: Scalars['String']['output'];
  severity: Scalars['String']['output'];
  summary: Scalars['String']['output'];
};

export type Knowledge = {
  __typename?: 'Knowledge';
  author: User;
  authorID: Scalars['String']['output'];
  commitID: Scalars['String']['output'];
  content: Scalars['String']['output'];
  createdAt: Scalars['String']['output'];
  name: Scalars['String']['output'];
  slug: Scalars['String']['output'];
  state: Scalars['String']['output'];
  topic: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
};

export type MemorySortField =
  | 'CREATED_AT'
  | 'LAST_USED_AT'
  | 'SCORE';

export type Mutation = {
  __typename?: 'Mutation';
  archiveKnowledge: Scalars['Boolean']['output'];
  archiveTicket: Ticket;
  archiveTickets: Array<Ticket>;
  bindAlertsToTicket: Ticket;
  createKnowledge: Knowledge;
  createTag: TagMetadata;
  createTicket: Ticket;
  createTicketFromAlerts: Ticket;
  declineAlerts: Array<Alert>;
  deleteTag: Scalars['Boolean']['output'];
  reopenTicket: Ticket;
  resolveTicket: Ticket;
  unarchiveTicket: Ticket;
  updateAlertTags: Alert;
  updateKnowledge: Knowledge;
  updateTag: TagMetadata;
  updateTicket: Ticket;
  updateTicketConclusion: Ticket;
  updateTicketTags: Ticket;
};


export type MutationArchiveKnowledgeArgs = {
  slug: Scalars['String']['input'];
  topic: Scalars['String']['input'];
};


export type MutationArchiveTicketArgs = {
  id: Scalars['ID']['input'];
};


export type MutationArchiveTicketsArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type MutationBindAlertsToTicketArgs = {
  alertIds: Array<Scalars['ID']['input']>;
  ticketId: Scalars['ID']['input'];
};


export type MutationCreateKnowledgeArgs = {
  input: CreateKnowledgeInput;
};


export type MutationCreateTagArgs = {
  name: Scalars['String']['input'];
};


export type MutationCreateTicketArgs = {
  description: Scalars['String']['input'];
  isTest?: InputMaybe<Scalars['Boolean']['input']>;
  title: Scalars['String']['input'];
};


export type MutationCreateTicketFromAlertsArgs = {
  alertIds: Array<Scalars['ID']['input']>;
  description?: InputMaybe<Scalars['String']['input']>;
  title?: InputMaybe<Scalars['String']['input']>;
};


export type MutationDeclineAlertsArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type MutationDeleteTagArgs = {
  id: Scalars['ID']['input'];
};


export type MutationReopenTicketArgs = {
  id: Scalars['ID']['input'];
};


export type MutationResolveTicketArgs = {
  conclusion: Scalars['String']['input'];
  id: Scalars['ID']['input'];
  reason: Scalars['String']['input'];
};


export type MutationUnarchiveTicketArgs = {
  id: Scalars['ID']['input'];
};


export type MutationUpdateAlertTagsArgs = {
  alertId: Scalars['ID']['input'];
  tagIds: Array<Scalars['ID']['input']>;
};


export type MutationUpdateKnowledgeArgs = {
  input: UpdateKnowledgeInput;
};


export type MutationUpdateTagArgs = {
  input: UpdateTagInput;
};


export type MutationUpdateTicketArgs = {
  description?: InputMaybe<Scalars['String']['input']>;
  id: Scalars['ID']['input'];
  title: Scalars['String']['input'];
};


export type MutationUpdateTicketConclusionArgs = {
  conclusion: Scalars['String']['input'];
  id: Scalars['ID']['input'];
  reason: Scalars['String']['input'];
};


export type MutationUpdateTicketTagsArgs = {
  tagIds: Array<Scalars['ID']['input']>;
  ticketId: Scalars['ID']['input'];
};

export type Query = {
  __typename?: 'Query';
  activities: ActivitiesResponse;
  alert?: Maybe<Alert>;
  alerts: AlertsResponse;
  availableTagColorNames: Array<Scalars['String']['output']>;
  availableTagColors: Array<Scalars['String']['output']>;
  dashboard: DashboardStats;
  getAgentMemory?: Maybe<AgentMemory>;
  knowledgeTopics: Array<TopicSummary>;
  knowledgesByTopic: Array<Knowledge>;
  listAgentMemories: AgentMemoriesResponse;
  listAgentSummaries: AgentSummariesResponse;
  session?: Maybe<Session>;
  sessionMessages: Array<SessionMessage>;
  similarTickets: TicketsResponse;
  similarTicketsForAlert: TicketsResponse;
  tags: Array<TagMetadata>;
  ticket?: Maybe<Ticket>;
  ticketComments: CommentsResponse;
  ticketSessions: Array<Session>;
  tickets: TicketsResponse;
  unboundAlerts: AlertsResponse;
};


export type QueryActivitiesArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryAlertArgs = {
  id: Scalars['ID']['input'];
};


export type QueryAlertsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  status?: InputMaybe<AlertStatus>;
};


export type QueryGetAgentMemoryArgs = {
  agentID: Scalars['String']['input'];
  memoryID: Scalars['ID']['input'];
};


export type QueryKnowledgesByTopicArgs = {
  topic: Scalars['String']['input'];
};


export type QueryListAgentMemoriesArgs = {
  agentID: Scalars['String']['input'];
  keyword?: InputMaybe<Scalars['String']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  maxScore?: InputMaybe<Scalars['Float']['input']>;
  minScore?: InputMaybe<Scalars['Float']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  sortBy?: InputMaybe<MemorySortField>;
  sortOrder?: InputMaybe<SortOrder>;
};


export type QueryListAgentSummariesArgs = {
  keyword?: InputMaybe<Scalars['String']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};


export type QuerySessionArgs = {
  id: Scalars['ID']['input'];
};


export type QuerySessionMessagesArgs = {
  sessionId: Scalars['ID']['input'];
};


export type QuerySimilarTicketsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  threshold: Scalars['Float']['input'];
  ticketId: Scalars['ID']['input'];
};


export type QuerySimilarTicketsForAlertArgs = {
  alertId: Scalars['ID']['input'];
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  threshold: Scalars['Float']['input'];
};


export type QueryTicketArgs = {
  id: Scalars['ID']['input'];
};


export type QueryTicketCommentsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  ticketId: Scalars['ID']['input'];
};


export type QueryTicketSessionsArgs = {
  ticketId: Scalars['ID']['input'];
};


export type QueryTicketsArgs = {
  assigneeID?: InputMaybe<Scalars['String']['input']>;
  keyword?: InputMaybe<Scalars['String']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  statuses?: InputMaybe<Array<Scalars['String']['input']>>;
};


export type QueryUnboundAlertsArgs = {
  keyword?: InputMaybe<Scalars['String']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  threshold?: InputMaybe<Scalars['Float']['input']>;
  ticketId?: InputMaybe<Scalars['ID']['input']>;
};

export type Session = {
  __typename?: 'Session';
  createdAt: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  intent?: Maybe<Scalars['String']['output']>;
  query?: Maybe<Scalars['String']['output']>;
  slackURL?: Maybe<Scalars['String']['output']>;
  status: Scalars['String']['output'];
  ticketID: Scalars['ID']['output'];
  updatedAt: Scalars['String']['output'];
  user?: Maybe<User>;
  userID?: Maybe<Scalars['String']['output']>;
};

export type SessionMessage = {
  __typename?: 'SessionMessage';
  content: Scalars['String']['output'];
  createdAt: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  sessionID: Scalars['ID']['output'];
  type: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
};

export type SortOrder =
  | 'ASC'
  | 'DESC';

export type TagMetadata = {
  __typename?: 'TagMetadata';
  color: Scalars['String']['output'];
  createdAt: Scalars['String']['output'];
  description?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
};

export type TagObject = {
  __typename?: 'TagObject';
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
};

export type Ticket = {
  __typename?: 'Ticket';
  alerts: Array<Alert>;
  alertsCount: Scalars['Int']['output'];
  alertsPaginated: AlertsResponse;
  archivedAt?: Maybe<Scalars['String']['output']>;
  assignee?: Maybe<User>;
  comments: Array<Comment>;
  commentsCount: Scalars['Int']['output'];
  conclusion?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['String']['output'];
  description: Scalars['String']['output'];
  finding?: Maybe<Finding>;
  id: Scalars['ID']['output'];
  isTest: Scalars['Boolean']['output'];
  reason?: Maybe<Scalars['String']['output']>;
  resolvedAt?: Maybe<Scalars['String']['output']>;
  slackLink?: Maybe<Scalars['String']['output']>;
  status: Scalars['String']['output'];
  summary: Scalars['String']['output'];
  tagObjects: Array<TagObject>;
  tags: Array<Scalars['String']['output']>;
  title: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
};


export type TicketAlertsPaginatedArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};

export type TicketsResponse = {
  __typename?: 'TicketsResponse';
  tickets: Array<Ticket>;
  totalCount: Scalars['Int']['output'];
};

export type TopicSummary = {
  __typename?: 'TopicSummary';
  count: Scalars['Int']['output'];
  topic: Scalars['String']['output'];
};

export type UpdateKnowledgeInput = {
  content: Scalars['String']['input'];
  name: Scalars['String']['input'];
  slug: Scalars['String']['input'];
  topic: Scalars['String']['input'];
};

export type UpdateTagInput = {
  color: Scalars['String']['input'];
  description?: InputMaybe<Scalars['String']['input']>;
  id: Scalars['ID']['input'];
  name: Scalars['String']['input'];
};

export type User = {
  __typename?: 'User';
  icon?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
};

export type GetTicketsQueryVariables = Exact<{
  statuses?: InputMaybe<Array<Scalars['String']['input']> | Scalars['String']['input']>;
  keyword?: InputMaybe<Scalars['String']['input']>;
  assigneeID?: InputMaybe<Scalars['String']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetTicketsQuery = { __typename?: 'Query', tickets: { __typename?: 'TicketsResponse', totalCount: number, tickets: Array<{ __typename?: 'Ticket', id: string, status: string, title: string, description: string, conclusion?: string | null, reason?: string | null, isTest: boolean, createdAt: string, updatedAt: string, alertsCount: number, commentsCount: number, tags: Array<string>, assignee?: { __typename?: 'User', id: string, name: string } | null, tagObjects: Array<{ __typename?: 'TagObject', id: string, name: string }> }> } };

export type GetTicketQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetTicketQuery = { __typename?: 'Query', ticket?: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, summary: string, conclusion?: string | null, reason?: string | null, isTest: boolean, slackLink?: string | null, createdAt: string, updatedAt: string, alertsCount: number, tags: Array<string>, finding?: { __typename?: 'Finding', severity: string, summary: string, reason: string, recommendation: string } | null, assignee?: { __typename?: 'User', id: string, name: string } | null, tagObjects: Array<{ __typename?: 'TagObject', id: string, name: string }>, comments: Array<{ __typename?: 'Comment', id: string, content: string, createdAt: string, updatedAt: string, user?: { __typename?: 'User', id: string, name: string } | null }> } | null };

export type GetTicketAlertsQueryVariables = Exact<{
  id: Scalars['ID']['input'];
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetTicketAlertsQuery = { __typename?: 'Query', ticket?: { __typename?: 'Ticket', id: string, alertsPaginated: { __typename?: 'AlertsResponse', totalCount: number, alerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, data: string, createdAt: string, tags: Array<string>, attributes: Array<{ __typename?: 'AlertAttribute', key: string, value: string, link?: string | null, auto: boolean }>, tagObjects: Array<{ __typename?: 'TagObject', id: string, name: string }> }> } } | null };

export type GetAlertQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetAlertQuery = { __typename?: 'Query', alert?: { __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, data: string, createdAt: string, tags: Array<string>, attributes: Array<{ __typename?: 'AlertAttribute', key: string, value: string, link?: string | null, auto: boolean }>, tagObjects: Array<{ __typename?: 'TagObject', id: string, name: string }>, ticket?: { __typename?: 'Ticket', id: string, status: string, title: string, description: string } | null } | null };

export type GetAlertsQueryVariables = Exact<{
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  status?: InputMaybe<AlertStatus>;
}>;


export type GetAlertsQuery = { __typename?: 'Query', alerts: { __typename?: 'AlertsResponse', totalCount: number, alerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, data: string, status: AlertStatus, createdAt: string, tags: Array<string>, attributes: Array<{ __typename?: 'AlertAttribute', key: string, value: string, link?: string | null, auto: boolean }>, tagObjects: Array<{ __typename?: 'TagObject', id: string, name: string }>, ticket?: { __typename?: 'Ticket', id: string, status: string, title: string, description: string } | null }> } };

export type GetDashboardQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDashboardQuery = { __typename?: 'Query', dashboard: { __typename?: 'DashboardStats', openTicketsCount: number, unboundAlertsCount: number, declinedAlertsCount: number, openTickets: Array<{ __typename?: 'Ticket', id: string, status: string, title: string, description: string, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null }>, unboundAlerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, createdAt: string }> } };

export type GetActivitiesQueryVariables = Exact<{
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetActivitiesQuery = { __typename?: 'Query', activities: { __typename?: 'ActivitiesResponse', totalCount: number, activities: Array<{ __typename?: 'Activity', id: string, type: string, userID?: string | null, alertID?: string | null, ticketID?: string | null, commentID?: string | null, createdAt: string, metadata?: string | null, user?: { __typename?: 'User', id: string, name: string } | null, alert?: { __typename?: 'Alert', id: string, title: string, description?: string | null } | null, ticket?: { __typename?: 'Ticket', id: string, title: string, description: string } | null }> } };

export type GetTicketCommentsQueryVariables = Exact<{
  ticketId: Scalars['ID']['input'];
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetTicketCommentsQuery = { __typename?: 'Query', ticketComments: { __typename?: 'CommentsResponse', totalCount: number, comments: Array<{ __typename?: 'Comment', id: string, content: string, createdAt: string, updatedAt: string, user?: { __typename?: 'User', id: string, name: string } | null }> } };

export type ResolveTicketMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  conclusion: Scalars['String']['input'];
  reason: Scalars['String']['input'];
}>;


export type ResolveTicketMutation = { __typename?: 'Mutation', resolveTicket: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, conclusion?: string | null, reason?: string | null, resolvedAt?: string | null, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null } };

export type ReopenTicketMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type ReopenTicketMutation = { __typename?: 'Mutation', reopenTicket: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, conclusion?: string | null, reason?: string | null, resolvedAt?: string | null, archivedAt?: string | null, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null } };

export type ArchiveTicketMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type ArchiveTicketMutation = { __typename?: 'Mutation', archiveTicket: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, resolvedAt?: string | null, archivedAt?: string | null, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null } };

export type ArchiveTicketsMutationVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type ArchiveTicketsMutation = { __typename?: 'Mutation', archiveTickets: Array<{ __typename?: 'Ticket', id: string, status: string, title: string, description: string, resolvedAt?: string | null, archivedAt?: string | null, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null }> };

export type UnarchiveTicketMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type UnarchiveTicketMutation = { __typename?: 'Mutation', unarchiveTicket: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, resolvedAt?: string | null, archivedAt?: string | null, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null } };

export type UpdateTicketConclusionMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  conclusion: Scalars['String']['input'];
  reason: Scalars['String']['input'];
}>;


export type UpdateTicketConclusionMutation = { __typename?: 'Mutation', updateTicketConclusion: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, conclusion?: string | null, reason?: string | null, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null } };

export type UpdateTicketMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  title: Scalars['String']['input'];
  description?: InputMaybe<Scalars['String']['input']>;
}>;


export type UpdateTicketMutation = { __typename?: 'Mutation', updateTicket: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, summary: string, conclusion?: string | null, reason?: string | null, isTest: boolean, slackLink?: string | null, createdAt: string, updatedAt: string, tags: Array<string>, finding?: { __typename?: 'Finding', severity: string, summary: string, reason: string, recommendation: string } | null, assignee?: { __typename?: 'User', id: string, name: string } | null, tagObjects: Array<{ __typename?: 'TagObject', id: string, name: string }>, alerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, data: string, createdAt: string, tags: Array<string>, attributes: Array<{ __typename?: 'AlertAttribute', key: string, value: string, link?: string | null, auto: boolean }>, tagObjects: Array<{ __typename?: 'TagObject', id: string, name: string }> }>, comments: Array<{ __typename?: 'Comment', id: string, content: string, createdAt: string, updatedAt: string, user?: { __typename?: 'User', id: string, name: string } | null }> } };

export type CreateTicketMutationVariables = Exact<{
  title: Scalars['String']['input'];
  description: Scalars['String']['input'];
  isTest?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type CreateTicketMutation = { __typename?: 'Mutation', createTicket: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null } };

export type GetSimilarTicketsQueryVariables = Exact<{
  ticketId: Scalars['ID']['input'];
  threshold: Scalars['Float']['input'];
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetSimilarTicketsQuery = { __typename?: 'Query', similarTickets: { __typename?: 'TicketsResponse', totalCount: number, tickets: Array<{ __typename?: 'Ticket', id: string, status: string, title: string, description: string, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null }> } };

export type GetSimilarTicketsForAlertQueryVariables = Exact<{
  alertId: Scalars['ID']['input'];
  threshold: Scalars['Float']['input'];
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetSimilarTicketsForAlertQuery = { __typename?: 'Query', similarTicketsForAlert: { __typename?: 'TicketsResponse', totalCount: number, tickets: Array<{ __typename?: 'Ticket', id: string, status: string, title: string, description: string, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null }> } };

export type GetNewAlertsQueryVariables = Exact<{
  threshold?: InputMaybe<Scalars['Float']['input']>;
  keyword?: InputMaybe<Scalars['String']['input']>;
  ticketId?: InputMaybe<Scalars['ID']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetNewAlertsQuery = { __typename?: 'Query', unboundAlerts: { __typename?: 'AlertsResponse', totalCount: number, alerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, data: string, createdAt: string, attributes: Array<{ __typename?: 'AlertAttribute', key: string, value: string, link?: string | null, auto: boolean }> }> } };

export type BindAlertsToTicketMutationVariables = Exact<{
  ticketId: Scalars['ID']['input'];
  alertIds: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type BindAlertsToTicketMutation = { __typename?: 'Mutation', bindAlertsToTicket: { __typename?: 'Ticket', id: string, title: string, alertsCount: number, alerts: Array<{ __typename?: 'Alert', id: string, title: string, createdAt: string }> } };

export type CreateTicketFromAlertsMutationVariables = Exact<{
  alertIds: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
  title?: InputMaybe<Scalars['String']['input']>;
  description?: InputMaybe<Scalars['String']['input']>;
}>;


export type CreateTicketFromAlertsMutation = { __typename?: 'Mutation', createTicketFromAlerts: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, summary: string, isTest: boolean, createdAt: string, updatedAt: string, alertsCount: number, assignee?: { __typename?: 'User', id: string, name: string } | null, alerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null }> } };

export type GetTagsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetTagsQuery = { __typename?: 'Query', tags: Array<{ __typename?: 'TagMetadata', id: string, name: string, description?: string | null, color: string, createdAt: string, updatedAt: string }> };

export type CreateTagMutationVariables = Exact<{
  name: Scalars['String']['input'];
}>;


export type CreateTagMutation = { __typename?: 'Mutation', createTag: { __typename?: 'TagMetadata', id: string, name: string, description?: string | null, color: string, createdAt: string, updatedAt: string } };

export type DeleteTagMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteTagMutation = { __typename?: 'Mutation', deleteTag: boolean };

export type UpdateTagMutationVariables = Exact<{
  input: UpdateTagInput;
}>;


export type UpdateTagMutation = { __typename?: 'Mutation', updateTag: { __typename?: 'TagMetadata', id: string, name: string, description?: string | null, color: string, createdAt: string, updatedAt: string } };

export type GetAvailableTagColorsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAvailableTagColorsQuery = { __typename?: 'Query', availableTagColors: Array<string> };

export type GetAvailableTagColorNamesQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAvailableTagColorNamesQuery = { __typename?: 'Query', availableTagColorNames: Array<string> };

export type UpdateAlertTagsMutationVariables = Exact<{
  alertId: Scalars['ID']['input'];
  tagIds: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type UpdateAlertTagsMutation = { __typename?: 'Mutation', updateAlertTags: { __typename?: 'Alert', id: string, title: string, tags: Array<string>, tagObjects: Array<{ __typename?: 'TagObject', id: string, name: string }> } };

export type UpdateTicketTagsMutationVariables = Exact<{
  ticketId: Scalars['ID']['input'];
  tagIds: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type UpdateTicketTagsMutation = { __typename?: 'Mutation', updateTicketTags: { __typename?: 'Ticket', id: string, title: string, tags: Array<string>, tagObjects: Array<{ __typename?: 'TagObject', id: string, name: string }> } };

export type DeclineAlertsMutationVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type DeclineAlertsMutation = { __typename?: 'Mutation', declineAlerts: Array<{ __typename?: 'Alert', id: string, status: AlertStatus, title: string }> };

export type GetKnowledgeTopicsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetKnowledgeTopicsQuery = { __typename?: 'Query', knowledgeTopics: Array<{ __typename?: 'TopicSummary', topic: string, count: number }> };

export type GetKnowledgesByTopicQueryVariables = Exact<{
  topic: Scalars['String']['input'];
}>;


export type GetKnowledgesByTopicQuery = { __typename?: 'Query', knowledgesByTopic: Array<{ __typename?: 'Knowledge', slug: string, name: string, topic: string, content: string, commitID: string, authorID: string, createdAt: string, updatedAt: string, state: string, author: { __typename?: 'User', id: string, name: string, icon?: string | null } }> };

export type CreateKnowledgeMutationVariables = Exact<{
  input: CreateKnowledgeInput;
}>;


export type CreateKnowledgeMutation = { __typename?: 'Mutation', createKnowledge: { __typename?: 'Knowledge', slug: string, name: string, topic: string, content: string, commitID: string, authorID: string, createdAt: string, updatedAt: string, state: string, author: { __typename?: 'User', id: string, name: string, icon?: string | null } } };

export type UpdateKnowledgeMutationVariables = Exact<{
  input: UpdateKnowledgeInput;
}>;


export type UpdateKnowledgeMutation = { __typename?: 'Mutation', updateKnowledge: { __typename?: 'Knowledge', slug: string, name: string, topic: string, content: string, commitID: string, authorID: string, createdAt: string, updatedAt: string, state: string, author: { __typename?: 'User', id: string, name: string, icon?: string | null } } };

export type ArchiveKnowledgeMutationVariables = Exact<{
  topic: Scalars['String']['input'];
  slug: Scalars['String']['input'];
}>;


export type ArchiveKnowledgeMutation = { __typename?: 'Mutation', archiveKnowledge: boolean };

export type GetTicketSessionsQueryVariables = Exact<{
  ticketId: Scalars['ID']['input'];
}>;


export type GetTicketSessionsQuery = { __typename?: 'Query', ticketSessions: Array<{ __typename?: 'Session', id: string, ticketID: string, status: string, userID?: string | null, query?: string | null, slackURL?: string | null, intent?: string | null, createdAt: string, updatedAt: string, user?: { __typename?: 'User', id: string, name: string, icon?: string | null } | null }> };

export type GetSessionQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetSessionQuery = { __typename?: 'Query', session?: { __typename?: 'Session', id: string, ticketID: string, status: string, userID?: string | null, query?: string | null, slackURL?: string | null, intent?: string | null, createdAt: string, updatedAt: string, user?: { __typename?: 'User', id: string, name: string, icon?: string | null } | null } | null };

export type GetSessionMessagesQueryVariables = Exact<{
  sessionId: Scalars['ID']['input'];
}>;


export type GetSessionMessagesQuery = { __typename?: 'Query', sessionMessages: Array<{ __typename?: 'SessionMessage', id: string, sessionID: string, type: string, content: string, createdAt: string, updatedAt: string }> };

export type ListAgentSummariesQueryVariables = Exact<{
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  keyword?: InputMaybe<Scalars['String']['input']>;
}>;


export type ListAgentSummariesQuery = { __typename?: 'Query', listAgentSummaries: { __typename?: 'AgentSummariesResponse', totalCount: number, agents: Array<{ __typename?: 'AgentSummary', agentID: string, memoriesCount: number, latestMemoryAt?: string | null }> } };

export type ListAgentMemoriesQueryVariables = Exact<{
  agentID: Scalars['String']['input'];
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  sortBy?: InputMaybe<MemorySortField>;
  sortOrder?: InputMaybe<SortOrder>;
  keyword?: InputMaybe<Scalars['String']['input']>;
  minScore?: InputMaybe<Scalars['Float']['input']>;
  maxScore?: InputMaybe<Scalars['Float']['input']>;
}>;


export type ListAgentMemoriesQuery = { __typename?: 'Query', listAgentMemories: { __typename?: 'AgentMemoriesResponse', totalCount: number, memories: Array<{ __typename?: 'AgentMemory', id: string, agentID: string, query: string, claim: string, score: number, createdAt: string, lastUsedAt?: string | null }> } };

export type GetAgentMemoryQueryVariables = Exact<{
  agentID: Scalars['String']['input'];
  memoryID: Scalars['ID']['input'];
}>;


export type GetAgentMemoryQuery = { __typename?: 'Query', getAgentMemory?: { __typename?: 'AgentMemory', id: string, agentID: string, query: string, claim: string, score: number, createdAt: string, lastUsedAt?: string | null } | null };


export const GetTicketsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetTickets"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"keyword"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"assigneeID"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"tickets"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"statuses"},"value":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}}},{"kind":"Argument","name":{"kind":"Name","value":"keyword"},"value":{"kind":"Variable","name":{"kind":"Name","value":"keyword"}}},{"kind":"Argument","name":{"kind":"Name","value":"assigneeID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"assigneeID"}}},{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"tickets"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"conclusion"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"alertsCount"}},{"kind":"Field","name":{"kind":"Name","value":"commentsCount"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"tagObjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetTicketsQuery__
 *
 * To run a query within a React component, call `useGetTicketsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetTicketsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetTicketsQuery({
 *   variables: {
 *      statuses: // value for 'statuses'
 *      keyword: // value for 'keyword'
 *      assigneeID: // value for 'assigneeID'
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *   },
 * });
 */
export function useGetTicketsQuery(baseOptions?: Apollo.QueryHookOptions<GetTicketsQuery, GetTicketsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetTicketsQuery, GetTicketsQueryVariables>(GetTicketsDocument, options);
      }
export function useGetTicketsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetTicketsQuery, GetTicketsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetTicketsQuery, GetTicketsQueryVariables>(GetTicketsDocument, options);
        }
// @ts-ignore
export function useGetTicketsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetTicketsQuery, GetTicketsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketsQuery, GetTicketsQueryVariables>;
export function useGetTicketsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketsQuery, GetTicketsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketsQuery | undefined, GetTicketsQueryVariables>;
export function useGetTicketsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketsQuery, GetTicketsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTicketsQuery, GetTicketsQueryVariables>(GetTicketsDocument, options);
        }
export type GetTicketsQueryHookResult = ReturnType<typeof useGetTicketsQuery>;
export type GetTicketsLazyQueryHookResult = ReturnType<typeof useGetTicketsLazyQuery>;
export type GetTicketsSuspenseQueryHookResult = ReturnType<typeof useGetTicketsSuspenseQuery>;
export type GetTicketsQueryResult = Apollo.QueryResult<GetTicketsQuery, GetTicketsQueryVariables>;
export const GetTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ticket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"conclusion"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"finding"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"severity"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"recommendation"}}]}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"slackLink"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"alertsCount"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"tagObjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetTicketQuery__
 *
 * To run a query within a React component, call `useGetTicketQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetTicketQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetTicketQuery({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useGetTicketQuery(baseOptions: Apollo.QueryHookOptions<GetTicketQuery, GetTicketQueryVariables> & ({ variables: GetTicketQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetTicketQuery, GetTicketQueryVariables>(GetTicketDocument, options);
      }
export function useGetTicketLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetTicketQuery, GetTicketQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetTicketQuery, GetTicketQueryVariables>(GetTicketDocument, options);
        }
// @ts-ignore
export function useGetTicketSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetTicketQuery, GetTicketQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketQuery, GetTicketQueryVariables>;
export function useGetTicketSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketQuery, GetTicketQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketQuery | undefined, GetTicketQueryVariables>;
export function useGetTicketSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketQuery, GetTicketQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTicketQuery, GetTicketQueryVariables>(GetTicketDocument, options);
        }
export type GetTicketQueryHookResult = ReturnType<typeof useGetTicketQuery>;
export type GetTicketLazyQueryHookResult = ReturnType<typeof useGetTicketLazyQuery>;
export type GetTicketSuspenseQueryHookResult = ReturnType<typeof useGetTicketSuspenseQuery>;
export type GetTicketQueryResult = Apollo.QueryResult<GetTicketQuery, GetTicketQueryVariables>;
export const GetTicketAlertsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetTicketAlerts"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ticket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"alertsPaginated"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"attributes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"auto"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"tagObjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetTicketAlertsQuery__
 *
 * To run a query within a React component, call `useGetTicketAlertsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetTicketAlertsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetTicketAlertsQuery({
 *   variables: {
 *      id: // value for 'id'
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *   },
 * });
 */
export function useGetTicketAlertsQuery(baseOptions: Apollo.QueryHookOptions<GetTicketAlertsQuery, GetTicketAlertsQueryVariables> & ({ variables: GetTicketAlertsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetTicketAlertsQuery, GetTicketAlertsQueryVariables>(GetTicketAlertsDocument, options);
      }
export function useGetTicketAlertsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetTicketAlertsQuery, GetTicketAlertsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetTicketAlertsQuery, GetTicketAlertsQueryVariables>(GetTicketAlertsDocument, options);
        }
// @ts-ignore
export function useGetTicketAlertsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetTicketAlertsQuery, GetTicketAlertsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketAlertsQuery, GetTicketAlertsQueryVariables>;
export function useGetTicketAlertsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketAlertsQuery, GetTicketAlertsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketAlertsQuery | undefined, GetTicketAlertsQueryVariables>;
export function useGetTicketAlertsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketAlertsQuery, GetTicketAlertsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTicketAlertsQuery, GetTicketAlertsQueryVariables>(GetTicketAlertsDocument, options);
        }
export type GetTicketAlertsQueryHookResult = ReturnType<typeof useGetTicketAlertsQuery>;
export type GetTicketAlertsLazyQueryHookResult = ReturnType<typeof useGetTicketAlertsLazyQuery>;
export type GetTicketAlertsSuspenseQueryHookResult = ReturnType<typeof useGetTicketAlertsSuspenseQuery>;
export type GetTicketAlertsQueryResult = Apollo.QueryResult<GetTicketAlertsQuery, GetTicketAlertsQueryVariables>;
export const GetAlertDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAlert"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"alert"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"attributes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"auto"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"tagObjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"ticket"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetAlertQuery__
 *
 * To run a query within a React component, call `useGetAlertQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAlertQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAlertQuery({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useGetAlertQuery(baseOptions: Apollo.QueryHookOptions<GetAlertQuery, GetAlertQueryVariables> & ({ variables: GetAlertQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAlertQuery, GetAlertQueryVariables>(GetAlertDocument, options);
      }
export function useGetAlertLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAlertQuery, GetAlertQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAlertQuery, GetAlertQueryVariables>(GetAlertDocument, options);
        }
// @ts-ignore
export function useGetAlertSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetAlertQuery, GetAlertQueryVariables>): Apollo.UseSuspenseQueryResult<GetAlertQuery, GetAlertQueryVariables>;
export function useGetAlertSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAlertQuery, GetAlertQueryVariables>): Apollo.UseSuspenseQueryResult<GetAlertQuery | undefined, GetAlertQueryVariables>;
export function useGetAlertSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAlertQuery, GetAlertQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAlertQuery, GetAlertQueryVariables>(GetAlertDocument, options);
        }
export type GetAlertQueryHookResult = ReturnType<typeof useGetAlertQuery>;
export type GetAlertLazyQueryHookResult = ReturnType<typeof useGetAlertLazyQuery>;
export type GetAlertSuspenseQueryHookResult = ReturnType<typeof useGetAlertSuspenseQuery>;
export type GetAlertQueryResult = Apollo.QueryResult<GetAlertQuery, GetAlertQueryVariables>;
export const GetAlertsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAlerts"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"AlertStatus"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"alerts"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"attributes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"auto"}}]}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"tagObjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"ticket"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetAlertsQuery__
 *
 * To run a query within a React component, call `useGetAlertsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAlertsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAlertsQuery({
 *   variables: {
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *      status: // value for 'status'
 *   },
 * });
 */
export function useGetAlertsQuery(baseOptions?: Apollo.QueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAlertsQuery, GetAlertsQueryVariables>(GetAlertsDocument, options);
      }
export function useGetAlertsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAlertsQuery, GetAlertsQueryVariables>(GetAlertsDocument, options);
        }
// @ts-ignore
export function useGetAlertsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>): Apollo.UseSuspenseQueryResult<GetAlertsQuery, GetAlertsQueryVariables>;
export function useGetAlertsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>): Apollo.UseSuspenseQueryResult<GetAlertsQuery | undefined, GetAlertsQueryVariables>;
export function useGetAlertsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAlertsQuery, GetAlertsQueryVariables>(GetAlertsDocument, options);
        }
export type GetAlertsQueryHookResult = ReturnType<typeof useGetAlertsQuery>;
export type GetAlertsLazyQueryHookResult = ReturnType<typeof useGetAlertsLazyQuery>;
export type GetAlertsSuspenseQueryHookResult = ReturnType<typeof useGetAlertsSuspenseQuery>;
export type GetAlertsQueryResult = Apollo.QueryResult<GetAlertsQuery, GetAlertsQueryVariables>;
export const GetDashboardDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDashboard"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"dashboard"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"openTicketsCount"}},{"kind":"Field","name":{"kind":"Name","value":"unboundAlertsCount"}},{"kind":"Field","name":{"kind":"Name","value":"declinedAlertsCount"}},{"kind":"Field","name":{"kind":"Name","value":"openTickets"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"unboundAlerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetDashboardQuery__
 *
 * To run a query within a React component, call `useGetDashboardQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetDashboardQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetDashboardQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetDashboardQuery(baseOptions?: Apollo.QueryHookOptions<GetDashboardQuery, GetDashboardQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetDashboardQuery, GetDashboardQueryVariables>(GetDashboardDocument, options);
      }
export function useGetDashboardLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetDashboardQuery, GetDashboardQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetDashboardQuery, GetDashboardQueryVariables>(GetDashboardDocument, options);
        }
// @ts-ignore
export function useGetDashboardSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetDashboardQuery, GetDashboardQueryVariables>): Apollo.UseSuspenseQueryResult<GetDashboardQuery, GetDashboardQueryVariables>;
export function useGetDashboardSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetDashboardQuery, GetDashboardQueryVariables>): Apollo.UseSuspenseQueryResult<GetDashboardQuery | undefined, GetDashboardQueryVariables>;
export function useGetDashboardSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetDashboardQuery, GetDashboardQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetDashboardQuery, GetDashboardQueryVariables>(GetDashboardDocument, options);
        }
export type GetDashboardQueryHookResult = ReturnType<typeof useGetDashboardQuery>;
export type GetDashboardLazyQueryHookResult = ReturnType<typeof useGetDashboardLazyQuery>;
export type GetDashboardSuspenseQueryHookResult = ReturnType<typeof useGetDashboardSuspenseQuery>;
export type GetDashboardQueryResult = Apollo.QueryResult<GetDashboardQuery, GetDashboardQueryVariables>;
export const GetActivitiesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetActivities"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"activities"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"activities"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"userID"}},{"kind":"Field","name":{"kind":"Name","value":"alertID"}},{"kind":"Field","name":{"kind":"Name","value":"ticketID"}},{"kind":"Field","name":{"kind":"Name","value":"commentID"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"metadata"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"alert"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}},{"kind":"Field","name":{"kind":"Name","value":"ticket"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetActivitiesQuery__
 *
 * To run a query within a React component, call `useGetActivitiesQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetActivitiesQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetActivitiesQuery({
 *   variables: {
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *   },
 * });
 */
export function useGetActivitiesQuery(baseOptions?: Apollo.QueryHookOptions<GetActivitiesQuery, GetActivitiesQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetActivitiesQuery, GetActivitiesQueryVariables>(GetActivitiesDocument, options);
      }
export function useGetActivitiesLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetActivitiesQuery, GetActivitiesQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetActivitiesQuery, GetActivitiesQueryVariables>(GetActivitiesDocument, options);
        }
// @ts-ignore
export function useGetActivitiesSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetActivitiesQuery, GetActivitiesQueryVariables>): Apollo.UseSuspenseQueryResult<GetActivitiesQuery, GetActivitiesQueryVariables>;
export function useGetActivitiesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetActivitiesQuery, GetActivitiesQueryVariables>): Apollo.UseSuspenseQueryResult<GetActivitiesQuery | undefined, GetActivitiesQueryVariables>;
export function useGetActivitiesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetActivitiesQuery, GetActivitiesQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetActivitiesQuery, GetActivitiesQueryVariables>(GetActivitiesDocument, options);
        }
export type GetActivitiesQueryHookResult = ReturnType<typeof useGetActivitiesQuery>;
export type GetActivitiesLazyQueryHookResult = ReturnType<typeof useGetActivitiesLazyQuery>;
export type GetActivitiesSuspenseQueryHookResult = ReturnType<typeof useGetActivitiesSuspenseQuery>;
export type GetActivitiesQueryResult = Apollo.QueryResult<GetActivitiesQuery, GetActivitiesQueryVariables>;
export const GetTicketCommentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetTicketComments"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ticketComments"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ticketId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}}},{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetTicketCommentsQuery__
 *
 * To run a query within a React component, call `useGetTicketCommentsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetTicketCommentsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetTicketCommentsQuery({
 *   variables: {
 *      ticketId: // value for 'ticketId'
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *   },
 * });
 */
export function useGetTicketCommentsQuery(baseOptions: Apollo.QueryHookOptions<GetTicketCommentsQuery, GetTicketCommentsQueryVariables> & ({ variables: GetTicketCommentsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>(GetTicketCommentsDocument, options);
      }
export function useGetTicketCommentsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>(GetTicketCommentsDocument, options);
        }
// @ts-ignore
export function useGetTicketCommentsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>;
export function useGetTicketCommentsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketCommentsQuery | undefined, GetTicketCommentsQueryVariables>;
export function useGetTicketCommentsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>(GetTicketCommentsDocument, options);
        }
export type GetTicketCommentsQueryHookResult = ReturnType<typeof useGetTicketCommentsQuery>;
export type GetTicketCommentsLazyQueryHookResult = ReturnType<typeof useGetTicketCommentsLazyQuery>;
export type GetTicketCommentsSuspenseQueryHookResult = ReturnType<typeof useGetTicketCommentsSuspenseQuery>;
export type GetTicketCommentsQueryResult = Apollo.QueryResult<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>;
export const ResolveTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ResolveTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"conclusion"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"reason"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"resolveTicket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"conclusion"},"value":{"kind":"Variable","name":{"kind":"Name","value":"conclusion"}}},{"kind":"Argument","name":{"kind":"Name","value":"reason"},"value":{"kind":"Variable","name":{"kind":"Name","value":"reason"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"conclusion"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"resolvedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type ResolveTicketMutationFn = Apollo.MutationFunction<ResolveTicketMutation, ResolveTicketMutationVariables>;

/**
 * __useResolveTicketMutation__
 *
 * To run a mutation, you first call `useResolveTicketMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useResolveTicketMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [resolveTicketMutation, { data, loading, error }] = useResolveTicketMutation({
 *   variables: {
 *      id: // value for 'id'
 *      conclusion: // value for 'conclusion'
 *      reason: // value for 'reason'
 *   },
 * });
 */
export function useResolveTicketMutation(baseOptions?: Apollo.MutationHookOptions<ResolveTicketMutation, ResolveTicketMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ResolveTicketMutation, ResolveTicketMutationVariables>(ResolveTicketDocument, options);
      }
export type ResolveTicketMutationHookResult = ReturnType<typeof useResolveTicketMutation>;
export type ResolveTicketMutationResult = Apollo.MutationResult<ResolveTicketMutation>;
export type ResolveTicketMutationOptions = Apollo.BaseMutationOptions<ResolveTicketMutation, ResolveTicketMutationVariables>;
export const ReopenTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ReopenTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reopenTicket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"conclusion"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"resolvedAt"}},{"kind":"Field","name":{"kind":"Name","value":"archivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type ReopenTicketMutationFn = Apollo.MutationFunction<ReopenTicketMutation, ReopenTicketMutationVariables>;

/**
 * __useReopenTicketMutation__
 *
 * To run a mutation, you first call `useReopenTicketMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useReopenTicketMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [reopenTicketMutation, { data, loading, error }] = useReopenTicketMutation({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useReopenTicketMutation(baseOptions?: Apollo.MutationHookOptions<ReopenTicketMutation, ReopenTicketMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ReopenTicketMutation, ReopenTicketMutationVariables>(ReopenTicketDocument, options);
      }
export type ReopenTicketMutationHookResult = ReturnType<typeof useReopenTicketMutation>;
export type ReopenTicketMutationResult = Apollo.MutationResult<ReopenTicketMutation>;
export type ReopenTicketMutationOptions = Apollo.BaseMutationOptions<ReopenTicketMutation, ReopenTicketMutationVariables>;
export const ArchiveTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveTicket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"resolvedAt"}},{"kind":"Field","name":{"kind":"Name","value":"archivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type ArchiveTicketMutationFn = Apollo.MutationFunction<ArchiveTicketMutation, ArchiveTicketMutationVariables>;

/**
 * __useArchiveTicketMutation__
 *
 * To run a mutation, you first call `useArchiveTicketMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useArchiveTicketMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [archiveTicketMutation, { data, loading, error }] = useArchiveTicketMutation({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useArchiveTicketMutation(baseOptions?: Apollo.MutationHookOptions<ArchiveTicketMutation, ArchiveTicketMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ArchiveTicketMutation, ArchiveTicketMutationVariables>(ArchiveTicketDocument, options);
      }
export type ArchiveTicketMutationHookResult = ReturnType<typeof useArchiveTicketMutation>;
export type ArchiveTicketMutationResult = Apollo.MutationResult<ArchiveTicketMutation>;
export type ArchiveTicketMutationOptions = Apollo.BaseMutationOptions<ArchiveTicketMutation, ArchiveTicketMutationVariables>;
export const ArchiveTicketsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveTickets"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveTickets"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"resolvedAt"}},{"kind":"Field","name":{"kind":"Name","value":"archivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type ArchiveTicketsMutationFn = Apollo.MutationFunction<ArchiveTicketsMutation, ArchiveTicketsMutationVariables>;

/**
 * __useArchiveTicketsMutation__
 *
 * To run a mutation, you first call `useArchiveTicketsMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useArchiveTicketsMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [archiveTicketsMutation, { data, loading, error }] = useArchiveTicketsMutation({
 *   variables: {
 *      ids: // value for 'ids'
 *   },
 * });
 */
export function useArchiveTicketsMutation(baseOptions?: Apollo.MutationHookOptions<ArchiveTicketsMutation, ArchiveTicketsMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ArchiveTicketsMutation, ArchiveTicketsMutationVariables>(ArchiveTicketsDocument, options);
      }
export type ArchiveTicketsMutationHookResult = ReturnType<typeof useArchiveTicketsMutation>;
export type ArchiveTicketsMutationResult = Apollo.MutationResult<ArchiveTicketsMutation>;
export type ArchiveTicketsMutationOptions = Apollo.BaseMutationOptions<ArchiveTicketsMutation, ArchiveTicketsMutationVariables>;
export const UnarchiveTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnarchiveTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unarchiveTicket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"resolvedAt"}},{"kind":"Field","name":{"kind":"Name","value":"archivedAt"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type UnarchiveTicketMutationFn = Apollo.MutationFunction<UnarchiveTicketMutation, UnarchiveTicketMutationVariables>;

/**
 * __useUnarchiveTicketMutation__
 *
 * To run a mutation, you first call `useUnarchiveTicketMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUnarchiveTicketMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [unarchiveTicketMutation, { data, loading, error }] = useUnarchiveTicketMutation({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useUnarchiveTicketMutation(baseOptions?: Apollo.MutationHookOptions<UnarchiveTicketMutation, UnarchiveTicketMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UnarchiveTicketMutation, UnarchiveTicketMutationVariables>(UnarchiveTicketDocument, options);
      }
export type UnarchiveTicketMutationHookResult = ReturnType<typeof useUnarchiveTicketMutation>;
export type UnarchiveTicketMutationResult = Apollo.MutationResult<UnarchiveTicketMutation>;
export type UnarchiveTicketMutationOptions = Apollo.BaseMutationOptions<UnarchiveTicketMutation, UnarchiveTicketMutationVariables>;
export const UpdateTicketConclusionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateTicketConclusion"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"conclusion"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"reason"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateTicketConclusion"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"conclusion"},"value":{"kind":"Variable","name":{"kind":"Name","value":"conclusion"}}},{"kind":"Argument","name":{"kind":"Name","value":"reason"},"value":{"kind":"Variable","name":{"kind":"Name","value":"reason"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"conclusion"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type UpdateTicketConclusionMutationFn = Apollo.MutationFunction<UpdateTicketConclusionMutation, UpdateTicketConclusionMutationVariables>;

/**
 * __useUpdateTicketConclusionMutation__
 *
 * To run a mutation, you first call `useUpdateTicketConclusionMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateTicketConclusionMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateTicketConclusionMutation, { data, loading, error }] = useUpdateTicketConclusionMutation({
 *   variables: {
 *      id: // value for 'id'
 *      conclusion: // value for 'conclusion'
 *      reason: // value for 'reason'
 *   },
 * });
 */
export function useUpdateTicketConclusionMutation(baseOptions?: Apollo.MutationHookOptions<UpdateTicketConclusionMutation, UpdateTicketConclusionMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateTicketConclusionMutation, UpdateTicketConclusionMutationVariables>(UpdateTicketConclusionDocument, options);
      }
export type UpdateTicketConclusionMutationHookResult = ReturnType<typeof useUpdateTicketConclusionMutation>;
export type UpdateTicketConclusionMutationResult = Apollo.MutationResult<UpdateTicketConclusionMutation>;
export type UpdateTicketConclusionMutationOptions = Apollo.BaseMutationOptions<UpdateTicketConclusionMutation, UpdateTicketConclusionMutationVariables>;
export const UpdateTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"title"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"description"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateTicket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"title"},"value":{"kind":"Variable","name":{"kind":"Name","value":"title"}}},{"kind":"Argument","name":{"kind":"Name","value":"description"},"value":{"kind":"Variable","name":{"kind":"Name","value":"description"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"conclusion"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"finding"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"severity"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"recommendation"}}]}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"slackLink"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"tagObjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"attributes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"auto"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"tagObjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode;
export type UpdateTicketMutationFn = Apollo.MutationFunction<UpdateTicketMutation, UpdateTicketMutationVariables>;

/**
 * __useUpdateTicketMutation__
 *
 * To run a mutation, you first call `useUpdateTicketMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateTicketMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateTicketMutation, { data, loading, error }] = useUpdateTicketMutation({
 *   variables: {
 *      id: // value for 'id'
 *      title: // value for 'title'
 *      description: // value for 'description'
 *   },
 * });
 */
export function useUpdateTicketMutation(baseOptions?: Apollo.MutationHookOptions<UpdateTicketMutation, UpdateTicketMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateTicketMutation, UpdateTicketMutationVariables>(UpdateTicketDocument, options);
      }
export type UpdateTicketMutationHookResult = ReturnType<typeof useUpdateTicketMutation>;
export type UpdateTicketMutationResult = Apollo.MutationResult<UpdateTicketMutation>;
export type UpdateTicketMutationOptions = Apollo.BaseMutationOptions<UpdateTicketMutation, UpdateTicketMutationVariables>;
export const CreateTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"title"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"description"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"isTest"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createTicket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"title"},"value":{"kind":"Variable","name":{"kind":"Name","value":"title"}}},{"kind":"Argument","name":{"kind":"Name","value":"description"},"value":{"kind":"Variable","name":{"kind":"Name","value":"description"}}},{"kind":"Argument","name":{"kind":"Name","value":"isTest"},"value":{"kind":"Variable","name":{"kind":"Name","value":"isTest"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type CreateTicketMutationFn = Apollo.MutationFunction<CreateTicketMutation, CreateTicketMutationVariables>;

/**
 * __useCreateTicketMutation__
 *
 * To run a mutation, you first call `useCreateTicketMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useCreateTicketMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [createTicketMutation, { data, loading, error }] = useCreateTicketMutation({
 *   variables: {
 *      title: // value for 'title'
 *      description: // value for 'description'
 *      isTest: // value for 'isTest'
 *   },
 * });
 */
export function useCreateTicketMutation(baseOptions?: Apollo.MutationHookOptions<CreateTicketMutation, CreateTicketMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<CreateTicketMutation, CreateTicketMutationVariables>(CreateTicketDocument, options);
      }
export type CreateTicketMutationHookResult = ReturnType<typeof useCreateTicketMutation>;
export type CreateTicketMutationResult = Apollo.MutationResult<CreateTicketMutation>;
export type CreateTicketMutationOptions = Apollo.BaseMutationOptions<CreateTicketMutation, CreateTicketMutationVariables>;
export const GetSimilarTicketsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSimilarTickets"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"threshold"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Float"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"similarTickets"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ticketId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}}},{"kind":"Argument","name":{"kind":"Name","value":"threshold"},"value":{"kind":"Variable","name":{"kind":"Name","value":"threshold"}}},{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"tickets"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetSimilarTicketsQuery__
 *
 * To run a query within a React component, call `useGetSimilarTicketsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetSimilarTicketsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetSimilarTicketsQuery({
 *   variables: {
 *      ticketId: // value for 'ticketId'
 *      threshold: // value for 'threshold'
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *   },
 * });
 */
export function useGetSimilarTicketsQuery(baseOptions: Apollo.QueryHookOptions<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables> & ({ variables: GetSimilarTicketsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>(GetSimilarTicketsDocument, options);
      }
export function useGetSimilarTicketsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>(GetSimilarTicketsDocument, options);
        }
// @ts-ignore
export function useGetSimilarTicketsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>): Apollo.UseSuspenseQueryResult<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>;
export function useGetSimilarTicketsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>): Apollo.UseSuspenseQueryResult<GetSimilarTicketsQuery | undefined, GetSimilarTicketsQueryVariables>;
export function useGetSimilarTicketsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>(GetSimilarTicketsDocument, options);
        }
export type GetSimilarTicketsQueryHookResult = ReturnType<typeof useGetSimilarTicketsQuery>;
export type GetSimilarTicketsLazyQueryHookResult = ReturnType<typeof useGetSimilarTicketsLazyQuery>;
export type GetSimilarTicketsSuspenseQueryHookResult = ReturnType<typeof useGetSimilarTicketsSuspenseQuery>;
export type GetSimilarTicketsQueryResult = Apollo.QueryResult<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>;
export const GetSimilarTicketsForAlertDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSimilarTicketsForAlert"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"alertId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"threshold"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Float"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"similarTicketsForAlert"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"alertId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"alertId"}}},{"kind":"Argument","name":{"kind":"Name","value":"threshold"},"value":{"kind":"Variable","name":{"kind":"Name","value":"threshold"}}},{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"tickets"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetSimilarTicketsForAlertQuery__
 *
 * To run a query within a React component, call `useGetSimilarTicketsForAlertQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetSimilarTicketsForAlertQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetSimilarTicketsForAlertQuery({
 *   variables: {
 *      alertId: // value for 'alertId'
 *      threshold: // value for 'threshold'
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *   },
 * });
 */
export function useGetSimilarTicketsForAlertQuery(baseOptions: Apollo.QueryHookOptions<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables> & ({ variables: GetSimilarTicketsForAlertQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables>(GetSimilarTicketsForAlertDocument, options);
      }
export function useGetSimilarTicketsForAlertLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables>(GetSimilarTicketsForAlertDocument, options);
        }
// @ts-ignore
export function useGetSimilarTicketsForAlertSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables>): Apollo.UseSuspenseQueryResult<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables>;
export function useGetSimilarTicketsForAlertSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables>): Apollo.UseSuspenseQueryResult<GetSimilarTicketsForAlertQuery | undefined, GetSimilarTicketsForAlertQueryVariables>;
export function useGetSimilarTicketsForAlertSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables>(GetSimilarTicketsForAlertDocument, options);
        }
export type GetSimilarTicketsForAlertQueryHookResult = ReturnType<typeof useGetSimilarTicketsForAlertQuery>;
export type GetSimilarTicketsForAlertLazyQueryHookResult = ReturnType<typeof useGetSimilarTicketsForAlertLazyQuery>;
export type GetSimilarTicketsForAlertSuspenseQueryHookResult = ReturnType<typeof useGetSimilarTicketsForAlertSuspenseQuery>;
export type GetSimilarTicketsForAlertQueryResult = Apollo.QueryResult<GetSimilarTicketsForAlertQuery, GetSimilarTicketsForAlertQueryVariables>;
export const GetNewAlertsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetNewAlerts"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"threshold"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Float"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"keyword"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unboundAlerts"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"threshold"},"value":{"kind":"Variable","name":{"kind":"Name","value":"threshold"}}},{"kind":"Argument","name":{"kind":"Name","value":"keyword"},"value":{"kind":"Variable","name":{"kind":"Name","value":"keyword"}}},{"kind":"Argument","name":{"kind":"Name","value":"ticketId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}}},{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"attributes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"auto"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetNewAlertsQuery__
 *
 * To run a query within a React component, call `useGetNewAlertsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetNewAlertsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetNewAlertsQuery({
 *   variables: {
 *      threshold: // value for 'threshold'
 *      keyword: // value for 'keyword'
 *      ticketId: // value for 'ticketId'
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *   },
 * });
 */
export function useGetNewAlertsQuery(baseOptions?: Apollo.QueryHookOptions<GetNewAlertsQuery, GetNewAlertsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetNewAlertsQuery, GetNewAlertsQueryVariables>(GetNewAlertsDocument, options);
      }
export function useGetNewAlertsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetNewAlertsQuery, GetNewAlertsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetNewAlertsQuery, GetNewAlertsQueryVariables>(GetNewAlertsDocument, options);
        }
// @ts-ignore
export function useGetNewAlertsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetNewAlertsQuery, GetNewAlertsQueryVariables>): Apollo.UseSuspenseQueryResult<GetNewAlertsQuery, GetNewAlertsQueryVariables>;
export function useGetNewAlertsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetNewAlertsQuery, GetNewAlertsQueryVariables>): Apollo.UseSuspenseQueryResult<GetNewAlertsQuery | undefined, GetNewAlertsQueryVariables>;
export function useGetNewAlertsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetNewAlertsQuery, GetNewAlertsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetNewAlertsQuery, GetNewAlertsQueryVariables>(GetNewAlertsDocument, options);
        }
export type GetNewAlertsQueryHookResult = ReturnType<typeof useGetNewAlertsQuery>;
export type GetNewAlertsLazyQueryHookResult = ReturnType<typeof useGetNewAlertsLazyQuery>;
export type GetNewAlertsSuspenseQueryHookResult = ReturnType<typeof useGetNewAlertsSuspenseQuery>;
export type GetNewAlertsQueryResult = Apollo.QueryResult<GetNewAlertsQuery, GetNewAlertsQueryVariables>;
export const BindAlertsToTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BindAlertsToTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"alertIds"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bindAlertsToTicket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ticketId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}}},{"kind":"Argument","name":{"kind":"Name","value":"alertIds"},"value":{"kind":"Variable","name":{"kind":"Name","value":"alertIds"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"alertsCount"}},{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode;
export type BindAlertsToTicketMutationFn = Apollo.MutationFunction<BindAlertsToTicketMutation, BindAlertsToTicketMutationVariables>;

/**
 * __useBindAlertsToTicketMutation__
 *
 * To run a mutation, you first call `useBindAlertsToTicketMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useBindAlertsToTicketMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [bindAlertsToTicketMutation, { data, loading, error }] = useBindAlertsToTicketMutation({
 *   variables: {
 *      ticketId: // value for 'ticketId'
 *      alertIds: // value for 'alertIds'
 *   },
 * });
 */
export function useBindAlertsToTicketMutation(baseOptions?: Apollo.MutationHookOptions<BindAlertsToTicketMutation, BindAlertsToTicketMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<BindAlertsToTicketMutation, BindAlertsToTicketMutationVariables>(BindAlertsToTicketDocument, options);
      }
export type BindAlertsToTicketMutationHookResult = ReturnType<typeof useBindAlertsToTicketMutation>;
export type BindAlertsToTicketMutationResult = Apollo.MutationResult<BindAlertsToTicketMutation>;
export type BindAlertsToTicketMutationOptions = Apollo.BaseMutationOptions<BindAlertsToTicketMutation, BindAlertsToTicketMutationVariables>;
export const CreateTicketFromAlertsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateTicketFromAlerts"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"alertIds"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"title"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"description"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createTicketFromAlerts"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"alertIds"},"value":{"kind":"Variable","name":{"kind":"Name","value":"alertIds"}}},{"kind":"Argument","name":{"kind":"Name","value":"title"},"value":{"kind":"Variable","name":{"kind":"Name","value":"title"}}},{"kind":"Argument","name":{"kind":"Name","value":"description"},"value":{"kind":"Variable","name":{"kind":"Name","value":"description"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"alertsCount"}},{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}}]}}]} as unknown as DocumentNode;
export type CreateTicketFromAlertsMutationFn = Apollo.MutationFunction<CreateTicketFromAlertsMutation, CreateTicketFromAlertsMutationVariables>;

/**
 * __useCreateTicketFromAlertsMutation__
 *
 * To run a mutation, you first call `useCreateTicketFromAlertsMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useCreateTicketFromAlertsMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [createTicketFromAlertsMutation, { data, loading, error }] = useCreateTicketFromAlertsMutation({
 *   variables: {
 *      alertIds: // value for 'alertIds'
 *      title: // value for 'title'
 *      description: // value for 'description'
 *   },
 * });
 */
export function useCreateTicketFromAlertsMutation(baseOptions?: Apollo.MutationHookOptions<CreateTicketFromAlertsMutation, CreateTicketFromAlertsMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<CreateTicketFromAlertsMutation, CreateTicketFromAlertsMutationVariables>(CreateTicketFromAlertsDocument, options);
      }
export type CreateTicketFromAlertsMutationHookResult = ReturnType<typeof useCreateTicketFromAlertsMutation>;
export type CreateTicketFromAlertsMutationResult = Apollo.MutationResult<CreateTicketFromAlertsMutation>;
export type CreateTicketFromAlertsMutationOptions = Apollo.BaseMutationOptions<CreateTicketFromAlertsMutation, CreateTicketFromAlertsMutationVariables>;
export const GetTagsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetTags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"tags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"color"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetTagsQuery__
 *
 * To run a query within a React component, call `useGetTagsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetTagsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetTagsQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetTagsQuery(baseOptions?: Apollo.QueryHookOptions<GetTagsQuery, GetTagsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetTagsQuery, GetTagsQueryVariables>(GetTagsDocument, options);
      }
export function useGetTagsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetTagsQuery, GetTagsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetTagsQuery, GetTagsQueryVariables>(GetTagsDocument, options);
        }
// @ts-ignore
export function useGetTagsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetTagsQuery, GetTagsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTagsQuery, GetTagsQueryVariables>;
export function useGetTagsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTagsQuery, GetTagsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTagsQuery | undefined, GetTagsQueryVariables>;
export function useGetTagsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTagsQuery, GetTagsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTagsQuery, GetTagsQueryVariables>(GetTagsDocument, options);
        }
export type GetTagsQueryHookResult = ReturnType<typeof useGetTagsQuery>;
export type GetTagsLazyQueryHookResult = ReturnType<typeof useGetTagsLazyQuery>;
export type GetTagsSuspenseQueryHookResult = ReturnType<typeof useGetTagsSuspenseQuery>;
export type GetTagsQueryResult = Apollo.QueryResult<GetTagsQuery, GetTagsQueryVariables>;
export const CreateTagDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateTag"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"name"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createTag"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"name"},"value":{"kind":"Variable","name":{"kind":"Name","value":"name"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"color"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type CreateTagMutationFn = Apollo.MutationFunction<CreateTagMutation, CreateTagMutationVariables>;

/**
 * __useCreateTagMutation__
 *
 * To run a mutation, you first call `useCreateTagMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useCreateTagMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [createTagMutation, { data, loading, error }] = useCreateTagMutation({
 *   variables: {
 *      name: // value for 'name'
 *   },
 * });
 */
export function useCreateTagMutation(baseOptions?: Apollo.MutationHookOptions<CreateTagMutation, CreateTagMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<CreateTagMutation, CreateTagMutationVariables>(CreateTagDocument, options);
      }
export type CreateTagMutationHookResult = ReturnType<typeof useCreateTagMutation>;
export type CreateTagMutationResult = Apollo.MutationResult<CreateTagMutation>;
export type CreateTagMutationOptions = Apollo.BaseMutationOptions<CreateTagMutation, CreateTagMutationVariables>;
export const DeleteTagDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteTag"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteTag"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}]}]}}]} as unknown as DocumentNode;
export type DeleteTagMutationFn = Apollo.MutationFunction<DeleteTagMutation, DeleteTagMutationVariables>;

/**
 * __useDeleteTagMutation__
 *
 * To run a mutation, you first call `useDeleteTagMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useDeleteTagMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [deleteTagMutation, { data, loading, error }] = useDeleteTagMutation({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useDeleteTagMutation(baseOptions?: Apollo.MutationHookOptions<DeleteTagMutation, DeleteTagMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<DeleteTagMutation, DeleteTagMutationVariables>(DeleteTagDocument, options);
      }
export type DeleteTagMutationHookResult = ReturnType<typeof useDeleteTagMutation>;
export type DeleteTagMutationResult = Apollo.MutationResult<DeleteTagMutation>;
export type DeleteTagMutationOptions = Apollo.BaseMutationOptions<DeleteTagMutation, DeleteTagMutationVariables>;
export const UpdateTagDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateTag"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateTagInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateTag"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"color"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type UpdateTagMutationFn = Apollo.MutationFunction<UpdateTagMutation, UpdateTagMutationVariables>;

/**
 * __useUpdateTagMutation__
 *
 * To run a mutation, you first call `useUpdateTagMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateTagMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateTagMutation, { data, loading, error }] = useUpdateTagMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useUpdateTagMutation(baseOptions?: Apollo.MutationHookOptions<UpdateTagMutation, UpdateTagMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateTagMutation, UpdateTagMutationVariables>(UpdateTagDocument, options);
      }
export type UpdateTagMutationHookResult = ReturnType<typeof useUpdateTagMutation>;
export type UpdateTagMutationResult = Apollo.MutationResult<UpdateTagMutation>;
export type UpdateTagMutationOptions = Apollo.BaseMutationOptions<UpdateTagMutation, UpdateTagMutationVariables>;
export const GetAvailableTagColorsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAvailableTagColors"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"availableTagColors"}}]}}]} as unknown as DocumentNode;

/**
 * __useGetAvailableTagColorsQuery__
 *
 * To run a query within a React component, call `useGetAvailableTagColorsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAvailableTagColorsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAvailableTagColorsQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetAvailableTagColorsQuery(baseOptions?: Apollo.QueryHookOptions<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>(GetAvailableTagColorsDocument, options);
      }
export function useGetAvailableTagColorsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>(GetAvailableTagColorsDocument, options);
        }
// @ts-ignore
export function useGetAvailableTagColorsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>): Apollo.UseSuspenseQueryResult<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>;
export function useGetAvailableTagColorsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>): Apollo.UseSuspenseQueryResult<GetAvailableTagColorsQuery | undefined, GetAvailableTagColorsQueryVariables>;
export function useGetAvailableTagColorsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>(GetAvailableTagColorsDocument, options);
        }
export type GetAvailableTagColorsQueryHookResult = ReturnType<typeof useGetAvailableTagColorsQuery>;
export type GetAvailableTagColorsLazyQueryHookResult = ReturnType<typeof useGetAvailableTagColorsLazyQuery>;
export type GetAvailableTagColorsSuspenseQueryHookResult = ReturnType<typeof useGetAvailableTagColorsSuspenseQuery>;
export type GetAvailableTagColorsQueryResult = Apollo.QueryResult<GetAvailableTagColorsQuery, GetAvailableTagColorsQueryVariables>;
export const GetAvailableTagColorNamesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAvailableTagColorNames"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"availableTagColorNames"}}]}}]} as unknown as DocumentNode;

/**
 * __useGetAvailableTagColorNamesQuery__
 *
 * To run a query within a React component, call `useGetAvailableTagColorNamesQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAvailableTagColorNamesQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAvailableTagColorNamesQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetAvailableTagColorNamesQuery(baseOptions?: Apollo.QueryHookOptions<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>(GetAvailableTagColorNamesDocument, options);
      }
export function useGetAvailableTagColorNamesLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>(GetAvailableTagColorNamesDocument, options);
        }
// @ts-ignore
export function useGetAvailableTagColorNamesSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>): Apollo.UseSuspenseQueryResult<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>;
export function useGetAvailableTagColorNamesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>): Apollo.UseSuspenseQueryResult<GetAvailableTagColorNamesQuery | undefined, GetAvailableTagColorNamesQueryVariables>;
export function useGetAvailableTagColorNamesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>(GetAvailableTagColorNamesDocument, options);
        }
export type GetAvailableTagColorNamesQueryHookResult = ReturnType<typeof useGetAvailableTagColorNamesQuery>;
export type GetAvailableTagColorNamesLazyQueryHookResult = ReturnType<typeof useGetAvailableTagColorNamesLazyQuery>;
export type GetAvailableTagColorNamesSuspenseQueryHookResult = ReturnType<typeof useGetAvailableTagColorNamesSuspenseQuery>;
export type GetAvailableTagColorNamesQueryResult = Apollo.QueryResult<GetAvailableTagColorNamesQuery, GetAvailableTagColorNamesQueryVariables>;
export const UpdateAlertTagsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAlertTags"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"alertId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"tagIds"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateAlertTags"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"alertId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"alertId"}}},{"kind":"Argument","name":{"kind":"Name","value":"tagIds"},"value":{"kind":"Variable","name":{"kind":"Name","value":"tagIds"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"tagObjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode;
export type UpdateAlertTagsMutationFn = Apollo.MutationFunction<UpdateAlertTagsMutation, UpdateAlertTagsMutationVariables>;

/**
 * __useUpdateAlertTagsMutation__
 *
 * To run a mutation, you first call `useUpdateAlertTagsMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateAlertTagsMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateAlertTagsMutation, { data, loading, error }] = useUpdateAlertTagsMutation({
 *   variables: {
 *      alertId: // value for 'alertId'
 *      tagIds: // value for 'tagIds'
 *   },
 * });
 */
export function useUpdateAlertTagsMutation(baseOptions?: Apollo.MutationHookOptions<UpdateAlertTagsMutation, UpdateAlertTagsMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateAlertTagsMutation, UpdateAlertTagsMutationVariables>(UpdateAlertTagsDocument, options);
      }
export type UpdateAlertTagsMutationHookResult = ReturnType<typeof useUpdateAlertTagsMutation>;
export type UpdateAlertTagsMutationResult = Apollo.MutationResult<UpdateAlertTagsMutation>;
export type UpdateAlertTagsMutationOptions = Apollo.BaseMutationOptions<UpdateAlertTagsMutation, UpdateAlertTagsMutationVariables>;
export const UpdateTicketTagsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateTicketTags"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"tagIds"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateTicketTags"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ticketId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}}},{"kind":"Argument","name":{"kind":"Name","value":"tagIds"},"value":{"kind":"Variable","name":{"kind":"Name","value":"tagIds"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"tagObjects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode;
export type UpdateTicketTagsMutationFn = Apollo.MutationFunction<UpdateTicketTagsMutation, UpdateTicketTagsMutationVariables>;

/**
 * __useUpdateTicketTagsMutation__
 *
 * To run a mutation, you first call `useUpdateTicketTagsMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateTicketTagsMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateTicketTagsMutation, { data, loading, error }] = useUpdateTicketTagsMutation({
 *   variables: {
 *      ticketId: // value for 'ticketId'
 *      tagIds: // value for 'tagIds'
 *   },
 * });
 */
export function useUpdateTicketTagsMutation(baseOptions?: Apollo.MutationHookOptions<UpdateTicketTagsMutation, UpdateTicketTagsMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateTicketTagsMutation, UpdateTicketTagsMutationVariables>(UpdateTicketTagsDocument, options);
      }
export type UpdateTicketTagsMutationHookResult = ReturnType<typeof useUpdateTicketTagsMutation>;
export type UpdateTicketTagsMutationResult = Apollo.MutationResult<UpdateTicketTagsMutation>;
export type UpdateTicketTagsMutationOptions = Apollo.BaseMutationOptions<UpdateTicketTagsMutation, UpdateTicketTagsMutationVariables>;
export const DeclineAlertsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeclineAlerts"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"declineAlerts"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}}]}}]}}]} as unknown as DocumentNode;
export type DeclineAlertsMutationFn = Apollo.MutationFunction<DeclineAlertsMutation, DeclineAlertsMutationVariables>;

/**
 * __useDeclineAlertsMutation__
 *
 * To run a mutation, you first call `useDeclineAlertsMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useDeclineAlertsMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [declineAlertsMutation, { data, loading, error }] = useDeclineAlertsMutation({
 *   variables: {
 *      ids: // value for 'ids'
 *   },
 * });
 */
export function useDeclineAlertsMutation(baseOptions?: Apollo.MutationHookOptions<DeclineAlertsMutation, DeclineAlertsMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<DeclineAlertsMutation, DeclineAlertsMutationVariables>(DeclineAlertsDocument, options);
      }
export type DeclineAlertsMutationHookResult = ReturnType<typeof useDeclineAlertsMutation>;
export type DeclineAlertsMutationResult = Apollo.MutationResult<DeclineAlertsMutation>;
export type DeclineAlertsMutationOptions = Apollo.BaseMutationOptions<DeclineAlertsMutation, DeclineAlertsMutationVariables>;
export const GetKnowledgeTopicsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetKnowledgeTopics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"knowledgeTopics"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"topic"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetKnowledgeTopicsQuery__
 *
 * To run a query within a React component, call `useGetKnowledgeTopicsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetKnowledgeTopicsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetKnowledgeTopicsQuery({
 *   variables: {
 *   },
 * });
 */
export function useGetKnowledgeTopicsQuery(baseOptions?: Apollo.QueryHookOptions<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>(GetKnowledgeTopicsDocument, options);
      }
export function useGetKnowledgeTopicsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>(GetKnowledgeTopicsDocument, options);
        }
// @ts-ignore
export function useGetKnowledgeTopicsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>): Apollo.UseSuspenseQueryResult<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>;
export function useGetKnowledgeTopicsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>): Apollo.UseSuspenseQueryResult<GetKnowledgeTopicsQuery | undefined, GetKnowledgeTopicsQueryVariables>;
export function useGetKnowledgeTopicsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>(GetKnowledgeTopicsDocument, options);
        }
export type GetKnowledgeTopicsQueryHookResult = ReturnType<typeof useGetKnowledgeTopicsQuery>;
export type GetKnowledgeTopicsLazyQueryHookResult = ReturnType<typeof useGetKnowledgeTopicsLazyQuery>;
export type GetKnowledgeTopicsSuspenseQueryHookResult = ReturnType<typeof useGetKnowledgeTopicsSuspenseQuery>;
export type GetKnowledgeTopicsQueryResult = Apollo.QueryResult<GetKnowledgeTopicsQuery, GetKnowledgeTopicsQueryVariables>;
export const GetKnowledgesByTopicDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetKnowledgesByTopic"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"topic"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"knowledgesByTopic"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"topic"},"value":{"kind":"Variable","name":{"kind":"Name","value":"topic"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"topic"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"commitID"}},{"kind":"Field","name":{"kind":"Name","value":"authorID"}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"icon"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"state"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetKnowledgesByTopicQuery__
 *
 * To run a query within a React component, call `useGetKnowledgesByTopicQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetKnowledgesByTopicQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetKnowledgesByTopicQuery({
 *   variables: {
 *      topic: // value for 'topic'
 *   },
 * });
 */
export function useGetKnowledgesByTopicQuery(baseOptions: Apollo.QueryHookOptions<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables> & ({ variables: GetKnowledgesByTopicQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables>(GetKnowledgesByTopicDocument, options);
      }
export function useGetKnowledgesByTopicLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables>(GetKnowledgesByTopicDocument, options);
        }
// @ts-ignore
export function useGetKnowledgesByTopicSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables>): Apollo.UseSuspenseQueryResult<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables>;
export function useGetKnowledgesByTopicSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables>): Apollo.UseSuspenseQueryResult<GetKnowledgesByTopicQuery | undefined, GetKnowledgesByTopicQueryVariables>;
export function useGetKnowledgesByTopicSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables>(GetKnowledgesByTopicDocument, options);
        }
export type GetKnowledgesByTopicQueryHookResult = ReturnType<typeof useGetKnowledgesByTopicQuery>;
export type GetKnowledgesByTopicLazyQueryHookResult = ReturnType<typeof useGetKnowledgesByTopicLazyQuery>;
export type GetKnowledgesByTopicSuspenseQueryHookResult = ReturnType<typeof useGetKnowledgesByTopicSuspenseQuery>;
export type GetKnowledgesByTopicQueryResult = Apollo.QueryResult<GetKnowledgesByTopicQuery, GetKnowledgesByTopicQueryVariables>;
export const CreateKnowledgeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateKnowledge"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"CreateKnowledgeInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createKnowledge"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"topic"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"commitID"}},{"kind":"Field","name":{"kind":"Name","value":"authorID"}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"icon"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"state"}}]}}]}}]} as unknown as DocumentNode;
export type CreateKnowledgeMutationFn = Apollo.MutationFunction<CreateKnowledgeMutation, CreateKnowledgeMutationVariables>;

/**
 * __useCreateKnowledgeMutation__
 *
 * To run a mutation, you first call `useCreateKnowledgeMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useCreateKnowledgeMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [createKnowledgeMutation, { data, loading, error }] = useCreateKnowledgeMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useCreateKnowledgeMutation(baseOptions?: Apollo.MutationHookOptions<CreateKnowledgeMutation, CreateKnowledgeMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<CreateKnowledgeMutation, CreateKnowledgeMutationVariables>(CreateKnowledgeDocument, options);
      }
export type CreateKnowledgeMutationHookResult = ReturnType<typeof useCreateKnowledgeMutation>;
export type CreateKnowledgeMutationResult = Apollo.MutationResult<CreateKnowledgeMutation>;
export type CreateKnowledgeMutationOptions = Apollo.BaseMutationOptions<CreateKnowledgeMutation, CreateKnowledgeMutationVariables>;
export const UpdateKnowledgeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateKnowledge"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateKnowledgeInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateKnowledge"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"slug"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"topic"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"commitID"}},{"kind":"Field","name":{"kind":"Name","value":"authorID"}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"icon"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"state"}}]}}]}}]} as unknown as DocumentNode;
export type UpdateKnowledgeMutationFn = Apollo.MutationFunction<UpdateKnowledgeMutation, UpdateKnowledgeMutationVariables>;

/**
 * __useUpdateKnowledgeMutation__
 *
 * To run a mutation, you first call `useUpdateKnowledgeMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateKnowledgeMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateKnowledgeMutation, { data, loading, error }] = useUpdateKnowledgeMutation({
 *   variables: {
 *      input: // value for 'input'
 *   },
 * });
 */
export function useUpdateKnowledgeMutation(baseOptions?: Apollo.MutationHookOptions<UpdateKnowledgeMutation, UpdateKnowledgeMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateKnowledgeMutation, UpdateKnowledgeMutationVariables>(UpdateKnowledgeDocument, options);
      }
export type UpdateKnowledgeMutationHookResult = ReturnType<typeof useUpdateKnowledgeMutation>;
export type UpdateKnowledgeMutationResult = Apollo.MutationResult<UpdateKnowledgeMutation>;
export type UpdateKnowledgeMutationOptions = Apollo.BaseMutationOptions<UpdateKnowledgeMutation, UpdateKnowledgeMutationVariables>;
export const ArchiveKnowledgeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ArchiveKnowledge"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"topic"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"slug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"archiveKnowledge"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"topic"},"value":{"kind":"Variable","name":{"kind":"Name","value":"topic"}}},{"kind":"Argument","name":{"kind":"Name","value":"slug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"slug"}}}]}]}}]} as unknown as DocumentNode;
export type ArchiveKnowledgeMutationFn = Apollo.MutationFunction<ArchiveKnowledgeMutation, ArchiveKnowledgeMutationVariables>;

/**
 * __useArchiveKnowledgeMutation__
 *
 * To run a mutation, you first call `useArchiveKnowledgeMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useArchiveKnowledgeMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [archiveKnowledgeMutation, { data, loading, error }] = useArchiveKnowledgeMutation({
 *   variables: {
 *      topic: // value for 'topic'
 *      slug: // value for 'slug'
 *   },
 * });
 */
export function useArchiveKnowledgeMutation(baseOptions?: Apollo.MutationHookOptions<ArchiveKnowledgeMutation, ArchiveKnowledgeMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<ArchiveKnowledgeMutation, ArchiveKnowledgeMutationVariables>(ArchiveKnowledgeDocument, options);
      }
export type ArchiveKnowledgeMutationHookResult = ReturnType<typeof useArchiveKnowledgeMutation>;
export type ArchiveKnowledgeMutationResult = Apollo.MutationResult<ArchiveKnowledgeMutation>;
export type ArchiveKnowledgeMutationOptions = Apollo.BaseMutationOptions<ArchiveKnowledgeMutation, ArchiveKnowledgeMutationVariables>;
export const GetTicketSessionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetTicketSessions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ticketSessions"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ticketId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ticketId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"ticketID"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"userID"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"icon"}}]}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"slackURL"}},{"kind":"Field","name":{"kind":"Name","value":"intent"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetTicketSessionsQuery__
 *
 * To run a query within a React component, call `useGetTicketSessionsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetTicketSessionsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetTicketSessionsQuery({
 *   variables: {
 *      ticketId: // value for 'ticketId'
 *   },
 * });
 */
export function useGetTicketSessionsQuery(baseOptions: Apollo.QueryHookOptions<GetTicketSessionsQuery, GetTicketSessionsQueryVariables> & ({ variables: GetTicketSessionsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetTicketSessionsQuery, GetTicketSessionsQueryVariables>(GetTicketSessionsDocument, options);
      }
export function useGetTicketSessionsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetTicketSessionsQuery, GetTicketSessionsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetTicketSessionsQuery, GetTicketSessionsQueryVariables>(GetTicketSessionsDocument, options);
        }
// @ts-ignore
export function useGetTicketSessionsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetTicketSessionsQuery, GetTicketSessionsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketSessionsQuery, GetTicketSessionsQueryVariables>;
export function useGetTicketSessionsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketSessionsQuery, GetTicketSessionsQueryVariables>): Apollo.UseSuspenseQueryResult<GetTicketSessionsQuery | undefined, GetTicketSessionsQueryVariables>;
export function useGetTicketSessionsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketSessionsQuery, GetTicketSessionsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTicketSessionsQuery, GetTicketSessionsQueryVariables>(GetTicketSessionsDocument, options);
        }
export type GetTicketSessionsQueryHookResult = ReturnType<typeof useGetTicketSessionsQuery>;
export type GetTicketSessionsLazyQueryHookResult = ReturnType<typeof useGetTicketSessionsLazyQuery>;
export type GetTicketSessionsSuspenseQueryHookResult = ReturnType<typeof useGetTicketSessionsSuspenseQuery>;
export type GetTicketSessionsQueryResult = Apollo.QueryResult<GetTicketSessionsQuery, GetTicketSessionsQueryVariables>;
export const GetSessionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSession"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"session"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"ticketID"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"userID"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"icon"}}]}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"slackURL"}},{"kind":"Field","name":{"kind":"Name","value":"intent"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetSessionQuery__
 *
 * To run a query within a React component, call `useGetSessionQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetSessionQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetSessionQuery({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useGetSessionQuery(baseOptions: Apollo.QueryHookOptions<GetSessionQuery, GetSessionQueryVariables> & ({ variables: GetSessionQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetSessionQuery, GetSessionQueryVariables>(GetSessionDocument, options);
      }
export function useGetSessionLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetSessionQuery, GetSessionQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetSessionQuery, GetSessionQueryVariables>(GetSessionDocument, options);
        }
// @ts-ignore
export function useGetSessionSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetSessionQuery, GetSessionQueryVariables>): Apollo.UseSuspenseQueryResult<GetSessionQuery, GetSessionQueryVariables>;
export function useGetSessionSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSessionQuery, GetSessionQueryVariables>): Apollo.UseSuspenseQueryResult<GetSessionQuery | undefined, GetSessionQueryVariables>;
export function useGetSessionSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSessionQuery, GetSessionQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetSessionQuery, GetSessionQueryVariables>(GetSessionDocument, options);
        }
export type GetSessionQueryHookResult = ReturnType<typeof useGetSessionQuery>;
export type GetSessionLazyQueryHookResult = ReturnType<typeof useGetSessionLazyQuery>;
export type GetSessionSuspenseQueryHookResult = ReturnType<typeof useGetSessionSuspenseQuery>;
export type GetSessionQueryResult = Apollo.QueryResult<GetSessionQuery, GetSessionQueryVariables>;
export const GetSessionMessagesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetSessionMessages"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sessionId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sessionMessages"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"sessionId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sessionId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"sessionID"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetSessionMessagesQuery__
 *
 * To run a query within a React component, call `useGetSessionMessagesQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetSessionMessagesQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetSessionMessagesQuery({
 *   variables: {
 *      sessionId: // value for 'sessionId'
 *   },
 * });
 */
export function useGetSessionMessagesQuery(baseOptions: Apollo.QueryHookOptions<GetSessionMessagesQuery, GetSessionMessagesQueryVariables> & ({ variables: GetSessionMessagesQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetSessionMessagesQuery, GetSessionMessagesQueryVariables>(GetSessionMessagesDocument, options);
      }
export function useGetSessionMessagesLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetSessionMessagesQuery, GetSessionMessagesQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetSessionMessagesQuery, GetSessionMessagesQueryVariables>(GetSessionMessagesDocument, options);
        }
// @ts-ignore
export function useGetSessionMessagesSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetSessionMessagesQuery, GetSessionMessagesQueryVariables>): Apollo.UseSuspenseQueryResult<GetSessionMessagesQuery, GetSessionMessagesQueryVariables>;
export function useGetSessionMessagesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSessionMessagesQuery, GetSessionMessagesQueryVariables>): Apollo.UseSuspenseQueryResult<GetSessionMessagesQuery | undefined, GetSessionMessagesQueryVariables>;
export function useGetSessionMessagesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSessionMessagesQuery, GetSessionMessagesQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetSessionMessagesQuery, GetSessionMessagesQueryVariables>(GetSessionMessagesDocument, options);
        }
export type GetSessionMessagesQueryHookResult = ReturnType<typeof useGetSessionMessagesQuery>;
export type GetSessionMessagesLazyQueryHookResult = ReturnType<typeof useGetSessionMessagesLazyQuery>;
export type GetSessionMessagesSuspenseQueryHookResult = ReturnType<typeof useGetSessionMessagesSuspenseQuery>;
export type GetSessionMessagesQueryResult = Apollo.QueryResult<GetSessionMessagesQuery, GetSessionMessagesQueryVariables>;
export const ListAgentSummariesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ListAgentSummaries"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"keyword"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"listAgentSummaries"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"keyword"},"value":{"kind":"Variable","name":{"kind":"Name","value":"keyword"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"agents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"agentID"}},{"kind":"Field","name":{"kind":"Name","value":"memoriesCount"}},{"kind":"Field","name":{"kind":"Name","value":"latestMemoryAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useListAgentSummariesQuery__
 *
 * To run a query within a React component, call `useListAgentSummariesQuery` and pass it any options that fit your needs.
 * When your component renders, `useListAgentSummariesQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useListAgentSummariesQuery({
 *   variables: {
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *      keyword: // value for 'keyword'
 *   },
 * });
 */
export function useListAgentSummariesQuery(baseOptions?: Apollo.QueryHookOptions<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>(ListAgentSummariesDocument, options);
      }
export function useListAgentSummariesLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>(ListAgentSummariesDocument, options);
        }
// @ts-ignore
export function useListAgentSummariesSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>): Apollo.UseSuspenseQueryResult<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>;
export function useListAgentSummariesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>): Apollo.UseSuspenseQueryResult<ListAgentSummariesQuery | undefined, ListAgentSummariesQueryVariables>;
export function useListAgentSummariesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>(ListAgentSummariesDocument, options);
        }
export type ListAgentSummariesQueryHookResult = ReturnType<typeof useListAgentSummariesQuery>;
export type ListAgentSummariesLazyQueryHookResult = ReturnType<typeof useListAgentSummariesLazyQuery>;
export type ListAgentSummariesSuspenseQueryHookResult = ReturnType<typeof useListAgentSummariesSuspenseQuery>;
export type ListAgentSummariesQueryResult = Apollo.QueryResult<ListAgentSummariesQuery, ListAgentSummariesQueryVariables>;
export const ListAgentMemoriesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ListAgentMemories"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"agentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sortBy"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"MemorySortField"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sortOrder"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"SortOrder"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"keyword"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"minScore"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Float"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"maxScore"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Float"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"listAgentMemories"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"agentID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"agentID"}}},{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"sortBy"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sortBy"}}},{"kind":"Argument","name":{"kind":"Name","value":"sortOrder"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sortOrder"}}},{"kind":"Argument","name":{"kind":"Name","value":"keyword"},"value":{"kind":"Variable","name":{"kind":"Name","value":"keyword"}}},{"kind":"Argument","name":{"kind":"Name","value":"minScore"},"value":{"kind":"Variable","name":{"kind":"Name","value":"minScore"}}},{"kind":"Argument","name":{"kind":"Name","value":"maxScore"},"value":{"kind":"Variable","name":{"kind":"Name","value":"maxScore"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"memories"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"agentID"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"claim"}},{"kind":"Field","name":{"kind":"Name","value":"score"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastUsedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useListAgentMemoriesQuery__
 *
 * To run a query within a React component, call `useListAgentMemoriesQuery` and pass it any options that fit your needs.
 * When your component renders, `useListAgentMemoriesQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useListAgentMemoriesQuery({
 *   variables: {
 *      agentID: // value for 'agentID'
 *      offset: // value for 'offset'
 *      limit: // value for 'limit'
 *      sortBy: // value for 'sortBy'
 *      sortOrder: // value for 'sortOrder'
 *      keyword: // value for 'keyword'
 *      minScore: // value for 'minScore'
 *      maxScore: // value for 'maxScore'
 *   },
 * });
 */
export function useListAgentMemoriesQuery(baseOptions: Apollo.QueryHookOptions<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables> & ({ variables: ListAgentMemoriesQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables>(ListAgentMemoriesDocument, options);
      }
export function useListAgentMemoriesLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables>(ListAgentMemoriesDocument, options);
        }
// @ts-ignore
export function useListAgentMemoriesSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables>): Apollo.UseSuspenseQueryResult<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables>;
export function useListAgentMemoriesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables>): Apollo.UseSuspenseQueryResult<ListAgentMemoriesQuery | undefined, ListAgentMemoriesQueryVariables>;
export function useListAgentMemoriesSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables>(ListAgentMemoriesDocument, options);
        }
export type ListAgentMemoriesQueryHookResult = ReturnType<typeof useListAgentMemoriesQuery>;
export type ListAgentMemoriesLazyQueryHookResult = ReturnType<typeof useListAgentMemoriesLazyQuery>;
export type ListAgentMemoriesSuspenseQueryHookResult = ReturnType<typeof useListAgentMemoriesSuspenseQuery>;
export type ListAgentMemoriesQueryResult = Apollo.QueryResult<ListAgentMemoriesQuery, ListAgentMemoriesQueryVariables>;
export const GetAgentMemoryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAgentMemory"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"agentID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"memoryID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"getAgentMemory"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"agentID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"agentID"}}},{"kind":"Argument","name":{"kind":"Name","value":"memoryID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"memoryID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"agentID"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"claim"}},{"kind":"Field","name":{"kind":"Name","value":"score"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastUsedAt"}}]}}]}}]} as unknown as DocumentNode;

/**
 * __useGetAgentMemoryQuery__
 *
 * To run a query within a React component, call `useGetAgentMemoryQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAgentMemoryQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAgentMemoryQuery({
 *   variables: {
 *      agentID: // value for 'agentID'
 *      memoryID: // value for 'memoryID'
 *   },
 * });
 */
export function useGetAgentMemoryQuery(baseOptions: Apollo.QueryHookOptions<GetAgentMemoryQuery, GetAgentMemoryQueryVariables> & ({ variables: GetAgentMemoryQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAgentMemoryQuery, GetAgentMemoryQueryVariables>(GetAgentMemoryDocument, options);
      }
export function useGetAgentMemoryLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAgentMemoryQuery, GetAgentMemoryQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAgentMemoryQuery, GetAgentMemoryQueryVariables>(GetAgentMemoryDocument, options);
        }
// @ts-ignore
export function useGetAgentMemorySuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetAgentMemoryQuery, GetAgentMemoryQueryVariables>): Apollo.UseSuspenseQueryResult<GetAgentMemoryQuery, GetAgentMemoryQueryVariables>;
export function useGetAgentMemorySuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAgentMemoryQuery, GetAgentMemoryQueryVariables>): Apollo.UseSuspenseQueryResult<GetAgentMemoryQuery | undefined, GetAgentMemoryQueryVariables>;
export function useGetAgentMemorySuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAgentMemoryQuery, GetAgentMemoryQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAgentMemoryQuery, GetAgentMemoryQueryVariables>(GetAgentMemoryDocument, options);
        }
export type GetAgentMemoryQueryHookResult = ReturnType<typeof useGetAgentMemoryQuery>;
export type GetAgentMemoryLazyQueryHookResult = ReturnType<typeof useGetAgentMemoryLazyQuery>;
export type GetAgentMemorySuspenseQueryHookResult = ReturnType<typeof useGetAgentMemorySuspenseQuery>;
export type GetAgentMemoryQueryResult = Apollo.QueryResult<GetAgentMemoryQuery, GetAgentMemoryQueryVariables>;
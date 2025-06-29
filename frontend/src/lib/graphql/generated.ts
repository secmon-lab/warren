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

export type Alert = {
  __typename?: 'Alert';
  attributes: Array<AlertAttribute>;
  createdAt: Scalars['String']['output'];
  data: Scalars['String']['output'];
  description?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  schema: Scalars['String']['output'];
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

export type DashboardStats = {
  __typename?: 'DashboardStats';
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

export type Mutation = {
  __typename?: 'Mutation';
  createTicket: Ticket;
  updateMultipleTicketsStatus: Array<Ticket>;
  updateTicket: Ticket;
  updateTicketConclusion: Ticket;
  updateTicketStatus: Ticket;
};


export type MutationCreateTicketArgs = {
  description: Scalars['String']['input'];
  isTest?: InputMaybe<Scalars['Boolean']['input']>;
  title: Scalars['String']['input'];
};


export type MutationUpdateMultipleTicketsStatusArgs = {
  ids: Array<Scalars['ID']['input']>;
  status: Scalars['String']['input'];
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


export type MutationUpdateTicketStatusArgs = {
  id: Scalars['ID']['input'];
  status: Scalars['String']['input'];
};

export type Query = {
  __typename?: 'Query';
  activities: ActivitiesResponse;
  alert?: Maybe<Alert>;
  alerts: Array<Alert>;
  dashboard: DashboardStats;
  similarTickets: TicketsResponse;
  ticket?: Maybe<Ticket>;
  ticketComments: CommentsResponse;
  tickets: TicketsResponse;
};


export type QueryActivitiesArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryAlertArgs = {
  id: Scalars['ID']['input'];
};


export type QuerySimilarTicketsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  threshold: Scalars['Float']['input'];
  ticketId: Scalars['ID']['input'];
};


export type QueryTicketArgs = {
  id: Scalars['ID']['input'];
};


export type QueryTicketCommentsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  ticketId: Scalars['ID']['input'];
};


export type QueryTicketsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  statuses?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type Ticket = {
  __typename?: 'Ticket';
  alerts: Array<Alert>;
  assignee?: Maybe<User>;
  comments: Array<Comment>;
  conclusion?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['String']['output'];
  description: Scalars['String']['output'];
  finding?: Maybe<Finding>;
  id: Scalars['ID']['output'];
  isTest: Scalars['Boolean']['output'];
  reason?: Maybe<Scalars['String']['output']>;
  slackLink?: Maybe<Scalars['String']['output']>;
  status: Scalars['String']['output'];
  summary: Scalars['String']['output'];
  title: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
};

export type TicketsResponse = {
  __typename?: 'TicketsResponse';
  tickets: Array<Ticket>;
  totalCount: Scalars['Int']['output'];
};

export type User = {
  __typename?: 'User';
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
};

export type GetTicketsQueryVariables = Exact<{
  statuses?: InputMaybe<Array<Scalars['String']['input']> | Scalars['String']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetTicketsQuery = { __typename?: 'Query', tickets: { __typename?: 'TicketsResponse', totalCount: number, tickets: Array<{ __typename?: 'Ticket', id: string, status: string, title: string, description: string, conclusion?: string | null, reason?: string | null, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null, alerts: Array<{ __typename?: 'Alert', id: string, title: string, createdAt: string }>, comments: Array<{ __typename?: 'Comment', id: string, content: string, createdAt: string, user?: { __typename?: 'User', id: string, name: string } | null }> }> } };

export type GetTicketQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetTicketQuery = { __typename?: 'Query', ticket?: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, summary: string, conclusion?: string | null, reason?: string | null, isTest: boolean, slackLink?: string | null, createdAt: string, updatedAt: string, finding?: { __typename?: 'Finding', severity: string, summary: string, reason: string, recommendation: string } | null, assignee?: { __typename?: 'User', id: string, name: string } | null, alerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, data: string, createdAt: string, attributes: Array<{ __typename?: 'AlertAttribute', key: string, value: string, link?: string | null, auto: boolean }> }>, comments: Array<{ __typename?: 'Comment', id: string, content: string, createdAt: string, updatedAt: string, user?: { __typename?: 'User', id: string, name: string } | null }> } | null };

export type GetAlertQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetAlertQuery = { __typename?: 'Query', alert?: { __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, data: string, createdAt: string, attributes: Array<{ __typename?: 'AlertAttribute', key: string, value: string, link?: string | null, auto: boolean }>, ticket?: { __typename?: 'Ticket', id: string, status: string, title: string, description: string } | null } | null };

export type GetAlertsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetAlertsQuery = { __typename?: 'Query', alerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, data: string, createdAt: string, attributes: Array<{ __typename?: 'AlertAttribute', key: string, value: string, link?: string | null, auto: boolean }>, ticket?: { __typename?: 'Ticket', id: string, status: string, title: string, description: string } | null }> };

export type GetDashboardQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDashboardQuery = { __typename?: 'Query', dashboard: { __typename?: 'DashboardStats', openTicketsCount: number, unboundAlertsCount: number, openTickets: Array<{ __typename?: 'Ticket', id: string, status: string, title: string, description: string, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null }>, unboundAlerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null, createdAt: string }> } };

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

export type UpdateTicketStatusMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  status: Scalars['String']['input'];
}>;


export type UpdateTicketStatusMutation = { __typename?: 'Mutation', updateTicketStatus: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null } };

export type UpdateMultipleTicketsStatusMutationVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
  status: Scalars['String']['input'];
}>;


export type UpdateMultipleTicketsStatusMutation = { __typename?: 'Mutation', updateMultipleTicketsStatus: Array<{ __typename?: 'Ticket', id: string, status: string, title: string, description: string, isTest: boolean, createdAt: string, updatedAt: string, assignee?: { __typename?: 'User', id: string, name: string } | null }> };

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


export type UpdateTicketMutation = { __typename?: 'Mutation', updateTicket: { __typename?: 'Ticket', id: string, status: string, title: string, description: string, summary: string, conclusion?: string | null, reason?: string | null, isTest: boolean, slackLink?: string | null, createdAt: string, updatedAt: string, finding?: { __typename?: 'Finding', severity: string, summary: string, reason: string, recommendation: string } | null, assignee?: { __typename?: 'User', id: string, name: string } | null, alerts: Array<{ __typename?: 'Alert', id: string, title: string, description?: string | null, schema: string, data: string, createdAt: string, attributes: Array<{ __typename?: 'AlertAttribute', key: string, value: string, link?: string | null, auto: boolean }> }>, comments: Array<{ __typename?: 'Comment', id: string, content: string, createdAt: string, updatedAt: string, user?: { __typename?: 'User', id: string, name: string } | null }> } };

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


export const GetTicketsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetTickets"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"tickets"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"statuses"},"value":{"kind":"Variable","name":{"kind":"Name","value":"statuses"}}},{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"tickets"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"conclusion"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode;

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
export function useGetTicketsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketsQuery, GetTicketsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTicketsQuery, GetTicketsQueryVariables>(GetTicketsDocument, options);
        }
export type GetTicketsQueryHookResult = ReturnType<typeof useGetTicketsQuery>;
export type GetTicketsLazyQueryHookResult = ReturnType<typeof useGetTicketsLazyQuery>;
export type GetTicketsSuspenseQueryHookResult = ReturnType<typeof useGetTicketsSuspenseQuery>;
export type GetTicketsQueryResult = Apollo.QueryResult<GetTicketsQuery, GetTicketsQueryVariables>;
export const GetTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ticket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"conclusion"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"finding"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"severity"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"recommendation"}}]}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"slackLink"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"attributes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"auto"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode;

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
export function useGetTicketSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketQuery, GetTicketQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTicketQuery, GetTicketQueryVariables>(GetTicketDocument, options);
        }
export type GetTicketQueryHookResult = ReturnType<typeof useGetTicketQuery>;
export type GetTicketLazyQueryHookResult = ReturnType<typeof useGetTicketLazyQuery>;
export type GetTicketSuspenseQueryHookResult = ReturnType<typeof useGetTicketSuspenseQuery>;
export type GetTicketQueryResult = Apollo.QueryResult<GetTicketQuery, GetTicketQueryVariables>;
export const GetAlertDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAlert"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"alert"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"attributes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"auto"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"ticket"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}}]}}]} as unknown as DocumentNode;

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
export function useGetAlertSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAlertQuery, GetAlertQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAlertQuery, GetAlertQueryVariables>(GetAlertDocument, options);
        }
export type GetAlertQueryHookResult = ReturnType<typeof useGetAlertQuery>;
export type GetAlertLazyQueryHookResult = ReturnType<typeof useGetAlertLazyQuery>;
export type GetAlertSuspenseQueryHookResult = ReturnType<typeof useGetAlertSuspenseQuery>;
export type GetAlertQueryResult = Apollo.QueryResult<GetAlertQuery, GetAlertQueryVariables>;
export const GetAlertsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetAlerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"attributes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"auto"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"ticket"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}}]}}]}}]}}]} as unknown as DocumentNode;

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
export function useGetAlertsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAlertsQuery, GetAlertsQueryVariables>(GetAlertsDocument, options);
        }
export type GetAlertsQueryHookResult = ReturnType<typeof useGetAlertsQuery>;
export type GetAlertsLazyQueryHookResult = ReturnType<typeof useGetAlertsLazyQuery>;
export type GetAlertsSuspenseQueryHookResult = ReturnType<typeof useGetAlertsSuspenseQuery>;
export type GetAlertsQueryResult = Apollo.QueryResult<GetAlertsQuery, GetAlertsQueryVariables>;
export const GetDashboardDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDashboard"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"dashboard"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"openTicketsCount"}},{"kind":"Field","name":{"kind":"Name","value":"unboundAlertsCount"}},{"kind":"Field","name":{"kind":"Name","value":"openTickets"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"unboundAlerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]}}]} as unknown as DocumentNode;

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
export function useGetTicketCommentsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>(GetTicketCommentsDocument, options);
        }
export type GetTicketCommentsQueryHookResult = ReturnType<typeof useGetTicketCommentsQuery>;
export type GetTicketCommentsLazyQueryHookResult = ReturnType<typeof useGetTicketCommentsLazyQuery>;
export type GetTicketCommentsSuspenseQueryHookResult = ReturnType<typeof useGetTicketCommentsSuspenseQuery>;
export type GetTicketCommentsQueryResult = Apollo.QueryResult<GetTicketCommentsQuery, GetTicketCommentsQueryVariables>;
export const UpdateTicketStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateTicketStatus"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateTicketStatus"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type UpdateTicketStatusMutationFn = Apollo.MutationFunction<UpdateTicketStatusMutation, UpdateTicketStatusMutationVariables>;

/**
 * __useUpdateTicketStatusMutation__
 *
 * To run a mutation, you first call `useUpdateTicketStatusMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateTicketStatusMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateTicketStatusMutation, { data, loading, error }] = useUpdateTicketStatusMutation({
 *   variables: {
 *      id: // value for 'id'
 *      status: // value for 'status'
 *   },
 * });
 */
export function useUpdateTicketStatusMutation(baseOptions?: Apollo.MutationHookOptions<UpdateTicketStatusMutation, UpdateTicketStatusMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateTicketStatusMutation, UpdateTicketStatusMutationVariables>(UpdateTicketStatusDocument, options);
      }
export type UpdateTicketStatusMutationHookResult = ReturnType<typeof useUpdateTicketStatusMutation>;
export type UpdateTicketStatusMutationResult = Apollo.MutationResult<UpdateTicketStatusMutation>;
export type UpdateTicketStatusMutationOptions = Apollo.BaseMutationOptions<UpdateTicketStatusMutation, UpdateTicketStatusMutationVariables>;
export const UpdateMultipleTicketsStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateMultipleTicketsStatus"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateMultipleTicketsStatus"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}},{"kind":"Argument","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode;
export type UpdateMultipleTicketsStatusMutationFn = Apollo.MutationFunction<UpdateMultipleTicketsStatusMutation, UpdateMultipleTicketsStatusMutationVariables>;

/**
 * __useUpdateMultipleTicketsStatusMutation__
 *
 * To run a mutation, you first call `useUpdateMultipleTicketsStatusMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useUpdateMultipleTicketsStatusMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [updateMultipleTicketsStatusMutation, { data, loading, error }] = useUpdateMultipleTicketsStatusMutation({
 *   variables: {
 *      ids: // value for 'ids'
 *      status: // value for 'status'
 *   },
 * });
 */
export function useUpdateMultipleTicketsStatusMutation(baseOptions?: Apollo.MutationHookOptions<UpdateMultipleTicketsStatusMutation, UpdateMultipleTicketsStatusMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<UpdateMultipleTicketsStatusMutation, UpdateMultipleTicketsStatusMutationVariables>(UpdateMultipleTicketsStatusDocument, options);
      }
export type UpdateMultipleTicketsStatusMutationHookResult = ReturnType<typeof useUpdateMultipleTicketsStatusMutation>;
export type UpdateMultipleTicketsStatusMutationResult = Apollo.MutationResult<UpdateMultipleTicketsStatusMutation>;
export type UpdateMultipleTicketsStatusMutationOptions = Apollo.BaseMutationOptions<UpdateMultipleTicketsStatusMutation, UpdateMultipleTicketsStatusMutationVariables>;
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
export const UpdateTicketDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateTicket"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"title"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"description"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateTicket"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"title"},"value":{"kind":"Variable","name":{"kind":"Name","value":"title"}}},{"kind":"Argument","name":{"kind":"Name","value":"description"},"value":{"kind":"Variable","name":{"kind":"Name","value":"description"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"conclusion"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"isTest"}},{"kind":"Field","name":{"kind":"Name","value":"finding"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"severity"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}},{"kind":"Field","name":{"kind":"Name","value":"recommendation"}}]}},{"kind":"Field","name":{"kind":"Name","value":"assignee"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"slackLink"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"alerts"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"schema"}},{"kind":"Field","name":{"kind":"Name","value":"data"}},{"kind":"Field","name":{"kind":"Name","value":"attributes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"auto"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"user"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]}}]} as unknown as DocumentNode;
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
export function useGetSimilarTicketsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>(GetSimilarTicketsDocument, options);
        }
export type GetSimilarTicketsQueryHookResult = ReturnType<typeof useGetSimilarTicketsQuery>;
export type GetSimilarTicketsLazyQueryHookResult = ReturnType<typeof useGetSimilarTicketsLazyQuery>;
export type GetSimilarTicketsSuspenseQueryHookResult = ReturnType<typeof useGetSimilarTicketsSuspenseQuery>;
export type GetSimilarTicketsQueryResult = Apollo.QueryResult<GetSimilarTicketsQuery, GetSimilarTicketsQueryVariables>;